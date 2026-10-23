package service

import (
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// DownstreamModelPricing adds explicit units to the existing public price
// catalog. Amounts are base rates; group and request multipliers still apply.
type DownstreamModelPricing struct {
	model.Pricing
	PriceConfigured    bool     `json:"price_configured"`
	PriceUnit          string   `json:"price_unit"`
	InputPrice         *float64 `json:"input_price,omitempty"`
	OutputPrice        *float64 `json:"output_price,omitempty"`
	ExpressionCurrency string   `json:"expression_currency,omitempty"`
}

func hasVideoPricingEndpoint(item model.Pricing) bool {
	for _, endpoint := range item.MappedEndpoints {
		if endpoint.Kind == "video" {
			return true
		}
	}
	return slices.Contains(item.SupportedEndpointTypes, constant.EndpointTypeOpenAIVideo)
}

// BuildDownstreamModelPricing reads the same configuration as relay billing,
// never publishing the self-use fallback ratio as a configured sale price.
// exchangeRate converts USD amounts to the requested response currency. Billing
// expression coefficients and dimensionless ratios are always left unchanged.
func BuildDownstreamModelPricing(item model.Pricing, exchangeRate float64) DownstreamModelPricing {
	result := DownstreamModelPricing{Pricing: item, PriceUnit: "unconfigured"}
	// Read rates afresh rather than treating the UI catalog's fallback as a price.
	result.ModelPrice, result.ModelRatio, result.CompletionRatio = 0, 0, 0
	result.BillingMode = billing_setting.GetBillingMode(item.ModelName)
	result.BillingExpr = ""
	result.VideoBillingMode = ""
	result.VideoResolutionPrices = nil
	if result.BillingMode == billing_setting.BillingModeTieredExpr {
		if expr, ok := billing_setting.GetBillingExpr(item.ModelName); ok && strings.TrimSpace(expr) != "" {
			result.PriceConfigured = true
			result.PriceUnit = "expression"
			result.BillingExpr = expr
			result.ExpressionCurrency = "USD"
		}
		return result
	}

	video := hasVideoPricingEndpoint(item)
	mode := billing_setting.GetVideoBillingMode(item.ModelName)
	if video {
		result.VideoBillingMode = mode
	}
	if billing_setting.IsVideoResolutionTierMode(mode) {
		result.VideoBillingMode = mode
		if prices, ok := billing_setting.GetVideoResolutionPrices(item.ModelName); ok {
			result.VideoResolutionPrices = make(map[string]float64, len(prices))
			for resolution, price := range prices {
				result.VideoResolutionPrices[resolution] = price * exchangeRate
			}
			result.PriceConfigured = true
			result.QuotaType = 1
			result.PriceUnit = "request"
			if mode == billing_setting.VideoBillingModeTieredSeconds {
				result.PriceUnit = "second"
			}
		}
		return result
	}
	if price, ok := ratio_setting.GetModelPrice(item.ModelName, false); ok {
		result.PriceConfigured = true
		result.QuotaType = 1
		result.ModelPrice = price * exchangeRate
		result.PriceUnit = "request"
		if video && mode != billing_setting.VideoBillingModeFixed && !common.StringsContains(constant.TaskPricePatches, item.ModelName) {
			result.PriceUnit = "second"
		}
		return result
	}
	if ratio_setting.HasConfiguredModelRatio(item.ModelName) {
		ratio, _, _ := ratio_setting.GetModelRatio(item.ModelName)
		result.PriceConfigured = true
		result.QuotaType = 0
		result.ModelRatio = ratio
		result.CompletionRatio = ratio_setting.GetCompletionRatio(item.ModelName)
		// Media adaptors can apply their own duration/resolution multipliers to a
		// ratio-based task. Preserve that contract rather than calling it tokens.
		media := video
		for _, endpoint := range item.MappedEndpoints {
			media = media || endpoint.Kind == "image"
		}
		if media {
			result.PriceUnit = "request_multiplier"
			return result
		}
		result.PriceUnit = "million_tokens"
		input := ratio * 1_000_000 / common.QuotaPerUnit * exchangeRate
		output := input * result.CompletionRatio
		result.InputPrice, result.OutputPrice = &input, &output
	}
	return result
}
