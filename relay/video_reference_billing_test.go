package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestV6KReferenceFeesUsePublicPricesWithOrWithoutMapping(t *testing.T) {
	before := &billing_setting.VideoPricingConfig{Modes: billing_setting.GetVideoBillingModeCopy(), Generation: billing_setting.GetVideoResolutionPricesCopy(), Reference: billing_setting.GetVideoReferencePricesCopy()}
	t.Cleanup(func() { billing_setting.ApplyVideoPricingConfig(before) })
	for _, tc := range []struct{ public, upstream string }{
		{"SD2.0", "SD2.0"}, {"SD2.5", "SD2.5"}, {"custom-native-video", "custom-native-video"},
		{"Seedance2.0:(无视频参考)", "Seedance2.0:(无视频参考)"},
		{"mapped-alias", "SD2.5"}, {"mapped-custom", "custom-upstream-video"}, {"native-without-upstream-metadata", ""},
	} {
		t.Run(tc.public, func(t *testing.T) {
			billing_setting.ApplyVideoPricingConfig(&billing_setting.VideoPricingConfig{
				Modes:      map[string]string{tc.public: "tiered_seconds"},
				Generation: map[string]map[string]float64{tc.public: {"720p": 0.1}},
				Reference:  map[string]map[string]float64{tc.public: {"720p": 0.2}},
			})
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 5, Resolution: "720p"})
			c.Set("v6k_reference_video_seconds", 7.5)
			info := &relaycommon.RelayInfo{OriginModelName: tc.public, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeV6K, UpstreamModelName: tc.upstream}, PriceData: types.PriceData{UsePrice: true, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
			require.NoError(t, applyVideoReferenceBilling(c, info))
			require.NotNil(t, info.PriceData.VideoReferenceBilling)
			require.NoError(t, applyTaskOtherRatiosToQuota(&info.PriceData, "tiered_seconds", false))
			require.Equal(t, int(2*common.QuotaPerUnit), info.PriceData.Quota)
		})
	}
}

func TestReferenceFeeIsAddedOnceAndTaskSnapshotIsFrozen(t *testing.T) {
	oldModes, _ := common.Marshal(billing_setting.GetVideoBillingModeCopy())
	oldPrices, _ := common.Marshal(billing_setting.GetVideoReferencePricesCopy())
	oldGeneration, _ := common.Marshal(billing_setting.GetVideoResolutionPricesCopy())
	t.Cleanup(func() {
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoResolutionPricesField, string(oldGeneration))
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, string(oldModes))
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, string(oldPrices))
	})
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, `{"public-sd":"tiered_seconds"}`))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoResolutionPricesField, `{"public-sd":{"720p":0.1}}`))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, `{"public-sd":{"720p":0.2}}`))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 5, Resolution: "720p"})
	c.Set("v6k_reference_video_seconds", 7.5)
	info := &relaycommon.RelayInfo{OriginModelName: "public-sd", ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeV6K, UpstreamModelName: "SD2.5"}, PriceData: types.PriceData{ModelPrice: 0.1, UsePrice: true, OtherRatios: map[string]float64{"seconds": 5}, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 2}}}
	require.NoError(t, applyVideoReferenceBilling(c, info))
	for i := 0; i < 2; i++ {
		require.NoError(t, applyTaskOtherRatiosToQuota(&info.PriceData, billing_setting.VideoBillingModeTieredSeconds, false))
		require.Equal(t, int(4*common.QuotaPerUnit), info.PriceData.Quota)
	}
	context := taskBillingContext(info)
	saved, err := common.Marshal(context)
	require.NoError(t, err)
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, `{"public-sd":{"720p":9}}`))
	var restored model.TaskBillingContext
	require.NoError(t, common.Unmarshal(saved, &restored))
	require.Equal(t, 0.2, restored.VideoReferenceBilling.ReferencePrice)
	quota, err := restored.VideoReferenceBilling.Quota(common.QuotaPerUnit, restored.GroupRatio)
	require.NoError(t, err)
	require.Equal(t, info.PriceData.Quota, quota)
	require.True(t, restored.PerCallBilling)
	info.PriceData.GroupRatioInfo.GroupRatio = 0
	require.NoError(t, applyTaskOtherRatiosToQuota(&info.PriceData, "tiered_seconds", false))
	require.Zero(t, info.PriceData.Quota)
}

func TestReferenceFeeMissingTierIsNotFreeAndZeroIsExplicit(t *testing.T) {
	oldModes, _ := common.Marshal(billing_setting.GetVideoBillingModeCopy())
	oldPrices, _ := common.Marshal(billing_setting.GetVideoReferencePricesCopy())
	oldGeneration, _ := common.Marshal(billing_setting.GetVideoResolutionPricesCopy())
	t.Cleanup(func() {
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoResolutionPricesField, string(oldGeneration))
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, string(oldModes))
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, string(oldPrices))
	})
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, `{"sd":"tiered_seconds"}`))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoResolutionPricesField, `{"sd":{"720p":0.1}}`))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, `{"sd":{"480p":0.1}}`))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 5, Resolution: "720p"})
	c.Set("v6k_reference_video_seconds", float64(2))
	info := &relaycommon.RelayInfo{OriginModelName: "sd", ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeV6K, UpstreamModelName: "SD2.0"}, PriceData: types.PriceData{ModelPrice: 0.1, UsePrice: true, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
	require.Error(t, applyVideoReferenceBilling(c, info))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, `{"sd":{"720p":0}}`))
	require.NoError(t, applyVideoReferenceBilling(c, info))
	require.NoError(t, applyTaskOtherRatiosToQuota(&info.PriceData, "tiered_seconds", false))
	require.Equal(t, int(0.5*common.QuotaPerUnit), info.PriceData.Quota)
}
