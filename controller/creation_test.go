package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
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

func TestCreationReferenceImageUploadAndFetch(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "reference.png")
	require.NoError(t, err)
	_, err = part.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
	})
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/creation/reference-images", body)
	ctx.Request.Host = "example.test"
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

	UploadCreationReferenceImage(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.True(t, strings.HasPrefix(payload.Data.URL, "http://example.test/api/creation/reference-images/"))
	filename := filepath.Base(payload.Data.URL)
	t.Cleanup(func() { _ = cleanupCreationReferenceImageForTest(filename) })

	fetchRecorder := httptest.NewRecorder()
	fetchCtx, _ := gin.CreateTestContext(fetchRecorder)
	fetchCtx.Params = gin.Params{{Key: "filename", Value: filename}}
	fetchCtx.Request = httptest.NewRequest(http.MethodGet, "/api/creation/reference-images/"+filename, nil)

	GetCreationReferenceImage(fetchCtx)

	require.Equal(t, http.StatusOK, fetchRecorder.Code)
	require.Equal(t, "image/png", fetchRecorder.Header().Get("Content-Type"))
	require.NotEmpty(t, fetchRecorder.Body.Bytes())
}

func TestCreationReferenceImageRejectsNonImages(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "notes.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("not an image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/creation/reference-images", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

	UploadCreationReferenceImage(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "image must be")
}

func TestBuildCreationModelCatalogOverridesMediaModelsByName(t *testing.T) {
	catalog := buildCreationModelCatalog([]model.Pricing{
		{
			ModelName:              "gpt-image2",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "kling-v3",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "sora2",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
	}, nil, "")

	require.Empty(t, catalog.Modes[0].Models)
	require.Equal(t, []string{"gpt-image2"}, creationModelIDs(catalog.Modes[1].Models))
	require.Equal(t, []string{"kling-v3", "sora2"}, creationModelIDs(catalog.Modes[2].Models))
	require.NotEmpty(t, catalog.Modes[2].Models[1].Description)
	require.Contains(t, catalog.Modes[2].Models[1].Tags, "video")
}

func TestBuildCreationModelCatalogUsesManualCategories(t *testing.T) {
	catalog := buildCreationModelCatalogWithCategories([]model.Pricing{
		{
			ModelName:              "ko3",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "video-2.0",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "chat-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
	}, nil, "", map[string]string{
		"ko3":       creationModeImage,
		"video-2.0": creationModeVideo,
	}, nil, 1)

	require.Equal(t, []string{"chat-model"}, creationModelIDs(catalog.Modes[0].Models))
	require.Equal(t, []string{"ko3"}, creationModelIDs(catalog.Modes[1].Models))
	require.Equal(t, []string{"video-2.0"}, creationModelIDs(catalog.Modes[2].Models))
	require.Contains(t, catalog.Modes[1].Models[0].Tags, creationModeImage)
	require.Contains(t, catalog.Modes[2].Models[0].Tags, creationModeVideo)
}

func TestBuildCreationModelCatalogUsesManualDescriptions(t *testing.T) {
	catalog := buildCreationModelCatalogWithCategories([]model.Pricing{
		{
			ModelName:              "gpt-5.4",
			Description:            "pricing description",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		},
		{
			ModelName:              "custom-image",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration},
		},
	}, nil, "", nil, map[string]string{
		"gpt-5.4":      "manual chat description",
		"custom-image": "manual image description",
	}, 1)

	require.Equal(t, "manual chat description", catalog.Modes[0].Models[0].Description)
	require.Equal(t, "manual image description", catalog.Modes[1].Models[0].Description)
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

func TestBuildCreationModelCatalogAddsCostSummary(t *testing.T) {
	catalog := buildCreationModelCatalog([]model.Pricing{
		{
			ModelName:              "chat-model",
			ModelRatio:             1.5,
			CompletionRatio:        2,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIResponse},
		},
		{
			ModelName:              "image-model",
			QuotaType:              1,
			ModelPrice:             0.2,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration},
		},
		{
			ModelName:              "video-model",
			BillingMode:            billing_setting.BillingModeTieredExpr,
			BillingExpr:            `tier("base", p * 2 + c * 8)`,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		},
	}, nil, "", 1.25)

	chatCost := catalog.Modes[0].Models[0].Cost
	require.NotNil(t, chatCost)
	require.Equal(t, creationCostModePerToken, chatCost.BillingMode)
	require.NotNil(t, chatCost.InputPricePerMillion)
	require.NotNil(t, chatCost.OutputPricePerMillion)
	require.Equal(t, 3.75, *chatCost.InputPricePerMillion)
	require.Equal(t, 7.5, *chatCost.OutputPricePerMillion)

	imageCost := catalog.Modes[1].Models[0].Cost
	require.NotNil(t, imageCost)
	require.Equal(t, creationCostModePerRequest, imageCost.BillingMode)
	require.NotNil(t, imageCost.RequestPrice)
	require.NotNil(t, imageCost.RequestQuota)
	require.Equal(t, 0.25, *imageCost.RequestPrice)
	require.Equal(t, 125000, *imageCost.RequestQuota)

	videoCost := catalog.Modes[2].Models[0].Cost
	require.NotNil(t, videoCost)
	require.Equal(t, creationCostModeDynamic, videoCost.BillingMode)

	payload, err := common.Marshal(catalog)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "billing_expr")
}

func TestBuildCreationModelCatalogKeepsExplicitZeroCostFields(t *testing.T) {
	catalog := buildCreationModelCatalog([]model.Pricing{
		{
			ModelName:              "zero-token-model",
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIResponse},
		},
		{
			ModelName:              "zero-request-model",
			QuotaType:              1,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration},
		},
	}, nil, "")

	payload, err := common.Marshal(catalog)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"input_price_per_million":0`)
	require.Contains(t, string(payload), `"output_price_per_million":0`)
	require.Contains(t, string(payload), `"request_price":0`)
	require.Contains(t, string(payload), `"request_quota":0`)
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

func TestParseCreationModelCategoriesNormalizesAndValidates(t *testing.T) {
	categories, err := parseCreationModelCategories(`{" KO3 ":" IMAGE "}`)

	require.NoError(t, err)
	require.Equal(t, map[string]string{"ko3": creationModeImage}, categories)

	_, err = parseCreationModelCategories(`{"ko3":"audio"}`)
	require.Error(t, err)
}

func TestParseCreationModelDescriptionsNormalizesAndValidates(t *testing.T) {
	descriptions, err := parseCreationModelDescriptions(`{" KO3 ":"  image model  ","blank":" "}`)

	require.NoError(t, err)
	require.Equal(t, map[string]string{"ko3": "image model"}, descriptions)

	_, err = parseCreationModelDescriptions(`[]`)
	require.Error(t, err)
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
