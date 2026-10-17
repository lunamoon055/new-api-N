package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ReferralRewardPending  = "pending"
	ReferralRewardPaid     = "paid"
	ReferralRewardReversed = "reversed"
)

// ReferralReward is a verified, direct-customer transaction. Amounts are in
// RMB units (1 unit = ¥0.0001). SourceOrderID is the administrator-verified paid shop order; the
// original wallet quota alone is never evidence of a paid purchase.
type ReferralReward struct {
	ID              int64  `json:"id" gorm:"primaryKey"`
	TaskRecordID    int64  `json:"task_record_id" gorm:"uniqueIndex;not null"`
	CustomerID      int    `json:"customer_id" gorm:"index;not null"`
	ReferrerID      int    `json:"referrer_id" gorm:"index;not null"`
	ReferralKind    string `json:"referral_kind" gorm:"type:varchar(16);not null"`
	SourceOrderID   string `json:"source_order_id" gorm:"type:varchar(128);index;not null"`
	CostEvidenceID  string `json:"cost_evidence_id" gorm:"type:varchar(128);not null"`
	ChannelID       int    `json:"channel_id"`
	ModelName       string `json:"model_name" gorm:"type:varchar(191)"`
	SaleUnits       int64  `json:"sale_units"`
	FeeUnits        int64  `json:"fee_units"`
	CostUnits       int64  `json:"cost_units"`
	ExtraCostUnits  int64  `json:"extra_cost_units"`
	MarginUnits     int64  `json:"margin_units"`
	RewardUnits     int64  `json:"reward_units"`
	PlatformUnits   int64  `json:"platform_units"`
	Status          string `json:"status" gorm:"type:varchar(16);index;not null"`
	EligibleAt      int64  `json:"eligible_at" gorm:"index"`
	CreatedAt       int64  `json:"created_at"`
	ReviewedBy      int    `json:"reviewed_by"`
	PaidAt          int64  `json:"paid_at"`
	PayoutReference string `json:"payout_reference" gorm:"type:varchar(128)"`
	ReversedAt      int64  `json:"reversed_at"`
}

func referralRewardAmount(sale, cost, extra int64, agent bool) (fee, margin, reward, platform int64) {
	if sale <= 0 || cost < 0 || extra < 0 {
		return 0, 0, 0, 0
	}
	fee = (sale*3 + 99) / 100
	margin = sale - fee - cost - extra
	if margin <= 0 {
		return fee, margin, 0, margin
	}
	share, cap := int64(20), int64(5)
	if agent {
		share, cap = 30, 8
	}
	reward = margin * share / 100
	if sale*cap/100 < reward {
		reward = sale * cap / 100
	}
	protection := margin - (sale*15+99)/100
	if protection < reward {
		reward = protection
	}
	if reward < 0 {
		reward = 0
	}
	platform = margin - reward
	return
}

func CreateReferralRewardForTask(adminID int, taskID int64, sourceOrderID, costEvidenceID string, saleUnits, costUnits, extraCostUnits int64) (*ReferralReward, error) {
	if common.QuotaForInviter > 0 {
		return nil, errors.New("disable fixed registration invitation rewards before using margin-based rewards")
	}
	sourceOrderID = strings.TrimSpace(sourceOrderID)
	costEvidenceID = strings.TrimSpace(costEvidenceID)
	if adminID <= 0 || taskID <= 0 || sourceOrderID == "" || len(sourceOrderID) > 128 || costEvidenceID == "" || len(costEvidenceID) > 128 || saleUnits <= 0 || costUnits < 0 || extraCostUnits < 0 || saleUnits > 1000000000 || costUnits > 1000000000 || extraCostUnits > 1000000000 {
		return nil, errors.New("invalid verified cost or order data")
	}
	var reward ReferralReward
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := withRowLock(tx).First(&task, taskID).Error; err != nil {
			return err
		}
		if task.Status != TaskStatusSuccess || task.BillingStatus != TaskBillingStatusSettled || task.Quota <= 0 || task.FinishTime <= 0 {
			return errors.New("task is not successfully settled")
		}
		if task.PrivateData.BillingSource == "subscription" {
			return errors.New("subscription-funded tasks are ineligible")
		}
		var customer User
		if err := withRowLock(tx).Select("id", "inviter_id", "referral_kind", "referral_approved_at", "first_paid_at").First(&customer, task.UserId).Error; err != nil {
			return err
		}
		if customer.InviterId <= 0 || customer.InviterId == customer.Id {
			return errors.New("task has no direct referrer")
		}
		var referrer User
		if err := tx.Select("id", "status").First(&referrer, customer.InviterId).Error; err != nil || referrer.Status != common.UserStatusEnabled {
			return errors.New("referrer account is unavailable")
		}
		firstPaidAt := customer.FirstPaidAt
		if firstPaidAt == 0 || task.FinishTime < firstPaidAt {
			firstPaidAt = task.FinishTime
			if err := tx.Model(&User{}).Where("id = ?", customer.Id).Update("first_paid_at", firstPaidAt).Error; err != nil {
				return err
			}
		}
		isAgent := customer.ReferralKind == "agent"
		kind := "promotion"
		if isAgent {
			var application AgentApplication
			if err := tx.Where("user_id = ? AND status = ?", customer.InviterId, AgentStatusApproved).First(&application).Error; err != nil {
				return errors.New("referrer is not an approved agent")
			}
			if customer.ReferralApprovedAt <= 0 || customer.ReferralApprovedAt != application.ReviewedAt || task.SubmitTime < customer.ReferralApprovedAt || task.FinishTime > time.Unix(customer.ReferralApprovedAt, 0).AddDate(1, 0, 0).Unix() {
				return errors.New("task is outside the agent reward period")
			}
			kind = "agent"
		} else if task.FinishTime > firstPaidAt+180*24*3600 {
			return errors.New("task is outside the promotion reward period")
		}
		fee, margin, amount, platform := referralRewardAmount(saleUnits, costUnits, extraCostUnits, isAgent)
		if amount <= 0 {
			return errors.New("verified margin is insufficient for a protected reward")
		}
		reward = ReferralReward{
			TaskRecordID: task.ID, CustomerID: customer.Id, ReferrerID: customer.InviterId,
			ReferralKind: kind, SourceOrderID: sourceOrderID, ChannelID: task.ChannelId,
			CostEvidenceID: costEvidenceID,
			ModelName:      task.Properties.ModelName(), SaleUnits: saleUnits,
			FeeUnits: fee, CostUnits: costUnits, ExtraCostUnits: extraCostUnits,
			MarginUnits: margin, RewardUnits: amount, PlatformUnits: platform,
			Status: ReferralRewardPending, EligibleAt: task.FinishTime + 14*24*3600,
			CreatedAt: time.Now().Unix(), ReviewedBy: adminID,
		}
		return tx.Create(&reward).Error
	})
	if err != nil {
		return nil, err
	}
	return &reward, nil
}

func ListReferralRewards(referrerID int, status string, limit, offset int) ([]ReferralReward, int64, error) {
	query := DB.Model(&ReferralReward{})
	if referrerID > 0 {
		query = query.Where("referrer_id = ?", referrerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rewards []ReferralReward
	err := query.Order("id desc").Limit(limit).Offset(offset).Find(&rewards).Error
	return rewards, total, err
}

// MarkReferralRewardPaid records an externally completed payout. It does not
// credit wallet quota, because RMB-to-wallet conversion varies by deployment.
func MarkReferralRewardPaid(id int64, payoutReference string) error {
	payoutReference = strings.TrimSpace(payoutReference)
	if id <= 0 || payoutReference == "" || len(payoutReference) > 128 {
		return errors.New("payout reference required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reward ReferralReward
		if err := withRowLock(tx).First(&reward, id).Error; err != nil {
			return err
		}
		if reward.Status != ReferralRewardPending || reward.EligibleAt > time.Now().Unix() {
			return errors.New("reward is not payable")
		}
		var task Task
		if err := tx.First(&task, reward.TaskRecordID).Error; err != nil {
			return err
		}
		if task.Status != TaskStatusSuccess || task.BillingStatus != TaskBillingStatusSettled {
			return errors.New("task is no longer successfully settled")
		}
		result := tx.Model(&reward).Where("status = ?", ReferralRewardPending).Updates(map[string]any{
			"status": ReferralRewardPaid, "paid_at": time.Now().Unix(), "payout_reference": payoutReference,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("reward changed during payout")
		}
		return nil
	})
}

func ReverseReferralRewardForTask(taskID int64) error {
	if taskID <= 0 {
		return nil
	}
	return DB.Model(&ReferralReward{}).
		Where("task_record_id = ? AND status != ?", taskID, ReferralRewardReversed).
		Updates(map[string]any{"status": ReferralRewardReversed, "reversed_at": time.Now().Unix()}).Error
}
