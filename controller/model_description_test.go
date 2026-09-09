package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateModelDescriptionOnlyChangesDescription(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	original := model.Model{
		ModelName:    "model-note-existing",
		Description:  "old note",
		Icon:         "OpenAI",
		Tags:         "chat,reasoning",
		VendorID:     42,
		Endpoints:    `{"openai":"/v1/chat/completions"}`,
		Status:       1,
		SyncOfficial: 1,
		NameRule:     model.NameRuleExact,
	}
	require.NoError(t, original.Insert())

	body, err := common.Marshal(updateModelDescriptionRequest{
		ModelName:   original.ModelName,
		Description: "  public administrator note  ",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/models/description",
		bytes.NewReader(body),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateModelDescription(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var updated model.Model
	require.NoError(t, db.First(&updated, original.Id).Error)
	require.Equal(t, "public administrator note", updated.Description)
	require.Equal(t, original.ModelName, updated.ModelName)
	require.Equal(t, original.Icon, updated.Icon)
	require.Equal(t, original.Tags, updated.Tags)
	require.Equal(t, original.VendorID, updated.VendorID)
	require.Equal(t, original.Endpoints, updated.Endpoints)
	require.Equal(t, original.Status, updated.Status)
	require.Equal(t, original.SyncOfficial, updated.SyncOfficial)
	require.Equal(t, original.NameRule, updated.NameRule)
}

func TestUpdateModelDescriptionCreatesMetadataForVisibleModel(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	modelName := "model-note-visible-without-metadata"
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     modelName,
		ChannelId: 1,
		Enabled:   true,
	}).Error)
	model.InvalidatePricingCache()

	body, err := common.Marshal(updateModelDescriptionRequest{
		ModelName:   modelName,
		Description: "note for a visible model",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/models/description",
		bytes.NewReader(body),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateModelDescription(ctx)

	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var created model.Model
	require.NoError(t, db.Where("model_name = ?", modelName).First(&created).Error)
	require.Equal(t, "note for a visible model", created.Description)
	require.Equal(t, 1, created.Status)
	require.Equal(t, 0, created.SyncOfficial)
	require.Equal(t, model.NameRuleExact, created.NameRule)

	pricingByName := pricingByModelName(model.GetPricing())
	require.Equal(t, created.Description, pricingByName[modelName].Description)
}

func TestUpdateModelDescriptionRejectsOversizedNote(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	original := model.Model{
		ModelName:    "model-note-too-long",
		Description:  "old note",
		Status:       1,
		SyncOfficial: 1,
	}
	require.NoError(t, original.Insert())

	body, err := common.Marshal(updateModelDescriptionRequest{
		ModelName:   original.ModelName,
		Description: strings.Repeat("字", maxModelDescriptionRunes+1),
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/models/description",
		bytes.NewReader(body),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateModelDescription(ctx)

	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)

	var unchanged model.Model
	require.NoError(t, db.First(&unchanged, original.Id).Error)
	require.Equal(t, original.Description, unchanged.Description)
}
