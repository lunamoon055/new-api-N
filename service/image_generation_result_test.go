package service

import (
	"encoding/base64"
	"io"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCaptureImageGenerationDataStoresBase64AndKeepsRemoteURLs(t *testing.T) {
	t.Setenv("IMAGE_RESULT_STORAGE_PATH", t.TempDir())
	ctx, _ := gin.CreateTestContext(nil)

	const pixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	urls := CaptureImageGenerationData(ctx, []dto.ImageData{
		{Url: "https://cdn.example/result.png"},
		{B64Json: pixelPNG},
		{Url: "javascript:alert(1)"},
	})

	require.Len(t, urls, 2)
	require.Equal(t, "https://cdn.example/result.png", urls[0])
	require.Regexp(t, `^/api/image-results/.+\.png$`, urls[1])
	require.Equal(t, urls, GetCapturedImageGenerationResultURLs(ctx))

	filename := urls[1][len("/api/image-results/"):]
	file, mimeType, size, err := OpenImageGenerationResult(filename)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	require.Equal(t, "image/png", mimeType)
	decoded, err := base64.StdEncoding.DecodeString(pixelPNG)
	require.NoError(t, err)
	require.Equal(t, int64(len(decoded)), size)
	stored, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, decoded, stored)
}

func TestBuildImageGenerationLogIncludesAllPreviewURLs(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set("X-Oneapi-Request-Id", "request-from-context")
	now := time.Unix(1_800_000_010, 0)
	start := time.Unix(1_800_000_000, 0)
	info := &relaycommon.RelayInfo{
		UserId:          9,
		StartTime:       start,
		OriginModelName: "gpt-image-2",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 29},
	}
	request := &dto.ImageRequest{
		Prompt:  "draw a lighthouse",
		Size:    "2048x2048",
		Quality: "high",
	}
	imageURLs := []string{"https://cdn.example/one.png", "/api/image-results/two.png"}

	task, err := buildImageGenerationLog(ctx, info, request, imageURLs, now)
	require.NoError(t, err)
	require.Equal(t, constant.MjActionImageGeneration, task.Action)
	require.Equal(t, "request-from-context", task.MjId)
	require.Equal(t, 9, task.UserId)
	require.Equal(t, 29, task.ChannelId)
	require.Equal(t, "https://cdn.example/one.png", task.ImageUrl)
	require.Equal(t, "100%", task.Progress)
	require.Equal(t, "SUCCESS", task.Status)
	require.Equal(t, start.UnixMilli(), task.SubmitTime)
	require.Equal(t, now.UnixMilli(), task.FinishTime)
	require.Contains(t, task.Properties, `"image_urls":["https://cdn.example/one.png","/api/image-results/two.png"]`)
}
