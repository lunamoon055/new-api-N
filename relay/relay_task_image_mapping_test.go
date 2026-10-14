package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAsyncImageSubmissionMapsBeforeValidationAndResetsOnRetry(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-alias", RequestURLPath: "/v1/images/async-generations",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	for _, target := range []string{"gpt-image-2.5", "nano-banana2"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		// n=0 guarantees this integration check stops before price lookup or any paid request.
		c.Request = httptest.NewRequest(http.MethodPost, info.RequestURLPath, strings.NewReader(`{"model":"public-alias","prompt":"garden portrait","n":0}`))
		c.Request.Header.Set("Content-Type", "application/json")
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://linksky.top")
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "public-alias")
		mapping, err := common.Marshal(map[string]string{"public-alias": target})
		require.NoError(t, err)
		c.Set("model_mapping", string(mapping))
		_, taskErr := RelayTaskSubmit(c, info)
		require.NotNil(t, taskErr)
		require.Equal(t, "n must be 1", taskErr.Message)
		require.Equal(t, "public-alias", info.OriginModelName)
		require.Equal(t, target, info.UpstreamModelName)
		require.Equal(t, "public-alias", taskBillingContext(info).OriginModelName)
	}
}

func TestVideoSubmissionMapsBeforeValidationForEveryOpenAICompatibleHost(t *testing.T) {
	info := &relaycommon.RelayInfo{OriginModelName: "opaque alias", RequestURLPath: "/v1/videos", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	for _, target := range []string{"grok-1.5-700-B", "gz-sd2.5-720p", "sd-2-c1"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		// An invalid zero duration stops before pricing, pre-consumption or any upstream request.
		c.Request = httptest.NewRequest(http.MethodPost, info.RequestURLPath, strings.NewReader(`{"model":"opaque alias","prompt":"demo","duration":0,"resolution":"720p"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://custom-proxy.example")
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "opaque alias")
		mapping, err := common.Marshal(map[string]string{"opaque alias": "intermediate", "intermediate": target})
		require.NoError(t, err)
		c.Set("model_mapping", string(mapping))
		_, taskErr := RelayTaskSubmit(c, info)
		require.NotNil(t, taskErr)
		require.Contains(t, taskErr.Message, "duration")
		require.Equal(t, target, info.UpstreamModelName)
		require.Equal(t, "opaque alias", info.OriginModelName)
		require.Equal(t, "opaque alias", taskBillingContext(info).OriginModelName)
	}
}
