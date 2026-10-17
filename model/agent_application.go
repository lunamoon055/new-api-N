package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	AgentStatusPending  = "pending"
	AgentStatusApproved = "approved"
	AgentStatusRejected = "rejected"
	AgentStatusRevoked  = "revoked"
)

type AgentApplication struct {
	ID         int64  `json:"id" gorm:"primaryKey"`
	UserID     int    `json:"user_id" gorm:"uniqueIndex;not null"`
	Status     string `json:"status" gorm:"type:varchar(16);index;not null"`
	Note       string `json:"note" gorm:"type:varchar(500)"`
	ReviewNote string `json:"review_note" gorm:"type:varchar(500)"`
	ReviewerID int    `json:"reviewer_id"`
	AppliedAt  int64  `json:"applied_at"`
	ReviewedAt int64  `json:"reviewed_at"`
	UpdatedAt  int64  `json:"updated_at"`
	Username   string `json:"username" gorm:"-"`
}

func GetAgentApplication(userID int) (*AgentApplication, error) {
	var application AgentApplication
	err := DB.Where("user_id = ?", userID).First(&application).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &application, err
}

func IsApprovedAgent(userID int) bool {
	return approvedAgentAt(DB, userID) > 0
}

func approvedAgentAt(tx *gorm.DB, userID int) int64 {
	if userID <= 0 {
		return 0
	}
	var application AgentApplication
	if err := tx.Select("reviewed_at").Where("user_id = ? AND status = ?", userID, AgentStatusApproved).
		First(&application).Error; err != nil {
		return 0
	}
	return application.ReviewedAt
}

func SubmitAgentApplication(userID int, note string) error {
	note = strings.TrimSpace(note)
	if userID <= 0 || note == "" {
		return errors.New("application details required")
	}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing AgentApplication
		err := withRowLock(tx).Where("user_id = ?", userID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&AgentApplication{UserID: userID, Status: AgentStatusPending, Note: note, AppliedAt: now, UpdatedAt: now}).Error
		}
		if err != nil {
			return err
		}
		if existing.Status == AgentStatusPending || existing.Status == AgentStatusApproved {
			return errors.New("application already pending or approved")
		}
		return tx.Model(&existing).Updates(map[string]any{
			"status": AgentStatusPending, "note": note, "review_note": "",
			"reviewer_id": 0, "applied_at": now, "reviewed_at": 0, "updated_at": now,
		}).Error
	})
}

func ReviewAgentApplication(id int64, reviewerID int, decision, note string) error {
	if decision != AgentStatusApproved && decision != AgentStatusRejected && decision != AgentStatusRevoked {
		return errors.New("invalid review decision")
	}
	if (decision == AgentStatusRejected || decision == AgentStatusRevoked) && note == "" {
		return errors.New("review note required for rejection or revocation")
	}
	if decision == AgentStatusApproved && common.QuotaForInviter > 0 {
		return errors.New("disable fixed registration invitation rewards before approving agents")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var application AgentApplication
		if err := withRowLock(tx).First(&application, id).Error; err != nil {
			return err
		}
		if decision == AgentStatusRevoked {
			if application.Status != AgentStatusApproved {
				return errors.New("only approved agents can be revoked")
			}
		} else if application.Status != AgentStatusPending {
			return errors.New("only pending applications can be reviewed")
		}
		now := time.Now().Unix()
		result := tx.Model(&application).Where("status = ?", application.Status).Updates(map[string]any{
			"status": decision, "review_note": note, "reviewer_id": reviewerID,
			"reviewed_at": now, "updated_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("application changed during review")
		}
		return nil
	})
}

func ListAgentApplications(status string, limit, offset int) ([]AgentApplication, int64, error) {
	query := DB.Model(&AgentApplication{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var applications []AgentApplication
	if err := query.Order("applied_at desc, id desc").Limit(limit).Offset(offset).Find(&applications).Error; err != nil {
		return nil, 0, err
	}
	for i := range applications {
		applications[i].Username, _ = GetUsernameById(applications[i].UserID, false)
	}
	return applications, total, nil
}
