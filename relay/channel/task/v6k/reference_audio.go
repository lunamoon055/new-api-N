package v6k

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	audioUploadContextKey       = "v6k_audio_uploads"
	audioReferenceMaxBytes      = 15 << 20
	audioReferenceMaxTotalBytes = 60 << 20
	audioReferenceTimeout       = 30 * time.Second
	audioReferenceUserAgent     = "New-API-Reference-Audio/1.0"
)

type referenceAudioUpload struct {
	data        []byte
	contentType string
	filename    string
}

// Only the exact model involved in the reported failure uses this compatibility
// path. Do not change the transport of other, already integrated video models.
func prepareReferenceAudioUploads(c *gin.Context, modelName string, payload map[string]any, proxy string) error {
	c.Set(audioUploadContextKey, []referenceAudioUpload(nil))
	if modelName != "TTTseadanco2.5/720" {
		return nil
	}
	refs, _ := payload["audios"].([]any)
	if len(refs) == 0 {
		return nil
	}
	for _, ref := range refs {
		if object, ok := ref.(map[string]any); ok && object["duration_seconds"] != nil {
			// V6K documents no way to attach a duration hint to a binary part.
			// Preserve the entire URL list and its order instead of losing hints
			// or mixing URL/file references in a different order.
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), audioReferenceTimeout)
	defer cancel()
	uploads := make([]referenceAudioUpload, 0, len(refs))
	total := 0
	for index, ref := range refs {
		upload, err := fetchReferenceAudio(ctx, referenceURL(ref), proxy, min(audioReferenceMaxBytes, audioReferenceMaxTotalBytes-total))
		if err != nil {
			return fmt.Errorf("第 %d 段参考音频读取失败：%w；请换用可公开读取的音频直链，或通过 multipart 的 audio 字段上传文件", index+1, err)
		}
		upload.filename = fmt.Sprintf("reference-audio-%d%s", index+1, audioExtension(upload.contentType))
		total += len(upload.data)
		uploads = append(uploads, upload)
	}
	c.Set(audioUploadContextKey, uploads)
	return nil
}

func referenceAudioUploads(c *gin.Context) []referenceAudioUpload {
	value, _ := c.Get(audioUploadContextKey)
	uploads, _ := value.([]referenceAudioUpload)
	return uploads
}

func fetchReferenceAudio(ctx context.Context, sourceURL, proxy string, maxBytes int) (referenceAudioUpload, error) {
	var result referenceAudioUpload
	if maxBytes <= 0 {
		return result, fmt.Errorf("音频总大小超过本站 60 MiB 中转上限")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return result, fmt.Errorf("音频链接格式无效")
	}
	req.Header.Set("User-Agent", audioReferenceUserAgent)
	req.Header.Set("Accept", "audio/*, application/octet-stream;q=0.8")
	// Reference hosts never receive the channel API key, caller cookies or
	// Authorization. The protected client pins DNS and checks every redirect.
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return result, fmt.Errorf("音频下载代理配置不可用")
	}
	resp, err := service.DoSSRFProtectedRequest(client, req)
	if err != nil {
		// Do not return err: transport errors can contain signed source URLs,
		// proxy credentials or private network addresses.
		if ctx.Err() != nil {
			return result, fmt.Errorf("音频下载超时或请求已取消")
		}
		return result, fmt.Errorf("音频链接无法安全访问或连接失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("音频链接返回 HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > int64(maxBytes) {
		return result, fmt.Errorf("音频文件过大，本站中转限制为单段 15 MiB、合计 60 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)+1))
	if err != nil {
		return result, fmt.Errorf("音频内容未能完整下载")
	}
	if len(data) > maxBytes {
		return result, fmt.Errorf("音频文件过大，本站中转限制为单段 15 MiB、合计 60 MiB")
	}
	if len(data) == 0 {
		return result, fmt.Errorf("音频文件为空")
	}
	contentType := referenceAudioContentType(data)
	if contentType == "" {
		return result, fmt.Errorf("链接未返回可识别的音频文件，可能返回了网页或拦截页面")
	}
	result.data, result.contentType = data, contentType
	return result, nil
}

func referenceAudioContentType(data []byte) string {
	detected, _, _ := mime.ParseMediaType(http.DetectContentType(data))
	if strings.HasPrefix(detected, "audio/") || detected == "application/ogg" {
		return detected
	}
	// DetectContentType does not recognize every AAC/FLAC/MP4 audio signature.
	// Check bytes rather than trusting a URL suffix or a server-supplied MIME.
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte("fLaC")):
		return "audio/flac"
	case len(data) >= 4 && data[0] == 0xff && data[1]&0xf6 == 0xf0:
		return "audio/aac"
	case len(data) >= 4 && data[0] == 0xff && data[1]&0xe0 == 0xe0 &&
		data[1]&0x18 != 0x08 && data[1]&0x06 != 0 &&
		data[2]&0xf0 != 0xf0 && data[2]&0x0c != 0x0c:
		return "audio/mpeg"
	case len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")):
		return "audio/mp4"
	case detected == "video/webm":
		return "audio/webm"
	}
	return ""
}

func audioExtension(contentType string) string {
	switch contentType {
	case "audio/mpeg":
		return ".mp3"
	case "audio/wave", "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/aac":
		return ".aac"
	case "audio/mp4":
		return ".m4a"
	case "audio/flac":
		return ".flac"
	case "audio/webm":
		return ".webm"
	case "application/ogg", "audio/ogg":
		return ".ogg"
	default:
		return ".audio"
	}
}
