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
type xjzVideoSpec struct {
	minDuration     int
	maxDuration     int
	durations       []int
	resolution      string
	resolutions     []string
	ratios          []string
	imageField      string
	videoField      string
	audioField      string
	minImages       int
	maxImages       int
	maxVideos       int
	maxAudios       int
	audioNeedsImage bool
	allowFrames     bool
}

var xjzWideRatios = []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}
var xjzStandardRatios = []string{"16:9", "9:16", "1:1"}

var xjzVideoSpecs = map[string]xjzVideoSpec{
	"特价dola2.5-900-720P-固定30s": {
		durations: []int{30}, resolution: "720p", ratios: xjzWideRatios,
		imageField: "images", minImages: 1, maxImages: 9,
	},
	"特价dola-sd2.0-900-720p": {
		durations: []int{5, 10, 15}, resolution: "720p", ratios: xjzWideRatios,
		imageField: "images", minImages: 1, maxImages: 1,
	},
	"TTP-grok": {
		minDuration: 6, maxDuration: 15, resolutions: []string{"480p", "720p"}, ratios: xjzStandardRatios,
		imageField: "images", maxImages: 7,
	},
	"gz稳定sd2.0-903-720P": {
		minDuration: 5, maxDuration: 15, resolution: "720p", ratios: xjzWideRatios,
		imageField: "images", audioField: "reference_audios", maxImages: 9, maxAudios: 3,
	},
	"Xminimex-h3-2K-933": {
		minDuration: 5, maxDuration: 15, resolution: "2k", ratios: xjzWideRatios,
		imageField: "images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 9, maxVideos: 3, maxAudios: 3,
	},
	"特价sd2.0-720p-933-过脸": {
		minDuration: 5, maxDuration: 15, resolution: "720p", ratios: xjzWideRatios,
		imageField: "images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 9, maxVideos: 3, maxAudios: 3,
	},
	"特价稳sd-2.5-30s-720P可10图": {
		durations: []int{30}, resolution: "720p", ratios: xjzWideRatios,
		imageField: "reference_images", maxImages: 10,
	},
	"特价稳sd2.0-933-720p": {
		// The text says 5-15 seconds while one catalog parameter says 4-15.
		// Use the narrower range until the provider resolves the discrepancy.
		minDuration: 5, maxDuration: 15, resolution: "720P", ratios: xjzWideRatios,
		imageField: "reference_images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 9, maxVideos: 3, maxAudios: 3,
	},
	"gz稳定sd2.5-720全参过脸": {
		minDuration: 4, maxDuration: 30, resolution: "720P", ratios: xjzWideRatios,
		imageField: "reference_images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 30, maxVideos: 10, maxAudios: 10,
	},
	"gz稳定sd2.5-480全参过脸": {
		minDuration: 4, maxDuration: 30, resolution: "480P", ratios: xjzWideRatios,
		imageField: "reference_images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 30, maxVideos: 10, maxAudios: 10,
	},
	"gz稳定sd2.5-1080全参过脸": {
		minDuration: 4, maxDuration: 30, resolution: "1080p", ratios: xjzWideRatios,
		imageField: "reference_images", videoField: "reference_videos", audioField: "reference_audios",
		maxImages: 30, maxVideos: 10, maxAudios: 10,
	},
	"特价sd2.0-933-过脸": {
		minDuration: 4, maxDuration: 15, resolution: "720p", ratios: []string{"16:9", "9:16", "1:1", "4:3"},
		imageField: "image_urls", videoField: "video_urls", audioField: "audio_urls",
		maxImages: 9, maxVideos: 3, maxAudios: 3, audioNeedsImage: true,
	},
	"MiniMaxH3": {
		minDuration: 4, maxDuration: 15, resolution: "2K", ratios: xjzStandardRatios,
		imageField: "image_urls", audioField: "audio_reference",
		maxImages: 9, maxAudios: 3, audioNeedsImage: true, allowFrames: true,
	},
}

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
	imageKeys := []string{spec.imageField}
	audioKeys := []string{spec.audioField}
	if spec.allowFrames {
		imageKeys = append(imageKeys, "image_url")
		audioKeys = append(audioKeys, "audio_url")
	}
	req.Images, _ = xjzCollectURLs(normalized, imageKeys)
	req.Videos, _ = xjzCollectURLs(normalized, []string{spec.videoField})
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
	if !xjzContains(spec.ratios, ratio) {
		return nil, 0, 0, 0, 0, fmt.Errorf("ratio %q is unsupported for %s", ratio, modelName)
	}
	payload["ratio"] = ratio
	delete(payload, "aspect_ratio")

	resolution := strings.TrimSpace(xjzString(payload["resolution"]))
	if spec.resolution != "" {
		if resolution != "" && !strings.EqualFold(resolution, spec.resolution) {
			return nil, 0, 0, 0, 0, fmt.Errorf("resolution must be %s", spec.resolution)
		}
		resolution = spec.resolution
	} else if !xjzContains(spec.resolutions, resolution) {
		return nil, 0, 0, 0, 0, fmt.Errorf("resolution must be one of %s", strings.Join(spec.resolutions, ", "))
	}
	payload["resolution"] = resolution

	imageURLs, videoURLs, audioURLs, err := xjzReferences(payload, spec)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	imageCount := len(imageURLs)
	if spec.allowFrames {
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
	if imageCount < spec.minImages || imageCount > spec.maxImages {
		return nil, 0, 0, 0, 0, fmt.Errorf("image references must be between %d and %d", spec.minImages, spec.maxImages)
	}
	if len(videoURLs) > spec.maxVideos {
		return nil, 0, 0, 0, 0, fmt.Errorf("video references must not exceed %d", spec.maxVideos)
	}
	if len(audioURLs) > spec.maxAudios {
		return nil, 0, 0, 0, 0, fmt.Errorf("audio references must not exceed %d", spec.maxAudios)
	}
	if spec.audioNeedsImage && len(audioURLs) > 0 && imageCount+len(videoURLs) == 0 {
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
		if len(spec.durations) == 1 {
			return spec.durations[0], nil
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
	if len(spec.durations) > 0 {
		for _, allowed := range spec.durations {
			if duration == allowed {
				return duration, nil
			}
		}
		return 0, fmt.Errorf("duration %d is unsupported for this model", duration)
	}
	if duration < spec.minDuration || duration > spec.maxDuration {
		return 0, fmt.Errorf("duration must be between %d and %d", spec.minDuration, spec.maxDuration)
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
	if spec.imageField == "" && len(images) > 0 {
		return nil, nil, nil, fmt.Errorf("image references are unsupported")
	}
	if spec.videoField == "" && len(videos) > 0 {
		return nil, nil, nil, fmt.Errorf("video references are unsupported")
	}
	if spec.audioField == "" && len(audios) > 0 {
		return nil, nil, nil, fmt.Errorf("audio references are unsupported")
	}
	if spec.allowFrames {
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
			payload[spec.imageField] = images
		}
		if len(videos) > 0 {
			payload[spec.videoField] = videos
		}
		if len(audios) > 0 {
			payload[spec.audioField] = audios
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
