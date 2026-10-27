package controller

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type updateModelDisplayStatusRequest struct {
	ModelName string `json:"model_name"`
	Available *bool  `json:"available"`
}

type publicPricingModel struct {
	model.Pricing
	DisplayAvailable bool `json:"display_available"`
}

// UpdateModelDisplayStatus is protected by the models group's AdminAuth.
// Only the independent Model Square display flag is written.
func UpdateModelDisplayStatus(c *gin.Context) {
	var req updateModelDisplayStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Available == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "model_name and a boolean available are required"})
		return
	}
	req.ModelName = strings.TrimSpace(req.ModelName)
	if req.ModelName == "" || utf8.RuneCountInString(req.ModelName) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "model_name must contain 1 to 128 characters"})
		return
	}
	visible := false
	for _, item := range model.GetPricing() {
		if item.ModelName == req.ModelName {
			visible = true
			break
		}
	}
	if !visible {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "model is not listed in Model Square"})
		return
	}
	item, err := model.SaveModelDisplayStatus(req.ModelName, *req.Available)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "unable to save model display status"})
		return
	}
	common.ApiSuccess(c, item)
}

func pricingWithDisplayStatus(pricing []model.Pricing) ([]publicPricingModel, error) {
	statuses, err := model.GetModelDisplayStatuses()
	if err != nil {
		return nil, err
	}
	items := make([]publicPricingModel, 0, len(pricing))
	for _, item := range pricing {
		available, set := statuses[item.ModelName]
		if !set {
			available = true
		}
		items = append(items, publicPricingModel{Pricing: item, DisplayAvailable: available})
	}
	return items, nil
}
