package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetImageGenerationResult(c *gin.Context) {
	file, mimeType, size, err := service.OpenImageGenerationResult(c.Param("filename"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, size, mimeType, file, nil)
}
