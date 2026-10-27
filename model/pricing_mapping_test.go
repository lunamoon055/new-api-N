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

func TestSuanliai005MappedAliasExposesVideoContract(t *testing.T) {
	upstream := "005系列/sd2.5(30-10-10/5-30秒/720P)"
	metadata := map[string]*Model{upstream: {Endpoints: `{"openai":{"path":"/v1/chat/completions"}}`}}
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, upstream, "Seedance2.5:720-S（条）", metadata)
	require.Len(t, endpoints, 1)
	require.Equal(t, "/v1/videos", endpoints[0].Path)
	require.Equal(t, "/v1/videos/{task_id}", endpoints[0].QueryPath)
	require.Equal(t, "video", endpoints[0].Kind)
	p := endpoints[0].Parameters
	require.Equal(t, 5, p.Durations[0])
	require.Equal(t, 30, p.Durations[len(p.Durations)-1])
	require.Equal(t, []string{"720p"}, p.Resolutions)
	require.Equal(t, common.DefaultVideoAspectRatios(), p.Ratios)
	require.Equal(t, 30, p.MaxImages)
	require.Equal(t, 10, p.MaxVideos)
	require.Equal(t, 10, p.MaxAudios)
	require.Equal(t, p, endpoints[0].DocumentationFields)
	require.Nil(t, common.GetVideoIntegrationParameters("005系列/sd2.5-other"))
}

func TestVideoDocumentationFieldsStayStableAcrossProviderMappings(t *testing.T) {
	for _, tc := range []struct {
		channel  int
		upstream string
	}{
		{constant.ChannelTypeV6K, "TTTseadanco2.5/720"},
		{constant.ChannelTypeZongheng, "XXseadanco2.0M"},
		{constant.ChannelTypeOpenAI, "005系列/sd2.5(30-10-10/5-30秒/720P)"},
		{constant.ChannelTypeOpenAI, "sd-2.5-c1"},
		{constant.ChannelTypeOpenAI, "w3-c1"},
	} {
		endpoints := getPricingIntegrationEndpointsForModels(tc.channel, tc.upstream, "public-video", nil)
		require.NotEmpty(t, endpoints)
		ep := endpoints[0]
		require.Equal(t, "/v1/videos", ep.Path)
		doc := ep.DocumentationFields
		require.NotNil(t, doc, tc.upstream)
		require.Equal(t, "duration", doc.DurationField)
		require.Equal(t, "ratio", doc.RatioField)
		require.Equal(t, "resolution", doc.ResolutionField)
		require.Equal(t, "referenceImages", doc.ImageField)
		require.Equal(t, "referenceVideos", doc.VideoField)
		require.Equal(t, "referenceAudios", doc.AudioField)
		require.Equal(t, ep.Parameters.Durations, doc.Durations)
		require.Equal(t, ep.Parameters.MaxAudios, doc.MaxAudios)
		if tc.channel == constant.ChannelTypeV6K {
			require.Equal(t, "seconds", ep.Parameters.DurationField)
			require.Equal(t, "audios", ep.Parameters.AudioField)
		}
	}
}

func TestV6KZHNewModelsExposeVideoContractsWithoutInternalFlags(t *testing.T) {
	count := 0
	for _, id := range common.V6KVideoModelIDs() {
		parameters := common.GetV6KValidatedVideoParameters(id)
		if parameters == nil {
			continue
		}
		count++
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeV6K, id, id, nil)
		require.Len(t, endpoints, 1)
		require.Equal(t, "video", endpoints[0].Kind)
		require.Equal(t, "/v1/videos", endpoints[0].Path)
		require.Equal(t, "/v1/videos/{task_id}", endpoints[0].QueryPath)
		require.Equal(t, parameters, endpoints[0].Parameters)
		encoded, err := common.Marshal(endpoints)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "validate_request")
	}
	require.Equal(t, 11, count)
}

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
	require.Equal(t, "output_resolution", imageEndpoint.Parameters.ResolutionField)
}

func TestEveryXJZVideoCanBeMappedToAnArbitraryPublicAlias(t *testing.T) {
	for target, spec := range common.XJZVideoSpecs {
		t.Run(target, func(t *testing.T) {
			endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, target, "opaque public ID", nil)
			require.Len(t, endpoints, 1)
			require.Equal(t, "video", endpoints[0].Kind)
			require.Equal(t, "/v1/videos", endpoints[0].Path)
			require.Equal(t, spec.ImageField, endpoints[0].Parameters.ImageField)
			require.Contains(t, endpoints[0].Parameters.Durations, endpoints[0].ExampleDuration)
			require.NotEmpty(t, endpoints[0].ExampleResolution)
		})
	}
}

func TestSpacedSeedanceAliasesOverrideOnlyGenericChatFallback(t *testing.T) {
	for _, name := range []string{"Seedance 2.0:X", "Seedance 2.0:X1"} {
		metadata := map[string]*Model{"opaque-provider-id": {Endpoints: `{"openai":{"path":"/v1/chat/completions"}}`}}
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "opaque-provider-id", name, metadata)
		require.Equal(t, "video", endpoints[0].Kind)
		require.Equal(t, "/v1/videos", endpoints[0].Path)
		require.Equal(t, 5, endpoints[0].Parameters.Durations[0])
		metadata["opaque-provider-id"].Endpoints = `{"openai-video":{"path":"/v1/video/generations"}}`
		require.Equal(t, "/v1/video/generations", getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "opaque-provider-id", name, metadata)[0].Path)
	}
}

func TestPricingIntegrationParametersIncludeMappedLimitsWithoutUpstreamNames(t *testing.T) {
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "TTP-grok", "Grok", nil)
	require.Len(t, endpoints, 1)
	require.Equal(t, []string{"720p", "480p"}, endpoints[0].Parameters.Resolutions)
	require.Equal(t, "720p", endpoints[0].ExampleResolution)
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

func TestPricingImageContractUsesOutputResolutionAndProviderReferenceLimits(t *testing.T) {
	for name, maxImages := range map[string]int{
		"gpt-image-2": 17, "gpt-image-2.5-flare": 16, "gpt-image-2.5-sunburst": 16,
		"nano-banana-pro": 10, "nano-banana2": 14,
	} {
		t.Run(name, func(t *testing.T) {
			endpoints := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, name, nil)
			require.Len(t, endpoints, 1)
			require.Equal(t, "/v1/images/generations", endpoints[0].Path)
			require.Equal(t, "output_resolution", endpoints[0].Parameters.ResolutionField)
			require.Equal(t, []string{"1K", "2K", "4K"}, endpoints[0].Parameters.Resolutions)
			require.Equal(t, maxImages, endpoints[0].Parameters.MaxImages)
		})
	}
}

func TestMappedWANImageContractIncludesAllDocumentedResolutions(t *testing.T) {
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "特价/wan3.0-image", "wan3.0(图片+音频参考)", nil)
	require.Len(t, endpoints, 1)
	require.Equal(t, "/v1/videos", endpoints[0].Path)
	p := endpoints[0].Parameters
	require.Equal(t, 2, p.Durations[0])
	require.Equal(t, 30, p.Durations[len(p.Durations)-1])
	require.Equal(t, []string{"720p", "480p", "1080p"}, p.Resolutions)
	require.Equal(t, "resolution", p.ResolutionField)
	require.Equal(t, 1, p.MinImages)
	require.Equal(t, 10, p.MaxImages)
	require.Equal(t, 0, p.MaxVideos)
	require.Equal(t, 5, p.MaxAudios)
	// Other providers and fixed XJZ models retain their own contracts.
	require.Equal(t, []string{"720p"}, common.GetVideoIntegrationParameters("gz稳定sd2.0-903-720P").Resolutions)
}

func TestMeaiccWanMappedAliasExposesVideoContract(t *testing.T) {
	metadata := map[string]*Model{"w3-c1": {Endpoints: `{"openai":{"path":"/v1/chat/completions"}}`}}
	endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "w3-c1", "Wan 3.0", metadata)
	require.Len(t, endpoints, 1)
	require.Equal(t, "video", endpoints[0].Kind)
	require.Equal(t, "/v1/videos", endpoints[0].Path)
	require.Equal(t, "/v1/videos/{task_id}", endpoints[0].QueryPath)
	p := endpoints[0].Parameters
	require.Equal(t, 5, p.Durations[0])
	require.Equal(t, 30, p.Durations[len(p.Durations)-1])
	require.Equal(t, []string{"720p", "1080p"}, p.Resolutions)
	require.Equal(t, 10, p.MaxImages)
	require.Equal(t, 5, p.MaxVideos)
	require.Equal(t, 5, p.MaxAudios)
	require.True(t, p.AllowFrames)
	require.Nil(t, common.GetVideoIntegrationParameters("w3-c2"), "neighboring models must not inherit this contract")
}

func TestMappedSeedance25CContractLimitsImageReferences(t *testing.T) {
	for _, name := range []string{"Seedance2.5-c", "sd-2.5-c1"} {
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, name, "Seedance2.5-c", nil)
		require.Len(t, endpoints, 1)
		p := endpoints[0].Parameters
		require.Equal(t, "referenceImages", p.ImageField)
		require.Equal(t, 10, p.MaxImages)
		require.Zero(t, p.MaxVideos)
		require.Zero(t, p.MaxAudios)
		encoded, err := common.Marshal(p)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"max_videos":0`)
		require.Contains(t, string(encoded), `"max_audios":0`)
	}
	for _, name := range []string{"seedance-2.5", "sd2.5(满血原生过脸30-10-10/480P)"} {
		p := common.GetVideoIntegrationParameters(name)
		require.Equal(t, 30, p.MaxImages)
		require.Equal(t, 10, p.MaxVideos)
		require.Equal(t, 10, p.MaxAudios)
	}
	p := common.GetVideoIntegrationParameters("sd-2-c8")
	require.Equal(t, 9, p.MaxImages)
	require.Equal(t, 3, p.MaxVideos)
	require.Equal(t, 3, p.MaxAudios)
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
	endpoints = getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "gpt-4o", "Seedance2.5-c", nil)
	require.Equal(t, "text", endpoints[0].Kind)
	endpoints = getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "gpt-image-2", "Seedance2.5-c", nil)
	require.Equal(t, "image", endpoints[0].Kind)
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
	require.NoError(t, database.Create(&Channel{Id: 60, Type: constant.ChannelTypeV6K}).Error)
	require.NoError(t, database.Create(&Ability{Model: "v6k-test-unpriced", Group: "default", ChannelId: 60, Enabled: true}).Error)
	updatePricing()
	require.Len(t, pricingMap, 3)
	byName := make(map[string]Pricing, len(pricingMap))
	for _, pricing := range pricingMap {
		byName[pricing.ModelName] = pricing
	}
	public := byName["Grok"]
	original := byName["TTP-grok"]
	require.Nil(t, public.PriceConfigured) // Preserve the existing catalog contract.
	v6k := byName["v6k-test-unpriced"]
	require.NotNil(t, v6k.PriceConfigured)
	require.False(t, *v6k.PriceConfigured)
	require.Equal(t, "dynamic", v6k.VideoBillingMode)
	require.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, public.SupportedEndpointTypes)
	require.Equal(t, original.SupportedEndpointTypes, public.SupportedEndpointTypes)
	require.Len(t, public.MappedEndpoints, 1)
	require.Equal(t, "/v1/videos", public.MappedEndpoints[0].Path)
	require.Equal(t, 6, public.MappedEndpoints[0].ExampleDuration)
	require.Equal(t, "720p", public.MappedEndpoints[0].ExampleResolution)
	encoded, err := common.Marshal(public)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"model_name":"Grok"`)
	require.Contains(t, string(encoded), `"mapped_endpoints"`)
	require.False(t, strings.Contains(string(encoded), "TTP-grok"))
}

func TestZonghengPricingOverridesModelNameHeuristics(t *testing.T) {
	for _, name := range []string{"XXseadanco2.0M", "kling-video-v3", "TTP-grok"} {
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeZongheng, name, "alias", nil)
		require.Len(t, endpoints, 1)
		require.Equal(t, "/v1/videos", endpoints[0].Path)
		require.Equal(t, "/v1/videos/{task_id}", endpoints[0].QueryPath)
		require.NotNil(t, endpoints[0].Parameters)
		require.False(t, endpoints[0].Parameters.OmitRatio)
		require.Equal(t, common.DefaultVideoAspectRatios(), endpoints[0].Parameters.Ratios)
		require.Equal(t, "reference_videos", endpoints[0].Parameters.VideoField)
	}
	require.Nil(t, common.GetVideoIntegrationParameters("XXseadanco2.0M"))
	other := getPricingIntegrationEndpoints(constant.ChannelTypeOpenAI, "kling-video-v3", nil)
	require.Equal(t, "/v1/video/async-generations", other[0].Path)
}

func TestV6KPricingUsesUnifiedVideoContractAndExactModelHints(t *testing.T) {
	for _, name := range []string{"kling-v3", "grok--1.5", "TTseadanco 2.0 Fast/480", "dola2.0"} {
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeV6K, name, "public-alias", nil)
		require.Len(t, endpoints, 1)
		require.Equal(t, "/v1/videos", endpoints[0].Path)
		require.Equal(t, "/v1/videos/{task_id}", endpoints[0].QueryPath)
		require.Equal(t, "seconds", endpoints[0].Parameters.DurationField)
		require.Equal(t, "videos", endpoints[0].Parameters.VideoField)
		require.Equal(t, "audios", endpoints[0].Parameters.AudioField)
	}
}

func TestSiteModelNotesOverrideOpaqueMappingsAndGenericChatMetadata(t *testing.T) {
	for _, name := range []string{"无限制-Video", "Seedance2.0-c", "Seedance2.0-fast-c", "Seedance2.0-fast2-c", "Seedance2.0:720（秒）", "Seedance2.5:480（秒）", "Seedance2.5:720（秒）", "Seedance2.5:720（条）"} {
		metadata := map[string]*Model{"opaque": {Endpoints: `{"openai":{"path":"/v1/chat/completions"}}`}}
		endpoints := getPricingIntegrationEndpointsForModels(constant.ChannelTypeOpenAI, "opaque", name, metadata)
		require.Len(t, endpoints, 1, name)
		require.Equal(t, "video", endpoints[0].Kind, name)
		require.Equal(t, "/v1/videos", endpoints[0].Path, name)
		require.Equal(t, common.GetSiteVideoIntegrationParameters(name), endpoints[0].Parameters, name)
	}
}
