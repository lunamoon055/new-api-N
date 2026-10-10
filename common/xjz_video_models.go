package common

import "strings"

// IsXJZVideoModel matches the public model IDs in the XJZ video documentation.
// These IDs are case-sensitive and may differ from the shorter display names.
func IsXJZVideoModel(modelName string) bool {
	switch strings.TrimSpace(modelName) {
	case "特价dola2.5-900-720P-固定30s",
		"特价dola-sd2.0-900-720p",
		"TTP-grok",
		"gz稳定sd2.0-903-720P",
		"Xminimex-h3-2K-933",
		"特价sd2.0-720p-933-过脸",
		"特价稳sd-2.5-30s-720P可10图",
		"特价稳sd2.0-933-720p",
		"gz稳定sd2.5-720全参过脸",
		"gz稳定sd2.5-480全参过脸",
		"gz稳定sd2.5-1080全参过脸",
		"特价sd2.0-933-过脸",
		"MiniMaxH3":
		return true
	default:
		return false
	}
}

// IsXJZPerCallVideoModel identifies the exact XJZ models whose provider cost
// is per request. Their local video billing mode must preserve that unit.
func IsXJZPerCallVideoModel(modelName string) bool {
	switch strings.TrimSpace(modelName) {
	case "特价dola2.5-900-720P-固定30s", "特价dola-sd2.0-900-720p", "TTP-grok",
		"gz稳定sd2.0-903-720P", "Xminimex-h3-2K-933", "特价稳sd-2.5-30s-720P可10图",
		"特价稳sd2.0-933-720p", "特价sd2.0-933-过脸", "MiniMaxH3":
		return true
	default:
		return false
	}
}
