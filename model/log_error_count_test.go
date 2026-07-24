package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCountErrorLogs(t *testing.T) {
	originalLogDB := LOG_DB
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/log-error-count.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Log{}))
	LOG_DB = db
	t.Cleanup(func() {
		LOG_DB = originalLogDB
	})

	logs := []Log{
		{UserId: 1, Username: "alice", CreatedAt: 100, Type: LogTypeError},
		{UserId: 1, Username: "alice", CreatedAt: 150, Type: LogTypeConsume},
		{UserId: 1, Username: "alice", CreatedAt: 200, Type: LogTypeError},
		{UserId: 2, Username: "bob", CreatedAt: 150, Type: LogTypeError},
		{UserId: 1, Username: "alice", CreatedAt: 300, Type: LogTypeError},
	}
	require.NoError(t, db.Create(&logs).Error)

	t.Run("all users within inclusive range", func(t *testing.T) {
		count, err := CountAllErrorLogs(100, 200, "")
		require.NoError(t, err)
		require.EqualValues(t, 3, count)
	})

	t.Run("exact username", func(t *testing.T) {
		count, err := CountAllErrorLogs(100, 200, "alice")
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
	})

	t.Run("current user only", func(t *testing.T) {
		count, err := CountUserErrorLogs(1, 100, 200)
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
	})
}
