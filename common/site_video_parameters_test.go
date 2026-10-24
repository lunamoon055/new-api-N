package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiteVideoNotesHaveExactPublicLimits(t *testing.T) {
	for _, tc := range []struct {
		model                            string
		min, max, images, videos, audios int
	}{
		{"无限制-Video", 4, 12, 5, 0, 3},
		{"Seedance2.0-c", 4, 15, 9, 3, 3},
		{"Seedance2.0-fast-c", 5, 15, 9, 3, 3},
		{"Seedance2.0-fast2-c", 5, 15, 9, 3, 3},
		{"Seedance2.0:720（秒）", 5, 15, 9, 3, 3},
		{"Seedance2.5:480（秒）", 5, 30, 30, 10, 10},
		{"Seedance2.5:720（秒）", 5, 30, 30, 10, 10},
		{"Seedance2.5:720（条）", 5, 30, 30, 10, 10},
		{"wan3.0(图片+音频参考)", 2, 30, 10, 0, 5},
	} {
		t.Run(tc.model, func(t *testing.T) {
			p := GetVideoIntegrationParameters(tc.model)
			require.NotNil(t, p)
			require.Equal(t, integrationDurationRange(tc.min, tc.max), p.Durations)
			require.Equal(t, tc.images, p.MaxImages)
			require.Equal(t, tc.videos, p.MaxVideos)
			require.Equal(t, tc.audios, p.MaxAudios)
		})
	}
	// The screenshot's 480 alias has a conflicting 720P note; keep its actual tier.
	require.Equal(t, []string{"480p"}, GetSiteVideoIntegrationParameters("Seedance2.5:480（秒）").Resolutions)
	// Exact aliases do not expand other provider or previously isolated models.
	require.Equal(t, 4, GetVideoIntegrationParameters("sd-2-c4").Durations[0])
	require.Equal(t, 10, GetVideoIntegrationParameters("Seedance2.5-c").MaxImages)
	require.Equal(t, 0, GetVideoIntegrationParameters("Seedance2.5-c").MaxAudios)
	require.Nil(t, GetSiteVideoIntegrationParameters("Seedance2.5:720（未知）"))
}
