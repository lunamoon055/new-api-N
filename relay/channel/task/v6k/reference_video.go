package v6k

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
)

const videoReferenceMaxBytes = 32 << 20
const videoReferenceTotalBytes = 96 << 20

var videoReferenceSlots = make(chan struct{}, 2)

func referenceVideoUploads(c *gin.Context) []referenceAudioUpload {
	value, _ := c.Get("v6k_video_uploads")
	uploads, _ := value.([]referenceAudioUpload)
	return uploads
}

func prepareReferenceVideoBilling(c *gin.Context, info *relaycommon.RelayInfo, payload map[string]any, uploadCount int) error {
	c.Set("v6k_video_uploads", []referenceAudioUpload(nil))
	c.Set("v6k_reference_video_seconds", float64(0))
	upstreamModel := info.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = info.OriginModelName
	}
	if !common.IsV6KSDVideoModel(upstreamModel) && !billing_setting.HasVideoReferencePrices(info.OriginModelName) {
		return nil
	}
	refs, _ := payload["videos"].([]any)
	if len(refs)+uploadCount == 0 {
		return nil
	}
	if billing_setting.GetVideoBillingMode(info.OriginModelName) != billing_setting.VideoBillingModeTieredSeconds {
		return fmt.Errorf("reference-video billing requires tiered_seconds pricing")
	}
	prices, configured := billing_setting.GetVideoReferencePrices(info.OriginModelName)
	resolution, _ := payload["resolution"].(string)
	if _, ok := prices[billing_setting.NormalizeVideoResolution(resolution)]; !configured || !ok {
		return fmt.Errorf("reference-video price for this resolution is not configured")
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	select {
	case videoReferenceSlots <- struct{}{}:
		defer func() { <-videoReferenceSlots }()
	case <-ctx.Done():
		return fmt.Errorf("reference-video verification is busy; retry later")
	}
	var uploads []referenceAudioUpload
	totalBytes, seconds := 0, float64(0)
	verify := func(data []byte) (string, error) {
		if len(data) > videoReferenceMaxBytes || totalBytes+len(data) > videoReferenceTotalBytes {
			return "", fmt.Errorf("reference videos exceed the site limit: 32 MiB each, 96 MiB total")
		}
		duration, contentType, err := common.VideoReferenceDuration(ctx, data)
		if err != nil {
			return "", err
		}
		totalBytes += len(data)
		seconds += duration
		return contentType, nil
	}
	for i, ref := range refs {
		data, err := downloadReferenceVideo(ctx, referenceURL(ref), info.ChannelSetting.Proxy, min(videoReferenceMaxBytes, videoReferenceTotalBytes-totalBytes))
		if err != nil {
			return fmt.Errorf("reference video %d: %w", i+1, err)
		}
		contentType, err := verify(data)
		if err != nil {
			return fmt.Errorf("reference video %d: %w", i+1, err)
		}
		ext := ".mp4"
		if contentType == "video/webm" {
			ext = ".webm"
		}
		uploads = append(uploads, referenceAudioUpload{data: data, contentType: contentType, filename: fmt.Sprintf("reference-video-%d%s", i+1, ext)})
	}
	if form := c.Request.MultipartForm; form != nil {
		for key, files := range form.File {
			field, _ := fileField(key)
			if field != "video" {
				continue
			}
			for _, file := range files {
				if file.Size > videoReferenceMaxBytes {
					return fmt.Errorf("reference video exceeds 32 MiB")
				}
				reader, err := file.Open()
				if err != nil {
					return fmt.Errorf("cannot read reference-video upload")
				}
				data, err := io.ReadAll(io.LimitReader(reader, videoReferenceMaxBytes+1))
				reader.Close()
				if err != nil {
					return fmt.Errorf("cannot read reference-video upload")
				}
				if _, err := verify(data); err != nil {
					return err
				}
			}
		}
	}
	c.Set("v6k_video_uploads", uploads)
	c.Set("v6k_reference_video_seconds", seconds)
	return nil
}

func downloadReferenceVideo(ctx context.Context, source, proxy string, limit int) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("reference-video total size limit exceeded")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid reference-video URL")
	}
	req.Header.Set("User-Agent", "New-API-Reference-Video/1.0")
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("reference-video proxy unavailable")
	}
	response, err := service.DoSSRFProtectedRequest(client, req)
	if err != nil {
		return nil, fmt.Errorf("reference-video URL cannot be accessed safely")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reference-video download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > int64(limit) {
		return nil, fmt.Errorf("reference-video size limit exceeded")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, fmt.Errorf("reference-video download incomplete or too large")
	}
	return data, nil
}
