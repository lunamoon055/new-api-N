package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Model tag routes are protected by the models group's AdminAuth middleware.
func GetModelTags(c *gin.Context) {
	tags, err := model.GetModelTags()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tags)
}

func BatchAddModelTags(c *gin.Context) {
	var req struct {
		IDs  []int    `json:"ids"`
		Tags []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求格式无效")
		return
	}
	count, err := model.BatchAddModelTags(req.IDs, req.Tags)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, gin.H{"updated_count": count})
}

func BatchClearModelTags(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求格式无效")
		return
	}
	count, err := model.BatchClearModelTags(req.IDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, gin.H{"updated_count": count})
}
