package perfmetrics

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoMetricsTest(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.PerfMetric{}))
	hotBuckets.Clear()
	t.Cleanup(func() {
		model.DB = previousDB
		hotBuckets.Clear()
		_ = sqlDB.Close()
	})
}

func TestVideoMetricsQueriesIncludePersistedOutcomes(t *testing.T) {
	setupVideoMetricsTest(t)
	finished := time.Now().Unix() - 60
	tasks := []model.Task{
		{Platform: "openai", Group: "vip", Status: model.TaskStatusSuccess, SubmitTime: finished - 60, FinishTime: finished, Properties: model.Properties{OriginModelName: "video-alias", UpstreamModelName: "provider-video"}},
	}
	for range 3 {
		tasks = append(tasks, model.Task{Platform: "openai", Group: "default", Status: model.TaskStatusFailure, SubmitTime: finished - 120, FinishTime: finished, Properties: model.Properties{OriginModelName: "video-alias"}})
	}
	require.NoError(t, model.DB.Create(&tasks).Error)
	for range 2 {
		result, err := Query(QueryParams{Model: "video-alias", Hours: 24})
		require.NoError(t, err)
		require.Len(t, result.Groups, 2)
		require.NotNil(t, result.Summary)
		require.EqualValues(t, 105000, result.Summary.AvgLatencyMs)
		require.Equal(t, 25.0, result.Summary.SuccessRate)
		require.Zero(t, result.Summary.AvgTps)
		require.Len(t, result.SummarySeries, 1)
		require.Equal(t, 25.0, result.SummarySeries[0].SuccessRate)
		require.EqualValues(t, 105000, result.SummarySeries[0].AvgLatencyMs)
		for _, group := range result.Groups {
			require.Zero(t, group.AvgTtftMs)
			require.NotEmpty(t, group.Series)
		}
		// Group filtering must also constrain the weighted overview.
		visible := SummarizeGroups(result.Groups[1:])
		require.Equal(t, 100.0, visible.SuccessRate)
		require.EqualValues(t, 60000, visible.AvgLatencyMs)
		visibleSeries := SummarizeGroupSeries(result.Groups[1:])
		require.Len(t, visibleSeries, 1)
		require.Equal(t, 100.0, visibleSeries[0].SuccessRate)
		serialized, err := common.Marshal(result)
		require.NoError(t, err)
		require.NotContains(t, string(serialized), "request_count")
		require.NotContains(t, string(serialized), "provider-video")
	}
	summary, err := QuerySummaryAll(24, nil)
	require.NoError(t, err)
	require.Len(t, summary.Models, 1)
	require.Equal(t, "video-alias", summary.Models[0].ModelName)
	require.Equal(t, 25.0, summary.Models[0].SuccessRate)
	require.EqualValues(t, 105000, summary.Models[0].AvgLatencyMs)
	require.EqualValues(t, 4, summary.Models[0].RequestCount)
}

func TestTextMetricsContinueUsingRelaySamples(t *testing.T) {
	setupVideoMetricsTest(t)
	Record(Sample{Model: "text-model", Group: "vip", Success: true, LatencyMs: 2000, HasTtft: true, TtftMs: 300, OutputTokens: 100, GenerationMs: 1000})
	result, err := Query(QueryParams{Model: "text-model", Hours: 24})
	require.NoError(t, err)
	require.Len(t, result.Groups, 1)
	require.EqualValues(t, 2000, result.Groups[0].AvgLatencyMs)
	require.EqualValues(t, 300, result.Groups[0].AvgTtftMs)
	require.Equal(t, 100.0, result.Groups[0].AvgTps)
	require.Equal(t, 100.0, result.Groups[0].SuccessRate)
}
