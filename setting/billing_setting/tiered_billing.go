package billing_setting

import (
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
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
	maxBillingModels              = 10000
	maxBillingExpressionBytes     = 8 * 1024 * 1024
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

var billingSettingMu sync.RWMutex

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
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
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	if mode, ok := billingSetting.VideoBillingMode[model]; ok {
		return NormalizeVideoBillingMode(mode)
	}
	return VideoBillingModeDynamic
}

func GetVideoResolutionPrices(model string) (map[string]float64, bool) {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
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
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	return lo.Assign(billingSetting.BillingExpr)
}

func GetVideoBillingModeCopy() map[string]string {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	return lo.Assign(billingSetting.VideoBillingMode)
}

func GetVideoResolutionPricesCopy() map[string]map[string]float64 {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
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
	if strings.TrimSpace(exprStr) == "" {
		return fmt.Errorf("billing expression is empty")
	}
	if len(exprStr) > 16*1024 {
		return fmt.Errorf("billing expression exceeds 16 KiB")
	}
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
		{P: 1000000, C: 1000000, Len: 4000000, CR: 500000, CC: 250000, CC1h: 250000, Img: 100000, ImgO: 100000, AI: 100000, AO: 100000},
		{P: 1000000000, C: 1000000000, Len: 1000000000},
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

// ParseAndValidateBillingConfig validates the complete mode/expression pair.
// Callers must save and apply the returned maps together.
func ParseAndValidateBillingConfig(modeJSON, exprJSON string) (map[string]string, map[string]string, error) {
	modes := make(map[string]string)
	expressions := make(map[string]string)
	if err := common.UnmarshalJsonStr(modeJSON, &modes); err != nil {
		return nil, nil, fmt.Errorf("invalid billing mode JSON: %w", err)
	}
	if err := common.UnmarshalJsonStr(exprJSON, &expressions); err != nil {
		return nil, nil, fmt.Errorf("invalid billing expression JSON: %w", err)
	}
	if modes == nil || expressions == nil {
		return nil, nil, fmt.Errorf("billing mode and expression must both be JSON objects")
	}
	if len(modes) > maxBillingModels || len(expressions) > maxBillingModels {
		return nil, nil, fmt.Errorf("billing configuration exceeds %d models", maxBillingModels)
	}

	for model, mode := range modes {
		trimmedModel := strings.TrimSpace(model)
		if trimmedModel == "" {
			return nil, nil, fmt.Errorf("billing mode contains an empty model name")
		}
		if trimmedModel != model {
			return nil, nil, fmt.Errorf("billing mode model name %q contains surrounding whitespace", model)
		}
		switch mode {
		case BillingModeRatio:
		case BillingModeTieredExpr:
			if strings.TrimSpace(expressions[model]) == "" {
				return nil, nil, fmt.Errorf("model %s uses tiered_expr but has no billing expression", model)
			}
		default:
			return nil, nil, fmt.Errorf("model %s has invalid billing mode %q", model, mode)
		}
	}

	totalExpressionBytes := 0
	for model, expression := range expressions {
		trimmedModel := strings.TrimSpace(model)
		if trimmedModel == "" {
			return nil, nil, fmt.Errorf("billing expressions contain an empty model name")
		}
		if trimmedModel != model {
			return nil, nil, fmt.Errorf("billing expression model name %q contains surrounding whitespace", model)
		}
		totalExpressionBytes += len(expression)
		if totalExpressionBytes > maxBillingExpressionBytes {
			return nil, nil, fmt.Errorf("billing expressions exceed %d bytes in total", maxBillingExpressionBytes)
		}
		if err := smokeTestExpr(expression); err != nil {
			return nil, nil, fmt.Errorf("model %s billing expression is invalid: %w", model, err)
		}
	}
	return modes, expressions, nil
}

func ApplyBillingConfig(modes, expressions map[string]string) {
	billingSettingMu.Lock()
	billingSetting.BillingMode = lo.Assign(modes)
	billingSetting.BillingExpr = lo.Assign(expressions)
	billingSettingMu.Unlock()
	billingexpr.InvalidateCache()
}

// ApplyAuxiliaryConfig replaces one non-tiered billing map after parsing it.
// Tiered mode and expression must be applied together through ApplyBillingConfig.
func ApplyAuxiliaryConfig(key, value string) error {
	switch key {
	case VideoBillingModeField:
		modes := make(map[string]string)
		if err := common.UnmarshalJsonStr(value, &modes); err != nil {
			return fmt.Errorf("invalid video billing mode JSON: %w", err)
		}
		if modes == nil {
			return fmt.Errorf("video billing mode must be a JSON object")
		}
		billingSettingMu.Lock()
		billingSetting.VideoBillingMode = modes
		billingSettingMu.Unlock()
		return nil
	case VideoResolutionPricesField:
		prices := make(map[string]map[string]float64)
		if err := common.UnmarshalJsonStr(value, &prices); err != nil {
			return fmt.Errorf("invalid video resolution prices JSON: %w", err)
		}
		if prices == nil {
			return fmt.Errorf("video resolution prices must be a JSON object")
		}
		billingSettingMu.Lock()
		billingSetting.VideoResolutionPrices = prices
		billingSettingMu.Unlock()
		return nil
	default:
		return fmt.Errorf("unsupported billing setting field %q", key)
	}
}
