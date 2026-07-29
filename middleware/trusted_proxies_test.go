package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func trustedProxyTestClientIP(t *testing.T, trusted string, remoteAddr string, forwardedFor string) string {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NoError(t, ConfigureTrustedProxies(engine, trusted))
	engine.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		request.Header.Set("X-Forwarded-For", forwardedFor)
	}
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	return recorder.Body.String()
}

func TestConfigureTrustedProxiesDefaultsToNone(t *testing.T) {
	clientIP := trustedProxyTestClientIP(t, "", "203.0.113.9:4321", "198.51.100.20")
	require.Equal(t, "203.0.113.9", clientIP)
}

func TestConfigureTrustedProxiesHonorsExplicitProxy(t *testing.T) {
	clientIP := trustedProxyTestClientIP(t, "127.0.0.1", "127.0.0.1:4321", "198.51.100.20")
	require.Equal(t, "198.51.100.20", clientIP)
}

func TestConfigureTrustedProxiesRejectsUnsafeValues(t *testing.T) {
	for _, value := range []string{
		"not-an-ip",
		"10.0.0.0/not-a-prefix",
		"0.0.0.0/0",
		"::/0",
	} {
		t.Run(value, func(t *testing.T) {
			err := ConfigureTrustedProxies(gin.New(), value)
			require.Error(t, err)
		})
	}
}

func TestConfigureTrustedProxiesAcceptsMultipleCIDRs(t *testing.T) {
	engine := gin.New()
	require.NoError(t, ConfigureTrustedProxies(engine, " 127.0.0.1, 10.0.0.0/8,127.0.0.1 "))
}
