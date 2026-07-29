package billingexpr

import (
	"fmt"
	"math"
)

// MaxBillingAmount is the maximum billable amount represented by one request.
// Expression output uses price-per-million-token units, so the corresponding
// raw expression limit is MaxBillingAmount * 1,000,000.
const MaxBillingAmount = 1_000_000_000.0

const maxExpressionResult = MaxBillingAmount * 1_000_000

// A float64 cannot represent every integer above 2^53 exactly. Keeping quota
// below this boundary also makes the float-to-int conversion deterministic.
const maxExactInteger = float64(1<<53 - 1)

func validateExpressionResult(result float64) error {
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return fmt.Errorf("billing expression result must be finite, got %v", result)
	}
	if result < 0 {
		return fmt.Errorf("billing expression result must not be negative, got %g", result)
	}
	if result > maxExpressionResult {
		return fmt.Errorf("billing expression result %g exceeds maximum %g", result, maxExpressionResult)
	}
	return nil
}

// QuotaRoundChecked validates a computed quota before converting it to int.
// quotaPerUnit is used to enforce the same maximum account value used by the
// token quota API, while maxExactInteger prevents conversion overflow.
func QuotaRoundChecked(value, quotaPerUnit float64) (int, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("billing quota must be finite, got %v", value)
	}
	if value < 0 {
		return 0, fmt.Errorf("billing quota must not be negative, got %g", value)
	}
	if math.IsNaN(quotaPerUnit) || math.IsInf(quotaPerUnit, 0) || quotaPerUnit < 0 {
		return 0, fmt.Errorf("quota per unit must be a finite non-negative value, got %v", quotaPerUnit)
	}

	maximum := MaxBillingAmount * quotaPerUnit
	if maximum > maxExactInteger {
		maximum = maxExactInteger
	}
	rounded := math.Round(value)
	if rounded > maximum {
		return 0, fmt.Errorf("billing quota %.0f exceeds maximum %.0f", rounded, maximum)
	}
	return int(rounded), nil
}
