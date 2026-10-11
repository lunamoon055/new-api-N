package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnabledAbilityIncludesChannelMapping(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/pricing-mapping.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&Channel{}, &Ability{}))
	previous := DB
	DB = database
	t.Cleanup(func() { DB = previous })

	mapping := `{"Grok":"grok-imagine-video-1.5"}`
	require.NoError(t, database.Create(&Channel{Id: 42, Type: constant.ChannelTypeOpenAI, ModelMapping: &mapping}).Error)
	require.NoError(t, database.Create(&Ability{Model: "Grok", Group: "default", ChannelId: 42, Enabled: true}).Error)
	abilities, err := GetAllEnableAbilityWithChannels()
	require.NoError(t, err)
	require.Len(t, abilities, 1)
	require.NotNil(t, abilities[0].ModelMapping)
	require.Equal(t, mapping, *abilities[0].ModelMapping)
}

func TestMappedPricingEndpointFollowsFinalUpstreamModel(t *testing.T) {
	mapping := `{"Grok":"video-alias","video-alias":"grok-imagine-video-1.5"}`
	target, mapped := resolvePricingMappedModel("Grok", &mapping)
	require.True(t, mapped)
	require.Equal(t, "grok-imagine-video-1.5", target)

	endpoints := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, target, nil)
	require.Len(t, endpoints, 1)
	endpoint := endpoints[0]
	require.Equal(t, "video", endpoint.Kind)
	require.Equal(t, "/v1/videos", endpoint.Path)
	require.Equal(t, "/v1/videos/{task_id}", endpoint.QueryPath)
	require.Equal(t, string(constant.EndpointTypeOpenAIVideo), endpoint.Type)

	imageMapping := `{"public-image":"gpt-image-2"}`
	imageTarget, mapped := resolvePricingMappedModel("public-image", &imageMapping)
	require.True(t, mapped)
	imageEndpoints := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, imageTarget, nil)
	require.NotEmpty(t, imageEndpoints)
	imageEndpoint := imageEndpoints[0]
	require.Equal(t, "image", imageEndpoint.Kind)
	require.Equal(t, "/v1/images/generations", imageEndpoint.Path)
}

func TestPricingIntegrationParametersIncludeMappedLimitsWithoutUpstreamNames(t *testing.T) {
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "TTP-grok", "Grok", nil)
	require.Len(t, endpoints, 1)
	require.Equal(t, []string{"480p", "720p"}, endpoints[0].Parameters.Resolutions)
	require.Equal(t, 6, endpoints[0].Parameters.Durations[0])
	require.Equal(t, 7, endpoints[0].Parameters.MaxImages)
	encoded, err := common.Marshal(endpoints)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "TTP-grok")
	image := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "midjourney-默认1次4张1K", nil)
	require.Equal(t, "/v1/images/generations", image[0].Path)
	require.Equal(t, "size", image[0].Parameters.ResolutionField)
	require.Equal(t, []string{"1024x1024"}, image[0].Parameters.Resolutions)
	native := getPricingIntegrationEndpoints(constant.ChannelTypeGemini, "nano-banana2", nil)
	require.Equal(t, "text", native[0].Kind)
	require.Contains(t, native[0].Path, "generateContent")
}

func TestMappedPricingEndpointsForOnlineVideoCatalog(t *testing.T) {
	tests := []struct {
		public   string
		upstream string
	}{
		{"官方h3-1080p", "特价/官方h3-1080p"},
		{"官方h3-2k", "特价/官方h3-2k"},
		{"wan3.0(图片+音频参考)", "特价/wan3.0-image"},
		{"sd2.5(满血原生过脸30-10-10/480P)", "全能sd2.5(满血原生过脸30-10-10/480P)"},
		{"sd2.0(9-3-3/4-15秒原生不卡脸720P)", "003系列/sd2.0(9-3-3/4-15秒原生不卡脸720P)"},
		{"sd2.5-720p(9图30秒小几率卡脸内置彩绘过脸)", "特价稳定sd2.5-720p-ch1(9图30秒小几率卡脸内置彩绘过脸)"},
		{"Seedance2.5-c", "sd-2.5-c1"},
	}
	for _, tt := range tests {
		t.Run(tt.public, func(t *testing.T) {
			raw, err := common.Marshal(map[string]string{tt.public: tt.upstream})
			require.NoError(t, err)
			mapping := string(raw)
			target, mapped := resolvePricingMappedModel(tt.public, &mapping)
			require.True(t, mapped)
			require.Equal(t, tt.upstream, target)
			// Both the original upstream ID and its public alias must agree.
			original := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, target, nil)
			public := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, target, tt.public, nil)
			require.Equal(t, original, public)
			require.Len(t, public, 1)
			require.Equal(t, "video", public[0].Kind)
			require.Equal(t, "/v1/videos", public[0].Path)
			require.Equal(t, "/v1/videos/{task_id}", public[0].QueryPath)
		})
	}
}

func TestOpaqueMappedVideoPreservesPublicProtocolAndExplicitMetadata(t *testing.T) {
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "provider-internal-id", "Seedance2.5-c", nil)
	require.Len(t, endpoints, 1)
	require.Equal(t, "/v1/videos", endpoints[0].Path)

	metadata := map[string]*Model{
		"provider-internal-id": {Endpoints: `{"openai-video":{"path":"/v1/video/generations"}}`},
	}
	endpoints = getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "provider-internal-id", "Seedance2.5-c", metadata)
	require.Equal(t, "/v1/video/generations", endpoints[0].Path)
	// A recognized upstream protocol also outranks the public-name fallback.
	endpoints = getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "minimax-h3-2k", "官方h3-2k", nil)
	require.Equal(t, "/v1/video/async-generations", endpoints[0].Path)

	endpoints = getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "gpt-4o", "public-chat", nil)
	require.Equal(t, "text", endpoints[0].Kind)
	require.Equal(t, "/v1/chat/completions", endpoints[0].Path)
	endpoints = getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "wan3.0-1080p", nil)
	require.Equal(t, "/v1/video/async-generations", endpoints[0].Path)
}

func TestMappedPricingEndpointHonorsConfiguredUpstreamEndpoint(t *testing.T) {
	metadata := map[string]*Model{
		"upstream": {Endpoints: `{"openai-video":{"path":"/v1/video/generations","method":"POST","query_path":"/v1/video/generations/{task_id}"}}`},
	}
	endpoints := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "upstream", metadata)
	require.Len(t, endpoints, 1)
	endpoint := endpoints[0]
	require.Equal(t, "/v1/video/generations", endpoint.Path)
	require.Equal(t, "/v1/video/generations/{task_id}", endpoint.QueryPath)
	require.Equal(t, "GET", endpoint.QueryMethod)
	metadata["upstream"].Endpoints = `{"openai":{"path":"/v1/chat/completions"},"openai-video":{"path":"/v1/video/generations"}}`
	endpoints = getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "upstream", metadata)
	require.Len(t, endpoints, 2)
	require.Equal(t, "/v1/chat/completions", endpoints[0].Path)
	require.Equal(t, "/v1/video/generations", endpoints[1].Path)
}

func TestMappedPricingEndpointRejectsCyclesAndKeepsDistinctRoutes(t *testing.T) {
	cycle := `{"alias":"next","next":"alias"}`
	resolved, mapped := resolvePricingMappedModel("alias", &cycle)
	require.False(t, mapped)
	require.Equal(t, "alias", resolved)

	first := PricingMappedEndpoint{Type: "openai-video", Path: "/v1/videos", Method: "POST"}
	second := PricingMappedEndpoint{Type: "openai-video", Path: "/v1/video/async-generations", Method: "POST"}
	endpoints := appendPricingEndpoint(nil, first)
	endpoints = appendPricingEndpoint(endpoints, first)
	endpoints = appendPricingEndpoint(endpoints, second)
	require.Equal(t, []PricingMappedEndpoint{first, second}, endpoints)
}

func TestPricingPublishesMappedRouteUnderPublicModelName(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/pricing-route.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&Channel{}, &Ability{}, &Model{}, &Vendor{}))
	previousDB := DB
	previousPricing := pricingMap
	previousVendors := vendorsList
	previousEndpointMap := supportedEndpointMap
	previousEndpointTypes := modelSupportEndpointTypes
	previousGroups := modelEnableGroups
	previousQuotaTypes := modelQuotaTypeMap
	previousUpdated := lastGetPricingTime
	DB = database
	t.Cleanup(func() {
		DB = previousDB
		pricingMap = previousPricing
		vendorsList = previousVendors
		supportedEndpointMap = previousEndpointMap
		modelSupportEndpointTypes = previousEndpointTypes
		modelEnableGroups = previousGroups
		modelQuotaTypeMap = previousQuotaTypes
		lastGetPricingTime = previousUpdated
	})

	mapping := `{"Grok":"TTP-grok"}`
	require.NoError(t, database.Create(&Channel{Id: 43, Type: constant.ChannelTypeOpenAI, ModelMapping: &mapping}).Error)
	require.NoError(t, database.Create(&Ability{Model: "Grok", Group: "default", ChannelId: 43, Enabled: true}).Error)
	require.NoError(t, database.Create(&Ability{Model: "TTP-grok", Group: "default", ChannelId: 43, Enabled: true}).Error)
	updatePricing()
	require.Len(t, pricingMap, 2)
	byName := make(map[string]Pricing, len(pricingMap))
	for _, pricing := range pricingMap {
		byName[pricing.ModelName] = pricing
	}
	public := byName["Grok"]
	original := byName["TTP-grok"]
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, public.SupportedEndpointTypes)
	require.Equal(t, original.SupportedEndpointTypes, public.SupportedEndpointTypes)
	require.Len(t, public.MappedEndpoints, 1)
	require.Equal(t, "/v1/videos", public.MappedEndpoints[0].Path)
	require.Equal(t, 6, public.MappedEndpoints[0].ExampleDuration)
	require.Equal(t, "480p", public.MappedEndpoints[0].ExampleResolution)
	encoded, err := common.Marshal(public)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"model_name":"Grok"`)
	require.Contains(t, string(encoded), `"mapped_endpoints"`)
	require.False(t, strings.Contains(string(encoded), "TTP-grok"))
}
