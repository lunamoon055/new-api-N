package model

import (
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func preserveNoticeOptionState(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Option{}))
	keys := []string{"Notice", noticePublicationOptionKey}
	var original []Option
	require.NoError(t, DB.Where("key IN ?", keys).Find(&original).Error)
	common.OptionMapRWMutex.RLock()
	originalMap := make(map[string]string)
	for _, key := range keys {
		if value, ok := common.OptionMap[key]; ok {
			originalMap[key] = value
		}
	}
	common.OptionMapRWMutex.RUnlock()
	require.NoError(t, DB.Where("key IN ?", keys).Delete(&Option{}).Error)
	t.Cleanup(func() {
		DB.Where("key IN ?", keys).Delete(&Option{})
		for _, option := range original {
			DB.Save(&option)
		}
		common.OptionMapRWMutex.Lock()
		for _, key := range keys {
			delete(common.OptionMap, key)
		}
		for key, value := range originalMap {
			common.OptionMap[key] = value
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func TestNoticePublicationChangesOnlyWithContent(t *testing.T) {
	preserveNoticeOptionState(t)
	now := time.Unix(1_800_000_000, 0)
	require.NoError(t, updateNoticeOption("A new model", now))
	content, publishedAt := GetNoticeWithPublication()
	require.Equal(t, "A new model", content)
	require.Equal(t, now.Unix(), publishedAt)

	require.NoError(t, updateNoticeOption(" A new model \n", now.Add(time.Hour)))
	_, publishedAt = GetNoticeWithPublication()
	require.Equal(t, now.Unix(), publishedAt, "resaving must not restart the popup window")

	require.NoError(t, updateNoticeOption("Another model", now.Add(2*time.Hour)))
	_, publishedAt = GetNoticeWithPublication()
	require.Equal(t, now.Add(2*time.Hour).Unix(), publishedAt)

	var persisted Option
	require.NoError(t, DB.First(&persisted, "key = ?", noticePublicationOptionKey).Error)
	require.Equal(t, publishedAt, noticePublicationTime("Another model", persisted.Value))
	require.Zero(t, noticePublicationTime("Stale content from another snapshot", persisted.Value))

	require.NoError(t, UpdateOption("Notice", ""))
	content, publishedAt = GetNoticeWithPublication()
	require.Empty(t, content)
	require.Zero(t, publishedAt)
}

func TestLegacyNoticeIsNotRepublishedByUnchangedSave(t *testing.T) {
	preserveNoticeOptionState(t)
	require.NoError(t, DB.Save(&Option{Key: "Notice", Value: "Legacy notice"}).Error)
	require.NoError(t, updateNoticeOption("Legacy notice", time.Now()))
	content, publishedAt := GetNoticeWithPublication()
	require.Equal(t, "Legacy notice", content)
	require.Zero(t, publishedAt)
}

func TestNoticePublicationWriteFailureRollsBackContent(t *testing.T) {
	preserveNoticeOptionState(t)
	now := time.Unix(1_800_000_000, 0)
	require.NoError(t, updateNoticeOption("Original notice", now))
	const callbackName = "test:notice_publication_write_failure"
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if option, ok := tx.Statement.Dest.(*Option); ok && option.Key == noticePublicationOptionKey {
			tx.AddError(errors.New("publication write failed"))
		}
	}))
	t.Cleanup(func() { DB.Callback().Update().Remove(callbackName) })

	require.Error(t, updateNoticeOption("Changed notice", now.Add(time.Hour)))
	var persisted Option
	require.NoError(t, DB.First(&persisted, "key = ?", "Notice").Error)
	require.Equal(t, "Original notice", persisted.Value)
	content, publishedAt := GetNoticeWithPublication()
	require.Equal(t, "Original notice", content)
	require.Equal(t, now.Unix(), publishedAt)
}
