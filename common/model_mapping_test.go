package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResolveModelMappingKeepsExactPublicAndFinalIDs(t *testing.T) {
	for _, target := range []string{"gpt-4o", "gpt-image-2", "grok-1.5-700-B", "Seedance 2.0:X1"} {
		raw, err := Marshal(map[string]string{"public": "next", "next": " " + target + " ", target: target})
		require.NoError(t, err)
		resolved, mapped, err := ResolveModelMapping("public", string(raw))
		require.NoError(t, err)
		require.True(t, mapped)
		require.Equal(t, target, resolved)
	}
	_, _, err := ResolveModelMapping("public", `{"public":"next","next":"public"}`)
	require.ErrorContains(t, err, "model_mapping_contains_cycle")
	_, _, err = ResolveModelMapping("public", `invalid`)
	require.ErrorContains(t, err, "unmarshal_model_mapping_failed")
}
