package openai

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeUsageAccumulatorPrefersReportedUsage(t *testing.T) {
	accumulator := &realtimeUsageAccumulator{}
	accumulator.addInput(10, 2)
	accumulator.addOutput(5, 3)

	reported := &dto.RealtimeUsage{
		TotalTokens:  150,
		InputTokens:  100,
		OutputTokens: 50,
		InputTokenDetails: dto.InputTokenDetails{
			CachedTokens: 20,
			TextTokens:   70,
			AudioTokens:  10,
		},
		OutputTokenDetails: dto.OutputTokenDetails{
			TextTokens:  40,
			AudioTokens: 10,
		},
	}

	first := accumulator.complete(reported, 0, 0)
	require.Equal(t, reported, first)

	// The next response has no upstream usage, so only its local estimates and
	// response.done fallback tokens are added to the cumulative total.
	accumulator.addInput(7, 0)
	accumulator.addOutput(11, 0)
	second := accumulator.complete(nil, 2, 1)

	assert.Equal(t, 171, second.TotalTokens)
	assert.Equal(t, 110, second.InputTokens)
	assert.Equal(t, 61, second.OutputTokens)
	assert.Equal(t, 79, second.InputTokenDetails.TextTokens)
	assert.Equal(t, 11, second.InputTokenDetails.AudioTokens)
	assert.Equal(t, 51, second.OutputTokenDetails.TextTokens)
	assert.Equal(t, 10, second.OutputTokenDetails.AudioTokens)
	assert.Equal(t, second, accumulator.finalize())
}

func TestRealtimeUsageAccumulatorConcurrentUpdates(t *testing.T) {
	accumulator := &realtimeUsageAccumulator{}
	const workers = 100

	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			accumulator.addInput(1, 1)
		}()
		go func() {
			defer wg.Done()
			accumulator.addOutput(2, 1)
		}()
	}
	wg.Wait()

	usage := accumulator.finalize()
	assert.Equal(t, workers*5, usage.TotalTokens)
	assert.Equal(t, workers*2, usage.InputTokens)
	assert.Equal(t, workers*3, usage.OutputTokens)
	assert.Equal(t, workers, usage.InputTokenDetails.TextTokens)
	assert.Equal(t, workers, usage.InputTokenDetails.AudioTokens)
	assert.Equal(t, workers*2, usage.OutputTokenDetails.TextTokens)
	assert.Equal(t, workers, usage.OutputTokenDetails.AudioTokens)
}
