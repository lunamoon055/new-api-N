package billing_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReferencePricesPreserveZeroValidateAndCopy(t *testing.T) {
	old, _ := common.Marshal(GetVideoReferencePricesCopy())
	t.Cleanup(func() { _ = ApplyAuxiliaryConfig(VideoReferencePricesField, string(old)) })
	require.NoError(t, ApplyAuxiliaryConfig(VideoReferencePricesField, `{"model":{"720p":0,"4K":0.12}}`))
	prices, ok := GetVideoReferencePrices("model")
	require.True(t, ok)
	require.Equal(t, float64(0), prices["720p"])
	require.Equal(t, 0.12, prices["4k"])
	prices["720p"] = 5
	again, _ := GetVideoReferencePrices("model")
	require.Zero(t, again["720p"])
	for _, raw := range []string{`null`, `[]`, `{"model":{"720p":-1}}`, `{"model":{"invalid":1}}`, `{"model":{"720p":1,"720":2}}`} {
		require.Error(t, ApplyAuxiliaryConfig(VideoReferencePricesField, raw))
		prices, _ = GetVideoReferencePrices("model")
		require.Zero(t, prices["720p"])
	}
}
