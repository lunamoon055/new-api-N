package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRedemptionLinkRequiresLoginExceptForOpeningLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("redemption-link-route-test"))))
	SetApiRouter(engine)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/user/topup/link"},
		{http.MethodPost, "/api/user/topup/link/redeem"},
		{http.MethodDelete, "/api/user/topup/link"},
	} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(route.method, route.path, nil))
		require.Equal(t, http.StatusUnauthorized, response.Code, route.path)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/user/topup/link", bytes.NewBufferString(`{"key":"local-route-test-code"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"success":true`)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
}
