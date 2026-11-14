package relay

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func applyVideoReferenceBilling(c *gin.Context, info *relaycommon.RelayInfo) error {
	if info == nil || info.ChannelMeta == nil || info.ChannelType != constant.ChannelTypeV6K {
		return nil
	}
	mode, generationPrices, prices := billing_setting.GetVideoPricingSnapshot(info.OriginModelName)
	upstreamModel := info.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = info.OriginModelName
	}
	// An explicit reference-price configuration enables metering even when the
	// public ID is sent upstream unchanged or maps to another V6K video model.
	if !common.IsV6KSDVideoModel(upstreamModel) && len(prices) == 0 {
		return nil
	}
	if mode != billing_setting.VideoBillingModeTieredSeconds {
		return fmt.Errorf("reference-video billing requires resolution-tiered seconds pricing")
	}
	request, err := relaycommon.GetTaskRequest(c)
	if err != nil || request.Duration <= 0 {
		return fmt.Errorf("video generation duration is required")
	}
	resolution := billing_setting.NormalizeVideoResolution(request.Resolution)
	referenceSeconds := c.GetFloat64("v6k_reference_video_seconds")
	referencePrice, tierConfigured := prices[resolution]
	if referenceSeconds > 0 && !tierConfigured {
		return fmt.Errorf("reference-video price for %s / %s is not configured", info.OriginModelName, resolution)
	}
	generationPrice, configured := generationPrices[resolution]
	if !configured {
		return fmt.Errorf("generation price is not configured for %s", resolution)
	}
	info.PriceData.ModelPrice = generationPrice
	info.PriceData.VideoReferenceBilling = &types.VideoReferenceBilling{Resolution: resolution, GenerationSeconds: float64(request.Duration), ReferenceSeconds: referenceSeconds, GenerationPrice: info.PriceData.ModelPrice, ReferencePrice: referencePrice}
	return nil
}
