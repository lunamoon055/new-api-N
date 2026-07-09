package billing_setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio              = "ratio"
	BillingModeTieredExpr         = "tiered_expr"
	BillingModeField              = "billing_mode"
	BillingExprField              = "billing_expr"
	VideoBillingModeField         = "video_billing_mode"
	VideoResolutionPricesField    = "video_resolution_prices"
	VideoBillingModeDynamic       = "dynamic"
	VideoBillingModeFixed         = "fixed"
	VideoBillingModeTieredSeconds = "tiered_seconds"
	VideoBillingModeTieredRequest = "tiered_request"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr, billing_setting.video_billing_mode, billing_setting.video_resolution_prices
type BillingSetting struct {
	BillingMode           map[string]string             `json:"billing_mode"`
	BillingExpr           map[string]string             `json:"billing_expr"`
	VideoBillingMode      map[string]string             `json:"video_billing_mode"`
	VideoResolutionPrices map[string]map[string]float64 `json:"video_resolution_prices"`
}

var billingSetting = BillingSetting{
	BillingMode:           make(map[string]string),
	BillingExpr:           make(map[string]string),
	VideoBillingMode:      make(map[string]string),
	VideoResolutionPrices: make(map[string]map[string]float64),
}

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func NormalizeVideoBillingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case VideoBillingModeFixed:
		return VideoBillingModeFixed
	case VideoBillingModeTieredSeconds:
		return VideoBillingModeTieredSeconds
	case VideoBillingModeTieredRequest:
		return VideoBillingModeTieredRequest
	default:
		return VideoBillingModeDynamic
	}
}

func IsVideoResolutionTierMode(mode string) bool {
	switch NormalizeVideoBillingMode(mode) {
	case VideoBillingModeTieredSeconds, VideoBillingModeTieredRequest:
		return true
	default:
		return false
	}
}

func NormalizeVideoResolution(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	switch normalized {
	case "480", "480p":
		return "480p"
	case "720", "720p":
		return "720p"
	case "1080", "1080p":
		return "1080p"
	case "4k", "2160", "2160p":
		return "4k"
	default:
		return ""
	}
}

func GetVideoBillingMode(model string) string {
	if mode, ok := billingSetting.VideoBillingMode[model]; ok {
		return NormalizeVideoBillingMode(mode)
	}
	return VideoBillingModeDynamic
}

func GetVideoResolutionPrices(model string) (map[string]float64, bool) {
	prices, ok := billingSetting.VideoResolutionPrices[model]
	if !ok || len(prices) == 0 {
		return nil, false
	}
	copied := make(map[string]float64, len(prices))
	for resolution, price := range prices {
		normalizedResolution := NormalizeVideoResolution(resolution)
		if normalizedResolution == "" {
			continue
		}
		copied[normalizedResolution] = price
	}
	return copied, len(copied) > 0
}

func GetBillingExpr(model string) (string, bool) {
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	return lo.Assign(billingSetting.BillingExpr)
}

func GetVideoBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.VideoBillingMode)
}

func GetVideoResolutionPricesCopy() map[string]map[string]float64 {
	copied := make(map[string]map[string]float64, len(billingSetting.VideoResolutionPrices))
	for model, prices := range billingSetting.VideoResolutionPrices {
		modelPrices := make(map[string]float64, len(prices))
		for resolution, price := range prices {
			modelPrices[resolution] = price
		}
		copied[model] = modelPrices
	}
	return copied
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 4)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	if modes := GetVideoBillingModeCopy(); len(modes) > 0 {
		extra[VideoBillingModeField] = modes
	}
	if prices := GetVideoResolutionPricesCopy(); len(prices) > 0 {
		extra[VideoResolutionPricesField] = prices
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}
