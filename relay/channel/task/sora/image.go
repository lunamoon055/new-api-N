package sora

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const asyncImageReferenceMaxBytes = 20 * 1024 * 1024

type asyncImageCapability struct {
	resolutions       map[string]struct{}
	defaultResolution string
	aspectRatios      map[string]struct{}
	maxImages         int
}

type asyncImageRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	AspectRatio      string   `json:"aspect_ratio,omitempty"`
	Size             string   `json:"size,omitempty"`
	OutputResolution string   `json:"output_resolution,omitempty"`
	ImageURLs        []string `json:"image_urls,omitempty"`
	N                *int     `json:"n,omitempty"`
}

var asyncImageCapabilities = map[string]asyncImageCapability{
	"gpt-image-2": newAsyncImageCapability(
		[]string{"1K", "2K", "4K"}, "2K",
		[]string{"3:1", "21:9", "2:1", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16", "1:2", "1:3"},
		17,
	),
	// The supplied contract lists gpt-image-2.5 but omits its capability row.
	// Keep it compatible with the gpt-image-2 family and avoid a tighter,
	// undocumented reference-image cap.
	"gpt-image-2.5": newAsyncImageCapability(
		[]string{"1K", "2K", "4K"}, "2K",
		[]string{"3:1", "21:9", "2:1", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16", "1:2", "1:3"},
		17,
	),
	"nano-banana-pro": newAsyncImageCapability(
		[]string{"1K", "2K", "4K"}, "1K",
		[]string{"21:9", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16"},
		10,
	),
	"nano-banana2": newAsyncImageCapability(
		[]string{"1K", "2K", "4K"}, "1K",
		[]string{"21:9", "16:9", "3:2", "4:3", "5:4", "1:1", "4:5", "3:4", "2:3", "9:16"},
		14,
	),
	"seedream-5-0": newAsyncImageCapability(
		[]string{"2K", "3K"}, "2K",
		[]string{"16:9", "4:3", "1:1", "3:4", "9:16"},
		14,
	),
}

func newAsyncImageCapability(resolutions []string, defaultResolution string, aspectRatios []string, maxImages int) asyncImageCapability {
	resolutionSet := make(map[string]struct{}, len(resolutions))
	for _, resolution := range resolutions {
		resolutionSet[resolution] = struct{}{}
	}
	aspectRatioSet := make(map[string]struct{}, len(aspectRatios))
	for _, ratio := range aspectRatios {
		aspectRatioSet[ratio] = struct{}{}
	}
	return asyncImageCapability{
		resolutions:       resolutionSet,
		defaultResolution: defaultResolution,
		aspectRatios:      aspectRatioSet,
		maxImages:         maxImages,
	}
}

func validateAsyncImageRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return asyncImageRequestError("Content-Type must be application/json")
	}
	var req asyncImageRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	var raw map[string]any
	if err := common.UnmarshalBodyReusable(c, &raw); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	modelName := strings.TrimSpace(req.Model)
	capability, ok := asyncImageCapabilities[modelName]
	if !ok {
		return asyncImageRequestError("unsupported async image model %q", modelName)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if utf8.RuneCountInString(prompt) < 3 {
		return asyncImageRequestError("prompt must contain at least 3 characters")
	}
	allowedFields := map[string]struct{}{
		"model": {}, "prompt": {}, "aspect_ratio": {}, "output_resolution": {},
		"image_urls": {}, "size": {}, "n": {},
	}
	for field := range raw {
		if _, exists := allowedFields[field]; !exists {
			return asyncImageRequestError("field %s is not supported", field)
		}
	}
	if req.N != nil && *req.N != 1 {
		return asyncImageRequestError("n must be 1")
	}

	aspectRatio := strings.TrimSpace(req.AspectRatio)
	sizeAlias := strings.TrimSpace(req.Size)
	if aspectRatio == "" {
		aspectRatio = sizeAlias
	} else if sizeAlias != "" && sizeAlias != aspectRatio {
		return asyncImageRequestError("size conflicts with aspect_ratio")
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if _, ok := capability.aspectRatios[aspectRatio]; !ok {
		return asyncImageRequestError("unsupported aspect_ratio %q for model %s", aspectRatio, modelName)
	}

	outputResolution := normalizeAsyncImageResolution(req.OutputResolution)
	if outputResolution == "" {
		if strings.TrimSpace(req.OutputResolution) != "" {
			return asyncImageRequestError("unsupported output_resolution %q for model %s", req.OutputResolution, modelName)
		}
		outputResolution = capability.defaultResolution
	}
	if _, ok := capability.resolutions[outputResolution]; !ok {
		return asyncImageRequestError("unsupported output_resolution %q for model %s", req.OutputResolution, modelName)
	}
	if len(req.ImageURLs) > capability.maxImages {
		return asyncImageRequestError("model %s accepts at most %d reference images", modelName, capability.maxImages)
	}
	for index, imageURL := range req.ImageURLs {
		if err := validateAsyncImageReference(imageURL); err != nil {
			return asyncImageRequestError("invalid image_urls[%d]: %v", index, err)
		}
	}

	info.Action = constant.TaskActionImageGenerate
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:           prompt,
		Model:            modelName,
		ImageURLs:        append([]string(nil), req.ImageURLs...),
		Size:             aspectRatio,
		AspectRatio:      aspectRatio,
		OutputResolution: outputResolution,
	})
	return nil
}

func asyncImageRequestError(format string, args ...any) *dto.TaskError {
	return service.TaskErrorWrapperLocal(fmt.Errorf(format, args...), "invalid_request", http.StatusBadRequest)
}

func normalizeAsyncImageResolution(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "1K":
		return "1K"
	case "2K":
		return "2K"
	case "3K":
		return "3K"
	case "4K":
		return "4K"
	default:
		return ""
	}
}

func validateAsyncImageReference(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("reference URL must not be empty")
	}
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return validateAsyncImageDataURL(value)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("reference must be an absolute HTTP/HTTPS URL or a supported image data URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("reference URL scheme must be HTTP or HTTPS")
	}
	return nil
}

func validateAsyncImageDataURL(value string) error {
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return fmt.Errorf("invalid data URL")
	}
	metadata := strings.ToLower(value[len("data:"):comma])
	parts := strings.Split(metadata, ";")
	if len(parts) < 2 || parts[len(parts)-1] != "base64" {
		return fmt.Errorf("image data URL must use base64 encoding")
	}
	switch parts[0] {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return fmt.Errorf("image data URL must be PNG, JPEG, or WebP")
	}
	decoded, err := base64.StdEncoding.DecodeString(value[comma+1:])
	if err != nil {
		return fmt.Errorf("invalid base64 image data")
	}
	if len(decoded) >= asyncImageReferenceMaxBytes {
		return fmt.Errorf("decoded image must be smaller than 20 MiB")
	}
	return nil
}

func buildAsyncImageRequestBody(body []byte, originModelName, upstreamModelName string) ([]byte, error) {
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	capability, ok := asyncImageCapabilities[strings.TrimSpace(originModelName)]
	if !ok {
		return nil, fmt.Errorf("unsupported async image model %q", originModelName)
	}
	modelName := strings.TrimSpace(upstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(originModelName)
	}
	payload["model"] = modelName
	payload["prompt"] = strings.TrimSpace(stringFromAny(payload["prompt"]))

	aspectRatio := strings.TrimSpace(stringFromAny(payload["aspect_ratio"]))
	if aspectRatio == "" {
		aspectRatio = strings.TrimSpace(stringFromAny(payload["size"]))
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	payload["aspect_ratio"] = aspectRatio
	delete(payload, "size")

	outputResolution := normalizeAsyncImageResolution(stringFromAny(payload["output_resolution"]))
	if outputResolution == "" {
		outputResolution = capability.defaultResolution
	}
	payload["output_resolution"] = outputResolution
	delete(payload, "n")
	return common.Marshal(payload)
}
