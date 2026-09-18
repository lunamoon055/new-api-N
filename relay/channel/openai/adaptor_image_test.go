package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLPreservesImageAndChatEndpoints(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	testCases := []struct {
		name      string
		path      string
		relayMode int
		wantURL   string
	}{
		{
			name:      "image generation",
			path:      "/v1/images/generations",
			relayMode: relayconstant.RelayModeImagesGenerations,
			wantURL:   "https://linksky.top/v1/images/generations",
		},
		{
			name:      "chat completions",
			path:      "/v1/chat/completions",
			relayMode: relayconstant.RelayModeChatCompletions,
			wantURL:   "https://linksky.top/v1/chat/completions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				RequestURLPath: tc.path,
				RelayMode:      tc.relayMode,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelBaseUrl: "https://linksky.top",
					ChannelType:    constant.ChannelTypeOpenAI,
				},
			}
			got, err := adaptor.GetRequestURL(info)
			require.NoError(t, err)
			require.Equal(t, tc.wantURL, got)
		})
	}
}

func TestConvertImageRequestPreservesGptImage2References(t *testing.T) {
	t.Parallel()

	var request dto.ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{
		"model": "gpt-image2",
		"prompt": "make it cinematic",
		"output_resolution": "1K",
		"aspect_ratio": "1:1",
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "make it cinematic"},
					{"type": "image_url", "image_url": {"url": "https://cdn.example/source.png"}}
				]
			}
		]
	}`), &request))

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}
	converted, err := adaptor.ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		info,
		request,
	)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	require.Equal(t, "gpt-image2", payload["model"])
	require.Equal(t, "make it cinematic", payload["prompt"])
	require.Equal(t, "1K", payload["output_resolution"])
	require.Equal(t, "1:1", payload["aspect_ratio"])

	messages, ok := payload["messages"].([]any)
	require.True(t, ok, "messages should be preserved for gpt-image2 references")
	require.Len(t, messages, 1)

	firstMessage, ok := messages[0].(map[string]any)
	require.True(t, ok)
	content, ok := firstMessage["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 2)
	imageContent, ok := content[1].(map[string]any)
	require.True(t, ok)
	imageURL, ok := imageContent["image_url"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://cdn.example/source.png", imageURL["url"])
}

func TestConvertImageRequestPreservesRequestedImageSize(t *testing.T) {
	t.Parallel()

	for _, size := range []string{"1024x1024", "2048x1152", "3840x2160"} {
		t.Run(size, func(t *testing.T) {
			t.Parallel()

			request := dto.ImageRequest{
				Model:  "gpt-image-2.5-flare",
				Prompt: "make it cinematic",
				Size:   size,
			}
			adaptor := &Adaptor{}
			info := &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeImagesGenerations,
			}

			converted, err := adaptor.ConvertImageRequest(
				gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
				info,
				request,
			)
			require.NoError(t, err)

			body, err := common.Marshal(converted)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.Unmarshal(body, &payload))
			require.Equal(t, size, payload["size"])
		})
	}
}

func TestConvertImageRequestPreservesNanoBananaPayload(t *testing.T) {
	t.Parallel()

	var request dto.ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{
		"model":"nano-banana-pro",
		"prompt":"create a product photo",
		"size":"2048x2048",
		"aspect_ratio":"1:1",
		"images":["https://cdn.example/reference.png"]
	}`), &request))

	converted, err := (&Adaptor{}).ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		&relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations},
		request,
	)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	require.Equal(t, "nano-banana-pro", payload["model"])
	require.Equal(t, "create a product photo", payload["prompt"])
	require.Equal(t, "2048x2048", payload["size"])
	require.Equal(t, "1:1", payload["aspect_ratio"])
	require.Equal(t, []any{"https://cdn.example/reference.png"}, payload["images"])
}

func TestOpenaiHandlerWithUsagePassesImageResponseThrough(t *testing.T) {
	t.Parallel()

	const responseJSON = `{"created":1789700000,"data":[{"b64_json":"aW1hZ2U="}],"usage":{"input_tokens":7,"output_tokens":11,"total_tokens":18}}`
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(responseJSON)),
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	usage, apiErr := OpenaiHandlerWithUsage(ctx, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}, response)
	require.Nil(t, apiErr)
	require.Equal(t, responseJSON, recorder.Body.String())
	require.Equal(t, 18, usage.TotalTokens)
}
