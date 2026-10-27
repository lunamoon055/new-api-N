package sora

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const suanliai005Model = "005系列/sd2.5(30-10-10/5-30秒/720P)"

// Exact mapped ID: do not apply new transport rules to other 005/004 models.
func isSuanliai005Request(info *relaycommon.RelayInfo) bool {
	return info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) == suanliai005Model
}

var suanliai005Sizes = map[string]string{
	"16:9": "1280x720", "9:16": "720x1280", "1:1": "720x720",
	"4:3": "960x720", "3:4": "720x960", "2:3": "720x1080",
	"3:2": "1080x720", "21:9": "1680x720",
}

func normalizeSuanliai005Payload(source map[string]any) (map[string]any, error) {
	payload := make(map[string]any, len(source))
	for key, value := range source {
		payload[key] = value
	}
	// Accept the provider's own documented pixel-size input as well as the
	// gateway's fixed ratio/resolution fields, checking conflicts first.
	if raw := payload["size"]; raw != nil {
		size, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("size must be a string")
		}
		var ratio string
		for candidate, dimensions := range suanliai005Sizes {
			if strings.EqualFold(strings.TrimSpace(size), dimensions) {
				ratio = candidate
				break
			}
		}
		if ratio == "" {
			return nil, fmt.Errorf("size must use a supported 720p aspect ratio")
		}
		for _, key := range []string{"ratio", "aspect_ratio"} {
			if value := payload[key]; value != nil && value != ratio {
				return nil, fmt.Errorf("size conflicts with %s", key)
			}
		}
		payload["ratio"] = ratio
		delete(payload, "size")
	}
	p := common.GetVideoIntegrationParameters(suanliai005Model)
	spec := xjzVideoSpec{
		Durations: p.Durations, Resolution: "720p", Ratios: p.Ratios,
		ImageField: "images", VideoField: "videos", AudioField: "audios",
		MaxImages: p.MaxImages, MaxVideos: p.MaxVideos, MaxAudios: p.MaxAudios,
	}
	normalized, _, _, _, _, err := normalizeVideoPayloadForSpec(payload, suanliai005Model, spec)
	return normalized, err
}

func validateSuanliai005Request(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	invalid := func(err error) *dto.TaskError {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return invalid(fmt.Errorf("this model requires application/json with public reference URLs"))
	}
	var source map[string]any
	if err := common.UnmarshalBodyReusable(c, &source); err != nil {
		return invalid(err)
	}
	payload, err := normalizeSuanliai005Payload(source)
	if err != nil {
		return invalid(err)
	}
	req := relaycommon.TaskSubmitReq{
		Model: xjzString(payload["model"]), Prompt: xjzString(payload["prompt"]),
		Duration: payload["duration"].(int), Ratio: xjzString(payload["ratio"]), Resolution: "720p",
	}
	req.Images, _ = xjzCollectURLs(payload, []string{"images"})
	req.Videos, _ = xjzCollectURLs(payload, []string{"videos"})
	req.Audios, _ = xjzCollectURLs(payload, []string{"audios"})
	info.Action = constant.TaskActionTextGenerate
	if len(req.Images)+len(req.Videos)+len(req.Audios) > 0 {
		info.Action = constant.TaskActionGenerate
	}
	c.Set("task_request", req)
	c.Set("suanliai_005_request", payload)
	return nil
}

func buildSuanliai005RequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	value, exists := c.Get("suanliai_005_request")
	if !exists {
		return nil, fmt.Errorf("validated Suanliai request is unavailable")
	}
	payload := make(map[string]any)
	for key, value := range value.(map[string]any) {
		payload[key] = value
	}
	payload["model"] = info.UpstreamModelName
	payload["seconds"] = payload["duration"]
	payload["size"] = suanliai005Sizes[xjzString(payload["ratio"])]
	delete(payload, "duration")
	delete(payload, "ratio")
	delete(payload, "resolution")
	encoded, err := common.Marshal(payload)
	return bytes.NewReader(encoded), err
}
