package sora

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// The public XJZ catalog uses these exact API IDs. In particular, the
// shorter labels shown in its sidebar are not valid model IDs. The catalog's
// structured parameters take precedence over its generic example request.
type xjzVideoSpec = common.XJZVideoSpec

var xjzVideoSpecs = common.XJZVideoSpecs

func isXJZVideoRequest(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	return common.IsXJZVideoModel(info.OriginModelName) ||
		(info.ChannelMeta != nil && common.IsXJZVideoModel(info.UpstreamModelName))
}

func xjzVideoModelName(info *relaycommon.RelayInfo) string {
	if info == nil {
		return ""
	}
	if info.ChannelMeta != nil && common.IsXJZVideoModel(info.UpstreamModelName) {
		return info.UpstreamModelName
	}
	return info.OriginModelName
}

func validateXJZVideoJSONRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("XJZ video requests require application/json"), "invalid_request", http.StatusBadRequest)
	}
	var payload map[string]any
	if err := common.UnmarshalBodyReusable(c, &payload); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_json", http.StatusBadRequest)
	}
	modelName := xjzVideoModelName(info)
	normalized, duration, imageCount, videoCount, audioCount, err := normalizeXJZVideoPayload(payload, modelName)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	req := relaycommon.TaskSubmitReq{
		Model:      strings.TrimSpace(xjzString(normalized["model"])),
		Prompt:     xjzString(normalized["prompt"]),
		Duration:   duration,
		Ratio:      xjzString(normalized["ratio"]),
		Resolution: xjzString(normalized["resolution"]),
	}
	spec := xjzVideoSpecs[modelName]
	imageKeys := []string{spec.ImageField}
	audioKeys := []string{spec.AudioField}
	if spec.AllowFrames {
		imageKeys = append(imageKeys, "image_url")
		audioKeys = append(audioKeys, "audio_url")
	}
	req.Images, _ = xjzCollectURLs(normalized, imageKeys)
	req.Videos, _ = xjzCollectURLs(normalized, []string{spec.VideoField})
	req.Audios, _ = xjzCollectURLs(normalized, audioKeys)
	req.StartImageURL = xjzString(normalized["start_image_url"])
	req.EndImageURL = xjzString(normalized["end_image_url"])
	info.Action = constant.TaskActionTextGenerate
	if imageCount+videoCount+audioCount > 0 {
		info.Action = constant.TaskActionGenerate
	}
	c.Set("task_request", req)
	return nil
}

func normalizeXJZVideoPayload(source map[string]any, modelName string) (map[string]any, int, int, int, int, error) {
	spec, ok := xjzVideoSpecs[modelName]
	if !ok {
		return nil, 0, 0, 0, 0, fmt.Errorf("unsupported XJZ video model %q", modelName)
	}
	if source == nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("request must be a JSON object")
	}
	payload := make(map[string]any, len(source)+3)
	for key, value := range source {
		payload[key] = value
	}
	if strings.TrimSpace(xjzString(payload["model"])) == "" {
		return nil, 0, 0, 0, 0, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(xjzString(payload["prompt"])) == "" {
		return nil, 0, 0, 0, 0, fmt.Errorf("prompt is required")
	}
	for _, key := range []string{"ratio", "aspect_ratio", "resolution", "size"} {
		if value, exists := payload[key]; exists && value != nil {
			if _, ok := value.(string); !ok {
				return nil, 0, 0, 0, 0, fmt.Errorf("%s must be a string", key)
			}
		}
	}
	if value, exists := payload["generate_audio"]; exists && value != nil {
		if _, ok := value.(bool); !ok {
			return nil, 0, 0, 0, 0, fmt.Errorf("generate_audio must be a boolean")
		}
	}
	if payload["size"] != nil && strings.TrimSpace(xjzString(payload["size"])) != "" {
		return nil, 0, 0, 0, 0, fmt.Errorf("size is unsupported; use resolution")
	}
	delete(payload, "size")

	duration, err := xjzDuration(payload, spec)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	payload["duration"] = duration
	delete(payload, "seconds")

	ratio := strings.TrimSpace(xjzString(payload["ratio"]))
	aliasRatio := strings.TrimSpace(xjzString(payload["aspect_ratio"]))
	if ratio != "" && aliasRatio != "" && ratio != aliasRatio {
		return nil, 0, 0, 0, 0, fmt.Errorf("ratio conflicts with aspect_ratio")
	}
	if ratio == "" {
		ratio = aliasRatio
	}
	if ratio == "" {
		ratio = "16:9"
	}
	if !xjzContains(spec.Ratios, ratio) {
		return nil, 0, 0, 0, 0, fmt.Errorf("ratio %q is unsupported for %s", ratio, modelName)
	}
	payload["ratio"] = ratio
	delete(payload, "aspect_ratio")

	resolution := strings.TrimSpace(xjzString(payload["resolution"]))
	if spec.Resolution != "" {
		if resolution != "" && !strings.EqualFold(resolution, spec.Resolution) {
			return nil, 0, 0, 0, 0, fmt.Errorf("resolution must be %s", spec.Resolution)
		}
		resolution = spec.Resolution
	} else if !xjzContains(spec.Resolutions, resolution) {
		return nil, 0, 0, 0, 0, fmt.Errorf("resolution must be one of %s", strings.Join(spec.Resolutions, ", "))
	}
	payload["resolution"] = resolution

	imageURLs, videoURLs, audioURLs, err := xjzReferences(payload, spec)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	imageCount := len(imageURLs)
	if spec.AllowFrames {
		for _, key := range []string{"start_image_url", "end_image_url"} {
			value := strings.TrimSpace(xjzString(payload[key]))
			if value == "" {
				delete(payload, key)
				continue
			}
			if err := xjzValidateURL(value); err != nil {
				return nil, 0, 0, 0, 0, fmt.Errorf("%s: %w", key, err)
			}
			imageCount++
		}
	} else if payload["start_image_url"] != nil || payload["end_image_url"] != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("start/end frames are unsupported for %s", modelName)
	}
	if imageCount < spec.MinImages || imageCount > spec.MaxImages {
		return nil, 0, 0, 0, 0, fmt.Errorf("image references must be between %d and %d", spec.MinImages, spec.MaxImages)
	}
	if len(videoURLs) > spec.MaxVideos {
		return nil, 0, 0, 0, 0, fmt.Errorf("video references must not exceed %d", spec.MaxVideos)
	}
	if len(audioURLs) > spec.MaxAudios {
		return nil, 0, 0, 0, 0, fmt.Errorf("audio references must not exceed %d", spec.MaxAudios)
	}
	if spec.AudioNeedsImage && len(audioURLs) > 0 && imageCount+len(videoURLs) == 0 {
		return nil, 0, 0, 0, 0, fmt.Errorf("audio references require an image or video reference")
	}
	return payload, duration, imageCount, len(videoURLs), len(audioURLs), nil
}

func xjzDuration(payload map[string]any, spec xjzVideoSpec) (int, error) {
	raw, exists := payload["duration"]
	if !exists || raw == nil {
		raw, exists = payload["seconds"]
	}
	if !exists || raw == nil {
		if len(spec.Durations) == 1 {
			return spec.Durations[0], nil
		}
		return 0, fmt.Errorf("duration is required")
	}
	duration, ok := xjzInteger(raw)
	if !ok {
		return 0, fmt.Errorf("duration must be an integer")
	}
	if other, exists := payload["seconds"]; exists && payload["duration"] != nil {
		otherDuration, otherOK := xjzInteger(other)
		if !otherOK || otherDuration != duration {
			return 0, fmt.Errorf("seconds conflicts with duration")
		}
	}
	if len(spec.Durations) > 0 {
		for _, allowed := range spec.Durations {
			if duration == allowed {
				return duration, nil
			}
		}
		return 0, fmt.Errorf("duration %d is unsupported for this model", duration)
	}
	if duration < spec.MinDuration || duration > spec.MaxDuration {
		return 0, fmt.Errorf("duration must be between %d and %d", spec.MinDuration, spec.MaxDuration)
	}
	return duration, nil
}

func xjzInteger(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
			return 0, false
		}
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func xjzReferences(payload map[string]any, spec xjzVideoSpec) ([]string, []string, []string, error) {
	imageKeys := []string{"images", "reference_images", "referenceImages", "image_urls", "image_url", "image", "input_reference", "extra_images"}
	videoKeys := []string{"reference_videos", "referenceVideos", "video_urls", "video_url", "video_reference", "videos", "extra_videos"}
	audioKeys := []string{"reference_audios", "referenceAudios", "audio_urls", "audio_url", "audio_reference", "audios", "extra_audios"}
	images, err := xjzCollectURLs(payload, imageKeys)
	if err != nil {
		return nil, nil, nil, err
	}
	videos, err := xjzCollectURLs(payload, videoKeys)
	if err != nil {
		return nil, nil, nil, err
	}
	audios, err := xjzCollectURLs(payload, audioKeys)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, keys := range [][]string{imageKeys, videoKeys, audioKeys} {
		for _, key := range keys {
			delete(payload, key)
		}
	}
	if spec.ImageField == "" && len(images) > 0 {
		return nil, nil, nil, fmt.Errorf("image references are unsupported")
	}
	if spec.VideoField == "" && len(videos) > 0 {
		return nil, nil, nil, fmt.Errorf("video references are unsupported")
	}
	if spec.AudioField == "" && len(audios) > 0 {
		return nil, nil, nil, fmt.Errorf("audio references are unsupported")
	}
	if spec.AllowFrames {
		if len(images) == 1 {
			payload["image_url"] = images[0]
		} else if len(images) > 1 {
			payload["image_urls"] = images
		}
		if len(audios) == 1 {
			payload["audio_url"] = audios[0]
		} else if len(audios) > 1 {
			payload["audio_reference"] = audios
		}
	} else {
		if len(images) > 0 {
			payload[spec.ImageField] = images
		}
		if len(videos) > 0 {
			payload[spec.VideoField] = videos
		}
		if len(audios) > 0 {
			payload[spec.AudioField] = audios
		}
	}
	return images, videos, audios, nil
}

func xjzCollectURLs(payload map[string]any, keys []string) ([]string, error) {
	var urls []string
	for _, key := range keys {
		value, exists := payload[key]
		if !exists || value == nil {
			continue
		}
		var entries []any
		switch typed := value.(type) {
		case string:
			entries = []any{typed}
		case []string:
			entries = make([]any, len(typed))
			for index, item := range typed {
				entries[index] = item
			}
		case []any:
			entries = typed
		default:
			return nil, fmt.Errorf("%s must be a URL or URL array", key)
		}
		for _, entry := range entries {
			var value string
			switch typed := entry.(type) {
			case string:
				value = strings.TrimSpace(typed)
			case map[string]any:
				value = strings.TrimSpace(xjzString(typed["url"]))
			default:
				return nil, fmt.Errorf("%s must contain URLs", key)
			}
			if value == "" {
				return nil, fmt.Errorf("%s contains an empty URL", key)
			}
			if err := xjzValidateURL(value); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			urls = append(urls, value)
		}
	}
	return urls, nil
}

func xjzValidateURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("reference must be a public HTTP or HTTPS URL")
	}
	return nil
}

func xjzString(value any) string {
	result, _ := value.(string)
	return result
}

func xjzContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
