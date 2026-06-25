package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestIsAsyncGenerationsVideoTaskIncludesLinkskyModels(t *testing.T) {
	for _, modelName := range []string{
		"sora2",
		"sora-2",
		"kling-v3",
		"video-2.0",
		"video-2.0-fast",
		"ko3",
		"veo31",
		"veo31-fast",
		"veo31-ref",
		"grok-imagine-video",
	} {
		t.Run(modelName, func(t *testing.T) {
			task := &model.Task{
				Properties: model.Properties{
					OriginModelName: modelName,
				},
			}

			require.True(t, isAsyncGenerationsVideoTask(task))
		})
	}
}

func TestExtractTaskDataVideoURLAcceptsDTNestedDataURL(t *testing.T) {
	payload, err := common.Marshal(map[string]any{
		"code":    0,
		"message": "success",
		"data": map[string]any{
			"task_id":  "dt_task_upstream",
			"status":   "completed",
			"progress": 100,
			"data": []any{
				map[string]any{"url": "https://cdn.example.com/dt-video.mp4"},
			},
		},
	})
	require.NoError(t, err)

	task := &model.Task{Data: payload}

	require.Equal(t, "https://cdn.example.com/dt-video.mp4", extractTaskDataVideoURL(task))
}

func TestExtractTaskDataVideoURLAcceptsJSONStringNestedDTData(t *testing.T) {
	nestedPayload, err := common.Marshal(map[string]any{
		"task_id":  "dt_task_upstream",
		"status":   "completed",
		"progress": 100,
		"data": []any{
			map[string]any{"url": "https://cdn.example.com/dt-video.mp4"},
		},
	})
	require.NoError(t, err)

	payload, err := common.Marshal(map[string]any{
		"code":    0,
		"message": "success",
		"data":    string(nestedPayload),
	})
	require.NoError(t, err)

	task := &model.Task{Data: payload}

	require.Equal(t, "https://cdn.example.com/dt-video.mp4", extractTaskDataVideoURL(task))
}

func TestExtractTaskDataVideoURLIgnoresRelativeProxyURL(t *testing.T) {
	payload, err := common.Marshal(map[string]any{
		"metadata": map[string]any{
			"url": "/v1/videos/task_public/content",
		},
	})
	require.NoError(t, err)

	task := &model.Task{Data: payload}

	require.Empty(t, extractTaskDataVideoURL(task))
}
