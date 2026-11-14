package billing_setting

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
)

type VideoPricingConfig struct {
	Modes      map[string]string             `json:"video_billing_mode"`
	Generation map[string]map[string]float64 `json:"video_resolution_prices"`
	Reference  map[string]map[string]float64 `json:"video_reference_prices"`
}

func ParseVideoPricingConfig(raw string) (*VideoPricingConfig, error) {
	var cfg VideoPricingConfig
	if err := common.UnmarshalJsonStr(raw, &cfg); err != nil || cfg.Modes == nil || cfg.Generation == nil || cfg.Reference == nil {
		return nil, fmt.Errorf("video pricing requires all three configuration maps")
	}
	for _, mode := range cfg.Modes {
		if mode != VideoBillingModeDynamic && mode != VideoBillingModeFixed && mode != VideoBillingModeTieredSeconds && mode != VideoBillingModeTieredRequest {
			return nil, fmt.Errorf("unsupported video billing mode")
		}
	}
	for _, prices := range []map[string]map[string]float64{cfg.Generation, cfg.Reference} {
		encoded, err := common.Marshal(prices)
		if err != nil {
			return nil, err
		}
		parsed, err := ParseVideoReferencePrices(string(encoded))
		if err != nil {
			return nil, err
		}
		for name, tiers := range parsed {
			prices[name] = tiers
		}
	}
	for name, tiers := range cfg.Reference {
		if len(tiers) > 0 && cfg.Modes[name] != VideoBillingModeTieredSeconds {
			return nil, fmt.Errorf("reference-video prices require tiered_seconds: %s", name)
		}
		for resolution := range tiers {
			if _, ok := cfg.Generation[name][resolution]; !ok {
				return nil, fmt.Errorf("generation price missing for %s / %s", name, resolution)
			}
		}
	}
	return &cfg, nil
}

func ApplyVideoPricingConfig(cfg *VideoPricingConfig) {
	billingSettingMu.Lock()
	defer billingSettingMu.Unlock()
	billingSetting.VideoBillingMode = cfg.Modes
	billingSetting.VideoResolutionPrices = cfg.Generation
	billingSetting.VideoReferencePrices = cfg.Reference
}

func GetVideoPricingSnapshot(name string) (string, map[string]float64, map[string]float64) {
	billingSettingMu.RLock()
	defer billingSettingMu.RUnlock()
	gen, ref := map[string]float64{}, map[string]float64{}
	for key, value := range billingSetting.VideoResolutionPrices[name] {
		gen[NormalizeVideoResolution(key)] = value
	}
	for key, value := range billingSetting.VideoReferencePrices[name] {
		ref[NormalizeVideoResolution(key)] = value
	}
	return NormalizeVideoBillingMode(billingSetting.VideoBillingMode[name]), gen, ref
}
