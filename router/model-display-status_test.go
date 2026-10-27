package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestModelDisplayStatusAdminPermissionsAndPublicResponse(t *testing.T) {
	previousDB, previousRedis := model.DB, common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/display-status.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}, &model.ModelDisplayStatus{}))
	model.DB, common.RedisEnabled = db, false
	model.InvalidatePricingCache()
	t.Cleanup(func() {
		model.InvalidatePricingCache()
		model.DB, common.RedisEnabled = previousDB, previousRedis
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	for i, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
		require.NoError(t, db.Create(&model.User{Id: 7001 + i, Username: "display-status-user-" + strconv.Itoa(i), AffCode: "displaytest" + strconv.Itoa(i), Role: role, Status: common.UserStatusEnabled, Group: "default", SessionVersion: 1}).Error)
	}
	const name = "display-status-route-model"
	require.NoError(t, db.Create(&model.Ability{Model: name, Group: "default", ChannelId: 1, Enabled: true}).Error)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("model-display-status-test-key"))))
	engine.GET("/test-session/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		session := sessions.Default(c)
		session.Set("id", id)
		session.Set("session_version", int64(1))
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)
	request := func(body string, id int) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/models/display-status", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if id > 0 {
			seed := httptest.NewRecorder()
			engine.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/test-session/"+strconv.Itoa(id), nil))
			require.Equal(t, http.StatusNoContent, seed.Code)
			req.AddCookie(seed.Result().Cookies()[0])
			req.Header.Set("New-Api-User", strconv.Itoa(id))
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}
	body := `{"model_name":"` + name + `","available":false}`
	w := request(body, 0)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	w = request(body, 7001)
	require.Contains(t, w.Body.String(), `"success":false`)
	var count int64
	require.NoError(t, db.Model(&model.ModelDisplayStatus{}).Count(&count).Error)
	require.Zero(t, count)
	w = request(body, 7002)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"available":false`)
	public := httptest.NewRecorder()
	engine.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/api/pricing", nil))
	require.Equal(t, http.StatusOK, public.Code, public.Body.String())
	require.Contains(t, public.Body.String(), `"model_name":"`+name+`"`)
	require.Contains(t, public.Body.String(), `"display_available":false`)
	w = request(`{"model_name":"`+name+`","available":true}`, 7003)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"available":true`)
}
