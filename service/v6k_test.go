package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestV6KDiscoveryIntersectsAuthorizedModelsWithVideoCatalog(t *testing.T) {
	InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			require.Equal(t, "Bearer key", r.Header.Get("Authorization"))
			fmt.Fprint(w, `{"data":[{"id":"TTseadanco 2.0 Fast/480"},{"id":"视频:A"},{"id":"chat"},{"id":"视频:A"}]}`)
		case "/api/pricing":
			require.Empty(t, r.Header.Get("Authorization"))
			fmt.Fprint(w, `{"success":true,"data":[{"model_name":"视频:A","supported_endpoint_types":["openai-video"]},{"model_name":"TTseadanco 2.0 Fast/480","supported_endpoint_types":["openai-video"]},{"model_name":"chat","supported_endpoint_types":["openai"]},{"model_name":"not-authorized","supported_endpoint_types":["openai-video"]}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	ids, err := FetchV6KVideoModels(context.Background(), server.URL+"/v1/", "key", "")
	require.NoError(t, err)
	require.Equal(t, []string{"TTseadanco 2.0 Fast/480", "视频:A"}, ids)
}

func TestV6KDiscoveryDoesNotUseSnapshotAfterAuthFailure(t *testing.T) {
	InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	ids, err := FetchV6KVideoModels(context.Background(), server.URL, "bad", "")
	require.Error(t, err)
	require.Nil(t, ids)
}

func TestV6KPollScheduleAndPublishedHints(t *testing.T) {
	task := &model.Task{}
	SetV6KNextPoll(task, 100)
	require.EqualValues(t, 120, task.PrivateData.NextPollAt)
	now := time.Now().Unix()
	scheduleV6KPollRetry(task, "100")
	require.GreaterOrEqual(t, task.PrivateData.NextPollAt, now+100)
	require.Len(t, common.V6KVideoModelIDs(), 34)
	require.Len(t, common.GetV6KVideoParameters("JJseadanco2.5").Durations, 26)
	require.Equal(t, []int{15}, common.GetV6KVideoParameters("dola2.0").Durations)
	require.Equal(t, []int{5, 10, 15}, common.GetV6KVideoParameters("dola2.5").Durations)
	require.Empty(t, common.GetV6KVideoParameters("XXsedanco2.0S").Durations) // unpublished; do not infer from the name
	require.Equal(t, "seconds", common.GetV6KVideoParameters("new-model").DurationField)
}

func TestV6KPollingHTTPErrorPreservesTaskAndBacksOff(t *testing.T) {
	task := &model.Task{TaskID: "task_public", Status: model.TaskStatusQueued, Quota: 100, PrivateData: model.TaskPrivateData{UpstreamTaskID: "provider-id"}}
	adaptor := &videoPollingHTTPErrorAdaptor{}
	channel := &model.Channel{Type: constant.ChannelTypeV6K, Key: "stored-key"}
	now := time.Now().Unix()
	err := updateVideoSingleTask(context.Background(), adaptor, channel, "provider-id", map[string]*model.Task{"provider-id": task})
	require.Error(t, err)
	require.Equal(t, model.TaskStatus(model.TaskStatusQueued), task.Status)
	require.Equal(t, 100, task.Quota)
	require.GreaterOrEqual(t, task.PrivateData.NextPollAt, now+40)
	adaptor.fetchCalled = false
	require.NoError(t, updateVideoSingleTask(context.Background(), adaptor, channel, "provider-id", map[string]*model.Task{"provider-id": task}))
	require.False(t, adaptor.fetchCalled)
	options := taskVideoMediaDownloadOptions(constant.ChannelTypeV6K, "https://www.v6k.top", "stored-key", "")
	require.Equal(t, "https://www.v6k.top", options.CredentialOrigin)
	require.Equal(t, "Bearer stored-key", options.AuthValue)
}
