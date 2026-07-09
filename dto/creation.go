package dto

import "github.com/QuantumNous/new-api/constant"

type CreationModel struct {
	ID                     string                  `json:"id"`
	Description            string                  `json:"description,omitempty"`
	ManualDescription      string                  `json:"manual_description,omitempty"`
	Icon                   string                  `json:"icon,omitempty"`
	Tags                   []string                `json:"tags,omitempty"`
	VendorID               int                     `json:"vendor_id,omitempty"`
	Cost                   *CreationModelCost      `json:"cost,omitempty"`
	Metadata               *CreationModelMetadata  `json:"metadata,omitempty"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
}

type CreationModelMetadata struct {
	Provider                         string   `json:"provider,omitempty"`
	ID                               string   `json:"id,omitempty"`
	UpstreamModelID                  string   `json:"upstream_model_id,omitempty"`
	Name                             string   `json:"name,omitempty"`
	Description                      string   `json:"description,omitempty"`
	Type                             string   `json:"type,omitempty"`
	Category                         string   `json:"category,omitempty"`
	ModelType                        string   `json:"model_type,omitempty"`
	Endpoint                         string   `json:"endpoint,omitempty"`
	LegacyEndpoint                   string   `json:"legacy_endpoint,omitempty"`
	Billing                          string   `json:"billing,omitempty"`
	Resolutions                      []string `json:"resolutions,omitempty"`
	Ratios                           []string `json:"ratios,omitempty"`
	AspectRatios                     []string `json:"aspect_ratios,omitempty"`
	Sizes                            []string `json:"sizes,omitempty"`
	Durations                        []int    `json:"durations,omitempty"`
	MaxPromptLength                  int      `json:"max_prompt_length,omitempty"`
	MaxImages                        int      `json:"max_images,omitempty"`
	MaxVideos                        int      `json:"max_videos,omitempty"`
	MaxAudios                        int      `json:"max_audios,omitempty"`
	MaxMediaFiles                    int      `json:"max_media_files,omitempty"`
	MaxImageSizeMB                   int      `json:"max_image_size_mb,omitempty"`
	MaxVideoSizeMB                   int      `json:"max_video_size_mb,omitempty"`
	MaxAudioSizeMB                   int      `json:"max_audio_size_mb,omitempty"`
	MinReferenceVideoDurationSeconds int      `json:"min_reference_video_duration_seconds,omitempty"`
	MaxReferenceVideoDurationSeconds int      `json:"max_reference_video_duration_seconds,omitempty"`
	ConcurrencyOptions               []int    `json:"concurrency_options,omitempty"`
	MediaInputs                      []string `json:"media_inputs,omitempty"`
}

type CreationModelCost struct {
	BillingMode           string             `json:"billing_mode"`
	InputPricePerMillion  *float64           `json:"input_price_per_million,omitempty"`
	OutputPricePerMillion *float64           `json:"output_price_per_million,omitempty"`
	RequestPrice          *float64           `json:"request_price,omitempty"`
	RequestQuota          *int               `json:"request_quota,omitempty"`
	VideoBillingMode      string             `json:"video_billing_mode,omitempty"`
	VideoResolutionPrices map[string]float64 `json:"video_resolution_prices,omitempty"`
	VideoResolutionQuotas map[string]int     `json:"video_resolution_quotas,omitempty"`
	GroupRatio            float64            `json:"group_ratio,omitempty"`
}

type CreationModelGroup struct {
	Mode   string          `json:"mode"`
	Models []CreationModel `json:"models"`
}

type CreationVendor struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type CreationModelCatalog struct {
	Modes   []CreationModelGroup `json:"modes"`
	Vendors []CreationVendor     `json:"vendors"`
}
