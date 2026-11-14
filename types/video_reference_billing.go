package types

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// Frozen task prices and measured media usage. Never rebuild this from current
// settings while polling, settling or refunding an accepted task.
type VideoReferenceBilling struct {
	Resolution        string  `json:"resolution"`
	GenerationSeconds float64 `json:"generation_seconds"`
	ReferenceSeconds  float64 `json:"reference_seconds"`
	GenerationPrice   float64 `json:"generation_price"`
	ReferencePrice    float64 `json:"reference_price"`
}

func (b *VideoReferenceBilling) Quota(quotaPerUnit, groupRatio float64) (int, error) {
	for _, value := range []float64{b.GenerationSeconds, b.ReferenceSeconds, b.GenerationPrice, b.ReferencePrice, quotaPerUnit, groupRatio} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return 0, fmt.Errorf("invalid reference-video billing value")
		}
	}
	amount := decimal.NewFromFloat(b.GenerationSeconds).Mul(decimal.NewFromFloat(b.GenerationPrice)).Add(decimal.NewFromFloat(b.ReferenceSeconds).Mul(decimal.NewFromFloat(b.ReferencePrice)))
	quota := amount.Mul(decimal.NewFromFloat(quotaPerUnit)).Mul(decimal.NewFromFloat(groupRatio)).Ceil()
	if quota.GreaterThan(decimal.NewFromInt(int64(int(^uint(0) >> 1)))) {
		return 0, fmt.Errorf("video quota exceeds supported range")
	}
	return int(quota.IntPart()), nil
}
