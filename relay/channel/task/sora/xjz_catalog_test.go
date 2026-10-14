package sora

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

type documentedXJZModel struct {
	ID              string   `json:"id"`
	Durations       []int    `json:"durations"`
	Resolutions     []string `json:"resolutions"`
	Ratios          []string `json:"ratios"`
	RatioField      string   `json:"ratio_field"`
	ResolutionField string   `json:"resolution_field"`
	ImageField      string   `json:"image_field"`
	VideoField      string   `json:"video_field"`
	AudioField      string   `json:"audio_field"`
	MaxImages       int      `json:"max_images"`
	MaxVideos       int      `json:"max_videos"`
	MaxAudios       int      `json:"max_audios"`
	PerCall         bool     `json:"per_call"`
}

func TestXJZCurrentCatalogAndAppendixContracts(t *testing.T) {
	data, err := os.ReadFile("../../../../common/testdata/xjz-video-models-2026-10-14.json")
	require.NoError(t, err)
	var fixture struct {
		Models []documentedXJZModel `json:"models"`
	}
	require.NoError(t, common.Unmarshal(data, &fixture))
	require.Len(t, fixture.Models, 25)
	for _, doc := range fixture.Models {
		t.Run(doc.ID, func(t *testing.T) {
			require.True(t, common.IsXJZVideoModel(doc.ID))
			require.Equal(t, doc.PerCall, common.IsXJZPerCallVideoModel(doc.ID))
			parameters := common.GetVideoIntegrationParameters(doc.ID)
			require.Equal(t, doc.Durations, parameters.Durations)
			require.Equal(t, doc.RatioField, parameters.RatioField)
			require.Equal(t, doc.ResolutionField, parameters.ResolutionField)
			require.Equal(t, doc.Ratios, parameters.Ratios)
			require.Equal(t, doc.ImageField, parameters.ImageField)
			require.Equal(t, doc.VideoField, parameters.VideoField)
			require.Equal(t, doc.AudioField, parameters.AudioField)
			require.Equal(t, doc.MaxImages, parameters.MaxImages)
			require.Equal(t, doc.MaxVideos, parameters.MaxVideos)
			require.Equal(t, doc.MaxAudios, parameters.MaxAudios)

			// Check every resolution, duration boundary, and ratio against the
			// actual upstream request builder, retaining a caller's public alias.
			for _, resolution := range doc.Resolutions {
				for _, duration := range []int{doc.Durations[0], doc.Durations[len(doc.Durations)-1]} {
					for _, ratio := range doc.Ratios {
						c := newVideo2JSONContext(t, fmt.Sprintf(`{"model":"public-alias","prompt":"a product video","duration":%d,%q:%q,%q:%q,"generate_audio":false}`, duration, doc.RatioField, ratio, doc.ResolutionField, resolution))
						info := &relaycommon.RelayInfo{OriginModelName: "public-alias", RequestURLPath: "/v1/video/async-generations", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: doc.ID}}
						adaptor := &TaskAdaptor{baseURL: "https://api.xjzapi.cc/v1"}
						require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
						requestURL, err := adaptor.BuildRequestURL(info)
						require.NoError(t, err)
						require.Equal(t, "https://api.xjzapi.cc/v1/videos", requestURL)
						body, err := adaptor.BuildRequestBody(c, info)
						require.NoError(t, err)
						encoded, err := io.ReadAll(body)
						require.NoError(t, err)
						var payload map[string]any
						require.NoError(t, common.Unmarshal(encoded, &payload))
						require.Equal(t, doc.ID, payload["model"])
						require.Equal(t, float64(duration), payload["duration"])
						require.Equal(t, resolution, payload[doc.ResolutionField])
						require.Equal(t, ratio, payload[doc.RatioField])
						require.Equal(t, false, payload["generate_audio"])
						require.NotContains(t, payload, "seconds")
						if doc.RatioField == "ratio" {
							require.NotContains(t, payload, "aspect_ratio")
						} else {
							require.NotContains(t, payload, "ratio")
						}
						if doc.ResolutionField == "resolution" {
							require.NotContains(t, payload, "size")
						} else {
							require.NotContains(t, payload, "resolution")
						}
						require.Equal(t, float64(duration), adaptor.EstimateBilling(c, info)["seconds"])
					}
				}
			}
		})
	}
}

func TestXJZNewModelLimitsAndDefaults(t *testing.T) {
	for _, tc := range []struct {
		model   string
		request string
		want    string
	}{
		{"grok-1.5-700", `{"model":"grok-1.5-700","prompt":"demo","duration":2}`, "duration"},
		{"TTP-grok", `{"model":"TTP-grok","prompt":"demo","duration":3,"resolution":"720p"}`, "duration"},
		{"sd2.5-全参过真人-30-10-10", `{"model":"x","prompt":"demo","duration":4,"resolution":"720p"}`, "duration"},
		{"gz-sd2.5-480p", `{"model":"x","prompt":"demo","duration":5,"audio_url":"https://cdn.example/a.mp3"}`, "audio references are unsupported"},
		{"gz-sd720p", `{"model":"x","prompt":"demo","duration":5,"reference_videos":["https://cdn.example/a.mp4"],"audio_url":"https://cdn.example/a.mp3"}`, "require an image reference"},
		{"viraldance933-fast", `{"model":"x","prompt":"demo","duration":5,"audio_url":"https://cdn.example/a.mp3"}`, "require an image or video"},
		{"viraldance933x-fast", `{"model":"x","prompt":"demo","duration":15}`, "duration"},
		{"viraldance933x-fast", `{"model":"x","prompt":"demo","duration":5,"image_urls":["https://cdn.example/a.jpg","https://cdn.example/b.jpg"]}`, "image references"},
		{"viraldance2.5x-30", `{"model":"x","prompt":"demo","duration":5,"video_url":"https://cdn.example/a.mp4"}`, "video references are unsupported"},
		{"viraldance900", `{"model":"x","prompt":"demo","duration":5,"size":"720p","resolution":"480p"}`, "conflicts"},
	} {
		t.Run(tc.model+tc.want, func(t *testing.T) {
			var payload map[string]any
			require.NoError(t, common.Unmarshal([]byte(tc.request), &payload))
			_, _, _, _, _, err := normalizeXJZVideoPayload(payload, tc.model)
			require.ErrorContains(t, err, tc.want)
		})
	}
	for name, duration := range map[string]int{"sd2.5-720p-ch1": 30, "quanneng2.0": 15, "viraldance2.5-15-stable": 15, "viraldance2.5-480p-15-stable": 15} {
		_, got, _, _, _, err := normalizeXJZVideoPayload(map[string]any{"model": name, "prompt": "demo"}, name)
		require.NoError(t, err)
		require.Equal(t, duration, got)
	}
}

func TestXJZSingularReferencesAndReferenceVideoBilling(t *testing.T) {
	payload, _, images, videos, audios, err := normalizeXJZVideoPayload(map[string]any{
		"model": "viraldance933x-fast", "prompt": "demo", "duration": 5,
		"images": []string{"https://cdn.example/a.jpg"}, "videos": []string{"https://cdn.example/a.mp4"}, "audios": []string{"https://cdn.example/a.mp3"},
	}, "viraldance933x-fast")
	require.NoError(t, err)
	require.Equal(t, []int{1, 1, 1}, []int{images, videos, audios})
	require.Equal(t, "https://cdn.example/a.jpg", payload["image_url"])
	require.Equal(t, "https://cdn.example/a.mp4", payload["video_url"])
	require.Equal(t, "https://cdn.example/a.mp3", payload["audio_url"])

	for _, name := range []string{"gz-sd480p", "gz-sd720p", "gz-sd1080p", "gz-sd4k", "sd2.5-全参过真人-30-10-10", "特价sd2.0-720p-933-过脸"} {
		for count := 0; count <= 3; count++ {
			urls := make([]string, count)
			for i := range urls {
				urls[i] = fmt.Sprintf("https://cdn.example/%d.mp4", i)
			}
			body, err := common.Marshal(map[string]any{"model": name, "prompt": "demo", "duration": 8, "resolution": common.XJZVideoSpecs[name].Resolution, "reference_videos": urls})
			require.NoError(t, err)
			if name == "sd2.5-全参过真人-30-10-10" {
				body, err = common.Marshal(map[string]any{"model": name, "prompt": "demo", "duration": 8, "resolution": "720p", "reference_videos": urls})
				require.NoError(t, err)
			}
			c := newVideo2JSONContext(t, string(body))
			info := &relaycommon.RelayInfo{OriginModelName: name, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
			ratios := adaptor.EstimateBilling(c, info)
			require.Equal(t, float64(8), ratios["seconds"])
			if count > 0 && common.XJZVideoSpecs[name].ReferenceVideoMultiplier > 1 {
				require.Equal(t, float64(2), ratios["reference_video"])
			} else {
				require.NotContains(t, ratios, "reference_video")
			}
		}
	}
}

func TestXJZNewModelsKeepTransientUnknownPending(t *testing.T) {
	for _, name := range []string{"grok-1.5-700", "gz-sd2.5-720p", "viraldance933x-fast"} {
		body, err := common.Marshal(map[string]any{"id": "accepted-task", "model": name, "status": "unknown", "progress": 0})
		require.NoError(t, err)
		result, err := (&TaskAdaptor{}).ParseTaskResult(body)
		require.NoError(t, err)
		require.Equal(t, "QUEUED", result.Status)
	}
}
