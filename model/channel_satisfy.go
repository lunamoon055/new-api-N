package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func IsChannelEnabledForGroupModel(group string, modelName string, channelID int) bool {
	group = NormalizeChannelGroupName(group)
	modelName = strings.TrimSpace(modelName)
	if group == "" || modelName == "" || channelID <= 0 {
		return false
	}
	if !common.MemoryCacheEnabled {
		return isChannelEnabledForGroupModelDB(group, modelName, channelID)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	if group2model2channels == nil {
		return false
	}

	for _, lookupGroup := range channelGroupLookupNames(group) {
		if isChannelIDInList(group2model2channels[lookupGroup][modelName], channelID) {
			return true
		}
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized != "" && normalized != modelName {
		for _, lookupGroup := range channelGroupLookupNames(group) {
			if isChannelIDInList(group2model2channels[lookupGroup][normalized], channelID) {
				return true
			}
		}
	}
	return false
}

func IsChannelEnabledForAnyGroupModel(groups []string, modelName string, channelID int) bool {
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		if IsChannelEnabledForGroupModel(g, modelName, channelID) {
			return true
		}
	}
	return false
}

func isChannelEnabledForGroupModelDB(group string, modelName string, channelID int) bool {
	group = NormalizeChannelGroupName(group)
	modelName = strings.TrimSpace(modelName)
	var count int64
	err := DB.Model(&Ability{}).
		Where(commonGroupCol+" IN ? and model = ? and channel_id = ? and enabled = ?", channelGroupLookupNames(group), modelName, channelID, true).
		Count(&count).Error
	if err == nil && count > 0 {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized == "" || normalized == modelName {
		return false
	}
	count = 0
	err = DB.Model(&Ability{}).
		Where(commonGroupCol+" IN ? and model = ? and channel_id = ? and enabled = ?", channelGroupLookupNames(group), normalized, channelID, true).
		Count(&count).Error
	return err == nil && count > 0
}

func isChannelIDInList(list []int, channelID int) bool {
	for _, id := range list {
		if id == channelID {
			return true
		}
	}
	return false
}
