package v6k

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

// A public IP in the request exercises the actual SSRF transport; only its
// socket is routed to the local fixture. No test downloads or generates media
// on the internet, and private redirect targets must still be rejected.
func referenceAudioTestOrigin(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service.InitHttpClient()
	client := service.GetHttpClient()
	oldTransport := client.Transport
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = transport
	settings := system_setting.GetFetchSetting()
	oldSettings := *settings
	settings.EnableSSRFProtection = true
	settings.AllowPrivateIp = false
	settings.DomainFilterMode, settings.IpFilterMode = false, false
	settings.DomainList, settings.IpList = nil, nil
	settings.AllowedPorts = []string{"80"}
	settings.ApplyIPFilterForDomain = true
	t.Cleanup(func() {
		*settings = oldSettings
		client.Transport = oldTransport
		transport.CloseIdleConnections()
	})
	return "http://93.184.216.34"
}

func TestCanvasAudioURLsBecomeOrderedMultipartFiles(t *testing.T) {
	var gets, posts atomic.Int32
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/one.mp3", "/two.mp3":
			gets.Add(1)
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, audioReferenceUserAgent, r.UserAgent())
			require.Empty(t, r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			require.Empty(t, r.Header.Get("X-Api-Key"))
			fmt.Fprint(w, "ID3"+r.URL.Path)
		case "/v1/videos":
			posts.Add(1)
			require.Equal(t, "Bearer provider-test", r.Header.Get("Authorization"))
			require.NoError(t, r.ParseMultipartForm(1<<20))
			defer r.MultipartForm.RemoveAll()
			require.Equal(t, "TTTseadanco2.5/720", r.FormValue("model"))
			require.Equal(t, "15", r.FormValue("seconds"))
			require.Equal(t, "9:16", r.FormValue("ratio"))
			require.Equal(t, "720p", r.FormValue("resolution"))
			require.JSONEq(t, `["https://cdn.example/one.jpg","https://cdn.example/two.jpg"]`, r.FormValue("images"))
			require.JSONEq(t, `{"seed":0,"generate_audio":false}`, r.FormValue("metadata"))
			require.NotContains(t, r.MultipartForm.Value, "audios")
			require.Len(t, r.MultipartForm.File["audio"], 2)
			for index, file := range r.MultipartForm.File["audio"] {
				stream, err := file.Open()
				require.NoError(t, err)
				data, err := io.ReadAll(stream)
				stream.Close()
				require.NoError(t, err)
				require.Equal(t, []string{"ID3/one.mp3", "ID3/two.mp3"}[index], string(data))
				require.Equal(t, fmt.Sprintf("reference-audio-%d.mp3", index+1), file.Filename)
				require.Equal(t, "audio/mpeg", file.Header.Get("Content-Type"))
			}
			fmt.Fprint(w, `{"id":"mock-audio-task","status":"queued"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	payload := fmt.Sprintf(`{"model":"Seedance2.5:720（条）","prompt":"图1使用音频1，与图2交谈","duration":15,"ratio":"9:16","resolution":"720p","referenceImages":["https://cdn.example/one.jpg","https://cdn.example/two.jpg"],"referenceAudios":[%q,%q],"metadata":{"seed":0,"generate_audio":false}}`, origin+"/one.mp3", origin+"/two.mp3")
	c, _, info, a := testRequest(t, payload, origin)
	info.UpstreamModelName = "TTTseadanco2.5/720"
	c.Request.Header.Set("Authorization", "Bearer downstream-secret")
	c.Request.Header.Set("Cookie", "session=private")
	c.Request.Header.Set("X-Api-Key", "downstream-secret")
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Nil(t, info.Billing)
	require.EqualValues(t, 0, posts.Load())
	request, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	require.Equal(t, []string{origin + "/one.mp3", origin + "/two.mp3"}, request.Audios)
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, body)
	require.NoError(t, err)
	resp.Body.Close()
	_, err = a.DoRequest(c, info, bytes.NewReader(nil))
	require.Error(t, err)
	require.EqualValues(t, 2, gets.Load())
	require.EqualValues(t, 1, posts.Load())
}

func TestReferenceAudioFailuresAreLocalAndDoNotSubmit(t *testing.T) {
	var calls atomic.Int32
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/denied":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "secret-upstream-page")
		case "/empty":
		case "/html":
			w.Header().Set("Content-Type", "audio/mpeg")
			fmt.Fprint(w, "<html>Access denied</html>")
		case "/big":
			w.Header().Set("Content-Length", fmt.Sprint(audioReferenceMaxBytes+1))
		case "/redirect":
			http.Redirect(w, r, "http://127.0.0.1/private?token=secret", http.StatusFound)
		case "/valid":
			fmt.Fprint(w, "ID3valid-audio")
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
	})
	for _, test := range []struct{ path, detail string }{
		{"/denied", "HTTP 403"}, {"/empty", "为空"}, {"/html", "拦截页面"},
		{"/big", "15 MiB"}, {"/redirect", "无法安全访问"},
	} {
		t.Run(test.path, func(t *testing.T) {
			before := calls.Load()
			c, _, info, a := testRequest(t, fmt.Sprintf(`{"model":"alias","prompt":"test","seconds":15,"resolution":"720p","audios":[%q]}`, origin+test.path+"?token=secret"), origin)
			info.UpstreamModelName = "TTTseadanco2.5/720"
			taskErr := a.ValidateRequestAndSetAction(c, info)
			require.NotNil(t, taskErr)
			require.True(t, taskErr.LocalError)
			require.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			require.Equal(t, "invalid_reference_audio", taskErr.Code)
			service.LocalizeVideoTaskError(taskErr)
			require.Contains(t, taskErr.Message, "第 1 段")
			require.Contains(t, taskErr.Message, test.detail)
			require.NotContains(t, taskErr.Message, "secret")
			require.NotContains(t, taskErr.RawMessage, origin)
			require.False(t, c.GetBool("v6k_submission_attempted"))
			require.Nil(t, info.Billing)
			require.Empty(t, referenceAudioUploads(c))
			require.EqualValues(t, 1, calls.Load()-before)
		})
	}
	c, _, info, a := testRequest(t, fmt.Sprintf(`{"model":"alias","prompt":"test","seconds":15,"audios":[%q,%q]}`, origin+"/valid", origin+"/denied"), origin)
	info.UpstreamModelName = "TTTseadanco2.5/720"
	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.NotNil(t, taskErr)
	require.Contains(t, taskErr.Message, "第 2 段")
	require.Empty(t, referenceAudioUploads(c))
	require.False(t, c.GetBool("v6k_submission_attempted"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := calls.Load()
	_, err := fetchReferenceAudio(ctx, origin+"/cancel", "", audioReferenceMaxBytes)
	require.ErrorContains(t, err, "请求已取消")
	require.Equal(t, before, calls.Load())
}

func TestAudioURLAndExistingMultipartUploadKeepOrderAndCount(t *testing.T) {
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ID3downloaded-audio")
	})
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	for key, value := range map[string]string{
		"model": "alias", "prompt": "test", "seconds": "15", "resolution": "720p",
		"referenceAudios": fmt.Sprintf(`[{"url":%q}]`, origin+"/audio"),
	} {
		require.NoError(t, writer.WriteField(key, value))
	}
	file, err := writer.CreateFormFile("audio", "uploaded.mp3")
	require.NoError(t, err)
	_, err = file.Write([]byte("ID3original-upload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _, info, a := testRequest(t, "", origin)
	info.UpstreamModelName = "TTTseadanco2.5/720"
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(form.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	t.Cleanup(func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	})
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(c.GetString("v6k_content_type"))
	require.NoError(t, err)
	output, err := multipart.NewReader(body, params["boundary"]).ReadForm(1 << 20)
	require.NoError(t, err)
	defer output.RemoveAll()
	require.NotContains(t, output.Value, "audios")
	require.NotContains(t, output.Value, "referenceAudios")
	require.Len(t, output.File["audio"], 2)
	for index, part := range output.File["audio"] {
		stream, err := part.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(stream)
		stream.Close()
		require.NoError(t, err)
		require.Equal(t, []string{"ID3downloaded-audio", "ID3original-upload"}[index], string(data))
	}
}

func TestAudioCompatibilityPreservesDurationHintsAndOtherModels(t *testing.T) {
	for _, test := range []struct{ model, refs string }{
		{"TTTseadanco2.5/720", `[{"url":"https://cdn.example/audio.mp3","duration_seconds":6.8},"https://cdn.example/two.mp3"]`},
		{"TTTseadanco2.5/480", `["https://cdn.example/audio.mp3"]`},
		{"TTseadanco 2.0 Fast/720", `["https://cdn.example/audio.mp3"]`},
	} {
		t.Run(test.model, func(t *testing.T) {
			c, _, info, a := testRequest(t, fmt.Sprintf(`{"model":"alias","prompt":"test","seconds":15,"audios":%s}`, test.refs), "https://www.v6k.top")
			info.UpstreamModelName = test.model
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			body, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			var forwarded map[string]any
			require.NoError(t, common.DecodeJson(body, &forwarded))
			encoded, err := common.Marshal(forwarded["audios"])
			require.NoError(t, err)
			require.JSONEq(t, test.refs, string(encoded))
			require.Empty(t, c.GetString("v6k_content_type"))
		})
	}
}

func TestReferenceAudioReadIsBoundedWithoutContentLength(t *testing.T) {
	origin := referenceAudioTestOrigin(t, func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		fmt.Fprint(w, "ID3"+strings.Repeat("a", 100))
	})
	_, err := fetchReferenceAudio(context.Background(), origin+"/chunked", "", 64)
	require.ErrorContains(t, err, "音频文件过大")
}

func TestReferenceAudioContentTypeUsesContainerBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
		want string
	}{
		{"mp3 id3", []byte("ID3valid-container"), "audio/mpeg"},
		{"mp3 frame without id3", []byte{0xff, 0xfb, 0x90, 0x64}, "audio/mpeg"},
		{"aac adts", []byte{0xff, 0xf1, 0x50, 0x80, 0, 0, 0}, "audio/aac"},
		{"flac", []byte("fLaCtest"), "audio/flac"},
		{"m4a", []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'M', '4', 'A', ' '}, "audio/mp4"},
		{"html", []byte("<!doctype html><html>blocked</html>"), ""},
		{"json", []byte(`{"error":"denied"}`), ""},
		{"invalid frame", []byte{0xff, 0xff, 0xff, 0xff}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, referenceAudioContentType(test.data))
		})
	}
}

func multipartTestFields(t *testing.T, body io.Reader, contentType string) (map[string]any, int) {
	t.Helper()
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(body, params["boundary"])
	fields := map[string]any{}
	files := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		if part.FileName() != "" {
			require.Equal(t, "audio", part.FormName())
			files++
		} else {
			fields[part.FormName()] = string(data)
		}
	}
	return fields, files
}
