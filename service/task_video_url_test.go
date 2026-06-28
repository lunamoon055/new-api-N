package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

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

	require.Equal(t, "https://cdn.example.com/dt-video.mp4", ExtractTaskDataVideoURL(task))
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

	require.Equal(t, "https://cdn.example.com/dt-video.mp4", ExtractTaskDataVideoURL(task))
}

func TestExtractTaskDataVideoURLIgnoresRelativeProxyURL(t *testing.T) {
	payload, err := common.Marshal(map[string]any{
		"metadata": map[string]any{
			"url": "/v1/videos/task_public/content",
		},
	})
	require.NoError(t, err)

	task := &model.Task{TaskID: "task_public", Data: payload}

	require.Empty(t, ExtractTaskDataVideoURL(task))
}
