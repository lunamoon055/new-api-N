package controller

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	creationModeChat  = "chat"
	creationModeImage = "image"
	creationModeVideo = "video"

	creationModelCategoriesOptionKey = "CreationModelCategories"
)

var creationModeOrder = []string{
	creationModeChat,
	creationModeImage,
	creationModeVideo,
}

type creationModelMetadata struct {
	Mode        string
	Description string
	Tags        []string
}

var creationModelMetadataByName = map[string]creationModelMetadata{
	"gpt-5.3-codex": {
		Mode:        creationModeChat,
		Description: "面向代码生成、调试和复杂开发协作的高推理对话模型。",
		Tags:        []string{"chat", "code", "reasoning"},
	},
	"gpt-5.4": {
		Mode:        creationModeChat,
		Description: "综合能力更强，适合长文本写作、复杂问答和内容分析。",
		Tags:        []string{"chat", "advanced"},
	},
	"gpt-5.4-mini": {
		Mode:        creationModeChat,
		Description: "轻量快速，适合日常问答、文案草稿和低延迟创作。",
		Tags:        []string{"chat", "fast"},
	},
	"gpt-image2": {
		Mode:        creationModeImage,
		Description: "用于图片生成和视觉创作任务，适合海报、素材和参考图生成。",
		Tags:        []string{"image", "generation"},
	},
	"kling-v3": {
		Mode:        creationModeVideo,
		Description: "适合文本生成视频和动态镜头创作的异步视频模型。",
		Tags:        []string{"video", "async"},
	},
	"sora2": {
		Mode:        creationModeVideo,
		Description: "按 linksky 异步媒体接口接入的视频生成模型，适合短视频创作。",
		Tags:        []string{"video", "async"},
	},
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

func CreationRelayImage(c *gin.Context) {
	if newAPIError := setupCreationRelayContext(c, "creation-image"); newAPIError != nil {
		respondCreationRelayError(c, newAPIError)
		return
	}

	Relay(c, types.RelayFormatOpenAIImage)
}

func CreationRelayTask(c *gin.Context) {
	if newAPIError := setupCreationRelayContext(c, "creation-video"); newAPIError != nil {
		respondCreationRelayError(c, newAPIError)
		return
	}

	RelayTask(c)
}

func CreationRelayTaskFetch(c *gin.Context) {
	RelayTaskFetch(c)
}

func setupCreationRelayContext(c *gin.Context, tokenPrefix string) *types.NewAPIError {
	if c.GetBool("use_access_token") {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}

	userId := c.GetInt("id")
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	userCache.WriteContext(c)

	usingGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if usingGroup == "" {
		usingGroup = userCache.Group
		common.SetContextKey(c, constant.ContextKeyUsingGroup, usingGroup)
	}

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("%s-%s", tokenPrefix, usingGroup),
		Group:  usingGroup,
	}
	if err := middleware.SetupContextForToken(c, tempToken); err != nil {
		return types.NewError(err, types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}
	return nil
}

func respondCreationRelayError(c *gin.Context, newAPIError *types.NewAPIError) {
	c.JSON(newAPIError.StatusCode, gin.H{
		"error": newAPIError.ToOpenAIError(),
	})
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
	return buildCreationModelCatalogWithCategories(
		pricing,
		vendors,
		requestedMode,
		getCreationModelCategories(),
	)
}

func buildCreationModelCatalogWithCategories(
	pricing []model.Pricing,
	vendors []model.PricingVendor,
	requestedMode string,
	manualCategories map[string]string,
) dto.CreationModelCatalog {
	modelsByMode := make(map[string][]dto.CreationModel, len(creationModeOrder))
	usedVendorIDs := make(map[int]struct{})

	for _, item := range pricing {
		mode, hasManualMode := getManualCreationModelMode(item.ModelName, manualCategories)
		ok := hasManualMode
		if !hasManualMode {
			mode, ok = getCreationModelMode(item.ModelName, item.SupportedEndpointTypes)
		}
		if !ok || (requestedMode != "" && mode != requestedMode) {
			continue
		}
		metadata := getCreationModelMetadata(item.ModelName)
		metadataTags := metadata.Tags
		if hasManualMode {
			metadataTags = []string{mode}
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = metadata.Description
		}

		modelsByMode[mode] = append(modelsByMode[mode], dto.CreationModel{
			ID:                     item.ModelName,
			Description:            description,
			Icon:                   item.Icon,
			Tags:                   mergeCreationModelTags(splitCreationModelTags(item.Tags), metadataTags),
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

func getCreationModelCategories() map[string]string {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[creationModelCategoriesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	categories, _ := parseCreationModelCategories(raw)
	return categories
}

func getManualCreationModelMode(modelName string, categories map[string]string) (string, bool) {
	if len(categories) == 0 {
		return "", false
	}
	mode, ok := categories[strings.ToLower(strings.TrimSpace(modelName))]
	if !ok {
		return "", false
	}
	return mode, true
}

func validateCreationModelCategories(raw string) error {
	_, err := parseCreationModelCategories(raw)
	return err
}

func parseCreationModelCategories(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var parsed map[string]string
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return nil, fmt.Errorf("创作中心模型分类必须是 JSON 对象")
	}

	categories := make(map[string]string, len(parsed))
	for modelName, mode := range parsed {
		modelName = strings.ToLower(strings.TrimSpace(modelName))
		mode, ok := normalizeCreationMode(mode)
		if modelName == "" {
			return nil, fmt.Errorf("创作中心模型分类包含空模型名")
		}
		if !ok || mode == "" {
			return nil, fmt.Errorf("模型 %s 的分类必须是 chat、image 或 video", modelName)
		}
		categories[modelName] = mode
	}
	return categories, nil
}

func getCreationModelMode(modelName string, endpoints []constant.EndpointType) (string, bool) {
	if metadata := getCreationModelMetadata(modelName); metadata.Mode != "" {
		return metadata.Mode, true
	}

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

func getCreationModelMetadata(modelName string) creationModelMetadata {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if metadata, ok := creationModelMetadataByName[modelName]; ok {
		return metadata
	}
	switch {
	case strings.Contains(modelName, "gpt-image") ||
		strings.Contains(modelName, "nano-banana") ||
		strings.Contains(modelName, "imagen"):
		return creationModelMetadata{
			Mode:        creationModeImage,
			Description: "用于图片生成和视觉素材创作。",
			Tags:        []string{"image", "generation"},
		}
	case strings.HasPrefix(modelName, "sora") ||
		strings.HasPrefix(modelName, "veo") ||
		strings.Contains(modelName, "kling") ||
		strings.Contains(modelName, "grok-imagine-video"):
		return creationModelMetadata{
			Mode:        creationModeVideo,
			Description: "用于异步视频生成和媒体创作。",
			Tags:        []string{"video", "async"},
		}
	default:
		return creationModelMetadata{}
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

func mergeCreationModelTags(groups ...[]string) []string {
	tags := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, tag := range group {
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
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}
