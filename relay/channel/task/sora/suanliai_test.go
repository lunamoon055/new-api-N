package sora

import (
	"io"
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
