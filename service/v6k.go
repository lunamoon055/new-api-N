package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func NormalizeV6KBaseURL(value string) string {
	base := strings.TrimRight(strings.TrimSpace(value), "/")
	if strings.HasSuffix(strings.ToLower(base), "/v1") {
		base = base[:len(base)-3]
	}
	if base == "" {
		base = "https://www.v6k.top"
	}
	return base
}

// FetchV6KVideoModels intersects the key-authorized list with the site's public
// video catalog. Catalog requests never forward the provider credential.
func FetchV6KVideoModels(ctx context.Context, baseURL, key, proxy string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	baseURL = NormalizeV6KBaseURL(baseURL)
	client, err := GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	read := func(path string, authenticated bool, result any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			return err
		}
		if authenticated {
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("V6K model catalog returned HTTP %d", resp.StatusCode)
		}
		body, err := common.ReadResponseBodyWithLimit(resp.Body, common.MaxUpstreamResponseBodyBytes)
		if err != nil {
			return err
		}
		return common.Unmarshal(body, result)
	}
	var authorized struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := read("/v1/models", true, &authorized); err != nil {
		return nil, err
	}
	var catalog struct {
		Success bool `json:"success"`
		Data    []struct {
			ModelName string   `json:"model_name"`
			Endpoints []string `json:"supported_endpoint_types"`
		} `json:"data"`
	}
	if err := read("/api/pricing", false, &catalog); err != nil {
		return nil, err
	}
	if !catalog.Success {
		return nil, fmt.Errorf("V6K public video catalog is unavailable")
	}
	videos := make(map[string]bool)
	for _, item := range catalog.Data {
		if common.StringsContains(item.Endpoints, "openai-video") {
			videos[item.ModelName] = true
		}
	}
	ids := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range authorized.Data {
		if videos[item.ID] && !seen[item.ID] {
			ids = append(ids, item.ID)
			seen[item.ID] = true
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func SetV6KNextPoll(task *model.Task, now int64) {
	task.PrivateData.NextPollAt = now + 20
}

func scheduleV6KPollRetry(task *model.Task, retryAfter string) {
	snapshot := task.Snapshot()
	delay := int64(40)
	if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil && seconds > delay {
		delay = seconds
	} else if retryAt, err := http.ParseTime(retryAfter); err == nil {
		delay = max(delay, int64(time.Until(retryAt).Seconds()))
	}
	delay = min(delay, 86400)
	task.PrivateData.NextPollAt = time.Now().Unix() + delay
	if task.ID > 0 {
		if _, err := task.UpdateWithSnapshot(snapshot); err != nil {
			common.SysLog("persist V6K polling backoff failed: " + err.Error())
		}
	}
}
