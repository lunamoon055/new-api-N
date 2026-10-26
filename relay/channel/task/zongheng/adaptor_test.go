package zongheng

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const modelResponse = `{"object":"list","data":[{"id":"公开模型:Test","capabilities":{"capability_type":"video","duration_options":[5,10],"resolution_options":["720p"],"aspect_ratio_options":["16:9","9:16"],"quality_options":[],"max_image_refs":2,"max_video_refs":1,"max_audio_refs":0,"defaults":{"duration":5,"resolution":"720p","aspect_ratio":"16:9"}}},{"id":"chat","capabilities":{"capability_type":"chat"}}]}`

func testContext(t *testing.T, payload string, baseURL string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *TaskAdaptor) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeZongheng, ChannelBaseUrl: baseURL, UpstreamModelName: "公开模型:Test", ApiKey: "sk-test"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}, OriginModelName: "alias", RequestURLPath: "/v1/videos"}
	a := &TaskAdaptor{}
	a.Init(info)
	return c, w, info, a
}

func TestVideoLifecycleWithLiveCapabilitiesAndModelMapping(t *testing.T) {
	service.InitHttpClient()
	var submitted map[string]any
	var polls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/models":
			require.Equal(t, "1", r.URL.Query().Get("include_capabilities"))
			fmt.Fprint(w, modelResponse)
		case "/v1/videos":
			require.Equal(t, "task_public", r.Header.Get("Idempotency-Key"))
			require.NoError(t, common.DecodeJson(r.Body, &submitted))
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"code":0,"success":true,"task_id":"vid_original","status":"processing","next_poll_seconds":10}`)
		case "/v1/tasks/vid_original":
			polls++
			if polls == 1 {
				fmt.Fprint(w, `{"task_id":"vid_original","status":"processing","next_poll_seconds":15}`)
			} else {
				fmt.Fprint(w, `{"task_id":"vid_original","status":"succeeded","video_url":"https://cdn.example/video.mp4"}`)
			}
		default:
			t.Errorf("unexpected request: %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, w, info, a := testContext(t, `{"model":"alias","prompt":"山间日出","generate_audio":false,"referenceImages":["https://cdn.example/ref.png"],"referenceVideos":["https://cdn.example/ref.mp4"]}`, server.URL+"/v1/")
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Equal(t, constant.TaskActionGenerate, info.Action)
	require.Equal(t, map[string]float64{"seconds": 5}, a.EstimateBilling(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, body)
	require.NoError(t, err)
	id, raw, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "vid_original", id)
	require.Contains(t, string(raw), "vid_original")
	require.Contains(t, w.Body.String(), "task_public")
	require.NotContains(t, w.Body.String(), "vid_original")
	require.Equal(t, "/v1/videos", info.UpstreamEndpoint)
	require.Equal(t, "公开模型:Test", submitted["model"])
	require.Equal(t, false, submitted["generate_audio"])
	require.EqualValues(t, 5, submitted["duration"])
	require.Equal(t, "16:9", submitted["ratio"])
	require.Equal(t, []any{"https://cdn.example/ref.png"}, submitted["images"])
	require.Equal(t, []any{"https://cdn.example/ref.mp4"}, submitted["reference_videos"])
	require.NotContains(t, submitted, "referenceImages")
	require.NotContains(t, submitted, "size")
	for _, expected := range []string{model.TaskStatusInProgress, model.TaskStatusSuccess} {
		resp, err := a.FetchTask(server.URL+"/v1", "sk-test", map[string]any{"task_id": id, "model": "sora2"}, "")
		require.NoError(t, err)
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		result, err := a.ParseTaskResult(data)
		require.NoError(t, err)
		require.Equal(t, expected, result.Status)
	}
}

func TestInvalidParametersAndReferencesNeverSubmit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		fmt.Fprint(w, modelResponse)
	}))
	defer server.Close()
	for _, extra := range []string{`"duration":0`, `"duration":5.5`, `"duration":6`, `"duration":5,"seconds":"10"`, `"ratio":"1:1"`, `"resolution":"1080p"`, `"quality":"high"`, `"size":"1280x720"`, `"reference_audios":["https://cdn.example/a.wav"]`, `"images":["https://cdn.example/1","https://cdn.example/2","https://cdn.example/3"]`, `"end_frame":"https://cdn.example/end"`, `"images":["https://cdn.example/i"],"start_frame":"https://cdn.example/s"`, `"images":["http://localhost/private"]`} {
		t.Run(extra, func(t *testing.T) {
			c, _, info, a := testContext(t, `{"model":"alias","prompt":"test",`+extra+`}`, server.URL)
			require.NotNil(t, a.ValidateRequestAndSetAction(c, info))
		})
	}
}

func TestUnpublishedRatioUsesFullSiteMenu(t *testing.T) {
	catalog := strings.Replace(modelResponse, `"aspect_ratio_options":["16:9","9:16"]`, `"aspect_ratio_options":[]`, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		fmt.Fprint(w, catalog)
	}))
	defer server.Close()
	for _, ratio := range common.DefaultVideoAspectRatios() {
		c, _, info, a := testContext(t, `{"model":"alias","prompt":"test","duration":5,"resolution":"720p","ratio":"`+ratio+`"}`, server.URL)
		require.Nil(t, a.ValidateRequestAndSetAction(c, info), ratio)
	}
	c, _, info, a := testContext(t, `{"model":"alias","prompt":"test","duration":5,"resolution":"720p"}`, server.URL)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	request, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	require.Equal(t, "16:9", request.Ratio)
}

func TestReferenceUploadAndRetriesPreserveBody(t *testing.T) {
	var uploads, modelReads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			modelReads++
			fmt.Fprint(w, modelResponse)
		case "/v1/media":
			uploads++
			require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
			require.NoError(t, r.ParseMultipartForm(1024))
			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			require.Equal(t, "image/png", header.Header.Get("Content-Type"))
			data, err := io.ReadAll(file)
			require.NoError(t, err)
			require.Equal(t, "png", string(data))
			fmt.Fprintf(w, `{"code":0,"success":true,"data":[{"url":"https://cdn.example/upload-%d.png"}]}`, uploads)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c, _, info, a := testContext(t, `{"model":"alias","prompt":"test","images":["data:image/png;base64,cG5n"]}`, server.URL)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	first, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	firstBody, err := io.ReadAll(first)
	require.NoError(t, err)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	second, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	secondBody, err := io.ReadAll(second)
	require.NoError(t, err)
	require.Equal(t, string(firstBody), string(secondBody))
	require.Equal(t, 1, uploads)
	require.Equal(t, 1, modelReads)
}

func TestMultipartReferenceUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			fmt.Fprint(w, modelResponse)
			return
		}
		require.Equal(t, "/v1/media", r.URL.Path)
		fmt.Fprint(w, `{"code":0,"success":true,"data":[{"url":"https://cdn.example/local.png"}]}`)
	}))
	defer server.Close()
	var data bytes.Buffer
	writer := multipart.NewWriter(&data)
	require.NoError(t, writer.WriteField("model", "alias"))
	require.NoError(t, writer.WriteField("prompt", "test"))
	require.NoError(t, writer.WriteField("duration", "5"))
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="input_reference"; filename="local.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _, info, a := testContext(t, "", server.URL)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &data)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	raw, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Contains(t, string(raw), "https://cdn.example/local.png")
}

func TestTerminalFailuresAndUnknownStates(t *testing.T) {
	a := &TaskAdaptor{}
	result, err := a.ParseTaskResult([]byte(`{"task_id":"vid_fail","status":"failed","error_code":"invalid_prompt","error_detail":"rejected"}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusFailure, result.Status)
	require.Equal(t, "invalid_prompt: rejected", result.Reason)
	for _, body := range []string{`{"status":"unknown"}`, `{"status":"succeeded"}`, `{"code":1,"message":"temporary error"}`} {
		_, err := a.ParseTaskResult([]byte(body))
		require.Error(t, err)
	}
	video, err := a.ConvertToOpenAIVideo(&model.Task{TaskID: "task_public", Status: model.TaskStatusSuccess, PrivateData: model.TaskPrivateData{ResultURL: "https://stored.example/video.mp4"}})
	require.NoError(t, err)
	require.Contains(t, string(video), "https://stored.example/video.mp4")
}
