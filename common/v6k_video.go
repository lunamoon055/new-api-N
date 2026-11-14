package common

import (
	_ "embed"
	"sort"
	"strings"
)

// UI hints transcribed from V6K's public video catalog on 2026-10-24,
// with the explicitly selected ZH additions verified on 2026-10-26.
// SD2.0/SD2.5 duration ranges follow operator policy confirmed on 2026-11-13.
// Empty duration/resolution ranges remain unpublished. Missing ratios use the
// site's full ratio menu. Runtime model authorization remains with the upstream API.
//
//go:embed v6k_video_models.json
var v6kVideoSnapshot []byte

type v6kVideoSpec struct {
	ModelIntegrationParameters
	ValidateRequest bool `json:"validate_request,omitempty"`
}

var v6kVideoParameters = func() map[string]*v6kVideoSpec {
	values := map[string]*v6kVideoSpec{}
	_ = Unmarshal(v6kVideoSnapshot, &values)
	return values
}()

func IsV6KSDVideoModel(name string) bool { return name == "SD2.0" || name == "SD2.5" }

func GetV6KVideoParameters(modelName string) *ModelIntegrationParameters {
	if parameters := v6kVideoParameters[strings.TrimSpace(modelName)]; parameters != nil {
		return WithVideoRatioDefaults(&parameters.ModelIntegrationParameters)
	}
	return WithVideoRatioDefaults(&ModelIntegrationParameters{
		CapabilitiesUnknown: true,
		DurationField:       "seconds", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "images", VideoField: "videos", AudioField: "audios",
	})
}

// GetV6KValidatedVideoParameters limits stricter validation to exact models
// whose published capabilities were confirmed for this integration.
func GetV6KValidatedVideoParameters(modelName string) *ModelIntegrationParameters {
	if spec := v6kVideoParameters[strings.TrimSpace(modelName)]; spec != nil && spec.ValidateRequest {
		return WithVideoRatioDefaults(&spec.ModelIntegrationParameters)
	}
	return nil
}

func V6KVideoModelIDs() []string {
	ids := make([]string, 0, len(v6kVideoParameters))
	for id := range v6kVideoParameters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
