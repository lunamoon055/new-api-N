package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func capabilityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previous := model.DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.ChannelModelCapability{}, &model.Model{}, &model.Vendor{}, &model.Ability{}))
	model.DB = db
	model.InvalidatePricingCache()
	t.Cleanup(func() { model.InvalidatePricingCache(); model.DB = previous; sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	return db
}

func TestV6KCapabilityNotesPreservePartialUnknownAndExplicitZero(t *testing.T) {
	p := parseV6KCapabilityNote("30图10音/480P/720P//5-30秒/稳定50-70%")
	require.Equal(t, 30, p.MaxImages)
	require.Equal(t, 10, p.MaxAudios)
	require.True(t, common.IsIntegrationCapabilityUnknown(p, "max_videos"))
	require.False(t, common.IsIntegrationCapabilityUnknown(p, "max_images"))
	require.Equal(t, []string{"480p", "720p"}, p.Resolutions)
	require.Len(t, p.Durations, 26)
	require.Equal(t, 5, p.Durations[0])
	require.Equal(t, 30, p.Durations[25])
	zero := parseV6KCapabilityNote("9图0音0视/720P/固定15秒/")
	require.False(t, common.IsIntegrationCapabilityUnknown(zero, "max_videos"))
	require.Zero(t, zero.MaxVideos)
	require.Equal(t, []int{15}, zero.Durations)
	list := parseV6KCapabilityNote("9图/720P/5-10-15/")
	require.Equal(t, []int{5, 10, 15}, list.Durations)
	invalid := parseV6KCapabilityNote("稳定50-70% / ¥3 / 99999999999999999图 / 0-9999999秒")
	require.Empty(t, invalid.Durations)
	require.True(t, common.IsIntegrationCapabilityUnknown(invalid, "max_images"))
	unsupported := parseV6KCapabilityNote("30图10音/不支持视频参考/720P/5-30秒")
	require.False(t, common.IsIntegrationCapabilityUnknown(unsupported, "max_videos"))
	require.Zero(t, unsupported.MaxVideos)
}

func TestChannelCapabilityRefreshTracksMappingsPersistsAndRetainsDataOnFailure(t *testing.T) {
	db := capabilityTestDB(t)
	fail := false
	note := "30图10音10视/720P/5-30秒"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/pricing", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"), "public catalog must not receive channel secrets")
		if fail {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		body, _ := common.Marshal(map[string]any{"success": true, "data": []map[string]any{
			{"model_name": "new-model", "description": note, "supported_endpoint_types": []string{"openai-video"}},
			{"model_name": "other-model", "description": "5图0音0视/480P/固定10秒/", "supported_endpoint_types": []string{"openai-video"}},
		}})
		_, _ = w.Write(body)
	}))
	defer server.Close()
	base, mapping := server.URL, `{"public-video":"new-model"}`
	channel := model.Channel{Type: constant.ChannelTypeV6K, Key: "test-secret", BaseURL: &base, Models: "public-video", ModelMapping: &mapping, Status: 1}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "public-video", ChannelId: channel.Id, Enabled: true}).Error)
	result, err := RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedModels)
	require.Empty(t, result.UnconfirmedModels)
	pricing := model.GetPricing()
	require.Len(t, pricing, 1)
	ep := pricing[0].MappedEndpoints[0]
	require.Equal(t, "public-video", pricing[0].ModelName)
	require.Equal(t, "/v1/videos", ep.Path)
	require.Equal(t, "upstream_catalog", ep.CapabilitySource)
	require.Equal(t, 30, ep.Parameters.MaxImages)
	require.Equal(t, "seconds", ep.Parameters.DurationField)
	require.Equal(t, "duration", ep.DocumentationFields.DurationField)
	require.Equal(t, "referenceVideos", ep.DocumentationFields.VideoField)
	encoded, _ := common.Marshal(ep)
	require.NotContains(t, string(encoded), "new-model")
	require.NotContains(t, string(encoded), "test-secret")
	// No mapping change: a supplier update is reflected by the next refresh.
	note = "8图0音0视/720P/5-15秒"
	_, err = RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	require.Equal(t, 8, model.GetPricing()[0].MappedEndpoints[0].Parameters.MaxImages)
	fail = true
	_, err = RefreshChannelCapabilities(context.Background(), channel.Id)
	require.Error(t, err)
	model.InvalidatePricingCache()
	require.Equal(t, 8, model.GetPricing()[0].MappedEndpoints[0].Parameters.MaxImages, "persistent metadata survives a cache reset and an upstream failure")
	fail = false
	// A new mapped ID cannot inherit the previous model's limits before sync.
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("model_mapping", `{"public-video":"other-model"}`).Error)
	model.InvalidatePricingCache()
	require.True(t, model.GetPricing()[0].MappedEndpoints[0].Parameters.CapabilitiesUnknown)
	_, err = RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	require.Equal(t, 5, model.GetPricing()[0].MappedEndpoints[0].Parameters.MaxImages)
	// Credential rotation also invalidates documentation scope.
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("key", "different-secret").Error)
	model.InvalidatePricingCache()
	require.True(t, model.GetPricing()[0].MappedEndpoints[0].Parameters.CapabilitiesUnknown)
}

func TestChannelCapabilityRefreshMissingModelDoesNotReuseAnotherModel(t *testing.T) {
	db := capabilityTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"success":true,"data":[]}`)) }))
	defer server.Close()
	base := server.URL
	channel := model.Channel{Type: constant.ChannelTypeV6K, BaseURL: &base, Models: "unknown-video,unknown-video", Status: 1}
	require.NoError(t, db.Create(&channel).Error)
	result, err := RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"unknown-video"}, result.UnconfirmedModels)
	require.Equal(t, 1, result.UpdatedModels)
	var item model.ChannelModelCapability
	require.NoError(t, db.First(&item).Error)
	var parameters common.ModelIntegrationParameters
	require.NoError(t, common.UnmarshalJsonStr(item.Parameters, &parameters))
	require.True(t, parameters.CapabilitiesUnknown)
	require.Empty(t, parameters.Durations)
}

func TestZonghengDocumentationDoesNotInventOmittedReferenceLimits(t *testing.T) {
	var item ZonghengVideoModel
	require.NoError(t, common.UnmarshalJsonStr(`{"id":"video","capabilities":{"capability_type":"video","max_image_refs":3,"max_video_refs":0}}`, &item))
	p := zonghengDocumentationParameters(item)
	require.Equal(t, 3, p.MaxImages)
	require.Zero(t, p.MaxVideos)
	require.False(t, common.IsIntegrationCapabilityUnknown(p, "max_videos"))
	require.True(t, common.IsIntegrationCapabilityUnknown(p, "max_audios"))
}

func TestPublicCatalogProvidersParseExactIDsAndKeepTextModelsSeparate(t *testing.T) {
	setZonghengTestClient(t, func(req *http.Request) (*http.Response, error) {
		require.Empty(t, req.Header.Get("Authorization"))
		require.NotEmpty(t, req.Header.Get("User-Agent"))
		return zonghengTestResponse(http.StatusOK, `{"success":true,"data":[
		{"model_name":"005系列/sd2.5(30-10-10/5-30秒/720P)","supported_endpoint_types":["openai"]},
		{"model_name":"sd-2.5-c3","description":"30图10视频10音频，720p，4-30s，全比例","supported_endpoint_types":["openai"]},
		{"model_name":"w3-c1","description":"10图 / 5视频 / 5音频 / 720p/1080p / 5-30s","supported_endpoint_types":["openai"]},
		{"model_name":"gpt-image-2","supported_endpoint_types":["openai"]},
		{"model_name":"claude-sonnet-5","supported_endpoint_types":["openai"]}]}`), nil
	})
	suan, err := fetchPublicVideoCapabilityCatalog(context.Background(), "https://suanliai.top/v1", "", "suanliai")
	require.NoError(t, err)
	p := suan["005系列/sd2.5(30-10-10/5-30秒/720P)"]
	require.NotNil(t, p)
	require.Equal(t, 30, p.MaxImages)
	require.Equal(t, 10, p.MaxVideos)
	require.Empty(t, p.UnknownFields)
	require.NotContains(t, suan, "gpt-image-2")
	require.NotContains(t, suan, "claude-sonnet-5")
	mea, err := fetchPublicVideoCapabilityCatalog(context.Background(), "https://api.meaicc.com", "", "meaicc")
	require.NoError(t, err)
	require.Equal(t, 30, mea["sd-2.5-c3"].MaxImages)
	require.Equal(t, 4, mea["sd-2.5-c3"].Durations[0])
	require.Equal(t, 30, mea["sd-2.5-c3"].Durations[26])
	require.Equal(t, []string{"720p", "1080p"}, mea["w3-c1"].Resolutions)
	for _, host := range []string{"https://suanliai.top.other.test", "https://example.test/suanliai.top", "https://api.meaicc.com.other.test"} {
		require.False(t, SupportsChannelCapabilitySync(constant.ChannelTypeOpenAI, host))
	}
	require.True(t, SupportsChannelCapabilitySync(constant.ChannelTypeOpenAI, "https://suanliai.top/v1"))
	require.True(t, SupportsChannelCapabilitySync(constant.ChannelTypeSora, "https://api.meaicc.com"))
}

func TestChannelCapabilityProtocolTemplateSurvivesOpaqueRenameWithoutOldLimits(t *testing.T) {
	db := capabilityTestDB(t)
	setZonghengTestClient(t, func(req *http.Request) (*http.Response, error) {
		return zonghengTestResponse(http.StatusOK, `{"success":true,"data":[
		{"model_name":"sd-2-c1","description":"9图3音3视/720P/5-15秒","supported_endpoint_types":["openai"]},
		{"model_name":"opaque-next","description":"30图10音0视/480P/5-30秒","supported_endpoint_types":["openai-video"]}]}`), nil
	})
	base, mapping := "https://api.meaicc.com", `{"Public":"sd-2-c1"}`
	channel := model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: &base, Models: "Public", ModelMapping: &mapping, Status: 1}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "Public", ChannelId: channel.Id, Enabled: true}).Error)
	_, err := RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	require.Equal(t, 9, model.GetPricing()[0].MappedEndpoints[0].Parameters.MaxImages)
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("model_mapping", `{"Public":"opaque-next"}`).Error)
	model.InvalidatePricingCache()
	ep := model.GetPricing()[0].MappedEndpoints[0]
	require.Equal(t, "/v1/videos", ep.Path)
	require.True(t, ep.Parameters.CapabilitiesUnknown)
	require.Empty(t, ep.Parameters.Durations)
	require.Zero(t, ep.Parameters.MaxImages)
	require.Equal(t, "referenceImages", ep.DocumentationFields.ImageField)
	_, err = RefreshChannelCapabilities(context.Background(), channel.Id)
	require.NoError(t, err)
	ep = model.GetPricing()[0].MappedEndpoints[0]
	require.Equal(t, "/v1/videos", ep.Path)
	require.Equal(t, 30, ep.Parameters.MaxImages)
	require.Equal(t, 0, ep.Parameters.MaxVideos)
	require.False(t, ep.Parameters.CapabilitiesUnknown)
	require.Equal(t, []string{"480p"}, ep.Parameters.Resolutions)
	// An explicit change to a text model must discard the video template.
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("model_mapping", `{"Public":"gpt-4o"}`).Error)
	model.InvalidatePricingCache()
	require.Equal(t, "/v1/chat/completions", model.GetPricing()[0].MappedEndpoints[0].Path)
}
