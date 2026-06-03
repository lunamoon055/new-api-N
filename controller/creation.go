package controller

import (
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	creationModeChat  = "chat"
	creationModeImage = "image"
	creationModeVideo = "video"
)

var creationModeOrder = []string{
	creationModeChat,
	creationModeImage,
	creationModeVideo,
}

func GetCreationModels(c *gin.Context) {
	mode, valid := normalizeCreationMode(c.Query("mode"))
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid mode, must be chat, image or video",
		})
		return
	}

	pricing, _, _ := getPricingForRequest(c)
	common.ApiSuccess(c, buildCreationModelCatalog(pricing, model.GetVendors(), mode))
}

func normalizeCreationMode(mode string) (string, bool) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", creationModeChat, creationModeImage, creationModeVideo:
		return mode, true
	default:
		return "", false
	}
}

func buildCreationModelCatalog(pricing []model.Pricing, vendors []model.PricingVendor, requestedMode string) dto.CreationModelCatalog {
	modelsByMode := make(map[string][]dto.CreationModel, len(creationModeOrder))
	usedVendorIDs := make(map[int]struct{})

	for _, item := range pricing {
		mode, ok := getCreationModelMode(item.SupportedEndpointTypes)
		if !ok || (requestedMode != "" && mode != requestedMode) {
			continue
		}

		modelsByMode[mode] = append(modelsByMode[mode], dto.CreationModel{
			ID:                     item.ModelName,
			Description:            item.Description,
			Icon:                   item.Icon,
			Tags:                   splitCreationModelTags(item.Tags),
			VendorID:               item.VendorID,
			SupportedEndpointTypes: item.SupportedEndpointTypes,
		})
		if item.VendorID != 0 {
			usedVendorIDs[item.VendorID] = struct{}{}
		}
	}

	groups := make([]dto.CreationModelGroup, 0, len(creationModeOrder))
	for _, mode := range creationModeOrder {
		if requestedMode != "" && mode != requestedMode {
			continue
		}

		models := modelsByMode[mode]
		if models == nil {
			models = make([]dto.CreationModel, 0)
		}
		sort.Slice(models, func(i, j int) bool {
			return models[i].ID < models[j].ID
		})
		groups = append(groups, dto.CreationModelGroup{
			Mode:   mode,
			Models: models,
		})
	}

	catalogVendors := make([]dto.CreationVendor, 0, len(usedVendorIDs))
	for _, vendor := range vendors {
		if _, ok := usedVendorIDs[vendor.ID]; !ok {
			continue
		}
		catalogVendors = append(catalogVendors, dto.CreationVendor{
			ID:          vendor.ID,
			Name:        vendor.Name,
			Description: vendor.Description,
			Icon:        vendor.Icon,
		})
	}
	sort.Slice(catalogVendors, func(i, j int) bool {
		if catalogVendors[i].ID == catalogVendors[j].ID {
			return catalogVendors[i].Name < catalogVendors[j].Name
		}
		return catalogVendors[i].ID < catalogVendors[j].ID
	})

	return dto.CreationModelCatalog{
		Modes:   groups,
		Vendors: catalogVendors,
	}
}

func getCreationModelMode(endpoints []constant.EndpointType) (string, bool) {
	hasChat := false
	hasImage := false
	hasVideo := false

	for _, endpoint := range endpoints {
		switch endpoint {
		case constant.EndpointTypeImageGeneration:
			hasImage = true
		case constant.EndpointTypeOpenAIVideo:
			hasVideo = true
		case constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAIResponseCompact,
			constant.EndpointTypeAnthropic,
			constant.EndpointTypeGemini:
			hasChat = true
		}
	}

	switch {
	case hasImage:
		return creationModeImage, true
	case hasVideo:
		return creationModeVideo, true
	case hasChat:
		return creationModeChat, true
	default:
		return "", false
	}
}

func splitCreationModelTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	tags := make([]string, 0)
	seen := make(map[string]struct{})
	for _, tag := range strings.Split(raw, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}
