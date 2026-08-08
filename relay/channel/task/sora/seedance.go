/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

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

type seedance2Media struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type seedance2Input struct {
	Prompt string           `json:"prompt"`
	Media  []seedance2Media `json:"media,omitempty"`
}

type seedance2Parameters struct {
	Resolution string `json:"resolution,omitempty"`
	Ratio      string `json:"ratio,omitempty"`
	Duration   *int   `json:"duration,omitempty"`
}

type seedance2Request struct {
	Model      string              `json:"model"`
	Input      seedance2Input      `json:"input"`
	Parameters seedance2Parameters `json:"parameters"`
}

type seedance25Request struct {
	Model           string   `json:"model"`
	Prompt          string   `json:"prompt"`
	Duration        *int     `json:"duration,omitempty"`
	Ratio           string   `json:"ratio,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
	ReferenceImages []string `json:"referenceImages,omitempty"`
	ReferenceVideos []string `json:"referenceVideos,omitempty"`
	ReferenceAudios []string `json:"referenceAudios,omitempty"`
}

// isSeedance2ModelName identifies configured Seedance 2.x display aliases and
// mapped model names used by OpenAI-compatible upstream channels.
func isSeedance2ModelName(modelName string) bool {
	switch normalizeVideosModelName(modelName) {
	case "sd-2.0-933", "sd-2-c8", "seedance-2.0", "seedance-2.5":
		return true
	default:
		return false
	}
}

func isSeedance25ModelName(modelName string) bool {
	return normalizeVideosModelName(modelName) == "seedance-2.5"
}

func validateSeedance2JSONRequest(c *gin.Context) *dto.TaskError {
	var req videosRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if isSeedance25ModelName(req.Model) {
		if err := validateSeedance25Request(req); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		return nil
	}
	if err := validateSeedance2Request(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}

func validateSeedance25Request(req videosRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if req.Duration != nil && (*req.Duration < 4 || *req.Duration > 29) {
		return fmt.Errorf("duration must be between 4 and 29")
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

	imageCount := countNonBlank(req.ReferenceImages) +
		countNonBlank([]string{req.StartImageURL, req.EndImageURL})
	videoCount := countNonBlank(req.ReferenceVideos)
	audioCount := countNonBlank(req.ReferenceAudios)
	if imageCount > 30 {
		return fmt.Errorf("image references must not exceed 30")
	}
	if videoCount > 10 {
		return fmt.Errorf("video references must not exceed 10")
	}
	if audioCount > 10 {
		return fmt.Errorf("audio references must not exceed 10")
	}
	if imageCount+videoCount+audioCount > 50 {
		return fmt.Errorf("reference assets must not exceed 50")
	}

	for _, value := range append(
		append([]string{req.StartImageURL, req.EndImageURL}, req.ReferenceImages...),
		req.ReferenceVideos...,
	) {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("reference URL: %w", err)
		}
	}
	for _, value := range req.ReferenceAudios {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("reference URL: %w", err)
		}
	}
	return nil
}

func validateSeedance2Request(req videosRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if req.Duration != nil && (*req.Duration < 4 || *req.Duration > 15) {
		return fmt.Errorf("duration must be between 4 and 15")
	}
	if req.Ratio != "" && !isAllowedSeedance2Ratio(req.Ratio) {
		return fmt.Errorf("ratio must be 1:1, 16:9, 9:16, 4:3, or 3:4")
	}
	if req.Resolution != "" && strings.TrimSpace(req.Resolution) != "720p" {
		return fmt.Errorf("resolution must be 720p")
	}
	if strings.TrimSpace(req.FirstImage) != "" || strings.TrimSpace(req.LastImage) != "" {
		return fmt.Errorf("first_image and last_image are not supported")
	}

	imageCount := countNonBlank(req.ReferenceImages) +
		countNonBlank([]string{req.StartImageURL, req.EndImageURL})
	if imageCount > 9 {
		return fmt.Errorf("image references must not exceed 9")
	}
	if countNonBlank(req.ReferenceVideos) > 3 {
		return fmt.Errorf("video references must not exceed 3")
	}
	if countNonBlank(req.ReferenceAudios) > 3 {
		return fmt.Errorf("audio references must not exceed 3")
	}
	if imageCount+countNonBlank(req.ReferenceVideos)+countNonBlank(req.ReferenceAudios) > 15 {
		return fmt.Errorf("reference assets must not exceed 15")
	}

	for _, value := range append(
		append([]string{req.StartImageURL, req.EndImageURL}, req.ReferenceImages...),
		req.ReferenceVideos...,
	) {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("reference URL: %w", err)
		}
	}
	for _, value := range req.ReferenceAudios {
		if err := validateVideo2URL(value); err != nil {
			return fmt.Errorf("reference URL: %w", err)
		}
	}
	return nil
}

func isAllowedSeedance2Ratio(value string) bool {
	switch strings.TrimSpace(value) {
	case "1:1", "16:9", "9:16", "4:3", "3:4":
		return true
	default:
		return false
	}
}

func buildSeedance2RequestBody(body []byte, upstreamModel string) ([]byte, error) {
	var req videosRequest
	if err := common.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	request := seedance2Request{
		Model: strings.TrimSpace(upstreamModel),
		Input: seedance2Input{
			Prompt: req.Prompt,
		},
		Parameters: seedance2Parameters{
			Resolution: strings.TrimSpace(req.Resolution),
			Ratio:      strings.TrimSpace(req.Ratio),
		},
	}
	if request.Model == "" {
		request.Model = strings.TrimSpace(req.Model)
	}
	if req.Duration != nil && *req.Duration > 0 {
		request.Parameters.Duration = req.Duration
	}
	if request.Parameters.Resolution == "" {
		request.Parameters.Resolution = "720p"
	}

	appendMedia := func(mediaType, value string) {
		if value = strings.TrimSpace(value); value != "" {
			request.Input.Media = append(request.Input.Media, seedance2Media{
				Type: mediaType,
				URL:  value,
			})
		}
	}
	appendMedia("first_frame", req.StartImageURL)
	appendMedia("last_frame", req.EndImageURL)
	for _, value := range req.ReferenceImages {
		appendMedia("reference_image", value)
	}
	for _, value := range req.ReferenceVideos {
		appendMedia("reference_video", value)
	}
	for _, value := range req.ReferenceAudios {
		appendMedia("reference_voice", value)
	}

	return common.Marshal(request)
}

func buildSeedance25RequestBody(body []byte, upstreamModel string) ([]byte, error) {
	var req videosRequest
	if err := common.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	request := seedance25Request{
		Model:      strings.TrimSpace(upstreamModel),
		Prompt:     req.Prompt,
		Duration:   req.Duration,
		Ratio:      strings.TrimSpace(req.Ratio),
		Resolution: strings.TrimSpace(req.Resolution),
	}
	if request.Model == "" {
		request.Model = strings.TrimSpace(req.Model)
	}

	appendReference := func(target *[]string, value string) {
		if value = strings.TrimSpace(value); value != "" {
			*target = append(*target, value)
		}
	}
	// Older Creation Center sessions may still carry the internal start/end
	// image fields. The documented /v1/videos contract only accepts
	// referenceImages, so fold them into that array instead of forwarding
	// unsupported fields upstream.
	appendReference(&request.ReferenceImages, req.StartImageURL)
	for _, value := range req.ReferenceImages {
		appendReference(&request.ReferenceImages, value)
	}
	appendReference(&request.ReferenceImages, req.EndImageURL)
	for _, value := range req.ReferenceVideos {
		appendReference(&request.ReferenceVideos, value)
	}
	for _, value := range req.ReferenceAudios {
		appendReference(&request.ReferenceAudios, value)
	}

	return common.Marshal(request)
}
