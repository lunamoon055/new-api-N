package common

import "github.com/QuantumNous/new-api/constant"

const MegaBYAIProviderHost = "newapi.megabyai.cc"

// Exact IDs from the public catalog and the channel's authenticated model list,
// checked 2026-11-07. Do not apply this contract to other models or origins.
func IsMegaBYAIImageModel(modelName string) bool {
	switch modelName {
	case "gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst":
		return true
	}
	return false
}

func IsMegaBYAIImageChannel(channelType int, baseURL, modelName string) bool {
	return channelType == constant.ChannelTypeOpenAI && IsBaseURLHost(baseURL, MegaBYAIProviderHost) && IsMegaBYAIImageModel(modelName)
}

func MegaBYAIImageParameters() *ModelIntegrationParameters {
	// /docs#images only documents model, prompt and n. Missing reference limits
	// are unknown, not confirmed zero, and must not inherit another supplier's
	// resolution/ratio/reference fields merely because model IDs are identical.
	return &ModelIntegrationParameters{CapabilitiesUnknown: true}
}

func MegaBYAIImageEditParameters() *ModelIntegrationParameters {
	// All three models passed a single-file reference edit on 2026-11-07.
	// One is the currently verified client upload bound, not the upstream maximum.
	return &ModelIntegrationParameters{ImageField: "image[]", MinImages: 1, MaxImages: 1, UnknownFields: []string{"max_images"}}
}

// Shared by initial selection, affinity and retry. This is the set of image
// routes integrated for each supplier, not an inference from model metadata.
func IsSyncImageChannelCompatible(baseURL, modelName string) bool {
	return !IsBaseURLHost(baseURL, LinkskyProviderHost) || !IsLinkskyAsyncImageModelName(modelName)
}
