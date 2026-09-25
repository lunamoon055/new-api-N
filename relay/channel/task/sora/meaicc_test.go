package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func meaiccTestRequest(t *testing.T, baseURL, origin, upstream, path string, duration *int) (*TaskAdaptor, *relaycommon.RelayInfo, *gin.Context) {
	t.Helper()
	encoded, err := common.Marshal(videosRequest{
		Model: origin, Prompt: "a cat walking", Duration: duration,
		Ratio: "16:9", Resolution: "720p",
		StartImageURL:   "https://cdn.example/start.png",
		EndImageURL:     "https://cdn.example/end.png",
		ReferenceImages: []string{"https://cdn.example/ref.png"},
		ReferenceVideos: []string{"https://cdn.example/ref.mp4"},
		ReferenceAudios: []string{"https://cdn.example/ref.mp3"},
	})
	require.NoError(t, err)
	c := newVideo2JSONContext(t, string(encoded))
	c.Request.URL.Path = path
	info := &relaycommon.RelayInfo{
		OriginModelName: origin, RequestURLPath: path,
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: baseURL, UpstreamModelName: upstream,
		},
	}
	return &TaskAdaptor{baseURL: baseURL}, info, c
}

func TestMeaiccMappedSeedanceUsesNestedContract(t *testing.T) {
	for _, model := range []struct{ origin, upstream string }{
		{"Seedance2.0-c", "sd-2-c8"},
		{"Seedance2.0-fast-c", "sd-2-c4"},
		{"Seedance2.0-fast2-c", "sd-2-c5"},
		{"Seedance2.5-c", "sd-2.5-c1"},
		// Provider-specific selection precedes another provider's flat alias.
		{"seedance-2.5", "sd-2.5-c1"},
		{"sd-2.5-c1", "sd-2.5-c1"},
	} {
		for _, path := range []string{"/v1/videos", "/v1/video/async-generations", "/v1/video/generations"} {
			t.Run(model.origin+path, func(t *testing.T) {
				duration := 10
				a, info, c := meaiccTestRequest(t, "https://api.meaicc.com/v1/", model.origin, model.upstream, path, &duration)
				require.Nil(t, a.ValidateRequestAndSetAction(c, info))
				requestURL, err := a.BuildRequestURL(info)
				require.NoError(t, err)
				require.Equal(t, "https://api.meaicc.com/v1/videos", requestURL)
				require.Equal(t, "/v1/videos", info.UpstreamEndpoint)
				reader, err := a.BuildRequestBody(c, info)
				require.NoError(t, err)
				encoded, err := io.ReadAll(reader)
				require.NoError(t, err)
				var payload seedance2Request
				require.NoError(t, common.Unmarshal(encoded, &payload))
				require.Equal(t, model.upstream, payload.Model)
				require.Equal(t, "a cat walking", payload.Input.Prompt)
				require.Equal(t, seedance2Parameters{Resolution: "720p", Ratio: "16:9", Duration: &duration}, payload.Parameters)
				require.Equal(t, []seedance2Media{
					{Type: "first_frame", URL: "https://cdn.example/start.png"},
					{Type: "last_frame", URL: "https://cdn.example/end.png"},
					{Type: "reference_image", URL: "https://cdn.example/ref.png"},
					{Type: "reference_video", URL: "https://cdn.example/ref.mp4"},
					{Type: "reference_voice", URL: "https://cdn.example/ref.mp3"},
				}, payload.Input.Media)
				var fields map[string]any
				require.NoError(t, common.Unmarshal(encoded, &fields))
				require.NotContains(t, fields, "prompt")
				require.NotContains(t, fields, "referenceImages")
			})
		}
	}
}

func TestMeaiccContractDoesNotLeakToOtherChannelsOrModels(t *testing.T) {
	for _, tc := range []struct{ base, origin, upstream string }{
		{"https://other.example", "Seedance2.5-c", "sd-2.5-c1"},
		{"https://api.meaicc.com.other.example", "Seedance2.5-c", "sd-2.5-c1"},
		{"https://other.example/api.meaicc.com", "Seedance2.5-c", "sd-2.5-c1"},
		{"https://other.example", "seedance-2.5", "seedance-2.5"},
		{"https://api.meaicc.com", "Seedance2.5-c", "unrelated-model"},
	} {
		t.Run(tc.base+tc.upstream, func(t *testing.T) {
			duration := 10
			a, info, c := meaiccTestRequest(t, tc.base, tc.origin, tc.upstream, "/v1/video/async-generations", &duration)
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			requestURL, err := a.BuildRequestURL(info)
			require.NoError(t, err)
			if tc.upstream == "seedance-2.5" {
				require.Equal(t, tc.base+"/v1/videos", requestURL)
			} else {
				require.Equal(t, tc.base+"/v1/video/async-generations", requestURL)
			}
			reader, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			encoded, err := io.ReadAll(reader)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, common.Unmarshal(encoded, &fields))
			require.Equal(t, "a cat walking", fields["prompt"])
			require.NotContains(t, fields, "input")
			require.Contains(t, fields, "referenceImages")
		})
	}
}

func TestMeaiccDurationPreserved(t *testing.T) {
	zero, thirty := 0, 30
	for _, duration := range []*int{nil, &zero, &thirty} {
		a, info, c := meaiccTestRequest(t, "https://api.meaicc.com", "Seedance2.5-c", "sd-2.5-c1", "/v1/videos", duration)
		require.Nil(t, a.ValidateRequestAndSetAction(c, info))
		reader, err := a.BuildRequestBody(c, info)
		require.NoError(t, err)
		encoded, err := io.ReadAll(reader)
		require.NoError(t, err)
		var payload seedance2Request
		require.NoError(t, common.Unmarshal(encoded, &payload))
		require.Equal(t, duration, payload.Parameters.Duration)
	}
}

func TestMeaiccPollingUsesPersistedVideosEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/videos/task_test", r.URL.Path)
		_, _ = w.Write([]byte(`{"task_id":"task_test","status":"SUCCEEDED","object":"https://cdn.example/result.mp4"}`))
	}))
	defer server.Close()
	for _, upstream := range []string{"sd-2-c8", "sd-2-c4", "sd-2-c5", "sd-2.5-c1"} {
		t.Run(upstream, func(t *testing.T) {
			a, info, _ := meaiccTestRequest(t, "https://api.meaicc.com", "video-2.5", upstream, "/v1/video/async-generations", nil)
			_, err := a.BuildRequestURL(info)
			require.NoError(t, err)
			resp, err := a.FetchTask(server.URL, "test-key", map[string]any{
				"task_id": "task_test", "model": upstream, "origin_model": "video-2.5",
				"upstream_endpoint": info.UpstreamEndpoint,
			}, "")
			require.NoError(t, err)
			defer resp.Body.Close()
			encoded, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			result, err := a.ParseTaskResult(encoded)
			require.NoError(t, err)
			require.Equal(t, "SUCCESS", result.Status)
			require.Equal(t, "https://cdn.example/result.mp4", result.Url)
		})
	}
}
