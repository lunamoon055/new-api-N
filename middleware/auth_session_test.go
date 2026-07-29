package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuthSessionTestRouter(t *testing.T, dbVersion int64) *gin.Engine {
	t.Helper()

	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/auth-session.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.User{
		Id:             1,
		Username:       "session-user",
		Role:           common.RoleCommonUser,
		Status:         common.UserStatusEnabled,
		Group:          "default",
		SessionVersion: dbVersion,
	}).Error)
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("auth-session-test"))))
	engine.GET("/seed/:version", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 1)
		session.Set("username", "stale-cookie-user")
		session.Set("role", common.RoleRootUser)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "stale-cookie-group")
		if c.Param("version") != "legacy" {
			session.Set(sessionVersionKey, int64(1))
		}
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	engine.GET("/protected", UserAuth(), func(c *gin.Context) {
		session := sessions.Default(c)
		c.JSON(http.StatusOK, gin.H{
			"id":              c.GetInt("id"),
			"username":        c.GetString("username"),
			"group":           c.GetString("group"),
			"session_version": session.Get(sessionVersionKey),
		})
	})
	engine.GET("/admin", AdminAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "admin handler reached")
	})
	engine.GET("/optional", TryUserAuth(), func(c *gin.Context) {
		_, authenticated := c.Get("id")
		c.JSON(http.StatusOK, gin.H{"authenticated": authenticated})
	})
	return engine
}

func authSessionTestCookie(t *testing.T, engine http.Handler, version string) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/seed/"+version, nil)
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.NotEmpty(t, recorder.Result().Cookies())
	return recorder.Result().Cookies()[0]
}

func performAuthSessionTestRequest(engine http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("New-Api-User", "1")
	request.AddCookie(cookie)
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestLegacySessionUpgradeRequiresDatabaseVersionOne(t *testing.T) {
	t.Run("safe upgrade", func(t *testing.T) {
		engine := setupAuthSessionTestRouter(t, 1)
		cookie := authSessionTestCookie(t, engine, "legacy")

		recorder := performAuthSessionTestRequest(engine, "/protected", cookie)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.JSONEq(t, `{
			"id": 1,
			"username": "session-user",
			"group": "default",
			"session_version": 1
		}`, recorder.Body.String())
	})

	t.Run("revoked database version", func(t *testing.T) {
		engine := setupAuthSessionTestRouter(t, 2)
		cookie := authSessionTestCookie(t, engine, "legacy")

		recorder := performAuthSessionTestRequest(engine, "/protected", cookie)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
	})
}

func TestVersionedSessionIsRevokedAfterVersionBump(t *testing.T) {
	engine := setupAuthSessionTestRouter(t, 2)
	cookie := authSessionTestCookie(t, engine, "1")

	recorder := performAuthSessionTestRequest(engine, "/protected", cookie)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestTryUserAuthDoesNotExposeRevokedCookieIdentity(t *testing.T) {
	engine := setupAuthSessionTestRouter(t, 2)
	cookie := authSessionTestCookie(t, engine, "1")

	recorder := performAuthSessionTestRequest(engine, "/optional", cookie)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"authenticated":false}`, recorder.Body.String())
}

func TestSessionAuthorizationUsesCurrentDatabaseRole(t *testing.T) {
	engine := setupAuthSessionTestRouter(t, 1)
	cookie := authSessionTestCookie(t, engine, "1")

	recorder := performAuthSessionTestRequest(engine, "/admin", cookie)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "admin handler reached")
	require.Contains(t, recorder.Body.String(), fmt.Sprintf("%t", false))
}
