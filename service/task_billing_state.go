package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"gorm.io/gorm"
)

// PrepareTaskFinalBilling freezes the target charge before a terminal task
// status is persisted. The caller must save these fields with the terminal
// status using the existing task CAS.
func PrepareTaskFinalBilling(adaptor TaskPollingAdaptor, task *model.Task, taskResult *relaycommon.TaskInfo) string {
	baseQuota := task.Quota
	if task.BillingStatus == model.TaskBillingStatusSubmitPending {
		// The submit phase may not have been applied yet. Preserve the frozen
		// post-submit target so a fast terminal fetch cannot fall back to the
		// original pre-consume estimate.
		baseQuota = task.BillingTargetQuota
	}
	task.BillingStatus = model.TaskBillingStatusFinalizePending
	if task.Status == model.TaskStatusFailure {
		task.BillingTargetQuota = 0
		return task.FailReason
	}

	task.BillingTargetQuota = baseQuota
	if billingContext := task.PrivateData.BillingContext; billingContext != nil && billingContext.PerCallBilling {
		return "按次计费"
	}
	if adaptor != nil {
		if actualQuota := adaptor.AdjustBillingOnComplete(task, taskResult); actualQuota > 0 {
			task.BillingTargetQuota = actualQuota
			return "adaptor计费调整"
		}
	}
	if taskResult != nil && taskResult.TotalTokens > 0 {
		if actualQuota, reason, ok := CalculateTaskQuotaByTokens(task, taskResult.TotalTokens); ok {
			task.BillingTargetQuota = actualQuota
			return reason
		}
	}
	return "保持预扣额度"
}

// ApplyPendingTaskBilling performs one idempotent billing-state transition.
// Funds, token quota, task quota and BillingStatus are committed together.
func ApplyPendingTaskBilling(ctx context.Context, task *model.Task, reason string) (bool, error) {
	if task == nil {
		return false, nil
	}
	expectedStatus := task.BillingStatus
	var nextStatus model.TaskBillingStatus
	switch expectedStatus {
	case model.TaskBillingStatusSubmitPending:
		nextStatus = model.TaskBillingStatusPending
	case model.TaskBillingStatusFinalizePending:
		if task.Status == model.TaskStatusFailure && task.BillingTargetQuota == 0 {
			nextStatus = model.TaskBillingStatusRefunded
		} else {
			nextStatus = model.TaskBillingStatusSettled
		}
	default:
		return false, nil
	}

	tokenKey := ""
	if common.RedisEnabled && task.PrivateData.TokenId > 0 {
		tokenKey = resolveTokenKey(ctx, task.PrivateData.TokenId, task.UserId, task.TaskID)
		if tokenKey == "" {
			return false, fmt.Errorf("cannot resolve token cache key for task %s", task.TaskID)
		}
	}
	var eventPayload *model.TaskBillingEventPayload
	if expectedStatus == model.TaskBillingStatusFinalizePending {
		delta := task.BillingTargetQuota - task.Quota
		other := taskBillingOther(task)
		other["task_id"] = task.TaskID
		other["pre_consumed_quota"] = task.Quota
		other["actual_quota"] = task.BillingTargetQuota
		if reason != "" {
			other["reason"] = reason
		}
		eventPayload = &model.TaskBillingEventPayload{
			Content:    reason,
			ModelName:  taskModelName(task),
			Group:      task.Group,
			Other:      common.MapToJsonStr(other),
			LogEnabled: delta != 0 && (delta < 0 || common.LogConsumeEnabled),
		}
	}
	result, err := model.ApplyTaskBilling(task, expectedStatus, nextStatus, tokenKey, eventPayload)
	if err != nil {
		return false, err
	}
	if !result.Applied {
		return false, nil
	}

	task.Quota = result.TargetQuota
	task.BillingStatus = result.NextStatus
	if expectedStatus == model.TaskBillingStatusFinalizePending {
		if result.Delta == 0 {
			logger.LogInfo(ctx, fmt.Sprintf(
				"任务 %s 最终额度与预扣一致（%s）",
				task.TaskID,
				logger.LogQuota(result.TargetQuota),
			))
		}
		deliverAppliedTaskBillingEvent(ctx, result.EventID)
	}
	return true, nil
}

func deliverAppliedTaskBillingEvent(ctx context.Context, eventID string) {
	event, err := model.GetTaskBillingEventByEventID(eventID)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("读取任务计费事件 %s 失败: %s", eventID, err.Error()))
		return
	}
	if err := deliverTaskBillingEvent(event); err != nil {
		scheduleTaskBillingEventRetry(event, err)
		logger.LogError(ctx, fmt.Sprintf(
			"投递任务 %s 计费事件失败: %s",
			event.TaskID,
			err.Error(),
		))
	}
}

func deliverTaskBillingEvent(event *model.TaskBillingEvent) error {
	if err := model.RecordTaskBillingEventLog(event); err != nil {
		return err
	}
	return model.MarkTaskBillingEventDelivered(event.EventID)
}

func recoverPendingTaskBillingEvents(ctx context.Context, limit int) {
	events, err := model.QueryPendingTaskBillingEvents(limit)
	if err != nil {
		logger.LogError(ctx, "查询待投递任务计费事件失败: "+err.Error())
		return
	}
	for _, event := range events {
		if err := deliverTaskBillingEvent(event); err != nil {
			scheduleTaskBillingEventRetry(event, err)
			logger.LogError(ctx, fmt.Sprintf(
				"恢复投递任务 %s 计费事件失败: %s",
				event.TaskID,
				err.Error(),
			))
			continue
		}
		logger.LogInfo(ctx, fmt.Sprintf("已投递任务 %s 的计费事件", event.TaskID))
	}
}

func recoverPendingTaskBilling(ctx context.Context, limit int) {
	recoverPendingTaskBillingEvents(ctx, limit)
	tasks, err := model.QueryPendingTaskBilling(limit)
	if err != nil {
		logger.LogError(ctx, "查询待恢复任务计费失败: "+err.Error())
		return
	}
	for _, task := range tasks {
		reason := "恢复未完成的任务计费"
		if task.BillingStatus == model.TaskBillingStatusFinalizePending && task.FailReason != "" {
			reason = task.FailReason
		}
		applied, err := ApplyPendingTaskBilling(ctx, task, reason)
		if err != nil {
			scheduleTaskBillingRetry(task, err)
			logger.LogError(ctx, fmt.Sprintf(
				"恢复任务 %s 计费失败: %s",
				task.TaskID,
				err.Error(),
			))
			continue
		}
		if applied {
			logger.LogInfo(ctx, fmt.Sprintf("已恢复任务 %s 的未完成计费", task.TaskID))
		}
	}
	recoverPendingTaskBillingEvents(ctx, limit)
}

func scheduleTaskBillingRetry(task *model.Task, billingErr error) {
	if task == nil || billingErr == nil {
		return
	}
	retryCount := task.BillingRetryCount
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 6 {
		retryCount = 6
	}
	delay := 15 * time.Second * time.Duration(1<<retryCount)
	if delay > 15*time.Minute {
		delay = 15 * time.Minute
	}
	if err := model.ScheduleTaskBillingRetry(
		task.ID,
		task.BillingStatus,
		time.Now().Add(delay).Unix(),
		billingErr.Error(),
	); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		common.SysLog("记录任务计费重试失败: " + err.Error())
	}
}

func scheduleTaskBillingEventRetry(event *model.TaskBillingEvent, deliveryErr error) {
	if event == nil || deliveryErr == nil {
		return
	}
	retryCount := event.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 6 {
		retryCount = 6
	}
	delay := 15 * time.Second * time.Duration(1<<retryCount)
	if delay > 15*time.Minute {
		delay = 15 * time.Minute
	}
	if err := model.ScheduleTaskBillingEventRetry(
		event.EventID,
		time.Now().Add(delay).Unix(),
		deliveryErr.Error(),
	); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		common.SysLog("记录任务计费事件重试失败: " + err.Error())
	}
}
