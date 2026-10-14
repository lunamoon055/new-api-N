package common

// XJZVideoSpecs is shared by public integration metadata and relay validation.
type XJZVideoSpec struct {
	MinDuration              int
	MaxDuration              int
	Durations                []int
	Resolution               string
	Resolutions              []string
	Ratios                   []string
	ImageField               string
	VideoField               string
	AudioField               string
	MinImages                int
	MaxImages                int
	MaxVideos                int
	MaxAudios                int
	AudioNeedsImage          bool
	AllowFrames              bool
	PerCall                  bool
	DefaultDuration          int
	DefaultResolution        string
	RatioField               string
	ResolutionField          string
	AudioRequiresImage       bool
	ReferenceVideoMultiplier float64
}

func (spec XJZVideoSpec) RatioParameter() string {
	if spec.RatioField != "" {
		return spec.RatioField
	}
	return "ratio"
}

func (spec XJZVideoSpec) ResolutionParameter() string {
	if spec.ResolutionField != "" {
		return spec.ResolutionField
	}
	return "resolution"
}

var xjzWideRatios = []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}
var xjzStandardRatios = []string{"16:9", "9:16", "1:1"}

var XJZVideoSpecs = map[string]XJZVideoSpec{
	"特价dola2.5-900-720P-固定30s": {
		PerCall:   true,
		Durations: []int{30}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", MinImages: 1, MaxImages: 9,
	},
	"特价dola-sd2.0-900-720p": {
		PerCall:   true,
		Durations: []int{5, 10, 15}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", MinImages: 1, MaxImages: 1,
	},
	"TTP-grok": {
		PerCall: true,
		// Prefer the verified 720p default; 480p remains an explicit protocol option.
		MinDuration: 6, MaxDuration: 15, Resolutions: []string{"720p", "480p"}, Ratios: xjzStandardRatios,
		ImageField: "images", MaxImages: 7,
	},
	"gz稳定sd2.0-903-720P": {
		PerCall:     true,
		MinDuration: 5, MaxDuration: 15, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", AudioField: "reference_audios", MaxImages: 9, MaxAudios: 3,
	},
	"Xminimex-h3-2K-933": {
		PerCall:     true,
		MinDuration: 5, MaxDuration: 15, Resolution: "2k", Ratios: xjzWideRatios,
		ImageField: "images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3,
	},
	"特价sd2.0-720p-933-过脸": {
		MinDuration: 5, MaxDuration: 15, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3,
	},
	"特价稳sd-2.5-30s-720P可10图": {
		PerCall:   true,
		Durations: []int{30}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "reference_images", MaxImages: 10,
	},
	"特价稳sd2.0-933-720p": {
		PerCall: true,
		// The text says 5-15 seconds while one catalog parameter says 4-15.
		// Use the narrower range until the provider resolves the discrepancy.
		MinDuration: 5, MaxDuration: 15, Resolution: "720P", Ratios: xjzWideRatios,
		ImageField: "reference_images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3,
	},
	"gz稳定sd2.5-720全参过脸": {
		MinDuration: 4, MaxDuration: 30, Resolution: "720P", Ratios: xjzWideRatios,
		ImageField: "reference_images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 30, MaxVideos: 10, MaxAudios: 10,
	},
	"gz稳定sd2.5-480全参过脸": {
		MinDuration: 4, MaxDuration: 30, Resolution: "480P", Ratios: xjzWideRatios,
		ImageField: "reference_images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 30, MaxVideos: 10, MaxAudios: 10,
	},
	"gz稳定sd2.5-1080全参过脸": {
		MinDuration: 4, MaxDuration: 30, Resolution: "1080p", Ratios: xjzWideRatios,
		ImageField: "reference_images", VideoField: "reference_videos", AudioField: "reference_audios",
		MaxImages: 30, MaxVideos: 10, MaxAudios: 10,
	},
	"特价sd2.0-933-过脸": {
		PerCall:     true,
		MinDuration: 4, MaxDuration: 15, Resolution: "720p", Ratios: []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls", VideoField: "video_urls", AudioField: "audio_urls",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3, AudioNeedsImage: true,
	},
	"MiniMaxH3": {
		PerCall:     true,
		MinDuration: 4, MaxDuration: 15, Resolution: "2K", Ratios: xjzStandardRatios,
		ImageField: "image_urls", AudioField: "audio_reference",
		MaxImages: 9, MaxAudios: 3, AudioNeedsImage: true, AllowFrames: true,
	},
	// Public catalog and dedicated video documentation checked on 2026-10-14.
	"minimax-h3-2K-933": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "2k",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		PerCall: true,
	},
	"sd2.5-全参过真人-30-10-10": {
		MinDuration: 5, MaxDuration: 30,
		Resolutions: []string{"480p", "720p", "1080p"},
		Ratios:      []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField:  "reference_images",
		VideoField:  "reference_videos",
		AudioField:  "reference_audios",
		MaxImages:   30, MaxVideos: 10, MaxAudios: 10,
		DefaultResolution: "480p",
	},
	"sd2.0-全参过真人-720p": {
		MinDuration: 5, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
	},
	"grok-1.5-700": {
		MinDuration: 3, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		MaxImages:  7, MaxVideos: 0, MaxAudios: 0,
		PerCall: true,
	},
	"dola-sd-2.5": {
		MinDuration: 5, MaxDuration: 30,
		Resolutions: []string{"480p", "720p", "1080p"},
		Ratios:      []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField:  "reference_images",
		VideoField:  "reference_videos",
		AudioField:  "reference_audios",
		MaxImages:   30, MaxVideos: 10, MaxAudios: 10,
		PerCall:           true,
		DefaultResolution: "480p",
	},
	"dola-sd-2.0-900": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		MaxImages:  9, MaxVideos: 0, MaxAudios: 0,
		PerCall: true,
	},
	"dola-sd-2.0": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 0, MaxAudios: 3,
		PerCall: true,
	},
	"sd-2.0-933-720-fast-原生真人": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "reference_images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		PerCall: true,
	},
	"gz-sd2.5-1080p": {
		MinDuration: 4, MaxDuration: 30,
		Resolution: "1080P",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  30, MaxVideos: 10, MaxAudios: 10,
	},
	"gz-sd2.5-720p": {
		MinDuration: 4, MaxDuration: 30,
		Resolution: "720P",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  30, MaxVideos: 10, MaxAudios: 10,
	},
	"gz-sd2.5-480p": {
		MinDuration: 4, MaxDuration: 30,
		Resolution: "480P",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		MaxImages:  30, MaxVideos: 10, MaxAudios: 0,
	},
	"gz-sd4k": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "4K",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		AudioRequiresImage:       true,
		ReferenceVideoMultiplier: 2,
	},
	"gz-sd1080p": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "1080P",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		AudioRequiresImage:       true,
		ReferenceVideoMultiplier: 2,
	},
	"gz-sd720p": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		AudioRequiresImage:       true,
		ReferenceVideoMultiplier: 2,
	},
	"gz-sd480p": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "480p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		AudioRequiresImage:       true,
		ReferenceVideoMultiplier: 2,
	},
	"sd2.5-720p-ch1": {
		Durations:  []int{30},
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		MaxImages:  9, MaxVideos: 0, MaxAudios: 0,
		PerCall: true,
	},
	"viraldance900": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		MaxImages:  9, MaxVideos: 0, MaxAudios: 0,
		PerCall:         true,
		RatioField:      "aspect_ratio",
		ResolutionField: "size",
	},
	"viraldance933": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		VideoField: "video_urls",
		AudioField: "audio_urls",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		PerCall:         true,
		AudioNeedsImage: true,
	},
	"viraldance933-fast": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		VideoField: "video_urls",
		AudioField: "audio_urls",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		PerCall:         true,
		AudioNeedsImage: true,
	},
	"viraldance933x-fast": {
		MinDuration: 4, MaxDuration: 14,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1"},
		ImageField: "image_url",
		VideoField: "video_url",
		AudioField: "audio_url",
		MaxImages:  1, MaxVideos: 1, MaxAudios: 1,
		RatioField: "aspect_ratio",
	},
	"viraldance2.5-15-stable": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		AudioField: "audio_urls",
		MaxImages:  30, MaxVideos: 0, MaxAudios: 3,
		PerCall:         true,
		DefaultDuration: 15,
	},
	"viraldance2.5-480p-15-stable": {
		MinDuration: 4, MaxDuration: 15,
		Resolution: "480p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		AudioField: "audio_urls",
		MaxImages:  30, MaxVideos: 0, MaxAudios: 3,
		PerCall:         true,
		DefaultDuration: 15,
	},
	"viraldance2.5x-30": {
		MinDuration: 4, MaxDuration: 30,
		Resolution: "720p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		AudioField: "audio_urls",
		MaxImages:  30, MaxVideos: 0, MaxAudios: 3,
	},
	"viraldance2.5x-480p-30": {
		MinDuration: 4, MaxDuration: 30,
		Resolution: "480p",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls",
		AudioField: "audio_urls",
		MaxImages:  30, MaxVideos: 0, MaxAudios: 3,
	},
	"quanneng2.0": {
		Durations:  []int{15},
		Resolution: "720P",
		Ratios:     []string{"16:9", "9:16", "1:1", "4:3", "3:4", "2:3", "3:2", "21:9"},
		ImageField: "images",
		VideoField: "reference_videos",
		AudioField: "reference_audios",
		MaxImages:  9, MaxVideos: 3, MaxAudios: 3,
		PerCall:         true,
		DefaultDuration: 15,
	},
}
