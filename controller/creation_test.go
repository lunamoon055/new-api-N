package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildCreationModelCatalogCategorizesModels(t *testing.T) {
	catalog := buildCreationModelCatalog([]model.Pricing{
		{
			ModelName:              "z-chat-model",
			Tags:                   "recommended, fast, recommended",
			VendorID:               2,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "a-chat-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeAnthropic},
		},
		{
			ModelName:              "image-model",
			VendorID:               1,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration, constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "video-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		},
		{
			ModelName:              "embedding-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeEmbeddings},
		},
	}, []model.PricingVendor{
		{ID: 2, Name: "Second"},
		{ID: 1, Name: "First"},
		{ID: 3, Name: "Unused"},
	}, "")

	require.Equal(t, []string{creationModeChat, creationModeImage, creationModeVideo}, creationGroupModes(catalog.Modes))
	require.Equal(t, []string{"a-chat-model", "z-chat-model"}, creationModelIDs(catalog.Modes[0].Models))
	require.Equal(t, []string{"recommended", "fast"}, catalog.Modes[0].Models[1].Tags)
	require.Equal(t, []string{"image-model"}, creationModelIDs(catalog.Modes[1].Models))
	require.Equal(t, []string{"video-model"}, creationModelIDs(catalog.Modes[2].Models))
	require.Equal(t, []string{"First", "Second"}, creationVendorNames(catalog.Vendors))
}

func TestBuildCreationModelCatalogFiltersRequestedModeAndRedactsPricing(t *testing.T) {
	catalog := buildCreationModelCatalog([]model.Pricing{
		{
			ModelName:              "chat-model",
			ModelRatio:             1.5,
			BillingExpr:            `tier("base", p * 1 + c * 2)`,
			EnableGroup:            []string{"default"},
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIResponse},
		},
		{
			ModelName:              "video-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		},
	}, nil, creationModeChat)

	require.Equal(t, []string{creationModeChat}, creationGroupModes(catalog.Modes))
	require.Equal(t, []string{"chat-model"}, creationModelIDs(catalog.Modes[0].Models))

	payload, err := common.Marshal(catalog)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "model_ratio")
	require.NotContains(t, string(payload), "billing_expr")
	require.NotContains(t, string(payload), "enable_groups")
}

func TestGetCreationModelsRejectsUnknownModeBeforeLoadingCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/creation/models?mode=audio", nil)

	GetCreationModels(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid mode")
}

func TestNormalizeCreationModeTrimsAndNormalizesCase(t *testing.T) {
	mode, ok := normalizeCreationMode(" IMAGE ")

	require.True(t, ok)
	require.Equal(t, creationModeImage, mode)
}

func TestSplitCreationModelTagsOmitsBlankTags(t *testing.T) {
	require.Equal(t, []string{"fast", "recommended"}, splitCreationModelTags(" fast, , recommended "))
	require.Nil(t, splitCreationModelTags(strings.Repeat(" ", 3)))
}

func creationGroupModes(groups []dto.CreationModelGroup) []string {
	modes := make([]string, len(groups))
	for i, group := range groups {
		modes[i] = group.Mode
	}
	return modes
}

func creationModelIDs(models []dto.CreationModel) []string {
	ids := make([]string, len(models))
	for i, item := range models {
		ids[i] = item.ID
	}
	return ids
}

func creationVendorNames(vendors []dto.CreationVendor) []string {
	names := make([]string, len(vendors))
	for i, vendor := range vendors {
		names[i] = vendor.Name
	}
	return names
}
