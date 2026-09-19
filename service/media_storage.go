package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"sort"
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
}

type mediaStorageUploadResponse struct {
	URL string `json:"url"`
}

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
	provider.AuthPrefix = strings.TrimSpace(provider.AuthPrefix)
	provider.FieldName = strings.TrimSpace(provider.FieldName)
	if provider.AuthHeader == "" {
		provider.AuthHeader = "Authorization"
	}
	if provider.FieldName == "" {
		provider.FieldName = "file"
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
	}
	return nil
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

// UploadMediaURL tries enabled providers in priority order. A provider failure
// is deliberately non-fatal to generation callers: they can keep the original
// upstream URL and report the upload failure separately.
func UploadMediaURL(ctx context.Context, sourceURL, mediaType string) (string, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return "", fmt.Errorf("media source URL is empty")
	}
	if !HasEnabledMediaStorage() {
		return sourceURL, fmt.Errorf("no enabled media storage provider")
	}
	data, filename, contentType, err := downloadMedia(ctx, sourceURL, mediaType)
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
	var parsed mediaStorageUploadResponse
	if err := common.Unmarshal(responseBody, &parsed); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	parsed.URL = strings.TrimSpace(parsed.URL)
	if parsed.URL == "" {
		return "", fmt.Errorf("upload response does not contain url")
	}
	parsedURL, err := url.Parse(parsed.URL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("upload response contains an invalid url")
	}
	return parsed.URL, nil
}

func downloadMedia(ctx context.Context, sourceURL, mediaType string) ([]byte, string, string, error) {
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
	client := GetHttpClient()
	if client == nil {
		client = http.DefaultClient
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
	return data, filename, contentType, nil
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
