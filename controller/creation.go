package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	creationModeChat  = "chat"
	creationModeImage = "image"
	creationModeVideo = "video"

	creationModelCategoriesOptionKey   = "CreationModelCategories"
	creationModelDescriptionsOptionKey = "CreationModelDescriptions"

	creationCostModeDynamic    = "dynamic"
	creationCostModePerRequest = "per_request"
	creationCostModePerToken   = "per_token"
)

const (
	sanbaoCreationModelCapabilitiesTTL = 5 * time.Minute
	sanbaoProviderHost                 = "sanbaobeauty.com"
)

var creationModeOrder = []string{
	creationModeChat,
	creationModeImage,
	creationModeVideo,
}

var sanbaoCreationModelCapabilitiesCache = struct {
	sync.Mutex
	expiresAt time.Time
	values    map[string]dto.CreationModelMetadata
}{}

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
	"videos-standard": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的标准模型。",
		Tags:        []string{"video", "async"},
	},
	"videos-fast": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的快速模型。",
		Tags:        []string{"video", "async"},
	},
	"videos-mini": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的轻量模型。",
		Tags:        []string{"video", "async"},
	},
	"sd2-mini": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的轻量模型。",
		Tags:        []string{"video", "async"},
	},
	"sd2-fast": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的快速模型。",
		Tags:        []string{"video", "async"},
	},
	"sd2满血": {
		Mode:        creationModeVideo,
		Description: "用于视频生成和创作的标准模型。",
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

	pricing, group, usableGroup := getPricingForRequest(c)
	providerMetadata := getCreationProviderModelMetadata(c.Request.Context(), pricing, usableGroup)
	common.ApiSuccess(c, buildCreationModelCatalogWithProviderMetadata(
		pricing,
		model.GetVendors(),
		mode,
		providerMetadata,
		getCreationCatalogGroupRatio(group),
	))
}

func CreationRelayImage(c *gin.Context) {
	if newAPIError := setupCreationRelayContext(c, "creation-image"); newAPIError != nil {
		respondCreationRelayError(c, newAPIError)
		return
	}

	if shouldUseSanbaoCreationTaskRelay(c) {
		applySanbaoCreationTaskRelayContext(c)
		c.Request.URL.Path = "/pg/images/generations"
		RelayTask(c)
		return
	}

	Relay(c, types.RelayFormatOpenAIImage)
}

func CreationRelayTask(c *gin.Context) {
	if newAPIError := setupCreationRelayContext(c, "creation-video"); newAPIError != nil {
		respondCreationRelayError(c, newAPIError)
		return
	}

	if shouldUseSanbaoCreationTaskRelay(c) {
		applySanbaoCreationTaskRelayContext(c)
		c.Request.URL.Path = "/pg/video/async-generations"
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

	usingGroup := resolveCreationUsingGroup(c, userCache)

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

func resolveCreationUsingGroup(c *gin.Context, userCache *model.UserBase) string {
	usingGroup := ""
	if userCache != nil {
		usingGroup = strings.TrimSpace(userCache.Group)
	}
	if usingGroup == "" {
		usingGroup = strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	}
	if usingGroup == "" {
		usingGroup = strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
	}
	if usingGroup == "" {
		usingGroup = "default"
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, usingGroup)
	return usingGroup
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

func buildCreationModelCatalog(pricing []model.Pricing, vendors []model.PricingVendor, requestedMode string, groupRatio ...float64) dto.CreationModelCatalog {
	return buildCreationModelCatalogWithProviderMetadata(pricing, vendors, requestedMode, nil, groupRatio...)
}

func buildCreationModelCatalogWithProviderMetadata(
	pricing []model.Pricing,
	vendors []model.PricingVendor,
	requestedMode string,
	providerMetadata map[string]dto.CreationModelMetadata,
	groupRatio ...float64,
) dto.CreationModelCatalog {
	costGroupRatio := 1.0
	if len(groupRatio) > 0 && groupRatio[0] >= 0 {
		costGroupRatio = groupRatio[0]
	}
	return buildCreationModelCatalogWithCategoriesAndMetadata(
		pricing,
		vendors,
		requestedMode,
		getCreationModelCategories(),
		getCreationModelDescriptions(),
		costGroupRatio,
		providerMetadata,
	)
}

func buildCreationModelCatalogWithCategories(
	pricing []model.Pricing,
	vendors []model.PricingVendor,
	requestedMode string,
	manualCategories map[string]string,
	manualDescriptions map[string]string,
	groupRatio float64,
) dto.CreationModelCatalog {
	return buildCreationModelCatalogWithCategoriesAndMetadata(
		pricing,
		vendors,
		requestedMode,
		manualCategories,
		manualDescriptions,
		groupRatio,
		nil,
	)
}

func buildCreationModelCatalogWithCategoriesAndMetadata(
	pricing []model.Pricing,
	vendors []model.PricingVendor,
	requestedMode string,
	manualCategories map[string]string,
	manualDescriptions map[string]string,
	groupRatio float64,
	providerMetadata map[string]dto.CreationModelMetadata,
) dto.CreationModelCatalog {
	modelsByMode := make(map[string][]dto.CreationModel, len(creationModeOrder))
	usedVendorIDs := make(map[int]struct{})

	for _, item := range pricing {
		providerMeta, hasProviderMeta := getProviderCreationModelMetadata(item.ModelName, providerMetadata)
		mode, hasManualMode := getManualCreationModelMode(item.ModelName, manualCategories)
		ok := hasManualMode
		if !hasManualMode {
			if providerMode, hasProviderMode := getCreationModelModeFromProviderMetadata(providerMeta); hasProviderMode {
				mode, ok = providerMode, true
			} else {
				mode, ok = getCreationModelMode(item.ModelName, item.SupportedEndpointTypes)
			}
		}
		if !ok || (requestedMode != "" && mode != requestedMode) {
			continue
		}
		metadata := getCreationModelMetadata(item.ModelName)
		metadataTags := metadata.Tags
		if hasManualMode {
			metadataTags = []string{mode}
		}
		manualDescription, hasManualDescription := getManualCreationModelDescription(item.ModelName, manualDescriptions)
		description := strings.TrimSpace(item.Description)
		if hasManualDescription {
			description = manualDescription
		}
		if description == "" && hasProviderMeta {
			description = strings.TrimSpace(providerMeta.Description)
		}
		if description == "" {
			description = metadata.Description
		}
		var responseMetadata *dto.CreationModelMetadata
		if hasProviderMeta {
			meta := providerMeta
			responseMetadata = &meta
		}

		modelsByMode[mode] = append(modelsByMode[mode], dto.CreationModel{
			ID:                     item.ModelName,
			Description:            description,
			ManualDescription:      manualDescription,
			Icon:                   item.Icon,
			Tags:                   mergeCreationModelTags(splitCreationModelTags(item.Tags), metadataTags),
			VendorID:               item.VendorID,
			Cost:                   buildCreationModelCost(item, groupRatio),
			Metadata:               responseMetadata,
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

func getCreationCatalogGroupRatio(userGroup string) float64 {
	userGroup = strings.TrimSpace(userGroup)
	if userGroup == "" {
		return 1
	}
	if ratio, ok := ratio_setting.GetGroupGroupRatio(userGroup, userGroup); ok {
		return ratio
	}
	return ratio_setting.GetGroupRatio(userGroup)
}

func getCreationProviderModelMetadata(ctx context.Context, pricing []model.Pricing, usableGroup map[string]string) map[string]dto.CreationModelMetadata {
	if len(pricing) == 0 {
		return nil
	}

	modelNames := make([]string, 0, len(pricing))
	for _, item := range pricing {
		if strings.TrimSpace(item.ModelName) != "" {
			modelNames = append(modelNames, item.ModelName)
		}
	}
	if len(modelNames) == 0 {
		return nil
	}

	groups := make([]string, 0, len(usableGroup))
	for group := range usableGroup {
		if strings.TrimSpace(group) != "" {
			groups = append(groups, group)
		}
	}
	sort.Strings(groups)

	owners, err := model.GetPreferredModelOwnerChannelTypes(modelNames, groups)
	if err != nil {
		common.SysLog(fmt.Sprintf("GetPreferredModelOwnerChannelTypes error: %v", err))
		return nil
	}

	needsSanbaoMetadata := false
	for _, item := range pricing {
		if owners[item.ModelName] == constant.ChannelTypeSanbao || isLikelySanbaoModelName(item.ModelName) {
			needsSanbaoMetadata = true
			break
		}
	}
	if !needsSanbaoMetadata {
		return nil
	}

	sanbaoCapabilities := getSanbaoCreationModelCapabilities(ctx)
	if len(sanbaoCapabilities) == 0 {
		return nil
	}

	result := make(map[string]dto.CreationModelMetadata)
	for _, item := range pricing {
		if metadata, ok := sanbaoCapabilities[normalizeCreationModelMetadataKey(item.ModelName)]; ok {
			if !shouldAttachSanbaoCreationMetadata(item.ModelName, owners[item.ModelName], metadata) {
				continue
			}
			result[normalizeCreationModelMetadataKey(item.ModelName)] = metadata
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func getSanbaoCreationModelCapabilities(ctx context.Context) map[string]dto.CreationModelMetadata {
	sanbaoCreationModelCapabilitiesCache.Lock()
	if time.Now().Before(sanbaoCreationModelCapabilitiesCache.expiresAt) {
		cached := cloneCreationModelMetadataMap(sanbaoCreationModelCapabilitiesCache.values)
		sanbaoCreationModelCapabilitiesCache.Unlock()
		return cached
	}
	sanbaoCreationModelCapabilitiesCache.Unlock()

	channels, err := model.GetEnabledChannelsByTypeWithKeys(constant.ChannelTypeSanbao)
	if err != nil {
		common.SysLog(fmt.Sprintf("get Sanbao channels failed: %v", err))
	}
	hostChannels, hostErr := model.GetEnabledChannelsByBaseURLHostWithKeys(sanbaoProviderHost)
	if hostErr != nil {
		common.SysLog(fmt.Sprintf("get Sanbao base url channels failed: %v", hostErr))
	}
	channels = mergeSanbaoCreationChannels(channels, hostChannels)

	var capabilities map[string]dto.CreationModelMetadata
	for _, channel := range channels {
		if capabilities == nil {
			capabilities, err = fetchSanbaoCreationModelCapabilities(ctx, channel)
			if err != nil {
				common.SysLog(fmt.Sprintf("fetch Sanbao model capabilities failed: channel_id=%d err=%v", channel.Id, err))
				continue
			}
		}
		applySanbaoModelMappingAliases(capabilities, channel)
	}
	if len(capabilities) == 0 {
		var err error
		capabilities, err = fetchPublicSanbaoCreationModelCapabilities(ctx, "")
		if err != nil {
			common.SysLog(fmt.Sprintf("fetch public Sanbao model capabilities failed: %v", err))
			return nil
		}
	}

	sanbaoCreationModelCapabilitiesCache.Lock()
	sanbaoCreationModelCapabilitiesCache.values = cloneCreationModelMetadataMap(capabilities)
	sanbaoCreationModelCapabilitiesCache.expiresAt = time.Now().Add(sanbaoCreationModelCapabilitiesTTL)
	sanbaoCreationModelCapabilitiesCache.Unlock()
	return capabilities
}

func fetchSanbaoCreationModelCapabilities(ctx context.Context, channel *model.Channel) (map[string]dto.CreationModelMetadata, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	apiKey, _, apiErr := channel.GetNextEnabledKey()
	if apiErr != nil {
		return nil, apiErr
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Sanbao channel key is empty")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, normalizeSanbaoCreationBaseURL(channel.GetBaseURL())+"/openapi/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sanbao /models returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseSanbaoCreationModelCapabilities(body)
}

func fetchPublicSanbaoCreationModelCapabilities(ctx context.Context, baseURL string) (map[string]dto.CreationModelMetadata, error) {
	baseURL = normalizeSanbaoCreationBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "https://" + sanbaoProviderHost
	}

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/public/openapi-models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Sanbao public /models returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseSanbaoCreationModelCapabilities(body)
}

func parseSanbaoCreationModelCapabilities(body []byte) (map[string]dto.CreationModelMetadata, error) {
	var payload struct {
		Data []dto.CreationModelMetadata `json:"data"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		var items []dto.CreationModelMetadata
		if err := common.Unmarshal(body, &items); err != nil {
			return nil, err
		}
		payload.Data = items
	}

	result := make(map[string]dto.CreationModelMetadata, len(payload.Data))
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		item.Provider = "sanbao"
		if item.UpstreamModelID == "" {
			item.UpstreamModelID = id
		}
		result[normalizeCreationModelMetadataKey(id)] = item
		if name := strings.TrimSpace(item.Name); name != "" {
			result[normalizeCreationModelMetadataKey(name)] = item
		}
	}
	return result, nil
}

func mergeSanbaoCreationChannels(groups ...[]*model.Channel) []*model.Channel {
	result := make([]*model.Channel, 0)
	seen := make(map[int]struct{})
	for _, group := range groups {
		for _, channel := range group {
			if channel == nil {
				continue
			}
			if _, ok := seen[channel.Id]; ok {
				continue
			}
			seen[channel.Id] = struct{}{}
			result = append(result, channel)
		}
	}
	return result
}

func applySanbaoModelMappingAliases(capabilities map[string]dto.CreationModelMetadata, channel *model.Channel) {
	if len(capabilities) == 0 || channel == nil {
		return
	}
	rawMapping := strings.TrimSpace(channel.GetModelMapping())
	if rawMapping == "" || rawMapping == "{}" {
		return
	}

	var parsed map[string]string
	if err := common.UnmarshalJsonStr(rawMapping, &parsed); err != nil {
		common.SysLog(fmt.Sprintf("parse Sanbao model mapping failed: channel_id=%d err=%v", channel.Id, err))
		return
	}
	for alias, upstream := range parsed {
		alias = strings.TrimSpace(alias)
		upstream = strings.TrimSpace(upstream)
		if alias == "" || upstream == "" {
			continue
		}
		metadata, ok := capabilities[normalizeCreationModelMetadataKey(upstream)]
		if !ok {
			continue
		}
		metadata.ID = alias
		if metadata.UpstreamModelID == "" {
			metadata.UpstreamModelID = upstream
		}
		capabilities[normalizeCreationModelMetadataKey(alias)] = metadata
	}
}

func shouldAttachSanbaoCreationMetadata(modelName string, ownerChannelType int, metadata dto.CreationModelMetadata) bool {
	if ownerChannelType == constant.ChannelTypeSanbao {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(metadata.Provider), "sanbao") && isLikelySanbaoModelName(modelName) {
		return true
	}
	return false
}

func isLikelySanbaoModelName(modelName string) bool {
	key := normalizeCreationModelMetadataKey(modelName)
	return strings.HasPrefix(key, "sd2") ||
		strings.HasPrefix(key, "sd-2.0-") ||
		strings.HasPrefix(key, "sd-2-c") ||
		strings.HasPrefix(key, "seedance-2.") ||
		strings.HasPrefix(key, "grok_video") ||
		strings.HasPrefix(key, "gpt-image2-")
}

func isLikelySanbaoChannel(channel *model.Channel) bool {
	if channel == nil {
		return false
	}
	if channel.Type == constant.ChannelTypeSanbao {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(channel.GetBaseURL())), sanbaoProviderHost)
}

func shouldUseSanbaoCreationTaskRelay(c *gin.Context) bool {
	if common.GetContextKeyInt(c, constant.ContextKeyChannelType) == constant.ChannelTypeSanbao {
		return true
	}
	baseURL := common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl)
	return strings.Contains(strings.ToLower(strings.TrimSpace(baseURL)), sanbaoProviderHost)
}

func applySanbaoCreationTaskRelayContext(c *gin.Context) {
	channelType := constant.ChannelTypeSanbao
	c.Set("channel_type", channelType)
	c.Set("platform", strconv.Itoa(channelType))
	common.SetContextKey(c, constant.ContextKeyChannelType, channelType)
}

func getProviderCreationModelMetadata(modelName string, metadata map[string]dto.CreationModelMetadata) (dto.CreationModelMetadata, bool) {
	if len(metadata) == 0 {
		return dto.CreationModelMetadata{}, false
	}
	value, ok := metadata[normalizeCreationModelMetadataKey(modelName)]
	return value, ok
}

func getCreationModelModeFromProviderMetadata(metadata dto.CreationModelMetadata) (string, bool) {
	modelType := strings.ToLower(strings.TrimSpace(metadata.Type))
	if modelType == "" {
		modelType = strings.ToLower(strings.TrimSpace(metadata.Category))
	}
	if modelType == "" {
		modelType = strings.ToLower(strings.TrimSpace(metadata.ModelType))
	}
	switch {
	case strings.Contains(modelType, "image"):
		return creationModeImage, true
	case strings.Contains(modelType, "video"):
		return creationModeVideo, true
	default:
		return "", false
	}
}

func normalizeCreationModelMetadataKey(modelName string) string {
	return strings.ToLower(strings.TrimSpace(modelName))
}

func normalizeSanbaoCreationBaseURL(value string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(value), "/")
	lowerBaseURL := strings.ToLower(baseURL)
	for _, suffix := range []string{"/openapi/v1", "/openapi", "/v1"} {
		if strings.HasSuffix(lowerBaseURL, suffix) {
			return strings.TrimRight(baseURL[:len(baseURL)-len(suffix)], "/")
		}
	}
	return baseURL
}

func cloneCreationModelMetadataMap(src map[string]dto.CreationModelMetadata) map[string]dto.CreationModelMetadata {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]dto.CreationModelMetadata, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func buildCreationModelCost(item model.Pricing, groupRatio float64) *dto.CreationModelCost {
	if groupRatio < 0 {
		groupRatio = 1
	}
	videoBillingMode := billing_setting.GetVideoBillingMode(item.ModelName)
	if billing_setting.IsVideoResolutionTierMode(videoBillingMode) {
		if prices, ok := billing_setting.GetVideoResolutionPrices(item.ModelName); ok {
			return &dto.CreationModelCost{
				BillingMode:           videoBillingMode,
				VideoBillingMode:      videoBillingMode,
				VideoResolutionPrices: buildCreationVideoResolutionPrices(prices, groupRatio),
				VideoResolutionQuotas: buildCreationVideoResolutionQuotas(prices, groupRatio),
				GroupRatio:            groupRatio,
			}
		}
	}
	if item.BillingMode == billing_setting.BillingModeTieredExpr && strings.TrimSpace(item.BillingExpr) != "" {
		return &dto.CreationModelCost{
			BillingMode: creationCostModeDynamic,
			GroupRatio:  groupRatio,
		}
	}
	if item.QuotaType == 1 {
		requestPrice := item.ModelPrice * groupRatio
		requestQuota := int(item.ModelPrice * common.QuotaPerUnit * groupRatio)
		return &dto.CreationModelCost{
			BillingMode:  creationCostModePerRequest,
			RequestPrice: &requestPrice,
			RequestQuota: &requestQuota,
			GroupRatio:   groupRatio,
		}
	}

	inputPrice := item.ModelRatio * 2 * groupRatio
	outputPrice := inputPrice * item.CompletionRatio
	return &dto.CreationModelCost{
		BillingMode:           creationCostModePerToken,
		InputPricePerMillion:  &inputPrice,
		OutputPricePerMillion: &outputPrice,
		GroupRatio:            groupRatio,
	}
}

func buildCreationVideoResolutionPrices(prices map[string]float64, groupRatio float64) map[string]float64 {
	result := make(map[string]float64, len(prices))
	for resolution, price := range prices {
		normalizedResolution := billing_setting.NormalizeVideoResolution(resolution)
		if normalizedResolution == "" {
			continue
		}
		result[normalizedResolution] = price * groupRatio
	}
	return result
}

func buildCreationVideoResolutionQuotas(prices map[string]float64, groupRatio float64) map[string]int {
	result := make(map[string]int, len(prices))
	for resolution, price := range prices {
		normalizedResolution := billing_setting.NormalizeVideoResolution(resolution)
		if normalizedResolution == "" {
			continue
		}
		result[normalizedResolution] = int(price * common.QuotaPerUnit * groupRatio)
	}
	return result
}

func getCreationModelCategories() map[string]string {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[creationModelCategoriesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	categories, _ := parseCreationModelCategories(raw)
	return categories
}

func getCreationModelDescriptions() map[string]string {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[creationModelDescriptionsOptionKey]
	common.OptionMapRWMutex.RUnlock()
	descriptions, _ := parseCreationModelDescriptions(raw)
	return descriptions
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

func getManualCreationModelDescription(modelName string, descriptions map[string]string) (string, bool) {
	if len(descriptions) == 0 {
		return "", false
	}
	description, ok := descriptions[strings.ToLower(strings.TrimSpace(modelName))]
	if !ok {
		return "", false
	}
	return description, true
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

func validateCreationModelDescriptions(raw string) error {
	_, err := parseCreationModelDescriptions(raw)
	return err
}

func parseCreationModelDescriptions(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var parsed map[string]string
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return nil, fmt.Errorf("创作中心模型描述必须是 JSON 对象")
	}

	descriptions := make(map[string]string, len(parsed))
	for modelName, description := range parsed {
		modelName = strings.ToLower(strings.TrimSpace(modelName))
		description = strings.TrimSpace(description)
		if modelName == "" {
			return nil, fmt.Errorf("创作中心模型描述包含空模型名")
		}
		if description == "" {
			continue
		}
		descriptions[modelName] = description
	}
	return descriptions, nil
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
	modelName = normalizeCreationModelName(modelName)
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
		strings.HasPrefix(modelName, "video-2.0") ||
		strings.HasPrefix(modelName, "videos-") ||
		strings.HasPrefix(modelName, "sd2") ||
		strings.HasPrefix(modelName, "sd-2.0-") ||
		strings.HasPrefix(modelName, "sd-2-c") ||
		strings.HasPrefix(modelName, "seedance-2.") ||
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

func normalizeCreationModelName(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if index := strings.IndexAny(normalized, "(（"); index == 0 {
		if closeIndex := strings.IndexAny(normalized, ")）"); closeIndex > 0 {
			normalized = strings.TrimSpace(normalized[closeIndex+1:])
		}
	}
	return normalized
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
