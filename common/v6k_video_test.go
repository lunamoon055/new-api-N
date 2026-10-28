package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV6KZHNewModelsHaveExactPublishedCapabilities(t *testing.T) {
	for _, tc := range []struct {
		id                                       string
		minimum, maximum, images, videos, audios int
		resolutions                              []string
	}{
		{"Cseadanco2.0-1", 5, 15, 9, 3, 3, []string{"720p"}},
		{"Cseadanco2.0-fast", 5, 15, 9, 3, 3, []string{"720p"}},
		{"Cseadanco2.0-mini", 5, 15, 9, 3, 3, []string{"720p"}},
		{"Cseadanco2.5-720", 5, 30, 30, 10, 10, []string{"720p"}},
		{"TTTseadanco2.5/480", 5, 30, 30, 0, 10, []string{"480p"}},
		{"TTTseadanco2.5/720", 5, 30, 30, 0, 10, []string{"720p"}},
		{"XXseadanco2.5-1080", 5, 30, 30, 10, 10, []string{"1080p"}},
		{"XXseadanco2.5-480", 5, 30, 30, 10, 10, []string{"480p"}},
		{"XXseadanco2.5-720", 5, 30, 30, 10, 10, []string{"720p"}},
		{"wan3.M", 5, 30, 10, 0, 5, []string{"480p", "720p"}},
	} {
		t.Run(tc.id, func(t *testing.T) {
			p := GetV6KValidatedVideoParameters(tc.id)
			require.NotNil(t, p)
			require.Equal(t, integrationDurationRange(tc.minimum, tc.maximum), p.Durations)
			require.Equal(t, tc.resolutions, p.Resolutions)
			require.Equal(t, tc.images, p.MaxImages)
			require.Equal(t, tc.videos, p.MaxVideos)
			require.Equal(t, tc.audios, p.MaxAudios)
			require.Equal(t, "seconds", p.DurationField)
			require.False(t, p.OmitRatio)
			require.Equal(t, DefaultVideoAspectRatios(), p.Ratios)
		})
	}
	require.Equal(t, []int{5, 10, 15}, GetV6KValidatedVideoParameters("dola2.5").Durations)
	for _, original := range []string{"JJseadanco2.5", "TTseadanco2.5-720", "XXsedanco2.5S", "Cseadanco2.0-1-extra"} {
		require.Nil(t, GetV6KValidatedVideoParameters(original), "only exact selected IDs gain stricter validation")
	}
}

func TestV6KUnknownCapabilitiesAreNotExplicitZeroLimits(t *testing.T) {
	for _, name := range []string{"BBseadanco2.5-480", "BBseadanco2.5-720", "future-video-model"} {
		p := GetV6KVideoParameters(name)
		require.True(t, p.CapabilitiesUnknown, name)
		require.Empty(t, p.Durations)
		require.Empty(t, p.Resolutions)
		require.NotEmpty(t, p.ImageField)
		require.NotEmpty(t, p.VideoField)
		require.NotEmpty(t, p.AudioField)
		require.Nil(t, GetV6KValidatedVideoParameters(name))
	}
	for _, name := range []string{"BBseadanco-480", "BBseadanco-720", "TTTseadanco2.5/720"} {
		p := GetV6KVideoParameters(name)
		require.False(t, p.CapabilitiesUnknown, name)
		require.Equal(t, 0, p.MaxVideos)
		require.NotEmpty(t, p.Durations)
	}
}
