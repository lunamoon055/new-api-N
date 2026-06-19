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
