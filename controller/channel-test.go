package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
)

type testResult struct {
	context     *gin.Context
	localErr    error
	newAPIError *types.NewAPIError
}

var unsupportedChannelConnectionTestTypes = []int{
	constant.ChannelTypeMidjourney,
	constant.ChannelTypeMidjourneyPlus,
	constant.ChannelTypeSunoAPI,
	constant.ChannelTypeKling,
	constant.ChannelTypeJimeng,
	constant.ChannelTypeVidu,
}

func supportsChannelConnectionTest(channelType int) bool {
	for _, unsupported := range unsupportedChannelConnectionTestTypes {
		if channelType == unsupported {
			return false
		}
	}
	return true
}

type channelTestUserInfo struct {
	userID     int
	group      string
	usingGroup string
}

const (
	channelTestEndpointOpenAIVideoAsync = "openai-video-async"
	channelTestEndpointSanbaoImage      = "sanbao-image"
	channelTestEndpointSanbaoVideo      = "sanbao-video"
	channelTestEndpointSanbaoUpload     = "sanbao-upload"
	channelTestEndpointSanbaoImagePoll  = "sanbao-image-poll"
	channelTestEndpointSanbaoVideoPoll  = "sanbao-video-poll"
)

type channelTestLabRequest struct {
	Model        string          `json:"model"`
	EndpointType string          `json:"endpoint_type"`
	Stream       bool            `json:"stream"`
	Payload      json.RawMessage `json:"payload"`
}

func normalizeChannelTestEndpoint(channel *model.Channel, modelName, endpointType string) string {
	normalized := strings.TrimSpace(endpointType)
	if isLikelySanbaoChannel(channel) && !isChannelTestSanbaoEndpoint(normalized) {
		if normalized == "" ||
			normalized == string(constant.EndpointTypeImageGeneration) ||
			normalized == string(constant.EndpointTypeOpenAIVideo) ||
			normalized == channelTestEndpointOpenAIVideoAsync {
			if isLikelySanbaoImageModel(modelName) {
				return channelTestEndpointSanbaoImage
			}
			return channelTestEndpointSanbaoVideo
		}
	}
	if isChannelTestVideosApiModel(modelName) &&
		(normalized == "" ||
			normalized == string(constant.EndpointTypeOpenAIVideo) ||
			normalized == channelTestEndpointOpenAIVideoAsync) {
		return string(constant.EndpointTypeOpenAIVideo)
	}
	if normalized != "" {
		return normalized
	}
	if isChannelTestVideoModel(channel, modelName) {
		if isChannelTestAsyncVideoModel(modelName) {
			return channelTestEndpointOpenAIVideoAsync
		}
		return string(constant.EndpointTypeOpenAIVideo)
	}
	if strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix) {
		return string(constant.EndpointTypeOpenAIResponseCompact)
	}
	if channel != nil && channel.Type == constant.ChannelTypeCodex {
		return string(constant.EndpointTypeOpenAIResponse)
	}
	if isChannelTestEmbeddingModel(channel, modelName) {
		return string(constant.EndpointTypeEmbeddings)
	}
	if strings.Contains(strings.ToLower(modelName), "rerank") {
		return string(constant.EndpointTypeJinaRerank)
	}
	if common.IsImageGenerationModel(modelName) ||
		(channel != nil && channel.Type == constant.ChannelTypeVolcEngine && strings.Contains(strings.ToLower(modelName), "seedream")) {
		return string(constant.EndpointTypeImageGeneration)
	}
	if strings.Contains(strings.ToLower(modelName), "codex") {
		return string(constant.EndpointTypeOpenAIResponse)
	}
	return normalized
}

func isChannelTestSanbaoEndpoint(endpointType string) bool {
	switch strings.TrimSpace(endpointType) {
	case channelTestEndpointSanbaoImage,
		channelTestEndpointSanbaoVideo,
		channelTestEndpointSanbaoUpload,
		channelTestEndpointSanbaoImagePoll,
		channelTestEndpointSanbaoVideoPoll:
		return true
	default:
		return false
	}
}

func isLikelySanbaoImageModel(modelName string) bool {
	return strings.Contains(normalizeCreationModelMetadataKey(modelName), "gpt-image2")
}

func resolveChannelTestEndpoint(channel *model.Channel, modelName, endpointType string) (string, string, types.RelayFormat) {
	endpointType = normalizeChannelTestEndpoint(channel, modelName, endpointType)
	requestPath := "/v1/chat/completions"
	relayFormat := types.RelayFormatOpenAI

	switch constant.EndpointType(endpointType) {
	case constant.EndpointTypeOpenAI:
		requestPath = "/v1/chat/completions"
		relayFormat = types.RelayFormatOpenAI
	case constant.EndpointTypeOpenAIResponse:
		requestPath = "/v1/responses"
		relayFormat = types.RelayFormatOpenAIResponses
	case constant.EndpointTypeOpenAIResponseCompact:
		requestPath = "/v1/responses/compact"
		relayFormat = types.RelayFormatOpenAIResponsesCompaction
	case constant.EndpointTypeAnthropic:
		requestPath = "/v1/messages"
		relayFormat = types.RelayFormatClaude
	case constant.EndpointTypeGemini:
		requestPath = "/v1beta/models/{model}:generateContent"
		relayFormat = types.RelayFormatGemini
	case constant.EndpointTypeJinaRerank:
		requestPath = "/v1/rerank"
		relayFormat = types.RelayFormatRerank
	case constant.EndpointTypeImageGeneration:
		requestPath = "/v1/images/generations"
		relayFormat = types.RelayFormatOpenAIImage
	case constant.EndpointTypeEmbeddings:
		requestPath = "/v1/embeddings"
		relayFormat = types.RelayFormatEmbedding
	case constant.EndpointTypeOpenAIVideo:
		requestPath = "/v1/videos"
		relayFormat = types.RelayFormatTask
	}
	if endpointType == channelTestEndpointOpenAIVideoAsync {
		requestPath = "/v1/video/async-generations"
		relayFormat = types.RelayFormatTask
	}
	switch endpointType {
	case channelTestEndpointSanbaoImage:
		requestPath = "/pg/images/generations"
		relayFormat = types.RelayFormatTask
	case channelTestEndpointSanbaoVideo, channelTestEndpointSanbaoUpload:
		requestPath = "/pg/video/async-generations"
		relayFormat = types.RelayFormatTask
	case channelTestEndpointSanbaoImagePoll:
		requestPath = "/v1/images/generations/{task_id}"
		relayFormat = types.RelayFormatTask
	case channelTestEndpointSanbaoVideoPoll:
		requestPath = "/v1/video/async-generations/{task_id}"
		relayFormat = types.RelayFormatTask
	}

	return endpointType, requestPath, relayFormat
}

func isChannelTestEmbeddingModel(channel *model.Channel, modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return strings.Contains(modelName, "embedding") ||
		strings.HasPrefix(modelName, "m3e") ||
		strings.Contains(modelName, "bge-") ||
		strings.Contains(modelName, "embed") ||
		(channel != nil && channel.Type == constant.ChannelTypeMokaAI)
}

func isChannelTestVideoModel(channel *model.Channel, modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if channel != nil {
		switch channel.Type {
		case constant.ChannelTypeSora, constant.ChannelTypeDoubaoVideo:
			return true
		}
	}
	return strings.HasPrefix(modelName, "sora") ||
		strings.HasPrefix(modelName, "veo") ||
		strings.Contains(modelName, "kling") ||
		strings.Contains(modelName, "video-") ||
		strings.HasPrefix(modelName, "videos-") ||
		strings.HasPrefix(modelName, "sd2") ||
		strings.Contains(modelName, "grok-imagine-video")
}

func isChannelTestAsyncVideoModel(modelName string) bool {
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
	if isVideo2ModelName(normalizedModelName) {
		return true
	}
	switch normalizedModelName {
	case "sora2", "sora-2", "kling-v3", "ko3", "veo31", "veo31-fast", "veo31-ref", "grok-imagine-video":
		return true
	default:
		return false
	}
}

func isChannelTestVideosApiModel(modelName string) bool {
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(normalizedModelName, "videos-") ||
		strings.HasPrefix(normalizedModelName, "sd2")
}

func isVideo2ModelName(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "video-2.0", "video-2.0-fast", "video-2.0-mini",
		"video-2.0-480p", "video-2.0-fast-480p", "video-2.0-mini-480p":
		return true
	default:
		return false
	}
}

func testChannel(channel *model.Channel, requester channelTestUserInfo, testModel string, endpointType string, isStream bool) testResult {
	return testChannelWithPayload(channel, requester, testModel, endpointType, isStream, nil)
}

func isChannelTestTaskPollEndpoint(endpointType string) bool {
	switch strings.TrimSpace(endpointType) {
	case channelTestEndpointSanbaoImagePoll, channelTestEndpointSanbaoVideoPoll:
		return true
	default:
		return false
	}
}

func testChannelTaskPoll(channel *model.Channel, requester channelTestUserInfo, endpointType string, payload json.RawMessage) testResult {
	taskID := getChannelTestPayloadTaskID(payload)
	if taskID == "" {
		err := errors.New("task_id is required for polling template")
		return testResult{
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}

	requestPath := "/v1/video/async-generations/" + url.PathEscape(taskID)
	if endpointType == channelTestEndpointSanbaoImagePoll {
		requestPath = "/v1/images/generations/" + url.PathEscape(taskID)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, requestPath, nil)
	c.Params = gin.Params{{Key: "task_id", Value: taskID}}
	c.Set("task_id", taskID)
	c.Set("channel", channel.Type)
	c.Set("base_url", channel.GetBaseURL())
	prepareChannelTestUserContext(c, channel, requester)

	if taskErr := relay.RelayTaskFetch(c, relayconstant.RelayModeVideoFetchByID); taskErr != nil {
		err := taskErr.Error
		if err == nil {
			err = errors.New(taskErr.Message)
		}
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCode(taskErr.Code), taskErr.StatusCode),
		}
	}

	result := w.Result()
	respBody, err := readTestResponseBody(result.Body, false)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
		}
	}
	if result.StatusCode >= http.StatusBadRequest {
		err := fmt.Errorf("polling returned status %d: %s", result.StatusCode, strings.TrimSpace(string(respBody)))
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeBadResponse, result.StatusCode),
		}
	}
	if bodyErr := validateTestResponseBody(respBody, false); bodyErr != nil {
		return testResult{
			context:     c,
			localErr:    bodyErr,
			newAPIError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	return testResult{context: c}
}

func testChannelWithPayload(channel *model.Channel, requester channelTestUserInfo, testModel string, endpointType string, isStream bool, payload json.RawMessage) testResult {
	tik := time.Now()
	if !supportsChannelConnectionTest(channel.Type) {
		channelTypeName := constant.GetChannelTypeName(channel.Type)
		return testResult{
			localErr: fmt.Errorf("%s channel test is not supported", channelTypeName),
		}
	}
	if isChannelTestTaskPollEndpoint(endpointType) {
		return testChannelTaskPoll(channel, requester, endpointType, payload)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testModel = strings.TrimSpace(testModel)
	if testModel == "" {
		if channel.TestModel != nil && *channel.TestModel != "" {
			testModel = strings.TrimSpace(*channel.TestModel)
		} else {
			models := channel.GetModels()
			if len(models) > 0 {
				testModel = strings.TrimSpace(models[0])
			}
			if testModel == "" {
				testModel = "gpt-4o-mini"
			}
		}
	}
	if payloadModel := getChannelTestPayloadModel(payload); payloadModel != "" {
		testModel = payloadModel
	}

	endpointType, requestPath, relayFormat := resolveChannelTestEndpoint(channel, testModel, endpointType)
	normalizedPayload, normalizeErr := normalizeChannelTestPayload(testModel, endpointType, payload)
	if normalizeErr != nil {
		return testResult{
			context:     c,
			localErr:    normalizeErr,
			newAPIError: types.NewError(normalizeErr, types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}
	payload = normalizedPayload
	if strings.HasPrefix(requestPath, "/v1/responses/compact") {
		testModel = ratio_setting.WithCompactModelSuffix(testModel)
	}

	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: requestPath}, // 使用动态路径
		Body:   nil,
		Header: make(http.Header),
	}

	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("channel", channel.Type)
	c.Set("base_url", channel.GetBaseURL())
	prepareChannelTestUserContext(c, channel, requester)

	newAPIError := middleware.SetupContextForSelectedChannel(c, channel, testModel)
	if newAPIError != nil {
		return testResult{
			context:     c,
			localErr:    newAPIError,
			newAPIError: newAPIError,
		}
	}
	if relayFormat == types.RelayFormatTask {
		applyChannelTestTaskChannelType(c, nil, resolveChannelTestTaskChannelType(channel, endpointType))
	}

	request, buildErr := buildChannelTestRequest(testModel, endpointType, channel, isStream, relayFormat, payload)
	if buildErr != nil {
		return testResult{
			context:     c,
			localErr:    buildErr,
			newAPIError: types.NewError(buildErr, types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}

	info, err := genChannelTestRelayInfo(c, relayFormat, request, payload)

	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeGenRelayInfoFailed),
		}
	}

	info.IsChannelTest = true
	info.InitChannelMeta(c)

	err = attachTestBillingRequestInput(info, request)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	relayRequest, _ := request.(dto.Request)
	err = helper.ModelMappedHelper(c, info, relayRequest)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeChannelModelMappedError),
		}
	}

	testModel = info.UpstreamModelName
	// 更新请求中的模型名称
	if relayRequest != nil {
		relayRequest.SetModelName(testModel)
		if imageRequest, ok := relayRequest.(*dto.ImageRequest); ok {
			applyImageGenerationTestDefaults(imageRequest)
		}
	} else if taskReq, ok := request.(relaycommon.TaskSubmitReq); ok {
		taskReq.Model = testModel
		c.Set("task_request", taskReq)
	}

	apiType, _ := common.ChannelType2APIType(channel.Type)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact &&
		apiType != constant.APITypeOpenAI &&
		apiType != constant.APITypeCodex {
		return testResult{
			context:     c,
			localErr:    fmt.Errorf("responses compaction test only supports openai/codex channels, got api type %d", apiType),
			newAPIError: types.NewError(fmt.Errorf("unsupported api type: %d", apiType), types.ErrorCodeInvalidApiType),
		}
	}
	if info.RelayMode == relayconstant.RelayModeVideoSubmit {
		return runTaskChannelTest(c, channel, endpointType, info, request, tik)
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return testResult{
			context:     c,
			localErr:    fmt.Errorf("invalid api type: %d, adaptor is nil", apiType),
			newAPIError: types.NewError(fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), types.ErrorCodeInvalidApiType),
		}
	}

	//// 创建一个用于日志的 info 副本，移除 ApiKey
	//logInfo := info
	//logInfo.ApiKey = ""
	common.SysLog(fmt.Sprintf("testing channel %d with model %s , info %+v ", channel.Id, testModel, info.ToString()))

	var priceData types.PriceData
	if relayFormat == types.RelayFormatTask {
		priceData, err = helper.ModelPriceHelperPerCall(c, info)
	} else if relayRequest != nil {
		priceData, err = helper.ModelPriceHelper(c, info, 0, relayRequest.GetTokenCountMeta())
	} else {
		err = errors.New("invalid channel test request type")
	}
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}

	adaptor.Init(info)

	var convertedRequest any
	// 根据 RelayMode 选择正确的转换函数
	switch info.RelayMode {
	case relayconstant.RelayModeEmbeddings:
		// Embedding 请求 - request 已经是正确的类型
		if embeddingReq, ok := relayRequest.(*dto.EmbeddingRequest); ok {
			convertedRequest, err = adaptor.ConvertEmbeddingRequest(c, info, *embeddingReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid embedding request type"),
				newAPIError: types.NewError(errors.New("invalid embedding request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeImagesGenerations:
		// 图像生成请求 - request 已经是正确的类型
		if imageReq, ok := relayRequest.(*dto.ImageRequest); ok {
			convertedRequest, err = adaptor.ConvertImageRequest(c, info, *imageReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid image request type"),
				newAPIError: types.NewError(errors.New("invalid image request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeRerank:
		// Rerank 请求 - request 已经是正确的类型
		if rerankReq, ok := relayRequest.(*dto.RerankRequest); ok {
			convertedRequest, err = adaptor.ConvertRerankRequest(c, info.RelayMode, *rerankReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid rerank request type"),
				newAPIError: types.NewError(errors.New("invalid rerank request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponses:
		// Response 请求 - request 已经是正确的类型
		if responseReq, ok := relayRequest.(*dto.OpenAIResponsesRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *responseReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid response request type"),
				newAPIError: types.NewError(errors.New("invalid response request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponsesCompact:
		// Response compaction request - convert to OpenAIResponsesRequest before adapting
		switch req := relayRequest.(type) {
		case *dto.OpenAIResponsesCompactionRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, dto.OpenAIResponsesRequest{
				Model:              req.Model,
				Input:              req.Input,
				Instructions:       req.Instructions,
				PreviousResponseID: req.PreviousResponseID,
			})
		case *dto.OpenAIResponsesRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *req)
		default:
			return testResult{
				context:     c,
				localErr:    errors.New("invalid response compaction request type"),
				newAPIError: types.NewError(errors.New("invalid response compaction request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	default:
		// Chat/Completion 等其他请求类型
		if generalReq, ok := relayRequest.(*dto.GeneralOpenAIRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIRequest(c, info, generalReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid general request type"),
				newAPIError: types.NewError(errors.New("invalid general request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	}

	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
		}
	}
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	//jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings)
	//if err != nil {
	//	return testResult{
	//		context:     c,
	//		localErr:    err,
	//		newAPIError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
	//	}
	//}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
				return testResult{
					context:     c,
					localErr:    fixedErr,
					newAPIError: relaycommon.NewAPIErrorFromParamOverride(fixedErr),
				}
			}
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid),
			}
		}
	}

	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if !isChannelTestSuccessStatus(httpResp.StatusCode) {
			err := service.RelayErrorHandler(c.Request.Context(), httpResp, true)
			common.SysError(fmt.Sprintf(
				"channel test bad response: channel_id=%d name=%s type=%d model=%s endpoint_type=%s status=%d err=%v",
				channel.Id,
				channel.Name,
				channel.Type,
				testModel,
				endpointType,
				httpResp.StatusCode,
				err,
			))
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError),
			}
		}
	}
	usageA, respErr := adaptor.DoResponse(c, httpResp, info)
	if respErr != nil {
		return testResult{
			context:     c,
			localErr:    respErr,
			newAPIError: respErr,
		}
	}
	usage, usageErr := coerceTestUsage(usageA, isStream, info.GetEstimatePromptTokens())
	if usageErr != nil {
		return testResult{
			context:     c,
			localErr:    usageErr,
			newAPIError: types.NewOpenAIError(usageErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	result := w.Result()
	respBody, err := readTestResponseBody(result.Body, isStream)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
		}
	}
	if bodyErr := validateTestResponseBody(respBody, isStream); bodyErr != nil {
		return testResult{
			context:     c,
			localErr:    bodyErr,
			newAPIError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	info.SetEstimatePromptTokens(usage.PromptTokens)

	quota, tieredResult := settleTestQuota(info, priceData, usage)
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	other := buildTestLogOther(c, info, priceData, usage, tieredResult)
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId:        channel.Id,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		ModelName:        info.OriginModelName,
		TokenName:        "模型测试",
		Quota:            quota,
		Content:          "模型测试",
		UseTimeSeconds:   int(consumedTime),
		IsStream:         info.IsStream,
		Group:            info.UsingGroup,
		Other:            other,
	})
	common.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
	return testResult{
		context:     c,
		localErr:    nil,
		newAPIError: nil,
	}
}

func attachTestBillingRequestInput(info *relaycommon.RelayInfo, request any) error {
	if info == nil {
		return nil
	}

	if relayRequest, ok := request.(dto.Request); ok {
		input, err := helper.BuildBillingExprRequestInputFromRequest(relayRequest, info.RequestHeaders)
		if err != nil {
			return err
		}
		info.BillingRequestInput = &input
		return nil
	}

	body, err := common.Marshal(request)
	if err != nil {
		return err
	}
	input := billingexpr.RequestInput{
		Body:    body,
		Headers: info.RequestHeaders,
	}
	info.BillingRequestInput = &input
	return nil
}

func settleTestQuota(info *relaycommon.RelayInfo, priceData types.PriceData, usage *dto.Usage) (int, *billingexpr.TieredResult) {
	if usage != nil && info != nil && info.TieredBillingSnapshot != nil {
		isClaudeUsageSemantic := usage.UsageSemantic == "anthropic" || info.GetFinalRequestRelayFormat() == types.RelayFormatClaude
		usedVars := billingexpr.UsedVars(info.TieredBillingSnapshot.ExprString)
		if ok, quota, result := service.TryTieredSettle(info, service.BuildTieredTokenParams(usage, isClaudeUsageSemantic, usedVars)); ok {
			return quota, result
		}
	}

	quota := 0
	if !priceData.UsePrice {
		quota = usage.PromptTokens + int(math.Round(float64(usage.CompletionTokens)*priceData.CompletionRatio))
		quota = int(math.Round(float64(quota) * priceData.ModelRatio))
		if priceData.ModelRatio != 0 && quota <= 0 {
			quota = 1
		}
		return quota, nil
	}

	return int(priceData.ModelPrice * common.QuotaPerUnit), nil
}

func prepareChannelTestUserContext(c *gin.Context, channel *model.Channel, requester channelTestUserInfo) channelTestUserInfo {
	userID := requester.userID
	if userID == 0 {
		userID = c.GetInt("id")
	}
	group := strings.TrimSpace(requester.group)
	if group == "" {
		group = strings.TrimSpace(c.GetString("user_group"))
	}
	if group == "" {
		group = strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	}
	if group == "" {
		group = strings.TrimSpace(c.GetString("group"))
	}
	if group == "" {
		group = "default"
	}

	usingGroup := strings.TrimSpace(requester.usingGroup)
	if usingGroup == "" {
		usingGroup = pickChannelTestUsingGroup(channel, group)
	}

	c.Set("id", userID)
	common.SetContextKey(c, constant.ContextKeyUserId, userID)
	if c.GetString("username") == "" {
		c.Set("username", fmt.Sprintf("channel-test-%d", userID))
	}
	common.SetContextKey(c, constant.ContextKeyUserName, c.GetString("username"))
	c.Set("user_group", group)
	common.SetContextKey(c, constant.ContextKeyUserGroup, group)
	c.Set("group", usingGroup)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, usingGroup)
	if common.GetContextKeyInt(c, constant.ContextKeyUserStatus) == 0 {
		common.SetContextKey(c, constant.ContextKeyUserStatus, common.UserStatusEnabled)
	}
	common.SetContextKey(c, constant.ContextKeyUserQuota, math.MaxInt32)

	return channelTestUserInfo{
		userID:     userID,
		group:      group,
		usingGroup: usingGroup,
	}
}

func pickChannelTestUsingGroup(channel *model.Channel, fallback string) string {
	if channel != nil {
		for _, group := range channel.GetGroups() {
			if group != "" {
				return group
			}
		}
	}
	if strings.TrimSpace(fallback) == "" {
		return "default"
	}
	return fallback
}

func genChannelTestRelayInfo(c *gin.Context, relayFormat types.RelayFormat, request any, payload json.RawMessage) (*relaycommon.RelayInfo, error) {
	if relayFormat == types.RelayFormatTask {
		if err := seedTaskTestRequestBody(c, request, payload); err != nil {
			return nil, err
		}
		info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
		if err != nil {
			return nil, err
		}
		info.RelayMode = relayconstant.RelayModeVideoSubmit
		if taskReq, ok := request.(relaycommon.TaskSubmitReq); ok {
			c.Set("task_request", taskReq)
		}
		return info, nil
	}

	relayRequest, ok := request.(dto.Request)
	if !ok {
		return nil, errors.New("request is not a relay request")
	}
	return relaycommon.GenRelayInfo(c, relayFormat, relayRequest, nil)
}

func seedTaskTestRequestBody(c *gin.Context, request any, payload json.RawMessage) error {
	taskReq, ok := request.(relaycommon.TaskSubmitReq)
	if !ok {
		return errors.New("request is not a task request")
	}
	body, err := buildChannelTestTaskPayload(taskReq.Model, request, payload)
	if err != nil {
		return err
	}
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		return err
	}
	c.Set(common.KeyBodyStorage, storage)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", taskReq)
	return nil
}

func buildChannelTestTaskPayload(modelName string, request any, payload json.RawMessage) ([]byte, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || string(payload) == "null" {
		return common.Marshal(request)
	}
	if payload[0] != '{' {
		return nil, errors.New("request payload must be a JSON object")
	}

	var raw map[string]any
	if err := common.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	if strings.TrimSpace(gjson.GetBytes(payload, "model").String()) == "" && strings.TrimSpace(modelName) != "" {
		raw["model"] = strings.TrimSpace(modelName)
	}
	return common.Marshal(raw)
}

func runTaskChannelTest(c *gin.Context, channel *model.Channel, endpointType string, info *relaycommon.RelayInfo, request any, tik time.Time) testResult {
	taskReq, ok := request.(relaycommon.TaskSubmitReq)
	if !ok {
		return testResult{
			context:     c,
			localErr:    errors.New("invalid task test request type"),
			newAPIError: types.NewError(errors.New("invalid task test request type"), types.ErrorCodeConvertRequestFailed),
		}
	}
	c.Set("task_request", taskReq)
	taskChannelType := resolveChannelTestTaskChannelType(channel, endpointType)
	applyChannelTestTaskChannelType(c, info, taskChannelType)

	adaptor := relay.GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(taskChannelType)))
	if adaptor == nil {
		err := fmt.Errorf("invalid task api platform: %d", taskChannelType)
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeInvalidApiType),
		}
	}
	adaptor.Init(info)
	if taskErr := adaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		newAPIError := types.NewOpenAIError(taskErr.Error, types.ErrorCode(taskErr.Code), taskErr.StatusCode)
		return testResult{
			context:     c,
			localErr:    taskErr.Error,
			newAPIError: newAPIError,
		}
	}

	modelName := info.OriginModelName
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName
	if err := helper.ModelMappedHelper(c, info, nil); err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeChannelModelMappedError),
		}
	}

	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}
	info.PriceData = priceData
	if estimatedRatios := adaptor.EstimateBilling(c, info); len(estimatedRatios) > 0 {
		for key, ratio := range estimatedRatios {
			info.PriceData.AddOtherRatio(key, ratio)
		}
	}

	requestBody, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
		}
	}
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	if resp != nil && !isChannelTestSuccessStatus(resp.StatusCode) {
		err := service.RelayErrorHandler(c.Request.Context(), resp, true)
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError),
		}
	}
	taskID, _, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		newAPIError := types.NewOpenAIError(taskErr.Error, types.ErrorCode(taskErr.Code), taskErr.StatusCode)
		return testResult{
			context:     c,
			localErr:    taskErr.Error,
			newAPIError: newAPIError,
		}
	}
	if strings.TrimSpace(taskID) == "" {
		err := errors.New("upstream task id is empty")
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}

	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	common.SysLog(fmt.Sprintf("testing task channel #%d, model %s, task_id %s", channel.Id, info.OriginModelName, taskID))
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId:      channel.Id,
		ModelName:      info.OriginModelName,
		TokenName:      "模型测试",
		Quota:          int(priceData.ModelPrice * common.QuotaPerUnit),
		Content:        "模型测试",
		UseTimeSeconds: int(float64(milliseconds) / 1000.0),
		Group:          info.UsingGroup,
		Other:          service.GenerateTextOtherInfo(c, info, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio, priceData.CompletionRatio, 0, priceData.CacheRatio, priceData.ModelPrice, priceData.GroupRatioInfo.GroupSpecialRatio),
	})
	return testResult{context: c}
}

func isChannelTestSuccessStatus(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func resolveChannelTestTaskChannelType(channel *model.Channel, endpointType string) int {
	if isChannelTestSanbaoEndpoint(endpointType) || isLikelySanbaoChannel(channel) {
		return constant.ChannelTypeSanbao
	}
	if channel == nil {
		return constant.ChannelTypeUnknown
	}
	return channel.Type
}

func applyChannelTestTaskChannelType(c *gin.Context, info *relaycommon.RelayInfo, channelType int) {
	c.Set("channel_type", channelType)
	c.Set("platform", strconv.Itoa(channelType))
	common.SetContextKey(c, constant.ContextKeyChannelType, channelType)
	if info == nil || info.ChannelMeta == nil {
		return
	}
	info.ChannelMeta.ChannelType = channelType
	if apiType, ok := common.ChannelType2APIType(channelType); ok {
		info.ChannelMeta.ApiType = apiType
	}
}

func buildTestLogOther(c *gin.Context, info *relaycommon.RelayInfo, priceData types.PriceData, usage *dto.Usage, tieredResult *billingexpr.TieredResult) map[string]interface{} {
	other := service.GenerateTextOtherInfo(c, info, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio, priceData.CompletionRatio,
		usage.PromptTokensDetails.CachedTokens, priceData.CacheRatio, priceData.ModelPrice, priceData.GroupRatioInfo.GroupSpecialRatio)
	if tieredResult != nil {
		service.InjectTieredBillingInfo(other, info, tieredResult)
	}
	return other
}

func coerceTestUsage(usageAny any, isStream bool, estimatePromptTokens int) (*dto.Usage, error) {
	switch u := usageAny.(type) {
	case *dto.Usage:
		return u, nil
	case dto.Usage:
		return &u, nil
	case nil:
		if !isStream {
			return nil, errors.New("usage is nil")
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	default:
		if !isStream {
			return nil, fmt.Errorf("invalid usage type: %T", usageAny)
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	}
}

func readTestResponseBody(body io.ReadCloser, isStream bool) ([]byte, error) {
	defer func() { _ = body.Close() }()
	const maxStreamLogBytes = 8 << 10
	if isStream {
		return io.ReadAll(io.LimitReader(body, maxStreamLogBytes))
	}
	return io.ReadAll(body)
}

func detectErrorFromTestResponseBody(respBody []byte) error {
	b := bytes.TrimSpace(respBody)
	if len(b) == 0 {
		return nil
	}
	if message := detectErrorMessageFromJSONBytes(b); message != "" {
		return fmt.Errorf("upstream error: %s", message)
	}

	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}
		if message := detectErrorMessageFromJSONBytes(payload); message != "" {
			return fmt.Errorf("upstream error: %s", message)
		}
	}

	return nil
}

func validateStreamTestResponseBody(respBody []byte) error {
	b := bytes.TrimSpace(respBody)
	if len(b) == 0 {
		return errors.New("stream response body is empty")
	}

	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}

		return nil
	}

	return errors.New("stream response body does not contain a valid stream event")
}

func validateTestResponseBody(respBody []byte, isStream bool) error {
	if bodyErr := detectErrorFromTestResponseBody(respBody); bodyErr != nil {
		return bodyErr
	}
	if isStream {
		return validateStreamTestResponseBody(respBody)
	}
	return nil
}

func shouldUseStreamForAutomaticChannelTest(channel *model.Channel) bool {
	return channel != nil && channel.Type == constant.ChannelTypeCodex
}

func detectErrorMessageFromJSONBytes(jsonBytes []byte) string {
	if len(jsonBytes) == 0 {
		return ""
	}
	if jsonBytes[0] != '{' && jsonBytes[0] != '[' {
		return ""
	}
	errVal := gjson.GetBytes(jsonBytes, "error")
	if !errVal.Exists() || errVal.Type == gjson.Null {
		return ""
	}

	message := gjson.GetBytes(jsonBytes, "error.message").String()
	if message == "" {
		message = gjson.GetBytes(jsonBytes, "error.error.message").String()
	}
	if message == "" && errVal.Type == gjson.String {
		message = errVal.String()
	}
	if message == "" {
		message = errVal.Raw
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return "upstream returned error payload"
	}
	return message
}

func getChannelTestPayloadModel(payload json.RawMessage) string {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || payload[0] != '{' {
		return ""
	}
	return strings.TrimSpace(gjson.GetBytes(payload, "model").String())
}

func getChannelTestPayloadTaskID(payload json.RawMessage) string {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || payload[0] != '{' {
		return ""
	}
	for _, path := range []string{"task_id", "taskId", "id"} {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func normalizeChannelTestPayload(modelName, endpointType string, payload json.RawMessage) (json.RawMessage, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || string(payload) == "null" ||
		endpointType != string(constant.EndpointTypeOpenAIVideo) ||
		!isChannelTestVideosApiModel(modelName) {
		return payload, nil
	}
	if payload[0] != '{' {
		return nil, errors.New("request payload must be a JSON object")
	}

	var raw map[string]any
	if err := common.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	size := channelTestStringField(raw, "size")
	if channelTestStringField(raw, "ratio") == "" {
		raw["ratio"] = channelTestVideosRatio(size)
	}
	if channelTestStringField(raw, "resolution") == "" {
		raw["resolution"] = channelTestVideosResolution(size)
	}
	if _, exists := raw["duration"]; !exists {
		raw["duration"] = channelTestVideosDuration(raw["seconds"])
	}
	delete(raw, "size")
	delete(raw, "seconds")
	return common.Marshal(raw)
}

func channelTestStringField(values map[string]any, key string) string {
	value, exists := values[key]
	if !exists || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func channelTestVideosRatio(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "720x1280", "480x864", "496x864":
		return "9:16"
	case "1024x1024":
		return "1:1"
	default:
		return "16:9"
	}
}

func channelTestVideosResolution(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	if strings.Contains(size, "480") || strings.Contains(size, "496") || strings.Contains(size, "864") {
		return "480p"
	}
	return "720p"
}

func channelTestVideosDuration(value any) int {
	switch duration := value.(type) {
	case json.Number:
		if parsed, err := strconv.Atoi(duration.String()); err == nil && parsed > 0 {
			return parsed
		}
	case float64:
		if duration > 0 {
			return int(duration)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(duration)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 5
}

func buildChannelTestRequest(model string, endpointType string, channel *model.Channel, isStream bool, relayFormat types.RelayFormat, payload json.RawMessage) (any, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || string(payload) == "null" {
		return buildTestRequest(model, endpointType, channel, isStream), nil
	}
	if payload[0] != '{' {
		return nil, errors.New("request payload must be a JSON object")
	}

	switch relayFormat {
	case types.RelayFormatTask:
		var request relaycommon.TaskSubmitReq
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		return request, nil
	case types.RelayFormatEmbedding:
		var request dto.EmbeddingRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		return &request, nil
	case types.RelayFormatOpenAIImage:
		var request dto.ImageRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		return &request, nil
	case types.RelayFormatRerank:
		var request dto.RerankRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		return &request, nil
	case types.RelayFormatOpenAIResponses:
		var request dto.OpenAIResponsesRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		if isStream {
			request.Stream = lo.ToPtr(true)
		}
		return &request, nil
	case types.RelayFormatOpenAIResponsesCompaction:
		var request dto.OpenAIResponsesCompactionRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		return &request, nil
	case types.RelayFormatOpenAI, types.RelayFormatClaude, types.RelayFormatGemini:
		var request dto.GeneralOpenAIRequest
		if err := common.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		if strings.TrimSpace(request.Model) == "" {
			request.Model = model
		}
		if isStream {
			request.Stream = lo.ToPtr(true)
			request.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
		}
		return &request, nil
	default:
		return nil, fmt.Errorf("unsupported test endpoint relay format: %v", relayFormat)
	}
}

func buildTestRequest(model string, endpointType string, channel *model.Channel, isStream bool) any {
	testResponsesInput := json.RawMessage(`[{"role":"user","content":"hi"}]`)

	// 根据端点类型构建不同的测试请求
	if endpointType != "" {
		if endpointType == channelTestEndpointOpenAIVideoAsync {
			return relaycommon.TaskSubmitReq{
				Model:    model,
				Prompt:   "a short product video",
				Size:     channelTestVideoSize(model),
				Duration: 4,
			}
		}
		if endpointType == channelTestEndpointSanbaoImage {
			return relaycommon.TaskSubmitReq{
				Model:  model,
				Prompt: "a commercial poster with cinematic light",
			}
		}
		if endpointType == channelTestEndpointSanbaoVideo || endpointType == channelTestEndpointSanbaoUpload {
			return relaycommon.TaskSubmitReq{
				Model:      model,
				Prompt:     "a short product video",
				Resolution: "720p",
				Duration:   5,
			}
		}
		switch constant.EndpointType(endpointType) {
		case constant.EndpointTypeOpenAIVideo:
			if isChannelTestVideosApiModel(model) {
				return relaycommon.TaskSubmitReq{
					Model:      model,
					Prompt:     "a short product video",
					Ratio:      "16:9",
					Resolution: "720p",
					Duration:   5,
				}
			}
			return relaycommon.TaskSubmitReq{
				Model:    model,
				Prompt:   "a short product video",
				Size:     channelTestVideoSize(model),
				Duration: 4,
			}
		case constant.EndpointTypeEmbeddings:
			// 返回 EmbeddingRequest
			return &dto.EmbeddingRequest{
				Model: model,
				Input: []any{"hello world"},
			}
		case constant.EndpointTypeImageGeneration:
			// 返回 ImageRequest
			return buildImageGenerationTestRequest(model)
		case constant.EndpointTypeJinaRerank:
			// 返回 RerankRequest
			return &dto.RerankRequest{
				Model:     model,
				Query:     "What is Deep Learning?",
				Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
				TopN:      lo.ToPtr(2),
			}
		case constant.EndpointTypeOpenAIResponse:
			// 返回 OpenAIResponsesRequest
			return &dto.OpenAIResponsesRequest{
				Model:  model,
				Input:  json.RawMessage(`[{"role":"user","content":"hi"}]`),
				Stream: lo.ToPtr(isStream),
			}
		case constant.EndpointTypeOpenAIResponseCompact:
			// 返回 OpenAIResponsesCompactionRequest
			return &dto.OpenAIResponsesCompactionRequest{
				Model: model,
				Input: testResponsesInput,
			}
		case constant.EndpointTypeAnthropic, constant.EndpointTypeGemini, constant.EndpointTypeOpenAI:
			// 返回 GeneralOpenAIRequest
			maxTokens := uint(16)
			if constant.EndpointType(endpointType) == constant.EndpointTypeGemini {
				maxTokens = 3000
			}
			req := &dto.GeneralOpenAIRequest{
				Model:  model,
				Stream: lo.ToPtr(isStream),
				Messages: []dto.Message{
					{
						Role:    "user",
						Content: "hi",
					},
				},
				MaxTokens: lo.ToPtr(maxTokens),
			}
			if isStream {
				req.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
			}
			return req
		}
	}

	if isChannelTestVideoModel(channel, model) {
		if isChannelTestVideosApiModel(model) {
			return relaycommon.TaskSubmitReq{
				Model:      model,
				Prompt:     "a short product video",
				Ratio:      "16:9",
				Resolution: "720p",
				Duration:   5,
			}
		}
		return relaycommon.TaskSubmitReq{
			Model:    model,
			Prompt:   "a short product video",
			Size:     channelTestVideoSize(model),
			Duration: 4,
		}
	}

	// 自动检测逻辑（保持原有行为）
	if strings.Contains(strings.ToLower(model), "rerank") {
		return &dto.RerankRequest{
			Model:     model,
			Query:     "What is Deep Learning?",
			Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
			TopN:      lo.ToPtr(2),
		}
	}

	// 先判断是否为 Embedding 模型
	if strings.Contains(strings.ToLower(model), "embedding") ||
		strings.HasPrefix(model, "m3e") ||
		strings.Contains(model, "bge-") {
		// 返回 EmbeddingRequest
		return &dto.EmbeddingRequest{
			Model: model,
			Input: []any{"hello world"},
		}
	}

	// Responses compaction models (must use /v1/responses/compact)
	if strings.HasSuffix(model, ratio_setting.CompactModelSuffix) {
		return &dto.OpenAIResponsesCompactionRequest{
			Model: model,
			Input: testResponsesInput,
		}
	}

	// Responses-only models (e.g. codex series)
	if strings.Contains(strings.ToLower(model), "codex") {
		return &dto.OpenAIResponsesRequest{
			Model:  model,
			Input:  json.RawMessage(`[{"role":"user","content":"hi"}]`),
			Stream: lo.ToPtr(isStream),
		}
	}

	// Chat/Completion 请求 - 返回 GeneralOpenAIRequest
	testRequest := &dto.GeneralOpenAIRequest{
		Model:  model,
		Stream: lo.ToPtr(isStream),
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hi",
			},
		},
	}
	if isStream {
		testRequest.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}

	if strings.HasPrefix(model, "o") {
		testRequest.MaxCompletionTokens = lo.ToPtr(uint(16))
	} else if strings.Contains(model, "thinking") {
		if !strings.Contains(model, "claude") {
			testRequest.MaxTokens = lo.ToPtr(uint(50))
		}
	} else if strings.Contains(model, "gemini") {
		testRequest.MaxTokens = lo.ToPtr(uint(3000))
	} else {
		testRequest.MaxTokens = lo.ToPtr(uint(16))
	}

	return testRequest
}

func channelTestVideoSize(modelName string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(modelName)), "-480p") {
		return "496x864"
	}
	return "720x1280"
}

func buildImageGenerationTestRequest(modelName string) *dto.ImageRequest {
	request := &dto.ImageRequest{
		Model:  modelName,
		Prompt: "a cute cat",
		N:      lo.ToPtr(uint(1)),
		Size:   "1024x1024",
	}
	applyImageGenerationTestDefaults(request)
	return request
}

func applyImageGenerationTestDefaults(request *dto.ImageRequest) {
	if request == nil || !strings.EqualFold(strings.TrimSpace(request.Model), "gpt-image2") {
		return
	}
	request.N = nil
	request.Size = ""
	request.OutputResolution = json.RawMessage(`"1K"`)
	request.AspectRatio = json.RawMessage(`"1:1"`)
}

func TestChannel(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		channel, err = model.GetChannelById(channelId, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	//defer func() {
	//	if channel.ChannelInfo.IsMultiKey {
	//		go func() { _ = channel.SaveChannelInfo() }()
	//	}
	//}()
	testModel := c.Query("model")
	endpointType := c.Query("endpoint_type")
	isStream, _ := strconv.ParseBool(c.Query("stream"))
	tik := time.Now()
	requester := channelTestUserInfo{
		userID:     c.GetInt("id"),
		group:      c.GetString("user_group"),
		usingGroup: c.GetString("group"),
	}
	if requester.group == "" {
		requester.group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	}
	if requester.usingGroup == "" {
		requester.usingGroup = common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	}
	result := testChannel(channel, requester, testModel, endpointType, isStream)
	if result.localErr != nil {
		resp := gin.H{
			"success": false,
			"message": result.localErr.Error(),
			"time":    0.0,
		}
		if result.newAPIError != nil {
			resp["error_code"] = result.newAPIError.GetErrorCode()
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if result.newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    result.newAPIError.Error(),
			"time":       consumedTime,
			"error_code": result.newAPIError.GetErrorCode(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
	})
}

func TestChannelLab(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		channel, err = model.GetChannelById(channelId, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	var request channelTestLabRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}

	tik := time.Now()
	requester := channelTestUserInfo{
		userID:     c.GetInt("id"),
		group:      c.GetString("user_group"),
		usingGroup: c.GetString("group"),
	}
	if requester.group == "" {
		requester.group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	}
	if requester.usingGroup == "" {
		requester.usingGroup = common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	}

	result := testChannelWithPayload(channel, requester, request.Model, request.EndpointType, request.Stream, request.Payload)
	if result.localErr != nil {
		resp := gin.H{
			"success": false,
			"message": result.localErr.Error(),
			"time":    0.0,
		}
		if result.newAPIError != nil {
			resp["error_code"] = result.newAPIError.GetErrorCode()
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if result.newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    result.newAPIError.Error(),
			"time":       consumedTime,
			"error_code": result.newAPIError.GetErrorCode(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
	})
}

var testAllChannelsLock sync.Mutex
var testAllChannelsRunning bool = false

func testAllChannels(notify bool) error {

	testAllChannelsLock.Lock()
	if testAllChannelsRunning {
		testAllChannelsLock.Unlock()
		return errors.New("测试已在运行中")
	}
	testAllChannelsRunning = true
	testAllChannelsLock.Unlock()
	channels, getChannelErr := model.GetAllChannels(0, 0, true, false)
	if getChannelErr != nil {
		return getChannelErr
	}
	var disableThreshold = int64(common.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // a impossible value
	}
	gopool.Go(func() {
		// 使用 defer 确保无论如何都会重置运行状态，防止死锁
		defer func() {
			testAllChannelsLock.Lock()
			testAllChannelsRunning = false
			testAllChannelsLock.Unlock()
		}()

		for _, channel := range channels {
			if channel.Status == common.ChannelStatusManuallyDisabled {
				continue
			}
			isChannelEnabled := channel.Status == common.ChannelStatusEnabled
			tik := time.Now()
			result := testChannel(channel, channelTestUserInfo{
				userID:     0,
				group:      "default",
				usingGroup: pickChannelTestUsingGroup(channel, "default"),
			}, "", "", shouldUseStreamForAutomaticChannelTest(channel))
			tok := time.Now()
			milliseconds := tok.Sub(tik).Milliseconds()

			shouldBanChannel := false
			newAPIError := result.newAPIError
			// request error disables the channel
			if newAPIError != nil {
				shouldBanChannel = service.ShouldDisableChannel(result.newAPIError)
			}

			// 当错误检查通过，才检查响应时间
			if common.AutomaticDisableChannelEnabled && !shouldBanChannel {
				if milliseconds > disableThreshold {
					err := fmt.Errorf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0)
					newAPIError = types.NewOpenAIError(err, types.ErrorCodeChannelResponseTimeExceeded, http.StatusRequestTimeout)
					shouldBanChannel = true
				}
			}

			// disable channel
			if isChannelEnabled && shouldBanChannel && channel.GetAutoBan() {
				processChannelError(result.context, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(result.context, constant.ContextKeyChannelKey), channel.GetAutoBan()), newAPIError)
			}

			// enable channel
			if !isChannelEnabled && service.ShouldEnableChannel(newAPIError, channel.Status) {
				service.EnableChannel(channel.Id, common.GetContextKeyString(result.context, constant.ContextKeyChannelKey), channel.Name)
			}

			channel.UpdateResponseTime(milliseconds)
			time.Sleep(common.RequestInterval)
		}

		if notify {
			service.NotifyRootUser(dto.NotifyTypeChannelTest, "通道测试完成", "所有通道测试已完成")
		}
	})
	return nil
}

func TestAllChannels(c *gin.Context) {
	err := testAllChannels(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

var autoTestChannelsOnce sync.Once

func AutomaticallyTestChannels() {
	// 只在Master节点定时测试渠道
	if !common.IsMasterNode {
		return
	}
	autoTestChannelsOnce.Do(func() {
		for {
			if !operation_setting.GetMonitorSetting().AutoTestChannelEnabled {
				time.Sleep(1 * time.Minute)
				continue
			}
			for {
				frequency := operation_setting.GetMonitorSetting().AutoTestChannelMinutes
				time.Sleep(time.Duration(int(math.Round(frequency))) * time.Minute)
				common.SysLog(fmt.Sprintf("automatically test channels with interval %f minutes", frequency))
				common.SysLog("automatically testing all channels")
				_ = testAllChannels(false)
				common.SysLog("automatically channel test finished")
				if !operation_setting.GetMonitorSetting().AutoTestChannelEnabled {
					break
				}
			}
		}
	})
}
