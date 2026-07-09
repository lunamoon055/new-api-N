package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyVideoResolutionTierPriceUsesRequestResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:      "video-priced",
		Resolution: "720p",
		Duration:   8,
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-priced",
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 0.9},
			OtherRatios:    map[string]float64{"seconds": 8},
		},
	}
	priceData := info.PriceData
	prices := map[string]float64{
		"480p":  0.01,
		"720p":  0.02,
		"1080p": 0.04,
		"4k":    0.08,
	}

	err := applyVideoResolutionTierPrice(c, info, &priceData, billing_setting.VideoBillingModeTieredSeconds, prices)

	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.02, priceData.ModelPrice)
	require.Equal(t, int(0.02*common.QuotaPerUnit*0.9), priceData.Quota)
}

func TestApplyVideoResolutionTierPriceErrorsWhenResolutionPriceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:      "video-priced",
		Resolution: "4k",
		Duration:   8,
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-priced",
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	priceData := info.PriceData

	err := applyVideoResolutionTierPrice(c, info, &priceData, billing_setting.VideoBillingModeTieredRequest, map[string]float64{
		"720p": 0.02,
	})

	require.ErrorContains(t, err, "4k")
}
