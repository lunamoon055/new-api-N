package common

import "strings"

func IsLinkskyAsyncImageModelName(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "gpt-image-2", "gpt-image-2.5", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "nano-banana-pro", "nano-banana2", "seedream-5-0":
		return true
	default:
		return false
	}
}
