package common

// XJZVideoSpecs is shared by public integration metadata and relay validation.
type XJZVideoSpec struct {
	MinDuration     int
	MaxDuration     int
	Durations       []int
	Resolution      string
	Resolutions     []string
	Ratios          []string
	ImageField      string
	VideoField      string
	AudioField      string
	MinImages       int
	MaxImages       int
	MaxVideos       int
	MaxAudios       int
	AudioNeedsImage bool
	AllowFrames     bool
}

var xjzWideRatios = []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}
var xjzStandardRatios = []string{"16:9", "9:16", "1:1"}

var XJZVideoSpecs = map[string]XJZVideoSpec{
	"特价dola2.5-900-720P-固定30s": {
		Durations: []int{30}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", MinImages: 1, MaxImages: 9,
	},
	"特价dola-sd2.0-900-720p": {
		Durations: []int{5, 10, 15}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", MinImages: 1, MaxImages: 1,
	},
	"TTP-grok": {
		MinDuration: 6, MaxDuration: 15, Resolutions: []string{"480p", "720p"}, Ratios: xjzStandardRatios,
		ImageField: "images", MaxImages: 7,
	},
	"gz稳定sd2.0-903-720P": {
		MinDuration: 5, MaxDuration: 15, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "images", AudioField: "reference_audios", MaxImages: 9, MaxAudios: 3,
	},
	"Xminimex-h3-2K-933": {
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
		Durations: []int{30}, Resolution: "720p", Ratios: xjzWideRatios,
		ImageField: "reference_images", MaxImages: 10,
	},
	"特价稳sd2.0-933-720p": {
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
		MinDuration: 4, MaxDuration: 15, Resolution: "720p", Ratios: []string{"16:9", "9:16", "1:1", "4:3"},
		ImageField: "image_urls", VideoField: "video_urls", AudioField: "audio_urls",
		MaxImages: 9, MaxVideos: 3, MaxAudios: 3, AudioNeedsImage: true,
	},
	"MiniMaxH3": {
		MinDuration: 4, MaxDuration: 15, Resolution: "2K", Ratios: xjzStandardRatios,
		ImageField: "image_urls", AudioField: "audio_reference",
		MaxImages: 9, MaxAudios: 3, AudioNeedsImage: true, AllowFrames: true,
	},
}
