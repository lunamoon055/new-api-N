package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoRatioDefaultsFillOnlyUnconfiguredRanges(t *testing.T) {
	missing := &ModelIntegrationParameters{OmitRatio: true, Durations: []int{5, 10, 15}, MaxImages: 9}
	filled := WithVideoRatioDefaults(missing)
	require.Equal(t, DefaultVideoAspectRatios(), filled.Ratios)
	require.Equal(t, "ratio", filled.RatioField)
	require.False(t, filled.OmitRatio)
	require.Equal(t, missing.Durations, filled.Durations)
	require.Equal(t, 9, filled.MaxImages)
	require.True(t, missing.OmitRatio)
	require.Empty(t, missing.Ratios)

	explicit := &ModelIntegrationParameters{Ratios: []string{"9:16", "1:1"}, RatioField: "aspect_ratio"}
	require.Equal(t, explicit, WithVideoRatioDefaults(explicit))
	require.Nil(t, WithVideoRatioDefaults(nil))
	for _, name := range V6KVideoModelIDs() {
		p := GetV6KVideoParameters(name)
		require.NotEmpty(t, p.Ratios, name)
		require.False(t, p.OmitRatio, name)
	}
	require.Equal(t, []string{"16:9", "9:16", "1:1"}, GetV6KVideoParameters("grok--1.5").Ratios)
	require.Equal(t, DefaultVideoAspectRatios(), GetV6KVideoParameters("future-model").Ratios)
	require.Equal(t, DefaultVideoAspectRatios(), GetSiteVideoIntegrationParameters("无限制-Video").Ratios)
	require.Equal(t, []string{"16:9", "9:16", "1:1", "4:3", "3:4"}, GetSiteVideoIntegrationParameters("Seedance2.0-c").Ratios)
}
