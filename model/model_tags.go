package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxModelTagsRunes = 255

// GetModelTags returns the existing tags across all non-deleted metadata records.
func GetModelTags() ([]string, error) {
	var values []string
	if err := DB.Model(&Model{}).Pluck("tags", &values).Error; err != nil {
		return nil, err
	}
	tags := normalizeLookupValues(strings.Split(strings.Join(values, ","), ","))
	sort.Strings(tags)
	return tags, nil
}

// BatchAddModelTags appends tags atomically without rewriting model metadata.
// Read the current tags under a row lock so concurrent additions are preserved.
func BatchAddModelTags(ids []int, tags []string) (int, error) {
	uniqueIDs, err := normalizeModelTagIDs(ids)
	if err != nil {
		return 0, err
	}
	tags = normalizeLookupValues(tags)
	if len(tags) == 0 {
		return 0, errors.New("请至少选择或输入一个标签")
	}
	for _, tag := range tags {
		if strings.ContainsAny(tag, ",\r\n") {
			return 0, errors.New("单个标签不能包含逗号或换行")
		}
	}
	if utf8.RuneCountInString(strings.Join(tags, ",")) > maxModelTagsRunes {
		return 0, errors.New("标签总长度不能超过 255 个字符")
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		items, err := lockModelsForTagUpdate(tx, uniqueIDs)
		if err != nil {
			return err
		}
		for i := range items {
			merged := normalizeLookupValues(append(strings.Split(items[i].Tags, ","), tags...))
			items[i].Tags = strings.Join(merged, ",")
			if utf8.RuneCountInString(items[i].Tags) > maxModelTagsRunes {
				return fmt.Errorf("模型 %s 的标签总长度超过 255 个字符，请减少标签后重试", items[i].ModelName)
			}
		}
		now := common.GetTimestamp()
		for _, item := range items {
			if err := tx.Model(&Model{}).Where("id = ?", item.Id).Updates(map[string]any{
				"tags":         item.Tags,
				"updated_time": now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(uniqueIDs), nil
}

// BatchClearModelTags clears all tags on the selected records in one transaction.
func BatchClearModelTags(ids []int) (int, error) {
	uniqueIDs, err := normalizeModelTagIDs(ids)
	if err != nil {
		return 0, err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockModelsForTagUpdate(tx, uniqueIDs); err != nil {
			return err
		}
		return tx.Model(&Model{}).Where("id IN ?", uniqueIDs).Updates(map[string]any{
			"tags":         "",
			"updated_time": common.GetTimestamp(),
		}).Error
	})
	if err != nil {
		return 0, err
	}
	return len(uniqueIDs), nil
}

func normalizeModelTagIDs(ids []int) ([]int, error) {
	if len(ids) == 0 || len(ids) > 1000 {
		return nil, errors.New("请选择 1 到 1000 个模型")
	}
	uniqueIDs := make([]int, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("模型 ID 无效")
		}
		if !seen[id] {
			uniqueIDs = append(uniqueIDs, id)
			seen[id] = true
		}
	}
	sort.Ints(uniqueIDs)
	return uniqueIDs, nil
}

func lockModelsForTagUpdate(tx *gorm.DB, ids []int) ([]Model, error) {
	var items []Model
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, errors.New("部分所选模型已不存在，请刷新列表后重试")
	}
	return items, nil
}
