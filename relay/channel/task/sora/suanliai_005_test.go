package sora

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestSuanliai005PublicContractToProviderWireFormat(t *testing.T) {
	for _, path := range []string{"/v1/videos", "/v1/video/async-generations", "/v1/video/generations"} {
		t.Run(path, func(t *testing.T) {
			c := newVideo2JSONContext(t, `{"model":"Seedance2.5:720-S（条）","prompt":"demo","duration":30,"ratio":"21:9","resolution":"720p","referenceImages":["https://cdn.example/i.png"],"referenceVideos":["https://cdn.example/v.mp4"],"referenceAudios":["https://cdn.example/a.mp3"],"metadata":{"seed":0,"generate_audio":false}}`)
			info := &relaycommon.RelayInfo{OriginModelName: "Seedance2.5:720-S（条）", RequestURLPath: path,
				TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: suanliai005Model}}
			a := &TaskAdaptor{baseURL: "https://suanliai.top/v1/"}
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			endpoint, err := a.BuildRequestURL(info)
			require.NoError(t, err)
			require.Equal(t, "https://suanliai.top/v1/videos", endpoint)
			reader, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			body, err := io.ReadAll(reader)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, common.Unmarshal(body, &got))
			require.Equal(t, suanliai005Model, got["model"])
			require.Equal(t, float64(30), got["seconds"])
			require.Equal(t, "1680x720", got["size"])
			for key, url := range map[string]string{"images": "https://cdn.example/i.png", "videos": "https://cdn.example/v.mp4", "audios": "https://cdn.example/a.mp3"} {
				require.Equal(t, []any{url}, got[key])
			}
			require.Equal(t, map[string]any{"seed": float64(0), "generate_audio": false}, got["metadata"])
			for _, key := range []string{"duration", "referenceImages", "referenceVideos", "referenceAudios", "ratio", "resolution"} {
				require.NotContains(t, got, key)
			}
			req, err := relaycommon.GetTaskRequest(c)
			require.NoError(t, err)
			require.Equal(t, info.OriginModelName, req.Model)
			require.Equal(t, 30, req.Duration)
			require.Equal(t, "21:9", req.Ratio)
		})
	}
}

func TestSuanliai005LimitsAliasesAndIsolation(t *testing.T) {
	payload := map[string]any{"model": "alias", "prompt": "demo", "seconds": "30", "size": "720x1280"}
	for field, count := range map[string]int{"images": 30, "videos": 10, "audios": 10} {
		urls := make([]string, count)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://cdn.example/%s/%d", field, i)
		}
		payload[field] = urls
	}
	normalized, err := normalizeSuanliai005Payload(payload)
	require.NoError(t, err)
	require.Equal(t, "9:16", normalized["ratio"])
	for _, overrides := range []map[string]any{
		{"seconds": 4}, {"seconds": 31}, {"duration": 5}, {"resolution": "1080p"},
		{"ratio": "16:9"}, {"size": "1920x1080"}, {"ratio": 16},
		{"images": append(append([]string{}, payload["images"].([]string)...), "https://cdn.example/extra.png")},
		{"videos": append(append([]string{}, payload["videos"].([]string)...), "https://cdn.example/extra.mp4")},
		{"audios": append(append([]string{}, payload["audios"].([]string)...), "https://cdn.example/extra.mp3")},
		{"start_image_url": "https://cdn.example/start.png"},
	} {
		copy := make(map[string]any)
		for key, value := range payload {
			copy[key] = value
		}
		for key, value := range overrides {
			copy[key] = value
		}
		_, err := normalizeSuanliai005Payload(copy)
		require.Error(t, err, overrides)
	}
	for _, name := range []string{"005系列/sd2.5-other", "004系列/sd2.5(30-10-10支持过人脸4-30秒720P)"} {
		require.False(t, isSuanliai005Request(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: name}}))
	}
}

func TestSuanliai005TaskIDAndPolling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/videos/provider_005", r.URL.Path)
		require.Equal(t, "Bearer test-only", r.Header.Get("Authorization"))
		fmt.Fprint(w, `{"task_id":"provider_005","status":"completed","url":"https://cdn.example/result.mp4"}`)
	}))
	defer server.Close()
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: suanliai005Model}}
	c := newVideo2JSONContext(t, `{}`)
	id, _, taskErr := a.DoResponse(c, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"task_id":"provider_005","status":"queued","progress":0}`))}, info)
	require.Nil(t, taskErr)
	require.Equal(t, "provider_005", id)
	response, err := a.FetchTask(server.URL+"/v1", "test-only", map[string]any{"task_id": id, "upstream_endpoint": "/v1/videos", "model": suanliai005Model}, "")
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	result, err := a.ParseTaskResult(data)
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example/result.mp4", result.Url)
}
