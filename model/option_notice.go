package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const noticePublicationOptionKey = "NoticePublication"

var noticeOptionMu sync.Mutex

type noticePublication struct {
	ContentHash string `json:"content_hash"`
	PublishedAt int64  `json:"published_at"`
}

func noticeContentHash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

// GetNoticeWithPublication reads a matching content/time pair. Legacy notices
// without publication metadata remain available but are not treated as new.
func GetNoticeWithPublication() (string, int64) {
	common.OptionMapRWMutex.RLock()
	content := common.OptionMap["Notice"]
	publicationJSON := common.OptionMap[noticePublicationOptionKey]
	common.OptionMapRWMutex.RUnlock()
	return content, noticePublicationTime(content, publicationJSON)
}

func noticePublicationTime(content, publicationJSON string) int64 {
	var publication noticePublication
	if strings.TrimSpace(content) == "" ||
		common.UnmarshalJsonStr(publicationJSON, &publication) != nil ||
		publication.ContentHash != noticeContentHash(content) {
		return 0
	}
	return publication.PublishedAt
}

func updateNoticeOption(content string, now time.Time) error {
	noticeOptionMu.Lock()
	defer noticeOptionMu.Unlock()

	publicationJSON := ""
	if err := DB.Transaction(func(tx *gorm.DB) error {
		var previous []Option
		if err := tx.Where("key IN ?", []string{"Notice", noticePublicationOptionKey}).Find(&previous).Error; err != nil {
			return err
		}
		values := make(map[string]string, len(previous))
		for _, option := range previous {
			values[option.Key] = option.Value
		}
		publishedAt := noticePublicationTime(values["Notice"], values[noticePublicationOptionKey])
		if strings.TrimSpace(content) == "" {
			publishedAt = 0
		} else if strings.TrimSpace(content) != strings.TrimSpace(values["Notice"]) {
			publishedAt = now.Unix()
		}
		encoded, err := common.Marshal(noticePublication{
			ContentHash: noticeContentHash(content),
			PublishedAt: publishedAt,
		})
		if err != nil {
			return err
		}
		publicationJSON = string(encoded)
		for _, option := range []Option{
			{Key: "Notice", Value: content},
			{Key: noticePublicationOptionKey, Value: publicationJSON},
		} {
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap["Notice"] = content
	common.OptionMap[noticePublicationOptionKey] = publicationJSON
	common.OptionMapRWMutex.Unlock()
	return nil
}
