package sora

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsVideosModelName(t *testing.T) {
	require.True(t, isVideosModelName(" videos-standard "))
	require.True(t, isVideosModelName("videos-fast"))
	require.True(t, isVideosModelName("videos-mini"))
	require.True(t, isVideosModelName("sd2-mini"))
	require.True(t, isVideosModelName("sd2-fast"))
	require.True(t, isVideosModelName("sd2满血"))
	require.False(t, isVideosModelName("video-2.0-fast"))
	require.False(t, isVideosModelName("sora2"))
}

func TestValidateVideosRequest(t *testing.T) {
	validDuration := 15
	valid := videosRequest{
		Prompt: "make a vertical product video",
		Duration: &validDuration,
		Ratio: "9:16",
		Resolution: "720p",
		ReferenceImages: []string{"https://cdn.example/a.png"},
		ReferenceVideos: []string{"https://cdn.example/a.mp4"},
		ReferenceAudios: []string{"https://cdn.example/a.mp3"},
	}
	require.NoError(t, validateVideosRequest(valid))

	noDuration := valid
	noDuration.Duration = nil
	require.NoError(t, validateVideosRequest(noDuration))

	zero := 0
	tests := []struct {
		name     string
		mutate   func(*videosRequest)
		contains string
	}{
		{
			name: "duration below range",
			mutate: func(req *videosRequest) {
				req.Duration = &zero
			},
			contains: "duration",
		},
		{
			name: "invalid ratio",
			mutate: func(req *videosRequest) {
				req.Ratio = "4:3"
			},
			contains: "ratio",
		},
		{
			name: "invalid resolution",
			mutate: func(req *videosRequest) {
				req.Resolution = "1080p"
			},
			contains: "resolution",
		},
		{
			name: "first image unsupported",
			mutate: func(req *videosRequest) {
				req.FirstImage = "https://cdn.example/first.png"
			},
			contains: "first_image",
		},
		{
			name: "too many images",
			mutate: func(req *videosRequest) {
				req.ReferenceImages = []string{
					"https://cdn.example/1.png",
					"https://cdn.example/2.png",
					"https://cdn.example/3.png",
					"https://cdn.example/4.png",
					"https://cdn.example/5.png",
					"https://cdn.example/6.png",
					"https://cdn.example/7.png",
					"https://cdn.example/8.png",
					"https://cdn.example/9.png",
					"https://cdn.example/10.png",
				}
			},
			contains: "image references",
		},
		{
			name: "too many videos",
			mutate: func(req *videosRequest) {
				req.ReferenceVideos = []string{
					"https://cdn.example/1.mp4",
					"https://cdn.example/2.mp4",
					"https://cdn.example/3.mp4",
					"https://cdn.example/4.mp4",
				}
			},
			contains: "video references",
		},
		{
			name: "too many audios",
			mutate: func(req *videosRequest) {
				req.ReferenceAudios = []string{
					"https://cdn.example/1.mp3",
					"https://cdn.example/2.mp3",
					"https://cdn.example/3.mp3",
					"https://cdn.example/4.mp3",
				}
			},
			contains: "audio references",
		},
		{
			name: "unsupported reference url",
			mutate: func(req *videosRequest) {
				req.ReferenceAudios = []string{"file:///tmp/a.mp3"}
			},
			contains: "referenceAudios",
		},
		{
			name: "blank prompt",
			mutate: func(req *videosRequest) {
				req.Prompt = "  "
			},
			contains: "prompt",
		},
		{
			name: "long prompt",
			mutate: func(req *videosRequest) {
				req.Prompt = strings.Repeat("a", 5001)
			},
			contains: "5000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := valid
			test.mutate(&req)
			require.ErrorContains(t, validateVideosRequest(req), test.contains)
		})
	}
}

func newVideosJSONContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/videos",
		strings.NewReader(body),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c
}

func TestVideosValidationAndBodyPassThrough(t *testing.T) {
	c := newVideosJSONContext(t, `{
		"model":"sd2-mini",
		"prompt":"demo",
		"duration":5,
		"ratio":"16:9",
		"resolution":"720p",
		"referenceImages":["https://cdn.example/one.png"],
		"referenceVideos":["https://cdn.example/one.mp4"],
		"referenceAudios":["https://cdn.example/one.mp3"]
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "sd2-mini",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "videos-mini",
		},
	}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info))

	req, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	require.Equal(t, "sd2-mini", req.Model)
	require.Equal(t, 5, req.Duration)
	require.Equal(t, "16:9", req.Ratio)
	require.Equal(t, "720p", req.Resolution)
	require.Len(t, req.ReferenceImages, 1)
	require.Len(t, req.ReferenceVideos, 1)
	require.Len(t, req.ReferenceAudios, 1)
}

func TestVideosValidationRejectsUnsupportedFirstAndLastImage(t *testing.T) {
	c := newVideosJSONContext(t, `{
		"model":"videos-mini",
		"prompt":"demo",
		"duration":5,
		"ratio":"16:9",
		"resolution":"720p",
		"first_image":"https://cdn.example/first.png",
		"last_image":"https://cdn.example/last.png"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "videos-mini",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "videos-mini",
		},
	}

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
	require.NotNil(t, taskErr)
	require.Equal(t, "invalid_request", taskErr.Code)
	require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	require.ErrorContains(t, taskErr.Error, "first_image")
}

func TestVideosEstimateBillingDefaultsToFiveSeconds(t *testing.T) {
	c := newVideosJSONContext(t, `{
		"model":"videos-mini",
		"prompt":"demo",
		"ratio":"16:9",
		"resolution":"720p"
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "videos-mini",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "videos-mini",
		},
	}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info))

	ratios := (&TaskAdaptor{}).EstimateBilling(c, info)
	require.Equal(t, 5.0, ratios["seconds"])
}
