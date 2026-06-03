package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDistributorTestContext(method, path, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	c.Request = request
	return c
}

func TestGetModelRequestRecognizesAsyncVideoSubmit(t *testing.T) {
	c := newDistributorTestContext(http.MethodPost, "/v1/video/async-generations", `{"model":"sora2","prompt":"sunset"}`)

	modelRequest, shouldSelectChannel, err := getModelRequest(c)

	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.Equal(t, "sora2", modelRequest.Model)
	require.Equal(t, relayconstant.RelayModeVideoSubmit, c.GetInt("relay_mode"))
}

func TestGetModelRequestRecognizesAsyncVideoFetch(t *testing.T) {
	c := newDistributorTestContext(http.MethodGet, "/v1/video/async-generations/task_abc", "")

	_, shouldSelectChannel, err := getModelRequest(c)

	require.NoError(t, err)
	require.False(t, shouldSelectChannel)
	require.Equal(t, relayconstant.RelayModeVideoFetchByID, c.GetInt("relay_mode"))
}
