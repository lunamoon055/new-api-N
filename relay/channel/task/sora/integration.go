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

// Validation uses the mapped upstream contract while the request/context keeps
// the caller's public model ID for routing and accounting.
func validateIntegrationVideoRequest(c *gin.Context, modelName string, allowAutomaticDuration bool) *dto.TaskError {
	var req videosRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	// Meaicc preserves an explicit zero (provider automatic duration) in the
	// original body. Skip only this local bound check; never rewrite it upstream.
	if allowAutomaticDuration && req.Duration != nil && *req.Duration == 0 {
		req.Duration = nil
	}
	if err := validateIntegrationVideoParameters(req, common.GetVideoIntegrationParameters(modelName)); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}

func validateIntegrationVideoParameters(req videosRequest, parameters *common.ModelIntegrationParameters) error {
	if parameters == nil {
		return fmt.Errorf("model integration parameters are unavailable")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if req.Duration != nil {
		allowed := false
		for _, duration := range parameters.Durations {
			if *req.Duration == duration {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("duration is unsupported for this model")
		}
	}
	if req.Ratio != "" && !integrationContains(parameters.Ratios, req.Ratio) {
		return fmt.Errorf("ratio is unsupported for this model")
	}
	if req.Resolution != "" && !integrationContains(parameters.Resolutions, req.Resolution) {
		return fmt.Errorf("resolution is unsupported for this model")
	}
	if req.FirstImage != "" || req.LastImage != "" {
		return fmt.Errorf("first_image and last_image are not supported; use referenceImages")
	}
	frames := countNonBlank([]string{req.StartImageURL, req.EndImageURL})
	if frames > 0 && !parameters.AllowFrames {
		return fmt.Errorf("start/end frames are unsupported for this model")
	}
	images := countNonBlank(req.ReferenceImages) + frames
	videos, audios := countNonBlank(req.ReferenceVideos), countNonBlank(req.ReferenceAudios)
	if images < parameters.MinImages || images > parameters.MaxImages {
		return fmt.Errorf("image references must be between %d and %d", parameters.MinImages, parameters.MaxImages)
	}
	if videos > parameters.MaxVideos {
		return fmt.Errorf("video references must not exceed %d", parameters.MaxVideos)
	}
	if audios > parameters.MaxAudios {
		return fmt.Errorf("audio references must not exceed %d", parameters.MaxAudios)
	}
	if parameters.AudioNeedsImage && audios > 0 && images+videos == 0 {
		return fmt.Errorf("audio references require an image or video reference")
	}
	values := append([]string{}, req.ReferenceImages...)
	values = append(values, req.ReferenceVideos...)
	values = append(values, req.ReferenceAudios...)
	values = append(values, req.StartImageURL, req.EndImageURL)
	for _, value := range values {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("reference URL: %w", err)
		}
	}
	return nil
}

func integrationContains(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}
