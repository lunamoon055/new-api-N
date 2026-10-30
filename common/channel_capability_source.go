package common

import "github.com/QuantumNous/new-api/constant"

// Exact origins prevent lookalike hosts or a provider name in a URL path from
// opting an unrelated channel into a supplier-specific documentation schema.
func ChannelCapabilityProvider(channelType int, baseURL string) string {
	switch channelType {
	case constant.ChannelTypeV6K:
		return "v6k"
	case constant.ChannelTypeZongheng:
		return "zongheng"
	}
	if channelType != constant.ChannelTypeOpenAI && channelType != constant.ChannelTypeSora {
		return ""
	}
	if IsBaseURLHost(baseURL, "suanliai.top") {
		return "suanliai"
	}
	if IsBaseURLHost(baseURL, "api.meaicc.com") {
		return "meaicc"
	}
	return ""
}
