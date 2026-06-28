package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func ExtractTaskDataVideoURL(task *model.Task) string {
	if task == nil || len(task.Data) == 0 {
		return ""
	}
	var payload any
	if err := common.Unmarshal(task.Data, &payload); err != nil {
		return ""
	}
	return FindTaskVideoURL(payload, task.TaskID)
}

func FindTaskVideoURL(value any, taskID string) string {
	switch v := value.(type) {
	case string:
		value := strings.TrimSpace(v)
		if IsDirectTaskVideoURL(value, taskID) {
			return value
		}
		if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
			var nested any
			if err := common.Unmarshal([]byte(value), &nested); err == nil {
				return FindTaskVideoURL(nested, taskID)
			}
		}
	case []any:
		for _, item := range v {
			if url := FindTaskVideoURL(item, taskID); url != "" {
				return url
			}
		}
	case map[string]any:
		for _, key := range []string{"video_url", "url", "result_url", "output_url", "download_url"} {
			if url := FindTaskVideoURL(v[key], taskID); url != "" {
				return url
			}
		}
		for _, key := range []string{
			"metadata",
			"result",
			"response",
			"data",
			"outputs",
			"output",
			"results",
			"videos",
			"urls",
			"video_urls",
		} {
			if url := FindTaskVideoURL(v[key], taskID); url != "" {
				return url
			}
		}
	}
	return ""
}

func IsDirectTaskVideoURL(rawURL string, taskID string) bool {
	if rawURL == "" || IsTaskVideoProxyURL(rawURL, taskID) {
		return false
	}
	return strings.HasPrefix(rawURL, "http://") ||
		strings.HasPrefix(rawURL, "https://") ||
		strings.HasPrefix(rawURL, "data:")
}

func IsTaskVideoProxyURL(rawURL string, taskID string) bool {
	if rawURL == "" || taskID == "" {
		return false
	}
	return strings.Contains(rawURL, "/v1/videos/"+taskID+"/content")
}
