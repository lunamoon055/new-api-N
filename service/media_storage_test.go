package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestValidateMediaStorageProviders(t *testing.T) {
	valid := MediaStorageProvider{
		ID:         "ghlink",
		UploadURL:  "https://media.ghlink.top/upload",
		AuthHeader: "authCode",
		FieldName:  "file",
	}
	require.NoError(t, ValidateMediaStorageProviders([]MediaStorageProvider{valid}))
	require.Equal(t, "url", normalizeMediaStorageProvider(valid).ResponseURLPath)

	nested := valid
	nested.ResponseURLPath = "data.url"
	require.NoError(t, ValidateMediaStorageProviders([]MediaStorageProvider{nested}))

	invalid := valid
	invalid.UploadURL = "file:///tmp/uploads"
	require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{invalid}))

	invalid = valid
	invalid.AuthHeader = "bad header"
	require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{invalid}))

	invalid = valid
	invalid.ResponseURLPath = "data..url"
	require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{invalid}))

	duplicate := valid
	duplicate.Name = "second"
	require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{valid, duplicate}))
}

func TestUploadMediaURLKeepsSourceWhenNoProviderIsEnabled(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[MediaStorageOptionKey]
	common.OptionMap[MediaStorageOptionKey] = `[{"id":"disabled","upload_url":"https://media.ghlink.top/upload","enabled":false}]`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[MediaStorageOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})

	const source = "https://cdn.example/video.mp4"
	stored, err := UploadMediaURL(context.Background(), source, "video/mp4")
	require.Error(t, err)
	require.Equal(t, source, stored)
}

func TestMediaStorageUploadResponseURLPath(t *testing.T) {
	url, err := extractMediaStorageResponseURL(
		[]byte(`{"status":"Saved","url":"https://media.example/file.mp4"}`),
		"url",
	)
	require.NoError(t, err)
	require.Equal(t, "https://media.example/file.mp4", url)

	url, err = extractMediaStorageResponseURL(
		[]byte(`{"status":"Saved","data":{"url":"https://media.example/file.mp4"}}`),
		"data.url",
	)
	require.NoError(t, err)
	require.Equal(t, "https://media.example/file.mp4", url)

	_, err = extractMediaStorageResponseURL([]byte(`Saved`), "url")
	require.Error(t, err)
	_, err = extractMediaStorageResponseURL([]byte(`{"status":"Saved"}`), "url")
	require.Error(t, err)
}

func TestUploadToMediaStorageUsesConfiguredResponseURLPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret", r.Header.Get("authCode"))
		require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "new-api-media-storage-test.png")
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"status":"Saved","data":{"url":"https://media.example/file.png"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	fetchSetting := system_setting.GetFetchSetting()
	previous := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() { *fetchSetting = previous })

	url, err := uploadToMediaStorage(context.Background(), MediaStorageProvider{
		UploadURL:       server.URL,
		AuthHeader:      "authCode",
		Token:           "secret",
		FieldName:       "file",
		ResponseURLPath: "data.url",
	}, []byte("test"), "new-api-media-storage-test.png", "image/png")
	require.NoError(t, err)
	require.Equal(t, "https://media.example/file.png", url)
}
