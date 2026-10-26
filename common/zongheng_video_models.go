package common

import (
	_ "embed"
	"sync"
)

// Snapshot of the 26 video models returned by the authorized catalog on
// 2026-10-23. These UI hints are scoped to Zongheng channels; the relay always
// validates against live capabilities before creating a video.
//
//go:embed zongheng_video_models.json
var zonghengVideoSnapshot []byte

var zonghengVideoParameters = struct {
	sync.RWMutex
	models map[string]*ModelIntegrationParameters
}{models: func() map[string]*ModelIntegrationParameters {
	models := map[string]*ModelIntegrationParameters{}
	_ = Unmarshal(zonghengVideoSnapshot, &models)
	return models
}()}

func GetZonghengVideoParameters(modelName string) *ModelIntegrationParameters {
	zonghengVideoParameters.RLock()
	defer zonghengVideoParameters.RUnlock()
	return WithVideoRatioDefaults(zonghengVideoParameters.models[modelName])
}

func SetZonghengVideoParameters(modelName string, parameters *ModelIntegrationParameters) {
	zonghengVideoParameters.Lock()
	defer zonghengVideoParameters.Unlock()
	zonghengVideoParameters.models[modelName] = parameters
}
