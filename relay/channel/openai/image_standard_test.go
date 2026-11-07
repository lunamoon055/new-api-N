package openai

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestStandardImageOptionsTranslateWithoutDroppingExplicitValues(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		var request dto.ImageRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"`+name+`","prompt":"cup","resolution":"2K","ratio":"16:9","output_format":"webp","output_compression":0,"watermark":false,"safety_identifier":""}`, &request))
		converted, err := normalizeStandardImageRequest(request)
		require.NoError(t, err)
		require.Equal(t, "2048x1152", converted.Size)
		encoded, err := common.Marshal(converted)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"output_compression":0`)
		require.Contains(t, string(encoded), `"watermark":false`)
		require.Contains(t, string(encoded), `"safety_identifier":""`)
		require.NotContains(t, string(encoded), `"resolution"`)
		require.NotContains(t, string(encoded), `"ratio"`)
		request.Size = "1536x864"
		converted, err = normalizeStandardImageRequest(request)
		require.NoError(t, err)
		require.Equal(t, "1536x864", converted.Size)
	}
	_, err := normalizeStandardImageRequest(dto.ImageRequest{Model: "gpt-image-2", Quality: "xhigh"})
	require.Error(t, err)
	_, err = normalizeStandardImageRequest(dto.ImageRequest{Model: "gpt-image-2.5-flare", Quality: "xhigh"})
	require.NoError(t, err)
	_, err = normalizeStandardImageRequest(dto.ImageRequest{Model: "gpt-image-2", Size: "4096x4096"})
	require.Error(t, err)
	values, err := standardImageFormValues(map[string][]string{"prompt": {"cup"}, "resolution": {"4K"}, "ratio": {"9:16"}, "output_compression": {"0"}, "watermark": {"false"}}, "gpt-image-2.5-flare")
	require.NoError(t, err)
	require.Equal(t, []string{"2160x3840"}, values["size"])
	require.Equal(t, []string{"0"}, values["output_compression"])
	require.NotContains(t, values, "ratio")
	for legacy, expected := range map[string]string{"standard": "medium", "hd": "high"} {
		values, err = standardImageFormValues(map[string][]string{"quality": {legacy}}, "gpt-image-2")
		require.NoError(t, err)
		require.Equal(t, []string{expected}, values["quality"])
	}
}

func TestStandardJSONImageEditPreservesReferenceObjectsAndMask(t *testing.T) {
	input := `{"model":"gpt-image-2.5-flare","prompt":"green cup","images":[{"image_url":"https://example.com/ref.png"},{"file_id":"file-ref"}],"mask":{"image_url":"https://example.com/mask.png"},"quality":"xhigh","n":2,"size":"1536x864","output_format":"webp","output_compression":0}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/edits", strings.NewReader(input))
	c.Request.Header.Set("Content-Type", "application/json")
	var request dto.ImageRequest
	require.NoError(t, common.UnmarshalJsonStr(input, &request))
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: "https://newapi.megabyai.cc"}, RelayMode: relayconstant.RelayModeImagesEdits}
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, request)
	require.NoError(t, err)
	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.JSONEq(t, input, string(encoded))
}

func TestStandardMultipartEditPreservesMarkupAndNormalizesOptions(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"model": "gpt-image-2.5-flare", "prompt": "green cup", "quality": "hd", "resolution": "2K", "ratio": "16:9", "n": "2", "output_format": "webp", "output_compression": "0"} {
		require.NoError(t, writer.WriteField(key, value))
	}
	for field, value := range map[string]string{"image": "reference-bytes", "markup": "annotation-bytes"} {
		part, err := writer.CreateFormFile(field, field+".png")
		require.NoError(t, err)
		_, err = io.WriteString(part, value)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: "https://newapi.megabyai.cc"}, RelayMode: relayconstant.RelayModeImagesEdits}
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{Model: "gpt-image-2.5-flare"})
	require.NoError(t, err)
	out := httptest.NewRequest("POST", "/v1/images/edits", converted.(io.Reader))
	out.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	require.NoError(t, out.ParseMultipartForm(1<<20))
	require.Equal(t, "2048x1152", out.FormValue("size"))
	require.Equal(t, "high", out.FormValue("quality"))
	require.Equal(t, "2", out.FormValue("n"))
	require.Equal(t, "0", out.FormValue("output_compression"))
	require.Empty(t, out.FormValue("resolution"))
	for field, expected := range map[string]string{"image[]": "reference-bytes", "markup": "annotation-bytes"} {
		require.Len(t, out.MultipartForm.File[field], 1)
		file, err := out.MultipartForm.File[field][0].Open()
		require.NoError(t, err)
		value, err := io.ReadAll(file)
		file.Close()
		require.NoError(t, err)
		require.Equal(t, expected, string(value))
	}
}
