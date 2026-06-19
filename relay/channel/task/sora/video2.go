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
	for field, values := range map[string][]string{
		"image reference": imageURLs,
		"video reference": videoURLs,
		"audio_url":       {req.AudioURL},
	} {
		for _, value := range values {
			if err := validateVideo2URL(value); err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
		}
	}

	return nil
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
