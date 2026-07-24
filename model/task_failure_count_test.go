package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountFailedTasks(t *testing.T) {
	truncateTables(t)

	users := []User{
		{Id: 1, Username: "alice", Password: "password", AffCode: "alice-code"},
		{Id: 2, Username: "bob", Password: "password", AffCode: "bob-code"},
	}
	require.NoError(t, DB.Create(&users).Error)

	tasks := []Task{
		{TaskID: "alice-start", UserId: 1, SubmitTime: 100, Status: TaskStatusFailure},
		{TaskID: "alice-success", UserId: 1, SubmitTime: 150, Status: TaskStatusSuccess},
		{TaskID: "alice-end", UserId: 1, SubmitTime: 200, Status: TaskStatusFailure},
		{TaskID: "bob-failure", UserId: 2, SubmitTime: 150, Status: TaskStatusFailure},
		{TaskID: "alice-outside", UserId: 1, SubmitTime: 300, Status: TaskStatusFailure},
	}
	require.NoError(t, DB.Create(&tasks).Error)

	t.Run("all users within inclusive submit time range", func(t *testing.T) {
		count, err := CountAllFailedTasks(100, 200, "")
		require.NoError(t, err)
		require.EqualValues(t, 3, count)
	})

	t.Run("exact username", func(t *testing.T) {
		count, err := CountAllFailedTasks(100, 200, "alice")
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
	})

	t.Run("unknown username", func(t *testing.T) {
		count, err := CountAllFailedTasks(100, 200, "nobody")
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("current user only", func(t *testing.T) {
		count, err := CountUserFailedTasks(1, 100, 200)
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
	})
}
