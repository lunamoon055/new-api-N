package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestMegaBYAIImagePricingDoesNotInheritXJZMetadata(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		metadata := map[string]*Model{name: {Endpoints: `{"image-generation":{"path":"/v1/images/generations"},"image-edit":{"path":"/v1/images/edits"}}`}}
		endpoints := getPricingIntegrationEndpointsForChannel(constant.ChannelTypeOpenAI, "https://newapi.megabyai.cc", name, "public-alias", metadata)
		require.Len(t, endpoints, 2)
		require.Equal(t, "/v1/images/generations", endpoints[0].Path)
		require.Equal(t, "image", endpoints[0].Kind)
		require.False(t, endpoints[0].Parameters.CapabilitiesUnknown)
		require.Equal(t, "ratio", endpoints[0].Parameters.RatioField)
		require.Empty(t, endpoints[0].Parameters.ImageField)
		require.Equal(t, "openai_documentation", endpoints[0].CapabilitySource)
		require.Equal(t, "/v1/images/edits", endpoints[1].Path)
		require.Equal(t, "image[]", endpoints[1].Parameters.ImageField)
		require.Equal(t, "openai_documentation", endpoints[1].CapabilitySource)
		encoded, err := common.Marshal(endpoints)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), common.MegaBYAIProviderHost)
		require.NotContains(t, string(encoded), name)

		xjz := getPricingIntegrationEndpointsForChannel(constant.ChannelTypeOpenAI, "https://api.xjzapi.cc", name+"-原生4K", name, metadata)
		require.Len(t, xjz, 2)
		require.Equal(t, "image[]", xjz[1].Parameters.ImageField)
		require.Equal(t, 20, xjz[1].Parameters.MaxImages)
		for _, firstMega := range []bool{true, false} {
			var merged []PricingMappedEndpoint
			if firstMega {
				merged = appendPricingEndpoint(merged, endpoints[0])
				merged = appendPricingEndpoint(merged, xjz[0])
			} else {
				merged = appendPricingEndpoint(merged, xjz[0])
				merged = appendPricingEndpoint(merged, endpoints[0])
			}
			merged = appendPricingEndpoint(merged, xjz[1])
			merged = appendPricingEndpoint(merged, endpoints[1])
			require.Len(t, merged, 2)
			require.False(t, merged[0].Parameters.CapabilitiesUnknown)
			require.Equal(t, "resolution", merged[0].Parameters.ResolutionField)
			require.Equal(t, []string{"1K", "2K", "4K"}, merged[0].Parameters.Resolutions)
			require.ElementsMatch(t, common.OpenAIImageParameters(name, false).Ratios, merged[0].Parameters.Ratios)
			require.NotNil(t, merged[0].Parameters.ImageOptions)
			require.Empty(t, merged[0].Parameters.ImageField)
			require.Equal(t, "multiple_channels", merged[0].CapabilitySource)
			require.Equal(t, "image[]", merged[1].Parameters.ImageField)
			require.Equal(t, 20, merged[1].Parameters.MaxImages)
			require.False(t, common.IsIntegrationCapabilityUnknown(merged[1].Parameters, "max_images"))
			require.Equal(t, "resolution", merged[1].Parameters.ResolutionField)
		}
	}
}

func TestMegaBYAIIntegrationLeavesOtherImageCatalogMergingUnchanged(t *testing.T) {
	first := PricingMappedEndpoint{Type: "image-generation", Kind: "image", Path: "/v1/images/generations", Method: "POST", Parameters: common.XJZNativeImageParameters(false)}
	second := first
	second.Parameters = common.GetImageIntegrationParameters("gpt-image-2")
	merged := appendPricingEndpoint([]PricingMappedEndpoint{first}, second)
	require.Equal(t, first.Parameters, merged[0].Parameters)
}
