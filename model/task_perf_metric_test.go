package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestVideoTaskPerfMetricsUseTerminalOutcomesAndPublicNames(t *testing.T) {
	truncateTables(t)
	tasks := []Task{
		{TaskID: "success", Platform: "openai", Group: "vip", Status: TaskStatusSuccess, SubmitTime: 1000, StartTime: 1050, FinishTime: 1120, Properties: Properties{OriginModelName: "Seedance2.0-fast2-c", UpstreamModelName: "sd-2-c5"}},
		{TaskID: "failure", Platform: "openai", Group: "vip", Status: TaskStatusFailure, SubmitTime: 1000, FinishTime: 1180, Properties: Properties{OriginModelName: "Seedance2.0-fast2-c"}},
		{TaskID: "legacy-name", Platform: "openai", Group: "default", Status: TaskStatusSuccess, SubmitTime: 900, FinishTime: 1200, Properties: Properties{UpstreamModelName: "legacy-video"}},
		{TaskID: "outside", Platform: "openai", Group: "vip", Status: TaskStatusSuccess, SubmitTime: 900, FinishTime: 999, Properties: Properties{OriginModelName: "Seedance2.0-fast2-c"}},
	}
	for _, status := range []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress, TaskStatusUnknown, TaskStatusNotStart} {
		tasks = append(tasks, Task{TaskID: string(status), Platform: "openai", Group: "vip", Status: status, SubmitTime: 1000, FinishTime: 1100, Properties: Properties{OriginModelName: "Seedance2.0-fast2-c"}})
	}
	for _, source := range []struct {
		platform constant.TaskPlatform
		action   string
	}{
		{constant.TaskPlatformSuno, ""}, {constant.TaskPlatformMidjourney, ""}, {"openai", constant.TaskActionImageGenerate},
	} {
		tasks = append(tasks, Task{TaskID: string(source.platform) + source.action, Platform: source.platform, Action: source.action, Group: "vip", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1100, Properties: Properties{OriginModelName: "Seedance2.0-fast2-c"}})
	}
	require.NoError(t, DB.Create(&tasks).Error)

	for iteration := 0; iteration < 2; iteration++ {
		metrics, err := GetVideoTaskPerfMetrics("Seedance2.0-fast2-c", "vip", nil, 1000, 1200, 60)
		require.NoError(t, err)
		var requests, successes, latency int64
		for _, metric := range metrics {
			require.Equal(t, "Seedance2.0-fast2-c", metric.ModelName)
			require.Equal(t, "vip", metric.Group)
			requests += metric.RequestCount
			successes += metric.SuccessCount
			latency += metric.TotalLatencyMs
			require.Zero(t, metric.TtftCount)
			require.Zero(t, metric.OutputTokens)
		}
		require.EqualValues(t, 2, requests)
		require.EqualValues(t, 1, successes)
		require.EqualValues(t, 300000, latency)
	}
	upstream, err := GetVideoTaskPerfMetrics("sd-2-c5", "", nil, 1000, 1200, 60)
	require.NoError(t, err)
	require.Empty(t, upstream)
	legacy, err := GetVideoTaskPerfMetrics("legacy-video", "", nil, 1000, 1200, 60)
	require.NoError(t, err)
	require.Len(t, legacy, 1)
	require.EqualValues(t, 300000, legacy[0].TotalLatencyMs)
	var storedSamples int64
	require.NoError(t, DB.Model(&PerfMetric{}).Count(&storedSamples).Error)
	require.Zero(t, storedSamples)
}

func TestVideoTaskPerfMetricsGroupFiltersAndTimestampFallbacks(t *testing.T) {
	truncateTables(t)
	tasks := []Task{
		{TaskID: "default", Platform: "openai", Group: "", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1100, Properties: Properties{OriginModelName: "video"}},
		{TaskID: "vip", Platform: "openai", Group: "vip", Status: TaskStatusFailure, SubmitTime: 1000, UpdatedAt: 1150, Properties: Properties{OriginModelName: "video"}},
		{TaskID: "inactive", Platform: "openai", Group: "deleted-group", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1100, Properties: Properties{OriginModelName: "video"}},
		{TaskID: "start-fallback", Platform: "openai", Group: "vip", Status: TaskStatusSuccess, StartTime: 1000, FinishTime: 1200, Properties: Properties{OriginModelName: "video"}},
	}
	require.NoError(t, DB.Create(&tasks).Error)
	metrics, err := GetVideoTaskPerfMetrics("video", "default", nil, 1000, 1200, 3600)
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	require.EqualValues(t, 1, metrics[0].RequestCount)
	require.Equal(t, "default", metrics[0].Group)
	metrics, err = GetVideoTaskPerfMetrics("video", "", []string{"default", "vip"}, 1000, 1200, 3600)
	require.NoError(t, err)
	require.Len(t, metrics, 2)
	require.EqualValues(t, 2, metrics[1].RequestCount)
	require.EqualValues(t, 350000, metrics[1].TotalLatencyMs)
	metrics, err = GetVideoTaskPerfMetrics("video", "", []string{}, 1000, 1200, 3600)
	require.NoError(t, err)
	require.Empty(t, metrics)
	metrics, err = GetVideoTaskPerfMetrics("video", "", nil, 1201, 1300, 3600)
	require.NoError(t, err)
	require.Empty(t, metrics)
}

func TestVideoTaskPerfMetricsReadAllBatches(t *testing.T) {
	truncateTables(t)
	tasks := make([]Task, 501)
	for index := range tasks {
		tasks[index] = Task{Platform: "openai", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1100, Properties: Properties{OriginModelName: "batched-video"}}
	}
	require.NoError(t, DB.CreateInBatches(&tasks, 50).Error)
	metrics, err := GetVideoTaskPerfMetrics("batched-video", "", nil, 1000, 1200, 3600)
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	require.EqualValues(t, 501, metrics[0].RequestCount)
}
