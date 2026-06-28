package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
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
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))

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
