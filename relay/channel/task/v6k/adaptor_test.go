package v6k

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, payload string, baseURL string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *TaskAdaptor) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	info := &relaycommon.RelayInfo{OriginModelName: "alias", RequestURLPath: "/v1/videos", ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeV6K, ChannelBaseUrl: baseURL, ApiKey: "provider-test", UpstreamModelName: "TTseadanco 2.0 Fast/480"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}}
	a := &TaskAdaptor{}
	a.Init(info)
	return c, w, info, a
}

func TestUnifiedVideoLifecycleAndReferenceMetadata(t *testing.T) {
	service.InitHttpClient()
	var submitted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer provider-test", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/videos":
			require.Equal(t, http.MethodPost, r.Method)
			require.Empty(t, r.Header.Get("Idempotency-Key"))
			require.NoError(t, common.DecodeJson(r.Body, &submitted))
			fmt.Fprint(w, `{"id":"upstream_private","status":"queued","progress":0}`)
		case "/v1/videos/upstream_private":
			require.Equal(t, http.MethodGet, r.Method)
			fmt.Fprint(w, `{"id":"upstream_private","status":"completed","progress":100}`)
		default:
			t.Errorf("unexpected protocol path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, w, info, a := testRequest(t, `{"model":"alias","prompt":"日出","duration":"5","aspect_ratio":"16:9","resolution":"480p","referenceImages":["https://cdn.example/image.png"],"videos":[{"url":"https://cdn.example/video.mp4","duration_seconds":3.5}],"audios":["https://cdn.example/audio.mp3"],"input_reference":{"url":"https://cdn.example/first.png"},"metadata":{"generate_audio":false,"seed":0}}`, server.URL+"/v1/")
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Equal(t, constant.TaskActionGenerate, info.Action)
	require.Equal(t, map[string]float64{"seconds": 5}, a.EstimateBilling(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, body)
	require.NoError(t, err)
	id, raw, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "upstream_private", id)
	require.Contains(t, string(raw), id)
	require.NotContains(t, w.Body.String(), id)
	require.Contains(t, w.Body.String(), "task_public")
	require.Equal(t, "/v1/videos", info.UpstreamEndpoint)
	require.Equal(t, "TTseadanco 2.0 Fast/480", submitted["model"])
	require.EqualValues(t, 5, submitted["seconds"])
	require.NotContains(t, submitted, "duration")
	require.NotContains(t, submitted, "referenceImages")
	require.Equal(t, false, submitted["metadata"].(map[string]any)["generate_audio"])
	require.EqualValues(t, 0, submitted["metadata"].(map[string]any)["seed"])
	require.EqualValues(t, 3.5, submitted["videos"].([]any)[0].(map[string]any)["duration_seconds"])
	resp, err = a.FetchTask(server.URL+"/v1", "provider-test", map[string]any{"task_id": id, "model": "kling-v3"}, "")
	require.NoError(t, err)
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	result, err := a.ParseTaskResult(data)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, result.Status)
	require.Equal(t, server.URL+"/v1/videos/upstream_private/content", result.Url)
}

func TestInvalidVideoInputsAreRejectedBeforeSubmission(t *testing.T) {
	for _, field := range []string{
		`"seconds":0`, `"seconds":-1`, `"seconds":5.5`, `"seconds":3601`, `"seconds":5,"duration":10`,
		`"seconds":5,"metadata":{"seconds":100}`, `"seconds":5,"metadata":{"Model":"other"}`, `"seconds":5,"n":0`,
		`"seconds":5,"images":["http://127.0.0.1/a"]`, `"seconds":5,"images":["data:image/png;base64,YQ=="]`,
		`"seconds":5,"videos":[{"url":"https://cdn.example/v","duration_seconds":0}]`,
		`"seconds":5,"ratio":"16:9","aspect_ratio":"9:16"`, `"seconds":5,"ratio":"0:9"`,
		`"seconds":5,"size":"1280x720","resolution":"1080p"`, `"seconds":5,"size":"1280x720","ratio":"9:16"`,
		`"seconds":5,"end_image_url":"https://cdn.example/end"`, `"seconds":5,"input_reference":["https://cdn.example/a","https://cdn.example/b"]`,
	} {
		t.Run(field, func(t *testing.T) {
			c, _, info, a := testRequest(t, `{"model":"alias","prompt":"test",`+field+`}`, "https://www.v6k.top")
			require.NotNil(t, a.ValidateRequestAndSetAction(c, info))
			require.False(t, c.GetBool("v6k_submission_attempted"))
		})
	}
	c, _, info, a := testRequest(t, `{"model":"alias","prompt":"test"}`, "https://www.v6k.top")
	require.NotNil(t, a.ValidateRequestAndSetAction(c, info)) // no unbilled upstream default duration
}

func TestMultipartFilesAndFormArraysReachUnifiedEndpoint(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1024*1024))
		defer r.MultipartForm.RemoveAll()
		require.Equal(t, "5", r.FormValue("seconds"))
		require.Equal(t, "TTseadanco 2.0 Fast/480", r.FormValue("model"))
		require.Contains(t, r.FormValue("metadata"), `"generate_audio":false`)
		require.Contains(t, r.FormValue("images"), "https://cdn.example/ref")
		file, header, err := r.FormFile("input_reference")
		require.NoError(t, err)
		defer file.Close()
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, "first.png", header.Filename)
		require.Equal(t, "image-bytes", string(data))
		fmt.Fprint(w, `{"id":"uploaded","status":"queued"}`)
	}))
	defer server.Close()
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	for key, value := range map[string]string{"model": "alias", "prompt": "test", "duration": "5", "images[]": "https://cdn.example/ref", "metadata": `{"generate_audio":false}`} {
		require.NoError(t, writer.WriteField(key, value))
	}
	part, err := writer.CreateFormFile("input_reference", "first.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _, info, a := testRequest(t, "", server.URL)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(form.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	t.Cleanup(func() {
		if c.Request.MultipartForm != nil {
			c.Request.MultipartForm.RemoveAll()
		}
	})
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, body)
	require.NoError(t, err)
	resp.Body.Close()
}

func TestUnknownIsPendingAndOnlyFailedIsTerminalFailure(t *testing.T) {
	a := &TaskAdaptor{baseURL: "https://www.v6k.top"}
	for _, tc := range []struct{ status, want string }{{"unknown", model.TaskStatusQueued}, {"queued", model.TaskStatusQueued}, {"in_progress", model.TaskStatusInProgress}, {"completed", model.TaskStatusSuccess}, {"failed", model.TaskStatusFailure}} {
		result, err := a.ParseTaskResult([]byte(fmt.Sprintf(`{"id":"provider-id","status":%q,"error":{"message":"explicit failure"}}`, tc.status)))
		require.NoError(t, err)
		require.Equal(t, tc.want, result.Status)
		if tc.status == "failed" {
			require.Equal(t, "explicit failure", result.Reason)
		}
	}
	_, err := a.ParseTaskResult([]byte(`{"id":"provider-id","status":"unexpected"}`))
	require.Error(t, err)
	encoded, err := a.ConvertToOpenAIVideo(&model.Task{TaskID: "task_public", Status: model.TaskStatusSuccess, PrivateData: model.TaskPrivateData{ResultURL: "https://www.v6k.top/private/source"}})
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "/private/source")
	require.Contains(t, string(encoded), "/v1/videos/task_public/content")
}

func TestAmbiguousPostIsNotResubmitted(t *testing.T) {
	service.InitHttpClient()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()
	c, _, info, a := testRequest(t, `{"model":"alias","prompt":"test","seconds":5}`, server.URL)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	_, err = a.DoRequest(c, info, body)
	require.Error(t, err)
	_, err = a.DoRequest(c, info, strings.NewReader(`{}`))
	require.Error(t, err)
	require.EqualValues(t, 1, calls.Load())
}

func TestZHNewModelsForwardExactIDsAndMaximumPublishedReferences(t *testing.T) {
	for _, id := range common.V6KVideoModelIDs() {
		p := common.GetV6KValidatedVideoParameters(id)
		if p == nil {
			continue
		}
		t.Run(id, func(t *testing.T) {
			payload := map[string]any{"model": id, "prompt": "demo", "seconds": p.Durations[len(p.Durations)-1], "resolution": p.Resolutions[0]}
			for field, count := range map[string]int{"images": p.MaxImages, "videos": p.MaxVideos, "audios": p.MaxAudios} {
				refs := make([]string, count)
				for index := range refs {
					refs[index] = fmt.Sprintf("https://cdn.example/%s/%d", field, index)
				}
				payload[field] = refs
			}
			data, err := common.Marshal(payload)
			require.NoError(t, err)
			c, _, info, a := testRequest(t, string(data), "https://www.v6k.top")
			info.OriginModelName, info.UpstreamModelName = id, id
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			body, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			var forwarded map[string]any
			require.NoError(t, common.DecodeJson(body, &forwarded))
			require.Equal(t, id, forwarded["model"])
			require.EqualValues(t, p.Durations[len(p.Durations)-1], forwarded["seconds"])
			require.Equal(t, "16:9", forwarded["ratio"])
			require.NotContains(t, forwarded, "duration")
		})
	}
}

func TestV6KMissingRatioIsForwardedWithDefaultOrSizeRatio(t *testing.T) {
	for _, tc := range []struct{ name, fields, ratio string }{
		{"missing", `"seconds":15,"resolution":"720p"`, "16:9"},
		{"blank", `"seconds":15,"ratio":" "`, "16:9"},
		{"portrait size", `"seconds":15,"size":"720x1280"`, "9:16"},
		{"alias", `"seconds":15,"aspect_ratio":"2:3"`, "2:3"},
		{"explicit", `"seconds":15,"ratio":"21:9"`, "21:9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, info, a := testRequest(t, `{"model":"dola2.5","prompt":"海边日落",`+tc.fields+`}`, "https://www.v6k.top")
			info.OriginModelName, info.UpstreamModelName = "dola2.5", "dola2.5"
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			body, err := a.BuildRequestBody(c, info)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.DecodeJson(body, &payload))
			require.Equal(t, tc.ratio, payload["ratio"])
			require.EqualValues(t, 15, payload["seconds"])
			require.False(t, c.GetBool("v6k_submission_attempted"))
		})
	}
}

func TestZHNewModelCapabilityErrorsAreRejectedBeforeSubmission(t *testing.T) {
	for _, tc := range []struct{ id, fields string }{
		{"Cseadanco2.0-fast", `"seconds":30,"resolution":"720p"`},
		{"Cseadanco2.5-720", `"seconds":31,"resolution":"720p"`},
		{"XXseadanco2.5-1080", `"seconds":5,"resolution":"720p"`},
		{"TTTseadanco2.5/480", `"seconds":5,"videos":["https://cdn.example/v.mp4"]`},
		{"wan3.M", `"seconds":5,"resolution":"1080p"`},
		{"dola2.5", `"seconds":6`},
		{"dola2.5", `"seconds":5,"audios":["https://cdn.example/a.mp3"]`},
	} {
		t.Run(tc.id+tc.fields, func(t *testing.T) {
			c, _, info, a := testRequest(t, `{"model":"`+tc.id+`","prompt":"demo",`+tc.fields+`}`, "https://www.v6k.top")
			info.UpstreamModelName = tc.id
			require.NotNil(t, a.ValidateRequestAndSetAction(c, info))
			require.False(t, c.GetBool("v6k_submission_attempted"))
		})
	}
	payload := map[string]any{"seconds": 5, "images": make([]any, 9), "input_reference": "https://cdn.example/first.png"}
	require.ErrorContains(t, validateModelCapabilities("Cseadanco2.0-1", payload, nil), "images")
	require.ErrorContains(t, validateModelCapabilities("Cseadanco2.5-720", map[string]any{"images": make([]any, 30)}, map[string]int{"images": 1}), "images")
	require.ErrorContains(t, validateModelCapabilities("TTTseadanco2.5/720", map[string]any{}, map[string]int{"video": 1}), "videos")
	// Published hints for previously integrated models remain hints.
	require.NoError(t, validateModelCapabilities("JJseadanco2.5", map[string]any{"seconds": 3600}, nil))
}
