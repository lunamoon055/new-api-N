package sora

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

func TestXJZUnknownStatusKeepsAcceptedVideoTaskPending(t *testing.T) {
	const upstreamResponse = `{"id":"task_upstream","task_id":"task_upstream","object":"video","model":"特价sd2.0-720p-933-过脸","status":"unknown","progress":0,"created_at":1791630420}`
	parsed, err := (&TaskAdaptor{}).ParseTaskResult([]byte(upstreamResponse))
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusQueued), parsed.Status)
	require.Equal(t, "task_upstream", parsed.TaskID)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "特价sd2.0-720p-933-过脸"},
	}
	upstreamID, raw, taskErr := (&TaskAdaptor{}).DoResponse(c, &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(upstreamResponse)),
	}, info)
	require.Nil(t, taskErr)
	require.Equal(t, "task_upstream", upstreamID)
	require.Equal(t, upstreamResponse, string(raw))
	var publicResponse responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &publicResponse))
	require.Equal(t, "task_public", publicResponse.ID)
	require.Equal(t, "queued", publicResponse.Status)
}

func TestXJZVideoModelsUseVideosEndpointAndDocumentedDurations(t *testing.T) {
	models := []struct {
		name       string
		duration   int
		resolution string
		reference  string
	}{
		{"特价dola2.5-900-720P-固定30s", 30, "720p", `,"images":["https://cdn.example/one.png"]`},
		{"特价dola-sd2.0-900-720p", 5, "720p", `,"images":["https://cdn.example/one.png"]`},
		{"TTP-grok", 6, "720p", ""},
		{"gz稳定sd2.0-903-720P", 5, "720p", ""},
		{"Xminimex-h3-2K-933", 5, "2k", ""},
		{"特价sd2.0-720p-933-过脸", 5, "720p", ""},
		{"特价稳sd-2.5-30s-720P可10图", 30, "720p", ""},
		{"特价稳sd2.0-933-720p", 5, "720P", ""},
		{"gz稳定sd2.5-720全参过脸", 4, "720P", ""},
		{"gz稳定sd2.5-480全参过脸", 4, "480P", ""},
		{"gz稳定sd2.5-1080全参过脸", 4, "1080p", ""},
		{"特价sd2.0-933-过脸", 4, "720p", ""},
		{"MiniMaxH3", 4, "2K", ""},
	}
	require.Len(t, xjzVideoSpecs, 48)
	for _, tc := range models {
		t.Run(tc.name, func(t *testing.T) {
			reference := tc.reference
			if tc.name == "TTP-grok" {
				reference += `,"resolution":"720p"`
			}
			c := newVideo2JSONContext(t, fmt.Sprintf(`{"model":%q,"prompt":"a product video","duration":%d,"aspect_ratio":"16:9"%s}`, tc.name, tc.duration, reference))
			info := &relaycommon.RelayInfo{
				OriginModelName: tc.name,
				RequestURLPath:  "/v1/video/async-generations",
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelBaseUrl:    "https://api.xjzapi.cc/v1",
					UpstreamModelName: tc.name,
				},
			}
			adaptor := &TaskAdaptor{baseURL: "https://api.xjzapi.cc/v1"}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
			if tc.reference != "" {
				stored, err := relaycommon.GetTaskRequest(c)
				require.NoError(t, err)
				images, _, _ := stored.InputMaterialURLs()
				require.Equal(t, []string{"https://cdn.example/one.png"}, images)
			}
			requestURL, err := adaptor.BuildRequestURL(info)
			require.NoError(t, err)
			require.Equal(t, "https://api.xjzapi.cc/v1/videos", requestURL)
			require.Equal(t, "/v1/videos", info.UpstreamEndpoint)
			require.Equal(t, float64(tc.duration), adaptor.EstimateBilling(c, info)["seconds"])

			body, err := adaptor.BuildRequestBody(c, info)
			require.NoError(t, err)
			encoded, err := io.ReadAll(body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.Unmarshal(encoded, &payload))
			require.Equal(t, tc.name, payload["model"])
			require.Equal(t, float64(tc.duration), payload["duration"])
			require.Equal(t, tc.resolution, payload["resolution"])
			require.Equal(t, "16:9", payload["ratio"])
			require.NotContains(t, payload, "aspect_ratio")
			require.NotContains(t, payload, "seconds")
		})
	}
}

func TestXJZVideoRejectsConflictingOrUnsupportedInputs(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		payload string
		want    string
	}{
		{"fixed 30 seconds", "特价dola2.5-900-720P-固定30s", `{"model":"x","prompt":"demo","duration":6,"images":["https://cdn.example/one.png"]}`, "duration"},
		{"discrete durations", "特价dola-sd2.0-900-720p", `{"model":"x","prompt":"demo","duration":6,"images":["https://cdn.example/one.png"]}`, "duration"},
		{"required image", "特价dola2.5-900-720P-固定30s", `{"model":"x","prompt":"demo","duration":30}`, "image references"},
		{"resolution", "gz稳定sd2.5-480全参过脸", `{"model":"x","prompt":"demo","duration":10,"resolution":"720p"}`, "resolution"},
		{"ratio conflict", "特价sd2.0-933-过脸", `{"model":"x","prompt":"demo","duration":10,"ratio":"16:9","aspect_ratio":"9:16"}`, "conflicts"},
		{"too many references", "特价dola-sd2.0-900-720p", `{"model":"x","prompt":"demo","duration":5,"images":["https://cdn.example/one.png","https://cdn.example/two.png"]}`, "image references"},
		{"audio needs visual", "MiniMaxH3", `{"model":"x","prompt":"demo","duration":5,"audio_reference":["https://cdn.example/one.mp3"]}`, "audio references require"},
		{"unsupported video", "MiniMaxH3", `{"model":"x","prompt":"demo","duration":5,"video_urls":["https://cdn.example/one.mp4"]}`, "video references are unsupported"},
		{"unsupported frame", "TTP-grok", `{"model":"x","prompt":"demo","duration":6,"resolution":"720p","start_image_url":"https://cdn.example/one.png"}`, "start/end frames"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var payload map[string]any
			require.NoError(t, common.Unmarshal([]byte(tc.payload), &payload))
			_, _, _, _, _, err := normalizeXJZVideoPayload(payload, tc.model)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestXJZVideoNormalizesReferencesAndFixedDuration(t *testing.T) {
	payload := map[string]any{
		"model": "MiniMaxH3", "prompt": "demo", "duration": float64(5),
		"image_urls":      []any{"https://cdn.example/one.png"},
		"audio_reference": []any{"https://cdn.example/one.mp3"},
		"generate_audio":  false,
	}
	normalized, duration, images, videos, audios, err := normalizeXJZVideoPayload(payload, "MiniMaxH3")
	require.NoError(t, err)
	require.Equal(t, 5, duration)
	require.Equal(t, 1, images)
	require.Zero(t, videos)
	require.Equal(t, 1, audios)
	require.Equal(t, "https://cdn.example/one.png", normalized["image_url"])
	require.Equal(t, "https://cdn.example/one.mp3", normalized["audio_url"])
	require.Equal(t, false, normalized["generate_audio"])

	fixed := map[string]any{"model": "特价稳sd-2.5-30s-720P可10图", "prompt": "demo"}
	normalized, duration, _, _, _, err = normalizeXJZVideoPayload(fixed, "特价稳sd-2.5-30s-720P可10图")
	require.NoError(t, err)
	require.Equal(t, 30, duration)
	require.Equal(t, 30, normalized["duration"])
}

func TestXJZVideoFetchUsesStandardVideosPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/videos/upstream-task", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"id":"upstream-task","status":"completed","video":{"url":"https://cdn.example/video.mp4"}}`))
	}))
	defer server.Close()
	resp, err := (&TaskAdaptor{}).FetchTask(server.URL+"/v1", "test-key", map[string]any{
		"task_id": "upstream-task", "model": "MiniMaxH3", "upstream_endpoint": "/v1/video/async-generations",
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resultBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	result, err := (&TaskAdaptor{}).ParseTaskResult(resultBody)
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example/video.mp4", result.Url)
}
