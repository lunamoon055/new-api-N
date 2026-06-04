package router

import "testing"

func TestShouldServeDefaultIndex(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/creation", want: true},
		{path: "/creation/", want: true},
		{path: "/creation/history", want: true},
		{path: "/", want: false},
		{path: "/channels", want: false},
		{path: "/assets/index.js", want: false},
	}

	for _, tt := range tests {
		if got := shouldServeDefaultIndex(tt.path); got != tt.want {
			t.Fatalf("shouldServeDefaultIndex(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
