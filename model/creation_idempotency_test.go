package model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReserveCreationIdempotencyIsAtomic(t *testing.T) {
	truncateTables(t)

	const workers = 8
	var wait sync.WaitGroup
	start := make(chan struct{})
	created := make(chan bool, workers)
	errors := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			entry, won, err := ReserveCreationIdempotency(77, 9, "generation:atomic", "hash-a")
			if err == nil {
				require.Equal(t, "hash-a", entry.RequestHash)
			}
			created <- won
			errors <- err
		}()
	}
	close(start)
	wait.Wait()
	close(created)
	close(errors)

	winners := 0
	for won := range created {
		if won {
			winners++
		}
	}
	for err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, 1, winners)

	var count int64
	require.NoError(t, DB.Model(&CreationIdempotency{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestCompleteCreationIdempotencyPersistsReplay(t *testing.T) {
	truncateTables(t)
	entry, created, err := ReserveCreationIdempotency(5, 3, "generation:replay", "hash-b")
	require.NoError(t, err)
	require.True(t, created)

	require.NoError(t, CompleteCreationIdempotency(entry.ID, "hash-b", 201, []byte(`{"Content-Type":"application/json"}`), []byte(`{"task_id":"task_1"}`), false))
	replayed, err := GetCreationIdempotency(5, "generation:replay")
	require.NoError(t, err)
	require.Equal(t, CreationIdempotencyCompleted, replayed.Status)
	require.Equal(t, 201, replayed.ResponseStatus)
	require.JSONEq(t, `{"task_id":"task_1"}`, string(replayed.ResponseBody))
}
