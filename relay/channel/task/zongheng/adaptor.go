package zongheng

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// Video-only channel: no chat/StreamOptions protocol is advertised here.
type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL, apiKey, proxy string
}

type taskResponse struct {
	Code            int    `json:"code"`
	Success         *bool  `json:"success"`
	TaskID          string `json:"task_id"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	VideoURL        string `json:"video_url"`
	URL             string `json:"url"`
	ResultURL       string `json:"result_url"`
	DownloadURL     string `json:"download_url"`
	ErrorCode       string `json:"error_code"`
	ErrorDetail     string `json:"error_detail"`
	Message         string `json:"message"`
	NextPollSeconds int    `json:"next_poll_seconds"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = service.NormalizeZonghengBaseURL(info.ChannelBaseUrl)
	a.apiKey = info.ApiKey
	a.proxy = info.ChannelSetting.Proxy
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	info.UpstreamEndpoint = "/v1/videos"
	return a.baseURL + "/v1/videos", nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if info.PublicTaskID == "" {
		return fmt.Errorf("public task ID is required for idempotent submission")
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", info.PublicTaskID)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	cacheKey := "zongheng_prepared:" + a.requestIdentity(info.UpstreamModelName)
	if cached, ok := c.Get(cacheKey); ok {
		return bytes.NewReader(cached.([]byte)), nil
	}
	value, exists := c.Get(requestContextKey)
	req, ok := value.(videoRequest)
	if !exists || !ok {
		return nil, fmt.Errorf("validated Zongheng request is unavailable")
	}
	if info.UpstreamModelName != "" {
		req.Model = info.UpstreamModelName
	}
	if err := a.prepareReferences(c, &req); err != nil {
		return nil, err
	}
	data, err := common.Marshal(req)
	if err != nil {
		return nil, err
	}
	c.Set(cacheKey, data)
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	if value, ok := c.Get(requestContextKey); ok {
		if req, ok := value.(videoRequest); ok && req.Duration != nil {
			return map[string]float64{"seconds": float64(*req.Duration)}
		}
	}
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, body)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	defer resp.Body.Close()
	body, err := common.ReadResponseBodyWithLimit(resp.Body, common.MaxUpstreamResponseBodyBytes)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusBadGateway)
	}
	task, err := parseResponse(body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "invalid_response", http.StatusBadGateway)
	}
	if task.TaskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("Zongheng response has no task_id"), "invalid_response", http.StatusBadGateway)
	}
	// The raw response retains the upstream ID; clients only see the gateway ID.
	video := dto.NewOpenAIVideo()
	video.ID, video.TaskID = info.PublicTaskID, info.PublicTaskID
	video.Model, video.CreatedAt = info.OriginModelName, common.GetTimestamp()
	video.Status = "in_progress"
	c.JSON(http.StatusOK, video)
	return task.TaskID, body, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	id, _ := body["task_id"].(string)
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	req, err := http.NewRequest(http.MethodGet, service.NormalizeZonghengBaseURL(baseURL)+"/v1/tasks/"+url.PathEscape(id), nil)
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

func parseResponse(body []byte) (*taskResponse, error) {
	var task taskResponse
	if err := common.Unmarshal(body, &task); err != nil {
		return nil, err
	}
	if task.Code != 0 || (task.Success != nil && !*task.Success) {
		return nil, fmt.Errorf("Zongheng API error: %s", taskcommon.DefaultString(task.ErrorDetail, task.Message))
	}
	return &task, nil
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	task, err := parseResponse(body)
	if err != nil {
		return nil, err
	}
	result := &relaycommon.TaskInfo{TaskID: task.TaskID}
	switch task.Status {
	case "processing":
		result.Status = model.TaskStatusInProgress
	case "succeeded":
		result.Url = task.resultURL()
		if result.Url == "" {
			return nil, fmt.Errorf("Zongheng succeeded response has no video URL")
		}
		result.Status, result.Progress = model.TaskStatusSuccess, "100%"
	case "failed":
		result.Status, result.Progress = model.TaskStatusFailure, "100%"
		result.Reason = taskcommon.DefaultString(task.ErrorDetail, "video generation failed")
		if task.ErrorCode != "" {
			result.Reason = task.ErrorCode + ": " + result.Reason
		}
	default:
		return nil, fmt.Errorf("unrecognized Zongheng task status %q", task.Status)
	}
	return result, nil
}

func (task *taskResponse) resultURL() string {
	for _, value := range []string{task.VideoURL, task.URL, task.ResultURL, task.DownloadURL} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	video := task.ToOpenAIVideo()
	video.TaskID = task.TaskID
	if task.Status == model.TaskStatusSuccess && task.GetResultURL() != "" {
		video.SetMetadata("url", task.GetResultURL())
	}
	if task.Status == model.TaskStatusFailure {
		video.Error = &dto.OpenAIVideoError{Message: task.FailReason}
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) GetModelList() []string { return nil }
func (a *TaskAdaptor) GetChannelName() string { return "zongheng" }
