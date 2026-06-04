package common

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

type fakeServeFileSystem struct {
	paths map[string]bool
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
