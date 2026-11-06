package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNotificationsFeedIsPublicAndUncached(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	oldMap := common.OptionMap
	common.OptionMap = map[string]string{"Notice": "Public site notice"}
	common.OptionMapRWMutex.Unlock()
	cs := console_setting.GetConsoleSetting()
	oldEnabled, oldAnnouncements := cs.AnnouncementsEnabled, cs.Announcements
	cs.AnnouncementsEnabled = true
	cs.Announcements = `[{"id":1,"content":"Public timeline notice","publishDate":"2026-11-06T00:00:00Z"}]`
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldMap
		common.OptionMapRWMutex.Unlock()
		cs.AnnouncementsEnabled, cs.Announcements = oldEnabled, oldAnnouncements
	})
	for _, role := range []int{0, 1, 10, 100} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Set("role", role)
		GetNotifications(c)
		require.Equal(t, 200, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
		var response struct {
			Success bool `json:"success"`
			Data    struct {
				ServerTime int64 `json:"server_time"`
				Notice     struct {
					Content string `json:"content"`
				} `json:"notice"`
				Announcements []map[string]interface{} `json:"announcements"`
			} `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		require.InDelta(t, time.Now().UnixMilli(), response.Data.ServerTime, 1000)
		require.Equal(t, "Public site notice", response.Data.Notice.Content)
		require.Len(t, response.Data.Announcements, 1)
	}
	cs.AnnouncementsEnabled = false
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	GetNotifications(c)
	require.NotContains(t, recorder.Body.String(), "Public timeline notice")
}
