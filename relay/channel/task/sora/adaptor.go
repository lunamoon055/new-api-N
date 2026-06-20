package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string         `json:"id"`
	TaskID             string         `json:"task_id,omitempty"` //兼容旧接口
	Object             string         `json:"object"`
	Model              string         `json:"model"`
	Status             string         `json:"status"`
	Progress           int            `json:"progress"`
	Created            int64          `json:"created,omitempty"`
	CreatedAt          int64          `json:"created_at"`
	CompletedAt        int64          `json:"completed_at,omitempty"`
	ExpiresAt          int64          `json:"expires_at,omitempty"`
	Seconds            string         `json:"seconds,omitempty"`
	Size               string         `json:"size,omitempty"`
	RemixedFromVideoID string         `json:"remixed_from_video_id,omitempty"`
	URL                string         `json:"url,omitempty"`
	ResultURL          string         `json:"result_url,omitempty"`
	OutputURL          string         `json:"output_url,omitempty"`
	VideoURL           string         `json:"video_url,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func isAsyncGenerationsModel(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "sora2", "sora-2", "kling-v3", "video-2.0", "video-2.0-fast", "ko3",
		"veo31", "veo31-fast", "veo31-ref", "grok-imagine-video":
		return true
	default:
		return false
	}
}

func isAsyncGenerationsPath(path string) bool {
	return strings.HasPrefix(strings.Split(path, "?")[0], "/v1/video/async-generations")
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if getTaskAction(info) == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return validateVideo2JSONRequest(c, req.Model)
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if getTaskAction(info) == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if getTaskAction(info) == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, getOriginTaskID(info)), nil
	}
	if isAsyncGenerationsPath(info.RequestURLPath) || isAsyncGenerationsModel(info.UpstreamModelName) {
		return fmt.Sprintf("%s/v1/video/async-generations", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

func getTaskAction(info *relaycommon.RelayInfo) string {
	if info == nil || info.TaskRelayInfo == nil {
		return ""
	}
	return info.Action
}

func getOriginTaskID(info *relaycommon.RelayInfo) string {
	if info == nil || info.TaskRelayInfo == nil {
		return ""
	}
	return info.OriginTaskID
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			bodyMap["model"] = info.UpstreamModelName
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	rawPayload := responsePayloadMap(responseBody)
	if upstreamID == "" {
		upstreamID = extractTaskID(dResp, rawPayload)
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	if dResp.Status == "" {
		dResp.Status = extractTaskStatus(dResp, rawPayload)
	}
	if dResp.Progress == 0 {
		dResp.Progress = extractTaskProgress(dResp, rawPayload)
	}
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	modelName, _ := body["model"].(string)
	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	if isAsyncGenerationsModel(modelName) {
		uri = fmt.Sprintf("%s/v1/video/async-generations/%s", baseUrl, taskID)
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}
	rawPayload := responsePayloadMap(respBody)
	taskResult.TaskID = extractTaskIDForResult(resTask, rawPayload)

	switch normalizeTaskStatus(extractTaskStatus(resTask, rawPayload)) {
	case "queued":
		taskResult.Status = model.TaskStatusQueued
	case "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = extractVideoURLFromPayload(resTask, rawPayload)
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Reason = extractTaskErrorReason(resTask, rawPayload)
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if progress := extractTaskProgress(resTask, rawPayload); progress > 0 && progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	resTask := responseTask{}
	var rawPayload map[string]any
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &resTask); err != nil {
			return nil, errors.Wrap(err, "unmarshal sora task data failed")
		}
		rawPayload = responsePayloadMap(task.Data)
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.TaskID = task.TaskID
	openAIVideo.Model = taskcommon.DefaultString(
		taskcommon.DefaultString(resTask.Model, stringValue(taskResponsePayload(rawPayload), "model")),
		task.Properties.OriginModelName,
	)
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	if progress := extractTaskProgress(resTask, rawPayload); openAIVideo.Progress == 0 && progress > 0 {
		openAIVideo.Progress = progress
	}
	openAIVideo.CreatedAt = firstNonZeroInt64(resTask.CreatedAt, resTask.Created, task.CreatedAt)
	openAIVideo.CompletedAt = firstNonZeroInt64(resTask.CompletedAt, task.FinishTime, task.UpdatedAt)
	openAIVideo.ExpiresAt = resTask.ExpiresAt
	openAIVideo.Seconds = resTask.Seconds
	openAIVideo.Size = resTask.Size
	openAIVideo.RemixedFromVideoID = resTask.RemixedFromVideoID

	if resTask.Error != nil {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: resTask.Error.Message,
			Code:    resTask.Error.Code,
		}
	} else if task.Status == model.TaskStatusFailure && task.FailReason != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: task.FailReason,
		}
	}

	videoURL := extractVideoURLFromPayload(resTask, rawPayload)
	if videoURL == "" && task.Status == model.TaskStatusSuccess {
		videoURL = task.GetResultURL()
		if videoURL == "" {
			videoURL = taskcommon.BuildProxyURL(task.TaskID)
		}
	}
	if videoURL != "" {
		openAIVideo.SetMetadata("url", videoURL)
	}

	return common.Marshal(openAIVideo)
}

func responsePayloadMap(body []byte) map[string]any {
	var raw map[string]any
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil
	}
	return raw
}

func extractTaskID(resTask responseTask, raw map[string]any) string {
	if resTask.ID != "" {
		return strings.TrimSpace(resTask.ID)
	}
	if resTask.TaskID != "" {
		return strings.TrimSpace(resTask.TaskID)
	}
	if payload := taskResponsePayload(raw); payload != nil {
		return firstStringValue(payload, "id", "task_id", "taskId")
	}
	return ""
}

func extractTaskIDForResult(resTask responseTask, raw map[string]any) string {
	if resTask.TaskID != "" {
		return strings.TrimSpace(resTask.TaskID)
	}
	if resTask.ID != "" {
		return strings.TrimSpace(resTask.ID)
	}
	if payload := taskResponsePayload(raw); payload != nil {
		return firstStringValue(payload, "task_id", "id", "taskId")
	}
	return ""
}

func extractTaskStatus(resTask responseTask, raw map[string]any) string {
	if resTask.Status != "" {
		return strings.TrimSpace(resTask.Status)
	}
	if payload := taskResponsePayload(raw); payload != nil {
		return firstStringValue(payload, "status", "state")
	}
	return ""
}

func extractTaskProgress(resTask responseTask, raw map[string]any) int {
	if resTask.Progress > 0 {
		return resTask.Progress
	}
	if payload := taskResponsePayload(raw); payload != nil {
		return intValue(payload, "progress", "percent")
	}
	return 0
}

func normalizeTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "pending", "submitted", "created", "waiting":
		return "queued"
	case "processing", "in_progress", "running", "generating":
		return "in_progress"
	case "completed", "succeeded", "success", "done", "finished", "finish":
		return "completed"
	case "failed", "fail", "failure", "cancelled", "canceled", "error":
		return "failed"
	default:
		return ""
	}
}

func extractTaskErrorReason(resTask responseTask, raw map[string]any) string {
	if resTask.Error != nil {
		return strings.TrimSpace(resTask.Error.Message)
	}
	for _, values := range []map[string]any{taskResponsePayload(raw), raw} {
		if values == nil {
			continue
		}
		if errorValues := mapValue(values, "error"); errorValues != nil {
			if reason := firstStringValue(errorValues, "message", "code", "type"); reason != "" {
				return reason
			}
		}
		if reason := firstStringValue(values, "error", "message", "code"); reason != "" {
			return reason
		}
	}
	return ""
}

func taskResponsePayload(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	if hasTaskResponseFields(raw) {
		return raw
	}
	for _, key := range []string{"data", "result", "response"} {
		if nested := mapValue(raw, key); nested != nil {
			if hasTaskResponseFields(nested) {
				return nested
			}
			if payload := taskResponsePayload(nested); hasTaskResponseFields(payload) {
				return payload
			}
		}
	}
	return raw
}

func hasTaskResponseFields(values map[string]any) bool {
	if values == nil {
		return false
	}
	return firstStringValue(values,
		"id",
		"task_id",
		"taskId",
		"status",
		"state",
		"url",
		"video_url",
		"result_url",
		"output_url",
	) != "" || intValue(values, "progress", "percent") > 0 || mapValue(values, "error") != nil
}

func extractVideoURL(resTask responseTask) string {
	for _, url := range []string{
		resTask.VideoURL,
		resTask.URL,
		resTask.OutputURL,
		resTask.ResultURL,
		stringValue(resTask.Metadata, "video_url"),
		stringValue(resTask.Metadata, "url"),
		stringValue(resTask.Metadata, "output_url"),
		stringValue(resTask.Metadata, "result_url"),
	} {
		if strings.TrimSpace(url) != "" {
			return strings.TrimSpace(url)
		}
	}
	return ""
}

func extractVideoURLFromPayload(resTask responseTask, raw map[string]any) string {
	if url := extractVideoURL(resTask); url != "" {
		return url
	}
	for _, values := range []map[string]any{taskResponsePayload(raw), raw} {
		if values == nil {
			continue
		}
		if url := firstStringValue(values, "video_url", "url", "result_url", "output_url", "download_url"); url != "" {
			return url
		}
		if metadata := mapValue(values, "metadata"); metadata != nil {
			if url := firstStringValue(metadata, "video_url", "url", "result_url", "output_url", "download_url"); url != "" {
				return url
			}
		}
		if url := firstURLFromArrayFields(values, "data", "outputs", "output", "results", "videos", "urls", "video_urls"); url != "" {
			return url
		}
		if result := mapValue(values, "result"); result != nil {
			if url := firstStringValue(result, "video_url", "url", "result_url", "output_url", "download_url"); url != "" {
				return url
			}
			if url := firstURLFromArrayFields(result, "data", "outputs", "output", "results", "videos", "urls", "video_urls"); url != "" {
				return url
			}
		}
	}
	return ""
}

func firstStringValue(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if value := stringFromAny(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func intValue(values map[string]any, keys ...string) int {
	if values == nil {
		return 0
	}
	for _, key := range keys {
		switch value := values[key].(type) {
		case int:
			if value > 0 {
				return value
			}
		case int64:
			if value > 0 {
				return int(value)
			}
		case float64:
			if value > 0 {
				return int(value)
			}
		case string:
			trimmed := strings.TrimSuffix(strings.TrimSpace(value), "%")
			if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func mapValue(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	nested, _ := values[key].(map[string]any)
	return nested
}

func firstURLFromArrayFields(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if url := firstURLFromAny(values[key]); url != "" {
			return url
		}
	}
	return ""
}

func firstURLFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		for _, item := range v {
			if url := firstURLFromAny(item); url != "" {
				return url
			}
		}
	case map[string]any:
		if url := firstStringValue(v, "video_url", "url", "result_url", "output_url", "download_url"); url != "" {
			return url
		}
		if metadata := mapValue(v, "metadata"); metadata != nil {
			if url := firstStringValue(metadata, "video_url", "url", "result_url", "output_url", "download_url"); url != "" {
				return url
			}
		}
	}
	return ""
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
