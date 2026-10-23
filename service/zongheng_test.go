package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

type zonghengTestTransport func(*http.Request) (*http.Response, error)

func (transport zonghengTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func setZonghengTestClient(t *testing.T, transport zonghengTestTransport) {
	t.Helper()
	previous := httpClient
	httpClient = &http.Client{Transport: transport}
	t.Cleanup(func() { httpClient = previous })
}

func zonghengTestResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestZonghengCapabilitiesRetriesOnlyReadOnlyTransientFailures(t *testing.T) {
	for _, firstError := range []error{io.EOF, io.ErrUnexpectedEOF, &zonghengModelsHTTPError{status: http.StatusServiceUnavailable}} {
		t.Run(firstError.Error(), func(t *testing.T) {
			calls := 0
			setZonghengTestClient(t, func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/v1/models", req.URL.Path)
				calls++
				if calls < 3 {
					return nil, firstError
				}
				return zonghengTestResponse(http.StatusOK, `{"data":[{"id":"video","capabilities":{"capability_type":"video"}}]}`), nil
			})
			models, err := FetchZonghengVideoModels(context.Background(), "https://retry.example", "key", "")
			require.NoError(t, err)
			require.Len(t, models, 1)
			require.Equal(t, 3, calls)
		})
	}
}

func TestZonghengCapabilitiesDoesNotRetryAuthOrMalformedResponse(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest, http.StatusOK} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			setZonghengTestClient(t, func(*http.Request) (*http.Response, error) {
				calls++
				return zonghengTestResponse(status, `invalid json`), nil
			})
			_, err := FetchZonghengVideoModels(context.Background(), "https://invalid.example", "key", "")
			require.Error(t, err)
			require.Equal(t, 1, calls)
		})
	}
}

func TestZonghengCapabilitiesRetriesHTTP503ButHonorsLongRetryAfter(t *testing.T) {
	calls := 0
	setZonghengTestClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return zonghengTestResponse(http.StatusServiceUnavailable, "temporary"), nil
		}
		return zonghengTestResponse(http.StatusOK, `{"data":[]}`), nil
	})
	_, err := FetchZonghengVideoModels(context.Background(), "https://http-retry.example", "key", "")
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	calls = 0
	setZonghengTestClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		response := zonghengTestResponse(http.StatusTooManyRequests, "rate limited")
		response.Header.Set("Retry-After", "3600")
		return response, nil
	})
	_, err = FetchZonghengVideoModels(context.Background(), "https://rate-limited.example", "key", "")
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestZonghengCapabilitiesCacheIsCredentialAndOriginScoped(t *testing.T) {
	calls := 0
	setZonghengTestClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		name := req.URL.Host + "/" + strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		return zonghengTestResponse(http.StatusOK, fmt.Sprintf(`{"data":[{"id":%q,"capabilities":{"capability_type":"video"}}]}`, name)), nil
	})
	ctx := context.Background()
	first, err := GetZonghengVideoModels(ctx, "https://cache-scope.example", "key-a", "")
	require.NoError(t, err)
	cached, err := GetZonghengVideoModels(ctx, "https://cache-scope.example/v1/", "key-a", "")
	require.NoError(t, err)
	require.Equal(t, first, cached)
	require.Equal(t, 1, calls)
	otherKey, err := GetZonghengVideoModels(ctx, "https://cache-scope.example", "key-b", "")
	require.NoError(t, err)
	otherOrigin, err := GetZonghengVideoModels(ctx, "https://other-origin.example", "key-a", "")
	require.NoError(t, err)
	require.NotEqual(t, first[0].ID, otherKey[0].ID)
	require.NotEqual(t, first[0].ID, otherOrigin[0].ID)
	require.Equal(t, 3, calls)
}

func TestZonghengCapabilitiesNeverFallsBackToExpiredCache(t *testing.T) {
	calls := 0
	available := true
	setZonghengTestClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		if !available {
			return nil, io.EOF
		}
		return zonghengTestResponse(http.StatusOK, `{"data":[{"id":"video","capabilities":{"capability_type":"video"}}]}`), nil
	})
	ctx := context.Background()
	base := "https://expired-cache.example"
	_, err := GetZonghengVideoModels(ctx, base, "key", "")
	require.NoError(t, err)
	cacheKey := zonghengCapabilitiesKey(base, "key", "")
	zonghengCapabilitiesCache.Lock()
	entry := zonghengCapabilitiesCache.entries[cacheKey]
	entry.expiresAt = time.Now().Add(-time.Second)
	zonghengCapabilitiesCache.entries[cacheKey] = entry
	zonghengCapabilitiesCache.Unlock()
	available = false
	models, err := GetZonghengVideoModels(ctx, base, "key", "")
	require.True(t, errors.Is(err, io.EOF))
	require.Nil(t, models)
	require.Equal(t, 4, calls)
}

func TestZonghengCapabilitiesCancellationStopsRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	setZonghengTestClient(t, func(*http.Request) (*http.Response, error) { calls++; cancel(); return nil, io.EOF })
	_, err := FetchZonghengVideoModels(ctx, "https://canceled.example", "key", "")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
}

func TestZonghengCapabilityFailureHasActionablePublicError(t *testing.T) {
	result := TranslateVideoTaskError(`Get "https://provider.example/v1/models?include_capabilities=1": EOF`, "upstream_capabilities_unavailable", http.StatusBadGateway)
	require.Equal(t, "upstream_service", result.Category)
	require.True(t, result.Retryable)
	require.Contains(t, result.Message, "读取上游模型能力失败")
	require.NotContains(t, result.Message, "provider.example")
}

func TestZonghengModelDiscoveryFiltersOtherCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		require.Equal(t, "1", r.URL.Query().Get("include_capabilities"))
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		fmt.Fprint(w, `{"data":[{"id":"精确视频:ID","capabilities":{"capability_type":"video","max_image_refs":0,"max_video_refs":null}},{"id":"image","capabilities":{"capability_type":"image"}},{"id":"chat","capabilities":{"capability_type":"chat"}},{"id":"audio","capabilities":{"capability_type":"audio"}}]}`)
	}))
	defer server.Close()
	models, err := FetchZonghengVideoModels(context.Background(), server.URL+"/v1/", "sk-test", "")
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "精确视频:ID", models[0].ID)
	require.NotNil(t, models[0].Capabilities.MaxImages)
	require.Zero(t, *models[0].Capabilities.MaxImages)
	require.Nil(t, models[0].Capabilities.MaxVideos)
}

func TestZonghengPollDelayPersistsAndSkipsEarlyFetch(t *testing.T) {
	task := &model.Task{TaskID: "task_local", Data: []byte(`{"next_poll_seconds":60}`)}
	SetZonghengNextPoll(task, time.Now().Unix())
	require.Greater(t, task.PrivateData.NextPollAt, time.Now().Unix()+50)
	adaptor := &videoPollingHTTPErrorAdaptor{}
	require.NoError(t, updateVideoSingleTask(context.Background(), adaptor, &model.Channel{Type: constant.ChannelTypeZongheng}, task.TaskID, map[string]*model.Task{task.TaskID: task}))
	require.False(t, adaptor.parseCalled)
	require.False(t, adaptor.fetchCalled)
	before := task.Snapshot()
	SetZonghengNextPoll(task, time.Now().Unix()+1)
	require.False(t, before.Equal(task.Snapshot()))
}
