package openai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
