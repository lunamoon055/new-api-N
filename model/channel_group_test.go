package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestChannelChineseDefaultGroupMapsToDefault(t *testing.T) {
	truncateTables(t)
	originalMemoryCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCache })

	channel := &Channel{
		Type:   constant.ChannelTypeSora,
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
		Name:   "video2",
		Group:  "默认",
		Models: "video-2.0",
	}
	require.NoError(t, channel.Insert())

	var ability Ability
	require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
	require.Equal(t, "default", ability.Group)

	got, err := GetRandomSatisfiedChannel("default", "video-2.0", 0)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, channel.Id, got.Id)
}

func TestChannelGroupFilterMatchesLegacyChineseDefaultGroup(t *testing.T) {
	truncateTables(t)

	channel := &Channel{
		Type:   constant.ChannelTypeSora,
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
		Name:   "legacy-video2",
		Group:  "默认",
		Models: "video-2.0",
	}
	require.NoError(t, DB.Create(channel).Error)

	var count int64
	require.NoError(t, ApplyChannelGroupFilter(DB.Model(&Channel{}), "default").Count(&count).Error)
	require.Equal(t, int64(1), count)
}
