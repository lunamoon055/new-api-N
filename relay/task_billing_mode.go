package relay

import (
	"fmt"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
)

func applyTaskOtherRatiosToQuota(priceData *types.PriceData, videoBillingMode string, skipOtherRatios bool) error {
	if priceData == nil {
		return nil
	}

	mode := billing_setting.NormalizeVideoBillingMode(videoBillingMode)
	priceData.VideoBillingMode = mode
	priceData.AppliedOtherRatios = false

	if skipOtherRatios {
		return nil
	}

	if mode == billing_setting.VideoBillingModeFixed {
		if !priceData.UsePrice {
			return fmt.Errorf("fixed video billing requires ModelPrice")
		}
		return nil
	}

	for _, ra := range priceData.OtherRatios {
		if ra != 1.0 {
			priceData.Quota = int(float64(priceData.Quota) * ra)
			priceData.AppliedOtherRatios = true
		}
	}
	return nil
}
