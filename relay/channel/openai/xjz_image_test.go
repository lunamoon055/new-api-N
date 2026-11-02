package openai

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestXJZNativeImageJSONContract(t *testing.T) {
	for _, model := range []string{"gpt-image-2-原生4K", "gpt-image-2.5-flare-原生4K", "gpt-image-2.5-sunburst-原生4K"} {
		t.Run(model, func(t *testing.T) {
			var request dto.ImageRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"`+model+`","prompt":"cup","resolution":"4K","ratio":"21:9","size":"1024x1024"}`, &request))
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.xjzapi.cc"}, RelayMode: relayconstant.RelayModeImagesGenerations}
			converted, err := (&Adaptor{}).ConvertImageRequest(nil, info, request)
			require.NoError(t, err)
			encoded, err := common.Marshal(converted)
			require.NoError(t, err)
			require.JSONEq(t, `{"model":"`+model+`","prompt":"cup","resolution":"4K","ratio":"21:9"}`, string(encoded))
			request.Images = []byte(`["https://example.com/ref.png"]`)
			_, err = (&Adaptor{}).ConvertImageRequest(nil, info, request)
			require.ErrorContains(t, err, "/v1/images/edits")
			info.ChannelBaseUrl = "https://unrelated.example"
			converted, err = (&Adaptor{}).ConvertImageRequest(nil, info, request)
			require.NoError(t, err)
			require.Equal(t, request, converted, "other providers must not inherit XJZ conversions")
		})
	}
}

func TestXJZNativeImageMultipartContract(t *testing.T) {
	for _, count := range []int{1, 20, 21} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for key, value := range map[string]string{"model": "gpt-image-2", "prompt": "turn the cup red", "output_resolution": "2K", "aspect_ratio": "9:16", "size": "1024x1024"} {
			require.NoError(t, writer.WriteField(key, value))
		}
		for i := 0; i < count; i++ {
			part, err := writer.CreateFormFile("image[]", "cup.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("fixture"))
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.xjzapi.cc"}, RelayMode: relayconstant.RelayModeImagesEdits}
		converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{Model: "gpt-image-2-原生4K"})
		if count > 20 {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		out := httptest.NewRequest("POST", "/", converted.(io.Reader))
		out.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
		require.NoError(t, out.ParseMultipartForm(1<<20))
		require.Equal(t, "gpt-image-2-原生4K", out.FormValue("model"))
		require.Equal(t, "2K", out.FormValue("resolution"))
		require.Equal(t, "9:16", out.FormValue("ratio"))
		require.Empty(t, out.FormValue("size"))
		require.Empty(t, out.FormValue("output_resolution"))
		require.Len(t, out.MultipartForm.File["image[]"], count)
		file, err := out.MultipartForm.File["image[]"][0].Open()
		require.NoError(t, err)
		data, err := io.ReadAll(file)
		file.Close()
		require.NoError(t, err)
		require.Equal(t, "fixture", string(data))
	}
	_, err := xjzImageFormValues(map[string][]string{"ratio": {strings.Repeat("1", 4) + ":1"}})
	require.Error(t, err)
}
