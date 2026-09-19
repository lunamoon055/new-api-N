package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestSuanliaiModelsUseVideosEndpointAndTrimConfiguredV1(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://suanliai.top/v1"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "004系列/minimax-h3 768p",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "004系列/minimax-h3 768p",
		},
	}

	got, err := adaptor.BuildRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://suanliai.top/v1/videos", got)
	require.Equal(t, "/v1/videos", info.UpstreamEndpoint)
}

func TestSuanliai004AcceptsNumericSecondsAndDocumentedReferences(t *testing.T) {
	modelName := "004系列/minimax_h3(8图3音频)"
	c := newVideo2JSONContext(t, `{
		"model":"004系列/minimax_h3(8图3音频)",
		"prompt":"Image 1 按 Audio 1 的语气说话",
		"seconds":10,
		"ratio":"16:9",
		"resolution":"768p",
		"images":["https://cdn.example/person.jpg"],
		"audios":["https://cdn.example/voice.mp3"]
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: modelName,
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
	require.Equal(t, modelName, got["model"])
	require.Equal(t, float64(10), got["seconds"])
	require.Equal(t, []any{"https://cdn.example/person.jpg"}, got["images"])
	require.Equal(t, []any{"https://cdn.example/voice.mp3"}, got["audios"])
}

func TestSuanliaiOmniValidationUsesDocumentedDurationAndAspectRatio(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"omni-video-1-pro",
		"prompt":"镜头推进到灯塔",
		"duration":8,
		"aspect_ratio":"16:9",
		"seed":42,
		"image_url":"https://cdn.example/start.png"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "omni-video-1-pro",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "omni-video-1-pro",
		},
	}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info))
}

func TestSuanliaiTaskResultExtractsNestedVideoURL(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"vid_01",
		"status":"completed",
		"progress":100,
		"video":{"url":"https://suanliai.top/v1/videos/vid_01/content","mime_type":"video/mp4"}
	}`))

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), result.Status)
	require.Equal(t, "https://suanliai.top/v1/videos/vid_01/content", result.Url)
}

func TestBuildRequestURLSupportsOfficialVideoGenerationsCompatibilityPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://suanliai.top/v1"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "官转稳定-sd2-720p(933满血不卡脸)",
		RequestURLPath:  "/v1/video/generations",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "官转稳定-sd2-720p(933满血不卡脸)"},
	}

	got, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://suanliai.top/v1/video/generations", got)
	require.Equal(t, "/v1/video/generations", info.UpstreamEndpoint)
}

func TestSuanliaiDocumentedModelsUseTheirDocumentedEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		path     string
		expected string
	}{
		{name: "特价 wan video", model: "wan3.0-video", path: "/v1/videos", expected: "/v1/videos"},
		{name: "特价 sd mini", model: "sd-mini", path: "/v1/videos", expected: "/v1/videos"},
		{name: "官转", model: "官转稳定-sd2-720p(933满血不卡脸)", path: "/v1/videos", expected: "/v1/video/generations"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adaptor := &TaskAdaptor{baseURL: "https://suanliai.top"}
			info := &relaycommon.RelayInfo{
				OriginModelName: test.model,
				RequestURLPath:  test.path,
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: test.model},
			}
			got, err := adaptor.BuildRequestURL(info)
			require.NoError(t, err)
			require.Equal(t, "https://suanliai.top"+test.expected, got)
			require.Equal(t, test.expected, info.UpstreamEndpoint)
		})
	}
}

func TestTaskSubmitReqKeepsNewReferenceShapesAndExplicitFalse(t *testing.T) {
	var req relaycommon.TaskSubmitReq
	err := common.Unmarshal([]byte(`{
		"model":"wan3.0-video",
		"prompt":"demo",
		"reference_images":[{"url":"https://cdn.example/image.jpg","role":"first_frame"}],
		"reference_videos":[{"url":"https://cdn.example/video.mp4"}],
		"reference_audios":[{"url":"https://cdn.example/audio.mp3"}],
		"generate_audio":false,
		"n":0,
		"start_frame":"https://cdn.example/start.jpg"
	}`), &req)
	require.NoError(t, err)
	require.Len(t, req.ReferenceImageObjects, 1)
	require.Equal(t, "https://cdn.example/image.jpg", req.ReferenceImageObjects[0].URL)
	require.Equal(t, "first_frame", req.ReferenceImageObjects[0].Role)
	require.NotNil(t, req.GenerateAudio)
	require.False(t, *req.GenerateAudio)
	require.NotNil(t, req.N)
	require.Equal(t, 0, *req.N)
	images, videos, audios := req.InputMaterialURLs()
	require.Contains(t, images, "https://cdn.example/image.jpg")
	require.Contains(t, images, "https://cdn.example/start.jpg")
	require.Contains(t, videos, "https://cdn.example/video.mp4")
	require.Contains(t, audios, "https://cdn.example/audio.mp3")
}

func TestTaskSubmitReqAcceptsDocumentedOfficialAliases(t *testing.T) {
	var req relaycommon.TaskSubmitReq
	require.NoError(t, common.Unmarshal([]byte(`{
		"model_id":"官转稳定-sd2-720p(933满血不卡脸)",
		"text":"alias prompt",
		"seconds":"10",
		"aspect_ratio":"16:9"
	}`), &req))
	require.Equal(t, "官转稳定-sd2-720p(933满血不卡脸)", req.Model)
	require.Equal(t, "alias prompt", req.Prompt)
	require.Equal(t, "10", req.Seconds)
	require.Equal(t, "16:9", req.Ratio)
}

func TestFetchTaskUsesPersistedVideoGenerationsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/video/generations/task_123", r.URL.Path)
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "sk-test", map[string]any{
		"task_id":           "task_123",
		"upstream_endpoint": "/v1/video/generations",
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
