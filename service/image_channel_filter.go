package service

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Apply the same request constraints during initial selection, affinity and
// retry. A standard-only option must never be silently dropped by a native
// supplier adapter just because it has the same public model ID.
func SynchronousImageChannelFilter(c *gin.Context, publicModel string) model.ChannelFilter {
	fields := make(map[string]any)
	count := 0
	var largestFile int64
	jsonEdit := c != nil && c.Request != nil && strings.HasSuffix(c.Request.URL.Path, "/images/edits") && strings.HasPrefix(c.GetHeader("Content-Type"), "application/json")
	if c != nil && c.Request != nil {
		if mf := c.Request.MultipartForm; mf != nil {
			for key, v := range mf.Value {
				if len(v) > 0 {
					fields[key] = v[0]
				}
			}
			for key, files := range mf.File {
				if key == "image" || strings.HasPrefix(key, "image[") {
					count += len(files)
					for _, file := range files {
						largestFile = max(largestFile, file.Size)
					}
				} else if len(files) > 0 {
					fields[key] = true
				}
			}
		} else if strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			_ = common.UnmarshalBodyReusable(c, &fields)
			for _, key := range []string{"images", "image"} {
				if values, ok := fields[key].([]any); ok {
					count = len(values)
				}
			}
		}
	}
	return func(channel *model.Channel) bool {
		if channel == nil {
			return false
		}
		upstream, _, err := common.ResolveModelMapping(publicModel, channel.GetModelMapping())
		if err != nil || !common.IsSyncImageChannelCompatible(channel.GetBaseURL(), upstream) {
			return false
		}
		if common.IsOpenAIStandardImageChannel(channel.Type, channel.GetBaseURL(), upstream) {
			options := common.OpenAIImageStandardOptions(upstream)
			if q, ok := fields["quality"].(string); ok && q != "" {
				if q == "standard" {
					q = "medium"
				}
				if q == "hd" {
					q = "high"
				}
				if !common.StringsContains(options.Qualities, q) {
					return false
				}
			}
			if _, exists := fields["markup"]; exists && !options.SupportsMarkup {
				return false
			}
			return count <= options.MaxReferences && largestFile <= int64(options.MaxInputSizeMB)<<20
		}
		if common.IsBaseURLHost(channel.GetBaseURL(), "api.xjzapi.cc") && common.IsXJZNativeImageModel(upstream) {
			if jsonEdit {
				return false
			}
			if count > 20 || largestFile > 20<<20 {
				return false
			}
			if quality, ok := fields["quality"].(string); ok && quality != "" && quality != "standard" && quality != "hd" {
				return false
			}
			for _, key := range []string{"size", "background", "output_format", "output_compression", "mask", "markup", "safety_identifier"} {
				if value, exists := fields[key]; exists && value != nil && value != "" {
					return false
				}
			}
			if n, exists := fields["n"]; exists {
				v, _ := strconv.ParseFloat(strings.TrimSpace(toImageOptionString(n)), 64)
				if v != 1 {
					return false
				}
			}
			for _, pair := range []struct {
				keys   []string
				values []string
			}{
				{[]string{"ratio", "aspect_ratio"}, common.XJZNativeImageParameters(false).Ratios},
				{[]string{"resolution", "output_resolution"}, common.XJZNativeImageParameters(false).Resolutions},
			} {
				for _, key := range pair.keys {
					if v, ok := fields[key].(string); ok && v != "" && !common.StringsContains(pair.values, v) {
						return false
					}
				}
			}
		}
		return true
	}
}

func toImageOptionString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}
