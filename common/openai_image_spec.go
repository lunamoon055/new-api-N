package common

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

func IsOpenAIStandardImageChannel(channelType int, baseURL, modelName string) bool {
	return channelType == constant.ChannelTypeOpenAI && OpenAIImageStandardOptions(modelName) != nil && (IsBaseURLHost(baseURL, "api.openai.com") || IsBaseURLHost(baseURL, MegaBYAIProviderHost))
}

// OpenAI Images API, checked 2026-11-07. Only exact supported model IDs inherit
// this baseline; vendor suffixes keep their explicitly configured contracts.
type ImageStandardOptions struct {
	Source          string   `json:"source"`
	Sizes           []string `json:"sizes"`
	Qualities       []string `json:"qualities"`
	OutputFormats   []string `json:"output_formats"`
	Backgrounds     []string `json:"backgrounds"`
	MaxOutputs      int      `json:"max_outputs"`
	MaxReferences   int      `json:"max_references"`
	MaxInputSizeMB  int      `json:"max_input_size_mb"`
	SupportsMask    bool     `json:"supports_mask"`
	SupportsMarkup  bool     `json:"supports_markup"`
	MinPixels       int      `json:"min_pixels"`
	MaxPixels       int      `json:"max_pixels"`
	MaxEdge         int      `json:"max_edge"`
	SizeMultiple    int      `json:"size_multiple"`
	MaxAspectRatio  int      `json:"max_aspect_ratio"`
	MaxPromptLength int      `json:"max_prompt_length"`
}

func OpenAIImageStandardOptions(modelName string) *ImageStandardOptions {
	version := 0
	switch modelName {
	case "gpt-image-2", "gpt-image-2-2026-04-21":
		version = 2
	case "gpt-image-2.5-sunburst", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst-2026-09-08", "gpt-image-2.5-flare-2026-09-08":
		version = 25
	default:
		return nil
	}
	p := &ImageStandardOptions{
		Source:        "https://developers.openai.com/api/docs/guides/image-generation",
		Sizes:         []string{"auto", "1024x1024", "1536x1024", "1024x1536", "1536x864", "864x1536", "2048x2048", "2048x1152", "1152x2048", "3840x2160", "2160x3840"},
		Qualities:     []string{"auto", "low", "medium", "high"},
		OutputFormats: []string{"png", "jpeg", "webp"}, Backgrounds: []string{"auto", "opaque", "transparent"},
		MaxOutputs: 10, MaxReferences: 16, MaxInputSizeMB: 50, SupportsMask: true,
		MinPixels: 655360, MaxPixels: 8294400, MaxEdge: 3840, SizeMultiple: 16, MaxAspectRatio: 3, MaxPromptLength: 32000,
	}
	if version == 25 {
		p.Qualities = append(p.Qualities, "xhigh", "max")
		p.SupportsMarkup = true
	}
	return p
}

func OpenAIImageParameters(modelName string, edit bool) *ModelIntegrationParameters {
	standard := OpenAIImageStandardOptions(modelName)
	if standard == nil {
		return nil
	}
	p := &ModelIntegrationParameters{
		Resolutions:     []string{"1K", "2K", "4K"},
		Ratios:          []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "21:9", "2:1", "1:2", "3:1", "1:3", "5:4", "4:5"},
		ResolutionField: "resolution", RatioField: "ratio", MaxImageSizeMB: 50, ImageOptions: standard,
	}
	if edit {
		p.ImageField, p.MinImages, p.MaxImages = "image[]", 1, standard.MaxReferences
	}
	return p
}

func ValidateOpenAIImageSize(size string) error {
	if size == "" || size == "auto" {
		return nil
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return fmt.Errorf("size must be auto or WIDTHxHEIGHT")
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid image width")
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid image height")
	}
	if w <= 0 || h <= 0 || w > 3840 || h > 3840 || w%16 != 0 || h%16 != 0 || max(w, h) > 3*min(w, h) || w*h < 655360 || w*h > 8294400 {
		return fmt.Errorf("size requires edges divisible by 16, maximum edge 3840, aspect ratio 1:3 to 3:1, and 655360 to 8294400 pixels")
	}
	return nil
}

// Preserve the gateway's existing resolution/ratio inputs while translating to
// the official size field. Explicit size always wins; no preset can overwrite it.
func OpenAIImagePresetSize(resolution, ratio string) (string, error) {
	resolution = strings.ToUpper(resolution)
	if resolution == "" {
		resolution = "1K"
	}
	if ratio == "" {
		ratio = "1:1"
	}
	targets := map[string]int{"1K": 1536, "2K": 2048, "4K": 3840}
	edge, ok := targets[resolution]
	if !ok {
		return "", fmt.Errorf("unsupported image resolution: %s", resolution)
	}
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid image ratio")
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 || w > 1000 {
		return "", fmt.Errorf("invalid image ratio")
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 || h > 1000 || max(w, h) > 3*min(w, h) {
		return "", fmt.Errorf("image ratio must be between 1:3 and 3:1")
	}
	a, b := w, h
	for b != 0 {
		a, b = b, a%b
	}
	w, h = w/a, h/a
	if w == h && resolution == "1K" {
		edge = 1024
	}
	steps := min(edge/(max(w, h)*16), int(math.Sqrt(8294400/float64(w*h*256))))
	size := fmt.Sprintf("%dx%d", w*16*steps, h*16*steps)
	return size, ValidateOpenAIImageSize(size)
}
