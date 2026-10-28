package service

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTaskRequestSnapshotKeepsParametersAndRedactsCredentials(t *testing.T) {
	original := `{"model":"public-video","prompt":"a sunset","duration":30,"ratio":"16:9","resolution":"720p","referenceAudios":[{"url":"https://cdn.example/a.mp3?token=private-token","duration_seconds":28}],"metadata":{"seed":9007199254740993,"generate_audio":false,"zero":0,"api_key":"private-key"},"headers":{"Cookie":"session=private"},"nested":"{\"access_token\":\"nested-secret\"}"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos?api_key=never-log-query", strings.NewReader(original))
	c.Request.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.Request.Header.Set("Authorization", "Bearer never-log-header")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	snapshot := CaptureTaskRequestSnapshot(c)
	require.Equal(t, "json", snapshot.Format)
	require.Equal(t, "/v1/videos", snapshot.Path)
	require.True(t, snapshot.Redacted)
	require.False(t, snapshot.Omitted)
	encoded, err := common.Marshal(snapshot)
	require.NoError(t, err)
	for _, secret := range []string{"private-token", "private-key", "nested-secret", "session=private", "never-log-query", "never-log-header"} {
		require.NotContains(t, string(encoded), secret)
	}
	require.Contains(t, string(snapshot.Body), `"seed":9007199254740993`)
	require.Contains(t, string(snapshot.Body), `"generate_audio":false`)
	require.Contains(t, string(snapshot.Body), `"zero":0`)
	require.Contains(t, string(snapshot.Body), `"duration_seconds":28`)
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	forwarded, err := io.ReadAll(storage)
	require.NoError(t, err)
	require.Equal(t, original, string(forwarded))
}

func TestTaskRequestSnapshotMultipartKeepsFieldsAndFileMetadata(t *testing.T) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	require.NoError(t, writer.WriteField("model", "public-video"))
	require.NoError(t, writer.WriteField("seconds", "15"))
	require.NoError(t, writer.WriteField("metadata", `{"seed":0,"api_key":"private-key"}`))
	part, err := writer.CreateFormFile("audio", "voice.mp3")
	require.NoError(t, err)
	_, err = part.Write([]byte("private-binary-audio"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(buffer.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	snapshot := CaptureTaskRequestSnapshot(c)
	require.Equal(t, "multipart", snapshot.Format)
	require.False(t, snapshot.Omitted)
	require.True(t, snapshot.Redacted)
	require.Contains(t, string(snapshot.Body), "voice.mp3")
	require.Contains(t, string(snapshot.Body), `"seconds":["15"]`)
	require.NotContains(t, string(snapshot.Body), "private-binary-audio")
	require.NotContains(t, string(snapshot.Body), "private-key")
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	forwarded, err := io.ReadAll(storage)
	require.NoError(t, err)
	require.Equal(t, buffer.Bytes(), forwarded)
}

func TestTaskRequestSnapshotBoundsAndInlineMedia(t *testing.T) {
	for _, body := range []string{`{"image":"data:image/png;base64,PRIVATE","duration":5}`, `{"prompt":"` + strings.Repeat("a", taskSnapshotMaxInput) + `"}`} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		snapshot := CaptureTaskRequestSnapshot(c)
		require.True(t, snapshot.Omitted)
		require.LessOrEqual(t, len(snapshot.Body), taskSnapshotMaxBody)
		require.NotContains(t, string(snapshot.Body), "PRIVATE")
		common.CleanupBodyStorage(c)
	}
}
