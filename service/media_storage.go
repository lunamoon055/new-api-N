package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

const (
	MediaStorageOptionKey     = "MediaStorageProviders"
	mediaStorageMaxBytes      = 200 << 20
	mediaStorageUploadTimeout = 2 * time.Minute
	// Stage 1 deliberately retries the complete transfer a small number of
	// times. The durable spool/queue in the next stage will avoid downloading
	// the upstream asset again on every retry.
	mediaStorageVideoRetryAttempts = 3
)

// MediaStorageProvider is an administrator-configured upload target. Token is
// never returned by the settings read endpoint; it is only used server-side.
type MediaStorageProvider struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	UploadURL  string `json:"upload_url"`
	AuthHeader string `json:"auth_header"`
	AuthPrefix string `json:"auth_prefix"`
	Token      string `json:"token"`
	FieldName  string `json:"field_name"`
	Priority   int    `json:"priority"`
	// ResponseURLPath is a dot-separated JSON path, for example "url",
	// "data.url", or "0.src" for the first item in a JSON array. Keeping this
	// explicit avoids guessing undocumented response shapes while allowing
	// providers with a documented envelope.
	ResponseURLPath string `json:"response_url_path"`
}

// MediaDownloadOptions controls how a generated upstream asset is fetched
// before it is copied to media storage. Credentials are attached only when
// the source URL has the same origin as CredentialOrigin, and are stripped by
// the SSRF-protected client if a redirect crosses origins.
type MediaDownloadOptions struct {
	Proxy            string
	CredentialOrigin string
	AuthHeader       string
	AuthValue        string
}

var ErrMediaStorageProviderNotFound = errors.New("media storage provider not found")

const mediaStorageTestPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

// GetMediaStorageProviders reads the current provider list without caching it
// separately from the existing option system, so administrator changes apply
// to the next upload without a process restart.
func GetMediaStorageProviders() []MediaStorageProvider {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[MediaStorageOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var providers []MediaStorageProvider
	if err := common.UnmarshalJsonStr(raw, &providers); err != nil {
		logger.LogError(context.Background(), "invalid media storage provider configuration: "+err.Error())
		return nil
	}
	for i := range providers {
		providers[i] = normalizeMediaStorageProvider(providers[i])
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].Priority < providers[j].Priority
	})
	return providers
}

// HasEnabledMediaStorage reports whether a generation result should be
// copied to a configured media host. Keeping this check separate prevents a
// missing configuration from triggering a needless upstream download.
func HasEnabledMediaStorage() bool {
	for _, provider := range GetMediaStorageProviders() {
		if provider.Enabled {
			return true
		}
	}
	return false
}

func normalizeMediaStorageProvider(provider MediaStorageProvider) MediaStorageProvider {
	provider.ID = strings.TrimSpace(provider.ID)
	provider.Name = strings.TrimSpace(provider.Name)
	provider.UploadURL = strings.TrimSpace(provider.UploadURL)
	provider.AuthHeader = strings.TrimSpace(provider.AuthHeader)
	// Preserve intentional whitespace in the prefix (for example, the
	// trailing space in "Bearer "). Newlines are rejected during validation.
	provider.FieldName = strings.TrimSpace(provider.FieldName)
	provider.ResponseURLPath = strings.TrimSpace(provider.ResponseURLPath)
	if provider.AuthHeader == "" {
		provider.AuthHeader = "Authorization"
	}
	if provider.FieldName == "" {
		provider.FieldName = "file"
	}
	if provider.ResponseURLPath == "" {
		provider.ResponseURLPath = "0.src"
	}
	if provider.Priority < 0 {
		provider.Priority = 0
	}
	return provider
}

// ValidateMediaStorageProviders validates administrator input before it is
// persisted. It intentionally does not require a token because some image
// hosts are public or use a custom header configured later.
func ValidateMediaStorageProviders(providers []MediaStorageProvider) error {
	if len(providers) > 32 {
		return fmt.Errorf("too many media storage providers (maximum 32)")
	}
	seen := make(map[string]struct{}, len(providers))
	for index, rawProvider := range providers {
		provider := normalizeMediaStorageProvider(rawProvider)
		if provider.ID == "" {
			return fmt.Errorf("provider %d: id is required", index+1)
		}
		if _, exists := seen[provider.ID]; exists {
			return fmt.Errorf("provider %q is duplicated", provider.ID)
		}
		seen[provider.ID] = struct{}{}
		parsed, err := url.Parse(provider.UploadURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("provider %q: upload_url must be an absolute http(s) URL", provider.ID)
		}
		if provider.FieldName == "" {
			return fmt.Errorf("provider %q: field_name is required", provider.ID)
		}
		if !isValidMediaStorageHeaderName(provider.AuthHeader) || strings.ContainsAny(provider.AuthPrefix, "\r\n") {
			return fmt.Errorf("provider %q: authentication fields contain invalid characters", provider.ID)
		}
		if provider.FieldName == "" || strings.ContainsAny(provider.FieldName, "\r\n\"") {
			return fmt.Errorf("provider %q: field_name contains invalid characters", provider.ID)
		}
		if !isValidMediaStorageResponseURLPath(provider.ResponseURLPath) {
			return fmt.Errorf("provider %q: response_url_path must be a dot-separated JSON field path", provider.ID)
		}
	}
	return nil
}

func isValidMediaStorageResponseURLPath(value string) bool {
	if value == "" {
		return false
	}
	for _, segment := range strings.Split(value, ".") {
		if segment == "" {
			return false
		}
		for _, char := range segment {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') || char == '_' || char == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func isValidMediaStorageHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') {
			continue
		}
		switch char {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}

// UploadMediaURL tries enabled providers in priority order.
func UploadMediaURL(ctx context.Context, sourceURL, mediaType string) (string, error) {
	return UploadMediaURLWithOptions(ctx, sourceURL, mediaType, MediaDownloadOptions{})
}

// UploadMediaURLWithRetry retries a complete remote-media transfer. This is a
// short-term reliability guard for generated video results; the next-stage
// persistent spool/queue will make retries cheaper by reusing the downloaded
// file instead of fetching the upstream URL again.
func UploadMediaURLWithRetry(ctx context.Context, sourceURL, mediaType string) (string, error) {
	return UploadMediaURLWithOptionsRetry(ctx, sourceURL, mediaType, MediaDownloadOptions{})
}

// UploadMediaURLWithOptionsRetry is the credential-aware variant used for
// protected upstream video content endpoints.
func UploadMediaURLWithOptionsRetry(ctx context.Context, sourceURL, mediaType string, options MediaDownloadOptions) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= mediaStorageVideoRetryAttempts; attempt++ {
		storedURL, err := UploadMediaURLWithOptions(ctx, sourceURL, mediaType, options)
		if err == nil && strings.TrimSpace(storedURL) != "" {
			return storedURL, nil
		}
		if err == nil {
			err = fmt.Errorf("media storage returned an empty URL")
		}
		lastErr = err
		if attempt == mediaStorageVideoRetryAttempts {
			break
		}

		// Do not log the raw error here: HTTP client errors can echo a signed
		// upstream URL. The task ID is logged by the caller instead.
		logger.LogWarn(ctx, fmt.Sprintf(
			"media storage video transfer attempt %d/%d failed; retrying",
			attempt, mediaStorageVideoRetryAttempts,
		))
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("media storage transfer failed")
	}
	return "", fmt.Errorf("media storage video transfer failed after %d attempts: %w", mediaStorageVideoRetryAttempts, lastErr)
}

// UploadMediaURLWithOptions copies a remote media result while allowing a
// protected same-origin upstream content endpoint to be downloaded with its
// channel credential.
func UploadMediaURLWithOptions(ctx context.Context, sourceURL, mediaType string, options MediaDownloadOptions) (string, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return "", fmt.Errorf("media source URL is empty")
	}
	if !HasEnabledMediaStorage() {
		return sourceURL, fmt.Errorf("no enabled media storage provider")
	}
	data, filename, contentType, err := downloadMediaWithOptions(ctx, sourceURL, mediaType, options)
	if err != nil {
		return sourceURL, err
	}
	return UploadMediaBytes(ctx, data, filename, contentType)
}

// UploadMediaBytes uploads bytes to the first configured provider that accepts
// them. It is also usable for binary TTS responses that do not expose a URL.
func UploadMediaBytes(ctx context.Context, data []byte, filename, contentType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("media content is empty")
	}
	if len(data) > mediaStorageMaxBytes {
		return "", fmt.Errorf("media content exceeds %d bytes", mediaStorageMaxBytes)
	}
	providers := GetMediaStorageProviders()
	var lastErr error
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		result, err := uploadToMediaStorage(ctx, provider, data, filename, contentType)
		if err == nil {
			return result, nil
		}
		lastErr = fmt.Errorf("provider %s: %w", provider.ID, err)
		logger.LogWarn(ctx, lastErr.Error())
	}
	if lastErr == nil {
		return "", fmt.Errorf("no enabled media storage provider")
	}
	return "", lastErr
}

// TestMediaStorageProvider uploads a tiny PNG to one configured provider. It
// intentionally tests the persisted provider configuration even when the
// provider is disabled, so administrators can validate a provider before
// enabling it for generated media.
func TestMediaStorageProvider(ctx context.Context, providerID string) (string, error) {
	providerID = strings.TrimSpace(providerID)
	for _, provider := range GetMediaStorageProviders() {
		if provider.ID != providerID {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(mediaStorageTestPNGBase64)
		if err != nil {
			return "", fmt.Errorf("decode media storage test payload: %w", err)
		}
		return uploadToMediaStorage(ctx, provider, data, "new-api-media-storage-test.png", "image/png")
	}
	return "", ErrMediaStorageProviderNotFound
}

// AttachAudioStorageURL uploads a completed binary audio response and exposes
// the resulting URL in a response header. Audio endpoints historically return
// the binary stream, so replacing that body with JSON would break clients.
// The header is additive and upload failures leave the original body intact.
func AttachAudioStorageURL(c *gin.Context, data []byte, filename, contentType string) {
	if c == nil || c.Request == nil || !HasEnabledMediaStorage() {
		return
	}
	url, err := UploadMediaBytes(c.Request.Context(), data, filename, contentType)
	if err != nil {
		logger.LogWarn(c, "media storage upload failed for audio response: "+err.Error())
		return
	}
	c.Header("X-Media-Storage-URL", url)
}

func uploadToMediaStorage(ctx context.Context, provider MediaStorageProvider, data []byte, filename, contentType string) (string, error) {
	provider = normalizeMediaStorageProvider(provider)
	if filename == "" {
		filename = "generated.bin"
	}
	filename = path.Base(strings.ReplaceAll(filename, "\\", "/"))
	if filename == "." || filename == "/" {
		filename = "generated.bin"
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, provider.FieldName, filename))
	if parsedType, _, err := mime.ParseMediaType(contentType); err == nil {
		partHeader.Set("Content-Type", parsedType)
	} else {
		partHeader.Set("Content-Type", "application/octet-stream")
	}
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", err
	}
	if _, err = part.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	requestCtx, cancel := context.WithTimeout(ctx, mediaStorageUploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, provider.UploadURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if provider.Token != "" {
		req.Header.Set(provider.AuthHeader, provider.AuthPrefix+provider.Token)
	}
	client := GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	// A configured authentication header may have any valid HTTP token name,
	// so never forward credentials across redirects to another origin.
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
		// Do not include arbitrary upstream response text: it can contain
		// credentials or HTML from a proxy and is not actionable to callers.
		return "", fmt.Errorf("upload returned HTTP %d", resp.StatusCode)
	}
	parsedURLString, err := extractMediaStorageResponseURL(responseBody, provider.ResponseURLPath)
	if err != nil {
		return "", err
	}
	parsedURLString, err = resolveMediaStorageResponseURL(provider.UploadURL, parsedURLString)
	if err != nil {
		return "", fmt.Errorf("upload response contains an invalid url")
	}
	return parsedURLString, nil
}

func resolveMediaStorageResponseURL(uploadURL, responseURL string) (string, error) {
	base, err := url.Parse(uploadURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return "", fmt.Errorf("upload URL is invalid")
	}

	responseURL = strings.TrimSpace(responseURL)
	parsed, err := url.Parse(responseURL)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return "", fmt.Errorf("response URL is not http(s)")
		}
		return parsed.String(), nil
	}
	// A relative response is resolved against the configured upload origin.
	// Reject protocol-relative URLs so an upstream response cannot silently
	// redirect the stored result to another host.
	if parsed.Host != "" || (!strings.HasPrefix(responseURL, "/") && !strings.Contains(parsed.Path, "/")) {
		return "", fmt.Errorf("response URL is not an absolute or path URL")
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Host == "" || (resolved.Scheme != "http" && resolved.Scheme != "https") {
		return "", fmt.Errorf("resolved response URL is not http(s)")
	}
	return resolved.String(), nil
}

func extractMediaStorageResponseURL(responseBody []byte, responseURLPath string) (string, error) {
	responseURLPath = strings.TrimSpace(responseURLPath)
	if responseURLPath == "" {
		responseURLPath = "0.src"
	}

	paths := []string{responseURLPath}
	// CloudFlare-ImgBed/Sanyue ImgHub returns [{"src":"..."}]. Older
	// configurations in this project used "url", so keep those installations
	// working while the administrator updates the displayed path to 0.src.
	if responseURLPath == "url" {
		paths = append(paths, "0.src", "src")
	}

	var lastErr error
	for _, path := range paths {
		value, err := extractMediaStorageResponseURLAtPath(responseBody, path)
		if err == nil {
			return value, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func extractMediaStorageResponseURLAtPath(responseBody []byte, responseURLPath string) (string, error) {
	if !isValidMediaStorageResponseURLPath(responseURLPath) {
		return "", fmt.Errorf("upload response URL path is invalid")
	}

	current := json.RawMessage(responseBody)
	for _, segment := range strings.Split(responseURLPath, ".") {
		if index, err := strconv.Atoi(segment); err == nil {
			var array []json.RawMessage
			if err := common.Unmarshal(current, &array); err != nil {
				return "", fmt.Errorf("decode upload response array: %w", err)
			}
			if index < 0 || index >= len(array) {
				return "", fmt.Errorf("upload response does not contain %q", responseURLPath)
			}
			current = array[index]
			continue
		}

		var object map[string]json.RawMessage
		if err := common.Unmarshal(current, &object); err != nil {
			return "", fmt.Errorf("decode upload response object: %w", err)
		}
		next, ok := object[segment]
		if !ok {
			return "", fmt.Errorf("upload response does not contain %q", responseURLPath)
		}
		current = next
	}

	var value string
	if err := common.Unmarshal(current, &value); err != nil {
		return "", fmt.Errorf("upload response URL field is not a string: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("upload response URL field is empty")
	}
	return value, nil
}

func downloadMedia(ctx context.Context, sourceURL, mediaType string) ([]byte, string, string, error) {
	return downloadMediaWithOptions(ctx, sourceURL, mediaType, MediaDownloadOptions{})
}

func downloadMediaWithOptions(ctx context.Context, sourceURL, mediaType string, options MediaDownloadOptions) ([]byte, string, string, error) {
	if strings.HasPrefix(sourceURL, "data:") {
		return decodeDataURL(sourceURL, mediaType)
	}
	parsed, err := url.Parse(sourceURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, "", "", fmt.Errorf("unsupported media source URL")
	}
	requestCtx, cancel := context.WithTimeout(ctx, mediaStorageUploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	if options.AuthHeader != "" || options.AuthValue != "" || options.CredentialOrigin != "" {
		if !isValidMediaStorageHeaderName(options.AuthHeader) || strings.ContainsAny(options.AuthValue, "\r\n") {
			return nil, "", "", fmt.Errorf("invalid media download credential")
		}
		credentialOrigin, originErr := url.Parse(strings.TrimSpace(options.CredentialOrigin))
		if originErr != nil || credentialOrigin.Host == "" ||
			(credentialOrigin.Scheme != "http" && credentialOrigin.Scheme != "https") {
			return nil, "", "", fmt.Errorf("invalid media download credential origin")
		}
		if sameHTTPOrigin(parsed, credentialOrigin) {
			req.Header.Set(options.AuthHeader, options.AuthValue)
		}
	}
	client, err := GetHttpClientWithProxy(strings.TrimSpace(options.Proxy))
	if err != nil {
		return nil, "", "", fmt.Errorf("create media download client: %w", err)
	}
	resp, err := DoSSRFProtectedRequest(client, req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, mediaStorageMaxBytes+1))
	if err != nil {
		return nil, "", "", err
	}
	if len(data) == 0 || len(data) > mediaStorageMaxBytes {
		return nil, "", "", fmt.Errorf("downloaded media is empty or too large")
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "generated-" + time.Now().UTC().Format("20060102-150405")
	}
	if path.Ext(filename) == "" {
		if extension := mediaStorageExtension(contentType, mediaType); extension != "" {
			filename += extension
		}
	}
	return data, filename, contentType, nil
}

func mediaStorageExtension(contentType, fallbackType string) string {
	for _, rawType := range []string{contentType, fallbackType} {
		parsedType, _, err := mime.ParseMediaType(strings.TrimSpace(rawType))
		if err != nil {
			continue
		}
		switch strings.ToLower(parsedType) {
		case "video/mp4":
			return ".mp4"
		case "video/webm":
			return ".webm"
		case "audio/mpeg":
			return ".mp3"
		case "audio/wav", "audio/x-wav":
			return ".wav"
		case "image/jpeg":
			return ".jpg"
		case "image/png":
			return ".png"
		case "image/webp":
			return ".webp"
		}
	}
	return ""
}

func decodeDataURL(raw, mediaType string) ([]byte, string, string, error) {
	comma := strings.IndexByte(raw, ',')
	if comma <= 5 {
		return nil, "", "", fmt.Errorf("invalid data URL")
	}
	header, payload := raw[:comma], raw[comma+1:]
	if !strings.Contains(header, ";base64") {
		return nil, "", "", fmt.Errorf("data URL is not base64 encoded")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(payload)
	}
	if err != nil {
		return nil, "", "", err
	}
	contentType := strings.TrimPrefix(strings.TrimPrefix(header, "data:"), " ")
	if semicolon := strings.IndexByte(contentType, ';'); semicolon >= 0 {
		contentType = contentType[:semicolon]
	}
	if contentType == "" {
		contentType = mediaType
	}
	ext := ".bin"
	if dot := strings.LastIndex(contentType, "/"); dot >= 0 {
		ext = "." + contentType[dot+1:]
	}
	return data, "generated" + ext, contentType, nil
}
