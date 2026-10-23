package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withDownstreamPrices(t *testing.T) {
	t.Helper()
	prices, ratios, completion := ratio_setting.ModelPrice2JSONString(), ratio_setting.ModelRatio2JSONString(), ratio_setting.CompletionRatio2JSONString()
	quotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(prices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(ratios))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(completion))
		common.QuotaPerUnit = quotaPerUnit
		model.InvalidatePricingCache()
	})
	common.QuotaPerUnit = 500000
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"zz-price-image":0.04,"zz-price-fixed":0.3,"zz-price-video":0.2,"zz-price-free":0}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"zz-price-chat":1.5,"zz-price-audio":2,"zz-price-embed":0.1}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"zz-price-chat":4,"zz-price-audio":3}`))
	withVideoResolutionBillingConfig(t, map[string]string{
		"zz-price-fixed": "fixed", "zz-price-tier": "tiered_seconds", "zz-price-image-tier": "tiered_request",
	}, map[string]map[string]float64{
		"zz-price-tier": {"720p": 0.2, "1080p": 0.4}, "zz-price-image-tier": {"1k": 0},
	})
	withTieredBillingConfig(t, map[string]string{"zz-price-expr": "tiered_expr", "zz-price-empty-expr": "tiered_expr"}, map[string]string{
		"zz-price-expr": `tier("base", p * 2 + c * 8)`, "zz-price-empty-expr": "  ",
	})
}

func TestDownstreamPricingUnitsAndCurrency(t *testing.T) {
	withDownstreamPrices(t)
	video := []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	for _, tc := range []struct {
		name       string
		endpoints  []constant.EndpointType
		unit       string
		configured bool
		price      float64
	}{
		{"zz-price-chat", nil, "million_tokens", true, 0},
		{"zz-price-audio", nil, "million_tokens", true, 0},
		{"zz-price-embed", nil, "million_tokens", true, 0},
		{"zz-price-image", nil, "request", true, 0.28},
		{"zz-price-video", video, "second", true, 1.4},
		{"zz-price-fixed", video, "request", true, 2.1},
		{"zz-price-tier", video, "second", true, 0},
		{"zz-price-image-tier", nil, "request", true, 0},
		{"zz-price-free", nil, "request", true, 0},
		{"zz-price-expr", nil, "expression", true, 0},
		{"zz-price-empty-expr", nil, "unconfigured", false, 0},
		{"zz-price-unset", nil, "unconfigured", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := model.Pricing{ModelName: tc.name, SupportedEndpointTypes: tc.endpoints, ModelRatio: 37.5}
			got := service.BuildDownstreamModelPricing(item, 7)
			require.Equal(t, tc.unit, got.PriceUnit)
			require.Equal(t, tc.configured, got.PriceConfigured)
			require.InDelta(t, tc.price, got.ModelPrice, 1e-12)
			if tc.name == "zz-price-chat" {
				require.InDelta(t, 21, *got.InputPrice, 1e-12)
				require.InDelta(t, 84, *got.OutputPrice, 1e-12)
				require.Equal(t, 1.5, got.ModelRatio)
			}
			if tc.name == "zz-price-tier" {
				require.InDelta(t, 1.4, got.VideoResolutionPrices["720p"], 1e-12)
				require.InDelta(t, 2.8, got.VideoResolutionPrices["1080p"], 1e-12)
			}
			if tc.name == "zz-price-image-tier" {
				require.Contains(t, got.VideoResolutionPrices, "1k")
				require.Zero(t, got.VideoResolutionPrices["1k"])
			}
			if tc.name == "zz-price-expr" {
				require.Equal(t, `tier("base", p * 2 + c * 8)`, got.BillingExpr)
				require.Equal(t, "USD", got.ExpressionCurrency)
				require.Nil(t, got.InputPrice)
			}
			if !tc.configured {
				require.Zero(t, got.ModelRatio)
				require.Nil(t, got.InputPrice)
			}
		})
	}
	common.QuotaPerUnit = 1000000
	got := service.BuildDownstreamModelPricing(model.Pricing{ModelName: "zz-price-chat"}, 1)
	require.Equal(t, 1.5, *got.InputPrice)
}

type downstreamPriceResponse struct {
	Success    bool                             `json:"success"`
	Object     string                           `json:"object"`
	Currency   string                           `json:"currency"`
	GroupRatio map[string]float64               `json:"group_ratio"`
	Data       []service.DownstreamModelPricing `json:"data"`
}

func TestGetModelPricingFiltersTokenAndGroupWithoutExposingFallback(t *testing.T) {
	withDownstreamPrices(t)
	db := setupModelListControllerTestDB(t)
	model.InvalidatePricingCache()
	t.Cleanup(model.InvalidatePricingCache)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-price-chat", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-image", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-audio", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-embed", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-video", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-unset", ChannelId: 1, Enabled: true},
		{Group: "private", Model: "zz-price-secret", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-disabled", ChannelId: 1, Enabled: false},
	}).Error)
	request := func(query string, limited bool, allow map[string]bool, group string) downstreamPriceResponse {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/pricing"+query, nil)
		common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
		common.SetContextKey(ctx, constant.ContextKeyTokenGroup, group)
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, limited)
		common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, allow)
		GetModelPricing(ctx)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
		var got downstreamPriceResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &got))
		require.True(t, got.Success)
		require.Equal(t, "list", got.Object)
		return got
	}
	got := request("", false, nil, "")
	require.Len(t, got.Data, 6) // all modalities, no private/disabled models
	require.Equal(t, "USD", got.Currency)
	for _, item := range got.Data {
		require.Equal(t, []string{"default"}, item.EnableGroup)
		if item.ModelName == "zz-price-unset" {
			require.False(t, item.PriceConfigured)
			require.Zero(t, item.ModelRatio)
		}
	}
	got = request("", true, map[string]bool{"zz-price-image": true}, "")
	require.Len(t, got.Data, 1)
	require.Equal(t, "zz-price-image", got.Data[0].ModelName)
	require.Empty(t, request("", true, nil, "").Data)
	require.Empty(t, request("?model=zz-price-secret", false, nil, "").Data)
	require.Empty(t, request("", false, nil, "empty-group").Data)
	got = request("?model=zz-price-chat", false, nil, "")
	require.Len(t, got.Data, 1)

	// Automatic routing may use several groups, with a user-specific discount.
	autoGroups := setting.AutoGroups2JsonString()
	usableGroups := setting.UserUsableGroups2JSONString()
	groupRatios := ratio_setting.GroupGroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(autoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(usableGroups))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(groupRatios))
	})
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","private"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","private":"Private"}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"default":{"private":0.6}}`))
	got = request("", false, nil, "auto")
	require.Len(t, got.Data, 7)
	require.Equal(t, 0.6, got.GroupRatio["private"])

	oldRate := operation_setting.USDExchangeRate
	operation_setting.USDExchangeRate = 7
	t.Cleanup(func() { operation_setting.USDExchangeRate = oldRate })
	got = request("?currency=CNY&model=zz-price-image", false, nil, "")
	require.InDelta(t, .28, got.Data[0].ModelPrice, 1e-12)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/pricing?currency=EUR", nil)
	GetModelPricing(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	operation_setting.USDExchangeRate = 0
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/pricing?currency=CNY", nil)
	GetModelPricing(ctx)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestModelPricingAPIKeyAuthenticationIsReadOnly(t *testing.T) {
	withDownstreamPrices(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	model.InvalidatePricingCache()
	t.Cleanup(model.InvalidatePricingCache)
	user := model.User{Id: 190876, Username: "pricing-test", Group: "default", Status: common.UserStatusEnabled, Quota: 100000}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "pricingtestkey123456789", Name: "pricing-test", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 1000, ModelLimitsEnabled: true, ModelLimits: "zz-price-chat"}
	require.NoError(t, db.Create(&token).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-price-chat", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-price-image", ChannelId: 1, Enabled: true},
	}).Error)
	router := gin.New()
	router.GET("/v1/pricing", middleware.TokenAuth(), GetModelPricing)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/pricing", nil))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	request := httptest.NewRequest(http.MethodGet, "/v1/pricing", nil)
	request.Header.Set("Authorization", "Bearer sk-"+token.Key)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	var got downstreamPriceResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &got))
	require.Len(t, got.Data, 1)
	require.Equal(t, "zz-price-chat", got.Data[0].ModelName)
	var unchanged model.Token
	require.NoError(t, db.First(&unchanged, token.Id).Error)
	require.Equal(t, token.RemainQuota, unchanged.RemainQuota)
	require.Zero(t, unchanged.UsedQuota)
}
