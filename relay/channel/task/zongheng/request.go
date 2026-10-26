package zongheng

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
)

const requestContextKey = "zongheng_video_request"

type validatedRequest struct {
	Request videoRequest
	Model   *service.ZonghengVideoModel
}

func (a *TaskAdaptor) requestIdentity(modelName string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(a.baseURL+"\n"+a.apiKey+"\n"+modelName)))
}

type videoRequest struct {
	Model          string   `json:"model"`
	Prompt         string   `json:"prompt"`
	Duration       *int     `json:"duration,omitempty"`
	Ratio          string   `json:"ratio,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	Images         []string `json:"images,omitempty"`
	Videos         []string `json:"reference_videos,omitempty"`
	Audios         []string `json:"reference_audios,omitempty"`
	StartFrame     string   `json:"start_frame,omitempty"`
	EndFrame       string   `json:"end_frame,omitempty"`
	GenerateAudio  *bool    `json:"generate_audio,omitempty"`
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	invalid := func(err error) *dto.TaskError {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if info.Action == constant.TaskActionRemix || strings.Contains(info.RequestURLPath, "/images/") {
		return invalid(fmt.Errorf("Zongheng channel only supports video creation"))
	}
	var source map[string]any
	if err := common.UnmarshalBodyReusable(c, &source); err != nil {
		return invalid(err)
	}
	req, err := normalizeRequest(source)
	if err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(req.Model) == "" || strings.TrimSpace(req.Prompt) == "" {
		return invalid(fmt.Errorf("model and prompt are required"))
	}
	modelName := req.Model
	if info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	var selected *service.ZonghengVideoModel
	cacheKey := "zongheng_validated:" + a.requestIdentity(modelName)
	if value, ok := c.Get(cacheKey); ok {
		cached := value.(validatedRequest)
		req, selected = cached.Request, cached.Model
	} else {
		models, err := service.GetZonghengVideoModels(c.Request.Context(), a.baseURL, a.apiKey, a.proxy)
		if err != nil {
			return service.TaskErrorWrapperLocal(err, "upstream_capabilities_unavailable", http.StatusBadGateway)
		}
		for i := range models {
			if models[i].ID == modelName {
				selected = &models[i]
				break
			}
		}
	}
	if selected == nil {
		return invalid(fmt.Errorf("video model %q is not authorized or available on Zongheng", modelName))
	}
	if err := validateCapabilities(&req, selected); err != nil {
		return invalid(err)
	}
	c.Set(cacheKey, validatedRequest{Request: req, Model: selected})
	mode := billing_setting.GetVideoBillingMode(info.OriginModelName)
	if req.Duration == nil && mode != billing_setting.VideoBillingModeFixed && mode != billing_setting.VideoBillingModeTieredRequest {
		return invalid(fmt.Errorf("duration is required for seconds-based billing; configure fixed billing for models without duration options"))
	}
	c.Set(requestContextKey, req)
	c.Set("zongheng_video_capabilities", selected)
	info.Action = constant.TaskActionTextGenerate
	if len(req.Images)+len(req.Videos)+len(req.Audios) > 0 || req.StartFrame != "" || req.EndFrame != "" || strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		info.Action = constant.TaskActionGenerate
	}
	taskRequest := relaycommon.TaskSubmitReq{Model: req.Model, Prompt: req.Prompt, Ratio: req.Ratio, Resolution: req.Resolution, Images: req.Images, Videos: req.Videos, Audios: req.Audios, StartImageURL: req.StartFrame, EndImageURL: req.EndFrame}
	if req.Duration != nil {
		taskRequest.Duration = *req.Duration
		taskRequest.Seconds = strconv.Itoa(*req.Duration)
	}
	c.Set("task_request", taskRequest)
	return nil
}

func normalizeRequest(source map[string]any) (videoRequest, error) {
	var req videoRequest
	if source == nil {
		return req, fmt.Errorf("request must be a JSON object")
	}
	for canonical, aliases := range map[string][]string{
		"duration": {"seconds"}, "ratio": {"aspect_ratio"},
		"start_frame": {"start_image_url", "first_frame_image"}, "end_frame": {"end_image_url", "last_frame_image"},
	} {
		for _, alias := range aliases {
			value, exists := source[alias]
			if !exists || value == nil || value == "" {
				continue
			}
			if prior, ok := source[canonical]; ok && fmt.Sprint(prior) != fmt.Sprint(value) {
				return req, fmt.Errorf("%s conflicts with %s", canonical, alias)
			}
			source[canonical] = value
		}
	}
	if size, ok := source["size"]; ok && size != nil && size != "" {
		return req, fmt.Errorf("use ratio and resolution instead of size")
	}
	if raw, ok := source["duration"]; ok && raw != nil {
		duration, err := strconv.Atoi(fmt.Sprint(raw))
		if err != nil || duration <= 0 {
			return req, fmt.Errorf("duration must be a positive integer in seconds")
		}
		source["duration"] = duration
	}
	for canonical, aliases := range map[string][]string{
		"images":           {"images", "image", "image_url", "image_urls", "input_reference", "reference_images", "referenceImages"},
		"reference_videos": {"reference_videos", "videos", "video_url", "video_reference", "referenceVideos"},
		"reference_audios": {"reference_audios", "audios", "audio_url", "audio_reference", "referenceAudios"},
	} {
		var urls []string
		for _, alias := range aliases {
			value, exists := source[alias]
			if !exists || value == nil || value == "" {
				continue
			}
			data, err := common.Marshal(value)
			if err != nil {
				return req, err
			}
			var refs []relaycommon.TaskReference
			if err := common.Unmarshal(data, &refs); err != nil {
				var ref relaycommon.TaskReference
				if err := common.Unmarshal(data, &ref); err != nil {
					return req, fmt.Errorf("%s must contain reference URLs", alias)
				}
				refs = []relaycommon.TaskReference{ref}
			}
			for _, ref := range refs {
				if err := validateReference(ref.URL); err != nil {
					return req, fmt.Errorf("%s: %w", alias, err)
				}
				urls = append(urls, ref.URL)
			}
		}
		if len(urls) > 0 {
			source[canonical] = urls
		}
	}
	data, err := common.Marshal(source)
	if err != nil {
		return req, err
	}
	if err := common.Unmarshal(data, &req); err != nil {
		return req, err
	}
	for _, frame := range []string{req.StartFrame, req.EndFrame} {
		if frame != "" {
			if err := validateReference(frame); err != nil {
				return req, err
			}
		}
	}
	if req.EndFrame != "" && req.StartFrame == "" {
		return req, fmt.Errorf("end_frame requires start_frame")
	}
	if len(req.Images) > 0 && (req.StartFrame != "" || req.EndFrame != "") {
		return req, fmt.Errorf("choose either images or start/end frames")
	}
	return req, nil
}

func validateReference(value string) error {
	if strings.HasPrefix(value, "data:") {
		return nil
	} // Uploaded through /v1/media before submission.
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("references must be public HTTPS URLs or uploaded files")
	}
	return nil
}

func validateCapabilities(req *videoRequest, selected *service.ZonghengVideoModel) error {
	caps := selected.Capabilities
	ratios := caps.Ratios
	if len(ratios) == 0 {
		ratios = common.DefaultVideoAspectRatios()
	}
	if req.Duration == nil && len(caps.Durations) > 0 {
		req.Duration = caps.Defaults.Duration
	}
	if req.Ratio == "" {
		req.Ratio = caps.Defaults.Ratio
		if req.Ratio == "" {
			req.Ratio = ratios[0]
		}
	}
	if req.Resolution == "" && len(caps.Resolutions) > 0 {
		req.Resolution = caps.Defaults.Resolution
	}
	if req.Duration != nil {
		found := false
		for _, allowed := range caps.Durations {
			if allowed == *req.Duration {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("duration is not supported by this model")
		}
	}
	for _, field := range []struct {
		name, value string
		options     []string
	}{{"ratio", req.Ratio, ratios}, {"resolution", req.Resolution, caps.Resolutions}, {"quality", req.Quality, caps.Qualities}} {
		if field.value == "" {
			continue
		}
		found := false
		for _, allowed := range field.options {
			if field.value == allowed {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%s %q is not supported by this model", field.name, field.value)
		}
	}
	images := len(req.Images)
	if req.StartFrame != "" {
		images++
	}
	if req.EndFrame != "" {
		images++
	}
	for _, refs := range []struct {
		name  string
		count int
		max   *int
	}{{"images", images, caps.MaxImages}, {"reference_videos", len(req.Videos), caps.MaxVideos}, {"reference_audios", len(req.Audios), caps.MaxAudios}} {
		if refs.max != nil && refs.count > *refs.max {
			return fmt.Errorf("%s must not exceed %d", refs.name, *refs.max)
		}
	}
	return nil
}
