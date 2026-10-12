package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ManualMediaTransfer is restricted by the root-only media storage routes.
// Multipart uploads are bounded and queued without submitting another generation.
func ManualMediaTransfer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的转存任务 ID"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxManualMediaUploadBytes+(1<<20))
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": service.ErrInvalidManualMediaInput.Error()})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	var file io.ReadCloser
	var filename string
	if files := c.Request.MultipartForm.File["file"]; len(files) > 0 {
		if len(files) != 1 || files[0].Size > service.MaxManualMediaUploadBytes {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": service.ErrInvalidManualMediaInput.Error()})
			return
		}
		file, err = files[0].Open()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		defer file.Close()
		filename = files[0].Filename
	}
	// A nil interface must stay nil when no file was supplied.
	var reader io.Reader
	if file != nil {
		reader = file
	}
	won, err := service.QueueManualMediaTransfer(id, c.PostForm("provider_id"), c.PostForm("source_url"), reader, filename)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "转存任务不存在"})
		case errors.Is(err, service.ErrInvalidManualMediaInput):
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		default:
			common.ApiError(c, err)
		}
		return
	}
	if !won {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "任务状态已变化，请刷新后重试；仅失败或等待重试的任务允许手动转存"})
		return
	}
	common.ApiSuccess(c, gin.H{"id": id, "status": model.MediaTransferPending})
}
