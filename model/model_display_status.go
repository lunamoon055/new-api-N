package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

// ModelDisplayStatus is presentation metadata keyed by the exact public model
// name. Keep it separate from Model.Status and channel abilities: neither model
// routing, upstream metadata sync nor billing should change this override.
type ModelDisplayStatus struct {
	ModelName   string `json:"model_name" gorm:"primaryKey;size:128"`
	Available   bool   `json:"available" gorm:"not null"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func GetModelDisplayStatuses() (map[string]bool, error) {
	var rows []ModelDisplayStatus
	if err := DB.Find(&rows).Error; err != nil {
		return nil, err
	}
	statuses := make(map[string]bool, len(rows))
	for _, row := range rows {
		statuses[row.ModelName] = row.Available
	}
	return statuses, nil
}

func SaveModelDisplayStatus(modelName string, available bool) (*ModelDisplayStatus, error) {
	item := ModelDisplayStatus{ModelName: modelName, Available: available, UpdatedTime: common.GetTimestamp()}
	// Explicit assignments preserve false on both insert and update. GORM
	// generates the appropriate upsert syntax for SQLite, MySQL and PostgreSQL.
	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "model_name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"available": item.Available, "updated_time": item.UpdatedTime,
		}),
	}).Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
