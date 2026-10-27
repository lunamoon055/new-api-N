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

func TestModelTagsAdminPermissions(t *testing.T) {
	previousDB, previousRedis := model.DB, common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/model-tags.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Model{}, &model.Ability{}, &model.Channel{}, &model.Vendor{}))
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() {
		model.InvalidatePricingCache()
		model.DB, common.RedisEnabled = previousDB, previousRedis
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	for i, role := range []int{common.RoleCommonUser, common.RoleAdminUser} {
		require.NoError(t, db.Create(&model.User{Id: 7101 + i, Username: "tags-user-" + strconv.Itoa(i), AffCode: "tagstest" + strconv.Itoa(i), Role: role, Status: common.UserStatusEnabled, Group: "default", SessionVersion: 1}).Error)
	}
	item := model.Model{ModelName: "tag-route-model", Tags: "original", Status: 1}
	require.NoError(t, item.Insert())
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("model-tags-test-key"))))
	engine.GET("/test-session/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		session := sessions.Default(c)
		session.Set("id", id)
		session.Set("session_version", int64(1))
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)
	request := func(method, path string, id int) *httptest.ResponseRecorder {
		body, err := common.Marshal(map[string]any{"ids": []int{item.Id}, "tags": []string{"new"}})
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if id > 0 {
			seed := httptest.NewRecorder()
			engine.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/test-session/"+strconv.Itoa(id), nil))
			req.AddCookie(seed.Result().Cookies()[0])
			req.Header.Set("New-Api-User", strconv.Itoa(id))
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/api/models/tags"},
		{http.MethodPut, "/api/models/batch/tags"},
		{http.MethodPut, "/api/models/batch/tags/clear"},
	} {
		require.Equal(t, http.StatusUnauthorized, request(endpoint.method, endpoint.path, 0).Code)
		require.Contains(t, request(endpoint.method, endpoint.path, 7101).Body.String(), `"success":false`)
	}
	var unchanged model.Model
	require.NoError(t, db.First(&unchanged, item.Id).Error)
	require.Equal(t, "original", unchanged.Tags)
	require.Contains(t, request(http.MethodGet, "/api/models/tags", 7102).Body.String(), `"data":["original"]`)
	require.Contains(t, request(http.MethodPut, "/api/models/batch/tags", 7102).Body.String(), `"updated_count":1`)
	require.NoError(t, db.First(&unchanged, item.Id).Error)
	require.Equal(t, "original,new", unchanged.Tags)
	require.Contains(t, request(http.MethodPut, "/api/models/batch/tags/clear", 7102).Body.String(), `"updated_count":1`)
	require.NoError(t, db.First(&unchanged, item.Id).Error)
	require.Empty(t, unchanged.Tags)
}
