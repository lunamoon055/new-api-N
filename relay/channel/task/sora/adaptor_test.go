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

func TestModelListIncludesSupportedVideoModels(t *testing.T) {
	require.Contains(t, ModelList, "sora2")
	require.Contains(t, ModelList, "minimax-h3")
	require.Contains(t, ModelList, "video-2.0")
	require.Contains(t, ModelList, "video-2.0-fast")
	require.Contains(t, ModelList, "video-2.0-mini")
	require.Contains(t, ModelList, "video-2.0-480p")
	require.Contains(t, ModelList, "video-2.0-fast-480p")
	require.Contains(t, ModelList, "video-2.0-mini-480p")
	require.Contains(t, ModelList, "ko3")
	require.Contains(t, ModelList, "veo31")
	require.Contains(t, ModelList, "veo31-fast")
	require.Contains(t, ModelList, "veo31-ref")
	require.Contains(t, ModelList, "grok-imagine-video")
	require.Contains(t, ModelList, "seedance-2.5")
}

func TestMiniMaxH3RequestBodyPassesDocumentedFieldsThrough(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"minimax-h3",
		"prompt":"demo",
		"duration":8,
		"aspect_ratio":"21:9",
		"image_urls":["https://cdn.example/one.png","https://cdn.example/two.png"],
		"audio_url":"https://cdn.example/audio.ogg"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "minimax-h3",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "minimax-h3",
		},
	}
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))

	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(encoded, &got))
	require.Equal(t, "minimax-h3", got["model"])
	require.Equal(t, "demo", got["prompt"])
	require.Equal(t, float64(8), got["duration"])
	require.Equal(t, "21:9", got["aspect_ratio"])
	require.Equal(t, []any{"https://cdn.example/one.png", "https://cdn.example/two.png"}, got["image_urls"])
	require.Equal(t, "https://cdn.example/audio.ogg", got["audio_url"])
	require.NotContains(t, got, "resolution")
	require.NotContains(t, got, "async")
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

func TestBuildRequestURLKeepsVideosApiModelOnStandardEndpoint(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://api.example.com"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "sd2-mini",
		RequestURLPath:  "/v1/video/async-generations",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "videos-mini",
		},
	}

	got, err := adaptor.BuildRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1/videos", got)
}

func TestBuildRequestURLKeepsVideos4CatalogModelsOnStandardEndpoint(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://api.example.com"}
	for _, modelName := range []string{
		"videos-4 (4图3视频1音频)",
		"videos-4-fast (4图3视频1音频)",
		"videos-4-mini (4图3视频1音频)",
	} {
		t.Run(modelName, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: modelName,
				RequestURLPath:  "/v1/video/async-generations",
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: modelName,
				},
			}

			got, err := adaptor.BuildRequestURL(info)

			require.NoError(t, err)
			require.Equal(t, "https://api.example.com/v1/videos", got)
		})
	}
}

func TestBuildRequestURLUsesVideosAPIForSeedanceAlias(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://api.example.com"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "(线路3)sd-2.0-933",
		RequestURLPath:  "/v1/video/async-generations",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sd-2-c8",
		},
	}

	got, err := adaptor.BuildRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1/videos", got)
}

func TestSeedance25UsesDocumentedFlatRequestBody(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://api.example.com"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "Seedance-2.5",
		RequestURLPath:  "/v1/video/async-generations",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance-2.5",
		},
	}

	gotURL, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1/videos", gotURL)

	c := newVideo2JSONContext(t, `{
		"model":"Seedance-2.5",
		"prompt":"让角色从起始画面自然走向结束画面",
		"duration":29,
		"ratio":"16:9",
		"resolution":"480p",
		"start_image_url":"https://cdn.example/start.png",
		"end_image_url":"https://cdn.example/end.png",
		"referenceImages":["https://cdn.example/reference.png"],
		"referenceVideos":["https://cdn.example/reference.mp4"],
		"referenceAudios":["https://cdn.example/reference.mp3"]
	}`)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var got seedance25Request
	require.NoError(t, common.Unmarshal(encoded, &got))
	require.Equal(t, "seedance-2.5", got.Model)
	require.Equal(t, "让角色从起始画面自然走向结束画面", got.Prompt)
	require.Equal(t, "480p", got.Resolution)
	require.Equal(t, "16:9", got.Ratio)
	require.NotNil(t, got.Duration)
	require.Equal(t, 29, *got.Duration)
	require.Equal(t, []string{
		"https://cdn.example/start.png",
		"https://cdn.example/reference.png",
		"https://cdn.example/end.png",
	}, got.ReferenceImages)
	require.Equal(t, []string{"https://cdn.example/reference.mp4"}, got.ReferenceVideos)
	require.Equal(t, []string{"https://cdn.example/reference.mp3"}, got.ReferenceAudios)

	var raw map[string]any
	require.NoError(t, common.Unmarshal(encoded, &raw))
	require.Contains(t, raw, "prompt")
	require.NotContains(t, raw, "content")
	require.NotContains(t, raw, "generate_audio")
	require.NotContains(t, raw, "watermark")
	require.NotContains(t, raw, "start_image_url")
	require.NotContains(t, raw, "end_image_url")
}

func TestSeedance25UsesExpandedLimitsWithoutChangingSeedance20(t *testing.T) {
	images := make([]string, 30)
	for index := range images {
		images[index] = "https://cdn.example/reference.png"
	}
	videos := make([]string, 10)
	for index := range videos {
		videos[index] = "https://cdn.example/reference.mp4"
	}
	audios := make([]string, 10)
	for index := range audios {
		audios[index] = "https://cdn.example/reference.mp3"
	}
	duration := 29

	require.NoError(t, validateSeedance25Request(videosRequest{
		Model:           "Seedance-2.5",
		Prompt:          "测试扩展素材上限",
		Duration:        &duration,
		ReferenceImages: images,
		ReferenceVideos: videos,
		ReferenceAudios: audios,
	}))

	tooManyImages := append(append([]string{}, images...), "https://cdn.example/extra.png")
	require.EqualError(t, validateSeedance25Request(videosRequest{
		Prompt:          "测试图片上限",
		ReferenceImages: tooManyImages,
	}), "image references must not exceed 30")

	duration = 30
	require.EqualError(t, validateSeedance25Request(videosRequest{
		Prompt:   "测试时长上限",
		Duration: &duration,
	}), "duration must be between 4 and 29")

	require.EqualError(t, validateSeedance25Request(videosRequest{
		Prompt:     "测试分辨率选项",
		Resolution: "1080p",
	}), "resolution must be 720p or 480p")

	require.EqualError(t, validateSeedance25Request(videosRequest{
		Prompt: "测试比例选项",
		Ratio:  "4:3",
	}), "ratio must be 16:9, 9:16, or 1:1")

	require.Error(t, validateSeedance2Request(videosRequest{
		Prompt:          "Seedance 2.0 仍使用原上限",
		ReferenceImages: images,
	}))
}

func TestFetchTaskUsesVideosEndpointForSeedance25(t *testing.T) {
	var gotPath string
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"task_id":"seedance_upstream","status":"queued"}`))
	}))
	t.Cleanup(server.Close)

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "sk-test", map[string]any{
		"task_id":      "seedance_upstream",
		"model":        "seedance-2.5",
		"origin_model": "Seedance-2.5",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/videos/seedance_upstream", gotPath)
	require.Equal(t, "Bearer sk-test", gotAuthorization)
}

func TestDoResponseAcceptsDocumentedVideosShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{
			"id":"seedance_upstream",
			"task_id":"seedance_upstream",
			"object":"video",
			"model":"seedance-2.5",
			"status":"queued",
			"progress":0,
			"created_at":1782690295,
			"completed_at":null,
			"seconds":"5",
			"url":null,
			"video_url":null,
			"metadata":{},
			"error":null
		}`)),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public",
		},
	}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)

	require.Nil(t, taskErr)
	require.Equal(t, "seedance_upstream", taskID)
	require.Contains(t, string(taskData), `"task_id":"seedance_upstream"`)
	var body responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "task_public", body.ID)
	require.Equal(t, "task_public", body.TaskID)
	require.Equal(t, "queued", body.Status)
	require.Equal(t, "seedance-2.5", body.Model)
	require.Equal(t, "5", string(body.Seconds))
}

func TestSeedanceRequestUsesDocumentedNestedBody(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"(线路3)sd-2.0-933",
		"prompt":"让角色向前走",
		"duration":10,
		"ratio":"4:3",
		"resolution":"720p",
		"start_image_url":"https://cdn.example/start.png",
		"end_image_url":"https://cdn.example/end.png",
		"referenceImages":["https://cdn.example/ref.png"],
		"referenceVideos":["https://cdn.example/ref.mp4"],
		"referenceAudios":["https://cdn.example/ref.mp3"]
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "(线路3)sd-2.0-933",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sd-2-c8",
		},
	}
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var got seedance2Request
	require.NoError(t, common.Unmarshal(encoded, &got))
	require.Equal(t, "sd-2-c8", got.Model)
	require.Equal(t, "让角色向前走", got.Input.Prompt)
	require.Equal(t, "720p", got.Parameters.Resolution)
	require.Equal(t, "4:3", got.Parameters.Ratio)
	require.NotNil(t, got.Parameters.Duration)
	require.Equal(t, 10, *got.Parameters.Duration)
	require.Equal(t, []seedance2Media{
		{Type: "first_frame", URL: "https://cdn.example/start.png"},
		{Type: "last_frame", URL: "https://cdn.example/end.png"},
		{Type: "reference_image", URL: "https://cdn.example/ref.png"},
		{Type: "reference_video", URL: "https://cdn.example/ref.mp4"},
		{Type: "reference_voice", URL: "https://cdn.example/ref.mp3"},
	}, got.Input.Media)
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
		"video-2.0-mini",
		"video-2.0-480p",
		"video-2.0-fast-480p",
		"video-2.0-mini-480p",
		"minimax-h3",
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

func TestFetchTaskUsesOriginModelWhenUpstreamModelIsMapped(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"task_upstream","status":"running"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id":      "task_upstream",
		"model":        "provider-specific-h3",
		"origin_model": "minimax-h3",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
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

func TestParseTaskResultAcceptsNumericSeconds(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"completed",
		"seconds":10,
		"object":"https://cdn.example/video.mp4"
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "https://cdn.example/video.mp4", result.Url)
}

func TestParseTaskResultDoesNotTreatObjectNameAsVideoURL(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"completed",
		"object":"video"
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Empty(t, result.Url)
}

func TestParseTaskResultAcceptsObjectURLAndFailedReason(t *testing.T) {
	completed, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"completed",
		"object":"https://cdn.example/video.mp4"
	}`))
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example/video.mp4", completed.Url)

	failed, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_upstream",
		"status":"FAILED: quota exhausted"
	}`))
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusFailure), failed.Status)
	require.Equal(t, "quota exhausted", failed.Reason)
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

func TestDoResponseExposesFailedStatusReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{
			"id":"task_upstream",
			"status":"FAILED: quota exhausted"
		}`)),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public",
		},
	}

	taskID, _, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)

	require.Nil(t, taskErr)
	require.Equal(t, "task_upstream", taskID)
	var body responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "FAILED: quota exhausted", body.Status)
	require.NotNil(t, body.Error)
	require.Equal(t, "quota exhausted", body.Error.Message)
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
			"seconds":4,
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
