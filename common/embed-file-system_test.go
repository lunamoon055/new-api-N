package common

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

type fakeServeFileSystem struct {
	paths map[string]bool
}

func TestEmbeddedDocumentationDirectoryDoesNotRedirectSPARoute(t *testing.T) {
	fs := &embedFileSystem{FileSystem: http.FS(fstest.MapFS{
		"docs/model-integration.html": {Data: []byte("model integration document")},
		"static/app.js":               {Data: []byte("application script")},
	})}
	router := gin.New()
	router.Use(static.Serve("/", fs))
	router.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("application shell"))
	})
	for _, tt := range []struct {
		path string
		body string
	}{
		{"/docs", "application shell"},
		{"/docs/", "application shell"},
		{"/docs/model-integration.html", "model integration document"},
		{"/static/app.js", "application script"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != http.StatusOK || response.Header().Get("Location") != "" {
				t.Fatalf("unexpected redirect or status: %d %q", response.Code, response.Header().Get("Location"))
			}
			if response.Body.String() != tt.body {
				t.Fatalf("unexpected response: %q", response.Body.String())
			}
		})
	}
}

func (f fakeServeFileSystem) Exists(_ string, path string) bool {
	return f.paths[strings.TrimPrefix(path, "/")]
}

func (f fakeServeFileSystem) Open(_ string) (http.File, error) {
	return nil, os.ErrNotExist
}

func TestThemeAwareFileSystemKeepsAssetFamiliesStable(t *testing.T) {
	defaultFS := fakeServeFileSystem{paths: map[string]bool{
		"static/js/app.js": true,
		"default-only.js":  true,
	}}
	classicFS := fakeServeFileSystem{paths: map[string]bool{
		"assets/app.js":   true,
		"classic-only.js": true,
	}}
	themeFS := NewThemeAwareFS(defaultFS, classicFS)

	SetTheme("classic")
	t.Cleanup(func() { SetTheme("default") })

	if !themeFS.Exists("", "static/js/app.js") {
		t.Fatal("static assets should always be served from the default frontend")
	}
	if !themeFS.Exists("", "/static/js/app.js") {
		t.Fatal("static assets should tolerate a leading slash")
	}
	if !themeFS.Exists("", "assets/app.js") {
		t.Fatal("classic assets should always be served from the classic frontend")
	}
	if !themeFS.Exists("", "classic-only.js") {
		t.Fatal("theme-specific paths should use the active classic frontend")
	}
	if themeFS.Exists("", "default-only.js") {
		t.Fatal("theme-specific paths should not leak from the inactive default frontend")
	}

	SetTheme("default")
	if !themeFS.Exists("", "default-only.js") {
		t.Fatal("theme-specific paths should use the active default frontend")
	}
}
