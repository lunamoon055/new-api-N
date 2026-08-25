package model

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCreationIdempotencyAcrossConfiguredDialect(t *testing.T) {
	dialect := os.Getenv("CREATION_IDEMPOTENCY_TEST_DIALECT")
	if dialect == "" {
		t.Skip("external dialect test is enabled by CI")
	}

	var dialector gorm.Dialector
	switch dialect {
	case "sqlite":
		dialector = sqlite.Open(t.TempDir() + "/creation-idempotency.db")
	case "mysql":
		dialector = mysql.Open(os.Getenv("CREATION_IDEMPOTENCY_MYSQL_DSN"))
	case "postgres":
		dialector = postgres.Open(os.Getenv("CREATION_IDEMPOTENCY_POSTGRES_DSN"))
	default:
		t.Fatalf("unsupported dialect %q", dialect)
	}

	database, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&CreationIdempotency{}))
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })

	previous := DB
	DB = database
	t.Cleanup(func() { DB = previous })

	key := fmt.Sprintf("dialect:%s:%d", dialect, time.Now().UnixNano())
	const workers = 12
	type outcome struct {
		created bool
		err     error
	}
	results := make(chan outcome, workers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, created, reserveErr := ReserveCreationIdempotency(501, 601, key, "same-hash")
			results <- outcome{created: created, err: reserveErr}
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	winners := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.created {
			winners++
		}
	}
	require.Equal(t, 1, winners)
	entry, err := GetCreationIdempotency(501, key)
	require.NoError(t, err)
	require.NoError(t, CompleteCreationIdempotency(entry.ID, "same-hash", 202, []byte(`{"Content-Type":"application/json"}`), []byte(`{"task_id":"task_dialect"}`), false))
	replayed, err := GetCreationIdempotency(501, key)
	require.NoError(t, err)
	require.Equal(t, CreationIdempotencyCompleted, replayed.Status)
	require.Equal(t, 202, replayed.ResponseStatus)
}
