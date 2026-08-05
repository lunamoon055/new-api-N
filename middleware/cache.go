package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	cacheControlNoCache         = "no-cache"
	cacheControlNoStore         = "no-store"
	cacheControlImmutableAssets = "public, max-age=31536000, immutable"
)

func IsStaticAssetPath(path string) bool {
	return path == "/static" || strings.HasPrefix(path, "/static/") ||
		path == "/assets" || strings.HasPrefix(path, "/assets/")
}

func cacheControlForPath(path string) string {
	if IsStaticAssetPath(path) {
		return cacheControlImmutableAssets
	}
	return cacheControlNoCache
}

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Header("Cache-Control", cacheControlForPath(c.Request.URL.Path))
		c.Header("Cache-Version", "b688f2fb5be447c25e5aa3bd063087a83db32a288bf6a4f35f2d8db310e40b14")
		c.Next()
	}
}

// SetNoStore prevents browsers and intermediary CDNs from caching an error
// response for a versioned static asset that may exist in another deployment.
func SetNoStore(c *gin.Context) {
	c.Header("Cache-Control", cacheControlNoStore)
}
