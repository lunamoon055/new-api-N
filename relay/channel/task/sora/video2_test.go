package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestValidateVideo2Request(t *testing.T) {
	validDuration := 15
	asyncFalse := false
	valid := video2Request{
		Prompt:      "make a vertical product video",
		Duration:    &validDuration,
		AspectRatio: "9:16",
		Resolution:  "720p",
		Size:        "720x1280",
		ImageURLs: []string{
			"https://cdn.example/a.png",
		},
		VideoReference: []video2Reference{
			{URL: "https://cdn.example/a.mp4"},
		},
		AudioURL: "https://cdn.example/a.mp3",
		Async:    &asyncFalse,
	}
	require.NoError(t, validateVideo2Request(valid))

	validDataURLReferences := valid
	validDataURLReferences.ImageURLs = []string{"data:image/png;base64,AAAA"}
	require.NoError(t, validateVideo2Request(validDataURLReferences))

	zero := 0
	tests := []struct {
		name     string
		mutate   func(*video2Request)
		contains string
	}{
		{
			name: "duration below range",
			mutate: func(req *video2Request) {
				req.Duration = &zero
			},
			contains: "duration",
		},
		{
			name: "invalid aspect ratio",
			mutate: func(req *video2Request) {
				req.AspectRatio = "4:3"
			},
			contains: "aspect_ratio",
		},
		{
			name: "invalid resolution",
			mutate: func(req *video2Request) {
				req.Resolution = "1080p"
			},
			contains: "resolution",
		},
		{
			name: "conflicting size",
			mutate: func(req *video2Request) {
				req.Size = "1280x720"
			},
			contains: "size",
		},
		{
			name: "too many images",
			mutate: func(req *video2Request) {
				req.ImageURL = "https://cdn.example/one.png"
				req.ImageURLs = []string{
					"https://cdn.example/two.png",
					"https://cdn.example/three.png",
					"https://cdn.example/four.png",
					"https://cdn.example/five.png",
				}
			},
			contains: "image",
		},
		{
			name: "too many videos",
			mutate: func(req *video2Request) {
				req.VideoURL = "https://cdn.example/one.mp4"
				req.VideoReference = []video2Reference{
					{URL: "https://cdn.example/two.mp4"},
					{URL: "https://cdn.example/three.mp4"},
					{URL: "https://cdn.example/four.mp4"},
				}
			},
			contains: "video",
		},
		{
			name: "unsupported image format",
			mutate: func(req *video2Request) {
				req.ImageURL = "https://cdn.example/one.bmp"
			},
			contains: "image reference",
		},
		{
			name: "unsupported video format",
			mutate: func(req *video2Request) {
				req.VideoURL = "https://cdn.example/one.mov"
			},
			contains: "video reference",
		},
		{
			name: "unsupported audio format",
			mutate: func(req *video2Request) {
				req.AudioURL = "https://cdn.example/one.flac"
			},
			contains: "audio_url",
		},
		{
			name: "non http reference",
			mutate: func(req *video2Request) {
				req.AudioURL = "file:///tmp/a.mp3"
			},
			contains: "audio_url",
		},
		{
			name: "data video reference",
			mutate: func(req *video2Request) {
				req.VideoURL = "data:video/mp4;base64,AAAA"
			},
			contains: "video reference",
		},
		{
			name: "data audio reference",
			mutate: func(req *video2Request) {
				req.AudioURL = "data:audio/mpeg;base64,AAAA"
			},
			contains: "audio_url",
		},
		{
			name: "non image data reference",
			mutate: func(req *video2Request) {
				req.ImageURL = "data:text/plain;base64,AAAA"
			},
			contains: "image reference",
		},
		{
			name: "unsupported data image format",
			mutate: func(req *video2Request) {
				req.ImageURL = "data:image/bmp;base64,AAAA"
			},
			contains: "image reference",
		},
		{
			name: "blank prompt",
			mutate: func(req *video2Request) {
				req.Prompt = "  "
			},
			contains: "prompt",
		},
		{
			name: "long prompt",
			mutate: func(req *video2Request) {
				req.Prompt = strings.Repeat("a", 5001)
			},
			contains: "5000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := valid
			test.mutate(&req)
			require.ErrorContains(t, validateVideo2Request(req), test.contains)
		})
	}
}

func TestIsVideo2Model(t *testing.T) {
	require.True(t, isVideo2Model(" VIDEO-2.0 "))
	require.True(t, isVideo2Model("video-2.0-fast"))
	require.False(t, isVideo2Model("sora2"))
}

func newVideo2JSONContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/video/async-generations",
		strings.NewReader(body),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c
}

func TestVideo2ValidationAndBodyPassThrough(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"video-2.0-fast",
		"prompt":"demo",
		"duration":4,
		"aspect_ratio":"1:1",
		"resolution":"720p",
		"async":false,
		"image_urls":["https://cdn.example/one.png"],
		"video_reference":[{"url":"https://cdn.example/one.mp4"}],
		"audio_url":"https://cdn.example/one.mp3"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.0-fast",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "mapped-video-2.0-fast",
		},
	}
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))

	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(encoded, &got))
	require.Equal(t, "mapped-video-2.0-fast", got["model"])
	require.Equal(t, false, got["async"])
	require.Equal(t, float64(4), got["duration"])
	require.NotNil(t, got["image_urls"])
	require.NotNil(t, got["video_reference"])
	require.Equal(t, "https://cdn.example/one.mp3", got["audio_url"])
}

func TestVideo2ValidationRejectsInvalidDuration(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"video-2.0",
		"prompt":"demo",
		"duration":0
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-2.0",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)

	require.NotNil(t, taskErr)
	require.Equal(t, "invalid_request", taskErr.Code)
	require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	require.ErrorContains(t, taskErr.Error, "duration")
}

func TestNonVideo2ModelsSkipDedicatedValidation(t *testing.T) {
	c := newVideo2JSONContext(t, `{
		"model":"kling-v3",
		"prompt":"demo",
		"duration":0,
		"resolution":"1080p",
		"audio_url":"file:///tmp/audio.mp3"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "kling-v3",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info))
}
