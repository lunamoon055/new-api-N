package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

const (
	imageGenerationResultDir      = "image-results"
	maxImageGenerationResultBytes = 50 << 20
	maxImageGenerationResultCount = 16
)

var imageGenerationResultExtensions = map[string]string{
	"image/avif": ".avif",
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type imageGenerationLogProperties struct {
	Source    string   `json:"source"`
	Model     string   `json:"model"`
	Size      string   `json:"size,omitempty"`
	Quality   string   `json:"quality,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

// CaptureImageGenerationResponse extracts previewable image results without
// changing the response returned to the downstream client.
func CaptureImageGenerationResponse(c *gin.Context, responseBody []byte) []string {
	var response dto.ImageResponse
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return nil
	}
	return CaptureImageGenerationData(c, response.Data)
}

// RewriteImageGenerationResponse uploads image results before the response is
// sent downstream and replaces each successful result URL with the configured
// media-storage URL. If storage is unavailable, the original URL/base64
// result is retained by design.
func RewriteImageGenerationResponse(c *gin.Context, responseBody []byte) []byte {
	var response dto.ImageResponse
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return responseBody
	}
	if len(response.Data) == 0 {
		return responseBody
	}
	changed := false
	resultURLs := make([]string, 0, len(response.Data))
	seen := make(map[string]struct{}, len(response.Data))
	for index := range response.Data {
		urls := captureImageDataItem(c, &response.Data[index])
		if len(urls) == 0 {
			continue
		}
		for _, resultURL := range urls {
			if _, exists := seen[resultURL]; !exists && len(resultURLs) < maxImageGenerationResultCount {
				seen[resultURL] = struct{}{}
				resultURLs = append(resultURLs, resultURL)
			}
		}
		if response.Data[index].Url != urls[0] {
			changed = true
		}
		response.Data[index].Url = urls[0]
	}
	if len(resultURLs) > 0 {
		common.SetContextKey(c, constant.ContextKeyImageResultURLs, resultURLs)
	}
	if !changed {
		return responseBody
	}
	result, err := common.Marshal(response)
	if err != nil {
		return responseBody
	}
	return result
}

// CaptureImageGenerationData keeps remote URLs and persists base64 images as
// local preview files. Raw base64 data is deliberately never stored in the DB.
func CaptureImageGenerationData(c *gin.Context, data []dto.ImageData) []string {
	if c == nil || len(data) == 0 {
		return nil
	}

	resultURLs := make([]string, 0, len(data))
	seen := make(map[string]struct{}, len(data))
	appendURL := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || len(resultURLs) >= maxImageGenerationResultCount {
			return
		}
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		resultURLs = append(resultURLs, candidate)
	}

	for index := range data {
		if len(resultURLs) >= maxImageGenerationResultCount {
			break
		}
		itemURLs := captureImageDataItem(c, &data[index])
		if len(itemURLs) > 0 {
			data[index].Url = itemURLs[0]
		}
		for _, itemURL := range itemURLs {
			appendURL(itemURL)
		}
	}

	if len(resultURLs) > 0 {
		common.SetContextKey(c, constant.ContextKeyImageResultURLs, resultURLs)
	}
	return resultURLs
}

func captureImageDataItem(c *gin.Context, image *dto.ImageData) []string {
	if image == nil {
		return nil
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	appendURL := func(raw string) []string {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		if HasEnabledMediaStorage() {
			if uploadedURL, err := UploadMediaURL(ctx, raw, "image/jpeg"); err == nil && uploadedURL != "" {
				return []string{uploadedURL}
			}
		}
		if normalizedURL, ok := normalizeImageGenerationResultURL(raw); ok {
			return []string{normalizedURL}
		}
		return nil
	}
	storeBytes := func(data []byte, filename, contentType string) []string {
		if HasEnabledMediaStorage() {
			if uploadedURL, err := UploadMediaBytes(ctx, data, filename, contentType); err == nil && uploadedURL != "" {
				return []string{uploadedURL}
			}
		}
		return nil
	}

	if imageURL := strings.TrimSpace(image.Url); imageURL != "" {
		if strings.HasPrefix(imageURL, "data:image/") {
			if data, filename, contentType, err := decodeDataURL(imageURL, "image/jpeg"); err == nil {
				if uploaded := storeBytes(data, filename, contentType); len(uploaded) > 0 {
					return uploaded
				}
			}
			if storedURL, err := persistBase64ImageGenerationResult(imageURL); err == nil {
				return []string{storedURL}
			}
			return nil
		}
		return appendURL(imageURL)
	}
	if encoded := strings.TrimSpace(image.B64Json); encoded != "" {
		if data, filename, contentType, err := decodeBase64Image(encoded); err == nil {
			if uploaded := storeBytes(data, filename, contentType); len(uploaded) > 0 {
				return uploaded
			}
		}
		if storedURL, err := persistBase64ImageGenerationResult(encoded); err == nil {
			return []string{storedURL}
		}
	}
	return nil
}

func decodeBase64Image(encoded string) ([]byte, string, string, error) {
	encoded = strings.TrimSpace(encoded)
	var (
		data        []byte
		filename    string
		contentType string
		err         error
	)
	if strings.HasPrefix(encoded, "data:") {
		data, filename, contentType, err = decodeDataURL(encoded, "image/jpeg")
	} else {
		data, filename, contentType, err = decodeDataURL("data:image/jpeg;base64,"+encoded, "image/jpeg")
	}
	if err != nil {
		return nil, "", "", err
	}
	if detected := detectImageGenerationResultMIME(data); detected != "" {
		contentType = detected
		if extension, ok := imageGenerationResultExtensions[detected]; ok {
			filename = "generated" + extension
		}
	}
	return data, filename, contentType, nil
}

func GetCapturedImageGenerationResultURLs(c *gin.Context) []string {
	if c == nil {
		return nil
	}
	return common.GetContextKeyStringSlice(c, constant.ContextKeyImageResultURLs)
}

// RecordImageGenerationLog adds successful synchronous image requests to the
// existing drawing-log data source. Failure to write this auxiliary log must
// not change the already successful downstream response.
func RecordImageGenerationLog(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, imageURLs []string) error {
	task, err := buildImageGenerationLog(c, info, request, imageURLs, time.Now())
	if err != nil {
		return err
	}
	if err := task.Insert(); err != nil {
		return fmt.Errorf("image generation log: insert: %w", err)
	}
	return nil
}

func buildImageGenerationLog(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, imageURLs []string, now time.Time) (*model.Midjourney, error) {
	if info == nil || request == nil {
		return nil, errors.New("image generation log: missing request context")
	}

	startTime := info.StartTime
	if startTime.IsZero() {
		startTime = now
	}
	taskID := strings.TrimSpace(info.RequestId)
	if taskID == "" && c != nil {
		taskID = strings.TrimSpace(c.GetString(common.RequestIdKey))
	}
	if taskID == "" {
		taskID = fmt.Sprintf("image-%d-%d", now.UnixNano(), info.UserId)
	}

	propertiesJSON, err := common.Marshal(imageGenerationLogProperties{
		Source:    "image_generation",
		Model:     info.OriginModelName,
		Size:      request.Size,
		Quality:   request.Quality,
		ImageURLs: imageURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("image generation log: encode properties: %w", err)
	}

	imageURL := ""
	if len(imageURLs) > 0 {
		imageURL = imageURLs[0]
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	return &model.Midjourney{
		Code:        1,
		UserId:      info.UserId,
		Action:      constant.MjActionImageGeneration,
		MjId:        taskID,
		Prompt:      request.Prompt,
		Description: info.OriginModelName,
		State:       "image_generation",
		SubmitTime:  startTime.UnixMilli(),
		StartTime:   startTime.UnixMilli(),
		FinishTime:  now.UnixMilli(),
		ImageUrl:    imageURL,
		Status:      "SUCCESS",
		Progress:    "100%",
		ChannelId:   channelID,
		Properties:  string(propertiesJSON),
	}, nil
}

func normalizeImageGenerationResultURL(rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "/") && !strings.HasPrefix(rawURL, "//") {
		return rawURL, true
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return rawURL, true
}

func persistBase64ImageGenerationResult(encoded string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if comma := strings.Index(encoded, ","); strings.HasPrefix(encoded, "data:") && comma >= 0 {
		header := encoded[:comma]
		if !strings.Contains(header, ";base64") {
			return "", errors.New("image generation result is not base64 encoded")
		}
		encoded = encoded[comma+1:]
	}
	if encoded == "" || base64.StdEncoding.DecodedLen(len(encoded)) > maxImageGenerationResultBytes {
		return "", errors.New("image generation result is empty or too large")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return "", fmt.Errorf("decode image generation result: %w", err)
	}
	if len(data) == 0 || len(data) > maxImageGenerationResultBytes {
		return "", errors.New("image generation result is empty or too large")
	}

	mimeType := detectImageGenerationResultMIME(data)
	extension, ok := imageGenerationResultExtensions[mimeType]
	if !ok {
		return "", errors.New("image generation result has an unsupported format")
	}
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("prepare image generation result filename: %w", err)
	}
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), hex.EncodeToString(randomBytes), extension)
	dir := imageGenerationResultStorageDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create image generation result directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o600); err != nil {
		return "", fmt.Errorf("store image generation result: %w", err)
	}
	return "/api/image-results/" + filename, nil
}

func detectImageGenerationResultMIME(data []byte) string {
	mimeType := http.DetectContentType(data)
	if _, ok := imageGenerationResultExtensions[mimeType]; ok {
		return mimeType
	}
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		brand := string(data[8:12])
		if brand == "avif" || brand == "avis" {
			return "image/avif"
		}
	}
	return ""
}

func imageGenerationResultStorageDir() string {
	if configured := strings.TrimSpace(os.Getenv("IMAGE_RESULT_STORAGE_PATH")); configured != "" {
		return configured
	}
	databasePath := strings.TrimSpace(common.SQLitePath)
	databasePath = strings.TrimPrefix(databasePath, "file:")
	if index := strings.IndexAny(databasePath, "?#"); index >= 0 {
		databasePath = databasePath[:index]
	}
	baseDir := filepath.Dir(databasePath)
	if databasePath == "" || databasePath == ":memory:" || baseDir == "" {
		baseDir = "."
	}
	return filepath.Join(baseDir, imageGenerationResultDir)
}

func OpenImageGenerationResult(filename string) (*os.File, string, int64, error) {
	if !isSafeImageGenerationResultName(filename) {
		return nil, "", 0, os.ErrNotExist
	}
	path := filepath.Join(imageGenerationResultStorageDir(), filename)
	file, err := os.Open(path)
	if err != nil {
		return nil, "", 0, err
	}
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		_ = file.Close()
		if err == nil {
			err = os.ErrNotExist
		}
		return nil, "", 0, err
	}
	mimeType := imageGenerationResultMIMEFromName(filename)
	if mimeType == "" {
		_ = file.Close()
		return nil, "", 0, os.ErrNotExist
	}
	return file, mimeType, info.Size(), nil
}

func isSafeImageGenerationResultName(filename string) bool {
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\\`) {
		return false
	}
	return imageGenerationResultMIMEFromName(filename) != ""
}

func imageGenerationResultMIMEFromName(filename string) string {
	extension := strings.ToLower(filepath.Ext(filename))
	for mimeType, candidate := range imageGenerationResultExtensions {
		if extension == candidate {
			return mimeType
		}
	}
	return ""
}
