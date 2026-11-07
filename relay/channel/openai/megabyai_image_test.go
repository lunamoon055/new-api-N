package openai

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// No generation is sent to the paid supplier: exercise DTO conversion and the
// actual HTTP boundary against a local upstream for each configured model.
func TestMegaBYAIImageGenerationHTTPContract(t *testing.T) {
	service.InitHttpClient()
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, extra := range []string{"", `,"size":"1024x1024","watermark":false,"output_compression":0,"response_format":"b64_json"`} {
			t.Run(name+extra, func(t *testing.T) {
				input := `{"model":"` + name + `","prompt":"blue ceramic cup","n":1` + extra + `}`
				var request dto.ImageRequest
				require.NoError(t, common.UnmarshalJsonStr(input, &request))
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(input))
				c.Request.Header.Set("Content-Type", "application/json")
				info := &relaycommon.RelayInfo{
					ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: "https://newapi.megabyai.cc", ApiKey: "fixture-key"},
					RelayMode:   relayconstant.RelayModeImagesGenerations, RequestURLPath: "/v1/images/generations",
				}
				converted, err := (&Adaptor{}).ConvertImageRequest(c, info, request)
				require.NoError(t, err)
				body, err := common.Marshal(converted)
				require.NoError(t, err)
				require.JSONEq(t, input, string(body), "do not inject XJZ defaults or drop explicit false/zero")
				type captured struct {
					method, path, auth, contentType, xjzContract string
					body                                         []byte
					err                                          error
				}
				sent := make(chan captured, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					payload, err := io.ReadAll(r.Body)
					sent <- captured{r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"), r.Header.Get("X-Xiaojuzi-Media-Contract"), payload, err}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"created":1794000000,"data":[{"url":"https://example.com/result.png"}]}`)
				}))
				defer server.Close()
				info.ChannelBaseUrl = server.URL
				response, err := (&Adaptor{}).DoRequest(c, info, bytes.NewReader(body))
				require.NoError(t, err)
				resp := response.(*http.Response)
				defer resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode)
				actual := <-sent
				require.NoError(t, actual.err)
				require.Equal(t, http.MethodPost, actual.method)
				require.Equal(t, "/v1/images/generations", actual.path)
				require.Equal(t, "Bearer fixture-key", actual.auth)
				require.Equal(t, "application/json", actual.contentType)
				require.Empty(t, actual.xjzContract)
				require.JSONEq(t, input, string(actual.body))
			})
		}
	}
}

func TestMegaBYAIReferenceEditPreservesFileBytesAndUsesVerifiedArrayField(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, incomingField := range []string{"image", "image[]"} {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for key, value := range map[string]string{"model": name, "prompt": "turn the square green", "n": "1", "watermark": "false", "seed": "0"} {
				require.NoError(t, writer.WriteField(key, value))
			}
			part, err := writer.CreateFormFile(incomingField, "reference.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("reference-image-bytes"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: "https://newapi.megabyai.cc"}, RelayMode: relayconstant.RelayModeImagesEdits}
			converted, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{Model: name})
			require.NoError(t, err)
			out := httptest.NewRequest(http.MethodPost, "/v1/images/edits", converted.(io.Reader))
			out.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
			require.NoError(t, out.ParseMultipartForm(1<<20))
			require.Equal(t, name, out.FormValue("model"))
			require.Equal(t, "false", out.FormValue("watermark"))
			require.Equal(t, "0", out.FormValue("seed"))
			require.Empty(t, out.FormValue("resolution"))
			require.Empty(t, out.FormValue("ratio"))
			require.Empty(t, out.MultipartForm.File["image"])
			require.Len(t, out.MultipartForm.File["image[]"], 1)
			file, err := out.MultipartForm.File["image[]"][0].Open()
			require.NoError(t, err)
			data, err := io.ReadAll(file)
			file.Close()
			require.NoError(t, err)
			require.Equal(t, "reference-image-bytes", string(data))
		}
	}
}
