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
		Prompt:          "a cinematic product video",
		ReferenceImages: []string{"https://cdn.example.com/reference.png"},
		ReferenceVideos: []string{"https://cdn.example.com/reference.mp4"},
		ReferenceAudios: []string{"https://cdn.example.com/reference.wav"},
	})
	task := &model.Task{}

	attachTaskPromptFromRequest(ctx, task)

	require.Equal(t, "a cinematic product video", task.Properties.Input)
	require.Equal(t, []string{"https://cdn.example.com/reference.png"}, task.Properties.InputImages)
	require.Equal(t, []string{"https://cdn.example.com/reference.mp4"}, task.Properties.InputVideos)
	require.Equal(t, []string{"https://cdn.example.com/reference.wav"}, task.Properties.InputAudios)
}
