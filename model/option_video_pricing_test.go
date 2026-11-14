package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVideoPricingSavesAtomicallyWithoutChannelPrerequisites(t *testing.T) {
	previousDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	DB = db
	t.Cleanup(func() { DB = previousDB; sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	// No channels table exists: saving a price cannot depend on a channel,
	// its type, model list, or a mapping that may be added/removed later.
	require.NoError(t, DB.AutoMigrate(&Option{}))
	keys := []string{"billing_setting.video_billing_mode", "billing_setting.video_resolution_prices", "billing_setting.video_reference_prices"}
	var originals []Option
	require.NoError(t, DB.Where("key IN ?", keys).Find(&originals).Error)
	cache := map[string]string{}
	common.OptionMapRWMutex.RLock()
	for _, key := range keys {
		if value, ok := common.OptionMap[key]; ok {
			cache[key] = value
		}
	}
	common.OptionMapRWMutex.RUnlock()
	before := &billing_setting.VideoPricingConfig{Modes: billing_setting.GetVideoBillingModeCopy(), Generation: billing_setting.GetVideoResolutionPricesCopy(), Reference: billing_setting.GetVideoReferencePricesCopy()}
	t.Cleanup(func() {
		DB.Where("key IN ?", keys).Delete(&Option{})
		for _, row := range originals {
			DB.Save(&row)
		}
		billing_setting.ApplyVideoPricingConfig(before)
		common.OptionMapRWMutex.Lock()
		for _, key := range keys {
			delete(common.OptionMap, key)
			if value, ok := cache[key]; ok {
				common.OptionMap[key] = value
			}
		}
		common.OptionMapRWMutex.Unlock()
	})
	raw := `{"video_billing_mode":{"SD2.0":"tiered_seconds"},"video_resolution_prices":{"SD2.0":{"720p":0.1}},"video_reference_prices":{"SD2.0":{"720p":0.2}}}`
	require.NoError(t, UpdateVideoPricingConfig(raw))
	mode, gen, ref := billing_setting.GetVideoPricingSnapshot("SD2.0")
	require.Equal(t, "tiered_seconds", mode)
	require.Equal(t, 0.1, gen["720p"])
	require.Equal(t, 0.2, ref["720p"])
	require.Error(t, applyVideoPricingOptionsFromSnapshot(map[string]string{
		keys[0]: `{"SD2.0":"tiered_seconds"}`, keys[1]: `{"SD2.0":{"720p":9}}`, keys[2]: `{"SD2.0":{"1080p":8}}`,
	}))
	_, gen, ref = billing_setting.GetVideoPricingSnapshot("SD2.0")
	require.Equal(t, 0.1, gen["720p"])
	require.Equal(t, 0.2, ref["720p"])
	var saved []Option
	require.NoError(t, DB.Where("key IN ?", keys).Order("key").Find(&saved).Error)
	name := "test:reject-reference-price-update"
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if option, ok := tx.Statement.Dest.(*Option); ok && option.Key == keys[2] {
			tx.AddError(fmt.Errorf("injected price write failure"))
		}
	}))
	err = UpdateVideoPricingConfig(`{"video_billing_mode":{"SD2.0":"tiered_seconds"},"video_resolution_prices":{"SD2.0":{"720p":4}},"video_reference_prices":{"SD2.0":{"720p":5}}}`)
	DB.Callback().Update().Remove(name)
	require.Error(t, err)
	var after []Option
	require.NoError(t, DB.Where("key IN ?", keys).Order("key").Find(&after).Error)
	require.Equal(t, saved, after)
	_, gen, ref = billing_setting.GetVideoPricingSnapshot("SD2.0")
	require.Equal(t, 0.1, gen["720p"])
	require.Equal(t, 0.2, ref["720p"])
	// Full-form saves include existing models, including unbound public aliases.
	// None may block saving a different model merely because a mapping changed.
	raw = `{"video_billing_mode":{"SD2.0":"tiered_seconds","SD2.5":"tiered_seconds","Seedance2.0:(无视频参考)":"tiered_seconds","unmapped-custom-model":"tiered_seconds"},"video_resolution_prices":{"SD2.0":{"720p":0.1},"SD2.5":{"720p":0.2},"Seedance2.0:(无视频参考)":{"720p":0.3},"unmapped-custom-model":{"720p":0.4}},"video_reference_prices":{"SD2.0":{"720p":0.2},"SD2.5":{"720p":0},"Seedance2.0:(无视频参考)":{"720p":0.5},"unmapped-custom-model":{"720p":0.6}}}`
	require.NoError(t, UpdateVideoPricingConfig(raw))
	for name, price := range map[string]float64{"SD2.0": 0.2, "SD2.5": 0, "Seedance2.0:(无视频参考)": 0.5, "unmapped-custom-model": 0.6} {
		mode, _, refs := billing_setting.GetVideoPricingSnapshot(name)
		require.Equal(t, "tiered_seconds", mode)
		require.Equal(t, price, refs["720p"])
	}
	loadOptionsFromDatabase()
	_, _, ref = billing_setting.GetVideoPricingSnapshot("unmapped-custom-model")
	require.Equal(t, 0.6, ref["720p"])
	require.NoError(t, UpdateOption(keys[2], `{"unrelated-model":{"720p":0.1}}`))
	_, gen, _ = billing_setting.GetVideoPricingSnapshot("SD2.0")
	require.Equal(t, 0.1, gen["720p"], "updating a legacy reference map must preserve generation prices")
	require.Error(t, UpdateOption(keys[2], `{"SD2.0":{"720p":-1}}`))
}
