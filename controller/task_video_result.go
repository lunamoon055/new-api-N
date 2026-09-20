package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/logger"
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
	if resultURL == "" {
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}

	if strings.HasPrefix(resultURL, "data:") {
		if service.HasEnabledMediaStorage() {
			if storedURL, err := service.UploadMediaURL(context.Background(), resultURL, "video/mp4"); err == nil && storedURL != "" {
				task.PrivateData.ResultURL = storedURL
				task.FailReason = storedURL
				return
			} else if err != nil {
				logger.LogWarn(context.Background(), fmt.Sprintf("media storage upload failed for video task %s; keeping task proxy: %v", task.TaskID, err))
			}
		}
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return
	}

	if service.HasEnabledMediaStorage() {
		if storedURL, err := service.UploadVideoTaskURL(context.Background(), task, resultURL); err == nil && storedURL != "" {
			resultURL = storedURL
		} else if err != nil {
			logger.LogWarn(context.Background(), fmt.Sprintf("media storage upload failed for video task %s; keeping upstream URL: %v", task.TaskID, err))
		}
	}

	task.PrivateData.ResultURL = resultURL
	task.FailReason = resultURL
}
