package middleware

import (
	"net/http"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func CreationImageRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request.URL.Path = "/pg/images/generations"
		c.Set("relay_mode", relayconstant.RelayModeImagesGenerations)
		c.Next()
	}
}

func CreationVideoAsyncRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request.URL.Path = "/pg/video/async-generations"
		c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
		c.Next()
	}
}

func CreationVideoAsyncFetchConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		taskID := c.Param("task_id")
		c.Request.Method = http.MethodGet
		c.Request.URL.Path = "/v1/video/async-generations/" + taskID
		c.Set("task_id", taskID)
		c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
		c.Next()
	}
}
