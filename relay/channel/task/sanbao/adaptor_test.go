package sanbao

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBodyConvertsCreationVideoPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/async-generations", bytes.NewBufferString(`{
		"model":"site-video",
		"prompt":"让 @参考图片1 里的角色挥手",
		"aspect_ratio":"9:16",
		"resolution":"720p",
		"duration":5,
		"image_url":"https://cdn.example/ref.png",
		"video_reference":[{"url":"https://cdn.example/ref.mp4"}],
		"audio_url":"https://cdn.example/ref.mp3"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{}
	reader, err := adaptor.BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sanbao-video",
		},
	})

	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	require.Equal(t, "sanbao-video", payload["model"])
	require.Equal(t, "让 @参考图片1 里的角色挥手", payload["prompt"])
	require.Equal(t, "9:16", payload["ratio"])
	require.Equal(t, "720p", payload["resolution"])
	require.EqualValues(t, 5, payload["duration"])
	require.Equal(t, "all", payload["reference"])

	images, ok := payload["images"].([]any)
	require.True(t, ok)
	require.Len(t, images, 1)
	image := images[0].(map[string]any)
	require.Equal(t, "参考图片1", image["tag"])
	require.Equal(t, "https://cdn.example/ref.png", image["url"])

	videos, ok := payload["videos"].([]any)
	require.True(t, ok)
	require.Len(t, videos, 1)
	video := videos[0].(map[string]any)
	require.Equal(t, "参考视频1", video["tag"])
	require.Equal(t, "https://cdn.example/ref.mp4", video["url"])

	audios, ok := payload["audios"].([]any)
	require.True(t, ok)
	require.Len(t, audios, 1)
	audio := audios[0].(map[string]any)
	require.Equal(t, "参考音频", audio["tag"])
	require.Equal(t, "https://cdn.example/ref.mp3", audio["url"])
}

func TestDoResponseReturnsPublicTaskIDAndStoresUpstreamID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	adaptor := &TaskAdaptor{}
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{
			"data":{
				"id":"task_upstream",
				"status":"queued",
				"progress":10
			}
		}`)),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public",
		},
		OriginModelName: "site-video",
	}

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)

	require.Nil(t, taskErr)
	require.Equal(t, "task_upstream", taskID)
	require.Contains(t, string(taskData), "task_upstream")

	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "task_public", response["id"])
	require.Equal(t, "task_public", response["task_id"])
	require.Equal(t, "queued", response["status"])
}

func TestValidateImageRequestSetsImageAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/pg/images/generations", bytes.NewBufferString(`{
		"model":"site-image",
		"prompt":"生成一张海报"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		RequestURLPath: "/pg/images/generations",
		ChannelMeta:    &relaycommon.ChannelMeta{},
		TaskRelayInfo:  &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)

	require.Nil(t, taskErr)
	require.Equal(t, "imageGenerate", info.Action)
}

func TestBuildRequestBodyUploadsReferenceAssetToSanbao(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var uploadPath string
	var uploadKind string
	var uploadFileName string
	var uploadContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/asset/ref.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-data"))
		case "/openapi/v1/uploads/raw":
			uploadPath = r.URL.Path
			uploadKind = r.URL.Query().Get("kind")
			uploadFileName = r.URL.Query().Get("fileName")
			uploadContentType = r.Header.Get("Content-Type")
			require.Equal(t, "Bearer sk_sanbao_test", r.Header.Get("Authorization"))
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, []byte("png-data"), body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"file":{"url":"https://sanbao.example/uploaded/ref.png"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/async-generations", bytes.NewBufferString(fmt.Sprintf(`{
		"model":"site-video",
		"prompt":"参考 @参考图片1",
		"image_url":%q
	}`, server.URL+"/asset/ref.png")))
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &TaskAdaptor{
		apiKey:  "sk_sanbao_test",
		baseURL: server.URL,
	}
	reader, err := adaptor.BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sanbao-video",
		},
	})

	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.Equal(t, "/openapi/v1/uploads/raw", uploadPath)
	require.Equal(t, "image", uploadKind)
	require.Equal(t, "ref.png", uploadFileName)
	require.Equal(t, "image/png", uploadContentType)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	images := payload["images"].([]any)
	image := images[0].(map[string]any)
	require.Equal(t, "https://sanbao.example/uploaded/ref.png", image["url"])
}

func TestFetchTaskUsesSanbaoVideoEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"task_upstream","status":"processing"}}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"action":  "generate",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/openapi/v1/videos/task_upstream", gotPath)
}

func TestFetchTaskTrimsOpenAPIBasePath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"task_upstream","status":"processing"}}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL+"/openapi/v1", "sk-test", map[string]any{
		"task_id": "task_upstream",
		"action":  "generate",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/openapi/v1/videos/task_upstream", gotPath)
}

func TestNormalizeSanbaoBaseURLTrimsKnownAPIPaths(t *testing.T) {
	for _, baseURL := range []string{
		"https://sanbaobeauty.com/openapi/v1/",
		"https://sanbaobeauty.com/openapi",
		"https://sanbaobeauty.com/v1",
	} {
		require.Equal(t, "https://sanbaobeauty.com", normalizeSanbaoBaseURL(baseURL))
	}
}

func TestInitTrimsOpenAPIBasePath(t *testing.T) {
	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://sanbaobeauty.com/openapi/v1/",
		},
	})

	uri, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		RequestURLPath: "/pg/video/async-generations",
	})

	require.NoError(t, err)
	require.Equal(t, "https://sanbaobeauty.com/openapi/v1/videos", uri)
}

func TestFetchTaskUsesSanbaoImageEndpointForImageAction(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"task_upstream","status":"processing"}}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"action":  "imageGenerate",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/openapi/v1/images/task_upstream", gotPath)
}

func TestParseTaskResultMapsSucceededVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"data":{
			"id":"task_upstream",
			"status":"succeeded",
			"progress":100,
			"video_url":"https://cdn.example/preview.mp4",
			"download_url":"https://cdn.example/download.mp4"
		}
	}`))

	require.NoError(t, err)
	require.Equal(t, "task_upstream", result.TaskID)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "100%", result.Progress)
	require.Equal(t, "https://cdn.example/download.mp4", result.Url)
}

func TestParseTaskResultMapsFailureReason(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"data":{
			"id":"task_upstream",
			"status":"failed",
			"progress":100,
			"error":"素材无法访问"
		}
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusFailure), result.Status)
	require.Equal(t, "100%", result.Progress)
	require.Equal(t, "素材无法访问", result.Reason)
}
