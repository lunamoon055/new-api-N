package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func r2TestProvider() MediaStorageProvider {
	return MediaStorageProvider{ID: "r2-test", Type: "r2", Enabled: true,
		R2Endpoint: "https://account.r2.cloudflarestorage.com", R2Bucket: "media-bucket",
		R2PublicURL: "https://media.example.com/assets", R2KeyPrefix: "new-api/图片 assets",
		R2AccessKeyID: "r2-access", R2SecretAccessKey: "r2-secret"}
}

func TestValidateR2MediaStorage(t *testing.T) {
	valid := r2TestProvider()
	require.NoError(t, ValidateMediaStorageProviders([]MediaStorageProvider{valid}))
	valid.R2Endpoint += "/media-bucket/"
	require.NoError(t, ValidateMediaStorageProviders([]MediaStorageProvider{valid}))
	require.Equal(t, "https://account.r2.cloudflarestorage.com", normalizeMediaStorageProvider(valid).R2Endpoint)
	for _, change := range []func(*MediaStorageProvider){
		func(p *MediaStorageProvider) { p.Type = "unknown" },
		func(p *MediaStorageProvider) { p.R2Endpoint = "http://example.com" },
		func(p *MediaStorageProvider) { p.R2Endpoint += "/wrong-bucket" },
		func(p *MediaStorageProvider) { p.R2Endpoint += "?token=secret" },
		func(p *MediaStorageProvider) { p.R2Endpoint = "https://user:pass@example.com" },
		func(p *MediaStorageProvider) { p.R2PublicURL = p.R2Endpoint },
		func(p *MediaStorageProvider) { p.R2PublicURL = "https://example.com?token=x" },
		func(p *MediaStorageProvider) { p.R2Bucket = "../private" },
		func(p *MediaStorageProvider) { p.R2KeyPrefix = "assets/../private" },
		func(p *MediaStorageProvider) { p.R2SecretAccessKey = "" },
		func(p *MediaStorageProvider) { p.R2AccessKeyID = "********" },
	} {
		p := r2TestProvider()
		change(&p)
		require.Error(t, ValidateMediaStorageProviders([]MediaStorageProvider{p}))
	}
	draft := r2TestProvider()
	draft.Enabled = false
	draft.R2AccessKeyID, draft.R2SecretAccessKey = "", ""
	require.NoError(t, ValidateMediaStorageProviders([]MediaStorageProvider{draft}))
	_, err := uploadToMediaStorage(context.Background(), draft, []byte("x"), "x.png", "image/png")
	require.ErrorContains(t, err, "Access Key")
}

func TestR2MediaBytesAndSpooledFileUseSignedPut(t *testing.T) {
	data := []byte("generated media bytes")
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	var paths []string
	status := http.StatusOK
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/media-bucket/new-api/图片 assets/"+hash+".mp4", r.URL.Path)
		require.Contains(t, r.RequestURI, "%E5%9B%BE%E7%89%87%20assets")
		require.Equal(t, int64(len(data)), r.ContentLength)
		require.Equal(t, "video/mp4", r.Header.Get("Content-Type"))
		require.Equal(t, hash, r.Header.Get("X-Amz-Content-Sha256"))
		require.Contains(t, r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=r2-access/")
		require.Contains(t, r.Header.Get("Authorization"), "/auto/s3/aws4_request")
		require.NotEmpty(t, r.Header.Get("X-Amz-Date"))
		require.Empty(t, r.Header.Get("X-Amz-Acl"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, data, body)
		if status == http.StatusTemporaryRedirect {
			w.Header().Set("Location", "/must-not-follow")
		}
		w.WriteHeader(status)
		if status == http.StatusForbidden {
			fmt.Fprint(w, "echo r2-secret r2-access")
		}
	}))
	defer server.Close()
	previousClient := httpClient
	httpClient = server.Client()
	setting := system_setting.GetFetchSetting()
	previousSetting := *setting
	setting.EnableSSRFProtection = false
	t.Cleanup(func() { httpClient = previousClient; *setting = previousSetting })
	p := r2TestProvider()
	p.R2Endpoint = server.URL
	stored, err := uploadToMediaStorage(context.Background(), p, data, "source.mp4", "video/mp4")
	require.NoError(t, err)
	require.Equal(t, "https://media.example.com/assets/new-api/%E5%9B%BE%E7%89%87%20assets/"+hash+".mp4", stored)
	file := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.WriteFile(file, data, 0600))
	for i := 0; i < 2; i++ {
		fileURL, err := uploadMediaFile(context.Background(), p, file, int64(len(data)), "source.mp4", "video/mp4")
		require.NoError(t, err)
		require.Equal(t, stored, fileURL)
	}
	require.Len(t, paths, 3)
	status = http.StatusTemporaryRedirect
	_, err = uploadToMediaStorage(context.Background(), p, data, "source.mp4", "video/mp4")
	require.ErrorContains(t, err, "307")
	require.Len(t, paths, 4, "signed uploads must not follow redirects")
	status = http.StatusForbidden
	_, err = uploadToMediaStorage(context.Background(), p, data, "source.mp4", "video/mp4")
	require.ErrorContains(t, err, "403")
	require.NotContains(t, err.Error(), "r2-secret")
	require.NotContains(t, err.Error(), "r2-access")
	_, err = uploadToR2MediaStorage(context.Background(), p, strings.NewReader("x"), 2, "x.png", "image/png")
	require.ErrorContains(t, err, "size")
	require.Len(t, paths, 5)
	setting.EnableSSRFProtection = true
	setting.AllowPrivateIp = false
	_, err = uploadToMediaStorage(context.Background(), p, data, "source.mp4", "video/mp4")
	require.Error(t, err)
	require.Len(t, paths, 5, "R2 uses the existing SSRF guard")
}

func TestR2TestUploadRequiresReadablePublicURLWithoutCredentials(t *testing.T) {
	publicStatus := http.StatusOK
	var publicReads int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			require.NotEmpty(t, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			return
		}
		publicReads++
		require.Equal(t, http.MethodGet, r.Method)
		require.True(t, strings.HasPrefix(r.URL.Path, "/public/new-api/"))
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("X-Amz-Date"))
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(publicStatus)
		w.Write([]byte{137, 80})
	}))
	defer server.Close()
	previousClient := httpClient
	httpClient = server.Client()
	setting := system_setting.GetFetchSetting()
	previousSetting := *setting
	setting.EnableSSRFProtection = false
	t.Cleanup(func() { httpClient = previousClient; *setting = previousSetting })
	p := r2TestProvider()
	p.Enabled = false
	p.R2Endpoint = server.URL
	p.R2PublicURL = server.URL + "/public"
	raw, err := common.Marshal([]MediaStorageProvider{p})
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[MediaStorageOptionKey]
	common.OptionMap[MediaStorageOptionKey] = string(raw)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[MediaStorageOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})
	stored, err := TestMediaStorageProvider(context.Background(), p.ID)
	require.NoError(t, err)
	require.Contains(t, stored, "/public/new-api/")
	publicStatus = http.StatusNotFound
	_, err = TestMediaStorageProvider(context.Background(), p.ID)
	require.ErrorContains(t, err, "public URL is not readable")
	require.Equal(t, 2, publicReads)
}
