package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageRetryFilterKeepsStandardOptionsOnCompatibleChannels(t *testing.T) {
	mega := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc"), ModelMapping: common.GetPointer(`{"public-alias":"gpt-image-2.5-sunburst"}`)}
	xjz := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://api.xjzapi.cc"), ModelMapping: common.GetPointer(`{"public-alias":"gpt-image-2.5-sunburst-原生4K"}`)}
	for _, mode := range []int{relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(`{"model":"public-alias","n":2,"quality":"max","output_compression":0}`))
		c.Request.Header.Set("Content-Type", "application/json")
		filter := relayTaskChannelFilter(&relaycommon.RelayInfo{OriginModelName: "public-alias", RelayMode: mode}, c)
		require.True(t, filter(mega))
		require.False(t, filter(xjz), "retry must not silently drop standard options")
	}
}

func TestMegaBYAIImageRetryKeepsEndpointAndMappedModelScope(t *testing.T) {
	mapping := `{"public-alias":"intermediate","intermediate":"gpt-image-2.5-sunburst"}`
	mega := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc"), ModelMapping: &mapping}
	xjz := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://api.xjzapi.cc"), ModelMapping: &mapping}
	linksky := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://linksky.top"), ModelMapping: &mapping}
	for _, mode := range []int{relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits} {
		filter := relayTaskChannelFilter(&relaycommon.RelayInfo{OriginModelName: "public-alias", RelayMode: mode})
		require.NotNil(t, filter)
		require.True(t, filter(mega))
		require.True(t, filter(xjz))
		require.False(t, filter(linksky))
		require.False(t, filter(nil))
		bad := *mega
		bad.ModelMapping = common.GetPointer(`{"public-alias":"other","other":"public-alias"}`)
		require.False(t, filter(&bad))
	}
}

func TestMegaBYAIImageChannelTestUsesPublishedMinimumOnlyForDefaults(t *testing.T) {
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		channel := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc")}
		for _, defaultRequest := range []bool{true, false} {
			request := &dto.ImageRequest{Model: name, Prompt: "blue cup", N: common.GetPointer(uint(1)), Size: "1024x1024"}
			applyImageChannelTestDefaults(channel, request, defaultRequest)
			if defaultRequest {
				require.Empty(t, request.Size)
			} else {
				require.Equal(t, "1024x1024", request.Size)
			}
			require.Equal(t, uint(1), *request.N)
		}
	}
}
