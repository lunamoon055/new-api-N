package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	commonRelay "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubscriptionTaskLedger(t *testing.T, suffix string, recordStatus string, requestID string) (*Task, string) {
	t.Helper()

	userID := 9700
	tokenID := 9700
	channelID := 9700
	subscriptionID := 9700
	tokenKey := "sk-task-ledger-" + suffix
	now := time.Now().Unix()

	require.NoError(t, DB.Create(&User{
		Id:        userID,
		Username:  "task_ledger_" + suffix,
		Quota:     0,
		UsedQuota: 1_000,
	}).Error)
	require.NoError(t, DB.Create(&Token{
		Id:          tokenID,
		UserId:      userID,
		Key:         tokenKey,
		Name:        "task-ledger",
		RemainQuota: 2_000,
		UsedQuota:   1_000,
	}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:        channelID,
		Name:      "task-ledger",
		Key:       "sk-upstream",
		UsedQuota: 1_000,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          subscriptionID,
		UserId:      userID,
		AmountTotal: 10_000,
		AmountUsed:  5_000,
		Status:      "active",
		StartTime:   now - 60,
		EndTime:     now + 3600,
	}).Error)
	if requestID != "" {
		require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
			RequestId:          requestID,
			UserId:             userID,
			UserSubscriptionId: subscriptionID,
			PreConsumed:        1_000,
			Status:             recordStatus,
		}).Error)
	}

	task := &Task{
		TaskID:             "task_subscription_ledger_" + suffix,
		UserId:             userID,
		ChannelId:          channelID,
		Quota:              1_000,
		Status:             TaskStatusFailure,
		Progress:           "100%",
		Group:              "default",
		Data:               json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
		BillingStatus:      TaskBillingStatusFinalizePending,
		BillingTargetQuota: 0,
		PrivateData: TaskPrivateData{
			RequestId:      requestID,
			BillingSource:  "subscription",
			SubscriptionId: subscriptionID,
			TokenId:        tokenID,
		},
	}
	require.NoError(t, DB.Create(task).Error)
	return task, tokenKey
}

func taskBillingTestEventPayload() *TaskBillingEventPayload {
	return &TaskBillingEventPayload{
		Content:    "async task finalized",
		ModelName:  "test-model",
		Group:      "default",
		LogEnabled: true,
	}
}

func loadSubscriptionLedgerState(t *testing.T, taskID int64, requestID string) (Task, UserSubscription, Token, SubscriptionPreConsumeRecord) {
	t.Helper()
	var task Task
	var subscription UserSubscription
	var token Token
	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.First(&task, taskID).Error)
	require.NoError(t, DB.First(&subscription, 9700).Error)
	require.NoError(t, DB.First(&token, 9700).Error)
	if requestID != "" {
		require.NoError(t, DB.Where("request_id = ?", requestID).First(&record).Error)
	}
	return task, subscription, token, record
}

func TestInitTaskPersistsRequestIDInPrivateData(t *testing.T) {
	truncateTables(t)

	requestID := "req_task_private_data"
	task := InitTask("", &commonRelay.RelayInfo{
		RequestId: requestID,
		UserId:    9701,
		TaskRelayInfo: &commonRelay.TaskRelayInfo{
			PublicTaskID: "task_request_id",
		},
	})
	require.NoError(t, task.Insert())

	var persisted Task
	require.NoError(t, DB.First(&persisted, task.ID).Error)
	assert.Equal(t, requestID, persisted.PrivateData.RequestId)
}

func TestApplyTaskBilling_SubscriptionFullRefundClaimsLedgerOnce(t *testing.T) {
	truncateTables(t)

	requestID := "req_full_refund"
	task, tokenKey := seedSubscriptionTaskLedger(
		t,
		"full_refund",
		SubscriptionPreConsumeStatusConsumed,
		requestID,
	)
	stale := *task

	result, err := ApplyTaskBilling(
		task,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusRefunded,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.NoError(t, err)
	assert.True(t, result.Applied)

	result, err = ApplyTaskBilling(
		&stale,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusRefunded,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.NoError(t, err)
	assert.False(t, result.Applied)

	persisted, subscription, token, record := loadSubscriptionLedgerState(t, task.ID, requestID)
	assert.Equal(t, TaskBillingStatusRefunded, persisted.BillingStatus)
	assert.Zero(t, persisted.Quota)
	assert.Equal(t, int64(4_000), subscription.AmountUsed)
	assert.Equal(t, SubscriptionPreConsumeStatusRefunded, record.Status)
	assert.Equal(t, 3_000, token.RemainQuota)
	assert.Zero(t, token.UsedQuota)

	var eventCount int64
	require.NoError(t, DB.Model(&TaskBillingEvent{}).
		Where("task_record_id = ?", task.ID).
		Count(&eventCount).Error)
	assert.Equal(t, int64(1), eventCount)
}

func TestApplyTaskBilling_SubscriptionAlreadyRefundedDoesNotAdjustAmountUsed(t *testing.T) {
	truncateTables(t)

	requestID := "req_already_refunded"
	task, tokenKey := seedSubscriptionTaskLedger(
		t,
		"already_refunded",
		SubscriptionPreConsumeStatusRefunded,
		requestID,
	)

	result, err := ApplyTaskBilling(
		task,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusRefunded,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.NoError(t, err)
	assert.True(t, result.Applied)

	persisted, subscription, token, record := loadSubscriptionLedgerState(t, task.ID, requestID)
	assert.Equal(t, TaskBillingStatusRefunded, persisted.BillingStatus)
	assert.Equal(t, int64(5_000), subscription.AmountUsed)
	assert.Equal(t, SubscriptionPreConsumeStatusRefunded, record.Status)
	assert.Equal(t, 3_000, token.RemainQuota)
	assert.Zero(t, token.UsedQuota)
}

func TestApplyTaskBilling_SubscriptionPartialSettlementKeepsLedgerConsumed(t *testing.T) {
	truncateTables(t)

	requestID := "req_partial_settlement"
	task, tokenKey := seedSubscriptionTaskLedger(
		t,
		"partial_settlement",
		SubscriptionPreConsumeStatusConsumed,
		requestID,
	)
	task.Status = TaskStatusSuccess
	task.BillingTargetQuota = 600
	require.NoError(t, DB.Model(&Task{}).
		Where("id = ?", task.ID).
		Updates(map[string]any{
			"status":               task.Status,
			"billing_target_quota": task.BillingTargetQuota,
		}).Error)

	result, err := ApplyTaskBilling(
		task,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusSettled,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.NoError(t, err)
	assert.True(t, result.Applied)

	persisted, subscription, token, record := loadSubscriptionLedgerState(t, task.ID, requestID)
	assert.Equal(t, TaskBillingStatusSettled, persisted.BillingStatus)
	assert.Equal(t, 600, persisted.Quota)
	assert.Equal(t, int64(4_600), subscription.AmountUsed)
	assert.Equal(t, SubscriptionPreConsumeStatusConsumed, record.Status)
	assert.Equal(t, 2_400, token.RemainQuota)
	assert.Equal(t, 600, token.UsedQuota)
}

func TestApplyTaskBilling_LegacySubscriptionWithoutRequestIDStillRefunds(t *testing.T) {
	truncateTables(t)

	task, tokenKey := seedSubscriptionTaskLedger(t, "legacy", "", "")
	result, err := ApplyTaskBilling(
		task,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusRefunded,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.NoError(t, err)
	assert.True(t, result.Applied)

	_, subscription, _, _ := loadSubscriptionLedgerState(t, task.ID, "")
	assert.Equal(t, int64(4_000), subscription.AmountUsed)
}

func TestApplyTaskBilling_SubscriptionLedgerOwnershipMismatchRollsBack(t *testing.T) {
	truncateTables(t)

	requestID := "req_owner_mismatch"
	task, tokenKey := seedSubscriptionTaskLedger(
		t,
		"owner_mismatch",
		SubscriptionPreConsumeStatusConsumed,
		requestID,
	)
	require.NoError(t, DB.Model(&SubscriptionPreConsumeRecord{}).
		Where("request_id = ?", requestID).
		Update("user_subscription_id", 9999).Error)

	result, err := ApplyTaskBilling(
		task,
		TaskBillingStatusFinalizePending,
		TaskBillingStatusRefunded,
		tokenKey,
		taskBillingTestEventPayload(),
	)
	require.Error(t, err)
	assert.False(t, result.Applied)
	assert.Contains(t, err.Error(), "owner mismatch")

	persisted, subscription, token, record := loadSubscriptionLedgerState(t, task.ID, requestID)
	assert.Equal(t, TaskBillingStatusFinalizePending, persisted.BillingStatus)
	assert.Equal(t, 1_000, persisted.Quota)
	assert.Equal(t, int64(5_000), subscription.AmountUsed)
	assert.Equal(t, SubscriptionPreConsumeStatusConsumed, record.Status)
	assert.Equal(t, 2_000, token.RemainQuota)

	var eventCount int64
	require.NoError(t, DB.Model(&TaskBillingEvent{}).
		Where("task_record_id = ?", task.ID).
		Count(&eventCount).Error)
	assert.Zero(t, eventCount, fmt.Sprintf("unexpected task billing event count: %d", eventCount))
}

func TestRefundSubscriptionPreConsume_CASIsIdempotent(t *testing.T) {
	truncateTables(t)

	requestID := "req_direct_refund"
	task, _ := seedSubscriptionTaskLedger(
		t,
		"direct_refund",
		SubscriptionPreConsumeStatusConsumed,
		requestID,
	)
	require.NoError(t, RefundSubscriptionPreConsume(requestID))
	require.NoError(t, RefundSubscriptionPreConsume(requestID))

	_, subscription, _, record := loadSubscriptionLedgerState(t, task.ID, requestID)
	assert.Equal(t, int64(4_000), subscription.AmountUsed)
	assert.Equal(t, SubscriptionPreConsumeStatusRefunded, record.Status)
}

func TestTaskPrivateDataRequestIDJSONRoundTrip(t *testing.T) {
	privateData := TaskPrivateData{RequestId: "req_json_round_trip"}
	value, err := privateData.Value()
	require.NoError(t, err)

	var restored TaskPrivateData
	require.NoError(t, restored.Scan(value))
	assert.Equal(t, privateData.RequestId, restored.RequestId)
	assert.Equal(t, common.GetJsonType(json.RawMessage(value.([]byte))), "object")
}
