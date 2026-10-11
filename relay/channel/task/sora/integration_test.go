package sora

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestMappedVideoContractsValidateAndBuildUpstreamRequest(t *testing.T) {
	cases := []struct {
		public, upstream, base, resolution string
		duration                           int
		image                              bool
	}{
		{"Seedance2.0-c", "sd-2-c8", "https://api.meaicc.com", "720p", 15, false},
		{"Seedance2.0-fast-c", "sd-2-c4", "https://api.meaicc.com", "720p", 4, false},
		{"Seedance2.0-fast2-c", "sd-2-c5", "https://api.meaicc.com", "720p", 4, false},
		{"Seedance2.5-c", "sd-2.5-c1", "https://api.meaicc.com", "480p", 29, false},
		{"官方h3-2k", "特价/官方h3-2k", "https://upstream.example", "2k", 15, false},
		{"官方h3-1080p", "特价/官方h3-1080p", "https://upstream.example", "1080p", 15, false},
		{"wan3.0(图片+音频参考)", "特价/wan3.0-image", "https://upstream.example", "", 5, true},
		{"sd2.5(满血原生过脸30-10-10/480P)", "全能sd2.5(满血原生过脸30-10-10/480P)", "https://upstream.example", "480p", 30, false},
		{"sd2.5-720p(9图30秒小几率卡脸内置彩绘过脸)", "特价稳定sd2.5-720p-ch1(9图30秒小几率卡脸内置彩绘过脸)", "https://upstream.example", "720p", 30, false},
		{"sd2.0(9-3-3/4-15秒原生不卡脸720P)", "003系列/sd2.0(9-3-3/4-15秒原生不卡脸720P)", "https://upstream.example", "720p", 15, false},
	}
	for _, tc := range cases {
		t.Run(tc.public, func(t *testing.T) {
			payload := map[string]any{"model": tc.public, "prompt": "A quiet sunrise over the sea", "duration": tc.duration, "ratio": "16:9"}
			if tc.resolution != "" {
				payload["resolution"] = tc.resolution
			}
			if tc.image {
				payload["referenceImages"] = []string{"https://cdn.example/reference.png"}
			}
			raw, err := common.Marshal(payload)
			require.NoError(t, err)
			c := newVideo2JSONContext(t, string(raw))
			info := &relaycommon.RelayInfo{OriginModelName: tc.public, RequestURLPath: "/pg/videos", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.upstream, ChannelBaseUrl: tc.base}}
			adaptor := &TaskAdaptor{baseURL: tc.base}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
			url, err := adaptor.BuildRequestURL(info)
			require.NoError(t, err)
			require.Equal(t, tc.base+"/v1/videos", url)
			reader, err := adaptor.BuildRequestBody(c, info)
			require.NoError(t, err)
			body, err := io.ReadAll(reader)
			require.NoError(t, err)
			var actual map[string]any
			require.NoError(t, common.Unmarshal(body, &actual))
			require.Equal(t, tc.upstream, actual["model"])
			parameters := actual
			if tc.base == "https://api.meaicc.com" {
				parameters = actual["parameters"].(map[string]any)
			}
			require.Equal(t, float64(tc.duration), parameters["duration"])
			if tc.resolution != "" {
				require.Equal(t, tc.resolution, parameters["resolution"])
			}
			stored, err := relaycommon.GetTaskRequest(c)
			require.NoError(t, err)
			require.Equal(t, tc.public, stored.Model)
		})
	}
}

func TestIntegrationRejectsUnsupportedFixedParametersAndMissingReferences(t *testing.T) {
	duration := 4
	req := videosRequest{Prompt: "sunrise", Duration: &duration, Resolution: "2k"}
	require.Error(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("官方h3-2k")))
	duration = 5
	require.NoError(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("官方h3-2k")))
	req.Resolution = "720p"
	require.Error(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("官方h3-2k")))
	req.Resolution = ""
	require.Error(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("特价/wan3.0-image")))
	req.ReferenceImages = []string{"https://cdn.example/ref.png"}
	require.NoError(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("特价/wan3.0-image")))
	duration = 0
	require.Error(t, validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters("特价/wan3.0-image")))
}
