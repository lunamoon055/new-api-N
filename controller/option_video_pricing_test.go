package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVideoPricingOptionEndpointSavesUnmappedAndUnboundModels(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	model.DB = db
	before := &billing_setting.VideoPricingConfig{Modes: billing_setting.GetVideoBillingModeCopy(), Generation: billing_setting.GetVideoResolutionPricesCopy(), Reference: billing_setting.GetVideoReferencePricesCopy()}
	previousOptions := map[string]string{}
	common.OptionMapRWMutex.RLock()
	for _, key := range []string{"billing_setting.video_billing_mode", "billing_setting.video_resolution_prices", "billing_setting.video_reference_prices"} {
		if value, ok := common.OptionMap[key]; ok {
			previousOptions[key] = value
		}
	}
	common.OptionMapRWMutex.RUnlock()
	t.Cleanup(func() {
		model.DB = previousDB
		billing_setting.ApplyVideoPricingConfig(before)
		common.OptionMapRWMutex.Lock()
		for _, key := range []string{"billing_setting.video_billing_mode", "billing_setting.video_resolution_prices", "billing_setting.video_reference_prices"} {
			delete(common.OptionMap, key)
			if value, ok := previousOptions[key]; ok {
				common.OptionMap[key] = value
			}
		}
		common.OptionMapRWMutex.Unlock()
		model.InvalidatePricingCache()
		_ = sqlDB.Close()
	})
	request := func(key, value string) bool {
		body, err := common.Marshal(map[string]string{"key": key, "value": value})
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		UpdateOption(ctx)
		var response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		if response.Success {
			require.Equal(t, http.StatusOK, recorder.Code)
		}
		return response.Success
	}
	// Reproduce the full-form save without any channel or mapping table.
	raw := `{"video_billing_mode":{"SD2.0":"tiered_seconds","Seedance2.0:(无视频参考)":"tiered_seconds","custom-native-model":"tiered_seconds"},"video_resolution_prices":{"SD2.0":{"720p":0.1},"Seedance2.0:(无视频参考)":{"720p":0.2},"custom-native-model":{"720p":0.3}},"video_reference_prices":{"SD2.0":{"720p":0.2},"Seedance2.0:(无视频参考)":{"720p":0},"custom-native-model":{"720p":0.4}}}`
	require.True(t, request(model.VideoPricingConfigOptionKey, raw))
	for name, rate := range map[string]float64{"SD2.0": 0.2, "Seedance2.0:(无视频参考)": 0, "custom-native-model": 0.4} {
		mode, _, refs := billing_setting.GetVideoPricingSnapshot(name)
		require.Equal(t, "tiered_seconds", mode)
		require.Contains(t, refs, "720p")
		require.Equal(t, rate, refs["720p"])
	}
	var options []model.Option
	require.NoError(t, db.Find(&options).Error)
	require.Len(t, options, 3)
	require.True(t, request("billing_setting.video_reference_prices", `{"custom-native-model":{"720p":0.4}}`))
	require.False(t, request("billing_setting.video_reference_prices", `{"custom-native-model":{"720p":-1}}`))
	_, _, refs := billing_setting.GetVideoPricingSnapshot("custom-native-model")
	require.Equal(t, 0.4, refs["720p"])
}
