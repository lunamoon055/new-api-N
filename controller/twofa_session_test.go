package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTwoFASessionTestDB(t *testing.T, version int64) *model.User {
	t.Helper()

	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/twofa-session.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	user := &model.User{
		Id:             1,
		Username:       "twofa-session-user",
		DisplayName:    "2FA User",
		Role:           common.RoleCommonUser,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		SessionVersion: version,
	}
	require.NoError(t, db.Create(user).Error)
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})
	return user
}

func setupTwoFASessionTestRouter(t *testing.T, user *model.User, pendingVersion int64, createdAt int64) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("twofa-session-test"))))
	engine.GET("/seed", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(pending2FAUserIDKey, user.Id)
		session.Set(pending2FAUsernameKey, user.Username)
		session.Set(pending2FAVersionKey, pendingVersion)
		session.Set(pending2FACreatedAtKey, createdAt)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	engine.POST("/verify", Verify2FALogin)
	engine.GET("/login", func(c *gin.Context) {
		setupLogin(user, c)
	})
	engine.GET("/inspect", func(c *gin.Context) {
		session := sessions.Default(c)
		c.JSON(http.StatusOK, gin.H{
			"id":      session.Get("id"),
			"version": session.Get(sessionVersionSessionKey),
		})
	})
	return engine
}

func twoFASessionTestCookie(t *testing.T, engine http.Handler, path string) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	engine.ServeHTTP(recorder, request)
	require.NotEmpty(t, recorder.Result().Cookies())
	return recorder.Result().Cookies()[0]
}

func TestVerify2FALoginRejectsExpiredPendingSession(t *testing.T) {
	user := setupTwoFASessionTestDB(t, 1)
	engine := setupTwoFASessionTestRouter(t, user, 1, time.Now().Add(-pending2FAMaxAge).Unix())
	sessionCookie := twoFASessionTestCookie(t, engine, "/seed")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"code":"123456"}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(sessionCookie)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "会话已过期")
}

func TestVerify2FALoginRejectsRevokedPendingSession(t *testing.T) {
	user := setupTwoFASessionTestDB(t, 2)
	engine := setupTwoFASessionTestRouter(t, user, 1, time.Now().Unix())
	sessionCookie := twoFASessionTestCookie(t, engine, "/seed")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"code":"123456"}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(sessionCookie)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "会话已过期")
}

func TestSetupLoginWritesCurrentSessionVersion(t *testing.T) {
	user := setupTwoFASessionTestDB(t, 7)
	engine := setupTwoFASessionTestRouter(t, user, 7, time.Now().Unix())
	sessionCookie := twoFASessionTestCookie(t, engine, "/login")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/inspect", nil)
	request.AddCookie(sessionCookie)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"id":1,"version":7}`, recorder.Body.String())
}

func TestSetupLoginRejectsAuthenticationFromOlderVersion(t *testing.T) {
	user := setupTwoFASessionTestDB(t, 7)
	user.SessionVersion = 6
	engine := setupTwoFASessionTestRouter(t, user, 6, time.Now().Unix())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Empty(t, recorder.Result().Cookies())
}
