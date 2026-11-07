package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestMegaBYAIImageContractIsOriginAndModelScoped(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		require.True(t, IsMegaBYAIImageChannel(constant.ChannelTypeOpenAI, "https://newapi.megabyai.cc/v1/", name))
		require.True(t, IsSyncImageChannelCompatible("https://newapi.megabyai.cc", name))
	}
	for _, host := range []string{"https://api.xjzapi.cc", "https://newapi.megabyai.cc.evil.example", "https://example.com/newapi.megabyai.cc"} {
		require.False(t, IsMegaBYAIImageChannel(constant.ChannelTypeOpenAI, host, "gpt-image-2"))
		require.True(t, IsSyncImageChannelCompatible(host, "gpt-image-2"))
	}
	require.False(t, IsMegaBYAIImageChannel(constant.ChannelTypeGemini, "https://newapi.megabyai.cc", "gpt-image-2"))
	require.False(t, IsMegaBYAIImageModel("gpt-image-2-other"))
	p := MegaBYAIImageParameters("gpt-image-2")
	require.False(t, p.CapabilitiesUnknown)
	require.Equal(t, "resolution", p.ResolutionField)
	require.Equal(t, "ratio", p.RatioField)
	require.Empty(t, p.ImageField)
	edit := MegaBYAIImageEditParameters("gpt-image-2")
	require.Equal(t, "image[]", edit.ImageField)
	require.Equal(t, 1, edit.MinImages)
	require.Equal(t, 16, edit.MaxImages)
	require.False(t, IsIntegrationCapabilityUnknown(edit, "max_images"))
	require.False(t, edit.CapabilitiesUnknown)
}
