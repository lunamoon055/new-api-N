package common

import (
	_ "embed"
	"sort"
	"strings"
)

// UI hints transcribed from V6K's public video catalog on 2026-10-24.
// Empty ranges are unpublished, not permission to assume another provider's
// capabilities. Runtime model authorization remains with the upstream API.
//
//go:embed v6k_video_models.json
var v6kVideoSnapshot []byte

var v6kVideoParameters = func() map[string]*ModelIntegrationParameters {
	values := map[string]*ModelIntegrationParameters{}
	_ = Unmarshal(v6kVideoSnapshot, &values)
	return values
}()

func GetV6KVideoParameters(modelName string) *ModelIntegrationParameters {
	if parameters := v6kVideoParameters[strings.TrimSpace(modelName)]; parameters != nil {
		return parameters
	}
	return &ModelIntegrationParameters{
		DurationField: "seconds", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "images", VideoField: "videos", AudioField: "audios", OmitRatio: true,
	}
}

func V6KVideoModelIDs() []string {
	ids := make([]string, 0, len(v6kVideoParameters))
	for id := range v6kVideoParameters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
