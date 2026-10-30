package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestConflictingChannelCapabilitiesDoNotPublishFirstChannelsLimits(t *testing.T) {
	first := PricingMappedEndpoint{Type: "openai-video", Kind: "video", Path: "/v1/videos", Method: "POST", Parameters: &common.ModelIntegrationParameters{
		DurationField: "duration", Durations: []int{5, 10}, RatioField: "ratio", Ratios: []string{"16:9"}, ImageField: "images", MaxImages: 30, MaxVideos: 10,
	}}
	doc := *first.Parameters
	doc.ImageField = "referenceImages"
	first.DocumentationFields = &doc
	second := first
	p := *first.Parameters
	p.MaxImages = 9
	p.MaxVideos = 0
	p.Ratios = []string{"9:16"}
	second.Parameters = &p
	merged := appendPricingEndpoint([]PricingMappedEndpoint{first}, second)
	require.Len(t, merged, 1)
	require.True(t, merged[0].Parameters.CapabilitiesUnknown)
	require.Empty(t, merged[0].Parameters.Durations)
	require.Zero(t, merged[0].Parameters.MaxImages)
	require.Equal(t, common.DefaultVideoAspectRatios(), merged[0].Parameters.Ratios)
	require.True(t, merged[0].DocumentationFields.CapabilitiesUnknown)
	require.Equal(t, "referenceImages", merged[0].DocumentationFields.ImageField)
	require.Equal(t, "multiple_channels", merged[0].CapabilitySource)
	require.Equal(t, 30, first.Parameters.MaxImages, "merging does not mutate the original channel profile")
}
