package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelDisplayStatusIsExactPersistentAndSeparateFromRouting(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	t.Cleanup(model.InvalidatePricingCache)
	model.InvalidatePricingCache()
	const name = "display-status-model"
	original := model.Model{ModelName: name, Description: "original note", Status: 1, SyncOfficial: 1, Icon: "OpenAI", Endpoints: `{"openai":"/v1/chat/completions"}`}
	require.NoError(t, original.Insert())
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name + "-other", ChannelId: 1, Enabled: true}).Error)
	model.InvalidatePricingCache()

	request := func(available string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/models/display-status", strings.NewReader(`{"model_name":"`+name+`","available":`+available+`}`))
		c.Request.Header.Set("Content-Type", "application/json")
		UpdateModelDisplayStatus(c)
		return w
	}
	for _, available := range []bool{false, true, false} {
		encoded, err := common.Marshal(available)
		require.NoError(t, err)
		w := request(string(encoded))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var row model.ModelDisplayStatus
		require.NoError(t, db.Where("model_name = ?", name).First(&row).Error)
		require.Equal(t, available, row.Available)
		// Invalidate the existing pricing cache to simulate another process or
		// reload. The display override must survive and both models stay listed.
		model.InvalidatePricingCache()
		public, err := pricingWithDisplayStatus(model.GetPricing())
		require.NoError(t, err)
		got := map[string]bool{}
		for _, item := range public {
			got[item.ModelName] = item.DisplayAvailable
		}
		require.Equal(t, map[string]bool{name: available, name + "-other": true}, got)
	}
	var count int64
	require.NoError(t, db.Model(&model.ModelDisplayStatus{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	var unchanged model.Model
	require.NoError(t, db.First(&unchanged, original.Id).Error)
	require.Equal(t, original, unchanged)
	var abilities []model.Ability
	require.NoError(t, db.Find(&abilities).Error)
	for _, ability := range abilities {
		require.True(t, ability.Enabled)
	}

	// Actual API model discovery still includes the manually unavailable model.
	previousSelfUse := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() { operation_setting.SelfUseModeEnabled = previousSelfUse })
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set("token_group", "default")
	c.Set("group", "default")
	ListModels(c, constant.ChannelTypeOpenAI)
	require.Contains(t, w.Body.String(), name)
}

func TestModelDisplayStatusRejectsInvalidInputWithoutWriting(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	model.InvalidatePricingCache()
	t.Cleanup(model.InvalidatePricingCache)
	for _, body := range []string{
		`{}`, `{"model_name":"a"}`, `{"model_name":"a","available":null}`,
		`{"model_name":"a","available":"false"}`, `{"model_name":"a","available":0}`,
		`{"model_name":" ","available":false}`, `{"model_name":"a","available":false`,
		`{"model_name":"not-in-square","available":false}`,
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/models/display-status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		UpdateModelDisplayStatus(c)
		require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, w.Code, body)
	}
	var count int64
	require.NoError(t, db.Model(&model.ModelDisplayStatus{}).Count(&count).Error)
	require.Zero(t, count)
}
