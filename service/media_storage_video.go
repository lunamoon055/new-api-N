package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

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
	return UploadMediaURLWithOptions(ctx, strings.TrimSpace(resultURL), "video/mp4", options)
}
