package controller

import (
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// GetModelPricing is a read-only API-key surface independent of dashboard
// navigation visibility. It covers every modality while honoring token scope.
func GetModelPricing(c *gin.Context) {
	currency := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("currency", "USD")))
	rate := 1.0
	switch currency {
	case "USD":
	case "CNY":
		rate = operation_setting.USDExchangeRate
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_currency", "message": "currency must be USD or CNY"}})
		return
	}
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "invalid_exchange_rate", "message": "site currency exchange rate is not configured"}})
		return
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "invalid_billing_config", "message": "site quota conversion is not configured"}})
		return
	}
	groups, err := getModelListGroups(c)
	if err != nil {
		common.SysLog("get pricing groups: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "pricing_unavailable", "message": "unable to read pricing"}})
		return
	}
	groupRatios := make(map[string]float64, len(groups.ownerGroups))
	for _, group := range groups.ownerGroups {
		groupRatios[group] = service.GetUserGroupRatio(groups.userGroup, group)
	}
	limited := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	limits, _ := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	allowedModels, _ := limits.(map[string]bool)
	requestedModel := c.Query("model")
	items := make([]service.DownstreamModelPricing, 0)
	for _, item := range model.GetPricing() {
		if requestedModel != "" && item.ModelName != requestedModel {
			continue
		}
		if limited {
			if _, ok := allowedModels[ratio_setting.FormatMatchingModelName(item.ModelName)]; !ok {
				continue
			}
		}
		enabledGroups := make([]string, 0, len(groups.ownerGroups))
		for _, group := range groups.ownerGroups {
			if common.StringsContains(item.EnableGroup, group) || common.StringsContains(item.EnableGroup, "all") {
				enabledGroups = append(enabledGroups, group)
			}
		}
		if len(enabledGroups) == 0 {
			continue
		}
		item.EnableGroup = enabledGroups
		items = append(items, service.BuildDownstreamModelPricing(item, rate))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ModelName < items[j].ModelName })
	// Responses depend on token and user group; shared caches must not mix them.
	c.Header("Cache-Control", "private, no-store")
	c.Header("Vary", "Authorization")
	c.JSON(http.StatusOK, gin.H{
		"success": true, "object": "list", "data": items,
		"currency": currency, "base_currency": "USD", "exchange_rate": rate,
		"group_ratio": groupRatios, "token_group": groups.tokenGroup,
		"schema_version": "1", "price_scope": "base",
	})
}
