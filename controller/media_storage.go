package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type mediaStorageSettingsResponse struct {
	Providers []service.MediaStorageProvider `json:"providers"`
}

type mediaStorageSettingsRequest struct {
	Providers []service.MediaStorageProvider `json:"providers"`
}

type mediaStorageTestRequest struct {
	ProviderID string `json:"provider_id"`
}

func GetMediaStorageSettings(c *gin.Context) {
	providers := service.GetMediaStorageProviders()
	for i := range providers {
		if providers[i].Token != "" {
			providers[i].Token = "********"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    mediaStorageSettingsResponse{Providers: providers},
	})
}

func UpdateMediaStorageSettings(c *gin.Context) {
	var request mediaStorageSettingsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的图床配置"})
		return
	}
	if err := service.ValidateMediaStorageProviders(request.Providers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	previous := service.GetMediaStorageProviders()
	previousTokens := make(map[string]string, len(previous))
	for _, provider := range previous {
		previousTokens[provider.ID] = provider.Token
	}
	for i := range request.Providers {
		if strings.TrimSpace(request.Providers[i].Token) == "********" {
			request.Providers[i].Token = previousTokens[request.Providers[i].ID]
		}
	}
	raw, err := common.Marshal(request.Providers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "图床配置序列化失败"})
		return
	}
	if err := model.UpdateOption(service.MediaStorageOptionKey, string(raw)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func TestMediaStorage(c *gin.Context) {
	var request mediaStorageTestRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的图床测试请求"})
		return
	}
	if strings.TrimSpace(request.ProviderID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择要测试的图床"})
		return
	}

	url, err := service.TestMediaStorageProvider(c.Request.Context(), request.ProviderID)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, service.ErrMediaStorageProviderNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"provider_id": request.ProviderID,
			"url":         url,
		},
	})
}
