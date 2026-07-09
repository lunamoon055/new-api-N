package billing_setting

import "testing"

func TestGetVideoBillingModeDefaultsToDynamic(t *testing.T) {
	original := billingSetting.VideoBillingMode
	t.Cleanup(func() {
		billingSetting.VideoBillingMode = original
	})
	billingSetting.VideoBillingMode = map[string]string{}

	if got := GetVideoBillingMode("sora2"); got != VideoBillingModeDynamic {
		t.Fatalf("expected missing mode to default to %q, got %q", VideoBillingModeDynamic, got)
	}
}

func TestGetVideoBillingModeReturnsFixed(t *testing.T) {
	original := billingSetting.VideoBillingMode
	t.Cleanup(func() {
		billingSetting.VideoBillingMode = original
	})
	billingSetting.VideoBillingMode = map[string]string{
		"sora2": VideoBillingModeFixed,
	}

	if got := GetVideoBillingMode("sora2"); got != VideoBillingModeFixed {
		t.Fatalf("expected fixed mode, got %q", got)
	}
}

func TestGetVideoBillingModeReturnsResolutionTierModes(t *testing.T) {
	original := billingSetting.VideoBillingMode
	t.Cleanup(func() {
		billingSetting.VideoBillingMode = original
	})
	billingSetting.VideoBillingMode = map[string]string{
		"tiered-seconds-model": VideoBillingModeTieredSeconds,
		"tiered-request-model": VideoBillingModeTieredRequest,
	}

	if got := GetVideoBillingMode("tiered-seconds-model"); got != VideoBillingModeTieredSeconds {
		t.Fatalf("expected tiered seconds mode, got %q", got)
	}
	if got := GetVideoBillingMode("tiered-request-model"); got != VideoBillingModeTieredRequest {
		t.Fatalf("expected tiered request mode, got %q", got)
	}
}

func TestGetVideoBillingModeInvalidFallsBackToDynamic(t *testing.T) {
	original := billingSetting.VideoBillingMode
	t.Cleanup(func() {
		billingSetting.VideoBillingMode = original
	})
	billingSetting.VideoBillingMode = map[string]string{
		"sora2": "per-second",
	}

	if got := GetVideoBillingMode("sora2"); got != VideoBillingModeDynamic {
		t.Fatalf("expected invalid mode to fall back to %q, got %q", VideoBillingModeDynamic, got)
	}
}

func TestGetVideoResolutionPricesReturnsCopy(t *testing.T) {
	original := billingSetting.VideoResolutionPrices
	t.Cleanup(func() {
		billingSetting.VideoResolutionPrices = original
	})
	billingSetting.VideoResolutionPrices = map[string]map[string]float64{
		"video-model": {
			"480p":  0.01,
			"720p":  0.02,
			"1080p": 0.04,
			"4k":    0.08,
		},
	}

	prices, ok := GetVideoResolutionPrices("video-model")
	if !ok {
		t.Fatal("expected resolution prices")
	}
	prices["720p"] = 99

	pricesAgain, ok := GetVideoResolutionPrices("video-model")
	if !ok {
		t.Fatal("expected resolution prices on second read")
	}
	if got := pricesAgain["720p"]; got != 0.02 {
		t.Fatalf("expected copy mutation not to affect source, got %v", got)
	}
}

func TestGetVideoBillingModeCopy(t *testing.T) {
	original := billingSetting.VideoBillingMode
	t.Cleanup(func() {
		billingSetting.VideoBillingMode = original
	})
	billingSetting.VideoBillingMode = map[string]string{
		"sora2": VideoBillingModeFixed,
	}

	copied := GetVideoBillingModeCopy()
	copied["sora2"] = VideoBillingModeDynamic

	if got := GetVideoBillingMode("sora2"); got != VideoBillingModeFixed {
		t.Fatalf("expected copy mutation not to affect source, got %q", got)
	}
}
