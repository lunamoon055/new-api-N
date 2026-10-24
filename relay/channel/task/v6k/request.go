package v6k

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"sort"
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

var aliases = map[string]string{
	"duration": "seconds", "duration_seconds": "seconds", "durationSeconds": "seconds", "video_duration": "seconds",
	"aspect_ratio": "ratio", "referenceImages": "images", "reference_images": "images", "image_urls": "images", "reference_image_urls": "images",
	"referenceVideos": "videos", "reference_videos": "videos", "video_reference": "videos", "reference_video_urls": "videos",
	"referenceAudios": "audios", "reference_audios": "audios", "audio_reference": "audios", "reference_audio_urls": "audios",
	"image": "images", "image_url": "images", "video_url": "videos", "audio_url": "audios",
	"start_image_url": "input_reference", "first_frame_image": "input_reference", "start_frame": "input_reference",
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	invalid := func(err error) *dto.TaskError {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if info.Action == constant.TaskActionRemix || strings.Contains(info.RequestURLPath, "/images") {
		return invalid(fmt.Errorf("V6K channel supports video creation only"))
	}
	var source map[string]any
	if err := common.UnmarshalBodyReusable(c, &source); err != nil {
		return invalid(err)
	}
	payload, err := normalizeRequest(source)
	if err != nil {
		return invalid(err)
	}
	if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return invalid(err)
		}
		for field, files := range form.File {
			canonical, err := fileField(field)
			if err != nil {
				return invalid(err)
			}
			if canonical == "input_reference" && (len(files) > 1 || payload["input_reference"] != nil) {
				return invalid(fmt.Errorf("input_reference accepts exactly one image"))
			}
			if len(files) > 0 {
				c.Set("v6k_has_uploads", true)
			}
		}
	}
	seconds, _ := payload["seconds"].(int)
	if seconds == 0 {
		mode := billing_setting.GetVideoBillingMode(info.OriginModelName)
		if mode != billing_setting.VideoBillingModeFixed && mode != billing_setting.VideoBillingModeTieredRequest {
			return invalid(fmt.Errorf("seconds is required for duration-based billing; choose a supported duration or configure fixed billing"))
		}
	}
	request := relaycommon.TaskSubmitReq{Model: payload["model"].(string), Prompt: payload["prompt"].(string), Duration: seconds}
	request.Seconds = strconv.Itoa(seconds)
	request.Ratio, _ = payload["ratio"].(string)
	request.Resolution, _ = payload["resolution"].(string)
	request.Size, _ = payload["size"].(string)
	if request.Size != "" {
		width, height, err := sizeDimensions(request.Size)
		if err != nil {
			return invalid(err)
		}
		if request.Resolution == "" {
			request.Resolution = fmt.Sprintf("%dp", min(width, height))
		}
	}
	request.Images = referenceURLs(payload["images"])
	request.Videos = referenceURLs(payload["videos"])
	request.Audios = referenceURLs(payload["audios"])
	request.InputReference = referenceURL(payload["input_reference"])
	c.Set("v6k_request", payload)
	c.Set("task_request", request)
	info.Action = constant.TaskActionTextGenerate
	if len(request.Images)+len(request.Videos)+len(request.Audios) > 0 || request.InputReference != "" || c.GetBool("v6k_has_uploads") {
		info.Action = constant.TaskActionGenerate
	}
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	request, err := relaycommon.GetTaskRequest(c)
	if err == nil && request.Duration > 0 {
		return map[string]float64{"seconds": float64(request.Duration)}
	}
	return nil
}

func normalizeRequest(source map[string]any) (map[string]any, error) {
	if source == nil {
		return nil, fmt.Errorf("request must be a JSON object")
	}
	payload := map[string]any{}
	for field, value := range source {
		field = strings.TrimSuffix(field, "[]")
		if value == nil {
			continue
		}
		name := field
		if canonical, ok := aliases[field]; ok {
			name = canonical
		}
		var err error
		switch name {
		case "model", "prompt", "ratio", "resolution", "size":
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be a string", field)
			}
			value = strings.TrimSpace(text)
			if value == "" {
				continue
			}
		case "seconds":
			value, err = strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
			if err != nil || value.(int) < 1 || value.(int) > 3600 {
				return nil, fmt.Errorf("%s must be an integer between 1 and 3600", field)
			}
		case "images", "videos", "audios":
			value, err = normalizeReferences(value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", field, err)
			}
		case "input_reference":
			refs, e := normalizeReferences(value)
			if e != nil {
				return nil, e
			}
			if len(refs) != 1 {
				return nil, fmt.Errorf("input_reference accepts exactly one image")
			}
			value = refs[0]
		case "metadata":
			if text, ok := value.(string); ok {
				if err := common.UnmarshalJsonStr(text, &value); err != nil {
					return nil, err
				}
			}
			metadata, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("metadata must be an object")
			}
			for key := range metadata {
				lower := strings.ToLower(key)
				if _, ok := aliases[key]; ok || lower == "model" || lower == "prompt" || lower == "seconds" || lower == "resolution" || lower == "ratio" || lower == "size" || lower == "durationseconds" || strings.HasPrefix(lower, "duration") || strings.HasPrefix(lower, "video_duration") {
					return nil, fmt.Errorf("metadata cannot override %s", key)
				}
			}
		default:
			// Do not silently drop false/zero values or undocumented parameters.
			return nil, fmt.Errorf("unsupported V6K video field %s; use documented parameters", field)
		}
		if old, exists := payload[name]; exists {
			before, _ := common.Marshal(old)
			after, _ := common.Marshal(value)
			if !bytes.Equal(before, after) {
				return nil, fmt.Errorf("conflicting values for %s", name)
			}
		}
		payload[name] = value
	}
	if payload["model"] == nil || payload["prompt"] == nil {
		return nil, fmt.Errorf("model and prompt are required")
	}
	if ratio, ok := payload["ratio"].(string); ok {
		parts := strings.Split(ratio, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid ratio")
		}
		for _, part := range parts {
			n, e := strconv.Atoi(part)
			if e != nil || n < 1 || n > 9999 {
				return nil, fmt.Errorf("invalid ratio")
			}
		}
	}
	if size, ok := payload["size"].(string); ok {
		width, height, err := sizeDimensions(size)
		if err != nil {
			return nil, err
		}
		if ratio, ok := payload["ratio"].(string); ok {
			parts := strings.Split(ratio, ":")
			x, _ := strconv.Atoi(parts[0])
			y, _ := strconv.Atoi(parts[1])
			if width*y != height*x {
				return nil, fmt.Errorf("size conflicts with ratio")
			}
		}
		if resolution, ok := payload["resolution"].(string); ok {
			normalized := billing_setting.NormalizeVideoResolution(resolution)
			inferred := billing_setting.NormalizeVideoResolution(fmt.Sprintf("%dp", min(width, height)))
			if normalized == "" || normalized != inferred {
				return nil, fmt.Errorf("size conflicts with resolution")
			}
		}
	}
	return payload, nil
}

func sizeDimensions(size string) (int, int, error) {
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("size must contain pixel dimensions, e.g. 1280x720")
	}
	w, e1 := strconv.Atoi(parts[0])
	h, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || w < 1 || h < 1 || w > 65535 || h > 65535 {
		return 0, 0, fmt.Errorf("invalid size dimensions")
	}
	return w, h, nil
}

func normalizeReferences(value any) ([]any, error) {
	if text, ok := value.(string); ok && strings.HasPrefix(strings.TrimSpace(text), "[") {
		if err := common.UnmarshalJsonStr(text, &value); err != nil {
			return nil, err
		}
	}
	var refs []any
	switch typed := value.(type) {
	case []any:
		refs = typed
	case []string:
		for _, item := range typed {
			refs = append(refs, item)
		}
	default:
		refs = []any{value}
	}
	for _, ref := range refs {
		raw := referenceURL(ref)
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return nil, fmt.Errorf("references require public HTTP(S) URLs; upload local files with multipart")
		}
		host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
		ip := net.ParseIP(host)
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || (ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified())) {
			return nil, fmt.Errorf("references must be public URLs")
		}
		if object, ok := ref.(map[string]any); ok {
			for key := range object {
				if key != "url" && key != "duration_seconds" {
					return nil, fmt.Errorf("unsupported reference field %s", key)
				}
			}
			if duration, present := object["duration_seconds"]; present {
				number, err := strconv.ParseFloat(fmt.Sprint(duration), 64)
				if err != nil || number <= 0 || number > 3600 || math.IsNaN(number) || math.IsInf(number, 0) {
					return nil, fmt.Errorf("reference duration_seconds must be positive and at most 3600")
				}
			}
		}
	}
	return refs, nil
}

func referenceURL(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		result, _ := typed["url"].(string)
		return strings.TrimSpace(result)
	}
	return ""
}

func referenceURLs(value any) []string {
	refs, _ := value.([]any)
	urls := make([]string, 0, len(refs))
	for _, ref := range refs {
		urls = append(urls, referenceURL(ref))
	}
	return urls
}

func fileField(field string) (string, error) {
	field = strings.TrimSuffix(field, "[]")
	if canonical, ok := aliases[field]; ok {
		field = canonical
	}
	switch field {
	case "images":
		return "images", nil
	case "videos", "video":
		return "video", nil
	case "audios", "audio":
		return "audio", nil
	case "input_reference":
		return field, nil
	}
	return "", fmt.Errorf("unsupported V6K file field %s", field)
}

func buildMultipart(c *gin.Context, payload map[string]any) (io.Reader, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	fields := make([]string, 0, len(payload))
	for key := range payload {
		fields = append(fields, key)
	}
	sort.Strings(fields)
	for _, key := range fields {
		value := payload[key]
		text, ok := value.(string)
		if !ok {
			encoded, err := common.Marshal(value)
			if err != nil {
				return nil, err
			}
			text = string(encoded)
		}
		if err := writer.WriteField(key, text); err != nil {
			return nil, err
		}
	}
	for name, files := range form.File {
		field, err := fileField(name)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": field, "filename": filepath.Base(file.Filename)}))
			contentType := file.Header.Get("Content-Type")
			if _, _, err := mime.ParseMediaType(contentType); err != nil {
				contentType = "application/octet-stream"
			}
			header.Set("Content-Type", contentType)
			part, err := writer.CreatePart(header)
			if err != nil {
				return nil, err
			}
			reader, err := file.Open()
			if err != nil {
				return nil, err
			}
			_, copyErr := io.Copy(part, reader)
			reader.Close()
			if copyErr != nil {
				return nil, copyErr
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	c.Set("v6k_content_type", writer.FormDataContentType())
	return bytes.NewReader(buffer.Bytes()), nil
}
