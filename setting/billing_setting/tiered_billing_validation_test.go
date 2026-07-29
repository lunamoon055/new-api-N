package billing_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateBillingConfig(t *testing.T) {
	modes, expressions, err := ParseAndValidateBillingConfig(
		`{"gpt-test":"tiered_expr"}`,
		`{"gpt-test":"p * 2 + c * 8"}`,
	)
	require.NoError(t, err)
	assert.Equal(t, BillingModeTieredExpr, modes["gpt-test"])
	assert.Equal(t, "p * 2 + c * 8", expressions["gpt-test"])
}

func TestParseAndValidateBillingConfigRejectsInvalidPairs(t *testing.T) {
	tests := []struct {
		name  string
		modes string
		exprs string
	}{
		{name: "missing expression", modes: `{"m":"tiered_expr"}`, exprs: `{}`},
		{name: "negative expression", modes: `{"m":"tiered_expr"}`, exprs: `{"m":"-1"}`},
		{name: "non finite expression", modes: `{"m":"tiered_expr"}`, exprs: `{"m":"p / 0"}`},
		{name: "oversized result", modes: `{"m":"tiered_expr"}`, exprs: `{"m":"2e15"}`},
		{name: "unknown mode", modes: `{"m":"unknown"}`, exprs: `{}`},
		{name: "invalid JSON", modes: `{`, exprs: `{}`},
		{name: "null mode", modes: `null`, exprs: `{}`},
		{name: "null expression", modes: `{}`, exprs: `null`},
		{name: "whitespace model", modes: `{" m":"ratio"}`, exprs: `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseAndValidateBillingConfig(tt.modes, tt.exprs)
			require.Error(t, err)
		})
	}
}

func TestApplyBillingConfigReplacesBothMaps(t *testing.T) {
	originalModes := GetBillingModeCopy()
	originalExpressions := GetBillingExprCopy()
	t.Cleanup(func() { ApplyBillingConfig(originalModes, originalExpressions) })

	ApplyBillingConfig(
		map[string]string{"new-model": BillingModeTieredExpr},
		map[string]string{"new-model": "p"},
	)

	assert.Equal(t, BillingModeTieredExpr, GetBillingMode("new-model"))
	assert.Equal(t, BillingModeRatio, GetBillingMode("removed-model"))
	expression, ok := GetBillingExpr("new-model")
	require.True(t, ok)
	assert.Equal(t, "p", expression)
}
