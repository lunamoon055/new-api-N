package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSiteModelTestUsesPublicVideoContract(t *testing.T) {
	for _, name := range []string{"无限制-Video", "Seedance2.0-c", "Seedance2.0-fast-c", "Seedance2.0-fast2-c", "Seedance2.0:720（秒）", "Seedance2.5:480（秒）", "Seedance2.5:720（秒）", "Seedance2.5:720（条）"} {
		ch := &model.Channel{Type: constant.ChannelTypeOpenAI}
		_, path, _ := resolveChannelTestEndpoint(ch, name, "")
		require.Equal(t, "/v1/videos", path, name)
		request := buildTestRequest(name, "openai-video", ch, false).(relaycommon.TaskSubmitReq)
		p := common.GetSiteVideoIntegrationParameters(name)
		require.Contains(t, p.Durations, request.Duration, name)
		require.Equal(t, name, request.Model)
		require.Empty(t, request.Size)
	}
}
