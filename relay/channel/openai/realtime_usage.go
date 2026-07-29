package openai

import (
	"sync"

	"github.com/QuantumNous/new-api/dto"
)

// realtimeUsageAccumulator owns the mutable usage state shared by the two
// websocket reader goroutines. Reported upstream usage replaces local
// estimates for a completed response; estimates are only used as a fallback.
type realtimeUsageAccumulator struct {
	mu    sync.Mutex
	local dto.RealtimeUsage
	total dto.RealtimeUsage
}

func (a *realtimeUsageAccumulator) addInput(textTokens, audioTokens int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.local.TotalTokens += textTokens + audioTokens
	a.local.InputTokens += textTokens + audioTokens
	a.local.InputTokenDetails.TextTokens += textTokens
	a.local.InputTokenDetails.AudioTokens += audioTokens
}

func (a *realtimeUsageAccumulator) addOutput(textTokens, audioTokens int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.local.TotalTokens += textTokens + audioTokens
	a.local.OutputTokens += textTokens + audioTokens
	a.local.OutputTokenDetails.TextTokens += textTokens
	a.local.OutputTokenDetails.AudioTokens += audioTokens
}

func (a *realtimeUsageAccumulator) complete(reported *dto.RealtimeUsage, fallbackInputText, fallbackInputAudio int) *dto.RealtimeUsage {
	a.mu.Lock()
	defer a.mu.Unlock()

	if reported != nil {
		addRealtimeUsage(&a.total, reported)
	} else {
		a.local.TotalTokens += fallbackInputText + fallbackInputAudio
		a.local.InputTokens += fallbackInputText + fallbackInputAudio
		a.local.InputTokenDetails.TextTokens += fallbackInputText
		a.local.InputTokenDetails.AudioTokens += fallbackInputAudio
		addRealtimeUsage(&a.total, &a.local)
	}
	a.local = dto.RealtimeUsage{}
	return cloneRealtimeUsage(&a.total)
}

func (a *realtimeUsageAccumulator) finalize() *dto.RealtimeUsage {
	a.mu.Lock()
	defer a.mu.Unlock()

	addRealtimeUsage(&a.total, &a.local)
	a.local = dto.RealtimeUsage{}
	return cloneRealtimeUsage(&a.total)
}

func addRealtimeUsage(dst, src *dto.RealtimeUsage) {
	if dst == nil || src == nil {
		return
	}
	dst.TotalTokens += src.TotalTokens
	dst.InputTokens += src.InputTokens
	dst.OutputTokens += src.OutputTokens
	dst.InputTokenDetails.CachedTokens += src.InputTokenDetails.CachedTokens
	dst.InputTokenDetails.CachedCreationTokens += src.InputTokenDetails.CachedCreationTokens
	dst.InputTokenDetails.TextTokens += src.InputTokenDetails.TextTokens
	dst.InputTokenDetails.AudioTokens += src.InputTokenDetails.AudioTokens
	dst.InputTokenDetails.ImageTokens += src.InputTokenDetails.ImageTokens
	dst.OutputTokenDetails.TextTokens += src.OutputTokenDetails.TextTokens
	dst.OutputTokenDetails.AudioTokens += src.OutputTokenDetails.AudioTokens
	dst.OutputTokenDetails.ImageTokens += src.OutputTokenDetails.ImageTokens
	dst.OutputTokenDetails.ReasoningTokens += src.OutputTokenDetails.ReasoningTokens
}

func cloneRealtimeUsage(usage *dto.RealtimeUsage) *dto.RealtimeUsage {
	if usage == nil {
		return &dto.RealtimeUsage{}
	}
	cloned := *usage
	return &cloned
}
