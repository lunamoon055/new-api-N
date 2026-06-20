package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelListIncludesLinkskySora2(t *testing.T) {
	require.Contains(t, ModelList, "sora2")
	require.Contains(t, ModelList, "video-2.0")
	require.Contains(t, ModelList, "video-2.0-fast")
	require.Contains(t, ModelList, "ko3")
	require.Contains(t, ModelList, "veo31")
	require.Contains(t, ModelList, "veo31-fast")
	require.Contains(t, ModelList, "veo31-ref")
	require.Contains(t, ModelList, "grok-imagine-video")
}

func TestBuildRequestURLUsesAsyncGenerationsForLinkskyDocPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://linksky.top"}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/video/async-generations",
	}

	got, err := adaptor.BuildRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://linksky.top/v1/video/async-generations", got)
}

func TestFetchTaskUsesAsyncGenerationsForSora2(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"task_upstream","status":"completed"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"model":   "sora2",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
}

func TestFetchTaskUsesAsyncGenerationsForSoraDash2(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"task_upstream","status":"completed"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"model":   "sora-2",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
}

func TestFetchTaskUsesAsyncGenerationsForKlingV3(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"task_upstream","status":"running"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"model":   "kling-v3",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
}

func TestFetchTaskUsesAsyncGenerationsForLinkskyVideoModels(t *testing.T) {
	for _, modelName := range []string{
		"video-2.0",
		"video-2.0-fast",
		"ko3",
		"veo31",
		"veo31-fast",
		"veo31-ref",
		"grok-imagine-video",
	} {
		t.Run(modelName, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"task_upstream","status":"completed"}`))
			}))
			t.Cleanup(server.Close)

			adaptor := &TaskAdaptor{}
			resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
				"task_id": "task_upstream",
				"model":   modelName,
			}, "")

			require.NoError(t, err)
			require.NotNil(t, resp)
			_ = resp.Body.Close()
			require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
		})
	}
}

func TestParseTaskResultTreatsRunningAsInProgress(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"running",
		"progress":1
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusInProgress), result.Status)
	require.Equal(t, "1%", result.Progress)
}

func TestParseTaskResultCapturesCompletedMetadataURL(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"completed",
		"metadata":{"url":"https://cdn.example/video.mp4"},
		"progress":100
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "https://cdn.example/video.mp4", result.Url)
}

func TestDoResponseAcceptsNestedDataTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	adaptor := &TaskAdaptor{}
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{
			"code": 0,
			"message": "success",
			"data": {
				"task_id": "dt_task_upstream",
				"status": "queued",
				"progress": 10
			}
		}`)),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public",
		},
	}

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)

	require.Nil(t, taskErr)
	require.Equal(t, "dt_task_upstream", taskID)
	require.Contains(t, string(taskData), "dt_task_upstream")

	var body responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "task_public", body.ID)
	require.Equal(t, "task_public", body.TaskID)
}

func TestParseTaskResultAcceptsNestedDataVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"task_id": "dt_task_upstream",
			"status": "completed",
			"progress": 100,
			"data": [
				{"url": "https://cdn.example/dt-video.mp4"}
			]
		}
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "dt_task_upstream", result.TaskID)
	require.Equal(t, "https://cdn.example/dt-video.mp4", result.Url)
}

func TestParseTaskResultPrefersTopLevelVideoURLForJYShape(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"jy_task_upstream",
		"task_id":"jy_task_upstream",
		"status":"completed",
		"progress":100,
		"result_url":"https://cdn.example/jy-preview.png",
		"video_url":"https://cdn.example/jy-video.mp4",
		"data":[{"url":"https://cdn.example/dt-fallback.mp4"}]
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "jy_task_upstream", result.TaskID)
	require.Equal(t, "https://cdn.example/jy-video.mp4", result.Url)
}

func TestConvertToOpenAIVideoNormalizesCompletedAsyncTask(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  100,
		UpdatedAt:  180,
		Properties: model.Properties{OriginModelName: "sora2"},
		Data: []byte(`{
			"id":"task_upstream",
			"status":"completed",
			"metadata":{"url":"https://cdn.example/video.mp4"},
			"model":"sora2",
			"seconds":"4",
			"size":"1920x1080"
		}`),
	}

	body, err := adaptor.ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	var video dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(body, &video))
	require.Equal(t, "task_public", video.ID)
	require.Equal(t, "task_public", video.TaskID)
	require.Equal(t, dto.VideoStatusCompleted, video.Status)
	require.Equal(t, "sora2", video.Model)
	require.Equal(t, 100, video.Progress)
	require.Equal(t, "4", video.Seconds)
	require.Equal(t, "1920x1080", video.Size)
	require.Equal(t, "https://cdn.example/video.mp4", video.Metadata["url"])
}
