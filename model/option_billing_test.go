package model

import (
	"fmt"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preserveBillingOptionState(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Option{}))

	const modeKey = "billing_setting.billing_mode"
	const exprKey = "billing_setting.billing_expr"
	var originalOptions []Option
	require.NoError(t, DB.Where("key IN ?", []string{modeKey, exprKey}).Find(&originalOptions).Error)
	common.OptionMapRWMutex.RLock()
	originalMode, hadMode := common.OptionMap[modeKey]
	originalExpr, hadExpr := common.OptionMap[exprKey]
	common.OptionMapRWMutex.RUnlock()
	originalModes := billing_setting.GetBillingModeCopy()
	originalExpressions := billing_setting.GetBillingExprCopy()

	t.Cleanup(func() {
		DB.Where("key IN ?", []string{modeKey, exprKey}).Delete(&Option{})
		for i := range originalOptions {
			DB.Save(&originalOptions[i])
		}
		common.OptionMapRWMutex.Lock()
		if hadMode {
			common.OptionMap[modeKey] = originalMode
		} else {
			delete(common.OptionMap, modeKey)
		}
		if hadExpr {
			common.OptionMap[exprKey] = originalExpr
		} else {
			delete(common.OptionMap, exprKey)
		}
		common.OptionMapRWMutex.Unlock()
		billing_setting.ApplyBillingConfig(originalModes, originalExpressions)
	})
}

func TestUpdateBillingOptionsPersistsPairAtomically(t *testing.T) {
	preserveBillingOptionState(t)

	modeValue := `{"gpt-test":"tiered_expr"}`
	exprValue := `{"gpt-test":"p * 2 + c * 8"}`
	require.NoError(t, UpdateBillingOptions(modeValue, exprValue))

	var options []Option
	require.NoError(t, DB.Where("key IN ?", []string{"billing_setting.billing_mode", "billing_setting.billing_expr"}).Find(&options).Error)
	require.Len(t, options, 2)
	values := make(map[string]string, 2)
	for _, option := range options {
		values[option.Key] = option.Value
	}
	assert.Equal(t, modeValue, values["billing_setting.billing_mode"])
	assert.Equal(t, exprValue, values["billing_setting.billing_expr"])

	common.OptionMapRWMutex.RLock()
	assert.Equal(t, modeValue, common.OptionMap["billing_setting.billing_mode"])
	assert.Equal(t, exprValue, common.OptionMap["billing_setting.billing_expr"])
	common.OptionMapRWMutex.RUnlock()
}

func TestUpdateBillingOptionsRejectsInvalidPairWithoutChangingDatabase(t *testing.T) {
	preserveBillingOptionState(t)
	const modeKey = "billing_setting.billing_mode"
	const exprKey = "billing_setting.billing_expr"
	require.NoError(t, DB.Save(&Option{Key: modeKey, Value: `{}`}).Error)
	require.NoError(t, DB.Save(&Option{Key: exprKey, Value: `{}`}).Error)

	err := UpdateBillingOptions(`{"bad":"tiered_expr"}`, `{"bad":"p / 0"}`)
	require.Error(t, err)

	var mode Option
	var expression Option
	require.NoError(t, DB.First(&mode, "key = ?", modeKey).Error)
	require.NoError(t, DB.First(&expression, "key = ?", exprKey).Error)
	assert.Equal(t, `{}`, mode.Value)
	assert.Equal(t, `{}`, expression.Value)
}

func TestUpdateBillingOptionsConcurrentPublishMatchesDatabase(t *testing.T) {
	preserveBillingOptionState(t)

	const writers = 20
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			modelName := fmt.Sprintf("concurrent-%d", i)
			modeValue := fmt.Sprintf(`{"%s":"tiered_expr"}`, modelName)
			exprValue := fmt.Sprintf(`{"%s":"p + %d"}`, modelName, i)
			errs <- UpdateBillingOptions(modeValue, exprValue)
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	var mode Option
	var expression Option
	require.NoError(t, DB.First(&mode, "key = ?", "billing_setting.billing_mode").Error)
	require.NoError(t, DB.First(&expression, "key = ?", "billing_setting.billing_expr").Error)

	common.OptionMapRWMutex.RLock()
	assert.Equal(t, mode.Value, common.OptionMap["billing_setting.billing_mode"])
	assert.Equal(t, expression.Value, common.OptionMap["billing_setting.billing_expr"])
	common.OptionMapRWMutex.RUnlock()

	modes, expressions, err := billing_setting.ParseAndValidateBillingConfig(mode.Value, expression.Value)
	require.NoError(t, err)
	for modelName, modeValue := range modes {
		assert.Equal(t, modeValue, billing_setting.GetBillingMode(modelName))
		actualExpression, ok := billing_setting.GetBillingExpr(modelName)
		require.True(t, ok)
		assert.Equal(t, expressions[modelName], actualExpression)
	}
}
