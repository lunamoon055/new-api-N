package controller

import (
	"context"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

func applyVideoTaskResultURL(task *model.Task, taskResult *relaycommon.TaskInfo) {
	if task == nil || taskResult == nil || taskResult.Status != string(model.TaskStatusSuccess) {
		return
	}

	resultURL := strings.TrimSpace(taskResult.Url)
	if resultURL == "" || strings.HasPrefix(resultURL, "data:") {
		if resultURL != "" && service.HasEnabledMediaStorage() {
			if storedURL, err := service.UploadMediaURL(context.Background(), resultURL, "video/mp4"); err == nil && storedURL != "" {
				task.PrivateData.ResultURL = storedURL
				task.FailReason = storedURL
				return
			}
		}
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}
	if service.HasEnabledMediaStorage() {
		if storedURL, err := service.UploadMediaURL(context.Background(), resultURL, "video/mp4"); err == nil && storedURL != "" {
			resultURL = storedURL
		}
	}

	task.PrivateData.ResultURL = resultURL
	task.FailReason = resultURL
}
