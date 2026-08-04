package relay

import (
	"net/http"
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

func TestTaskModel2DtoIncludesPromptFromTaskProperties(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Properties: model.Properties{
			Input: "a cinematic product video",
		},
	}

	dto := TaskModel2Dto(task)

	require.Equal(t, "a cinematic product video", dto.Prompt)
}

func TestTaskModel2DtoRedactsInputMaterialsByDefault(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Properties: model.Properties{
			Input:       "a cinematic product video",
			InputImages: []string{"https://cdn.example.com/reference.png"},
			InputVideos: []string{"https://cdn.example.com/reference.mp4"},
			InputAudios: []string{"https://cdn.example.com/reference.wav"},
		},
	}

	dto := TaskModel2Dto(task)
	properties, ok := dto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, "a cinematic product video", properties.Input)
	require.Empty(t, properties.InputImages)
	require.Empty(t, properties.InputVideos)
	require.Empty(t, properties.InputAudios)
	require.Equal(t, []string{"https://cdn.example.com/reference.png"}, task.Properties.InputImages)

	payload, err := common.Marshal(dto)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "input_images")
	require.NotContains(t, string(payload), "reference.png")
}

func TestTaskModel2DtoWithInputMaterialsIncludesRootOnlyFields(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Properties: model.Properties{
			InputImages: []string{"https://cdn.example.com/reference.png"},
			InputVideos: []string{"https://cdn.example.com/reference.mp4"},
			InputAudios: []string{"https://cdn.example.com/reference.wav"},
		},
	}

	dto := TaskModel2DtoWithInputMaterials(task)
	properties, ok := dto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, task.Properties.InputImages, properties.InputImages)
	require.Equal(t, task.Properties.InputVideos, properties.InputVideos)
	require.Equal(t, task.Properties.InputAudios, properties.InputAudios)
}

func TestIsTaskSubmitSuccessStatusAcceptsAny2xx(t *testing.T) {
	require.True(t, isTaskSubmitSuccessStatus(http.StatusOK))
	require.True(t, isTaskSubmitSuccessStatus(http.StatusCreated))
	require.False(t, isTaskSubmitSuccessStatus(http.StatusBadRequest))
}
