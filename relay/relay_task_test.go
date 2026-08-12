package relay

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
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

func TestTaskModel2DtoIncludesRequestedModelName(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName:   "Seedance-2.5",
			UpstreamModelName: "seedance-2.5",
		},
	}

	taskDto := TaskModel2Dto(task)

	require.Equal(t, "Seedance-2.5", taskDto.ModelName)
}

func TestTaskModel2DtoFallsBackToUpstreamModelName(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			UpstreamModelName: "seedance-2.5",
		},
	}

	taskDto := TaskModel2Dto(task)

	require.Equal(t, "seedance-2.5", taskDto.ModelName)
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

func TestTaskModel2DtoTranslatesHistoricalVideoFailureAndRedactsRawError(t *testing.T) {
	rawReason := `{"error":{"message":"invalid image_urls[0]: image url returned 404"}}`
	task := &model.Task{
		Platform:   constant.TaskPlatform("37"),
		Status:     model.TaskStatusFailure,
		FailReason: rawReason,
		Data:       []byte(rawReason),
	}

	taskDto := TaskModel2Dto(task)

	require.Equal(t, "参考图片链接不存在或已失效，请重新上传图片后再试。", taskDto.FailReason)
	require.Empty(t, taskDto.RawFailReason)
	require.Empty(t, taskDto.ResultURL)
	payload, err := common.Marshal(taskDto)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "image url returned 404")
	require.Contains(t, string(payload), "参考图片链接不存在或已失效")
}

func TestTaskModel2DtoWithInputMaterialsIncludesRootFailureDiagnostics(t *testing.T) {
	rawReason := `{"error":{"message":"invalid audio_url: audio url returned 404"}}`
	task := &model.Task{
		Platform:   constant.TaskPlatform("37"),
		Status:     model.TaskStatusFailure,
		FailReason: "参考音频链接不存在或已失效，请重新上传音频后再试。",
		PrivateData: model.TaskPrivateData{
			UpstreamError: rawReason,
		},
		Data: []byte(rawReason),
	}

	taskDto := TaskModel2DtoWithInputMaterials(task)

	require.Equal(t, task.FailReason, taskDto.FailReason)
	require.Equal(t, rawReason, taskDto.RawFailReason)
	require.NotContains(t, string(taskDto.Data), "audio url returned 404")
}

func TestTaskModel2DtoRecoversHistoricalChineseFailureWithoutRetranslating(t *testing.T) {
	rawReason := "内容触发安全审核或版权限制，请调整输入内容或素材后重试"
	task := &model.Task{
		Platform:   constant.TaskPlatform("1"),
		Status:     model.TaskStatusFailure,
		FailReason: "视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。",
		Data:       []byte(`{"status":"FAILED: 内容触发安全审核或版权限制，请调整输入内容或素材后重试"}`),
	}

	publicDto := TaskModel2Dto(task)
	rootDto := TaskModel2DtoWithInputMaterials(task)

	require.Equal(t, rawReason, publicDto.FailReason)
	require.Empty(t, publicDto.RawFailReason)
	require.Equal(t, rawReason, rootDto.FailReason)
	require.Equal(t, rawReason, rootDto.RawFailReason)
}

func TestTaskModel2DtoKeepsNonVideoFailureUnchanged(t *testing.T) {
	rawReason := "midjourney upstream failed"
	task := &model.Task{
		Platform:   constant.TaskPlatformMidjourney,
		Status:     model.TaskStatusFailure,
		FailReason: rawReason,
		Data:       []byte(`{"description":"midjourney upstream failed"}`),
	}

	taskDto := TaskModel2Dto(task)

	require.Equal(t, rawReason, taskDto.FailReason)
	require.Equal(t, task.Data, taskDto.Data)
}

func TestNormalizeOpenAIVideoErrorResponseReturnsTranslatedMessage(t *testing.T) {
	rawReason := "PROVIDER_MODERATION_ERROR: TRADEMARK"
	body, err := common.Marshal(dto.OpenAIVideo{
		ID:     "task_public",
		Status: dto.VideoStatusFailed,
		Error: &dto.OpenAIVideoError{
			Code:    "provider_moderation_error",
			Message: rawReason,
		},
	})
	require.NoError(t, err)
	task := &model.Task{
		Status: model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{
			UpstreamError: rawReason,
		},
	}

	normalized := normalizeOpenAIVideoErrorResponse(body, task)

	var response dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(normalized, &response))
	require.NotNil(t, response.Error)
	require.Equal(t, "内容审核未通过，可能涉及商标或品牌侵权，请修改提示词或参考素材。", response.Error.Message)
	require.NotContains(t, string(normalized), rawReason)
}

func TestIsTaskSubmitSuccessStatusAcceptsAny2xx(t *testing.T) {
	require.True(t, isTaskSubmitSuccessStatus(http.StatusOK))
	require.True(t, isTaskSubmitSuccessStatus(http.StatusCreated))
	require.False(t, isTaskSubmitSuccessStatus(http.StatusBadRequest))
}
