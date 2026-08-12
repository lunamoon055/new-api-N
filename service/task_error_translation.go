package service

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const maxStoredTaskRawErrorRunes = 8000

var taskErrorSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)((?:authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|id[_-]?token|token|secret|password)\s*["']?\s*[:=]\s*["']?(?:bearer\s+)?)[^"'\s,}\]]+`),
	regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9._~+/=-]+`),
}

type TaskErrorTranslation struct {
	Message   string
	Category  string
	Retryable bool
}

func taskErrorTranslation(message string, category string, retryable bool) TaskErrorTranslation {
	return TaskErrorTranslation{
		Message:   message,
		Category:  category,
		Retryable: retryable,
	}
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}

func appendStructuredTaskErrorText(value any, depth int, values *[]string) {
	if depth > 5 || len(*values) >= 64 || value == nil {
		return
	}

	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return
		}
		*values = append(*values, trimmed)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			var nested any
			if common.Unmarshal([]byte(trimmed), &nested) == nil {
				appendStructuredTaskErrorText(nested, depth+1, values)
			}
		}
	case map[string]any:
		for _, key := range []string{"error", "message", "detail", "reason", "msg", "code", "type", "status"} {
			if nested, ok := typed[key]; ok {
				appendStructuredTaskErrorText(nested, depth+1, values)
			}
		}
	case []any:
		for _, nested := range typed {
			appendStructuredTaskErrorText(nested, depth+1, values)
		}
	case float64, bool:
		*values = append(*values, fmt.Sprint(typed))
	}
}

func taskErrorSearchText(rawMessage string, code string, statusCode int) string {
	parts := []string{rawMessage, code, strconv.Itoa(statusCode)}
	var payload any
	if common.Unmarshal([]byte(strings.TrimSpace(rawMessage)), &payload) == nil {
		appendStructuredTaskErrorText(payload, 0, &parts)
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func findChineseTaskErrorMessage(value any, depth int) string {
	if depth > 6 || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return ""
		}
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, `"`) {
			var nested any
			if common.Unmarshal([]byte(trimmed), &nested) == nil {
				if message := findChineseTaskErrorMessage(nested, depth+1); message != "" {
					return message
				}
			}
		}
		if containsHan(trimmed) {
			return trimmed
		}
	case map[string]any:
		// Prefer human-readable provider fields over codes and metadata.
		for _, key := range []string{"message", "msg", "detail", "reason", "error"} {
			if nested, ok := typed[key]; ok {
				if message := findChineseTaskErrorMessage(nested, depth+1); message != "" {
					return message
				}
			}
		}
	case []any:
		for _, nested := range typed {
			if message := findChineseTaskErrorMessage(nested, depth+1); message != "" {
				return message
			}
		}
	}

	return ""
}

func extractChineseTaskErrorMessage(rawMessage string) string {
	trimmed := strings.TrimSpace(rawMessage)
	if trimmed == "" {
		return ""
	}

	var payload any
	if common.Unmarshal([]byte(trimmed), &payload) == nil {
		if message := findChineseTaskErrorMessage(payload, 0); message != "" {
			return sanitizeTaskRawError(message)
		}
	}
	if containsHan(trimmed) {
		return sanitizeTaskRawError(trimmed)
	}
	return ""
}

func sanitizeTaskRawError(rawMessage string) string {
	masked := strings.TrimSpace(common.MaskSensitiveInfo(rawMessage))
	for _, pattern := range taskErrorSecretPatterns {
		masked = pattern.ReplaceAllString(masked, "${1}***")
	}
	if masked == "" {
		return ""
	}
	runes := []rune(masked)
	if len(runes) <= maxStoredTaskRawErrorRunes {
		return masked
	}
	return string(runes[:maxStoredTaskRawErrorRunes]) + "\n...[truncated]"
}

// TranslateVideoTaskError preserves already-readable Chinese provider messages
// and converts non-Chinese provider errors into stable customer-facing text.
func TranslateVideoTaskError(rawMessage string, code string, statusCode int) TaskErrorTranslation {
	text := taskErrorSearchText(rawMessage, code, statusCode)
	chineseMessage := extractChineseTaskErrorMessage(rawMessage)

	switch {
	case containsAny(text,
		"积分不足",
		"积分不够",
		"点数不足",
		"额度不足",
		"配额不足",
		"余额不足",
		"余额不够",
		"insufficient credit",
		"insufficient balance",
		"insufficient funds",
		"insufficient quota",
		"insufficient points",
		"credits insufficient",
		"balance insufficient",
		"not enough credit",
		"not enough balance",
		"not enough quota",
		"not enough points",
		"no credits",
		"out of credits",
		"credits exhausted",
		"quota exhausted",
		"credit balance is too low",
	):
		return taskErrorTranslation("积分不足，请联系管理员", "credits", false)
	case chineseMessage != "":
		return taskErrorTranslation(chineseMessage, "upstream_message", false)

	case containsAny(text, "image url returned 404", "image_url returned 404"):
		return taskErrorTranslation("参考图片链接不存在或已失效，请重新上传图片后再试。", "reference_media", false)
	case containsAny(text, "audio url returned 404", "audio_url returned 404"):
		return taskErrorTranslation("参考音频链接不存在或已失效，请重新上传音频后再试。", "reference_media", false)
	case containsAny(text, "video url returned 404", "video_url returned 404"):
		return taskErrorTranslation("参考视频链接不存在或已失效，请重新上传视频后再试。", "reference_media", false)
	case strings.Contains(text, "image") && containsAny(text, "graphql request failed", " eof"):
		return taskErrorTranslation("上游素材服务连接中断，图片暂时无法读取，请稍后重试或重新上传图片。", "reference_media", true)
	case strings.Contains(text, "invalid_height"):
		return taskErrorTranslation("参考视频的高度或分辨率不符合上游要求，请调整视频尺寸后重试。", "request_parameters", false)
	case strings.Contains(text, "duration_too_short") && strings.Contains(text, "audio"):
		return taskErrorTranslation("参考音频时长过短，不符合模型要求，请上传符合时长要求的音频。", "request_parameters", false)
	case strings.Contains(text, "duration_too_short") && strings.Contains(text, "video"):
		return taskErrorTranslation("参考视频时长过短，不符合模型要求，请上传符合时长要求的视频。", "request_parameters", false)
	case strings.Contains(text, "duration_too_short"):
		return taskErrorTranslation("参考素材时长过短，不符合模型要求，请重新上传符合时长要求的素材。", "request_parameters", false)
	case containsAny(text, "duration_too_long", "duration exceeds", "duration is too long"):
		return taskErrorTranslation("参考素材时长超过模型限制，请缩短后重新上传。", "request_parameters", false)
	case containsAny(text, "file too large", "payload too large", "request entity too large", "size exceeds"):
		return taskErrorTranslation("参考素材文件过大，请压缩文件或重新上传符合大小要求的素材。", "request_parameters", false)

	case strings.Contains(text, "trademark"):
		return taskErrorTranslation("内容审核未通过，可能涉及商标或品牌侵权，请修改提示词或参考素材。", "content_moderation", false)
	case strings.Contains(text, "child"):
		return taskErrorTranslation("请求处理超时，且内容可能涉及未成年人敏感内容，请修改提示词后重试。", "content_moderation", false)
	case strings.Contains(text, "nsfw") && strings.Contains(text, "extreme_violence"):
		return taskErrorTranslation("请求处理超时，且内容可能包含不适宜或极端暴力元素，请修改提示词后重试。", "content_moderation", false)
	case strings.Contains(text, "nsfw"):
		return taskErrorTranslation("内容审核未通过，可能包含不适宜内容，请修改提示词或参考素材。", "content_moderation", false)
	case containsAny(text, "inappropriate content", "terms of service"):
		return taskErrorTranslation("提示词可能包含不适宜内容，未通过内容审核，请修改提示词后重试。", "content_moderation", false)
	case containsAny(text, "provider_moderation_error", "moderation error", "content policy", "safety filter"):
		return taskErrorTranslation("内容审核未通过，请调整提示词或参考素材后重试。", "content_moderation", false)

	case containsAny(text, "missing prompt", "prompt is required") ||
		(strings.Contains(text, "field required") && strings.Contains(text, "prompt")):
		return taskErrorTranslation("视频提示词不能为空，请填写视频描述后重试。", "request_parameters", false)
	case strings.Contains(text, "provider_invalid_request"):
		return taskErrorTranslation("请求参数无效，请检查提示词、参考素材和生成参数。", "request_parameters", false)
	case containsAny(text, "unsupported duration", "unsupported resolution", "invalid resolution", "unsupported media", "unsupported file"):
		return taskErrorTranslation("当前模型不支持所选参数，请调整分辨率、时长或参考素材后重试。", "request_parameters", false)
	case strings.Contains(text, "invalid url") && containsAny(text, "/video/", "/videos/", "async-generations"):
		return taskErrorTranslation("当前视频渠道接口地址配置错误，请联系管理员检查渠道地址和模型映射。", "configuration", false)

	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden ||
		containsAny(text, "invalid api key", "unauthorized", "authentication failed", "permission denied"):
		return taskErrorTranslation("视频渠道鉴权失败，请联系管理员检查 API 密钥。", "authentication", false)
	case statusCode == http.StatusTooManyRequests || containsAny(text, "rate limit", "too many requests"):
		return taskErrorTranslation("当前视频服务繁忙，请稍后再试。", "rate_limit", true)
	case containsAny(text, "generation polling timed out", "polling timed out"):
		return taskErrorTranslation("视频生成等待超时，请稍后查询任务状态或重新发起生成。", "upstream_timeout", true)
	case strings.Contains(text, "provider_timeout"):
		return taskErrorTranslation("上游视频服务响应超时，请稍后重试。", "upstream_timeout", true)
	case containsAny(text, "timeout", "timed out", "deadline exceeded"):
		return taskErrorTranslation("上游视频服务响应超时，请稍后重试。", "upstream_timeout", true)
	case containsAny(text, "do_request_failed", "do request failed", "connection refused", "connection reset"):
		return taskErrorTranslation("调用上游视频服务失败，请稍后重试；如持续失败，请联系管理员。", "upstream_service", true)
	case strings.Contains(text, "provider_internal_error"):
		return taskErrorTranslation("上游视频服务发生内部错误，请稍后重试。", "upstream_service", true)

	case containsAny(text, "task_not_exist", "task not found"):
		return taskErrorTranslation("任务不存在或已失效，请检查任务 ID。", "task", false)
	case containsAny(text, "get_task_failed", "get tasks failed"):
		return taskErrorTranslation("任务状态查询失败，请稍后重试。", "task", true)
	case containsAny(text, "not_implemented", "invalid_api_platform"):
		return taskErrorTranslation("当前模型暂不支持该操作，请联系管理员。", "configuration", false)
	case containsAny(text, "invalid_channel_id", "task_channel_disable", "channel_no_available_key", "get_channel_failed"):
		return taskErrorTranslation("当前视频渠道暂不可用，请稍后重试或联系管理员。", "configuration", true)
	case containsAny(text, "model_price_error", "invalid_billing_config"):
		return taskErrorTranslation("模型计费配置异常，请联系管理员。", "configuration", false)
	case containsAny(text, "read_request_body_failed", "invalid_request"):
		return taskErrorTranslation("请求参数无效，请检查提示词、参考素材和生成参数。", "request_parameters", false)

	case containsAny(text, "an error occurred", "generate error"):
		return taskErrorTranslation("视频生成失败，上游服务返回未知错误，请稍后重试。", "generation", true)
	case containsAny(text, "generation status failed", "generation failed", "status failed"):
		return taskErrorTranslation("视频生成失败，请检查提示词和参考素材后重试。", "generation", true)
	case statusCode >= http.StatusInternalServerError:
		return taskErrorTranslation("上游视频服务暂时不可用，请稍后重试。", "upstream_service", true)
	default:
		return taskErrorTranslation("视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。", "generation", true)
	}
}

// LocalizeVideoTaskError mutates a TaskError just before persistence/response.
// RawMessage remains server-only and Data contains only safe structured hints.
func LocalizeVideoTaskError(taskError *dto.TaskError) {
	if taskError == nil {
		return
	}
	rawMessage := taskError.RawMessage
	if strings.TrimSpace(rawMessage) == "" {
		rawMessage = taskError.Message
	}
	taskError.RawMessage = sanitizeTaskRawError(rawMessage)
	translated := TranslateVideoTaskError(rawMessage, taskError.Code, taskError.StatusCode)
	taskError.Message = translated.Message
	taskError.Data = dto.TaskErrorPublicData{
		Category:  translated.Category,
		Retryable: translated.Retryable,
	}
}

// SetVideoTaskFailure stores the translated message publicly and the masked
// upstream error in TaskPrivateData for super-administrator diagnostics.
func SetVideoTaskFailure(task *model.Task, rawMessage string, code string, statusCode int) TaskErrorTranslation {
	translated := TranslateVideoTaskError(rawMessage, code, statusCode)
	if task == nil {
		return translated
	}
	task.FailReason = translated.Message
	task.PrivateData.UpstreamError = sanitizeTaskRawError(rawMessage)
	return translated
}

func containsHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func looksLikeStructuredTaskError(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(trimmed, "{") ||
		strings.HasPrefix(trimmed, "[") ||
		containsAny(trimmed, "provider_", "invalid_", "error", "failed", "timeout")
}

// VideoTaskFailureMessages supports both new tasks (private raw error plus
// translated FailReason) and historical tasks that stored only the raw error.
func VideoTaskFailureMessages(task *model.Task) (userMessage string, rawMessage string) {
	if task == nil {
		return "", ""
	}
	if task.Status != model.TaskStatusFailure {
		return task.FailReason, ""
	}

	rawMessage = strings.TrimSpace(task.PrivateData.UpstreamError)
	if rawMessage != "" {
		translated := TranslateVideoTaskError(rawMessage, "", 0)
		return translated.Message, rawMessage
	}

	if containsHan(task.FailReason) && !looksLikeStructuredTaskError(task.FailReason) {
		// Historical localized failures did not retain the upstream response.
		return task.FailReason, ""
	}
	rawMessage = sanitizeTaskRawError(task.FailReason)
	translated := TranslateVideoTaskError(rawMessage, "", 0)
	return translated.Message, rawMessage
}
