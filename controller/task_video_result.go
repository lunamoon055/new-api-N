package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

// applyVideoTaskResultURL stores only a media-storage URL when media storage is
// enabled. It returns true when the upstream task succeeded but the
// media copy did not, so the caller can keep the task in processing state and
// let the next polling cycle retry without exposing the upstream URL.
func applyVideoTaskResultURL(task *model.Task, taskResult *relaycommon.TaskInfo) bool {
	if task == nil || taskResult == nil || taskResult.Status != string(model.TaskStatusSuccess) {
		return false
	}

	resultURL := strings.TrimSpace(taskResult.Url)
	if resultURL == "" {
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return false
	}

	if strings.HasPrefix(resultURL, "data:") {
		if service.HasEnabledMediaStorage() {
			if storedURL, err := service.UploadMediaURLWithRetry(context.Background(), resultURL, "video/mp4"); err == nil && storedURL != "" {
				task.PrivateData.ResultURL = storedURL
				task.FailReason = storedURL
				return false
			} else if err != nil {
				logger.LogWarn(context.Background(), fmt.Sprintf("media storage upload failed for video task %s; keeping task processing", task.TaskID))
				service.MarkVideoMediaTransferPending(task)
				return true
			}
		}
		task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		return false
	}

	if service.HasEnabledMediaStorage() {
		if _, err := service.EnqueueVideoTaskMediaTransfer(context.Background(), task, resultURL); err == nil {
			return true
		} else {
			logger.LogWarn(context.Background(), fmt.Sprintf("enqueue media transfer failed for video task %s; using first-stage retry", task.TaskID))
			if storedURL, uploadErr := service.UploadVideoTaskURL(context.Background(), task, resultURL); uploadErr == nil && storedURL != "" {
				resultURL = storedURL
				// If a durable queue claim lost a concurrent unique insert,
				// the legacy fallback must restore the terminal task state
				// before the caller persists it.
				task.Status = model.TaskStatusSuccess
				task.MediaTransferState = ""
				task.Progress = "100%"
				if task.FinishTime == 0 {
					task.FinishTime = time.Now().Unix()
				}
			} else {
				service.MarkVideoMediaTransferPending(task)
				return true
			}
		}
	}

	task.PrivateData.ResultURL = resultURL
	task.FailReason = resultURL
	return false
}
