package openai

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func normalizeStandardImageRequest(request dto.ImageRequest) (dto.ImageRequest, error) {
	standard := common.OpenAIImageStandardOptions(request.Model)
	if standard == nil {
		return request, nil
	}
	if request.Quality == "standard" {
		request.Quality = "medium"
	}
	if request.Quality == "hd" {
		request.Quality = "high"
	}
	if len(request.Mask) > 0 && string(request.Mask) != "null" && len(request.Markup) > 0 && string(request.Markup) != "null" {
		return request, fmt.Errorf("mask and markup cannot be combined")
	}
	if len(request.Markup) > 0 && string(request.Markup) != "null" && !standard.SupportsMarkup {
		return request, fmt.Errorf("markup requires GPT Image 2.5")
	}
	if request.Size == "" {
		var resolution, ratio string
		raw := request.Resolution
		if len(raw) == 0 {
			raw = request.OutputResolution
		}
		if len(raw) > 0 {
			if err := common.Unmarshal(raw, &resolution); err != nil {
				return request, err
			}
		}
		if request.Ratio != nil {
			ratio = *request.Ratio
		} else if len(request.AspectRatio) > 0 {
			if err := common.Unmarshal(request.AspectRatio, &ratio); err != nil {
				return request, err
			}
		}
		if resolution != "" || ratio != "" {
			var err error
			request.Size, err = common.OpenAIImagePresetSize(resolution, ratio)
			if err != nil {
				return request, err
			}
		}
	}
	if err := common.ValidateOpenAIImageSize(request.Size); err != nil {
		return request, err
	}
	if request.N != nil && (*request.N < 1 || *request.N > uint(standard.MaxOutputs)) {
		return request, fmt.Errorf("n must be between 1 and %d", standard.MaxOutputs)
	}
	if request.Quality != "" && !common.StringsContains(standard.Qualities, request.Quality) {
		return request, fmt.Errorf("unsupported image quality: %s", request.Quality)
	}
	var format, background string
	if len(request.OutputFormat) > 0 {
		if err := common.Unmarshal(request.OutputFormat, &format); err != nil {
			return request, err
		}
	}
	if len(request.Background) > 0 {
		if err := common.Unmarshal(request.Background, &background); err != nil {
			return request, err
		}
	}
	if format != "" && !common.StringsContains(standard.OutputFormats, format) {
		return request, fmt.Errorf("unsupported standard image output format")
	}
	if background != "" && !common.StringsContains(standard.Backgrounds, background) {
		return request, fmt.Errorf("unsupported image background")
	}
	if background == "transparent" && format == "jpeg" {
		return request, fmt.Errorf("transparent backgrounds require png or webp")
	}
	if len(request.OutputCompression) > 0 && string(request.OutputCompression) != "null" {
		var value int
		if err := common.Unmarshal(request.OutputCompression, &value); err != nil {
			return request, err
		}
		if value < 0 || value > 100 {
			return request, fmt.Errorf("output_compression must be between 0 and 100")
		}
	}
	if request.Model == "gpt-image-2" || request.Model == "gpt-image-2-2026-04-21" {
		request.InputFidelity = nil
	}
	request.Resolution, request.OutputResolution, request.AspectRatio, request.Ratio = nil, nil, nil, nil
	return request, nil
}

func standardImageFormValues(values map[string][]string, model string) (map[string][]string, error) {
	raw := make(map[string]interface{})
	for key, v := range values {
		if len(v) == 0 {
			continue
		}
		raw[key] = v[0]
		if key == "n" {
			n, err := strconv.ParseUint(v[0], 10, 32)
			if err != nil {
				return nil, err
			}
			raw[key] = n
		}
		if key == "output_compression" || key == "partial_images" {
			n, err := strconv.Atoi(v[0])
			if err != nil {
				return nil, err
			}
			raw[key] = n
		}
		if key == "watermark" {
			value, err := strconv.ParseBool(v[0])
			if err != nil {
				return nil, err
			}
			raw[key] = value
		}
	}
	raw["model"] = model
	encoded, err := common.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var request dto.ImageRequest
	if err = common.Unmarshal(encoded, &request); err != nil {
		return nil, err
	}
	request, err = normalizeStandardImageRequest(request)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for key, v := range values {
		result[key] = v
	}
	for _, key := range []string{"resolution", "output_resolution", "ratio", "aspect_ratio"} {
		delete(result, key)
	}
	if len(request.InputFidelity) == 0 {
		delete(result, "input_fidelity")
	}
	if request.Size != "" {
		result["size"] = []string{request.Size}
	}
	if request.Quality != "" {
		result["quality"] = []string{request.Quality}
	}
	return result, nil
}
