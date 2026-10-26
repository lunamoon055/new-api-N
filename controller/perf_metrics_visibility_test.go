package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/stretchr/testify/require"
)

func TestSuccessRateVisibilityOmitsAllRatesAndPreservesOtherMetrics(t *testing.T) {
	cfg := config.GlobalConfig.Get("perf_metrics_setting").(*perf_metrics_setting.PerfMetricsSetting)
	previous := *cfg
	t.Cleanup(func() { *cfg = previous })
	series := []perfmetrics.BucketPoint{{Ts: 1000, AvgLatencyMs: 150000, SuccessRate: 0, AvgTps: 12}}
	result := perfmetrics.QueryResult{
		ModelName: "video-test", SeriesSchema: "task",
		Groups:  []perfmetrics.GroupResult{{Group: "default", AvgLatencyMs: 150000, SuccessRate: 0, Series: series}},
		Summary: &perfmetrics.MetricSummary{AvgLatencyMs: 150000, SuccessRate: 0}, SummarySeries: series,
	}
	summary := perfmetrics.SummaryAllResult{Models: []perfmetrics.ModelSummary{{ModelName: "video-test", AvgLatencyMs: 150000, SuccessRate: 0, AvgTps: 12, RequestCount: 10}}}
	for _, visible := range []bool{false, true} {
		cfg.ShowSuccessRate = visible
		for _, data := range []any{perfQueryForDisplay(result), perfSummaryForDisplay(summary)} {
			encoded, err := common.Marshal(data)
			require.NoError(t, err)
			if visible {
				require.Contains(t, string(encoded), `"success_rate":0`)
			} else {
				require.NotContains(t, string(encoded), `"success_rate"`)
			}
			require.Contains(t, string(encoded), `"avg_latency_ms":150000`)
			require.Contains(t, string(encoded), `"avg_tps":12`)
			require.NotContains(t, string(encoded), `"request_count"`)
		}
	}
	cfg.ShowSuccessRate = false
	encoded, err := common.Marshal(perfQueryForDisplay(perfmetrics.QueryResult{Groups: []perfmetrics.GroupResult{}}))
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"summary"`)
	require.NotContains(t, string(encoded), `"summary_series"`)
}
