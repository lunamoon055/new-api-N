package common

import "strings"

// ModelIntegrationParameters describes public request fields, never channel IDs
// or upstream mapping tables. The same XJZ spec also validates relay requests.
type ModelIntegrationParameters struct {
	// CapabilitiesUnknown distinguishes an unpublished profile from explicit
	// zero reference limits. Consumers must not interpret unknown zeros as unsupported.
	CapabilitiesUnknown bool `json:"capabilities_unknown,omitempty"`

	OmitRatio          bool     `json:"omit_ratio,omitempty"`
	Durations          []int    `json:"durations,omitempty"`
	Resolutions        []string `json:"resolutions,omitempty"`
	Ratios             []string `json:"ratios,omitempty"`
	DurationField      string   `json:"duration_field,omitempty"`
	RatioField         string   `json:"ratio_field,omitempty"`
	ResolutionField    string   `json:"resolution_field,omitempty"`
	RequestResolution  string   `json:"request_resolution,omitempty"`
	ImageField         string   `json:"image_field,omitempty"`
	VideoField         string   `json:"video_field,omitempty"`
	AudioField         string   `json:"audio_field,omitempty"`
	MinImages          int      `json:"min_images,omitempty"`
	MaxImages          int      `json:"max_images"`
	MaxVideos          int      `json:"max_videos"`
	MaxAudios          int      `json:"max_audios"`
	AudioNeedsImage    bool     `json:"audio_needs_image,omitempty"`
	AudioRequiresImage bool     `json:"audio_requires_image,omitempty"`
	AllowFrames        bool     `json:"allow_frames,omitempty"`
	MaxImageSizeMB     int      `json:"max_image_size_mb,omitempty"`
	MaxVideoSizeMB     int      `json:"max_video_size_mb,omitempty"`
	MaxAudioSizeMB     int      `json:"max_audio_size_mb,omitempty"`
}

// DefaultVideoAspectRatios is the site's full ratio menu for video models
// without a published range (operator policy, 2026-10-26).
func DefaultVideoAspectRatios() []string {
	return []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"}
}

// WithVideoRatioDefaults keeps explicit model ranges and fills only missing
// ratios. Copy the hints so cached upstream capabilities remain unchanged.
func WithVideoRatioDefaults(parameters *ModelIntegrationParameters) *ModelIntegrationParameters {
	if parameters == nil {
		return nil
	}
	result := *parameters
	if len(result.Ratios) == 0 {
		result.Ratios = DefaultVideoAspectRatios()
	}
	if result.RatioField == "" {
		result.RatioField = "ratio"
	}
	result.OmitRatio = false
	return &result
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
	// Classification only: keep the original public/upstream ID in requests.
	if strings.HasPrefix(name, "seedance") {
		name = strings.Join(strings.Fields(name), "")
	}
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
	if parameters := GetSiteVideoIntegrationParameters(modelName); parameters != nil {
		return parameters
	}
	if spec, ok := XJZVideoSpecs[strings.TrimSpace(modelName)]; ok {
		durations := append([]int(nil), spec.Durations...)
		if len(durations) == 0 {
			durations = integrationDurationRange(spec.MinDuration, spec.MaxDuration)
		}
		resolutions := append([]string(nil), spec.Resolutions...)
		if spec.Resolution != "" {
			resolutions = []string{strings.ToLower(spec.Resolution)}
		}
		return WithVideoRatioDefaults(&ModelIntegrationParameters{
			Durations: durations, Resolutions: resolutions, Ratios: spec.Ratios,
			DurationField: "duration", RatioField: spec.RatioParameter(), ResolutionField: spec.ResolutionParameter(), RequestResolution: spec.Resolution,
			ImageField: spec.ImageField, VideoField: spec.VideoField, AudioField: spec.AudioField,
			MinImages: spec.MinImages, MaxImages: spec.MaxImages, MaxVideos: spec.MaxVideos, MaxAudios: spec.MaxAudios,
			AudioNeedsImage: spec.AudioNeedsImage, AllowFrames: spec.AllowFrames,
			AudioRequiresImage: spec.AudioRequiresImage,
			MaxImageSizeMB:     10, MaxVideoSizeMB: 10, MaxAudioSizeMB: 10,
		})
	}
	name := NormalizeGatewayVideoModelName(modelName)
	p := &ModelIntegrationParameters{
		Durations: integrationDurationRange(4, 15), Resolutions: []string{"720p"},
		Ratios: []string{"16:9", "9:16", "1:1"}, DurationField: "duration", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "referenceImages", VideoField: "referenceVideos", AudioField: "referenceAudios", MaxImages: 9, MaxVideos: 3, MaxAudios: 3,
	}
	switch {
	case name == "005系列/sd2.5(30-10-10/5-30秒/720p)":
		// Suanliai's exact 005 model, catalog and /docs checked 2026-10-27.
		p.Durations = integrationDurationRange(5, 30)
		p.Ratios = nil // Unpublished: use the operator-approved full ratio menu.
		p.MaxImages, p.MaxVideos, p.MaxAudios = 30, 10, 10
	case name == "w3-c1":
		// Meaicc's public catalog and creation form, checked 2026-10-27.
		p.Durations = integrationDurationRange(5, 30)
		p.Resolutions = []string{"720p", "1080p"}
		p.Ratios = []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}
		p.MaxImages, p.MaxVideos, p.MaxAudios = 10, 5, 5
		p.AllowFrames = true
	case name == "seedance2.0:x", name == "seedance2.0:x1":
		p.Durations = integrationDurationRange(5, 15)
	case name == "sd-2.5-c1", name == "seedance2.5-c", name == "seedance-2.5":
		p.Durations = integrationDurationRange(4, 29)
		if name == "sd-2.5-c1" {
			p.Durations = integrationDurationRange(4, 30)
		}
		p.Resolutions = []string{"480p", "720p"}
		p.MaxImages, p.MaxVideos, p.MaxAudios = 10, 0, 0
		// The generic Seedance 2.5 contract has separate multimodal limits.
		if name == "seedance-2.5" {
			p.MaxImages, p.MaxVideos, p.MaxAudios = 30, 10, 10
		}
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
		// Suanli's image variant supports 2-30 seconds and three explicit tiers.
		p.Durations = integrationDurationRange(2, 30)
		p.Resolutions = []string{"720p", "480p", "1080p"}
		p.MinImages, p.MaxImages, p.MaxVideos, p.MaxAudios = 1, 10, 0, 5
		p.AudioNeedsImage = true
	default:
		return nil
	}
	return WithVideoRatioDefaults(p)
}

// GetSiteVideoIntegrationParameters preserves the exact public contracts in the
// site's model notes (2026-10-24). Public aliases must not inherit a different
// duration or reference limit merely because their upstream IDs are opaque.
func GetSiteVideoIntegrationParameters(modelName string) *ModelIntegrationParameters {
	p := &ModelIntegrationParameters{
		Durations: integrationDurationRange(5, 15), Resolutions: []string{"720p"},
		Ratios:        []string{"16:9", "9:16", "1:1", "4:3", "3:4"},
		DurationField: "duration", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "referenceImages", VideoField: "referenceVideos", AudioField: "referenceAudios",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3, AllowFrames: true,
	}
	switch strings.TrimSpace(modelName) {
	case "Seedance2.0-c":
		p.Durations = integrationDurationRange(4, 15)
	case "Seedance2.0-fast-c", "Seedance2.0-fast2-c", "Seedance2.0:720（秒）":
	case "Seedance2.5:480（秒）", "Seedance2.5:720（秒）", "Seedance2.5:720（条）":
		p.Durations = integrationDurationRange(5, 30)
		p.MaxImages, p.MaxVideos, p.MaxAudios = 30, 10, 10
		if strings.TrimSpace(modelName) == "Seedance2.5:480（秒）" {
			// The note says 720P, but this model's existing resolution tier is 480p.
			p.Resolutions = []string{"480p"}
		}
	case "无限制-Video":
		p.Durations = integrationDurationRange(4, 12)
		p.MaxImages, p.MaxVideos, p.MaxAudios = 5, 0, 3
		p.Resolutions, p.Ratios = nil, nil
		p.ResolutionField, p.RatioField = "", ""
		p.OmitRatio, p.AllowFrames = true, false
	default:
		return nil
	}
	return WithVideoRatioDefaults(p)
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
	if strings.HasPrefix(name, "gpt-image-2.5") {
		maxImages = 16
	}
	if name == "nano-banana-pro" {
		maxImages = 10
	}
	if name == "nano-banana2" {
		maxImages = 14
	}
	return &ModelIntegrationParameters{
		Resolutions: []string{"1K", "2K", "4K"}, Ratios: []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"},
		RatioField: "aspect_ratio", ResolutionField: "output_resolution", ImageField: "images", MaxImages: maxImages,
	}
}
