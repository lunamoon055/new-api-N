package dto

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImageIntegrationFieldsSurviveUpstreamMarshal(t *testing.T) {
	var request ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{"model":"gpt-image-2","prompt":"sunrise","resolution":"4K","aspect_ratio":"16:9","images":["https://cdn.example/ref.png"],"watermark":false}`), &request))
	encoded, err := common.Marshal(request)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, common.Unmarshal(encoded, &actual))
	require.Equal(t, "4K", actual["resolution"])
	require.Equal(t, "16:9", actual["aspect_ratio"])
	require.Equal(t, []any{"https://cdn.example/ref.png"}, actual["images"])
	require.Equal(t, false, actual["watermark"])
}
