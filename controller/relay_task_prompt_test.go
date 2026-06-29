package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAttachTaskPromptFromRequestStoresPromptInTaskProperties(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt: "a cinematic product video",
	})
	task := &model.Task{}

	attachTaskPromptFromRequest(ctx, task)

	require.Equal(t, "a cinematic product video", task.Properties.Input)
}
