package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupportsChannelConnectionTestRejectsAsyncVideoChannels(t *testing.T) {
	require.True(t, supportsChannelConnectionTest(constant.ChannelTypeSora))
	require.True(t, supportsChannelConnectionTest(constant.ChannelTypeOpenAI))
}

func TestResolveChannelTestEndpointUsesOpenAIVideoForVideoModels(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeSora,
		Models: "video-2.0,video-2.0-fast",
	}

	endpointType, requestPath, relayFormat := resolveChannelTestEndpoint(channel, "video-2.0-fast", "")

	require.Equal(t, string(constant.EndpointTypeOpenAIVideo), endpointType)
	require.Equal(t, "/v1/video/async-generations", requestPath)
	require.Equal(t, types.RelayFormat(types.RelayFormatTask), relayFormat)
}

func TestResolveChannelTestEndpointUsesImageGenerationForGptImage2(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeOpenAI,
		Models: "gpt-image2",
	}

	endpointType, requestPath, relayFormat := resolveChannelTestEndpoint(channel, "gpt-image2", "")

	require.Equal(t, string(constant.EndpointTypeImageGeneration), endpointType)
	require.Equal(t, "/v1/images/generations", requestPath)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIImage), relayFormat)
}

func TestBuildTestRequestUsesVideoPayloadForOpenAIVideoEndpoint(t *testing.T) {
	request := buildTestRequest("video-2.0-fast", string(constant.EndpointTypeOpenAIVideo), &model.Channel{}, false)

	require.IsType(t, relaycommon.TaskSubmitReq{}, request)
	videoRequest := request.(relaycommon.TaskSubmitReq)
	require.Equal(t, "video-2.0-fast", videoRequest.Model)
	require.NotEmpty(t, videoRequest.Prompt)
	require.Equal(t, "720x1280", videoRequest.Size)
	require.Equal(t, 4, videoRequest.Duration)
}

func TestBuildTestRequestUsesGptImage2PayloadForImageGenerationEndpoint(t *testing.T) {
	request := buildTestRequest("gpt-image2", string(constant.EndpointTypeImageGeneration), &model.Channel{}, false)

	require.IsType(t, &dto.ImageRequest{}, request)
	imageRequest := request.(*dto.ImageRequest)
	require.Equal(t, "gpt-image2", imageRequest.Model)
	require.Nil(t, imageRequest.N)
	require.Empty(t, imageRequest.Size)
	require.JSONEq(t, `"1K"`, string(imageRequest.OutputResolution))
	require.JSONEq(t, `"1:1"`, string(imageRequest.AspectRatio))
}

func TestPrepareChannelTestContextUsesRequesterUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	ctx.Set("id", 42)
	ctx.Set("username", "tester")
	ctx.Set("group", "vip")
	ctx.Set("user_group", "vip")

	info := prepareChannelTestUserContext(ctx, &model.Channel{Group: "default"}, channelTestUserInfo{
		userID: 42,
	})

	require.Equal(t, 42, ctx.GetInt("id"))
	require.Equal(t, "tester", ctx.GetString("username"))
	require.Equal(t, "vip", common.GetContextKeyString(ctx, constant.ContextKeyUserGroup))
	require.Equal(t, "default", ctx.GetString("group"))
	require.Equal(t, "default", info.usingGroup)
}

func TestRelayInfoForVideoTestUsesTaskRelayMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/async-generations", nil)
	ctx.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
	request := relaycommon.TaskSubmitReq{Model: "video-2.0-fast", Prompt: "hi"}

	info, err := genChannelTestRelayInfo(ctx, types.RelayFormatTask, request)

	require.NoError(t, err)
	require.Equal(t, relayconstant.RelayModeVideoSubmit, info.RelayMode)
	require.NotNil(t, info.TaskRelayInfo)

	var stored relaycommon.TaskSubmitReq
	require.NoError(t, common.UnmarshalBodyReusable(ctx, &stored))
	require.Equal(t, "video-2.0-fast", stored.Model)
	require.Equal(t, "hi", stored.Prompt)
}

func TestSettleTestQuotaUsesTieredBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    `param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`,
			ExprHash:      billingexpr.ExprHashString(`param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`),
			GroupRatio:    1,
			EstimatedTier: "stream",
			QuotaPerUnit:  common.QuotaPerUnit,
			ExprVersion:   1,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"stream":true}`),
		},
	}

	quota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
	}, &dto.Usage{
		PromptTokens: 1000,
	})

	require.Equal(t, 1500, quota)
	require.NotNil(t, result)
	require.Equal(t, "stream", result.MatchedTier)
}

func TestBuildTestLogOtherInjectsTieredInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr",
			ExprString:  `tier("base", p * 2)`,
		},
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 12,
		},
	}

	other := buildTestLogOther(ctx, info, priceData, usage, &billingexpr.TieredResult{
		MatchedTier: "base",
	})

	require.Equal(t, "tiered_expr", other["billing_mode"])
	require.Equal(t, "base", other["matched_tier"])
	require.NotEmpty(t, other["expr_b64"])
}
