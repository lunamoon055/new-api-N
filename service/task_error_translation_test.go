package service

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestTranslateVideoTaskErrorMappings(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		code       string
		statusCode int
		expected   string
	}{
		{
			name:     "image staging connection interrupted",
			raw:      `{"error":{"message":"invalid image_urls[0]: wait for init image failed: graphql request failed: Post \"https://example.ai/graphql\": EOF","type":"invalid_request_error"}}`,
			expected: "上游素材服务连接中断，图片暂时无法读取，请稍后重试或重新上传图片。",
		},
		{
			name:     "image url missing",
			raw:      `{"error":{"message":"invalid image_urls[0]: image url returned 404","type":"invalid_request_error"}}`,
			expected: "参考图片链接不存在或已失效，请重新上传图片后再试。",
		},
		{
			name:     "audio url missing",
			raw:      `{"error":{"message":"invalid audio_url: audio url returned 404","type":"invalid_request_error"}}`,
			expected: "参考音频链接不存在或已失效，请重新上传音频后再试。",
		},
		{
			name:     "invalid video height",
			raw:      `{"error":{"message":"invalid video_reference[0]: wait for staged video asset failed: uploaded media failed: INVALID_HEIGHT","type":"invalid_request_error"}}`,
			expected: "参考视频的高度或分辨率不符合上游要求，请调整视频尺寸后重试。",
		},
		{
			name:     "upstream request failed",
			raw:      `{"code":"do_request_failed","message":"do request failed: upstream error: do request failed","data":null}`,
			expected: "调用上游视频服务失败，请稍后重试；如持续失败，请联系管理员。",
		},
		{
			name:     "polling timeout",
			raw:      "Generation polling timed out",
			expected: "视频生成等待超时，请稍后查询任务状态或重新发起生成。",
		},
		{
			name:     "moderation",
			raw:      "PROVIDER_MODERATION_ERROR",
			expected: "内容审核未通过，请调整提示词或参考素材后重试。",
		},
		{
			name:     "nsfw moderation",
			raw:      "PROVIDER_MODERATION_ERROR: NSFW",
			expected: "内容审核未通过，可能包含不适宜内容，请修改提示词或参考素材。",
		},
		{
			name:     "provider internal error",
			raw:      "PROVIDER_INTERNAL_ERROR",
			expected: "上游视频服务发生内部错误，请稍后重试。",
		},
		{
			name:     "provider timeout",
			raw:      "PROVIDER_TIMEOUT",
			expected: "上游视频服务响应超时，请稍后重试。",
		},
		{
			name:     "provider invalid request",
			raw:      "PROVIDER_INVALID_REQUEST",
			expected: "请求参数无效，请检查提示词、参考素材和生成参数。",
		},
		{
			name:     "extreme violence",
			raw:      "PROVIDER_TIMEOUT: NSFW, EXTREME_VIOLENCE",
			expected: "请求处理超时，且内容可能包含不适宜或极端暴力元素，请修改提示词后重试。",
		},
		{
			name:     "inappropriate prompt",
			raw:      `{"error":{"message":"generation failed: generate error: Your prompt appears to contain inappropriate content, please be mindful of our Terms of Service. You may modify your prompt and try again.","type":"server_error"}}`,
			expected: "提示词可能包含不适宜内容，未通过内容审核，请修改提示词后重试。",
		},
		{
			name:     "trademark moderation",
			raw:      "PROVIDER_MODERATION_ERROR: TRADEMARK",
			expected: "内容审核未通过，可能涉及商标或品牌侵权，请修改提示词或参考素材。",
		},
		{
			name:     "child moderation",
			raw:      "PROVIDER_TIMEOUT: CHILD",
			expected: "请求处理超时，且内容可能涉及未成年人敏感内容，请修改提示词后重试。",
		},
		{
			name:     "audio too short",
			raw:      `{"error":{"message":"invalid audio_reference[0]: wait for staged audio asset failed: uploaded media failed: DURATION_TOO_SHORT","type":"invalid_request_error"}}`,
			expected: "参考音频时长过短，不符合模型要求，请上传符合时长要求的音频。",
		},
		{
			name:       "unknown generation error",
			raw:        `{"error":{"message":"generation failed: generate error: An error occurred.","type":"server_error"}}`,
			statusCode: http.StatusInternalServerError,
			expected:   "视频生成失败，上游服务返回未知错误，请稍后重试。",
		},
		{
			name:     "failed generation status",
			raw:      "generation status FAILED",
			expected: "视频生成失败，请检查提示词和参考素材后重试。",
		},
		{
			name:     "insufficient credits",
			raw:      "upstream error: insufficient credits",
			expected: "积分不足，请联系管理员",
		},
		{
			name:     "missing prompt nested detail",
			raw:      `{"detail":[{"type":"missing","loc":["body","prompt"],"msg":"Field required"}]}`,
			expected: "视频提示词不能为空，请填写视频描述后重试。",
		},
		{
			name:     "invalid route url",
			raw:      `{"error":{"message":"Invalid URL (POST /v1/video/async-generations)"}}`,
			expected: "当前视频渠道接口地址配置错误，请联系管理员检查渠道地址和模型映射。",
		},
		{
			name:       "authentication failure",
			raw:        "forbidden",
			statusCode: http.StatusForbidden,
			expected:   "视频渠道鉴权失败，请联系管理员检查 API 密钥。",
		},
		{
			name:       "rate limit",
			raw:        "too many requests",
			statusCode: http.StatusTooManyRequests,
			expected:   "当前视频服务繁忙，请稍后再试。",
		},
		{
			name:     "plain Chinese provider message passes through",
			raw:      "参考音频总时长超过 15 秒上限，请裁剪后重试",
			expected: "参考音频总时长超过 15 秒上限，请裁剪后重试",
		},
		{
			name:     "mixed field name and Chinese provider message passes through",
			raw:      "audio_urls 共 3 个参考音频，总时长 16.80 秒，超过总和 15 秒上限，请裁剪后重试",
			expected: "audio_urls 共 3 个参考音频，总时长 16.80 秒，超过总和 15 秒上限，请裁剪后重试",
		},
		{
			name:     "Chinese message is extracted from structured response",
			raw:      `{"error":{"message":"参考视频总时长超过 30 秒上限，请裁剪后重试","type":"invalid_request_error"}}`,
			expected: "参考视频总时长超过 30 秒上限，请裁剪后重试",
		},
		{
			name:     "Chinese validation detail is extracted from array",
			raw:      `{"detail":[{"type":"missing","loc":["prompt"],"msg":"缺少必填字段：prompt"}]}`,
			expected: "缺少必填字段：prompt",
		},
		{
			name:     "Chinese insufficient credits still uses unified message",
			raw:      "当前账户积分不足，请充值后重试",
			expected: "积分不足，请联系管理员",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			translated := TranslateVideoTaskError(test.raw, test.code, test.statusCode)
			require.Equal(t, test.expected, translated.Message)
			require.NotEmpty(t, translated.Category)
		})
	}
}

func TestLocalizeVideoTaskErrorDoesNotSerializeRawMessage(t *testing.T) {
	raw := `{"error":{"message":"invalid image_urls[0]: image url returned 404"}}`
	taskError := &dto.TaskError{
		Code:       "fail_to_fetch_task",
		Message:    raw,
		StatusCode: http.StatusBadRequest,
	}

	LocalizeVideoTaskError(taskError)

	require.Equal(t, "参考图片链接不存在或已失效，请重新上传图片后再试。", taskError.Message)
	require.Equal(t, raw, taskError.RawMessage)
	publicData, ok := taskError.Data.(dto.TaskErrorPublicData)
	require.True(t, ok)
	require.Equal(t, "reference_media", publicData.Category)
	require.False(t, publicData.Retryable)

	payload, err := common.Marshal(taskError)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "image url returned 404")
	require.Contains(t, string(payload), "参考图片链接不存在或已失效")
}

func TestLocalizeVideoTaskErrorPreservesReadableChineseProviderMessage(t *testing.T) {
	raw := "audio_urls 共 3 个参考音频，总时长 16.80 秒，超过总和 15 秒上限，请裁剪后重试"
	taskError := &dto.TaskError{
		Code:       "fail_to_fetch_task",
		Message:    raw,
		StatusCode: http.StatusBadRequest,
	}

	LocalizeVideoTaskError(taskError)

	require.Equal(t, raw, taskError.Message)
	require.Equal(t, raw, taskError.RawMessage)
	publicData, ok := taskError.Data.(dto.TaskErrorPublicData)
	require.True(t, ok)
	require.Equal(t, "upstream_message", publicData.Category)
	payload, err := common.Marshal(taskError)
	require.NoError(t, err)
	require.Contains(t, string(payload), raw)
}

func TestSetVideoTaskFailureKeepsMaskedRawErrorPrivate(t *testing.T) {
	raw := `download https://cdn.example.com/private/file.png?token=secret: image url returned 404`
	task := &model.Task{Status: model.TaskStatusFailure}

	SetVideoTaskFailure(task, raw, "fail_to_fetch_task", http.StatusBadRequest)

	require.Equal(t, "参考图片链接不存在或已失效，请重新上传图片后再试。", task.FailReason)
	require.NotEmpty(t, task.PrivateData.UpstreamError)
	require.NotContains(t, task.PrivateData.UpstreamError, "cdn.example.com")
	require.NotContains(t, task.PrivateData.UpstreamError, "private/file.png")
	require.NotContains(t, task.PrivateData.UpstreamError, "token=secret")
}

func TestSetVideoTaskFailureUsesErrorCodeForConsistentTranslation(t *testing.T) {
	task := &model.Task{Status: model.TaskStatusFailure}

	SetVideoTaskFailure(task, "EOF", "do_request_failed", http.StatusInternalServerError)

	require.Equal(t, "调用上游视频服务失败，请稍后重试；如持续失败，请联系管理员。", task.FailReason)
	require.Equal(t, "EOF", task.PrivateData.UpstreamError)
}

func TestSanitizeTaskRawErrorMasksCredentials(t *testing.T) {
	raw := `authorization: Bearer sk-secret api_key="api-secret" access_token=token-secret`

	sanitized := sanitizeTaskRawError(raw)

	require.NotContains(t, sanitized, "sk-secret")
	require.NotContains(t, sanitized, "api-secret")
	require.NotContains(t, sanitized, "token-secret")
}

func TestVideoTaskFailureMessagesTranslatesHistoricalRawReason(t *testing.T) {
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: `{"error":{"message":"invalid audio_url: audio url returned 404"}}`,
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, "参考音频链接不存在或已失效，请重新上传音频后再试。", message)
	require.Contains(t, raw, "audio url returned 404")
}

func TestVideoTaskFailureMessagesDoesNotInventRawErrorForHistoricalLocalizedReason(t *testing.T) {
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "任务超时，请稍后重试。",
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, task.FailReason, message)
	require.Empty(t, raw)
}

func TestVideoTaskFailureMessagesRepairsPreviouslyTranslatedChineseUpstreamMessage(t *testing.T) {
	upstreamMessage := "audio_urls 共 3 个参考音频，总时长 16.80 秒，超过总和 15 秒上限，请裁剪后重试"
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。",
		PrivateData: model.TaskPrivateData{
			UpstreamError: upstreamMessage,
		},
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, upstreamMessage, message)
	require.Equal(t, upstreamMessage, raw)
}

func TestVideoTaskFailureMessagesRecoversChineseUpstreamMessageFromHistoricalData(t *testing.T) {
	rawReason := "内容触发安全审核或版权限制，请调整输入内容或素材后重试"
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。",
		Data: []byte(
			`{"created_at":1786108415,"status":"FAILED: 内容触发安全审核或版权限制，请调整输入内容或素材后重试"}`,
		),
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, rawReason, message)
	require.Equal(t, rawReason, raw)
}

func TestVideoTaskFailureMessagesKeepsCreditNormalizationForHistoricalChineseData(t *testing.T) {
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。",
		Data:       []byte(`{"error":"上游积分不足，请充值后重试"}`),
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, "积分不足，请联系管理员", message)
	require.Equal(t, "上游积分不足，请充值后重试", raw)
}

func TestVideoTaskFailureMessagesIgnoresUnrelatedHistoricalDataStrings(t *testing.T) {
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "视频生成失败，请稍后重试。",
		Data:       []byte(`{"data":{"status":"failed","video_url":"https://cdn.example.com/failed.mp4"}}`),
	}

	message, raw := VideoTaskFailureMessages(task)

	require.Equal(t, task.FailReason, message)
	require.Empty(t, raw)
}
