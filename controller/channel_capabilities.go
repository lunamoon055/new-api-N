package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func RefreshChannelCapabilities(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid_channel_id"})
		return
	}
	channel, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "channel_not_found"})
		return
	}
	if !service.SupportsChannelCapabilitySync(channel.Type, channel.GetBaseURL()) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "capability_sync_unsupported"})
		return
	}
	result, err := service.RefreshChannelCapabilities(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "capability_sync_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
