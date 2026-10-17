package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultGroupRemainsAvailableWithoutConfiguredRatio(t *testing.T) {
	previous := GroupRatio2JSONString()
	t.Cleanup(func() { require.NoError(t, UpdateGroupRatioByJSONString(previous)) })
	require.NoError(t, UpdateGroupRatioByJSONString(`{"vip":0.8,"SVIP":0.6,"默认":1.2}`))
	require.Equal(t, float64(1), GetGroupRatioCopy()["default"])
	require.Equal(t, float64(1), GetGroupRatio("default"))
	require.True(t, ContainsGroupRatio("default"))
	require.False(t, ContainsGroupRatio("removed"))
	require.Equal(t, 0.8, GetGroupRatio("vip"))
	require.Equal(t, 1.2, GetGroupRatio("默认"))
	copy := GetGroupRatioCopy()
	delete(copy, "default")
	require.Contains(t, GetGroupRatioCopy(), "default")
	require.NoError(t, UpdateGroupRatioByJSONString(`{"default":1.5,"vip":0.8}`))
	require.Equal(t, 1.5, GetGroupRatioCopy()["default"])
}
