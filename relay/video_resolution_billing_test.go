package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
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

func TestApplyVideoResolutionTierPriceSupports1KAnd2K(t *testing.T) {
	tests := []struct {
		name       string
		resolution string
		mode       string
		price      float64
		seconds    float64
		wantQuota  int
	}{
		{
			name:       "1K per request",
			resolution: "1K",
			mode:       billing_setting.VideoBillingModeTieredRequest,
			price:      0.03,
			seconds:    8,
			wantQuota:  int(0.03 * common.QuotaPerUnit),
		},
		{
			name:       "2K per second",
			resolution: "2K",
			mode:       billing_setting.VideoBillingModeTieredSeconds,
			price:      0.05,
			seconds:    8,
			wantQuota:  int(0.05 * common.QuotaPerUnit * 8),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(nil)
			c.Set("task_request", relaycommon.TaskSubmitReq{
				Model:      "video-priced",
				Resolution: test.resolution,
				Duration:   int(test.seconds),
			})
			info := &relaycommon.RelayInfo{
				OriginModelName: "video-priced",
				PriceData: types.PriceData{
					GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
					OtherRatios:    map[string]float64{"seconds": test.seconds},
				},
			}
			priceData := info.PriceData
			prices := map[string]float64{
				"1k": 0.03,
				"2k": 0.05,
			}

			err := applyVideoResolutionTierPrice(c, info, &priceData, test.mode, prices)
			require.NoError(t, err)
			require.NoError(t, applyTaskOtherRatiosToQuota(&priceData, test.mode, false))
			require.Equal(t, test.price, priceData.ModelPrice)
			require.Equal(t, test.wantQuota, priceData.Quota)
		})
	}
}

func TestResolveTaskBillingResolutionSupports1KAnd2KSources(t *testing.T) {
	tests := []struct {
		name      string
		request   relaycommon.TaskSubmitReq
		modelName string
		expected  string
	}{
		{
			name:     "1K resolution",
			request:  relaycommon.TaskSubmitReq{Resolution: "1K"},
			expected: "1k",
		},
		{
			name:     "1K size",
			request:  relaycommon.TaskSubmitReq{Size: "1024x1024"},
			expected: "1k",
		},
		{
			name:     "2K size",
			request:  relaycommon.TaskSubmitReq{Size: "2560x1440"},
			expected: "2k",
		},
		{
			name:      "model suffix",
			modelName: "minimax-h3-2K",
			expected:  "2k",
		},
		{
			name:      "fixed resolution model",
			modelName: "(线路3)minimax-h3",
			expected:  "2k",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(nil)
			c.Set("task_request", test.request)
			info := &relaycommon.RelayInfo{OriginModelName: test.modelName}

			require.Equal(t, test.expected, resolveTaskBillingResolution(c, info))
		})
	}
}

func TestApplyImageResolutionTierPriceUsesOutputResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.video_billing_mode":      `{"gpt-image-2":"tiered_request"}`,
		"billing_setting.video_resolution_prices": `{"gpt-image-2":{"1k":0.06,"2k":0.07,"4k":0.09}}`,
	}))

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
	}
	request := &dto.ImageRequest{
		Model:            "gpt-image-2",
		OutputResolution: []byte(`"2K"`),
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 0.5},
	}

	err := ApplyImageResolutionTierPrice(info, request, &priceData)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.07, priceData.ModelPrice)
	require.Equal(t, int(0.07*common.QuotaPerUnit*0.5), priceData.QuotaToPreConsume)
	require.Equal(t, priceData, info.PriceData)
}

func TestApplyImageResolutionTierPriceErrorsWhenRequestedTierIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.video_billing_mode":      `{"gpt-image-2":"tiered_request"}`,
		"billing_setting.video_resolution_prices": `{"gpt-image-2":{"1k":0.06}}`,
	}))

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
	}
	request := &dto.ImageRequest{OutputResolution: []byte(`"4K"`)}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}

	err := ApplyImageResolutionTierPrice(info, request, &priceData)
	require.ErrorContains(t, err, "4k")
}

func TestResolveImageBillingResolution(t *testing.T) {
	tests := []struct {
		name     string
		request  *dto.ImageRequest
		expected string
	}{
		{
			name:     "output resolution has priority",
			request:  &dto.ImageRequest{OutputResolution: []byte(`"2K"`), Size: "1024x1024"},
			expected: "2k",
		},
		{
			name:     "official portrait size maps to 1K",
			request:  &dto.ImageRequest{Size: "1024x1536"},
			expected: "1k",
		},
		{
			name:     "missing resolution uses image default",
			request:  &dto.ImageRequest{},
			expected: "1k",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, resolveImageBillingResolution(test.request))
		})
	}
}
