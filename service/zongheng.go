package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const zonghengCapabilitiesTTL = time.Minute

type zonghengCapabilitiesEntry struct {
	models    []ZonghengVideoModel
	expiresAt time.Time
}

var zonghengCapabilitiesCache = struct {
	sync.Mutex
	entries map[[32]byte]zonghengCapabilitiesEntry
}{entries: make(map[[32]byte]zonghengCapabilitiesEntry)}

func zonghengCapabilitiesKey(baseURL, key, proxy string) [32]byte {
	return sha256.Sum256([]byte(NormalizeZonghengBaseURL(baseURL) + "\n" + key + "\n" + proxy))
}

// GetZonghengVideoModels reuses only a successful, recent capability response
// from this exact origin, credential and proxy. It never uses the UI snapshot
// or expired capabilities to authorize a submission after a network failure.
func GetZonghengVideoModels(ctx context.Context, baseURL, key, proxy string) ([]ZonghengVideoModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cacheKey := zonghengCapabilitiesKey(baseURL, key, proxy)
	zonghengCapabilitiesCache.Lock()
	entry, ok := zonghengCapabilitiesCache.entries[cacheKey]
	zonghengCapabilitiesCache.Unlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.models, nil
	}
	return FetchZonghengVideoModels(ctx, baseURL, key, proxy)
}

func storeZonghengCapabilities(cacheKey [32]byte, models []ZonghengVideoModel) {
	zonghengCapabilitiesCache.Lock()
	defer zonghengCapabilitiesCache.Unlock()
	now := time.Now()
	for key, entry := range zonghengCapabilitiesCache.entries {
		if !now.Before(entry.expiresAt) {
			delete(zonghengCapabilitiesCache.entries, key)
		}
	}
	if len(zonghengCapabilitiesCache.entries) >= 256 {
		var oldest [32]byte
		var expiry time.Time
		for key, entry := range zonghengCapabilitiesCache.entries {
			if expiry.IsZero() || entry.expiresAt.Before(expiry) {
				oldest, expiry = key, entry.expiresAt
			}
		}
		delete(zonghengCapabilitiesCache.entries, oldest)
	}
	zonghengCapabilitiesCache.entries[cacheKey] = zonghengCapabilitiesEntry{models: models, expiresAt: now.Add(zonghengCapabilitiesTTL)}
}

type zonghengModelsHTTPError struct {
	status     int
	retryAfter time.Duration
}

func (err *zonghengModelsHTTPError) Error() string {
	return fmt.Sprintf("Zongheng models returned HTTP %d", err.status)
}

func retryableZonghengModelsError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var statusErr *zonghengModelsHTTPError
	if errors.As(err, &statusErr) {
		switch statusErr.status {
		case 408, 425, 429, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary())
}

// Zongheng publishes capabilities per API key, rather than a fixed model list.
type ZonghengVideoModel struct {
	ID           string `json:"id"`
	Capabilities struct {
		Type        string   `json:"capability_type"`
		Durations   []int    `json:"duration_options"`
		Resolutions []string `json:"resolution_options"`
		Ratios      []string `json:"aspect_ratio_options"`
		Qualities   []string `json:"quality_options"`
		MaxImages   *int     `json:"max_image_refs"`
		MaxVideos   *int     `json:"max_video_refs"`
		MaxAudios   *int     `json:"max_audio_refs"`
		Defaults    struct {
			Duration   *int   `json:"duration"`
			Resolution string `json:"resolution"`
			Ratio      string `json:"aspect_ratio"`
		} `json:"defaults"`
	} `json:"capabilities"`
}

func SetZonghengNextPoll(task *model.Task, now int64) {
	var response struct {
		NextPollSeconds int64 `json:"next_poll_seconds"`
	}
	if common.Unmarshal(task.Data, &response) != nil {
		return
	}
	delay := response.NextPollSeconds
	if delay <= 0 {
		delay = 10
	}
	// Avoid arithmetic overflow from an invalid upstream response.
	if delay > 86400 {
		delay = 86400
	}
	task.PrivateData.NextPollAt = now + delay
}

func NormalizeZonghengBaseURL(value string) string {
	base := strings.TrimRight(strings.TrimSpace(value), "/")
	if strings.HasSuffix(base, "/v1") {
		base = strings.TrimSuffix(base, "/v1")
	}
	return base
}

func FetchZonghengVideoModels(ctx context.Context, baseURL, key, proxy string) ([]ZonghengVideoModel, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client, err := GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	var models []ZonghengVideoModel
	for attempt := 0; attempt < 3; attempt++ {
		models, err = fetchZonghengVideoModelsOnce(ctx, client, baseURL, key)
		if err == nil {
			storeZonghengCapabilities(zonghengCapabilitiesKey(baseURL, key, proxy), models)
			return models, nil
		}
		if !retryableZonghengModelsError(err) || attempt == 2 {
			break
		}
		delay := time.Duration(attempt+1) * 150 * time.Millisecond
		var statusErr *zonghengModelsHTTPError
		if errors.As(err, &statusErr) && statusErr.retryAfter > delay {
			delay = statusErr.retryAfter
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= delay {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, err
}

func fetchZonghengVideoModelsOnce(ctx context.Context, client *http.Client, baseURL, key string) ([]ZonghengVideoModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, NormalizeZonghengBaseURL(baseURL)+"/v1/models?include_capabilities=1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		statusErr := &zonghengModelsHTTPError{status: resp.StatusCode}
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
			if seconds > 86400 {
				seconds = 86400
			}
			statusErr.retryAfter = time.Duration(seconds) * time.Second
		} else if retryAt, err := http.ParseTime(resp.Header.Get("Retry-After")); err == nil {
			statusErr.retryAfter = time.Until(retryAt)
		}
		return nil, statusErr
	}
	body, err := common.ReadResponseBodyWithLimit(resp.Body, common.MaxUpstreamResponseBodyBytes)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []ZonghengVideoModel `json:"data"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	models := make([]ZonghengVideoModel, 0, len(result.Data))
	for _, item := range result.Data {
		if item.ID != "" && item.Capabilities.Type == "video" {
			models = append(models, item)
		}
	}
	return models, nil
}
