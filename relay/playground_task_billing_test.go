package relay

import (
	"github.com/QuantumNous/new-api/constant"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestEnsureTaskBillingConfigured(t *testing.T) {
	ratio_setting.InitRatioSettings()

	unpricedModel := "unpriced-playground-video-model"
	require.False(t, ratio_setting.HasConfiguredModelRatio(unpricedModel))

	tests := []struct {
		name      string
		info      *relaycommon.RelayInfo
		priceData types.PriceData
		mode      string
		wantError bool
	}{
		{
			name:      "unpriced playground video is blocked before pre-consumption",
			info:      &relaycommon.RelayInfo{IsPlayground: true, OriginModelName: unpricedModel},
			priceData: types.PriceData{ModelRatio: 37.5},
			wantError: true,
		},
		{
			name:      "normal API retains self-use fallback",
			info:      &relaycommon.RelayInfo{OriginModelName: unpricedModel},
			priceData: types.PriceData{ModelRatio: 37.5},
		},
		{
			name:      "unpriced XJZ video is blocked outside the playground",
			info:      &relaycommon.RelayInfo{OriginModelName: "特价sd2.0-933-过脸"},
			priceData: types.PriceData{ModelRatio: 37.5},
			wantError: true,
		},
		{
			name:      "mapped XJZ video is blocked outside the playground",
			info:      &relaycommon.RelayInfo{OriginModelName: unpricedModel, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "特价sd2.0-933-过脸"}},
			priceData: types.PriceData{ModelRatio: 37.5},
			wantError: true,
		},
		{
			name:      "explicit video price is allowed",
			info:      &relaycommon.RelayInfo{IsPlayground: true, OriginModelName: unpricedModel},
			priceData: types.PriceData{UsePrice: true, ModelPrice: 0.25},
		},
		{
			name:      "configured resolution tier price is allowed after tier application",
			info:      &relaycommon.RelayInfo{OriginModelName: "特价sd2.0-933-过脸"},
			priceData: types.PriceData{UsePrice: true, ModelPrice: 0.25},
			mode:      billing_setting.VideoBillingModeFixed,
		},
		{
			name:      "per-call video cannot multiply its price by seconds",
			info:      &relaycommon.RelayInfo{OriginModelName: "特价sd2.0-933-过脸"},
			priceData: types.PriceData{UsePrice: true, ModelPrice: 0.25},
			mode:      billing_setting.VideoBillingModeDynamic,
			wantError: true,
		},
		{
			name:      "per-second resolution tier price is allowed",
			info:      &relaycommon.RelayInfo{OriginModelName: "特价sd2.0-720p-933-过脸"},
			priceData: types.PriceData{UsePrice: true, ModelPrice: 0.25},
			mode:      billing_setting.VideoBillingModeTieredSeconds,
		},
		{
			name:      "configured model ratio is allowed",
			info:      &relaycommon.RelayInfo{IsPlayground: true, OriginModelName: "gpt-4"},
			priceData: types.PriceData{ModelRatio: 15},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ensureTaskBillingConfigured(test.info, test.priceData, test.mode)
			if test.wantError {
				require.ErrorContains(t, err, test.info.OriginModelName)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewXJZPerCallPricesCannotUsePerSecondBilling(t *testing.T) {
	for _, modelName := range []string{"grok-1.5-700", "dola-sd-2.5", "sd2.5-720p-ch1", "quanneng2.0", "viraldance900", "viraldance933-fast", "viraldance2.5-15-stable"} {
		info := &relaycommon.RelayInfo{OriginModelName: modelName}
		price := types.PriceData{UsePrice: true, ModelPrice: 1}
		require.Error(t, ensureTaskBillingConfigured(info, price, billing_setting.VideoBillingModeTieredSeconds))
		require.NoError(t, ensureTaskBillingConfigured(info, price, billing_setting.VideoBillingModeFixed))
		require.NoError(t, ensureTaskBillingConfigured(info, price, billing_setting.VideoBillingModeTieredRequest))
	}
}

func TestZonghengBillingUsesItsChannelContract(t *testing.T) {
	info := &relaycommon.RelayInfo{OriginModelName: "特价sd2.0-933-过脸", ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeZongheng}}
	require.Error(t, ensureTaskBillingConfigured(info, types.PriceData{}, billing_setting.VideoBillingModeDynamic))
	require.NoError(t, ensureTaskBillingConfigured(info, types.PriceData{UsePrice: true, ModelPrice: 0.25}, billing_setting.VideoBillingModeDynamic))
}
