package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type TaskBillingEventStatus string

const (
	TaskBillingEventStatusPending   TaskBillingEventStatus = "PENDING"
	TaskBillingEventStatusDelivered TaskBillingEventStatus = "DELIVERED"
)

// TaskBillingEvent is a transactional outbox record for the final adjustment
// of an asynchronous task. It is stored in the primary database so its
// creation and the corresponding quota/statistics changes commit together.
type TaskBillingEvent struct {
	ID           int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID      string                 `json:"event_id" gorm:"type:varchar(64);not null;uniqueIndex:idx_task_billing_event_id"`
	TaskRecordID int64                  `json:"task_record_id" gorm:"not null;uniqueIndex:idx_task_billing_event_task"`
	TaskID       string                 `json:"task_id" gorm:"type:varchar(191);not null;index"`
	UserID       int                    `json:"user_id" gorm:"not null;index"`
	ChannelID    int                    `json:"channel_id" gorm:"not null;index"`
	TokenID      int                    `json:"token_id" gorm:"not null;default:0"`
	LogType      int                    `json:"log_type" gorm:"not null"`
	LogEnabled   bool                   `json:"log_enabled" gorm:"not null;default:false"`
	Content      string                 `json:"content" gorm:"type:text"`
	ModelName    string                 `json:"model_name" gorm:"type:varchar(191);not null;default:''"`
	Quota        int                    `json:"quota" gorm:"not null;default:0"`
	Delta        int                    `json:"delta" gorm:"not null;default:0"`
	BeforeQuota  int                    `json:"before_quota" gorm:"not null;default:0"`
	TargetQuota  int                    `json:"target_quota" gorm:"not null;default:0"`
	Group        string                 `json:"group" gorm:"type:varchar(50);not null;default:''"`
	Other        string                 `json:"other" gorm:"type:text"`
	Status       TaskBillingEventStatus `json:"status" gorm:"type:varchar(16);not null;default:PENDING;index:idx_task_billing_event_pending,priority:1"`
	RetryCount   int                    `json:"retry_count" gorm:"not null;default:0"`
	NextRetryAt  int64                  `json:"next_retry_at" gorm:"not null;default:0;index:idx_task_billing_event_pending,priority:2"`
	LastError    string                 `json:"last_error" gorm:"type:varchar(512);not null;default:''"`
	CreatedAt    int64                  `json:"created_at" gorm:"not null;index"`
	UpdatedAt    int64                  `json:"updated_at" gorm:"not null"`
}

type TaskBillingEventPayload struct {
	Content    string
	ModelName  string
	Group      string
	Other      string
	LogEnabled bool
}

func GetTaskBillingEventByEventID(eventID string) (*TaskBillingEvent, error) {
	if eventID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var event TaskBillingEvent
	if err := DB.Where("event_id = ?", eventID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func QueryPendingTaskBillingEvents(limit int) ([]*TaskBillingEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	var events []*TaskBillingEvent
	err := DB.
		Where("status = ? AND next_retry_at <= ?", TaskBillingEventStatusPending, common.GetTimestamp()).
		Order("next_retry_at, id").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func MarkTaskBillingEventDelivered(eventID string) error {
	if eventID == "" {
		return gorm.ErrRecordNotFound
	}
	result := DB.Model(&TaskBillingEvent{}).
		Where("event_id = ? AND status = ?", eventID, TaskBillingEventStatusPending).
		Updates(map[string]any{
			"status":        TaskBillingEventStatusDelivered,
			"retry_count":   0,
			"next_retry_at": 0,
			"last_error":    "",
			"updated_at":    common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}

	var status TaskBillingEventStatus
	err := DB.Model(&TaskBillingEvent{}).
		Where("event_id = ?", eventID).
		Select("status").
		Scan(&status).Error
	if err != nil {
		return err
	}
	if status == TaskBillingEventStatusDelivered {
		return nil
	}
	return gorm.ErrRecordNotFound
}

func ScheduleTaskBillingEventRetry(eventID string, nextRetryAt int64, lastError string) error {
	if eventID == "" {
		return gorm.ErrRecordNotFound
	}
	if len(lastError) > 512 {
		lastError = lastError[:512]
	}
	result := DB.Model(&TaskBillingEvent{}).
		Where("event_id = ? AND status = ?", eventID, TaskBillingEventStatusPending).
		Updates(map[string]any{
			"retry_count":   gorm.Expr("retry_count + ?", 1),
			"next_retry_at": nextRetryAt,
			"last_error":    lastError,
			"updated_at":    common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
