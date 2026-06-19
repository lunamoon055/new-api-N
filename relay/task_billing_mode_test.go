package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
)

func TestApplyTaskOtherRatiosToQuotaDynamicMultipliesQuota(t *testing.T) {
	priceData := types.PriceData{
		Quota:       100,
		UsePrice:    true,
		OtherRatios: map[string]float64{"seconds": 4, "size": 1.5},
	}

	if err := applyTaskOtherRatiosToQuota(&priceData, billing_setting.VideoBillingModeDynamic, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if priceData.Quota != 600 {
		t.Fatalf("expected dynamic mode to multiply quota to 600, got %d", priceData.Quota)
	}
	if !priceData.AppliedOtherRatios {
		t.Fatal("expected dynamic mode to mark other ratios as applied")
	}
}

func TestApplyTaskOtherRatiosToQuotaFixedKeepsPerRequestQuota(t *testing.T) {
	priceData := types.PriceData{
		Quota:       100,
		UsePrice:    true,
		OtherRatios: map[string]float64{"seconds": 4, "size": 1.5},
	}

	if err := applyTaskOtherRatiosToQuota(&priceData, billing_setting.VideoBillingModeFixed, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if priceData.Quota != 100 {
		t.Fatalf("expected fixed mode to keep quota at 100, got %d", priceData.Quota)
	}
	if priceData.AppliedOtherRatios {
		t.Fatal("expected fixed mode not to mark other ratios as applied")
	}
	if priceData.VideoBillingMode != billing_setting.VideoBillingModeFixed {
		t.Fatalf("expected fixed mode metadata, got %q", priceData.VideoBillingMode)
	}
}

func TestApplyTaskOtherRatiosToQuotaFixedRequiresModelPrice(t *testing.T) {
	priceData := types.PriceData{
		Quota:       100,
		UsePrice:    false,
		OtherRatios: map[string]float64{"seconds": 4},
	}

	if err := applyTaskOtherRatiosToQuota(&priceData, billing_setting.VideoBillingModeFixed, false); err == nil {
		t.Fatal("expected fixed video billing without ModelPrice to fail")
	}
}

func TestApplyTaskOtherRatiosToQuotaTaskPricePatchKeepsExistingPerCallBehavior(t *testing.T) {
	priceData := types.PriceData{
		Quota:       100,
		UsePrice:    true,
		OtherRatios: map[string]float64{"seconds": 4},
	}

	if err := applyTaskOtherRatiosToQuota(&priceData, billing_setting.VideoBillingModeDynamic, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if priceData.Quota != 100 {
		t.Fatalf("expected patched per-call task to keep quota at 100, got %d", priceData.Quota)
	}
	if priceData.AppliedOtherRatios {
		t.Fatal("expected patched per-call task not to mark other ratios as applied")
	}
}
