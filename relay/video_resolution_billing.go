package relay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func applyVideoResolutionTierPrice(c *gin.Context, info *relaycommon.RelayInfo, priceData *types.PriceData, mode string, prices map[string]float64) error {
	if priceData == nil || !billing_setting.IsVideoResolutionTierMode(mode) {
		return nil
	}

	resolution := resolveTaskBillingResolution(c, info)
	price, ok := lookupVideoResolutionPrice(prices, resolution)
	if !ok {
		return fmt.Errorf("video resolution price for %s is not configured", resolution)
	}

	groupRatio := priceData.GroupRatioInfo.GroupRatio
	priceData.ModelPrice = price
	priceData.UsePrice = true
	priceData.Quota = int(price * common.QuotaPerUnit * groupRatio)
	priceData.FreeModel = false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		priceData.FreeModel = price == 0 || groupRatio == 0
	}
	return nil
}

func lookupVideoResolutionPrice(prices map[string]float64, resolution string) (float64, bool) {
	normalizedResolution := billing_setting.NormalizeVideoResolution(resolution)
	if normalizedResolution == "" {
		return 0, false
	}
	for key, price := range prices {
		if billing_setting.NormalizeVideoResolution(key) == normalizedResolution {
			return price, true
		}
	}
	return 0, false
}

func resolveTaskBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) string {
	if req, err := relaycommon.GetTaskRequest(c); err == nil {
		if resolution := billing_setting.NormalizeVideoResolution(req.Resolution); resolution != "" {
			return resolution
		}
		if resolution := resolveTaskMetadataResolution(req.Metadata); resolution != "" {
			return resolution
		}
		if resolution := resolveTaskSizeResolution(req.Size); resolution != "" {
			return resolution
		}
	}

	if info != nil {
		if resolution := resolveTaskModelResolution(info.OriginModelName); resolution != "" {
			return resolution
		}
		if resolution := resolveTaskModelResolution(info.UpstreamModelName); resolution != "" {
			return resolution
		}
	}

	return "720p"
}

func resolveTaskMetadataResolution(metadata map[string]interface{}) string {
	for _, key := range []string{"resolution", "output_resolution"} {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		if resolution := billing_setting.NormalizeVideoResolution(fmt.Sprint(value)); resolution != "" {
			return resolution
		}
	}
	return ""
}

func resolveTaskSizeResolution(size string) string {
	normalized := strings.ToLower(strings.TrimSpace(size))
	normalized = strings.ReplaceAll(normalized, " ", "")
	if resolution := billing_setting.NormalizeVideoResolution(normalized); resolution != "" {
		return resolution
	}

	switch normalized {
	case "496x864", "864x496", "640x640":
		return "480p"
	case "720x1280", "1280x720", "960x960", "720x720":
		return "720p"
	case "1024x1024", "1024x1792", "1792x1024":
		return "1k"
	case "1080x1920", "1920x1080", "1440x1440":
		return "1080p"
	case "1080x2048", "2048x1080", "1440x2560", "2560x1440", "2048x2048":
		return "2k"
	case "2160x3840", "3840x2160", "2880x2880":
		return "4k"
	}

	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return ""
	}
	width, widthErr := strconv.Atoi(parts[0])
	height, heightErr := strconv.Atoi(parts[1])
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return ""
	}

	shortSide := width
	if height < shortSide {
		shortSide = height
	}
	switch {
	case shortSide <= 540:
		return "480p"
	case shortSide <= 1000:
		return "720p"
	case shortSide <= 1500:
		return "1080p"
	default:
		return "4k"
	}
}

func resolveTaskModelResolution(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if strings.HasSuffix(normalized, "minimax-h3") {
		return "2k"
	}
	for _, separator := range []string{"-", "_", "."} {
		index := strings.LastIndex(normalized, separator)
		if index < 0 || index == len(normalized)-1 {
			continue
		}
		if resolution := billing_setting.NormalizeVideoResolution(normalized[index+1:]); resolution != "" {
			return resolution
		}
	}
	return ""
}
