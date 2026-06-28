package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/stretchr/testify/require"
)

func TestTaskModel2DtoUsesDirectVideoURLFromDataWhenResultURLIsProxy(t *testing.T) {
	payload, err := common.Marshal(map[string]any{
		"code": "success",
		"data": map[string]any{
			"result_url": taskcommon.BuildProxyURL("task_public"),
			"data": map[string]any{
				"data": []any{
					map[string]any{"url": "https://cdn.example.com/task-video.mp4"},
				},
			},
		},
	})
	require.NoError(t, err)

	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: taskcommon.BuildProxyURL("task_public"),
		},
		Data: payload,
	}

	dto := TaskModel2Dto(task)

	require.Equal(t, "https://cdn.example.com/task-video.mp4", dto.ResultURL)
}

func TestTaskModel2DtoKeepsProxyResultURLWhenNoDirectVideoURLExists(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: taskcommon.BuildProxyURL("task_public"),
		},
		Data: []byte(`{"metadata":{"url":"/v1/videos/task_public/content"}}`),
	}

	dto := TaskModel2Dto(task)

	require.Equal(t, taskcommon.BuildProxyURL("task_public"), dto.ResultURL)
}
