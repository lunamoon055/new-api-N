package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var validMediaTransferStatuses = map[model.MediaTransferStatus]struct{}{
	model.MediaTransferPending: {}, model.MediaTransferProcessing: {},
	model.MediaTransferRetry: {}, model.MediaTransferReady: {}, model.MediaTransferFailed: {},
}

// GetMediaTransferDashboard exposes only operational metadata. Encrypted
// source URLs, auth headers, lease tokens and response payloads never leave the
// server through this endpoint.
func GetMediaTransferDashboard(c *gin.Context) {
	status := model.MediaTransferStatus(strings.TrimSpace(strings.ToUpper(c.Query("status"))))
	if status != "" {
		if _, ok := validMediaTransferStatuses[status]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的转存任务状态"})
			return
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	dashboard, err := model.GetMediaTransferDashboard(string(status), (page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"summary":   dashboard.Summary,
		"items":     dashboard.Items,
		"total":     dashboard.Total,
		"page":      page,
		"page_size": pageSize,
		"providers": service.GetMediaStorageProviderHealth(),
	})
}

// RetryMediaTransfer requeues an exhausted transfer in place. It does not
// submit an upstream request, reserve quota, or refund a settled task.
func RetryMediaTransfer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的转存任务 ID"})
		return
	}
	won, err := model.RetryMediaTransferJob(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "转存任务不存在"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if !won {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "任务当前不允许重试，只能重试失败或等待中的转存任务"})
		return
	}
	service.WakeMediaTransferWorker()
	common.ApiSuccess(c, gin.H{"id": id, "status": model.MediaTransferPending})
}
