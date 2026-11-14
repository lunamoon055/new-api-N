package v6k

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/stretchr/testify/require"
)

func referenceVideoFixture(seconds float64) []byte {
	box := func(id, payload []byte) []byte {
		result := append([]byte{}, id...)
		result = append(result, byte(len(payload))|0x80)
		return append(result, payload...)
	}
	duration := make([]byte, 8)
	binary.BigEndian.PutUint64(duration, math.Float64bits(seconds*1000))
	return append(box([]byte{0x1a, 0x45, 0xdf, 0xa3}, nil), box([]byte{0x18, 0x53, 0x80, 0x67}, box([]byte{0x15, 0x49, 0xa9, 0x66}, box([]byte{0x44, 0x89}, duration)))...)
}

func configureReferenceVideoTestPrices(t *testing.T, modelNames ...string) {
	modes, _ := common.Marshal(billing_setting.GetVideoBillingModeCopy())
	prices, _ := common.Marshal(billing_setting.GetVideoReferencePricesCopy())
	t.Cleanup(func() {
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, string(modes))
		_ = billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, string(prices))
	})
	if len(modelNames) == 0 {
		modelNames = []string{"alias"}
	}
	modelModes := map[string]string{}
	modelPrices := map[string]map[string]float64{}
	for _, name := range modelNames {
		modelModes[name] = "tiered_seconds"
		modelPrices[name] = map[string]float64{"720p": 0.1, "4k": 0.9}
	}
	encodedModes, err := common.Marshal(modelModes)
	require.NoError(t, err)
	encodedPrices, err := common.Marshal(modelPrices)
	require.NoError(t, err)
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoBillingModeField, string(encodedModes)))
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, string(encodedPrices)))
}

func TestConfiguredReferenceVideosWorkWithoutMappingsAndWithCustomMappedTargets(t *testing.T) {
	for _, tc := range []struct {
		name, mapping string
		omitUpstream  bool
	}{
		{name: "SD2.0"},
		{name: "SD2.5", mapping: "{}"},
		{name: "Seedance2.0:(无视频参考)"},
		{name: "custom-native-video", omitUpstream: true},
		{name: "mapped-sd", mapping: `{"mapped-sd":"SD2.5"}`},
		{name: "mapped-custom", mapping: `{"mapped-custom":"intermediate","intermediate":"custom-upstream-video"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configureReferenceVideoTestPrices(t, tc.name)
			upstream, _, err := common.ResolveModelMapping(tc.name, tc.mapping)
			require.NoError(t, err)
			var gets, posts atomic.Int32
			fixture := referenceVideoFixture(7.5)
			origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets.Add(1)
					require.Empty(t, r.Header.Get("Authorization"))
					_, _ = w.Write(fixture)
					return
				}
				posts.Add(1)
				require.Equal(t, "/v1/videos", r.URL.Path)
				require.NoError(t, r.ParseMultipartForm(1<<20))
				defer r.MultipartForm.RemoveAll()
				require.Equal(t, upstream, r.FormValue("model"))
				require.Len(t, r.MultipartForm.File["video"], 1)
				fmt.Fprint(w, `{"id":"local-fixture-task","status":"queued"}`)
			})
			body := fmt.Sprintf(`{"model":%q,"prompt":"demo","duration":5,"resolution":"720p","referenceVideos":[{"url":%q,"duration_seconds":0.1}]}`, tc.name, origin+"/clip.webm")
			c, _, info, a := testRequest(t, body, origin)
			info.OriginModelName, info.UpstreamModelName = tc.name, upstream
			if tc.omitUpstream {
				info.UpstreamModelName = ""
			}
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			require.InDelta(t, 7.5, c.GetFloat64("v6k_reference_video_seconds"), 1e-9)
			forwarded, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			response, err := a.DoRequest(c, info, forwarded)
			require.NoError(t, err)
			response.Body.Close()
			require.EqualValues(t, 1, gets.Load())
			require.EqualValues(t, 1, posts.Load())
		})
	}
}

func TestReferenceVideoURLsAreMeasuredAndExactBytesAreForwardedOnce(t *testing.T) {
	configureReferenceVideoTestPrices(t)
	var gets, posts atomic.Int32
	fixtures := [][]byte{referenceVideoFixture(2.5), referenceVideoFixture(5)}
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/clip") {
			gets.Add(1)
			require.Empty(t, r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			index := 0
			if r.URL.Path == "/clip2.webm" {
				index = 1
			}
			_, _ = w.Write(fixtures[index])
			return
		}
		posts.Add(1)
		require.Equal(t, "/v1/videos", r.URL.Path)
		require.Equal(t, "Bearer provider-test", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		defer r.MultipartForm.RemoveAll()
		require.Equal(t, "SD2.5", r.FormValue("model"))
		require.Equal(t, "720p", r.FormValue("resolution"))
		require.Equal(t, "5", r.FormValue("seconds"))
		require.Empty(t, r.FormValue("videos"))
		require.Len(t, r.MultipartForm.File["video"], 2)
		for i, file := range r.MultipartForm.File["video"] {
			reader, err := file.Open()
			require.NoError(t, err)
			data, err := io.ReadAll(reader)
			reader.Close()
			require.NoError(t, err)
			require.True(t, bytes.Equal(fixtures[i], data))
		}
		fmt.Fprint(w, `{"id":"local-fixture-task","status":"queued"}`)
	})
	body := fmt.Sprintf(`{"model":"alias","prompt":"demo","duration":5,"resolution":"720p","referenceVideos":[{"url":%q,"duration_seconds":0.1},%q]}`, origin+"/clip1.webm", origin+"/clip2.webm")
	c, _, info, a := testRequest(t, body, origin)
	info.UpstreamModelName = "SD2.5"
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.InDelta(t, 7.5, c.GetFloat64("v6k_reference_video_seconds"), 1e-9)
	forwarded, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	response, err := a.DoRequest(c, info, forwarded)
	require.NoError(t, err)
	response.Body.Close()
	require.EqualValues(t, 2, gets.Load())
	require.EqualValues(t, 1, posts.Load())
	_, err = a.DoRequest(c, info, strings.NewReader("{}"))
	require.Error(t, err)
	require.EqualValues(t, 1, posts.Load())
}

func TestSDResolutionAndUnverifiableReferencesAreRejectedBeforeSubmit(t *testing.T) {
	configureReferenceVideoTestPrices(t)
	for _, body := range []string{`{"model":"alias","prompt":"demo","duration":5,"resolution":"4K"}`, `{"model":"alias","prompt":"demo","duration":5}`, `{"model":"alias","prompt":"demo","duration":5,"resolution":"720p","metadata":{"reference_video_seconds":0}}`} {
		c, _, info, a := testRequest(t, body, "https://upstream.example")
		info.UpstreamModelName = "SD2.5"
		require.NotNil(t, a.ValidateRequestAndSetAction(c, info))
		require.False(t, c.GetBool("v6k_submission_attempted"))
	}
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>access denied</html>") })
	c, _, info, a := testRequest(t, fmt.Sprintf(`{"model":"alias","prompt":"demo","duration":5,"resolution":"720p","videos":[%q]}`, origin+"/clip.webm"), "https://upstream.example")
	info.UpstreamModelName = "SD2.0"
	require.NotNil(t, a.ValidateRequestAndSetAction(c, info))
	require.False(t, c.GetBool("v6k_submission_attempted"))
	require.Zero(t, c.GetFloat64("v6k_reference_video_seconds"))
	// Existing models retain their previous reference handling and incur no new
	// download or reference-second fees without an explicit price configuration.
	require.NoError(t, billing_setting.ApplyAuxiliaryConfig(billing_setting.VideoReferencePricesField, `{}`))
	c, _, info, a = testRequest(t, `{"model":"alias","prompt":"demo","duration":5,"resolution":"720p","videos":["https://cdn.example/v.mp4"]}`, "https://upstream.example")
	info.UpstreamModelName = "TTseadanco2.0-720"
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Zero(t, c.GetFloat64("v6k_reference_video_seconds"))
}
