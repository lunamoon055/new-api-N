package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPerfMetricsVideoAPIUsesVisibleTerminalTasks(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.PerfMetric{}))
	previousGroups := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroups)) })
	finished := time.Now().Unix() - 60
	tasks := []model.Task{
		{Platform: "openai", Group: "default", Status: model.TaskStatusSuccess, SubmitTime: finished - 120, FinishTime: finished, Properties: model.Properties{OriginModelName: "Seedance2.0-fast2-c", UpstreamModelName: "sd-2-c5"}},
		{Platform: "openai", Group: "vip", Status: model.TaskStatusFailure, SubmitTime: finished - 180, FinishTime: finished, Properties: model.Properties{OriginModelName: "Seedance2.0-fast2-c"}},
		{Platform: "openai", Group: "hidden-group", Status: model.TaskStatusSuccess, SubmitTime: finished - 1, FinishTime: finished, Properties: model.Properties{OriginModelName: "Seedance2.0-fast2-c"}},
	}
	require.NoError(t, db.Create(&tasks).Error)
	router := gin.New()
	router.GET("/api/perf-metrics", GetPerfMetrics)
	router.GET("/api/perf-metrics/summary", GetPerfMetricsSummary)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/perf-metrics?model=Seedance2.0-fast2-c", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Success bool                    `json:"success"`
		Data    perfmetrics.QueryResult `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Len(t, body.Data.Groups, 2)
	require.Equal(t, 50.0, body.Data.Summary.SuccessRate)
	require.EqualValues(t, 150000, body.Data.Summary.AvgLatencyMs)
	require.Len(t, body.Data.SummarySeries, 1)
	require.Equal(t, 50.0, body.Data.SummarySeries[0].SuccessRate)
	require.NotContains(t, response.Body.String(), "hidden-group")
	require.NotContains(t, response.Body.String(), "sd-2-c5")
	require.NotContains(t, response.Body.String(), "request_count")
	t.Logf("VIDEO_METRICS_RESPONSE %s", response.Body.String())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/perf-metrics?model=Seedance2.0-fast2-c&group=hidden-group", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
	require.Empty(t, body.Data.Groups)
	// Omitted JSON fields do not clear a reused destination.
	require.NotContains(t, response.Body.String(), `"summary"`)
	require.NotContains(t, response.Body.String(), `"summary_series"`)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/perf-metrics/summary", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var summary struct {
		Success bool                         `json:"success"`
		Data    perfmetrics.SummaryAllResult `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &summary))
	require.True(t, summary.Success)
	require.Len(t, summary.Data.Models, 1)
	require.Equal(t, 50.0, summary.Data.Models[0].SuccessRate)
	require.EqualValues(t, 150000, summary.Data.Models[0].AvgLatencyMs)
}
