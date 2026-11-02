package model

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestXJZNativeImageEndpointsFollowMappedContract(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		endpoints := resolvePricingIntegrationEndpoints(constant.ChannelTypeOpenAI, name+"-原生4K", name, nil)
		require.Len(t, endpoints, 2)
		require.Equal(t, "/v1/images/generations", endpoints[0].Path)
		require.Zero(t, endpoints[0].Parameters.MaxImages)
		require.Equal(t, "/v1/images/edits", endpoints[1].Path)
		require.Equal(t, "image[]", endpoints[1].Parameters.ImageField)
		require.Equal(t, 20, endpoints[1].Parameters.MaxImages)
		require.Len(t, endpoints[1].Parameters.Ratios, 8)
		require.Equal(t, "ratio", endpoints[0].Parameters.RatioField)
	}
	old := resolvePricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "gpt-image-2大并发", "gpt-image-2", nil)
	require.Len(t, old, 1)
	require.Equal(t, "images", old[0].Parameters.ImageField)
}
