package common

import "strings"

// IsXJZVideoModel matches the public model IDs in the XJZ video documentation.
// These IDs are case-sensitive and may differ from the shorter display names.
func IsXJZVideoModel(modelName string) bool {
	_, ok := XJZVideoSpecs[strings.TrimSpace(modelName)]
	return ok
}

// IsXJZPerCallVideoModel identifies the exact XJZ models whose provider cost
// is per request. Their local video billing mode must preserve that unit.
func IsXJZPerCallVideoModel(modelName string) bool {
	return XJZVideoSpecs[strings.TrimSpace(modelName)].PerCall
}
