package v6k

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// V6K's asynchronous video API has no StreamOptions support.
type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL string
}

type videoResponse struct {
	ID       string                `json:"id"`
	Status   string                `json:"status"`
	Progress *float64              `json:"progress"`
	URL      string                `json:"url"`
	Error    *dto.OpenAIVideoError `json:"error"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = service.NormalizeV6KBaseURL(info.ChannelBaseUrl)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	info.UpstreamEndpoint = "/v1/videos"
	return a.baseURL + "/v1/videos", nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	if contentType := c.GetString("v6k_content_type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	if c.GetBool("v6k_submission_attempted") {
		return nil, fmt.Errorf("V6K submission already attempted; do not create a duplicate task")
	}
	endpoint, err := a.BuildRequestURL(info)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	// Deliberately do not install GetBody or an undocumented idempotency header.
	// The provider tells callers to query accepted tasks rather than re-submit.
	req.GetBody = nil
	if err := a.BuildRequestHeader(c, req, info); err != nil {
		return nil, err
	}
	client, err := service.GetHttpClientWithProxy(info.ChannelSetting.Proxy)
	if err != nil {
		return nil, err
	}
	c.Set("v6k_submission_attempted", true)
	submitClient := *client
	submitClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return submitClient.Do(req)
}

func parseResponse(body []byte) (*videoResponse, error) {
	var response videoResponse
	if err := common.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, fmt.Errorf("V6K response has no video id")
	}
	return &response, nil
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	defer resp.Body.Close()
	body, err := common.ReadResponseBodyWithLimit(resp.Body, common.MaxUpstreamResponseBodyBytes)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusBadGateway)
	}
	response, err := parseResponse(body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "invalid_response", http.StatusBadGateway)
	}
	video := dto.NewOpenAIVideo()
	video.ID, video.TaskID, video.Model = info.PublicTaskID, info.PublicTaskID, info.OriginModelName
	video.Status, video.CreatedAt = dto.VideoStatusQueued, common.GetTimestamp()
	c.JSON(http.StatusOK, video)
	return response.ID, body, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	id, _ := body["task_id"].(string)
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("video task id is required")
	}
	req, err := http.NewRequest(http.MethodGet, service.NormalizeV6KBaseURL(baseURL)+"/v1/videos/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	response, err := parseResponse(body)
	if err != nil {
		return nil, err
	}
	result := &relaycommon.TaskInfo{TaskID: response.ID}
	if response.Progress != nil && !math.IsNaN(*response.Progress) && !math.IsInf(*response.Progress, 0) {
		result.Progress = fmt.Sprintf("%.0f%%", math.Max(0, math.Min(99, *response.Progress)))
	}
	switch response.Status {
	case "queued", "unknown":
		result.Status = model.TaskStatusQueued
	case "in_progress":
		result.Status = model.TaskStatusInProgress
	case "completed":
		result.Status, result.Progress = model.TaskStatusSuccess, "100%"
		result.Url = a.baseURL + "/v1/videos/" + url.PathEscape(response.ID) + "/content"
	case "failed":
		result.Status, result.Progress = model.TaskStatusFailure, "100%"
		result.Reason = "video generation failed"
		if response.Error != nil && response.Error.Message != "" {
			result.Reason = response.Error.Message
		}
	default:
		return nil, fmt.Errorf("unrecognized V6K video status %q", response.Status)
	}
	return result, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	video := task.ToOpenAIVideo()
	video.TaskID = task.TaskID
	if task.Status == model.TaskStatusSuccess {
		video.SetMetadata("url", taskcommon.BuildProxyURL(task.TaskID))
	}
	if task.Status == model.TaskStatusFailure {
		video.Error = &dto.OpenAIVideoError{Message: task.FailReason}
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) GetModelList() []string { return common.V6KVideoModelIDs() }
func (a *TaskAdaptor) GetChannelName() string { return "v6k" }

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	value, ok := c.Get("v6k_request")
	if !ok {
		return nil, fmt.Errorf("validated V6K request is unavailable")
	}
	payload := value.(map[string]any)
	copy := make(map[string]any, len(payload))
	for key, value := range payload {
		copy[key] = value
	}
	if info.UpstreamModelName != "" {
		copy["model"] = info.UpstreamModelName
	}
	if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		return buildMultipart(c, copy)
	}
	data, err := common.Marshal(copy)
	return bytes.NewReader(data), err
}
