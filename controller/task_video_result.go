package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func applyVideoTaskResultURL(task *model.Task, taskResult *relaycommon.TaskInfo) {
	if task == nil || taskResult == nil || taskResult.Status != string(model.TaskStatusSuccess) {
		return
	}

	resultURL := strings.TrimSpace(taskResult.Url)
	if resultURL == "" || strings.HasPrefix(resultURL, "data:") {
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}

	task.PrivateData.ResultURL = resultURL
	task.FailReason = resultURL
}
