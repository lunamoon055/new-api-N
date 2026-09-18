package helper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetAndValidOpenAIImageRequestNormalizesGPTImage2Quality(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		model       string
		quality     string
		wantQuality string
	}{
		{name: "standard becomes medium", model: "gpt-image-2", quality: "standard", wantQuality: "medium"},
		{name: "hd becomes high", model: "gpt-image-2.5-flare", quality: "hd", wantQuality: "high"},
		{name: "auto is omitted", model: "gpt-image-2", quality: "auto", wantQuality: ""},
		{name: "supported quality is unchanged", model: "gpt-image-2.5-sunburst", quality: "xhigh", wantQuality: "xhigh"},
		{name: "legacy alias is supported", model: "gpt-image2", quality: "standard", wantQuality: "medium"},
		{name: "dall-e remains unchanged", model: "dall-e-3", quality: "hd", wantQuality: "hd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"model":"` + tc.model + `","prompt":"test","quality":"` + tc.quality + `"}`
			request := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = request

			imageRequest, err := GetAndValidOpenAIImageRequest(ctx, relayconstant.RelayModeImagesGenerations)
			require.NoError(t, err)
			require.Equal(t, tc.wantQuality, imageRequest.Quality)
		})
	}
}
