package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func performCacheRequest(t *testing.T, path string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cache())
	router.GET("/*path", handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestCacheUsesNoCacheForDocumentsAndApplicationRoutes(t *testing.T) {
	for _, path := range []string{"/", "/usage-logs/task", "/creation?tab=video"} {
		t.Run(path, func(t *testing.T) {
			recorder := performCacheRequest(t, path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			if got := recorder.Header().Get("Cache-Control"); got != cacheControlNoCache {
				t.Fatalf("Cache-Control = %q, want %q", got, cacheControlNoCache)
			}
		})
	}
}

func TestCacheUsesImmutableCachingForVersionedAssetDirectories(t *testing.T) {
	for _, path := range []string{
		"/static/js/async/3011.944e51c330.js",
		"/assets/index-Ba12cd34.js",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := performCacheRequest(t, path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			if got := recorder.Header().Get("Cache-Control"); got != cacheControlImmutableAssets {
				t.Fatalf("Cache-Control = %q, want %q", got, cacheControlImmutableAssets)
			}
		})
	}
}

func TestSetNoStoreOverridesStaticAssetCaching(t *testing.T) {
	recorder := performCacheRequest(t, "/static/js/missing-old-chunk.js", func(c *gin.Context) {
		SetNoStore(c)
		c.Status(http.StatusNotFound)
	})

	if got := recorder.Header().Get("Cache-Control"); got != cacheControlNoStore {
		t.Fatalf("Cache-Control = %q, want %q", got, cacheControlNoStore)
	}
}
