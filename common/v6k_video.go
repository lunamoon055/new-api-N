package common

import (
	_ "embed"
	"sort"
	"strings"
)

// UI hints transcribed from V6K's public video catalog on 2026-10-24,
// with the explicitly selected ZH additions verified on 2026-10-26.
// Empty ranges are unpublished, not permission to assume another provider's
// capabilities. Runtime model authorization remains with the upstream API.
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

func GetV6KVideoParameters(modelName string) *ModelIntegrationParameters {
	if parameters := v6kVideoParameters[strings.TrimSpace(modelName)]; parameters != nil {
		return &parameters.ModelIntegrationParameters
	}
	return &ModelIntegrationParameters{
		DurationField: "seconds", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "images", VideoField: "videos", AudioField: "audios", OmitRatio: true,
	}
}

// GetV6KValidatedVideoParameters limits stricter validation to exact models
// whose published capabilities were confirmed for this integration.
func GetV6KValidatedVideoParameters(modelName string) *ModelIntegrationParameters {
	if spec := v6kVideoParameters[strings.TrimSpace(modelName)]; spec != nil && spec.ValidateRequest {
		return &spec.ModelIntegrationParameters
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
