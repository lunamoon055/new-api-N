package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSavingChannelAutomaticallyRefreshesSharedCapabilityMetadata(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelCapability{}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/pricing", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"success":true,"data":[{"model_name":"next-video","description":"30图10音0视/720P/5-30秒","supported_endpoint_types":["openai-video"]}]}`))
	}))
	defer server.Close()
	base, mapping := server.URL, `{"Public":"next-video"}`
	channel := model.Channel{Type: constant.ChannelTypeV6K, Name: "Fixture", Key: "fixture-key", BaseURL: &base, Models: "Public", ModelMapping: &mapping, Status: 1, Group: "default"}
	require.NoError(t, db.Create(&channel).Error)
	body, err := common.Marshal(channel)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/channel/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	UpdateChannel(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Eventually(t, func() bool {
		var count int64
		db.Model(&model.ChannelModelCapability{}).Where("channel_id = ?", channel.Id).Count(&count)
		return count == 1
	}, 3*time.Second, 10*time.Millisecond)
	pricing := model.GetPricing()
	require.Len(t, pricing, 1)
	require.Equal(t, 30, pricing[0].MappedEndpoints[0].DocumentationFields.MaxImages)
	require.Equal(t, "referenceImages", pricing[0].MappedEndpoints[0].DocumentationFields.ImageField)
	require.Equal(t, "upstream_catalog", pricing[0].MappedEndpoints[0].CapabilitySource)
}

func TestCapabilityRefreshFailureDoesNotExposeUpstreamDetails(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelCapability{}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	base := server.URL
	channel := model.Channel{Type: constant.ChannelTypeV6K, Key: "fixture-private-key", BaseURL: &base, Models: "Public", Status: 1}
	require.NoError(t, db.Create(&channel).Error)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/1/capabilities/refresh", nil)
	RefreshChannelCapabilities(c)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Contains(t, recorder.Body.String(), "capability_sync_failed")
	require.NotContains(t, recorder.Body.String(), "fixture-private-key")
	require.NotContains(t, recorder.Body.String(), server.URL)
}
