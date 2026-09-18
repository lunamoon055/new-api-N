package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayErrorHandlerPreservesLongUpstreamImageError(t *testing.T) {
	t.Parallel()

	const message = `generation failed: graphql request failed: Post "https://example.ai/graphql" http: server gave HTTP response to HTTPS client`
	const downstreamMessage = `generation failed: graphql request failed: Post "https://***.ai/***" http: server gave HTTP response to HTTPS client`
	response := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"` +
			strings.ReplaceAll(message, `"`, `\"`) +
			`","type":"server_error"}}`)),
	}

	apiErr := RelayErrorHandler(context.Background(), response, false)
	require.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
	openAIError := apiErr.ToOpenAIError()
	require.Equal(t, downstreamMessage, openAIError.Message)
	require.Equal(t, "server_error", openAIError.Type)
}

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}
