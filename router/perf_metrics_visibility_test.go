package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSuccessRateVisibilityPermissionsAndPersistence(t *testing.T) {
	const optionKey = "perf_metrics_setting.show_success_rate"
	previousDB, previousRedis := model.DB, common.RedisEnabled
	common.OptionMapRWMutex.Lock()
	previousSetting := perf_metrics_setting.GetSetting()
	previousOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/visibility.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Option{}))
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() {
		model.DB, common.RedisEnabled = previousDB, previousRedis
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		*config.GlobalConfig.Get("perf_metrics_setting").(*perf_metrics_setting.PerfMetricsSetting) = previousSetting
		common.OptionMap = previousOptions
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	for i, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
		require.NoError(t, db.Create(&model.User{Id: i + 1, Username: "visibility-user-" + strconv.Itoa(i), AffCode: "vistest" + strconv.Itoa(i), Role: role, Status: common.UserStatusEnabled, Group: "default", SessionVersion: 1}).Error)
	}
	require.NoError(t, model.UpdateOption(optionKey, "true"))
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("success-rate-visibility-test-key"))))
	engine.GET("/test-session/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		session := sessions.Default(c)
		session.Set("id", id)
		session.Set("session_version", int64(1))
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)
	request := func(method, body string, id int) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/perf-metrics/success-rate-visibility", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if id > 0 {
			seed := httptest.NewRecorder()
			engine.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/test-session/"+strconv.Itoa(id), nil))
			require.Equal(t, http.StatusNoContent, seed.Code)
			req.AddCookie(seed.Result().Cookies()[0])
			req.Header.Set("New-Api-User", strconv.Itoa(id))
		}
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		return response
	}
	response := request(http.MethodGet, "", 0)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"show_success_rate":true`)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	response = request(http.MethodPut, `{"show_success_rate":false}`, 0)
	require.Equal(t, http.StatusUnauthorized, response.Code)
	response = request(http.MethodPut, `{"show_success_rate":false}`, 1)
	require.Contains(t, response.Body.String(), `"success":false`)
	require.True(t, perf_metrics_setting.GetSetting().ShowSuccessRate)

	for _, body := range []string{`{}`, `{"show_success_rate":null}`, `{"show_success_rate":"false"}`, `{"show_success_rate":0}`} {
		response = request(http.MethodPut, body, 2)
		require.Equal(t, http.StatusBadRequest, response.Code, body)
	}
	response = request(http.MethodPut, `{"show_success_rate":false}`, 2)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"show_success_rate":false`)
	var saved model.Option
	require.NoError(t, db.Where("key = ?", optionKey).First(&saved).Error)
	require.Equal(t, "false", saved.Value)
	require.True(t, perf_metrics_setting.GetSetting().Enabled)
	response = request(http.MethodGet, "", 0)
	require.Contains(t, response.Body.String(), `"show_success_rate":false`)
	response = request(http.MethodPut, `{"show_success_rate":true}`, 3)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"show_success_rate":true`)
}
