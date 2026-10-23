package zongheng

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const maxMediaBytes = 25 * 1024 * 1024

func (a *TaskAdaptor) prepareReferences(c *gin.Context, req *videoRequest) error {
	// Cache uploads for this request and credential so transport retries send the
	// same URLs and body with the same Idempotency-Key.
	cacheKey := "zongheng_uploaded_media"
	cache := map[string]string{}
	if value, ok := c.Get(cacheKey); ok {
		cache, _ = value.(map[string]string)
	}
	if cache == nil {
		cache = map[string]string{}
	}
	c.Set(cacheKey, cache)
	uploadDataURL := func(value *string) error {
		if !strings.HasPrefix(*value, "data:") {
			return nil
		}
		identity := a.baseURL + a.apiKey + *value
		if uploaded := cache[identity]; uploaded != "" {
			*value = uploaded
			return nil
		}
		meta, encoded, ok := strings.Cut(*value, ",")
		if !ok || !strings.HasSuffix(meta, ";base64") {
			return fmt.Errorf("reference data URL must be base64 encoded")
		}
		if base64.StdEncoding.DecodedLen(len(encoded)) > maxMediaBytes {
			return fmt.Errorf("reference file exceeds 25MB")
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return err
		}
		contentType := strings.TrimSuffix(strings.TrimPrefix(meta, "data:"), ";base64")
		uploaded, err := a.uploadMedia(c, "reference", contentType, bytes.NewReader(data))
		if err != nil {
			return err
		}
		cache[identity], *value = uploaded, uploaded
		return nil
	}
	// Copy slices so uploaded URLs do not mutate the validated request on retries.
	req.Images = append([]string(nil), req.Images...)
	req.Videos = append([]string(nil), req.Videos...)
	req.Audios = append([]string(nil), req.Audios...)
	for _, refs := range [][]string{req.Images, req.Videos, req.Audios} {
		for i := range refs {
			if err := uploadDataURL(&refs[i]); err != nil {
				return err
			}
		}
	}
	if err := uploadDataURL(&req.StartFrame); err != nil {
		return err
	}
	if err := uploadDataURL(&req.EndFrame); err != nil {
		return err
	}
	if !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		return nil
	}
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return err
	}
	count := 0
	for field, files := range form.File {
		for i, file := range files {
			count++
			if count > 50 || file.Size > maxMediaBytes {
				return fmt.Errorf("upload accepts at most 50 files, 25MB each")
			}
			identity := fmt.Sprintf("%s%s:%s:%d", a.baseURL, a.apiKey, field, i)
			uploaded := cache[identity]
			if uploaded == "" {
				reader, err := file.Open()
				if err != nil {
					return err
				}
				uploaded, err = a.uploadMedia(c, file.Filename, file.Header.Get("Content-Type"), reader)
				reader.Close()
				if err != nil {
					return err
				}
				cache[identity] = uploaded
			}
			switch field {
			case "file", "images", "image", "input_reference":
				req.Images = append(req.Images, uploaded)
			case "reference_videos", "videos":
				req.Videos = append(req.Videos, uploaded)
			case "reference_audios", "audios":
				req.Audios = append(req.Audios, uploaded)
			case "start_frame", "start_image_url":
				req.StartFrame = uploaded
			case "end_frame", "end_image_url":
				req.EndFrame = uploaded
			default:
				return fmt.Errorf("unsupported reference file field %q", field)
			}
		}
	}
	if req.EndFrame != "" && req.StartFrame == "" {
		return fmt.Errorf("end_frame requires start_frame")
	}
	if len(req.Images) > 0 && (req.StartFrame != "" || req.EndFrame != "") {
		return fmt.Errorf("choose either images or start/end frames")
	}
	// Uploaded files count toward the capability snapshot used for this request.
	value, _ := c.Get("zongheng_video_capabilities")
	selected, ok := value.(*service.ZonghengVideoModel)
	if !ok {
		return fmt.Errorf("validated model capabilities are unavailable")
	}
	return validateCapabilities(req, selected)
}

func (a *TaskAdaptor) uploadMedia(c *gin.Context, filename, contentType string, reader io.Reader) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !(strings.HasPrefix(mediaType, "image/") || strings.HasPrefix(mediaType, "audio/") || strings.HasPrefix(mediaType, "video/")) {
		return "", fmt.Errorf("reference must be an image, audio or video file")
	}
	if filename == "reference" {
		if extensions, _ := mime.ExtensionsByType(mediaType); len(extensions) > 0 {
			filename += extensions[0]
		}
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, strings.NewReplacer("\"", "_", "\r", "_", "\n", "_").Replace(filename)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	n, err := io.Copy(part, io.LimitReader(reader, maxMediaBytes+1))
	if err != nil {
		return "", err
	}
	if n > maxMediaBytes {
		return "", fmt.Errorf("reference file exceeds 25MB")
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, a.baseURL+"/v1/media", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client, err := service.GetHttpClientWithProxy(a.proxy)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Zongheng media upload returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code    int  `json:"code"`
		Success bool `json:"success"`
		Data    []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := common.DecodeJson(io.LimitReader(resp.Body, common.MaxUpstreamResponseBodyBytes), &result); err != nil {
		return "", err
	}
	if result.Code != 0 || !result.Success || len(result.Data) != 1 {
		return "", fmt.Errorf("invalid Zongheng upload response")
	}
	if err := validateReference(result.Data[0].URL); err != nil || strings.HasPrefix(result.Data[0].URL, "data:") {
		return "", fmt.Errorf("invalid Zongheng uploaded media URL")
	}
	return result.Data[0].URL, nil
}
