package common

import "strings"

// ModelIntegrationParameters describes public request fields, never channel IDs
// or upstream mapping tables. The same XJZ spec also validates relay requests.
type ModelIntegrationParameters struct {
	Durations         []int    `json:"durations,omitempty"`
	Resolutions       []string `json:"resolutions,omitempty"`
	Ratios            []string `json:"ratios,omitempty"`
	DurationField     string   `json:"duration_field,omitempty"`
	RatioField        string   `json:"ratio_field,omitempty"`
	ResolutionField   string   `json:"resolution_field,omitempty"`
	RequestResolution string   `json:"request_resolution,omitempty"`
	ImageField        string   `json:"image_field,omitempty"`
	VideoField        string   `json:"video_field,omitempty"`
	AudioField        string   `json:"audio_field,omitempty"`
	MinImages         int      `json:"min_images,omitempty"`
	MaxImages         int      `json:"max_images"`
	MaxVideos         int      `json:"max_videos"`
	MaxAudios         int      `json:"max_audios"`
	AudioNeedsImage   bool     `json:"audio_needs_image,omitempty"`
	AllowFrames       bool     `json:"allow_frames,omitempty"`
}

func integrationDurationRange(minimum, maximum int) []int {
	values := make([]int, 0, maximum-minimum+1)
	for value := minimum; value <= maximum; value++ {
		values = append(values, value)
	}
	return values
}

func NormalizeGatewayVideoModelName(modelName string) string {
	name := strings.ToLower(strings.TrimSpace(modelName))
	for _, prefix := range []string{"特价/", "003系列/"} {
		name = strings.TrimPrefix(name, prefix)
	}
	if strings.HasPrefix(name, "全能sd2") {
		name = strings.TrimPrefix(name, "全能")
	}
	if strings.HasPrefix(name, "特价稳定sd2") {
		name = strings.TrimPrefix(name, "特价稳定")
	}
	return name
}

func GetVideoIntegrationParameters(modelName string) *ModelIntegrationParameters {
	if spec, ok := XJZVideoSpecs[strings.TrimSpace(modelName)]; ok {
		durations := append([]int(nil), spec.Durations...)
		if len(durations) == 0 {
			durations = integrationDurationRange(spec.MinDuration, spec.MaxDuration)
		}
		resolutions := append([]string(nil), spec.Resolutions...)
		if spec.Resolution != "" {
			resolutions = []string{strings.ToLower(spec.Resolution)}
		}
		return &ModelIntegrationParameters{
			Durations: durations, Resolutions: resolutions, Ratios: spec.Ratios,
			DurationField: "duration", RatioField: "ratio", ResolutionField: "resolution", RequestResolution: spec.Resolution,
			ImageField: spec.ImageField, VideoField: spec.VideoField, AudioField: spec.AudioField,
			MinImages: spec.MinImages, MaxImages: spec.MaxImages, MaxVideos: spec.MaxVideos, MaxAudios: spec.MaxAudios,
			AudioNeedsImage: spec.AudioNeedsImage, AllowFrames: spec.AllowFrames,
		}
	}
	name := NormalizeGatewayVideoModelName(modelName)
	p := &ModelIntegrationParameters{
		Durations: integrationDurationRange(4, 15), Resolutions: []string{"720p"},
		Ratios: []string{"16:9", "9:16", "1:1"}, DurationField: "duration", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "referenceImages", VideoField: "referenceVideos", AudioField: "referenceAudios", MaxImages: 9, MaxVideos: 3, MaxAudios: 3,
	}
	switch {
	case name == "sd-2.5-c1", name == "seedance2.5-c", name == "seedance-2.5":
		p.Durations = integrationDurationRange(4, 29)
		if name == "sd-2.5-c1" {
			p.Durations = integrationDurationRange(4, 30)
		}
		p.Resolutions = []string{"480p", "720p"}
		p.MaxImages, p.MaxVideos, p.MaxAudios = 30, 10, 10
		p.AllowFrames = true
	case strings.HasPrefix(name, "sd-2-c"), strings.HasPrefix(name, "seedance2.0"), strings.HasPrefix(name, "seedance-2.0"):
		p.Ratios = []string{"16:9", "9:16", "1:1", "4:3", "3:4"}
		p.AllowFrames = true
	case name == "官方h3-1080p", name == "官方h3-2k":
		p.Durations = integrationDurationRange(5, 15)
		p.Resolutions = []string{strings.TrimPrefix(name, "官方h3-")}
		p.MaxVideos = 0
	case strings.HasPrefix(name, "sd2.5"):
		p.Durations = integrationDurationRange(4, 30)
		if strings.Contains(name, "480p") {
			p.Resolutions = []string{"480p"}
		}
		if strings.Contains(name, "1080p") {
			p.Resolutions = []string{"1080p"}
		}
		if strings.Contains(name, "30-10-10") {
			p.MaxImages, p.MaxVideos, p.MaxAudios = 30, 10, 10
		}
	case strings.HasPrefix(name, "sd2.0"):
		if strings.Contains(name, "12秒") {
			p.Durations = []int{12}
			p.MaxVideos = 0
		}
	case strings.HasPrefix(name, "wan3.0(") || strings.HasPrefix(name, "wan3.0-image"):
		// The upstream image variant requires an image; do not submit a text-only request.
		p.MinImages, p.MaxVideos = 1, 0
		p.Resolutions, p.ResolutionField = nil, ""
	default:
		return nil
	}
	return p
}

func GetImageIntegrationParameters(modelName string) *ModelIntegrationParameters {
	name := strings.ToLower(strings.TrimSpace(modelName))
	// XJZ publishes this exact synchronous model under /v1/images/generations.
	// One request returns its four-image grid at a fixed 1024px size.
	if name == "midjourney-默认1次4张1k" {
		return &ModelIntegrationParameters{Resolutions: []string{"1024x1024"}, ResolutionField: "size"}
	}
	if !strings.HasPrefix(name, "gpt-image-2") && name != "nano-banana-pro" && name != "nano-banana2" {
		return nil
	}
	maxImages := 17
	if name == "nano-banana-pro" {
		maxImages = 10
	}
	if name == "nano-banana2" {
		maxImages = 14
	}
	return &ModelIntegrationParameters{
		Resolutions: []string{"1K", "2K", "4K"}, Ratios: []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"},
		RatioField: "aspect_ratio", ResolutionField: "resolution", ImageField: "images", MaxImages: maxImages,
	}
}
