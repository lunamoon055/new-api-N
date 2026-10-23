package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

type videoMetricsPollingAdaptor struct{ status string }

func (a *videoMetricsPollingAdaptor) Init(*relaycommon.RelayInfo) {}
func (a *videoMetricsPollingAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"state":"test"}`))}, nil
}
func (a *videoMetricsPollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{TaskID: "upstream-video", Status: a.status}, nil
}
func (a *videoMetricsPollingAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return 0
}

func TestVideoPollingTerminalStateFeedsPerformanceMetrics(t *testing.T) {
	truncate(t)
	seedUser(t, 701, 0)
	task := &model.Task{
		TaskID: "metrics-video", UserId: 701, Platform: "openai", Group: "default",
		Status: model.TaskStatusQueued, SubmitTime: time.Now().Unix() - 120,
		Properties:  model.Properties{OriginModelName: "video-alias", UpstreamModelName: "upstream-video"},
		PrivateData: model.TaskPrivateData{UpstreamTaskID: "upstream-video"},
	}
	require.NoError(t, model.DB.Create(task).Error)
	for _, status := range []string{"", string(model.TaskStatusInProgress), string(model.TaskStatusSuccess), string(model.TaskStatusSuccess)} {
		require.NoError(t, updateVideoSingleTask(context.Background(), &videoMetricsPollingAdaptor{status: status},
			&model.Channel{Type: constant.ChannelTypeOpenAI, Key: "test-only"}, "upstream-video", map[string]*model.Task{"upstream-video": task}))
		metrics, err := model.GetVideoTaskPerfMetrics("video-alias", "", nil, time.Now().Unix()-3600, time.Now().Unix(), 3600)
		require.NoError(t, err)
		if status != string(model.TaskStatusSuccess) {
			require.Empty(t, metrics)
		} else {
			require.Len(t, metrics, 1)
			require.EqualValues(t, 1, metrics[0].RequestCount)
			require.EqualValues(t, 1, metrics[0].SuccessCount)
			require.GreaterOrEqual(t, metrics[0].TotalLatencyMs, int64(120000))
		}
	}
}
