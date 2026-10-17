package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAgentApplicationRequiresReview(t *testing.T) {
	truncateTables(t)
	require.NoError(t, SubmitAgentApplication(701, "I will share a direct link"))
	application, err := GetAgentApplication(701)
	require.NoError(t, err)
	require.Equal(t, AgentStatusPending, application.Status)
	require.False(t, IsApprovedAgent(701))
	require.Error(t, SubmitAgentApplication(701, "duplicate"))
	require.NoError(t, ReviewAgentApplication(application.ID, 999, AgentStatusApproved, "approved"))
	require.True(t, IsApprovedAgent(701))
	require.Error(t, ReviewAgentApplication(application.ID, 999, AgentStatusApproved, "duplicate review"))
	require.NoError(t, ReviewAgentApplication(application.ID, 999, AgentStatusRevoked, "revoked"))
	require.False(t, IsApprovedAgent(701))
	require.NoError(t, SubmitAgentApplication(701, "reapply"))
}

func TestAgentAttributionIsCapturedAtRegistration(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&User{Id: 703, Username: "approved_referrer", AffCode: "approved-referrer"}).Error)
	require.NoError(t, SubmitAgentApplication(703, "direct promotion"))
	application, err := GetAgentApplication(703)
	require.NoError(t, err)
	require.NoError(t, ReviewAgentApplication(application.ID, 999, AgentStatusApproved, ""))
	child := &User{Username: "agent_child"}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return child.InsertWithTx(tx, 703)
	}))
	require.Equal(t, 703, child.InviterId)
	require.Equal(t, "agent", child.ReferralKind)
	require.Greater(t, child.ReferralApprovedAt, int64(0))
	require.NoError(t, ReviewAgentApplication(application.ID, 999, AgentStatusRevoked, "revoked"))
	second := &User{Username: "ordinary_child"}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return second.InsertWithTx(tx, 703)
	}))
	require.Equal(t, "promotion", second.ReferralKind)
}

func TestReferralRewardMarginCaps(t *testing.T) {
	fee, margin, reward, platform := referralRewardAmount(26000, 18000, 0, true)
	require.EqualValues(t, 780, fee)
	require.EqualValues(t, 7220, margin)
	require.EqualValues(t, 2080, reward)
	require.EqualValues(t, 5140, platform)

	_, _, reward, _ = referralRewardAmount(47000, 42000, 0, true)
	require.Zero(t, reward) // platform protection closes a low-margin model

	_, _, reward, _ = referralRewardAmount(26000, 18000, 0, false)
	require.EqualValues(t, 1300, reward) // 20% margin, capped at 5% of sale
}

func TestVerifiedAgentRewardRequiresSuccessfulPaidTask(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&User{Id: 711, Username: "reward_referrer", AffCode: "reward-referrer"}).Error)
	require.NoError(t, DB.Create(&User{Id: 712, Username: "reward_customer", AffCode: "reward-customer", InviterId: 711, ReferralKind: "agent", ReferralApprovedAt: now - 30*24*3600, UsedQuota: 260}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 1, Name: "reward-channel", Key: "sk-reward", UsedQuota: 260}).Error)
	require.NoError(t, DB.Create(&AgentApplication{
		UserID: 711, Status: AgentStatusApproved, AppliedAt: now - 40*24*3600,
		ReviewedAt: now - 30*24*3600,
	}).Error)
	task := &Task{
		TaskID: "reward-task-1", UserId: 712, ChannelId: 1, Quota: 260,
		Status: TaskStatusSuccess, BillingStatus: TaskBillingStatusSettled,
		SubmitTime: now - 15*24*3600, FinishTime: now - 15*24*3600,
		Properties: Properties{OriginModelName: "Seedance2.0-fast-c"},
	}
	require.NoError(t, DB.Create(task).Error)

	_, err := CreateReferralRewardForTask(999, task.ID, "", "upstream-bill-1", 26000, 18000, 0)
	require.Error(t, err)
	_, err = CreateReferralRewardForTask(999, task.ID, "paid-shop-order-1", "", 26000, 18000, 0)
	require.Error(t, err)
	reward, err := CreateReferralRewardForTask(999, task.ID, "paid-shop-order-1", "upstream-bill-1", 26000, 18000, 0)
	require.NoError(t, err)
	require.EqualValues(t, 2080, reward.RewardUnits)
	require.Equal(t, ReferralRewardPending, reward.Status)
	items, total, err := ListReferralRewards(711, "", 10, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	_, err = CreateReferralRewardForTask(999, task.ID, "paid-shop-order-1", "upstream-bill-1", 26000, 18000, 0)
	require.Error(t, err) // one commission per task
	youngTask := &Task{
		TaskID: "reward-task-2", UserId: 712, ChannelId: 1, Quota: 260,
		Status: TaskStatusSuccess, BillingStatus: TaskBillingStatusSettled,
		SubmitTime: now - 24*3600, FinishTime: now - 24*3600,
		Properties: Properties{OriginModelName: "Seedance2.0-fast-c"},
	}
	require.NoError(t, DB.Create(youngTask).Error)
	youngReward, err := CreateReferralRewardForTask(999, youngTask.ID, "paid-shop-order-2", "upstream-bill-2", 26000, 18000, 0)
	require.NoError(t, err)
	require.Error(t, MarkReferralRewardPaid(youngReward.ID, "too-early"))
	require.NoError(t, MarkReferralRewardPaid(reward.ID, "external-payout-1"))
	youngTask.Status = TaskStatusFailure
	youngTask.BillingStatus = TaskBillingStatusFinalizePending
	youngTask.BillingTargetQuota = 0
	require.NoError(t, DB.Model(youngTask).Updates(map[string]any{
		"status":               youngTask.Status,
		"billing_status":       youngTask.BillingStatus,
		"billing_target_quota": 0,
	}).Error)
	result, err := ApplyTaskBilling(youngTask, TaskBillingStatusFinalizePending, TaskBillingStatusRefunded, "", &TaskBillingEventPayload{ModelName: "Seedance2.0-fast-c"})
	require.NoError(t, err)
	require.True(t, result.Applied)
	var youngUpdated ReferralReward
	require.NoError(t, DB.First(&youngUpdated, youngReward.ID).Error)
	require.Equal(t, ReferralRewardReversed, youngUpdated.Status)
	require.NoError(t, ReverseReferralRewardForTask(task.ID))
	var updated ReferralReward
	require.NoError(t, DB.First(&updated, reward.ID).Error)
	require.Equal(t, ReferralRewardReversed, updated.Status)
	require.Equal(t, "external-payout-1", updated.PayoutReference)

	// Reapproval starts a new agent period; it must not revive earlier customers.
	require.NoError(t, DB.Model(&AgentApplication{}).Where("user_id = ?", 711).
		Update("reviewed_at", now-7*24*3600).Error)
	newTask := &Task{
		TaskID: "reward-task-after-reapproval", UserId: 712, ChannelId: 1, Quota: 260,
		Status: TaskStatusSuccess, BillingStatus: TaskBillingStatusSettled,
		SubmitTime: now - 24*3600, FinishTime: now - 24*3600,
		Properties: Properties{OriginModelName: "Seedance2.0-fast-c"},
	}
	require.NoError(t, DB.Create(newTask).Error)
	_, err = CreateReferralRewardForTask(999, newTask.ID, "paid-shop-order-3", "upstream-bill-3", 26000, 18000, 0)
	require.ErrorContains(t, err, "outside the agent reward period")
}
