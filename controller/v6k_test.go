package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestV6KSubmissionCannotRetryAcrossChannels(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("v6k_submission_attempted", true)
	for _, status := range []int{429, 500, 502, 503, 307} {
		require.False(t, shouldRetryTaskRelay(c, 1, &dto.TaskError{StatusCode: status}, 3))
	}
}

func TestV6KChannelTestUsesUnifiedProtocolForCollidingModelNames(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeV6K}
	endpoint, path, _ := resolveChannelTestEndpoint(channel, "kling-v3", "")
	require.Equal(t, "openai-video", endpoint)
	require.Equal(t, "/v1/videos", path)
	request := buildTestRequest("dola2.0", endpoint, channel, false).(relaycommon.TaskSubmitReq)
	require.Equal(t, 15, request.Duration)
	require.Equal(t, "720p", request.Resolution)
	require.Empty(t, request.Size)
	require.Empty(t, request.Ratio)
}

func TestV6KZHNewModelsUseTheirOwnTestDefaults(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeV6K}
	count := 0
	for _, id := range common.V6KVideoModelIDs() {
		p := common.GetV6KValidatedVideoParameters(id)
		if p == nil {
			continue
		}
		count++
		endpoint, path, _ := resolveChannelTestEndpoint(channel, id, "")
		require.Equal(t, "openai-video", endpoint)
		require.Equal(t, "/v1/videos", path)
		request := buildTestRequest(id, endpoint, channel, false).(relaycommon.TaskSubmitReq)
		require.Equal(t, p.Durations[0], request.Duration)
		require.Equal(t, p.Resolutions[0], request.Resolution)
		require.Empty(t, request.Ratio)
	}
	require.Equal(t, 11, count)
}

func TestV6KContentProxyKeepsTaskCredentialAndSupportsRangeAndHead(t *testing.T) {
	setupVideoProxyControllerTestDB(t)
	service.InitHttpClient()
	setting := system_setting.GetFetchSetting()
	previous := *setting
	setting.EnableSSRFProtection = false
	t.Cleanup(func() { *setting = previous })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/videos/provider-id/content", r.URL.Path)
		require.Equal(t, "Bearer stored-task-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Accept-Ranges", "bytes")
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "6")
			return
		}
		require.Equal(t, "bytes=0-2", r.Header.Get("Range"))
		require.Equal(t, `"etag"`, r.Header.Get("If-Range"))
		w.Header().Set("Content-Range", "bytes 0-2/6")
		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, "abc")
	}))
	defer server.Close()
	require.NoError(t, model.DB.Create(&model.Channel{Id: 60, Type: constant.ChannelTypeV6K, Name: "v6k", Key: "different-current-key", BaseURL: common.GetPointer(server.URL + "/v1"), Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, model.DB.Create(&model.Task{TaskID: "task_public", UserId: 1, ChannelId: 60, Status: model.TaskStatusSuccess, PrivateData: model.TaskPrivateData{Key: "stored-task-key", UpstreamTaskID: "provider-id"}}).Error)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("id", 1); c.Next() })
	engine.GET("/v1/videos/:task_id/content", VideoProxy)
	engine.HEAD("/v1/videos/:task_id/content", VideoProxy)
	request := httptest.NewRequest(http.MethodGet, "/v1/videos/task_public/content", nil)
	request.Header.Set("Range", "bytes=0-2")
	request.Header.Set("If-Range", `"etag"`)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, request)
	require.Equal(t, http.StatusPartialContent, w.Code)
	require.Equal(t, "abc", w.Body.String())
	require.Equal(t, "bytes 0-2/6", w.Header().Get("Content-Range"))
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/v1/videos/task_public/content", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Body.String())
	require.Equal(t, "6", w.Header().Get("Content-Length"))
}
