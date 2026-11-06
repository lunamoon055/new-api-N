package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/gin-gonic/gin"
)

// GetNotifications exposes the same public announcements as status/notice,
// with a server clock and no HTTP cache, for reliable recent-publication checks.
func GetNotifications(c *gin.Context) {
	content, publishedAt := model.GetNoticeWithPublication()
	common.OptionMapRWMutex.RLock()
	enabled := console_setting.GetConsoleSetting().AnnouncementsEnabled
	announcements := []map[string]interface{}{}
	if enabled {
		announcements = console_setting.GetAnnouncements()
	}
	common.OptionMapRWMutex.RUnlock()
	c.Header("Cache-Control", "no-store")
	common.ApiSuccess(c, gin.H{
		"server_time":   time.Now().UnixMilli(),
		"notice":        gin.H{"content": content, "published_at": publishedAt},
		"announcements": announcements,
	})
}
