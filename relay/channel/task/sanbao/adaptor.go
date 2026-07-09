package sanbao

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

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

const sanbaoRequestContextKey = "sanbao_task_request"

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

type requestPayload struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Ratio       string `json:"ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
	Reference   string `json:"reference,omitempty"`
	Quality     string `json:"quality,omitempty"`
	Images      []any  `json:"images,omitempty"`
	Videos      []any  `json:"videos,omitempty"`
	Audios      []any  `json:"audios,omitempty"`
}

type incomingRequest struct {
	Model          string           `json:"model"`
	Prompt         string           `json:"prompt"`
	Ratio          string           `json:"ratio,omitempty"`
	AspectRatio    string           `json:"aspect_ratio,omitempty"`
	AspectRatioAlt string           `json:"aspectRatio,omitempty"`
	Resolution     string           `json:"resolution,omitempty"`
	Duration       int              `json:"duration,omitempty"`
	Concurrency    int              `json:"concurrency,omitempty"`
	Reference      string           `json:"reference,omitempty"`
	Quality        string           `json:"quality,omitempty"`
	Image          referenceValue   `json:"image,omitempty"`
	Images         []referenceValue `json:"images,omitempty"`
	ImageURL       referenceValue   `json:"image_url,omitempty"`
	ImageURLs      []referenceValue `json:"image_urls,omitempty"`
	InputReference referenceValue   `json:"input_reference,omitempty"`
	StartImageURL  referenceValue   `json:"start_image_url,omitempty"`
	EndImageURL    referenceValue   `json:"end_image_url,omitempty"`
	VideoURL       referenceValue   `json:"video_url,omitempty"`
	VideoReference []referenceValue `json:"video_reference,omitempty"`
	AudioURL       referenceValue   `json:"audio_url,omitempty"`
}

type referenceValue struct {
	URL string `json:"url,omitempty"`
	Tag string `json:"tag,omitempty"`
}

func (r *referenceValue) UnmarshalJSON(data []byte) error {
	var value string
	if err := common.Unmarshal(data, &value); err == nil {
		r.URL = strings.TrimSpace(value)
		return nil
	}

	var obj struct {
		URL        string `json:"url"`
		Tag        string `json:"tag"`
		PreviewURL string `json:"previewUrl"`
	}
	if err := common.Unmarshal(data, &obj); err != nil {
		return err
	}
	r.URL = strings.TrimSpace(obj.URL)
	r.Tag = strings.TrimSpace(obj.Tag)
	return nil
}

type sanbaoEnvelope struct {
	Data sanbaoTask `json:"data"`
}

type sanbaoTask struct {
	ID            string         `json:"id"`
	TaskID        string         `json:"task_id"`
	Model         string         `json:"model"`
	Status        string         `json:"status"`
	Progress      int            `json:"progress"`
	Ratio         string         `json:"ratio"`
	Resolution    string         `json:"resolution"`
	Duration      int            `json:"duration"`
	ImageURL      string         `json:"image_url"`
	Images        []any          `json:"images"`
	VideoURL      string         `json:"video_url"`
	DownloadURL   string         `json:"download_url"`
	LinkExpiresIn int64          `json:"link_expires_in"`
	Cost          float64        `json:"cost"`
	Error         any            `json:"error"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	Metadata      map[string]any `json:"metadata"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = normalizeSanbaoBaseURL(info.ChannelBaseUrl)
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	req, err := parseIncomingRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Model) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest)
	}

	if isImageRequest(info) {
		info.Action = taskActionImageGenerate
	} else if req.hasReferenceAssets() {
		info.Action = constant.TaskActionGenerate
	} else {
		info.Action = constant.TaskActionTextGenerate
	}
	c.Set(sanbaoRequestContextKey, req)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:     req.Prompt,
		Model:      req.Model,
		Duration:   req.Duration,
		Resolution: req.Resolution,
	})
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if isImageRequest(info) {
		return a.baseURL + imageEndpoint, nil
	}
	return a.baseURL + videoEndpoint, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := getIncomingRequest(c)
	if err != nil {
		return nil, err
	}
	payload := req.toPayload(info)
	if err := a.uploadPayloadReferences(c, &payload); err != nil {
		return nil, err
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	task, err := parseSanbaoTask(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	upstreamID := task.taskID()
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = taskcommon.DefaultString(info.OriginModelName, task.Model)
	ov.Status = normalizeVideoStatus(task.Status)
	ov.Progress = task.Progress
	c.JSON(http.StatusOK, ov)
	return upstreamID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	endpoint := videoEndpoint
	if isImageFetchRequest(body) {
		endpoint = imageEndpoint
	}
	uri := fmt.Sprintf("%s%s/%s", normalizeSanbaoBaseURL(baseUrl), endpoint, taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	task, err := parseSanbaoTask(respBody)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code:     0,
		TaskID:   task.taskID(),
		Progress: progressString(task.Progress),
	}
	switch normalizeTaskStatus(task.Status) {
	case "queued":
		taskResult.Status = model.TaskStatusQueued
	case "processing":
		taskResult.Status = model.TaskStatusInProgress
	case "succeeded":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = task.resultURL()
		if taskResult.Progress == "" {
			taskResult.Progress = "100%"
		}
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Reason = task.errorMessage()
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
		if taskResult.Progress == "" {
			taskResult.Progress = "100%"
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
	}
	return &taskResult, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	openAIVideo := originTask.ToOpenAIVideo()
	var task sanbaoTask
	if len(originTask.Data) > 0 {
		parsed, err := parseSanbaoTask(originTask.Data)
		if err == nil {
			task = parsed
		}
	}
	if url := task.resultURL(); url != "" {
		openAIVideo.SetMetadata("url", url)
	}
	if task.Error != nil && originTask.Status == model.TaskStatusFailure {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: task.errorMessage(),
		}
	}
	return common.Marshal(openAIVideo)
}

func parseIncomingRequest(c *gin.Context) (incomingRequest, error) {
	var req incomingRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return req, err
	}
	return req, nil
}

func getIncomingRequest(c *gin.Context) (incomingRequest, error) {
	if value, ok := c.Get(sanbaoRequestContextKey); ok {
		if req, ok := value.(incomingRequest); ok {
			return req, nil
		}
	}
	return parseIncomingRequest(c)
}

func (req incomingRequest) toPayload(info *relaycommon.RelayInfo) requestPayload {
	modelName := req.Model
	if info != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}

	payload := requestPayload{
		Model:       modelName,
		Prompt:      req.Prompt,
		Ratio:       req.normalizedRatio(),
		Resolution:  req.Resolution,
		Duration:    req.Duration,
		Concurrency: req.Concurrency,
		Reference:   req.normalizedReference(),
		Quality:     req.Quality,
		Images:      req.imageAssets(),
		Videos:      req.videoAssets(),
		Audios:      req.audioAssets(),
	}
	return payload
}

func (req incomingRequest) normalizedRatio() string {
	for _, value := range []string{req.Ratio, req.AspectRatio, req.AspectRatioAlt} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (req incomingRequest) normalizedReference() string {
	if strings.TrimSpace(req.Reference) != "" {
		return strings.TrimSpace(req.Reference)
	}
	if req.hasReferenceAssets() {
		return "all"
	}
	return ""
}

func (req incomingRequest) hasReferenceAssets() bool {
	return len(req.imageAssets()) > 0 || len(req.videoAssets()) > 0 || len(req.audioAssets()) > 0
}

func (req incomingRequest) imageAssets() []any {
	values := make([]referenceValue, 0, 1+len(req.Images)+1+len(req.ImageURLs)+2)
	values = append(values, req.Image, req.InputReference, req.ImageURL)
	values = append(values, req.Images...)
	values = append(values, req.ImageURLs...)
	values = append(values, req.StartImageURL, req.EndImageURL)
	return taggedAssets(values, "参考图片")
}

func (req incomingRequest) videoAssets() []any {
	values := make([]referenceValue, 0, 1+len(req.VideoReference))
	values = append(values, req.VideoURL)
	values = append(values, req.VideoReference...)
	return taggedAssets(values, "参考视频")
}

func (req incomingRequest) audioAssets() []any {
	return taggedAssets([]referenceValue{req.AudioURL}, "参考音频")
}

func taggedAssets(values []referenceValue, tagPrefix string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.URL) == "" {
			continue
		}
		tag := strings.TrimSpace(value.Tag)
		if tag == "" {
			if tagPrefix == "参考音频" {
				tag = tagPrefix
			} else {
				tag = fmt.Sprintf("%s%d", tagPrefix, len(result)+1)
			}
		}
		result = append(result, map[string]any{
			"tag": tag,
			"url": strings.TrimSpace(value.URL),
		})
	}
	return result
}

func (a *TaskAdaptor) uploadPayloadReferences(c *gin.Context, payload *requestPayload) error {
	if payload == nil || a.apiKey == "" || a.baseURL == "" {
		return nil
	}
	if err := a.uploadAssets(c, payload.Images, "image"); err != nil {
		return err
	}
	if err := a.uploadAssets(c, payload.Videos, "video"); err != nil {
		return err
	}
	if err := a.uploadAssets(c, payload.Audios, "audio"); err != nil {
		return err
	}
	return nil
}

func (a *TaskAdaptor) uploadAssets(c *gin.Context, assets []any, kind string) error {
	for _, asset := range assets {
		values, ok := asset.(map[string]any)
		if !ok {
			continue
		}
		assetURL := firstString(values, "url")
		if assetURL == "" {
			continue
		}
		uploadedURL, err := a.uploadReferenceURL(c, kind, assetURL)
		if err != nil {
			return err
		}
		if uploadedURL != "" {
			values["url"] = uploadedURL
		}
	}
	return nil
}

func (a *TaskAdaptor) uploadReferenceURL(c *gin.Context, kind, assetURL string) (string, error) {
	data, contentType, fileName, err := readReferenceAsset(c, assetURL)
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = defaultContentType(kind)
	}
	if fileName == "" {
		fileName = defaultFileName(kind, contentType)
	}

	uploadURL := fmt.Sprintf(
		"%s%s?kind=%s&fileName=%s",
		strings.TrimRight(a.baseURL, "/"),
		uploadEndpoint,
		url.QueryEscape(kind),
		url.QueryEscape(fileName),
	)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", contentType)

	client, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return "", fmt.Errorf("new http client failed: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("sanbao upload failed: status %d body: %s", resp.StatusCode, respBody)
	}
	uploadedURL := extractUploadURL(respBody)
	if uploadedURL == "" {
		return "", fmt.Errorf("sanbao upload response missing file url")
	}
	return uploadedURL, nil
}

func readReferenceAsset(c *gin.Context, assetURL string) ([]byte, string, string, error) {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return nil, "", "", fmt.Errorf("reference url is empty")
	}
	if strings.HasPrefix(strings.ToLower(assetURL), "data:") {
		return readDataURL(assetURL)
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	client, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, "", "", fmt.Errorf("new http client failed: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", "", fmt.Errorf("download reference failed: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}
	contentType := normalizeContentType(resp.Header.Get("Content-Type"))
	fileName := fileNameFromURL(assetURL)
	return data, contentType, fileName, nil
}

func readDataURL(value string) ([]byte, string, string, error) {
	separator := strings.Index(value, ",")
	if separator < 0 {
		return nil, "", "", fmt.Errorf("invalid data url")
	}
	meta := strings.TrimPrefix(value[:separator], "data:")
	rawData := value[separator+1:]
	contentType := meta
	if semi := strings.Index(contentType, ";"); semi >= 0 {
		contentType = contentType[:semi]
	}
	if strings.Contains(strings.ToLower(meta), ";base64") {
		data, err := base64.StdEncoding.DecodeString(rawData)
		if err != nil {
			return nil, "", "", err
		}
		return data, normalizeContentType(contentType), "", nil
	}
	unescaped, err := url.QueryUnescape(rawData)
	if err != nil {
		return nil, "", "", err
	}
	return []byte(unescaped), normalizeContentType(contentType), "", nil
}

func extractUploadURL(body []byte) string {
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if file := mapValue(payload, "file"); file != nil {
		if value := firstString(file, "url"); value != "" {
			return value
		}
	}
	if data := mapValue(payload, "data"); data != nil {
		if value := firstString(data, "url"); value != "" {
			return value
		}
		if file := mapValue(data, "file"); file != nil {
			if value := firstString(file, "url"); value != "" {
				return value
			}
		}
	}
	return firstString(payload, "url")
}

func fileNameFromURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func defaultFileName(kind, contentType string) string {
	extension := ".bin"
	if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
		extension = exts[0]
	}
	return fmt.Sprintf("reference-%s%s", kind, extension)
}

func defaultContentType(kind string) string {
	switch kind {
	case "image":
		return "image/png"
	case "video":
		return "video/mp4"
	case "audio":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

func normalizeContentType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if separator := strings.Index(value, ";"); separator >= 0 {
		value = value[:separator]
	}
	return strings.TrimSpace(value)
}

func normalizeSanbaoBaseURL(value string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(value), "/")
	if strings.HasSuffix(baseURL, "/openapi/v1") {
		return strings.TrimSuffix(baseURL, "/openapi/v1")
	}
	return baseURL
}

func parseSanbaoTask(body []byte) (sanbaoTask, error) {
	var envelope sanbaoEnvelope
	if err := common.Unmarshal(body, &envelope); err == nil && envelope.Data.taskID() != "" {
		return envelope.Data, nil
	}
	var task sanbaoTask
	if err := common.Unmarshal(body, &task); err != nil {
		return task, err
	}
	return task, nil
}

func (task sanbaoTask) taskID() string {
	if strings.TrimSpace(task.ID) != "" {
		return strings.TrimSpace(task.ID)
	}
	return strings.TrimSpace(task.TaskID)
}

func (task sanbaoTask) resultURL() string {
	for _, value := range []string{task.DownloadURL, task.VideoURL, task.ImageURL, stringValue(task.Metadata, "download_url"), stringValue(task.Metadata, "video_url"), stringValue(task.Metadata, "image_url"), stringValue(task.Metadata, "url")} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return firstAssetURL(task.Images)
}

func (task sanbaoTask) errorMessage() string {
	switch value := task.Error.(type) {
	case string:
		return strings.TrimSpace(value)
	case map[string]any:
		return firstString(value, "message", "error", "code", "type")
	default:
		return ""
	}
}

func normalizeTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "pending", "submitted", "created", "waiting":
		return "queued"
	case "processing", "running", "in_progress", "generating":
		return "processing"
	case "succeeded", "completed", "success", "done", "finished", "finish":
		return "succeeded"
	case "failed", "failure", "fail", "error", "cancelled", "canceled":
		return "failed"
	default:
		return ""
	}
}

func normalizeVideoStatus(status string) string {
	switch normalizeTaskStatus(status) {
	case "queued":
		return dto.VideoStatusQueued
	case "processing":
		return dto.VideoStatusInProgress
	case "succeeded":
		return dto.VideoStatusCompleted
	case "failed":
		return dto.VideoStatusFailed
	default:
		return dto.VideoStatusQueued
	}
}

func progressString(progress int) string {
	if progress <= 0 {
		return ""
	}
	if progress > 100 {
		progress = 100
	}
	return fmt.Sprintf("%d%%", progress)
}

func isImageRequest(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	path := strings.ToLower(info.RequestURLPath)
	return strings.Contains(path, "/images")
}

func isImageFetchRequest(body map[string]any) bool {
	kind, _ := body["kind"].(string)
	if strings.EqualFold(strings.TrimSpace(kind), "image") {
		return true
	}
	action, _ := body["action"].(string)
	return strings.EqualFold(strings.TrimSpace(action), taskActionImageGenerate)
}

func firstAssetURL(values []any) string {
	for _, item := range values {
		switch value := item.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case map[string]any:
			if url := firstString(value, "url", "image_url", "download_url"); url != "" {
				return url
			}
		}
	}
	return ""
}

func firstString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func mapValue(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	nested, ok := values[key].(map[string]any)
	if !ok {
		return nil
	}
	return nested
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
