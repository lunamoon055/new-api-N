package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpenAIImageOfficialBaseline(t *testing.T) {
	base := OpenAIImageStandardOptions("gpt-image-2")
	require.Equal(t, 16, base.MaxReferences)
	require.Equal(t, 10, base.MaxOutputs)
	require.NotContains(t, base.Qualities, "xhigh")
	for _, name := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		p := OpenAIImageStandardOptions(name)
		require.Contains(t, p.Qualities, "xhigh")
		require.Contains(t, p.Qualities, "max")
		require.True(t, p.SupportsMarkup)
		for _, ratio := range OpenAIImageParameters(name, false).Ratios {
			for _, tier := range []string{"1K", "2K", "4K"} {
				size, err := OpenAIImagePresetSize(tier, ratio)
				require.NoError(t, err, tier+"/"+ratio)
				require.NoError(t, ValidateOpenAIImageSize(size))
			}
		}
	}
	for _, name := range []string{"gpt-image-2-other", "gpt-image-2.5", "gpt-image-2.6-flare", "nano-banana2"} {
		require.Nil(t, OpenAIImageStandardOptions(name))
	}
	for _, size := range []string{"auto", "1536x864", "3840x2160", "2160x3840", "2880x2880"} {
		require.NoError(t, ValidateOpenAIImageSize(size))
	}
	for _, size := range []string{"4096x4096", "1025x1024", "512x512", "3840x3840", "4000x1000", "-1x1024", "foo"} {
		require.Error(t, ValidateOpenAIImageSize(size), size)
	}
}
