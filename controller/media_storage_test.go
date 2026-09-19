package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetMediaStorageSettingsMasksToken(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[service.MediaStorageOptionKey]
	common.OptionMap[service.MediaStorageOptionKey] = `[{"id":"host-1","enabled":true,"upload_url":"https://media.example/upload","token":"top-secret"}]`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[service.MediaStorageOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	GetMediaStorageSettings(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "top-secret")
	require.Contains(t, recorder.Body.String(), `"token":"********"`)

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	GetOptions(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "top-secret")
	require.NotContains(t, recorder.Body.String(), service.MediaStorageOptionKey)
}

func TestGenericOptionEndpointRejectsMediaStorageCredentials(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewBufferString(`{"key":"MediaStorageProviders","value":"[]"}`))
	UpdateOption(c)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestTestMediaStorage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret", r.Header.Get("authCode"))
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"status":"Saved","url":"https://media.example/test.png"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	fetchSetting := system_setting.GetFetchSetting()
	previousFetchSetting := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() { *fetchSetting = previousFetchSetting })

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[service.MediaStorageOptionKey]
	common.OptionMap[service.MediaStorageOptionKey] = `[{
		"id":"host-1",
		"upload_url":"` + server.URL + `",
		"auth_header":"authCode",
		"token":"secret",
		"field_name":"file",
		"response_url_path":"url"
	}]`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[service.MediaStorageOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/option/media_storage/test", bytes.NewBufferString(`{"provider_id":"host-1"}`))
	TestMediaStorage(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"url":"https://media.example/test.png"`)
}
