package model

import (
	"time"

	"gorm.io/gorm/clause"
)

const (
	CreationIdempotencyProcessing = "processing"
	CreationIdempotencyCompleted  = "completed"
)

// CreationIdempotency stores the response associated with a token-scoped
// Idempotency-Key. The request hash prevents the same key from being reused for
// a different payload, while the unique index prevents concurrent duplicate
// submissions on SQLite, MySQL and PostgreSQL.
type CreationIdempotency struct {
	ID                int64  `gorm:"primaryKey;autoIncrement"`
	TokenID           int    `gorm:"not null;uniqueIndex:idx_creation_idempotency_token_key,priority:1"`
	UserID            int    `gorm:"not null;index"`
	Key               string `gorm:"size:128;not null;uniqueIndex:idx_creation_idempotency_token_key,priority:2"`
	RequestHash       string `gorm:"size:64;not null"`
	Status            string `gorm:"size:24;not null;index"`
	ResponseStatus    int    `gorm:"not null;default:0"`
	ResponseHeaders   []byte
	ResponseBody      []byte
	ResponseTruncated bool  `gorm:"not null;default:false"`
	CreatedAt         int64 `gorm:"not null;index"`
	UpdatedAt         int64 `gorm:"not null"`
	ExpiresAt         int64 `gorm:"not null;index"`
}

// ReserveCreationIdempotency atomically claims a token/key pair. created is
// true only for the request that is allowed to contact the upstream provider.
func ReserveCreationIdempotency(tokenID, userID int, key, requestHash string) (entry *CreationIdempotency, created bool, err error) {
	now := time.Now().Unix()
	entry = &CreationIdempotency{
		TokenID:     tokenID,
		UserID:      userID,
		Key:         key,
		RequestHash: requestHash,
		Status:      CreationIdempotencyProcessing,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now + 7*24*60*60,
	}
	result := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return entry, true, nil
	}
	entry = &CreationIdempotency{}
	if err := DB.Where(&CreationIdempotency{TokenID: tokenID, Key: key}).First(entry).Error; err != nil {
		return nil, false, err
	}
	return entry, false, nil
}

func GetCreationIdempotency(tokenID int, key string) (*CreationIdempotency, error) {
	entry := &CreationIdempotency{}
	if err := DB.Where(&CreationIdempotency{TokenID: tokenID, Key: key}).First(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

func CompleteCreationIdempotency(id int64, requestHash string, responseStatus int, responseHeaders, responseBody []byte, truncated bool) error {
	now := time.Now().Unix()
	return DB.Model(&CreationIdempotency{}).
		Where("id = ? AND request_hash = ? AND status = ?", id, requestHash, CreationIdempotencyProcessing).
		Updates(map[string]any{
			"status":             CreationIdempotencyCompleted,
			"response_status":    responseStatus,
			"response_headers":   responseHeaders,
			"response_body":      responseBody,
			"response_truncated": truncated,
			"updated_at":         now,
		}).Error
}

func DeleteExpiredCreationIdempotency(now int64) error {
	return DB.Where("expires_at < ?", now).Delete(&CreationIdempotency{}).Error
}
