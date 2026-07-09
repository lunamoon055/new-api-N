package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorReturnsSanbaoAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSanbao)))

	require.NotNil(t, adaptor)
	require.Equal(t, "sanbao", adaptor.GetChannelName())
}
