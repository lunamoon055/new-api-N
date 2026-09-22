package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGetRandomSatisfiedChannelWithFilterSeparatesProviderHosts(t *testing.T) {
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true

	channelSyncLock.Lock()
	previousGroups := group2model2channels
	previousChannels := channelsIDM
	linkskyURL := "https://linksky.top/"
	otherURL := "https://other-provider.example"
	priority := int64(10)
	weight := uint(100)
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-image-2": {1, 2},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, BaseURL: &linkskyURL, Priority: &priority, Weight: &weight},
		2: {Id: 2, BaseURL: &otherURL, Priority: &priority, Weight: &weight},
	}
	channelSyncLock.Unlock()

	defer func() {
		channelSyncLock.Lock()
		group2model2channels = previousGroups
		channelsIDM = previousChannels
		channelSyncLock.Unlock()
		common.MemoryCacheEnabled = previousCacheEnabled
	}()

	filter := func(channel *Channel) bool {
		return common.IsBaseURLHost(channel.GetBaseURL(), common.LinkskyProviderHost)
	}
	for range 20 {
		channel, err := GetRandomSatisfiedChannelWithFilter("default", "gpt-image-2", 0, filter)
		require.NoError(t, err)
		require.NotNil(t, channel)
		require.Equal(t, 1, channel.Id)
	}
}
