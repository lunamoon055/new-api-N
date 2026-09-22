package sora

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/stretchr/testify/require"
)

func TestAsyncImageMappedAliasesKeepPublicModelAndForwardExactTarget(t *testing.T) {
	for _, target := range []string{"gpt-image-2", "gpt-image-2.5", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "nano-banana-pro", "nano-banana2", "seedream-5-0"} {
		for _, name := range []string{target, "自定义公开别名", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
			t.Run(target+"/"+name, func(t *testing.T) {
				encoded, err := common.Marshal(map[string]any{"model": name, "prompt": "a garden portrait", "aspect_ratio": "9:16", "n": 1})
				require.NoError(t, err)
				c := newVideo2JSONContext(t, string(encoded))
				if name != target {
					mapping, err := common.Marshal(map[string]string{name: "intermediate", "intermediate": target})
					require.NoError(t, err)
					c.Set("model_mapping", string(mapping))
				}
				info := &relaycommon.RelayInfo{
					OriginModelName: name, RequestURLPath: "/v1/images/async-generations",
					TaskRelayInfo: &relaycommon.TaskRelayInfo{},
					ChannelMeta:   &relaycommon.ChannelMeta{ChannelBaseUrl: "https://linksky.top", UpstreamModelName: name},
				}
				require.NoError(t, helper.ModelMappedHelper(c, info, nil))
				adaptor := &TaskAdaptor{baseURL: "https://linksky.top"}
				require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
				require.Equal(t, name, info.OriginModelName)
				req, err := relaycommon.GetTaskRequest(c)
				require.NoError(t, err)
				require.Equal(t, name, req.Model)
				body, err := adaptor.BuildRequestBody(c, info)
				require.NoError(t, err)
				data, err := io.ReadAll(body)
				require.NoError(t, err)
				var payload map[string]any
				require.NoError(t, common.Unmarshal(data, &payload))
				require.Equal(t, target, payload["model"])
				wantResolution := "2K"
				if target == "nano-banana-pro" || target == "nano-banana2" {
					wantResolution = "1K"
				}
				require.Equal(t, wantResolution, payload["output_resolution"])
				url, err := adaptor.BuildRequestURL(info)
				require.NoError(t, err)
				require.Equal(t, "https://linksky.top/v1/images/async-generations", url)
			})
		}
	}
}

func TestAsyncImageCapabilityFollowsMappedTargetWithoutFallback(t *testing.T) {
	for _, target := range []string{"seedream-5-0", "gpt-image-2", "not-a-real-model"} {
		c := newVideo2JSONContext(t, `{"model":"gpt-image-2","prompt":"garden portrait","output_resolution":"3K"}`)
		info := &relaycommon.RelayInfo{
			OriginModelName: "gpt-image-2", RequestURLPath: "/v1/images/async-generations",
			TaskRelayInfo: &relaycommon.TaskRelayInfo{},
			ChannelMeta:   &relaycommon.ChannelMeta{ChannelBaseUrl: "https://linksky.top", UpstreamModelName: target},
		}
		err := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
		if target == "seedream-5-0" {
			require.Nil(t, err)
		} else {
			require.NotNil(t, err)
		}
		require.Equal(t, "gpt-image-2", info.OriginModelName)
	}
}
