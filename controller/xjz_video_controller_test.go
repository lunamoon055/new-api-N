package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

var xjzVideoTestCases = []struct {
	modelName string
	duration  int
}{
	{"特价dola2.5-900-720P-固定30s", 30},
	{"特价dola-sd2.0-900-720p", 5},
	{"TTP-grok", 6},
	{"gz稳定sd2.0-903-720P", 5},
	{"Xminimex-h3-2K-933", 5},
	{"特价sd2.0-720p-933-过脸", 5},
	{"特价稳sd-2.5-30s-720P可10图", 30},
	{"特价稳sd2.0-933-720p", 5},
	{"gz稳定sd2.5-720全参过脸", 4},
	{"gz稳定sd2.5-480全参过脸", 4},
	{"gz稳定sd2.5-1080全参过脸", 4},
	{"特价sd2.0-933-过脸", 4},
	{"MiniMaxH3", 4},
}

func TestXJZVideoChannelTestUsesDocumentedEndpointAndValidDurations(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeOpenAI}
	for _, tc := range xjzVideoTestCases {
		t.Run(tc.modelName, func(t *testing.T) {
			require.True(t, common.IsXJZVideoModel(" "+tc.modelName+" "))
			for _, requestedEndpoint := range []string{"", string(constant.EndpointTypeOpenAI), channelTestEndpointOpenAIVideoAsync} {
				endpoint, path, format := resolveChannelTestEndpoint(channel, tc.modelName, requestedEndpoint)
				require.Equal(t, string(constant.EndpointTypeOpenAIVideo), endpoint)
				require.Equal(t, "/v1/videos", path)
				require.Equal(t, types.RelayFormat(types.RelayFormatTask), format)
			}

			request := buildTestRequest(tc.modelName, string(constant.EndpointTypeOpenAIVideo), channel, false)
			taskRequest, ok := request.(relaycommon.TaskSubmitReq)
			require.True(t, ok)
			require.Equal(t, tc.modelName, taskRequest.Model)
			require.NotEmpty(t, taskRequest.Prompt)
			require.Equal(t, tc.duration, taskRequest.Duration)
			require.Equal(t, "16:9", taskRequest.Ratio)
			require.Empty(t, taskRequest.AspectRatio)
			require.Empty(t, taskRequest.Size)
			if tc.modelName == "TTP-grok" {
				require.Equal(t, "720p", taskRequest.Resolution)
			} else {
				require.Empty(t, taskRequest.Resolution)
			}
		})
	}
	for _, other := range []string{"特价dola-sd2.0-900", "minimaxh3", "MiniMaxH3-2K", "TTP-Grok"} {
		require.False(t, common.IsXJZVideoModel(other))
	}
}

func TestXJZVideoCatalogUsesVideoModeWithoutEndpointMetadata(t *testing.T) {
	pricing := make([]model.Pricing, 0, len(xjzVideoTestCases))
	for _, tc := range xjzVideoTestCases {
		mode, ok := getCreationModelMode(tc.modelName, nil)
		require.True(t, ok)
		require.Equal(t, creationModeVideo, mode)
		pricing = append(pricing, model.Pricing{
			ModelName:       tc.modelName,
			ModelRatio:      37.5,
			CompletionRatio: 1,
		})
	}

	// Stale provider metadata must not turn an exact, documented video ID into chat.
	providerMetadata := map[string]dto.CreationModelMetadata{
		normalizeCreationModelMetadataKey("MiniMaxH3"): {Type: "chat"},
	}
	catalog := buildCreationModelCatalogWithCategoriesAndMetadata(
		pricing, nil, creationModeVideo, nil, nil, nil, 1, providerMetadata,
	)
	require.Len(t, catalog.Modes, 1)
	require.Len(t, catalog.Modes[0].Models, len(xjzVideoTestCases))
	actual := make([]string, 0, len(catalog.Modes[0].Models))
	for _, item := range catalog.Modes[0].Models {
		actual = append(actual, item.ID)
	}
	expected := make([]string, 0, len(xjzVideoTestCases))
	for _, tc := range xjzVideoTestCases {
		expected = append(expected, tc.modelName)
	}
	require.ElementsMatch(t, expected, actual)
}

func TestXJZVideoCatalogDoesNotDisplaySelfUseFallbackAsPrice(t *testing.T) {
	modelName := "特价sd2.0-933-过脸"
	require.False(t, common.IsXJZVideoModel("特价sd2.0-933-过脸-2"))
	cost := buildCreationModelCost(model.Pricing{
		ModelName:       modelName,
		ModelRatio:      37.5,
		CompletionRatio: 1,
	}, 1)
	require.Nil(t, cost)
}

func TestXJZExpandedCatalogAndChannelTestParameters(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeOpenAI}
	for name, spec := range common.XJZVideoSpecs {
		t.Run(name, func(t *testing.T) {
			mode, ok := getCreationModelMode(name, nil)
			require.True(t, ok)
			require.Equal(t, creationModeVideo, mode)
			_, path, format := resolveChannelTestEndpoint(channel, name, "")
			require.Equal(t, "/v1/videos", path)
			require.Equal(t, types.RelayFormat(types.RelayFormatTask), format)
			request := buildXJZVideoTestRequest(name)
			parameters := common.GetVideoIntegrationParameters(name)
			require.Contains(t, parameters.Durations, request.Duration)
			if spec.RatioParameter() == "aspect_ratio" {
				require.Equal(t, "16:9", request.AspectRatio)
				require.Empty(t, request.Ratio)
			} else {
				require.Equal(t, "16:9", request.Ratio)
				require.Empty(t, request.AspectRatio)
			}
			if spec.ResolutionParameter() == "size" {
				require.Equal(t, spec.Resolution, request.Size)
			}
		})
	}
}
