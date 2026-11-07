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

func MegaBYAIImageParameters(modelName string) *ModelIntegrationParameters {
	return OpenAIImageParameters(modelName, false)
}

func MegaBYAIImageEditParameters(modelName string) *ModelIntegrationParameters {
	return OpenAIImageParameters(modelName, true)
}

// Shared by initial selection, affinity and retry. This is the set of image
// routes integrated for each supplier, not an inference from model metadata.
func IsSyncImageChannelCompatible(baseURL, modelName string) bool {
	return !IsBaseURLHost(baseURL, LinkskyProviderHost) || !IsLinkskyAsyncImageModelName(modelName)
}
