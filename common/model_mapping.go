package common

import (
	"fmt"
	"strings"
)

// ResolveModelMapping is shared by catalog metadata and actual relay requests.
// Public IDs are retained by callers; only the final upstream ID is resolved.
func ResolveModelMapping(modelName, rawMapping string) (string, bool, error) {
	if strings.TrimSpace(rawMapping) == "" {
		return modelName, false, nil
	}
	var mapping map[string]string
	if err := UnmarshalJsonStr(rawMapping, &mapping); err != nil {
		return modelName, false, fmt.Errorf("unmarshal_model_mapping_failed: %w", err)
	}
	current := modelName
	visited := map[string]bool{current: true}
	for {
		next := strings.TrimSpace(mapping[current])
		if next == "" || next == current {
			return current, current != modelName, nil
		}
		if visited[next] {
			return modelName, false, fmt.Errorf("model_mapping_contains_cycle")
		}
		visited[next] = true
		current = next
	}
}
