package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/gin-gonic/gin"
)

const successRateVisibilityOption = "perf_metrics_setting.show_success_rate"

func successRateVisible() bool {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return perf_metrics_setting.GetSetting().ShowSuccessRate
}

func GetSuccessRateVisibility(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"show_success_rate": successRateVisible()}})
}

func UpdateSuccessRateVisibility(c *gin.Context) {
	if c.GetInt("role") < common.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Administrator access required"})
		return
	}
	var request struct {
		ShowSuccessRate *bool `json:"show_success_rate"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.ShowSuccessRate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "show_success_rate must be a boolean"})
		return
	}
	if err := model.UpdateOption(successRateVisibilityOption, strconv.FormatBool(*request.ShowSuccessRate)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	GetSuccessRateVisibility(c)
}

// Shadow success_rate in public responses while keeping collection and aggregation intact.
type perfBucketWithoutSuccessRate struct {
	perfmetrics.BucketPoint
	SuccessRate *float64 `json:"success_rate,omitempty"`
}

type perfGroupWithoutSuccessRate struct {
	perfmetrics.GroupResult
	SuccessRate *float64                       `json:"success_rate,omitempty"`
	Series      []perfBucketWithoutSuccessRate `json:"series"`
}

type perfMetricWithoutSuccessRate struct {
	perfmetrics.MetricSummary
	SuccessRate *float64 `json:"success_rate,omitempty"`
}

type perfModelWithoutSuccessRate struct {
	perfmetrics.ModelSummary
	SuccessRate *float64 `json:"success_rate,omitempty"`
}

func perfSeriesForDisplay(series []perfmetrics.BucketPoint) []perfBucketWithoutSuccessRate {
	if series == nil {
		return nil
	}
	result := make([]perfBucketWithoutSuccessRate, len(series))
	for i, point := range series {
		result[i] = perfBucketWithoutSuccessRate{BucketPoint: point}
	}
	return result
}

func perfQueryForDisplay(result perfmetrics.QueryResult) any {
	if successRateVisible() {
		return result
	}
	groups := make([]perfGroupWithoutSuccessRate, len(result.Groups))
	for i, group := range result.Groups {
		groups[i] = perfGroupWithoutSuccessRate{GroupResult: group, Series: perfSeriesForDisplay(group.Series)}
	}
	var summary *perfMetricWithoutSuccessRate
	if result.Summary != nil {
		summary = &perfMetricWithoutSuccessRate{MetricSummary: *result.Summary}
	}
	return struct {
		perfmetrics.QueryResult
		Groups        []perfGroupWithoutSuccessRate  `json:"groups"`
		Summary       *perfMetricWithoutSuccessRate  `json:"summary,omitempty"`
		SummarySeries []perfBucketWithoutSuccessRate `json:"summary_series,omitempty"`
	}{QueryResult: result, Groups: groups, Summary: summary, SummarySeries: perfSeriesForDisplay(result.SummarySeries)}
}

func perfSummaryForDisplay(result perfmetrics.SummaryAllResult) any {
	if successRateVisible() {
		return result
	}
	models := make([]perfModelWithoutSuccessRate, len(result.Models))
	for i, metric := range result.Models {
		models[i] = perfModelWithoutSuccessRate{ModelSummary: metric}
	}
	return struct {
		Models []perfModelWithoutSuccessRate `json:"models"`
	}{Models: models}
}
