package openai

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func isXJZNativeImageRequest(info *relaycommon.RelayInfo, model string) bool {
	return info != nil && info.ChannelMeta != nil && common.IsBaseURLHost(info.ChannelBaseUrl, "api.xjzapi.cc") && common.IsXJZNativeImageModel(model)
}

func xjzImageOptions(resolution, ratio string) (string, string, error) {
	if resolution == "" {
		resolution = "1K"
	}
	if ratio == "" {
		ratio = "1:1"
	}
	p := common.XJZNativeImageParameters(false)
	if !common.StringsContains(p.Resolutions, resolution) || !common.StringsContains(p.Ratios, ratio) {
		return "", "", fmt.Errorf("unsupported XJZ image resolution or ratio: %s / %s", resolution, ratio)
	}
	return resolution, ratio, nil
}

func convertXJZNativeImageJSON(request dto.ImageRequest) (dto.ImageRequest, error) {
	if len(request.Images) > 0 || len(request.Image) > 0 || len(request.Mask) > 0 {
		return request, fmt.Errorf("XJZ native image references require multipart POST /v1/images/edits with image[] files; masks are unsupported")
	}
	if request.N != nil && *request.N != 1 {
		return request, fmt.Errorf("XJZ native image n must be 1")
	}
	var resolution, ratio string
	rawResolution := request.Resolution
	if len(rawResolution) == 0 {
		rawResolution = request.OutputResolution
	}
	if len(rawResolution) > 0 {
		if err := common.Unmarshal(rawResolution, &resolution); err != nil {
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
	resolution, ratio, err := xjzImageOptions(resolution, ratio)
	if err != nil {
		return request, err
	}
	encoded, err := common.Marshal(resolution)
	if err != nil {
		return request, err
	}
	return dto.ImageRequest{Model: request.Model, Prompt: request.Prompt, N: request.N,
		Resolution: encoded, Ratio: &ratio, Quality: request.Quality, ResponseFormat: request.ResponseFormat}, nil
}

// Multipart does not pass through JSON parameter overrides. Normalize aliases
// here as well so editing and generation use the same resolution/ratio contract.
func xjzImageFormValues(values map[string][]string) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, key := range []string{"prompt", "n", "quality", "response_format"} {
		result[key] = values[key]
	}
	first := func(keys ...string) string {
		for _, key := range keys {
			if v := values[key]; len(v) > 0 {
				return strings.TrimSpace(v[0])
			}
		}
		return ""
	}
	if n := first("n"); n != "" && n != "1" {
		return nil, fmt.Errorf("XJZ native image n must be 1")
	}
	resolution, ratio, err := xjzImageOptions(first("resolution", "output_resolution"), first("ratio", "aspect_ratio"))
	if err != nil {
		return nil, err
	}
	result["resolution"], result["ratio"] = []string{resolution}, []string{ratio}
	return result, nil
}
