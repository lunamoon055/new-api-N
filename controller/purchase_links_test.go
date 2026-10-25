package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPurchaseLinksPersistAndAppearInTopupInfo(t *testing.T) {
	redemptionLinkTestRouter(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Option{}))
	oldOptions, oldLegacyLink := common.OptionMap, common.TopUpLink
	common.OptionMap = make(map[string]string)
	common.TopUpLink = "https://shop.example/legacy"
	t.Cleanup(func() { common.OptionMap, common.TopUpLink = oldOptions, oldLegacyLink })

	links := operation_setting.GetPurchaseLinks()
	require.Len(t, links, 6)
	require.Equal(t, common.TopUpLink, links["other"])
	require.Empty(t, links["10"])

	raw := `{"10":"https://shop.example/ten","20":"https://shop.example/twenty","50":"https://shop.example/fifty","100":"https://shop.example/hundred","200":"https://shop.example/two-hundred","other":""}`
	require.NoError(t, model.UpdateOption(operation_setting.PurchaseLinksOptionKey, raw))
	var persisted model.Option
	require.NoError(t, model.DB.Where(&model.Option{Key: operation_setting.PurchaseLinksOptionKey}).First(&persisted).Error)
	require.NotEmpty(t, persisted.Value)

	router := gin.New()
	router.GET("/info", GetTopUpInfo)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/info", nil))
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			PurchaseLinks map[string]string `json:"purchase_links"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	require.True(t, result.Success)
	require.Len(t, result.Data.PurchaseLinks, 6)
	require.Equal(t, "https://shop.example/twenty", result.Data.PurchaseLinks["20"])
	require.Empty(t, result.Data.PurchaseLinks["other"], "an explicitly disabled Other tier must not fall back to the legacy URL")

	require.Error(t, model.UpdateOption(operation_setting.PurchaseLinksOptionKey, `{"10":"javascript:alert(1)"}`))
	var unchanged model.Option
	require.NoError(t, model.DB.Where(&model.Option{Key: operation_setting.PurchaseLinksOptionKey}).First(&unchanged).Error)
	require.Equal(t, persisted.Value, unchanged.Value)
	require.Equal(t, "https://shop.example/ten", operation_setting.GetPurchaseLinks()["10"])
}

func TestPurchaseRedemptionRejectsChangedAccountBeforeConsumingCode(t *testing.T) {
	_, user, code := redemptionLinkTestRouter(t)
	router := gin.New()
	router.POST("/redeem", func(c *gin.Context) { c.Set("id", user.Id) }, TopUp)
	response := callRedemptionLink(t, router, "POST", "/redeem", gin.H{"key": code.Key, "expected_user_id": user.Id + 1}, nil)
	require.Contains(t, response.Body.String(), `"success":false`)
	require.Contains(t, response.Body.String(), "账号已变化")
	var unchanged model.User
	require.NoError(t, model.DB.First(&unchanged, user.Id).Error)
	require.Equal(t, user.Quota, unchanged.Quota)
	response = callRedemptionLink(t, router, "POST", "/redeem", gin.H{"key": code.Key, "expected_user_id": user.Id}, nil)
	require.Contains(t, response.Body.String(), `"success":true`)
	require.NoError(t, model.DB.First(&unchanged, user.Id).Error)
	require.Equal(t, user.Quota+code.Quota, unchanged.Quota)
}
