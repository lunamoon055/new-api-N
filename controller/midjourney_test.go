package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

func TestShouldForwardMidjourneyImage(t *testing.T) {
	t.Parallel()

	require.True(t, shouldForwardMidjourneyImage(&model.Midjourney{Action: constant.MjActionImagine}))
	require.False(t, shouldForwardMidjourneyImage(&model.Midjourney{Action: constant.MjActionImageGeneration}))
	require.False(t, shouldForwardMidjourneyImage(nil))
}
