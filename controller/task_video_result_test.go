package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestApplyVideoTaskResultURLStoresDirectURLInPrivateData(t *testing.T) {
	task := &model.Task{TaskID: "task_dt_public"}
	taskResult := &relaycommon.TaskInfo{
		Status: string(model.TaskStatusSuccess),
		Url:    "https://cdn.example.com/dt-video.mp4",
	}

	applyVideoTaskResultURL(task, taskResult)

	require.Equal(t, "https://cdn.example.com/dt-video.mp4", task.PrivateData.ResultURL)
	require.Equal(t, "https://cdn.example.com/dt-video.mp4", task.FailReason)
}

func TestApplyVideoTaskResultURLStoresProxyWhenSuccessURLIsMissing(t *testing.T) {
	task := &model.Task{TaskID: "task_dt_public"}
	taskResult := &relaycommon.TaskInfo{
		Status: string(model.TaskStatusSuccess),
	}

	applyVideoTaskResultURL(task, taskResult)

	require.Equal(t, taskcommon.BuildProxyURL(task.TaskID), task.PrivateData.ResultURL)
	require.Empty(t, task.FailReason)
}

func TestApplyVideoTaskResultURLStoresProxyForDataURL(t *testing.T) {
	task := &model.Task{TaskID: "task_dt_public"}
	taskResult := &relaycommon.TaskInfo{
		Status: string(model.TaskStatusSuccess),
		Url:    "data:video/mp4;base64,AAAA",
	}

	applyVideoTaskResultURL(task, taskResult)

	require.Equal(t, taskcommon.BuildProxyURL(task.TaskID), task.PrivateData.ResultURL)
	require.Empty(t, task.FailReason)
}
