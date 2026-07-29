package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestApplyVideoProxyDownloadHeadersUsesSafeVideoFilename(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/octet-stream")
	headers.Set("Content-Disposition", `attachment; filename="1782445743626_1f171116"`)

	applyVideoProxyDownloadHeaders(headers, "task_video_123")

	require.Equal(t, "video/mp4", headers.Get("Content-Type"))
	require.Equal(t, `inline; filename="task_video_123.mp4"`, headers.Get("Content-Disposition"))
	require.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
}

func TestVideoProxySecurityHeadersAndResponseHeaderAllowlist(t *testing.T) {
	headers := http.Header{}
	applyVideoProxyPrivateCacheHeaders(headers)

	require.Equal(t, "private, no-store, max-age=0", headers.Get("Cache-Control"))
	require.Equal(t, "no-cache", headers.Get("Pragma"))
	require.Equal(t, "Authorization, Cookie", headers.Get("Vary"))

	upstreamHeaders := http.Header{
		"Content-Type":  {"video/webm"},
		"Content-Range": {"bytes 0-9/10"},
		"Set-Cookie":    {"upstream_session=secret"},
		"X-Upstream":    {"must-not-leak"},
	}
	copyVideoProxyResponseHeaders(headers, upstreamHeaders)

	require.Equal(t, "video/webm", headers.Get("Content-Type"))
	require.Equal(t, "bytes 0-9/10", headers.Get("Content-Range"))
	require.Empty(t, headers.Get("Set-Cookie"))
	require.Empty(t, headers.Get("X-Upstream"))
}

func TestEnsureAPIKeyForOriginOnlyAttachesToExactOrigin(t *testing.T) {
	const (
		baseURL = "https://generativelanguage.googleapis.com"
		apiKey  = "secret key"
	)

	sameOrigin := ensureAPIKeyForOrigin(
		"https://generativelanguage.googleapis.com/v1beta/files/video:download?alt=media",
		baseURL,
		apiKey,
	)
	parsedSameOrigin, err := http.NewRequest(http.MethodGet, sameOrigin, nil)
	require.NoError(t, err)
	require.Equal(t, apiKey, parsedSameOrigin.URL.Query().Get("key"))

	for _, untrustedURL := range []string{
		"https://attacker.example/video",
		"https://generativelanguage.googleapis.com.attacker.example/video",
		"http://generativelanguage.googleapis.com/video",
		"https://generativelanguage.googleapis.com:444/video",
	} {
		require.Equal(t, untrustedURL, ensureAPIKeyForOrigin(untrustedURL, baseURL, apiKey))
	}
}

func TestVideoProxyRedirectStripsCredentialsAcrossOrigins(t *testing.T) {
	fetchSetting := system_setting.GetFetchSetting()
	originalFetchSetting := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() {
		*fetchSetting = originalFetchSetting
	})

	client := newVideoProxyHTTPClient(&http.Client{})
	redirectRequest := httptest.NewRequest(http.MethodGet, "https://storage.example/video", nil)
	redirectRequest.Header.Set("Authorization", "Bearer secret")
	redirectRequest.Header.Set("Cookie", "session=secret")
	redirectRequest.Header.Set("X-Goog-Api-Key", "secret")
	originalRequest := httptest.NewRequest(http.MethodGet, "https://generativelanguage.googleapis.com/video", nil)

	require.NoError(t, client.CheckRedirect(redirectRequest, []*http.Request{originalRequest}))
	require.Empty(t, redirectRequest.Header.Get("Authorization"))
	require.Empty(t, redirectRequest.Header.Get("Cookie"))
	require.Empty(t, redirectRequest.Header.Get("X-Goog-Api-Key"))
}

func TestVideoProxyAllowsAdminSessionsToPreviewOtherUsersTask(t *testing.T) {
	for name, role := range map[string]int{
		"admin": common.RoleAdminUser,
		"root":  common.RoleRootUser,
	} {
		t.Run(name, func(t *testing.T) {
			setupVideoProxyControllerTestDB(t)
			insertVideoProxyTestChannel(t, 7)
			insertVideoProxyTestTask(t, "task_other_user", 22, 7)

			router := setupVideoProxyControllerTestRouter(t, 1, role)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/v1/videos/task_other_user/content", nil)

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
			require.Equal(t, "private, no-store, max-age=0", recorder.Header().Get("Cache-Control"))
			require.Equal(t, []byte{0, 0, 0}, recorder.Body.Bytes())
		})
	}
}

func TestVideoProxyKeepsCommonSessionScopedToOwnTasks(t *testing.T) {
	setupVideoProxyControllerTestDB(t)
	insertVideoProxyTestChannel(t, 7)
	insertVideoProxyTestTask(t, "task_other_user", 22, 7)

	router := setupVideoProxyControllerTestRouter(t, 1, common.RoleCommonUser)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/videos/task_other_user/content", nil)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func setupVideoProxyControllerTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldUsingSQLite := common.UsingSQLite

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/video-proxy.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}, &model.User{}))

	model.DB = db
	common.MemoryCacheEnabled = false
	common.UsingSQLite = true

	t.Cleanup(func() {
		model.DB = oldDB
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.UsingSQLite = oldUsingSQLite
	})
}

func insertVideoProxyTestChannel(t *testing.T, channelID int) {
	t.Helper()

	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeDoubaoVideo,
		Key:    "test-key",
		Name:   "test-channel",
		Status: common.ChannelStatusEnabled,
	}).Error)
}

func insertVideoProxyTestTask(t *testing.T, taskID string, userID int, channelID int) {
	t.Helper()

	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    taskID,
		UserId:    userID,
		ChannelId: channelID,
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		PrivateData: model.TaskPrivateData{
			ResultURL: "data:video/mp4;base64,AAAA",
		},
	}).Error)
}

func setupVideoProxyControllerTestRouter(t *testing.T, requesterID int, requesterRole int) *gin.Engine {
	t.Helper()

	require.NoError(t, model.DB.Create(&model.User{
		Id:             requesterID,
		Username:       "requester",
		Role:           requesterRole,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		SessionVersion: 1,
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("video-proxy-test"))))
	router.GET("/v1/videos/:task_id/content", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", requesterID)
		session.Set("username", "requester")
		session.Set("role", requesterRole)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Next()
	}, middleware.TokenOrUserAuth(), VideoProxy)
	return router
}

func TestIsAsyncGenerationsVideoTaskIncludesLinkskyModels(t *testing.T) {
	for _, modelName := range []string{
		"sora2",
		"sora-2",
		"kling-v3",
		"video-2.0",
		"video-2.0-fast",
		"video-2.0-mini",
		"video-2.0-480p",
		"video-2.0-fast-480p",
		"video-2.0-mini-480p",
		"ko3",
		"veo31",
		"veo31-fast",
		"veo31-ref",
		"grok-imagine-video",
	} {
		t.Run(modelName, func(t *testing.T) {
			task := &model.Task{
				Properties: model.Properties{
					OriginModelName: modelName,
				},
			}

			require.True(t, isAsyncGenerationsVideoTask(task))
		})
	}
}
