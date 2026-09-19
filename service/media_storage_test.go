package service

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
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

	invalid := valid
	invalid.UploadURL = "file:///tmp/uploads"
	require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{invalid}))

	invalid = valid
	invalid.AuthHeader = "bad header"
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

func TestMediaStorageUploadResponseRequiresHTTPSURL(t *testing.T) {
	var response mediaStorageUploadResponse
	require.NoError(t, common.Unmarshal([]byte(`{"status":"Saved","url":"https://media.example/file.mp4"}`), &response))
	require.Equal(t, "https://media.example/file.mp4", response.URL)
}
