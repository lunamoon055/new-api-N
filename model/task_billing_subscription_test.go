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

func seedPreConsumeSubscription(t *testing.T, id int, amountTotal int64, amountUsed int64) UserSubscription {
	t.Helper()
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&SubscriptionPlan{
		Id:               id,
		Title:            fmt.Sprintf("pre-consume-%d", id),
		TotalAmount:      amountTotal,
		QuotaResetPeriod: SubscriptionResetNever,
	}).Error)
	subscription := UserSubscription{
		Id:          id,
		UserId:      id,
		PlanId:      id,
		AmountTotal: amountTotal,
		AmountUsed:  amountUsed,
		Status:      "active",
		StartTime:   now - 60,
		EndTime:     now + 3600,
	}
	require.NoError(t, DB.Create(&subscription).Error)
	return subscription
}

func TestPreConsumeUserSubscriptionRejectsCrossUserRequestReuse(t *testing.T) {
	truncateTables(t)

	subscription := seedPreConsumeSubscription(t, 9801, 1_000, 100)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId:          "req_cross_user",
		UserId:             subscription.UserId,
		UserSubscriptionId: subscription.Id,
		PreConsumed:        100,
		Status:             SubscriptionPreConsumeStatusConsumed,
	}).Error)

	_, err := PreConsumeUserSubscription(
		"req_cross_user",
		subscription.UserId+1,
		"test-model",
		0,
		100,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner mismatch")

	var persisted UserSubscription
	require.NoError(t, DB.First(&persisted, subscription.Id).Error)
	assert.Equal(t, int64(100), persisted.AmountUsed)
}

func TestPreConsumeUserSubscriptionRejectsAmountMismatch(t *testing.T) {
	truncateTables(t)

	subscription := seedPreConsumeSubscription(t, 9802, 1_000, 100)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId:          "req_amount_mismatch",
		UserId:             subscription.UserId,
		UserSubscriptionId: subscription.Id,
		PreConsumed:        100,
		Status:             SubscriptionPreConsumeStatusConsumed,
	}).Error)

	_, err := PreConsumeUserSubscription(
		"req_amount_mismatch",
		subscription.UserId,
		"test-model",
		0,
		101,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount mismatch")

	var persisted UserSubscription
	require.NoError(t, DB.First(&persisted, subscription.Id).Error)
	assert.Equal(t, int64(100), persisted.AmountUsed)
}

func TestPreConsumeUserSubscriptionConcurrentReservationsDoNotOverspend(t *testing.T) {
	truncateTables(t)

	subscription := seedPreConsumeSubscription(t, 9803, 100, 0)
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		requestID := fmt.Sprintf("req_concurrent_reserve_%d", i)
		go func() {
			<-start
			_, err := PreConsumeUserSubscription(
				requestID,
				subscription.UserId,
				"test-model",
				0,
				75,
			)
			results <- err
		}()
	}
	close(start)

	successCount := 0
	failureCount := 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successCount++
			continue
		}
		failureCount++
		assert.Contains(t, err.Error(), "quota insufficient")
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, failureCount)

	var persisted UserSubscription
	require.NoError(t, DB.First(&persisted, subscription.Id).Error)
	assert.Equal(t, int64(75), persisted.AmountUsed)

	var recordCount int64
	require.NoError(t, DB.Model(&SubscriptionPreConsumeRecord{}).Count(&recordCount).Error)
	assert.Equal(t, int64(1), recordCount)
}

func TestPreConsumeUserSubscriptionRejectsAmountUsedOverflow(t *testing.T) {
	truncateTables(t)

	maxInt64 := int64(^uint64(0) >> 1)
	subscription := seedPreConsumeSubscription(t, 9804, 0, maxInt64-5)

	_, err := PreConsumeUserSubscription(
		"req_amount_used_overflow",
		subscription.UserId,
		"test-model",
		0,
		10,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota insufficient")

	var persisted UserSubscription
	require.NoError(t, DB.First(&persisted, subscription.Id).Error)
	assert.Equal(t, maxInt64-5, persisted.AmountUsed)

	var recordCount int64
	require.NoError(t, DB.Model(&SubscriptionPreConsumeRecord{}).
		Where("request_id = ?", "req_amount_used_overflow").
		Count(&recordCount).Error)
	assert.Zero(t, recordCount)
}

func TestInitTaskPersistsRequestIDInPrivateData(t *testing.T) {
	truncateTables(t)

	requestID := "req_task_private_data"
	task := InitTask("", &commonRelay.RelayInfo{
		RequestId: requestID,
		UserId:    9701,
		ChannelMeta: &commonRelay.ChannelMeta{
			ChannelId: 9701,
		},
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
