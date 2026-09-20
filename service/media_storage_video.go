package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

// MarkVideoMediaTransferPending hides the upstream result and reopens the
// task for polling after an upstream generation succeeded but media storage
// did not. It is shared by the ordinary polling and realtime task paths.
func MarkVideoMediaTransferPending(task *model.Task) {
	if task == nil {
		return
	}
	task.Status = model.TaskStatusInProgress
	task.MediaTransferState = string(model.MediaTransferPending)
	task.Progress = "99%"
	task.FinishTime = 0
	task.PrivateData.ResultURL = ""
	task.FailReason = ""
	// Replace any upstream response payload that might contain the expiring
	// source URL while the task is publicly reported as processing.
	task.SetData(map[string]string{"media_transfer": "pending"})
}

// UploadVideoTaskURL uploads a completed task URL using the channel context
// recorded on the task. This covers legacy task update paths that only have a
// task model available and need credentials for a protected same-origin
// /v1/videos/.../content endpoint.
func UploadVideoTaskURL(ctx context.Context, task *model.Task, resultURL string) (string, error) {
	if task == nil {
		return "", fmt.Errorf("video task is nil")
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		return "", fmt.Errorf("get video task channel: %w", err)
	}
	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}
	key := channel.Key
	if task.PrivateData.Key != "" {
		key = task.PrivateData.Key
	}
	options := taskVideoMediaDownloadOptions(channel.Type, baseURL, key, channel.GetSetting().Proxy)
	return UploadMediaURLWithOptionsRetry(ctx, strings.TrimSpace(resultURL), "video/mp4", options)
}

// EnqueueVideoTaskMediaTransfer creates the durable stage-two job using the
// channel credentials already associated with the task. It is used by legacy
// and realtime result paths that do not run through the main polling loop.
func EnqueueVideoTaskMediaTransfer(ctx context.Context, task *model.Task, resultURL string) (bool, error) {
	if task == nil {
		return false, fmt.Errorf("video task is nil")
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		return false, fmt.Errorf("get video task channel: %w", err)
	}
	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}
	key := channel.Key
	if task.PrivateData.Key != "" {
		key = task.PrivateData.Key
	}
	options := taskVideoMediaDownloadOptions(channel.Type, baseURL, key, channel.GetSetting().Proxy)
	previous := *task
	if task.ID > 0 {
		if persisted, loadErr := model.GetTaskByID(task.ID); loadErr == nil && persisted != nil {
			previous = *persisted
		}
	}
	return EnqueueVideoMediaTransfer(ctx, task, &previous, strings.TrimSpace(resultURL), options)
}
