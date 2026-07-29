package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var errTaskBillingCASLost = errors.New("task billing state changed")

type TaskBillingApplyResult struct {
	Applied     bool
	BeforeQuota int
	TargetQuota int
	Delta       int
	NextStatus  TaskBillingStatus
	EventID     string
}

// ApplyTaskBilling atomically adjusts the task's funding source and token
// quota, then advances its persistent billing state. Pending batch quota
// deltas are reconciled in the same transaction.
func ApplyTaskBilling(
	task *Task,
	expectedStatus, nextStatus TaskBillingStatus,
	tokenKey string,
	eventPayload *TaskBillingEventPayload,
) (TaskBillingApplyResult, error) {
	result := TaskBillingApplyResult{}
	if task == nil || task.ID <= 0 {
		return result, errors.New("invalid task")
	}
	if expectedStatus != TaskBillingStatusSubmitPending && expectedStatus != TaskBillingStatusFinalizePending {
		return result, fmt.Errorf("unsupported task billing status %q", expectedStatus)
	}
	if nextStatus != TaskBillingStatusPending &&
		nextStatus != TaskBillingStatusSettled &&
		nextStatus != TaskBillingStatusRefunded {
		return result, fmt.Errorf("unsupported next task billing status %q", nextStatus)
	}
	if task.BillingTargetQuota < 0 {
		return result, errors.New("task billing target quota cannot be negative")
	}
	if expectedStatus == TaskBillingStatusFinalizePending && eventPayload == nil {
		return result, errors.New("final task billing requires an outbox event")
	}
	if expectedStatus != TaskBillingStatusFinalizePending && eventPayload != nil {
		return result, errors.New("task billing outbox event is only supported during finalization")
	}

	beforeQuota := task.Quota
	targetQuota := task.BillingTargetQuota
	delta := targetQuota - beforeQuota
	withToken := task.PrivateData.TokenId > 0
	isSubscription := task.PrivateData.BillingSource == "subscription" &&
		task.PrivateData.SubscriptionId > 0
	isFullSubscriptionRefund := isSubscription &&
		expectedStatus == TaskBillingStatusFinalizePending &&
		nextStatus == TaskBillingStatusRefunded &&
		targetQuota == 0

	operation := func(tx *gorm.DB) error {
		update := tx.Model(&Task{}).
			Where(
				"id = ? AND billing_status = ? AND quota = ? AND billing_target_quota = ?",
				task.ID,
				expectedStatus,
				beforeQuota,
				targetQuota,
			).
			Updates(map[string]any{
				"billing_status":        nextStatus,
				"quota":                 targetQuota,
				"billing_retry_count":   0,
				"billing_next_retry_at": 0,
				"billing_last_error":    "",
				"updated_at":            common.GetTimestamp(),
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errTaskBillingCASLost
		}

		adjustSubscription := delta != 0
		if isFullSubscriptionRefund && strings.TrimSpace(task.PrivateData.RequestId) != "" {
			claim, err := claimSubscriptionPreConsumeRefundTx(
				tx,
				task.PrivateData.RequestId,
				task.UserId,
				task.PrivateData.SubscriptionId,
			)
			if err != nil {
				return err
			}
			if claim.Found {
				// Only the transaction that moves consumed->refunded owns the
				// amount_used refund. A prior refunded record means that
				// adjustment has already been applied elsewhere.
				adjustSubscription = claim.Claimed && delta != 0
			}
		}

		if delta != 0 {
			if isSubscription {
				if adjustSubscription {
					if err := adjustTaskSubscriptionQuota(tx, task.PrivateData.SubscriptionId, int64(delta)); err != nil {
						return err
					}
				}
			} else if err := adjustUserQuota(tx, task.UserId, -delta); err != nil {
				return err
			}
			if withToken {
				if err := adjustTokenQuota(tx, task.PrivateData.TokenId, -delta); err != nil {
					return err
				}
			}
		}
		if expectedStatus == TaskBillingStatusFinalizePending {
			if err := applyTaskBillingUsageDelta(tx, task.UserId, task.ChannelId, delta); err != nil {
				return err
			}
			event := &TaskBillingEvent{
				EventID:      fmt.Sprintf("task-billing-finalize-%d", task.ID),
				TaskRecordID: task.ID,
				TaskID:       task.TaskID,
				UserID:       task.UserId,
				ChannelID:    task.ChannelId,
				TokenID:      task.PrivateData.TokenId,
				LogType:      LogTypeConsume,
				LogEnabled:   eventPayload.LogEnabled,
				Content:      eventPayload.Content,
				ModelName:    eventPayload.ModelName,
				Quota:        delta,
				Delta:        delta,
				BeforeQuota:  beforeQuota,
				TargetQuota:  targetQuota,
				Group:        eventPayload.Group,
				Other:        eventPayload.Other,
				Status:       TaskBillingEventStatusPending,
				CreatedAt:    common.GetTimestamp(),
				UpdatedAt:    common.GetTimestamp(),
			}
			if delta < 0 {
				event.LogType = LogTypeRefund
				event.Quota = -delta
			}
			if err := tx.Create(event).Error; err != nil {
				return err
			}
			result.EventID = event.EventID
		}
		return nil
	}

	var err error
	if isSubscription {
		if withToken {
			err = withReconciledTokenQuota(task.PrivateData.TokenId, tokenKey, operation)
		} else {
			err = DB.Transaction(operation)
		}
	} else {
		err = withReconciledWalletQuota(
			task.UserId,
			task.PrivateData.TokenId,
			tokenKey,
			withToken,
			operation,
		)
	}
	if errors.Is(err, errTaskBillingCASLost) {
		return result, nil
	}
	if err != nil {
		return result, err
	}

	result.Applied = true
	result.BeforeQuota = beforeQuota
	result.TargetQuota = targetQuota
	result.Delta = delta
	result.NextStatus = nextStatus
	return result, nil
}

func applyTaskBillingUsageDelta(tx *gorm.DB, userID, channelID, delta int) error {
	if delta == 0 {
		return nil
	}
	userUpdate := tx.Unscoped().
		Model(&User{}).
		Where("id = ?", userID).
		Update("used_quota", gorm.Expr("used_quota + ?", delta))
	if userUpdate.Error != nil {
		return userUpdate.Error
	}
	if userUpdate.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	channelUpdate := tx.Model(&Channel{}).
		Where("id = ?", channelID).
		Update("used_quota", gorm.Expr("used_quota + ?", delta))
	if channelUpdate.Error != nil {
		return channelUpdate.Error
	}
	if channelUpdate.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func adjustTaskSubscriptionQuota(tx *gorm.DB, subscriptionID int, delta int64) error {
	query := tx.Model(&UserSubscription{}).Where("id = ?", subscriptionID)
	var update *gorm.DB
	if delta > 0 {
		update = query.
			Where("(amount_total <= 0 OR amount_used + ? <= amount_total)", delta).
			Update("amount_used", gorm.Expr("amount_used + ?", delta))
	} else {
		update = query.Update(
			"amount_used",
			gorm.Expr(
				"CASE WHEN amount_used + ? < 0 THEN 0 ELSE amount_used + ? END",
				delta,
				delta,
			),
		)
	}
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return fmt.Errorf("subscription quota adjustment rejected for subscription %d", subscriptionID)
	}
	return nil
}
