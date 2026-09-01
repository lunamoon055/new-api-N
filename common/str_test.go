package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskSensitiveInfoPreservesQuotedURLDelimiters(t *testing.T) {
	message := `Post "https://gateway.example.ai/graphql" http: server gave HTTP response to HTTPS client`

	masked := MaskSensitiveInfo(message)

	require.Equal(t, `Post "https://***.ai/***" http: server gave HTTP response to HTTPS client`, masked)
}
