package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const (
	mediaTransferMaxAttempts     = 8
	mediaTransferTick            = 5 * time.Second
	mediaTransferLeaseTick       = 30 * time.Second
	mediaTransferVerifyLimit     = 20 * time.Second
	mediaTransferDataLimit       = 32 << 10
	mediaTransferCircuitFailures = 3
	mediaTransferCircuitCooldown = 2 * time.Minute
)

var mediaTransferWake = make(chan struct{}, 1)

// WakeMediaTransferWorker asks the master worker to claim due jobs immediately
// instead of waiting for its periodic tick. It is safe for admin retries and
// normal enqueue paths alike.
func WakeMediaTransferWorker() {
	select {
	case mediaTransferWake <- struct{}{}:
	default:
	}
}

type mediaTransferCircuit struct {
	failures  int
	openUntil time.Time
	lastError string
}

type MediaStorageProviderHealth struct {
	ProviderID          string `json:"provider_id"`
	Available           bool   `json:"available"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	OpenUntil           int64  `json:"open_until,omitempty"`
	LastError           string `json:"last_error,omitempty"`
}

var mediaTransferCircuits = struct {
	sync.Mutex
	items map[string]mediaTransferCircuit
}{items: make(map[string]mediaTransferCircuit)}

func mediaProviderAvailable(providerID string, now time.Time) bool {
	mediaTransferCircuits.Lock()
	defer mediaTransferCircuits.Unlock()
	state, ok := mediaTransferCircuits.items[providerID]
	return !ok || state.openUntil.IsZero() || !now.Before(state.openUntil)
}

func recordMediaProviderFailure(providerID string, err error) {
	if providerID == "" {
		return
	}
	mediaTransferCircuits.Lock()
	defer mediaTransferCircuits.Unlock()
	state := mediaTransferCircuits.items[providerID]
	state.failures++
	state.lastError = mediaTransferErrorCategory(err)
	if state.failures >= mediaTransferCircuitFailures {
		state.openUntil = time.Now().Add(mediaTransferCircuitCooldown)
	}
	mediaTransferCircuits.items[providerID] = state
}

func recordMediaProviderSuccess(providerID string) {
	mediaTransferCircuits.Lock()
	delete(mediaTransferCircuits.items, providerID)
	mediaTransferCircuits.Unlock()
}

// GetMediaStorageProviderHealth returns process-local circuit state. The worker
// is single-master by design; after a restart all circuits start closed and are
// re-evaluated from live requests.
func GetMediaStorageProviderHealth() []MediaStorageProviderHealth {
	providers := GetMediaStorageProviders()
	mediaTransferCircuits.Lock()
	defer mediaTransferCircuits.Unlock()
	result := make([]MediaStorageProviderHealth, 0, len(providers))
	now := time.Now()
	for _, provider := range providers {
		state := mediaTransferCircuits.items[provider.ID]
		health := MediaStorageProviderHealth{
			ProviderID: provider.ID, Available: state.openUntil.IsZero() || !now.Before(state.openUntil),
			ConsecutiveFailures: state.failures, LastError: state.lastError,
		}
		if !state.openUntil.IsZero() {
			health.OpenUntil = state.openUntil.Unix()
		}
		result = append(result, health)
	}
	return result
}

func mediaTransferSpoolDir() string {
	if value := strings.TrimSpace(os.Getenv("MEDIA_SPOOL_DIR")); value != "" {
		return value
	}
	return "/data/media-spool"
}

func mediaTransferTimeout(name string, fallback time.Duration) time.Duration {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil && value > 0 {
			return value
		}
	}
	return fallback
}

func mediaTransferFilePath(id int64) string {
	return filepath.Join(mediaTransferSpoolDir(), strconv.FormatInt(id, 10)+".media")
}

// EnqueueVideoMediaTransfer commits a still-processing task and an encrypted
// transfer job together. The upstream generation is never called again by the
// worker. The caller prepares FINALIZE_PENDING billing before invoking this
// function, so upstream cost is settled independently of media delivery.
func EnqueueVideoMediaTransfer(ctx context.Context, task, previous *model.Task, sourceURL string, options MediaDownloadOptions) (bool, error) {
	if task == nil || previous == nil || task.ID <= 0 {
		return false, fmt.Errorf("video task is not persisted")
	}
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false, fmt.Errorf("video transfer requires an absolute http(s) source URL")
	}
	if err := os.MkdirAll(mediaTransferSpoolDir(), 0700); err != nil {
		return false, fmt.Errorf("create media spool directory: %w", err)
	}
	sourceCipher, err := encryptMediaTransferValue(sourceURL)
	if err != nil {
		return false, err
	}
	optionsJSON, err := common.Marshal(options)
	if err != nil {
		return false, err
	}
	optionsCipher, err := encryptMediaTransferValue(string(optionsJSON))
	if err != nil {
		return false, err
	}
	responseCipher := ""
	if len(task.Data) > 0 && len(task.Data) <= mediaTransferDataLimit {
		responseCipher, err = encryptMediaTransferValue(string(task.Data))
		if err != nil {
			return false, err
		}
	}
	job := &model.MediaTransferJob{
		SourceCiphertext: sourceCipher, OptionsCiphertext: optionsCipher,
		ResponseCiphertext: responseCipher,
	}
	MarkVideoMediaTransferPending(task)
	task.MediaTransferState = string(model.MediaTransferPending)
	won, err := model.EnqueueMediaTransferAndTask(task, previous, job)
	if err != nil || !won {
		return won, err
	}
	if task.BillingStatus == model.TaskBillingStatusFinalizePending {
		if _, billingErr := ApplyPendingTaskBilling(ctx, task, "上游生成成功，等待媒体转存"); billingErr != nil {
			// The billing transition is durable and recovered by the existing
			// billing recovery loop; it must not roll back the transfer job.
			logger.LogError(ctx, fmt.Sprintf("finalize upstream billing for media task %s failed: %v", task.TaskID, billingErr))
		}
	}
	WakeMediaTransferWorker()
	return true, nil
}

// StartMediaTransferWorker runs on the master node after the database has
// migrated. The DB lease prevents concurrent processing by multiple masters.
func StartMediaTransferWorker() {
	go func() {
		ticker := time.NewTicker(mediaTransferTick)
		defer ticker.Stop()
		for {
			processDueMediaTransfers(context.Background())
			select {
			case <-ticker.C:
			case <-mediaTransferWake:
			}
		}
	}()
}

func processDueMediaTransfers(ctx context.Context) {
	for {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			logger.LogError(ctx, "create media transfer lease token failed: "+err.Error())
			return
		}
		job, err := model.ClaimDueMediaTransfer(time.Now().Unix(), hex.EncodeToString(tokenBytes))
		if err != nil {
			logger.LogError(ctx, "claim media transfer failed: "+err.Error())
			return
		}
		if job == nil {
			return
		}
		processMediaTransferJob(ctx, job)
	}
}

func processMediaTransferJob(ctx context.Context, job *model.MediaTransferJob) {
	stopRenew := make(chan struct{})
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(mediaTransferLeaseTick)
		defer ticker.Stop()
		for {
			select {
			case <-stopRenew:
				return
			case <-ticker.C:
				if err := model.RenewMediaTransferLease(job.ID, job.LeaseToken); err != nil {
					logger.LogError(ctx, fmt.Sprintf("renew media transfer lease %d failed: %v", job.ID, err))
				}
			}
		}
	}()
	defer func() { close(stopRenew); <-renewDone }()

	if err := transferMediaJob(ctx, job); err != nil {
		attempts := job.Attempts + 1
		failed := attempts >= mediaTransferMaxAttempts
		backoff := mediaTransferBackoff(attempts)
		// Never persist raw HTTP errors: they may include signed source URLs.
		category := mediaTransferErrorCategory(err)
		if scheduleErr := model.ScheduleMediaTransferRetry(job, time.Now().Add(backoff).Unix(), category, failed); scheduleErr != nil {
			logger.LogError(ctx, fmt.Sprintf("schedule media transfer %d retry failed: %v", job.ID, scheduleErr))
		}
		if failed {
			logger.LogError(ctx, fmt.Sprintf("media transfer %d exhausted retries (%s); upstream task remains charged and awaiting operator review", job.ID, category))
		} else {
			logger.LogWarn(ctx, fmt.Sprintf("media transfer %d attempt %d failed (%s); retry scheduled", job.ID, attempts, category))
		}
	}
}

func mediaTransferBackoff(attempt int) time.Duration {
	intervals := []time.Duration{5, 20, 60, 300, 900, 1800, 3600, 7200}
	if attempt <= 0 {
		return 5 * time.Second
	}
	if attempt > len(intervals) {
		attempt = len(intervals)
	}
	return intervals[attempt-1] * time.Second
}

func mediaTransferErrorCategory(err error) string {
	if err == nil {
		return "unknown"
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return "timeout"
	}
	if strings.Contains(err.Error(), "HTTP 5") {
		return "upstream_or_storage_5xx"
	}
	if strings.Contains(err.Error(), "HTTP 4") {
		return "upstream_or_storage_4xx"
	}
	if strings.Contains(err.Error(), "too large") {
		return "file_too_large"
	}
	if strings.Contains(err.Error(), "verify") {
		return "verification_failed"
	}
	return "transfer_failed"
}

func transferMediaJob(ctx context.Context, job *model.MediaTransferJob) error {
	sourceURL, err := decryptMediaTransferValue(job.SourceCiphertext)
	if err != nil {
		return fmt.Errorf("decrypt source: %w", err)
	}
	optionsJSON, err := decryptMediaTransferValue(job.OptionsCiphertext)
	if err != nil {
		return fmt.Errorf("decrypt options: %w", err)
	}
	var options MediaDownloadOptions
	if err := common.Unmarshal([]byte(optionsJSON), &options); err != nil {
		return fmt.Errorf("decode options: %w", err)
	}
	// A successful upload followed by a transient verification failure is
	// checkpointed. Verify that URL first on the next attempt so a retry never
	// creates another copy unnecessarily.
	if job.CandidateURLCiphertext != "" {
		candidateURL, decryptErr := decryptMediaTransferValue(job.CandidateURLCiphertext)
		if decryptErr != nil {
			return fmt.Errorf("decrypt candidate URL: %w", decryptErr)
		}
		if verifyErr := verifyTransferredVideoURL(ctx, candidateURL); verifyErr == nil {
			recordMediaProviderSuccess(job.CandidateProvider)
			responseData := transferredVideoResponse(job, sourceURL, candidateURL)
			task, won, completeErr := model.CompleteMediaTransfer(job, candidateURL, responseData)
			if completeErr != nil {
				return completeErr
			}
			if !won {
				return fmt.Errorf("media transfer task transition was superseded")
			}
			if task.BillingStatus == model.TaskBillingStatusFinalizePending {
				if _, billingErr := ApplyPendingTaskBilling(ctx, task, "媒体转存完成"); billingErr != nil {
					logger.LogError(ctx, fmt.Sprintf("complete media task %s billing recovery needed: %v", task.TaskID, billingErr))
				}
			}
			filePath := mediaTransferFilePath(job.ID)
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				logger.LogWarn(ctx, fmt.Sprintf("remove completed media spool file %d failed: %v", job.ID, err))
			}
			return nil
		} else {
			recordMediaProviderFailure(job.CandidateProvider, verifyErr)
		}
	}
	filePath, size, sha, filename, contentType, err := ensureMediaTransferFile(ctx, job, sourceURL, options)
	if err != nil {
		return err
	}
	if err := model.SaveMediaTransferFile(job.ID, job.LeaseToken, size, sha, filename, contentType); err != nil {
		return err
	}
	providers := GetMediaStorageProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no media storage provider")
	}
	var lastErr error
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		if !mediaProviderAvailable(provider.ID, time.Now()) {
			continue
		}
		storedURL, uploadErr := uploadMediaFile(ctx, provider, filePath, size, filename, contentType)
		if uploadErr != nil {
			recordMediaProviderFailure(provider.ID, uploadErr)
			lastErr = uploadErr
			continue
		}
		job.CandidateProvider = provider.ID
		candidateCiphertext, encryptErr := encryptMediaTransferValue(storedURL)
		if encryptErr != nil {
			return encryptErr
		}
		job.CandidateURLCiphertext = candidateCiphertext
		if checkpointErr := model.SaveMediaTransferCandidate(job.ID, job.LeaseToken, candidateCiphertext, provider.ID); checkpointErr != nil {
			return checkpointErr
		}
		if verifyErr := verifyTransferredVideoURL(ctx, storedURL); verifyErr != nil {
			recordMediaProviderFailure(provider.ID, verifyErr)
			lastErr = verifyErr
			continue
		}
		recordMediaProviderSuccess(provider.ID)
		responseData := transferredVideoResponse(job, sourceURL, storedURL)
		task, won, completeErr := model.CompleteMediaTransfer(job, storedURL, responseData)
		if completeErr != nil {
			return completeErr
		}
		if !won {
			return fmt.Errorf("media transfer task transition was superseded")
		}
		if task.BillingStatus == model.TaskBillingStatusFinalizePending {
			if _, billingErr := ApplyPendingTaskBilling(ctx, task, "媒体转存完成"); billingErr != nil {
				logger.LogError(ctx, fmt.Sprintf("complete media task %s billing recovery needed: %v", task.TaskID, billingErr))
			}
		}
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logger.LogWarn(ctx, fmt.Sprintf("remove completed media spool file %d failed: %v", job.ID, err))
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no enabled media storage provider")
	}
	return lastErr
}

func transferredVideoResponse(job *model.MediaTransferJob, sourceURL, storedURL string) []byte {
	if job.ResponseCiphertext != "" {
		if raw, err := decryptMediaTransferValue(job.ResponseCiphertext); err == nil {
			var payload any
			if err := common.Unmarshal([]byte(raw), &payload); err == nil {
				payload = replaceMediaSourceURL(payload, sourceURL, storedURL)
				if data, err := common.Marshal(payload); err == nil {
					return data
				}
			}
		}
	}
	data, _ := common.Marshal(map[string]string{"media_transfer": "ready", "url": storedURL})
	return data
}

func replaceMediaSourceURL(value any, sourceURL, storedURL string) any {
	switch current := value.(type) {
	case string:
		return strings.ReplaceAll(current, sourceURL, storedURL)
	case []any:
		for i := range current {
			current[i] = replaceMediaSourceURL(current[i], sourceURL, storedURL)
		}
	case map[string]any:
		for key, nested := range current {
			current[key] = replaceMediaSourceURL(nested, sourceURL, storedURL)
		}
	}
	return value
}

func ensureMediaTransferFile(ctx context.Context, job *model.MediaTransferJob, sourceURL string, options MediaDownloadOptions) (string, int64, string, string, string, error) {
	filePath := mediaTransferFilePath(job.ID)
	if stat, err := os.Stat(filePath); err == nil && stat.Size() > 0 && stat.Size() <= mediaStorageMaxBytes {
		file, err := os.Open(filePath)
		if err != nil {
			return "", 0, "", "", "", err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		_ = file.Close()
		if copyErr != nil {
			return "", 0, "", "", "", copyErr
		}
		filename := job.FileName
		if filename == "" {
			filename = mediaTransferFileName(sourceURL, "video/mp4")
		}
		contentType := job.ContentType
		if contentType == "" {
			contentType = "video/mp4"
		}
		return filePath, stat.Size(), hex.EncodeToString(hash.Sum(nil)), filename, contentType, nil
	}
	if err := os.MkdirAll(mediaTransferSpoolDir(), 0700); err != nil {
		return "", 0, "", "", "", err
	}
	partial, err := os.CreateTemp(mediaTransferSpoolDir(), strconv.FormatInt(job.ID, 10)+"-*.part")
	if err != nil {
		return "", 0, "", "", "", err
	}
	defer os.Remove(partial.Name())
	defer partial.Close()
	requestCtx, cancel := context.WithTimeout(ctx, mediaTransferTimeout("MEDIA_DOWNLOAD_TIMEOUT", 15*time.Minute))
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", 0, "", "", "", err
	}
	parsed := req.URL
	if options.AuthHeader != "" || options.AuthValue != "" || options.CredentialOrigin != "" {
		if !isValidMediaStorageHeaderName(options.AuthHeader) || strings.ContainsAny(options.AuthValue, "\r\n") {
			return "", 0, "", "", "", fmt.Errorf("invalid media download credential")
		}
		credentialOrigin, parseErr := url.Parse(strings.TrimSpace(options.CredentialOrigin))
		if parseErr != nil || credentialOrigin.Host == "" || (credentialOrigin.Scheme != "http" && credentialOrigin.Scheme != "https") {
			return "", 0, "", "", "", fmt.Errorf("invalid media download credential origin")
		}
		if sameHTTPOrigin(parsed, credentialOrigin) {
			req.Header.Set(options.AuthHeader, options.AuthValue)
		}
	}
	client, err := GetHttpClientWithProxy(strings.TrimSpace(options.Proxy))
	if err != nil {
		return "", 0, "", "", "", err
	}
	resp, err := DoSSRFProtectedRequest(client, req)
	if err != nil {
		return "", 0, "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", 0, "", "", "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > mediaStorageMaxBytes {
		return "", 0, "", "", "", fmt.Errorf("downloaded media is too large")
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(partial, hash), io.LimitReader(resp.Body, mediaStorageMaxBytes+1))
	if err != nil {
		return "", 0, "", "", "", err
	}
	if size == 0 || size > mediaStorageMaxBytes {
		return "", 0, "", "", "", fmt.Errorf("downloaded media is empty or too large")
	}
	if err := partial.Sync(); err != nil {
		return "", 0, "", "", "", err
	}
	if err := partial.Close(); err != nil {
		return "", 0, "", "", "", err
	}
	if err := os.Rename(partial.Name(), filePath); err != nil {
		return "", 0, "", "", "", err
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "video/mp4"
	}
	return filePath, size, hex.EncodeToString(hash.Sum(nil)), mediaTransferFileName(sourceURL, contentType), contentType, nil
}

func mediaTransferFileName(sourceURL, contentType string) string {
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return "generated-video.mp4"
	}
	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "generated-video"
	}
	if path.Ext(filename) == "" {
		filename += mediaStorageExtension(contentType, "video/mp4")
	}
	return filename
}

func uploadMediaFile(ctx context.Context, provider MediaStorageProvider, filePath string, size int64, filename, contentType string) (string, error) {
	provider = normalizeMediaStorageProvider(provider)
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var headers bytes.Buffer
	writer := multipart.NewWriter(&headers)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, provider.FieldName, path.Base(filename)))
	if parsedType, _, err := mime.ParseMediaType(contentType); err == nil {
		partHeader.Set("Content-Type", parsedType)
	} else {
		partHeader.Set("Content-Type", "video/mp4")
	}
	if _, err := writer.CreatePart(partHeader); err != nil {
		return "", err
	}
	prefixLength := headers.Len()
	if err := writer.Close(); err != nil {
		return "", err
	}
	body := headers.Bytes()
	requestCtx, cancel := context.WithTimeout(ctx, mediaTransferTimeout("MEDIA_UPLOAD_TIMEOUT", 15*time.Minute))
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, provider.UploadURL,
		io.MultiReader(bytes.NewReader(body[:prefixLength]), file, bytes.NewReader(body[prefixLength:])))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(body)) + size
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if provider.Token != "" {
		req.Header.Set(provider.AuthHeader, provider.AuthPrefix+provider.Token)
	}
	client := GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	redirectSafeClient := *client
	baseCheckRedirect := client.CheckRedirect
	redirectSafeClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) > 0 && !sameHTTPOrigin(next.URL, via[len(via)-1].URL) {
			next.Header.Del(provider.AuthHeader)
		}
		if baseCheckRedirect != nil {
			return baseCheckRedirect(next, via)
		}
		return nil
	}
	resp, err := DoSSRFProtectedRequest(&redirectSafeClient, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload returned HTTP %d", resp.StatusCode)
	}
	storedURL, err := extractMediaStorageResponseURL(responseBody, provider.ResponseURLPath)
	if err != nil {
		return "", err
	}
	return resolveMediaStorageResponseURL(provider.UploadURL, storedURL)
}

func verifyTransferredVideoURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("verify returned an invalid public URL")
	}
	if strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".zeabur.internal") {
		return fmt.Errorf("verify returned an internal-only URL")
	}
	requestCtx, cancel := context.WithTimeout(ctx, mediaTransferVerifyLimit)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-1")
	client := GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := DoSSRFProtectedRequest(client, req)
	if err != nil {
		return fmt.Errorf("verify media URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("verify media URL returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength == 0 {
		return fmt.Errorf("verify media URL returned an empty body")
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "text/html") || strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("verify media URL returned a non-video response")
	}
	var firstByte [1]byte
	if _, err := io.ReadFull(resp.Body, firstByte[:]); err != nil {
		return fmt.Errorf("verify media URL has no readable bytes: %w", err)
	}
	return nil
}
