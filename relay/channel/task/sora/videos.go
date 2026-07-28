package sora

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type videosRequest struct {
	Prompt         string   `json:"prompt"`
	Duration       *int     `json:"duration,omitempty"`
	Ratio          string   `json:"ratio,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	ReferenceImages []string `json:"referenceImages,omitempty"`
	ReferenceVideos []string `json:"referenceVideos,omitempty"`
	ReferenceAudios []string `json:"referenceAudios,omitempty"`
	FirstImage     string   `json:"first_image,omitempty"`
	LastImage      string   `json:"last_image,omitempty"`
}

func isVideosModelName(modelName string) bool {
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
	switch normalizedModelName {
	case "videos-standard", "videos-fast", "videos-mini":
		return true
	default:
		return strings.HasPrefix(normalizedModelName, "sd2")
	}
}

func validateVideosRequest(req videosRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if req.Duration != nil && (*req.Duration < 4 || *req.Duration > 15) {
		return fmt.Errorf("duration must be between 4 and 15")
	}
	if req.Ratio != "" && !isAllowedVideosRatio(req.Ratio) {
		return fmt.Errorf("ratio must be 16:9, 9:16, or 1:1")
	}
	if req.Resolution != "" && !isAllowedVideosResolution(req.Resolution) {
		return fmt.Errorf("resolution must be 720p or 480p")
	}
	if strings.TrimSpace(req.FirstImage) != "" || strings.TrimSpace(req.LastImage) != "" {
		return fmt.Errorf("first_image and last_image are not supported")
	}
	if countNonBlank(req.ReferenceImages) > 9 {
		return fmt.Errorf("image references must not exceed 9")
	}
	if countNonBlank(req.ReferenceVideos) > 3 {
		return fmt.Errorf("video references must not exceed 3")
	}
	if countNonBlank(req.ReferenceAudios) > 3 {
		return fmt.Errorf("audio references must not exceed 3")
	}
	for _, value := range req.ReferenceImages {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("referenceImages: %w", err)
		}
	}
	for _, value := range req.ReferenceVideos {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("referenceVideos: %w", err)
		}
	}
	for _, value := range req.ReferenceAudios {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("referenceAudios: %w", err)
		}
	}
	return nil
}

func validateVideosJSONRequest(c *gin.Context) *dto.TaskError {
	var req videosRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateVideosRequest(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}

func isAllowedVideosRatio(value string) bool {
	switch strings.TrimSpace(value) {
	case "16:9", "9:16", "1:1":
		return true
	default:
		return false
	}
}

func isAllowedVideosResolution(value string) bool {
	switch strings.TrimSpace(value) {
	case "720p", "480p":
		return true
	default:
		return false
	}
}
