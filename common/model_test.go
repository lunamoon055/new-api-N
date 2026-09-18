package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestIsImageGenerationModelRecognizesCurrentImageFamilies(t *testing.T) {
	t.Parallel()

	imageModels := []string{
		"gpt-image-2",
		"gpt-image-2.5-flare",
		"gpt-image-2.5-sunburst",
		"gpt-image2",
		"nano-banana2",
		"nano-banana-pro",
		"gemini-2.5-flash-image",
		"gemini-3-pro-image-preview",
		"gemini-3.1-flash-image-preview",
	}
	for _, model := range imageModels {
		model := model
		t.Run(model, func(t *testing.T) {
			t.Parallel()
			require.True(t, IsImageGenerationModel(model))
		})
	}

	nonImageModels := []string{"gpt-5.4", "gemini-2.5-flash", "claude-sonnet-4"}
	for _, model := range nonImageModels {
		model := model
		t.Run(model, func(t *testing.T) {
			t.Parallel()
			require.False(t, IsImageGenerationModel(model))
		})
	}
}

func TestImageModelsAdvertiseImageAndOpenAIEndpoints(t *testing.T) {
	t.Parallel()

	for _, model := range []string{"gpt-image-2", "gpt-image-2.5-flare", "nano-banana2", "nano-banana-pro"} {
		endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeOpenAI, model)
		require.Equal(t, []constant.EndpointType{
			constant.EndpointTypeImageGeneration,
			constant.EndpointTypeOpenAI,
		}, endpoints, model)
	}
}

func TestNativeGeminiNanoBananaUsesGenerateContentEndpoints(t *testing.T) {
	t.Parallel()

	require.Equal(t, []constant.EndpointType{
		constant.EndpointTypeGemini,
		constant.EndpointTypeOpenAI,
	}, GetEndpointTypesByChannelType(constant.ChannelTypeGemini, "nano-banana-pro-preview"))

	require.Equal(t, []constant.EndpointType{
		constant.EndpointTypeImageGeneration,
		constant.EndpointTypeGemini,
		constant.EndpointTypeOpenAI,
	}, GetEndpointTypesByChannelType(constant.ChannelTypeGemini, "imagen-4.0-generate-001"))
}
