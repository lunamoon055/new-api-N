package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

const VideoPricingConfigOptionKey = "billing_setting.video_pricing_config"

func isVideoPricingOption(key string) bool {
	return key == "billing_setting.video_billing_mode" || key == "billing_setting.video_resolution_prices" || key == "billing_setting.video_reference_prices"
}

// Database refreshes must publish the same complete snapshot as admin saves.
// Otherwise another node's refresh can briefly mix an old generation rate with
// a new reference rate even though the database write itself was atomic.
func applyVideoPricingOptionsFromSnapshot(values map[string]string) error {
	raw := map[string]any{}
	common.OptionMapRWMutex.RLock()
	for _, field := range []string{"video_billing_mode", "video_resolution_prices", "video_reference_prices"} {
		key := "billing_setting." + field
		value, ok := values[key]
		if !ok {
			value = common.OptionMap[key]
		}
		if strings.TrimSpace(value) == "" {
			value = "{}"
		}
		var decoded any
		if err := common.UnmarshalJsonStr(value, &decoded); err != nil {
			common.OptionMapRWMutex.RUnlock()
			return err
		}
		raw[field] = decoded
	}
	common.OptionMapRWMutex.RUnlock()
	encoded, err := common.Marshal(raw)
	if err != nil {
		return err
	}
	cfg, err := billing_setting.ParseVideoPricingConfig(string(encoded))
	if err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	for field, value := range raw {
		encoded, _ := common.Marshal(value)
		common.OptionMap["billing_setting."+field] = string(encoded)
	}
	common.OptionMapRWMutex.Unlock()
	billing_setting.ApplyVideoPricingConfig(cfg)
	InvalidatePricingCache()
	ratio_setting.InvalidateExposedDataCache()
	return nil
}

// Prices are keyed by public model ID, independently of channel registration or
// mappings. Administrators may configure prices before adding a channel.
func UpdateVideoPricingConfig(raw string) error {
	billingOptionsMu.Lock()
	defer billingOptionsMu.Unlock()
	cfg, err := billing_setting.ParseVideoPricingConfig(raw)
	if err != nil {
		return err
	}
	values := []Option{}
	for key, value := range map[string]any{"billing_setting.video_billing_mode": cfg.Modes, "billing_setting.video_resolution_prices": cfg.Generation, "billing_setting.video_reference_prices": cfg.Reference} {
		encoded, err := common.Marshal(value)
		if err != nil {
			return err
		}
		values = append(values, Option{Key: key, Value: string(encoded)})
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		for _, option := range values {
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	for _, option := range values {
		common.OptionMap[option.Key] = option.Value
	}
	common.OptionMapRWMutex.Unlock()
	billing_setting.ApplyVideoPricingConfig(cfg)
	InvalidatePricingCache()
	ratio_setting.InvalidateExposedDataCache()
	return nil
}
