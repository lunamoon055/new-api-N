package middleware

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyLimitAllowsDocumentedReferenceVideoSize(t *testing.T) {
	previous := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 128
	t.Cleanup(func() { constant.MaxRequestBodyMB = previous })

	uploadRequest, err := http.NewRequest(http.MethodPost, "/api/creation/reference-files", nil)
	require.NoError(t, err)
	require.Equal(t, 202, requestBodyLimitMB(uploadRequest))

	regularRequest, err := http.NewRequest(http.MethodPost, "/api/creation/video/async-generations", nil)
	require.NoError(t, err)
	require.Equal(t, 128, requestBodyLimitMB(regularRequest))
}
