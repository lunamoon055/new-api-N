package sora

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestModelListIncludesLinkskySora2(t *testing.T) {
	require.Contains(t, ModelList, "sora2")
}

func TestBuildRequestURLUsesAsyncGenerationsForLinkskyDocPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://linksky.top"}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/video/async-generations",
	}

	got, err := adaptor.BuildRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://linksky.top/v1/video/async-generations", got)
}

func TestFetchTaskUsesAsyncGenerationsForSora2(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"task_upstream","status":"completed"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{
		"task_id": "task_upstream",
		"model":   "sora2",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	require.Equal(t, "/v1/video/async-generations/task_upstream", gotPath)
}
