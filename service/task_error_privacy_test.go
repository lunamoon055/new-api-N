package service

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

const privateUpstreamVideoModel = "优质特价/sd2.5(稳定版30图原生不卡脸5/10/30秒)"
const publicChannelUnavailableMessage = "当前模型服务暂不可用，请稍后重试或联系管理员。"

func upstreamChannelUnavailableError(t *testing.T) string {
	t.Helper()
	payload, err := common.Marshal(map[string]any{
		"code":    "fail_to_fetch_task",
		"message": "模型通道不可用：模型 " + privateUpstreamVideoModel + " 在当前令牌分组中没有可用渠道；请检查模型名、令牌分组权限或联系管理员检查渠道映射 (request id: upstream-private-id)",
		"data":    nil,
	})
	require.NoError(t, err)
	return string(payload)
}

func TestTranslateVideoTaskErrorHidesUpstreamChannelDetails(t *testing.T) {
	for _, raw := range []string{
		upstreamChannelUnavailableError(t),
		"分组 provider-vip 下模型 " + privateUpstreamVideoModel + " 无可用渠道（distributor）",
		"No available channel for model private-provider-model under group provider-vip (distributor)",
		`{"error":{"code":"model_not_found","message":"no available channel for private-provider-model"}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			translated := TranslateVideoTaskError(raw, "fail_to_fetch_task", http.StatusBadRequest)
			require.Equal(t, publicChannelUnavailableMessage, translated.Message)
			require.Equal(t, "configuration", translated.Category)
			require.True(t, translated.Retryable)
		})
	}
}

func TestLocalizeVideoTaskErrorKeepsChannelDiagnosticsServerOnly(t *testing.T) {
	raw := upstreamChannelUnavailableError(t)
	taskErr := &dto.TaskError{Code: "fail_to_fetch_task", Message: raw, StatusCode: http.StatusBadRequest}
	LocalizeVideoTaskError(taskErr)
	require.Equal(t, publicChannelUnavailableMessage, taskErr.Message)
	require.Contains(t, taskErr.RawMessage, privateUpstreamVideoModel)
	payload, err := common.Marshal(taskErr)
	require.NoError(t, err)
	require.NotContains(t, string(payload), privateUpstreamVideoModel)
	require.NotContains(t, string(payload), "upstream-private-id")
	require.Contains(t, string(payload), `"code":"fail_to_fetch_task"`)
}

func TestVideoTaskFailureMessagesRedactsHistoricalChannelErrors(t *testing.T) {
	raw := upstreamChannelUnavailableError(t)
	for _, source := range []string{"private", "data", "reason"} {
		t.Run(source, func(t *testing.T) {
			task := &model.Task{Status: model.TaskStatusFailure}
			switch source {
			case "private":
				task.PrivateData.UpstreamError = raw
			case "data":
				task.Data = []byte(raw)
			case "reason":
				// Historical Chinese FailReason bypassed translation entirely.
				task.FailReason = "模型通道不可用：模型 " + privateUpstreamVideoModel + " 在当前令牌分组中没有可用渠道"
			}
			message, privateRaw := VideoTaskFailureMessages(task)
			require.Equal(t, publicChannelUnavailableMessage, message)
			require.Contains(t, privateRaw, privateUpstreamVideoModel)
		})
	}
}

func TestSetVideoTaskFailureDoesNotExposeMappedModelInOtherReadableErrors(t *testing.T) {
	raw := "模型 " + privateUpstreamVideoModel + " 的参考图片数量超过 30 张，请减少图片后重试"
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		Properties: model.Properties{OriginModelName: "Seedance-public", UpstreamModelName: privateUpstreamVideoModel},
	}
	SetVideoTaskFailure(task, raw, "fail_to_fetch_task", http.StatusBadRequest)
	require.NotContains(t, task.FailReason, privateUpstreamVideoModel)
	require.Contains(t, task.FailReason, "参考图片数量超过 30 张")
	require.Equal(t, raw, task.PrivateData.UpstreamError)
	message, privateRaw := VideoTaskFailureMessages(task)
	require.Equal(t, task.FailReason, message)
	require.Equal(t, raw, privateRaw)
}
