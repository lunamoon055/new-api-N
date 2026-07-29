package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func persistBillingTask(t *testing.T, task *model.Task, taskID string, billingStatus model.TaskBillingStatus, targetQuota int) {
	t.Helper()
	task.TaskID = taskID
	task.BillingStatus = billingStatus
	task.BillingTargetQuota = targetQuota
	require.NoError(t, model.DB.Create(task).Error)
}

func reloadBillingTask(t *testing.T, id int64) *model.Task {
	t.Helper()
	var task model.Task
	require.NoError(t, model.DB.First(&task, id).Error)
	return &task
}

func TestPrepareTaskFinalBilling_SubmitPendingSuccessPreservesFrozenTarget(t *testing.T) {
	const preConsumed, frozenTarget = 1_000, 1_500

	task := makeTask(39, 39, preConsumed, 39, BillingSourceWallet, 0)
	task.Status = model.TaskStatusSuccess
	task.BillingStatus = model.TaskBillingStatusSubmitPending
	task.BillingTargetQuota = frozenTarget

	reason := PrepareTaskFinalBilling(
		&mockAdaptor{},
		task,
		&relaycommon.TaskInfo{Status: string(model.TaskStatusSuccess)},
	)

	assert.Equal(t, "保持预扣额度", reason)
	assert.Equal(t, model.TaskBillingStatusFinalizePending, task.BillingStatus)
	assert.Equal(t, frozenTarget, task.BillingTargetQuota)
	assert.Equal(t, preConsumed, task.Quota)
}

func TestApplyPendingTaskBilling_SubmitPendingAppliesOnce(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 40, 40, 40
	const userQuota, tokenQuota = 10_000, 5_000
	const preConsumed, targetQuota = 1_000, 1_500

	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-submit-pending", tokenQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	persistBillingTask(t, task, "task_submit_pending", model.TaskBillingStatusSubmitPending, targetQuota)
	staleTask := reloadBillingTask(t, task.ID)

	applied, err := ApplyPendingTaskBilling(ctx, task, "提交后计费调整")
	require.NoError(t, err)
	assert.True(t, applied)

	applied, err = ApplyPendingTaskBilling(ctx, staleTask, "重复提交调整")
	require.NoError(t, err)
	assert.False(t, applied)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskBillingStatusPending, persisted.BillingStatus)
	assert.Equal(t, targetQuota, persisted.Quota)
	assert.Equal(t, targetQuota, persisted.BillingTargetQuota)
	assert.Equal(t, userQuota-(targetQuota-preConsumed), getUserQuota(t, userID))
	assert.Equal(t, tokenQuota-(targetQuota-preConsumed), getTokenRemainQuota(t, tokenID))
	assert.Equal(t, targetQuota-preConsumed, getTokenUsedQuota(t, tokenID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestApplyPendingTaskBilling_FinalizeRefundIsIdempotent(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 41, 41, 41
	const userQuota, tokenQuota = 7_000, 2_000
	const preConsumed = 3_000

	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-finalize-refund", tokenQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatusFailure
	task.FailReason = "upstream failed"
	persistBillingTask(t, task, "task_finalize_refund", model.TaskBillingStatusFinalizePending, 0)
	staleTask := reloadBillingTask(t, task.ID)

	applied, err := ApplyPendingTaskBilling(ctx, task, task.FailReason)
	require.NoError(t, err)
	assert.True(t, applied)

	applied, err = ApplyPendingTaskBilling(ctx, staleTask, staleTask.FailReason)
	require.NoError(t, err)
	assert.False(t, applied)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskBillingStatusRefunded, persisted.BillingStatus)
	assert.Zero(t, persisted.Quota)
	assert.Equal(t, userQuota+preConsumed, getUserQuota(t, userID))
	assert.Equal(t, tokenQuota+preConsumed, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, -preConsumed, getTokenUsedQuota(t, tokenID))
	assert.Equal(t, int64(1), countLogs(t))

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
	assert.Equal(t, preConsumed, log.Quota)
}

func TestApplyPendingTaskBilling_SubscriptionInsufficientRollsBack(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID, subscriptionID = 42, 42, 42, 42
	const tokenQuota = 800
	const beforeQuota, targetQuota = 100, 300
	const subscriptionTotal, subscriptionUsed int64 = 1_000, 900

	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, "sk-subscription-rollback", tokenQuota)
	seedChannel(t, channelID)
	seedSubscription(t, subscriptionID, userID, subscriptionTotal, subscriptionUsed)

	task := makeTask(userID, channelID, beforeQuota, tokenID, BillingSourceSubscription, subscriptionID)
	task.Status = model.TaskStatusSuccess
	persistBillingTask(t, task, "task_subscription_rollback", model.TaskBillingStatusFinalizePending, targetQuota)

	applied, err := ApplyPendingTaskBilling(ctx, task, "补扣订阅额度")
	require.Error(t, err)
	assert.False(t, applied)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskBillingStatusFinalizePending, persisted.BillingStatus)
	assert.Equal(t, beforeQuota, persisted.Quota)
	assert.Equal(t, targetQuota, persisted.BillingTargetQuota)
	assert.Equal(t, subscriptionUsed, getSubscriptionUsed(t, subscriptionID))
	assert.Equal(t, tokenQuota, getTokenRemainQuota(t, tokenID))
	assert.Zero(t, getTokenUsedQuota(t, tokenID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestApplyPendingTaskBilling_ConcurrentAppliesOnce(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 43, 43, 43
	const userQuota, tokenQuota = 10_000, 5_000
	const beforeQuota, targetQuota = 1_000, 1_600
	const workerCount = 12

	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-concurrent-task-billing", tokenQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, beforeQuota, tokenID, BillingSourceWallet, 0)
	persistBillingTask(t, task, "task_concurrent_billing", model.TaskBillingStatusSubmitPending, targetQuota)

	copies := make([]*model.Task, workerCount)
	for i := range copies {
		copies[i] = reloadBillingTask(t, task.ID)
	}

	start := make(chan struct{})
	results := make(chan bool, workerCount)
	errors := make(chan error, workerCount)
	var wg sync.WaitGroup
	for _, taskCopy := range copies {
		wg.Add(1)
		go func(taskCopy *model.Task) {
			defer wg.Done()
			<-start
			applied, err := ApplyPendingTaskBilling(ctx, taskCopy, "并发提交调整")
			if err != nil {
				errors <- err
				return
			}
			results <- applied
		}(taskCopy)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
	appliedCount := 0
	for applied := range results {
		if applied {
			appliedCount++
		}
	}
	assert.Equal(t, 1, appliedCount)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskBillingStatusPending, persisted.BillingStatus)
	assert.Equal(t, targetQuota, persisted.Quota)
	assert.Equal(t, userQuota-(targetQuota-beforeQuota), getUserQuota(t, userID))
	assert.Equal(t, tokenQuota-(targetQuota-beforeQuota), getTokenRemainQuota(t, tokenID))
	assert.Equal(t, targetQuota-beforeQuota, getTokenUsedQuota(t, tokenID))
}

func TestRecoverPendingTaskBilling_RecoversSubmitAndFinalize(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const (
		submitUserID    = 44
		submitTokenID   = 44
		submitChannelID = 44
	)

	seedUser(t, submitUserID, 10_000)
	seedToken(t, submitTokenID, submitUserID, "sk-recover-submit", 5_000)
	seedChannel(t, submitChannelID)
	submitTask := makeTask(submitUserID, submitChannelID, 1_000, submitTokenID, BillingSourceWallet, 0)
	persistBillingTask(t, submitTask, "task_recover_submit", model.TaskBillingStatusSubmitPending, 1_400)

	refundTask := makeTask(submitUserID, submitChannelID, 2_500, submitTokenID, BillingSourceWallet, 0)
	refundTask.Status = model.TaskStatusFailure
	refundTask.FailReason = "recovered failure"
	persistBillingTask(t, refundTask, "task_recover_refund", model.TaskBillingStatusFinalizePending, 0)

	recoverPendingTaskBilling(ctx, 10)
	recoverPendingTaskBilling(ctx, 10)

	reloadedSubmit := reloadBillingTask(t, submitTask.ID)
	assert.Equal(t, model.TaskBillingStatusPending, reloadedSubmit.BillingStatus)
	assert.Equal(t, 1_400, reloadedSubmit.Quota)

	reloadedRefund := reloadBillingTask(t, refundTask.ID)
	assert.Equal(t, model.TaskBillingStatusRefunded, reloadedRefund.BillingStatus)
	assert.Zero(t, reloadedRefund.Quota)
	assert.Equal(t, 12_100, getUserQuota(t, submitUserID))
	assert.Equal(t, 7_100, getTokenRemainQuota(t, submitTokenID))
	assert.Equal(t, -2_100, getTokenUsedQuota(t, submitTokenID))

	assert.Empty(t, model.GetPendingTaskBilling(10))
	assert.Equal(t, int64(1), countLogs(t))
}

func TestRecoverPendingTaskBilling_BackoffPreventsFirstPageStarvation(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const (
		blockedUserID       = 45
		blockedSubscription = 45
		goodUserID          = 46
		goodTokenID         = 46
		goodChannelID       = 46
	)

	seedUser(t, blockedUserID, 0)
	seedSubscription(t, blockedSubscription, blockedUserID, 100, 95)

	blockedTasks := make([]*model.Task, 0, 2)
	for i := 0; i < 2; i++ {
		task := makeTask(blockedUserID, 45, 10, 0, BillingSourceSubscription, blockedSubscription)
		task.Status = model.TaskStatusSuccess
		persistBillingTask(
			t,
			task,
			"task_blocked_retry_"+string(rune('a'+i)),
			model.TaskBillingStatusFinalizePending,
			20,
		)
		blockedTasks = append(blockedTasks, task)
	}

	seedUser(t, goodUserID, 1_000)
	seedToken(t, goodTokenID, goodUserID, "sk-recovery-not-starved", 1_000)
	seedChannel(t, goodChannelID)
	goodTask := makeTask(goodUserID, goodChannelID, 300, goodTokenID, BillingSourceWallet, 0)
	goodTask.Status = model.TaskStatusFailure
	goodTask.FailReason = "refund after blocked first page"
	persistBillingTask(t, goodTask, "task_recovery_not_starved", model.TaskBillingStatusFinalizePending, 0)

	// The first page contains only permanently failing adjustments.
	recoverPendingTaskBilling(ctx, 2)
	for _, task := range blockedTasks {
		persisted := reloadBillingTask(t, task.ID)
		assert.Equal(t, model.TaskBillingStatusFinalizePending, persisted.BillingStatus)
		assert.Equal(t, 1, persisted.BillingRetryCount)
		assert.Greater(t, persisted.BillingNextRetryAt, time.Now().Unix())
		assert.NotEmpty(t, persisted.BillingLastError)
	}
	assert.Equal(t, model.TaskBillingStatusFinalizePending, reloadBillingTask(t, goodTask.ID).BillingStatus)

	// Backoff removes the failed first page from the immediately eligible set,
	// allowing a later row to make progress on the next recovery pass.
	recoverPendingTaskBilling(ctx, 2)

	persistedGood := reloadBillingTask(t, goodTask.ID)
	assert.Equal(t, model.TaskBillingStatusRefunded, persistedGood.BillingStatus)
	assert.Zero(t, persistedGood.Quota)
	assert.Equal(t, 1_300, getUserQuota(t, goodUserID))
	assert.Equal(t, 1_300, getTokenRemainQuota(t, goodTokenID))
	assert.Equal(t, int64(1), countLogs(t))
}
