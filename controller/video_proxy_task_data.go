package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func extractTaskDataVideoURL(task *model.Task) string {
	if task == nil || len(task.Data) == 0 {
		return ""
	}
	var payload any
	if err := common.Unmarshal(task.Data, &payload); err != nil {
		return ""
	}
	return findFirstVideoURL(payload)
}

func findFirstVideoURL(value any) string {
	switch v := value.(type) {
	case string:
		value := strings.TrimSpace(v)
		if isResolvableVideoURL(value) {
			return value
		}
		if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
			var nested any
			if err := common.Unmarshal([]byte(value), &nested); err == nil {
				return findFirstVideoURL(nested)
			}
		}
	case []any:
		for _, item := range v {
			if url := findFirstVideoURL(item); url != "" {
				return url
			}
		}
	case map[string]any:
		for _, key := range []string{"video_url", "url", "result_url", "output_url", "download_url"} {
			if url, ok := v[key].(string); ok {
				url = strings.TrimSpace(url)
				if isResolvableVideoURL(url) {
					return url
				}
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
			if url := findFirstVideoURL(v[key]); url != "" {
				return url
			}
		}
	}
	return ""
}

func isResolvableVideoURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") ||
		strings.HasPrefix(rawURL, "https://") ||
		strings.HasPrefix(rawURL, "data:")
}
