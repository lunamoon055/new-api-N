package sora

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const suanliai004Prefix = "004系列/"

func normalizeSuanliaiModelName(modelName string) string {
	return strings.ToLower(strings.TrimSpace(modelName))
}

func isSuanliai004ModelName(modelName string) bool {
	return strings.HasPrefix(normalizeSuanliaiModelName(modelName), suanliai004Prefix)
}

func isSuanliaiOmniModelName(modelName string) bool {
	normalized := normalizeSuanliaiModelName(modelName)
	switch normalized {
	case "omni-video-1", "omni-video-1-fast", "omni-video-1-pro":
		return true
	default:
		return false
	}
}

func isSuanliaiModelName(modelName string) bool {
	return isSuanliai004ModelName(modelName) || isSuanliaiOmniModelName(modelName)
}

func isSuanliaiModelPair(originModelName, upstreamModelName string) bool {
	return isSuanliaiModelName(originModelName) || isSuanliaiModelName(upstreamModelName)
}

type suanliaiOmniRequest struct {
	Prompt      string `json:"prompt"`
	ImageURL    string `json:"image_url,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
}

func validateSuanliaiJSONRequest(c *gin.Context, modelName string) *dto.TaskError {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if isSuanliaiOmniModelName(modelName) {
		return validateSuanliaiOmniRequest(c, modelName)
	}
	if isSuanliai004ModelName(modelName) {
		if err := validateSuanliai004Request(req, modelName); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
	}
	return nil
}

func validateSuanliaiOmniRequest(c *gin.Context, _ string) *dto.TaskError {
	var req suanliaiOmniRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt must not exceed 5000 characters"), "invalid_request", http.StatusBadRequest)
	}
	if req.Duration != nil && *req.Duration != 3 && *req.Duration != 5 && *req.Duration != 8 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("duration must be one of: 3, 5, 8"), "invalid_request", http.StatusBadRequest)
	}
	if req.AspectRatio != "" {
		switch strings.TrimSpace(req.AspectRatio) {
		case "16:9", "9:16", "1:1":
		default:
			return service.TaskErrorWrapperLocal(fmt.Errorf("aspect_ratio must be 16:9, 9:16, or 1:1"), "invalid_request", http.StatusBadRequest)
		}
	}
	if strings.TrimSpace(req.ImageURL) != "" {
		if err := validateVideo2URL(req.ImageURL); err != nil {
			return service.TaskErrorWrapperLocal(fmt.Errorf("image_url: %w", err), "invalid_request", http.StatusBadRequest)
		}
	}
	return nil
}

func validateSuanliai004Request(req relaycommon.TaskSubmitReq, modelName string) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 2000 {
		return fmt.Errorf("prompt must not exceed 2000 characters")
	}

	seconds := 0
	if rawSeconds := strings.TrimSpace(req.Seconds); rawSeconds != "" {
		parsed, err := strconv.Atoi(rawSeconds)
		if err != nil {
			return fmt.Errorf("seconds must be an integer")
		}
		seconds = parsed
	}
	if seconds == 0 {
		seconds = req.Duration
	}
	modelKey := normalizeSuanliaiModelName(modelName)
	switch {
	case strings.Contains(modelKey, "sd2.5"):
		if seconds > 0 && (seconds < 4 || seconds > 30) {
			return fmt.Errorf("seconds must be between 4 and 30")
		}
	case strings.Contains(modelKey, "minimax") || strings.Contains(modelKey, "sd2.0"):
		if seconds > 0 && (seconds < 5 || seconds > 15) {
			return fmt.Errorf("seconds must be between 5 and 15")
		}
	case strings.Contains(modelKey, "video-editing"):
		if seconds > 30 {
			return fmt.Errorf("seconds must not exceed 30")
		}
	}

	if req.Ratio != "" {
		switch strings.TrimSpace(req.Ratio) {
		case "16:9", "9:16", "1:1", "4:3", "3:4":
		default:
			return fmt.Errorf("ratio must be 16:9, 9:16, 1:1, 4:3, or 3:4")
		}
	}
	if req.Resolution != "" {
		switch strings.ToLower(strings.TrimSpace(req.Resolution)) {
		case "480p", "720p", "768p", "2k":
		default:
			return fmt.Errorf("resolution must be 480p, 720p, 768p, or 2k")
		}
	}

	images, videos, audios := suanliaiReferences(req)
	for kind, values := range map[string][]string{
		"image": images,
		"video": videos,
		"audio": audios,
	} {
		for _, value := range values {
			if err := validateVideo2URL(value); err != nil {
				return fmt.Errorf("%s reference: %w", kind, err)
			}
		}
	}

	maxImages, maxVideos, maxAudios := 30, 10, 10
	switch {
	case strings.Contains(modelKey, "8图3音频"):
		maxImages, maxVideos, maxAudios = 8, 0, 3
	case strings.Contains(modelKey, "原生过人脸9图"):
		maxImages, maxVideos, maxAudios = 9, 0, 0
	case strings.Contains(modelKey, "30图4-30秒"):
		maxImages, maxVideos, maxAudios = 30, 0, 0
	case strings.Contains(modelKey, "10-10-10"):
		maxImages, maxVideos, maxAudios = 10, 10, 10
	case strings.Contains(modelKey, "30-10-10"):
		maxImages, maxVideos, maxAudios = 30, 10, 10
	case strings.Contains(modelKey, "video-editing"):
		maxImages, maxVideos, maxAudios = 1, 1, 0
	case strings.Contains(modelKey, "grok"):
		maxImages, maxVideos, maxAudios = 7, 0, 0
	}
	if len(images) > maxImages {
		return fmt.Errorf("image references must not exceed %d", maxImages)
	}
	if len(videos) > maxVideos {
		return fmt.Errorf("video references must not exceed %d", maxVideos)
	}
	if len(audios) > maxAudios {
		return fmt.Errorf("audio references must not exceed %d", maxAudios)
	}
	if strings.Contains(modelKey, "video-editing") && (len(images) != 1 || len(videos) != 1) {
		return fmt.Errorf("video-editing requires exactly one video and one image")
	}
	return nil
}

func suanliaiReferences(req relaycommon.TaskSubmitReq) (images, videos, audios []string) {
	images = append(images, req.Image, req.ImageURL)
	images = append(images, req.Images...)
	images = append(images, req.ImageURLs...)
	images = append(images, req.ReferenceImages...)
	videos = append(videos, req.VideoURL)
	videos = append(videos, req.Videos...)
	for _, reference := range req.VideoReference {
		videos = append(videos, reference.URL)
	}
	videos = append(videos, req.ReferenceVideos...)
	audios = append(audios, req.AudioURL)
	audios = append(audios, req.Audios...)
	for _, reference := range req.AudioReference {
		audios = append(audios, reference.URL)
	}
	audios = append(audios, req.ReferenceAudios...)
	return compactSuanliaiReferences(images), compactSuanliaiReferences(videos), compactSuanliaiReferences(audios)
}

func compactSuanliaiReferences(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}
