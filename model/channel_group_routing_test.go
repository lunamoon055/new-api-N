package model_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelGroupsRouteIndependentlyAfterRemovingDefault(t *testing.T) {
	previousDB := model.DB
	previousMemory, previousRedis := common.MemoryCacheEnabled, common.RedisEnabled
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousTranslate := common.TranslateMessage
	require.NoError(t, i18n.Init())
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/channel-groups.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}, &model.Ability{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled, common.RedisEnabled = previousMemory, previousRedis
		common.TranslateMessage = previousTranslate
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
	})
	// Reproduce a customized ratio list that omits the registration group.
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"vip":0.9,"SVIP":0.8}`))
	for index, group := range []string{"default", "vip", "SVIP"} {
		userID := index + 1
		require.NoError(t, db.Create(&model.User{Id: userID, Username: fmt.Sprintf("group-user-%d", userID), AffCode: fmt.Sprintf("group%d", userID), Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: group, SessionVersion: 1, Quota: 1000000}).Error)
		require.NoError(t, db.Create(&model.Token{UserId: userID, Key: strings.Repeat(string(rune('a'+index)), 48), Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}).Error)
	}
	// Explicit token groups must keep their meaning, including default.
	require.NoError(t, db.Create(&model.Token{UserId: 2, Key: strings.Repeat("d", 48), Group: "default", Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}).Error)
	require.NoError(t, db.Create(&model.Token{UserId: 2, Key: strings.Repeat("e", 48), Group: "SVIP", Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}).Error)
	channel := &model.Channel{Id: 1, Name: "group-routing-test", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "local-test-only", Group: "default,vip,SVIP", Models: "chat-model,gpt-image-2,video-model", BaseURL: common.GetPointer("https://ghlink.top")}
	require.NoError(t, channel.Insert())

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("channel-groups-test"))))
	engine.GET("/seed/:id", func(c *gin.Context) {
		session := sessions.Default(c)
		userID := 2
		if c.Param("id") == "3" {
			userID = 3
		}
		session.Set("id", userID)
		session.Set("session_version", int64(1))
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	respond := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"channel": c.GetInt("channel_id"), "group": c.GetString("group")})
	}
	for _, path := range []string{"/v1/chat/completions", "/v1/images/generations", "/v1/videos"} {
		engine.POST(path, middleware.TokenAuth(), middleware.Distribute(), respond)
	}
	for _, path := range []string{"/pg/chat/completions", "/pg/images/generations", "/pg/videos"} {
		engine.POST(path, middleware.UserAuth(), middleware.Distribute(), respond)
	}
	engine.POST("/api/creation/images/generations", middleware.UserAuth(), middleware.CreationImageRequestConvert(), middleware.Distribute(), respond)
	engine.POST("/api/creation/images/async-generations", middleware.UserAuth(), middleware.CreationImageAsyncRequestConvert(), middleware.Distribute(), respond)
	engine.POST("/v1/images/async-generations", middleware.TokenAuth(), middleware.Distribute(), respond)

	for _, cached := range []bool{false, true} {
		t.Run(fmt.Sprintf("cache=%t", cached), func(t *testing.T) {
			common.MemoryCacheEnabled = cached
			for _, groups := range []string{"default,vip,SVIP", "vip,SVIP", "vip, SVIP", "vip,default,SVIP"} {
				channel.Group = groups
				require.NoError(t, channel.Update())
				model.InitChannelCache()
				for _, group := range []string{"vip", "SVIP"} {
					require.ElementsMatch(t, []string{"chat-model", "gpt-image-2", "video-model"}, model.GetGroupEnabledModels(group))
				}
				for _, name := range []string{"chat-model", "gpt-image-2", "video-model"} {
					path := map[string]string{"chat-model": "chat/completions", "gpt-image-2": "images/generations", "video-model": "videos"}[name]
					for index, group := range []string{"default", "vip", "SVIP"} {
						request := httptest.NewRequest(http.MethodPost, "/v1/"+path, strings.NewReader(fmt.Sprintf(`{"model":%q,"prompt":"demo","messages":[]}`, name)))
						request.Header.Set("Authorization", "Bearer sk-"+strings.Repeat(string(rune('a'+index)), 48))
						request.Header.Set("Content-Type", "application/json")
						recorder := httptest.NewRecorder()
						engine.ServeHTTP(recorder, request)
						if group == "default" && !strings.Contains(groups, "default") {
							require.NotEqual(t, http.StatusOK, recorder.Code)
						} else {
							require.Equal(t, http.StatusOK, recorder.Code, "groups=%q, user=%s: %s", groups, group, recorder.Body.String())
							require.Contains(t, recorder.Body.String(), `"group":"`+group+`"`)
						}
					}
					for _, id := range []string{"2", "3"} {
						seed := httptest.NewRecorder()
						engine.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/seed/"+id, nil))
						request := httptest.NewRequest(http.MethodPost, "/pg/"+path, strings.NewReader(fmt.Sprintf(`{"model":%q,"prompt":"demo","messages":[]}`, name)))
						request.AddCookie(seed.Result().Cookies()[0])
						request.Header.Set("New-Api-User", id)
						request.Header.Set("Content-Type", "application/json")
						recorder := httptest.NewRecorder()
						engine.ServeHTTP(recorder, request)
						require.Equal(t, http.StatusOK, recorder.Code, "session groups=%q: %s", groups, recorder.Body.String())
					}
				}
				// Use the exact session route from the reported Playground failure.
				// A synchronous GHLINK channel must remain available after re-adding
				// default, while the provider-specific async route must not accept it.
				seed := httptest.NewRecorder()
				engine.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/seed/2", nil))
				for _, endpoint := range []string{"generations", "async-generations"} {
					for _, prefix := range []string{"/api/creation/images/", "/v1/images/"} {
						request := httptest.NewRequest(http.MethodPost, prefix+endpoint, strings.NewReader(`{"model":"gpt-image-2","prompt":"a quiet landscape"}`))
						request.Header.Set("Content-Type", "application/json")
						if strings.HasPrefix(prefix, "/api/") {
							request.AddCookie(seed.Result().Cookies()[0])
							request.Header.Set("New-Api-User", "2")
							request.Header.Set("X-Playground-Group", "vip")
						} else {
							request.Header.Set("Authorization", "Bearer sk-"+strings.Repeat("b", 48))
						}
						recorder := httptest.NewRecorder()
						engine.ServeHTTP(recorder, request)
						if endpoint == "generations" {
							require.Equal(t, http.StatusOK, recorder.Code, "groups=%s: %s", groups, recorder.Body.String())
							require.Contains(t, recorder.Body.String(), `"group":"vip"`)
						} else {
							require.Equal(t, http.StatusServiceUnavailable, recorder.Code, "async image requests must not be sent to synchronous providers")
						}
					}
				}
				request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"chat-model","messages":[]}`))
				request.Header.Set("Authorization", "Bearer sk-"+strings.Repeat("d", 48))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				if strings.Contains(groups, "default") {
					require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
					require.Contains(t, recorder.Body.String(), `"group":"default"`)
				} else {
					require.NotEqual(t, http.StatusOK, recorder.Code, "must not silently switch an explicit default token to vip")
				}
				request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"chat-model","messages":[]}`))
				request.Header.Set("Authorization", "Bearer sk-"+strings.Repeat("e", 48))
				request.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				require.Equal(t, http.StatusForbidden, recorder.Code, "VIP must not gain access to SVIP")
			}
			if cached {
				// A cache refresh can overlap an ability rebuild. Group maps
				// must be initialized from the enabled channels themselves.
				require.NoError(t, db.Where("channel_id = ?", channel.Id).Delete(&model.Ability{}).Error)
				require.NotPanics(t, model.InitChannelCache)
				for _, group := range []string{"vip", "SVIP"} {
					selected, err := model.GetRandomSatisfiedChannel(group, "video-model", 0)
					require.NoError(t, err)
					require.NotNil(t, selected)
					require.Equal(t, channel.Id, selected.Id)
				}
			}
		})
	}
	channel.Group = "default,vip,SVIP"
	require.NoError(t, channel.Update())
	channel.Group = " , , "
	require.Error(t, channel.Update(), "an empty CSV group must not silently enable default")
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail-group-ability", func(tx *gorm.DB) {
		if tx.Statement.Table == "abilities" {
			tx.AddError(errors.New("simulated ability write failure"))
		}
	}))
	defer db.Callback().Create().Remove("fail-group-ability")
	channel.Group = "vip"
	require.Error(t, channel.Update())
	reloaded, err := model.GetChannelById(channel.Id, true)
	require.NoError(t, err)
	require.Equal(t, "default,vip,SVIP", reloaded.Group, "failed group changes must roll back the channel and its abilities together")
	for _, group := range []string{"default", "vip", "SVIP"} {
		require.Len(t, model.GetGroupEnabledModels(group), 3)
	}
}

func TestChannelGroupIdentifiersPreserveCaseAndIgnoreCSVSpacing(t *testing.T) {
	channel := &model.Channel{Group: " default, vip , SVIP, vip,,默认,"}
	require.Equal(t, []string{"default", "vip", "SVIP", "默认"}, channel.GetGroups())
}
