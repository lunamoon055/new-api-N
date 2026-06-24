package sora

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

var video2ImageExtensions = map[string]struct{}{
	"avif": {},
	"gif":  {},
	"jpeg": {},
	"jpg":  {},
	"png":  {},
	"webp": {},
}

var video2VideoExtensions = map[string]struct{}{
	"mp4": {},
}

var video2AudioExtensions = map[string]struct{}{
	"mp3": {},
	"wav": {},
}

var video2ImageMimeTypes = map[string]struct{}{
	"image/avif": {},
	"image/gif":  {},
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

type video2Reference struct {
	URL string `json:"url"`
}

type video2Request struct {
	Prompt         string            `json:"prompt"`
	Duration       *int              `json:"duration,omitempty"`
	Size           string            `json:"size,omitempty"`
	AspectRatio    string            `json:"aspect_ratio,omitempty"`
	Resolution     string            `json:"resolution,omitempty"`
	ImageURL       string            `json:"image_url,omitempty"`
	ImageURLs      []string          `json:"image_urls,omitempty"`
	VideoURL       string            `json:"video_url,omitempty"`
	VideoReference []video2Reference `json:"video_reference,omitempty"`
	StartImageURL  string            `json:"start_image_url,omitempty"`
	EndImageURL    string            `json:"end_image_url,omitempty"`
	AudioURL       string            `json:"audio_url,omitempty"`
	Async          *bool             `json:"async,omitempty"`
}

func isVideo2Model(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "video-2.0", "video-2.0-fast":
		return true
	default:
		return false
	}
}

func validateVideo2Request(req video2Request) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if req.Duration != nil && (*req.Duration < 4 || *req.Duration > 15) {
		return fmt.Errorf("duration must be between 4 and 15")
	}

	allowedRatios := map[string]string{
		"9:16": "720x1280",
		"16:9": "1280x720",
		"1:1":  "720x720",
	}
	if req.AspectRatio != "" {
		expectedSize, ok := allowedRatios[req.AspectRatio]
		if !ok {
			return fmt.Errorf("aspect_ratio must be 9:16, 16:9, or 1:1")
		}
		if req.Size != "" && req.Size != expectedSize {
			return fmt.Errorf("size conflicts with aspect_ratio")
		}
	}
	if req.Resolution != "" && req.Resolution != "720p" {
		return fmt.Errorf("resolution must be 720p")
	}
	if req.Size != "" && req.Size != "720x1280" && req.Size != "1280x720" && req.Size != "720x720" {
		return fmt.Errorf("size is invalid for Video2")
	}

	imageURLs := make([]string, 0, len(req.ImageURLs)+3)
	imageURLs = append(imageURLs, req.ImageURL)
	imageURLs = append(imageURLs, req.ImageURLs...)
	imageURLs = append(imageURLs, req.StartImageURL, req.EndImageURL)

	videoURLs := make([]string, 0, len(req.VideoReference)+1)
	videoURLs = append(videoURLs, req.VideoURL)
	for _, reference := range req.VideoReference {
		videoURLs = append(videoURLs, reference.URL)
	}

	if countNonBlank(imageURLs) > 4 {
		return fmt.Errorf("image references must not exceed 4")
	}
	if countNonBlank(videoURLs) > 3 {
		return fmt.Errorf("video references must not exceed 3")
	}
	for _, value := range imageURLs {
		if err := validateVideo2ImageReference(value); err != nil {
			return fmt.Errorf("image reference: %w", err)
		}
	}
	for _, value := range videoURLs {
		if err := validateVideo2URLWithExtension(value, video2VideoExtensions, "MP4"); err != nil {
			return fmt.Errorf("video reference: %w", err)
		}
	}
	if err := validateVideo2URLWithExtension(req.AudioURL, video2AudioExtensions, "MP3 or WAV"); err != nil {
		return fmt.Errorf("audio_url: %w", err)
	}

	return nil
}

func validateVideo2ImageReference(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if mime, ok := getVideo2DataURLMime(value); ok {
		if !strings.HasPrefix(mime, "image/") {
			return fmt.Errorf("URL must use http or https")
		}
		if _, ok := video2ImageMimeTypes[mime]; !ok {
			return fmt.Errorf("format must be PNG, JPEG, WebP, GIF, or AVIF")
		}
		return nil
	}
	return validateVideo2ReferenceWithExtension(value, video2ImageExtensions, video2ImageMimeTypes, "PNG, JPEG, WebP, GIF, or AVIF")
}

func countNonBlank(values []string) int {
	count := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	return count
}

func validateVideo2URL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("URL must use http or https")
	}
	return nil
}

func validateVideo2URLWithExtension(value string, allowed map[string]struct{}, label string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if err := validateVideo2URL(value); err != nil {
		return err
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("URL must use http or https")
	}
	extension := getVideo2URLExtension(parsed.Path)
	if _, ok := allowed[extension]; !ok {
		return fmt.Errorf("format must be %s", label)
	}
	return nil
}

func validateVideo2ReferenceWithExtension(value string, allowedExtensions map[string]struct{}, allowedMimes map[string]struct{}, label string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if mime, ok := getVideo2DataURLMime(value); ok {
		if _, ok := allowedMimes[mime]; !ok {
			return fmt.Errorf("format must be %s", label)
		}
		return nil
	}
	return validateVideo2URLWithExtension(value, allowedExtensions, label)
}

func getVideo2DataURLMime(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "data:") {
		return "", false
	}
	metaEnd := strings.Index(value, ",")
	if metaEnd < 0 {
		return "", true
	}
	meta := value[len("data:"):metaEnd]
	separator := strings.Index(meta, ";")
	if separator >= 0 {
		meta = meta[:separator]
	}
	return strings.ToLower(strings.TrimSpace(meta)), true
}

func getVideo2URLExtension(path string) string {
	index := strings.LastIndex(path, ".")
	if index < 0 || index == len(path)-1 {
		return ""
	}
	return strings.ToLower(path[index+1:])
}

func validateVideo2JSONRequest(c *gin.Context, modelName string) *dto.TaskError {
	if !isVideo2Model(modelName) {
		return nil
	}

	var req video2Request
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateVideo2Request(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}
