package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

// Exercise the exact deployable channel configuration after the production
// image DTO conversion, including omission and explicit false preservation.
func TestXJZImageChannelConfiguration(t *testing.T) {
	raw, err := os.ReadFile("../../../integrations/xjz-image/param-override.json")
	require.NoError(t, err)
	var override map[string]interface{}
	require.NoError(t, common.Unmarshal(raw, &override))
	raw, err = os.ReadFile("../../../integrations/xjz-image/model-mapping.json")
	require.NoError(t, err)
	var mapping map[string]string
	require.NoError(t, common.Unmarshal(raw, &mapping))
	raw, err = os.ReadFile("../../../integrations/xjz-image/header-override.json")
	require.NoError(t, err)
	var headers map[string]interface{}
	require.NoError(t, common.Unmarshal(raw, &headers))
	service.InitHttpClient()
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, fields := range []string{``, `,"output_resolution":"4K","aspect_ratio":"9:16","size":"1024x1024","watermark":false,"images":["https://example.com/reference.png"]`} {
			t.Run(name+fields, func(t *testing.T) {
				var request dto.ImageRequest
				require.Equal(t, name+"-原生4K", mapping[name])
				require.NoError(t, common.UnmarshalJsonStr(`{"model":"`+mapping[name]+`","prompt":"blue cup","n":1`+fields+`}`, &request))
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(""))
				c.Request.Header.Set("Content-Type", "application/json")
				converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}, request)
				require.NoError(t, err)
				body, err := common.Marshal(converted)
				require.NoError(t, err)
				body, err = relaycommon.ApplyParamOverride(body, override, nil)
				require.NoError(t, err)
				var actual map[string]interface{}
				require.NoError(t, common.Unmarshal(body, &actual))
				require.Equal(t, name+"-原生4K", actual["model"])
				require.Equal(t, float64(1), actual["n"])
				require.NotContains(t, actual, "size")
				require.NotContains(t, actual, "output_resolution")
				require.NotContains(t, actual, "aspect_ratio")
				if fields == "" {
					require.Equal(t, "1K", actual["resolution"])
					require.Equal(t, "1:1", actual["ratio"])
				} else {
					require.Equal(t, "4K", actual["resolution"])
					require.Equal(t, "9:16", actual["ratio"])
					require.Equal(t, false, actual["watermark"])
					require.Equal(t, []interface{}{"https://example.com/reference.png"}, actual["images"])
				}
				// Verify the actual HTTP boundary, including the deployable header
				// fixture. A mock response does not prove upstream availability.
				type capturedRequest struct {
					method, path string
					header       http.Header
					body         []byte
					err          error
				}
				captured := make(chan capturedRequest, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					payload, readErr := io.ReadAll(r.Body)
					captured <- capturedRequest{r.Method, r.URL.Path, r.Header.Clone(), payload, readErr}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"data":[{"url":"https://example.com/mock.png"}]}`)
				}))
				defer server.Close()
				info := &relaycommon.RelayInfo{
					ChannelMeta:    &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: server.URL, ApiKey: "test-key", HeadersOverride: headers},
					RelayMode:      relayconstant.RelayModeImagesGenerations,
					RequestURLPath: "/v1/images/generations",
				}
				response, err := (&Adaptor{}).DoRequest(c, info, bytes.NewReader(body))
				require.NoError(t, err)
				resp := response.(*http.Response)
				defer resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode)
				sent := <-captured
				require.NoError(t, sent.err)
				require.Equal(t, "POST", sent.method)
				require.Equal(t, "/v1/images/generations", sent.path)
				require.Equal(t, "1", sent.header.Get("X-Xiaojuzi-Media-Contract"))
				require.Equal(t, "Bearer test-key", sent.header.Get("Authorization"))
				require.JSONEq(t, string(body), string(sent.body))
			})
		}
	}
	other := []byte(`{"model":"gpt-image-2.5-flare-other","output_resolution":"2K","aspect_ratio":"16:9","size":"2048x1152"}`)
	result, err := relaycommon.ApplyParamOverride(other, override, nil)
	require.NoError(t, err)
	require.JSONEq(t, string(other), string(result))
}
