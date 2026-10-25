package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func redemptionLinkTestRouter(t *testing.T) (*gin.Engine, *model.User, *model.Redemption) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousSQLite, previousRedis := common.UsingSQLite, common.RedisEnabled
	previousPayment := *operation_setting.GetPaymentSetting()
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/redemption-link.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Redemption{}, &model.Log{}))
	model.DB, model.LOG_DB = database, database
	common.UsingSQLite, common.RedisEnabled = true, false
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.UsingSQLite, common.RedisEnabled = previousSQLite, previousRedis
		*operation_setting.GetPaymentSetting() = previousPayment
	})
	user := &model.User{Id: 12341, Username: "link-user", Quota: 100, Status: common.UserStatusEnabled}
	require.NoError(t, database.Create(user).Error)
	code := &model.Redemption{Key: common.GetUUID(), Name: "private admin note", Quota: 500, Status: common.RedemptionCodeStatusEnabled}
	require.NoError(t, database.Create(code).Error)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/link", CreateRedemptionLink)
	authenticated := router.Group("/")
	authenticated.Use(func(c *gin.Context) { c.Set("id", user.Id) })
	authenticated.GET("/link", PreviewRedemptionLink)
	authenticated.POST("/redeem", RedeemRedemptionLink)
	authenticated.DELETE("/link", DismissRedemptionLink)
	return router, user, code
}

func callRedemptionLink(t *testing.T, router http.Handler, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	data, err := common.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func createTestRedemptionIntent(t *testing.T, router http.Handler, key string) (*http.Cookie, redemptionLinkIntent) {
	t.Helper()
	response := callRedemptionLink(t, router, http.MethodPost, "/link", gin.H{"key": key}, nil)
	require.Contains(t, response.Body.String(), `"success":true`)
	var cookie *http.Cookie
	for _, candidate := range response.Result().Cookies() {
		if candidate.Name == redemptionLinkCookie && candidate.Value != "" {
			cookie = candidate
		}
	}
	require.NotNil(t, cookie)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	require.NotContains(t, cookie.Value, key)
	intent, err := decodeRedemptionLink(cookie.Value)
	require.NoError(t, err)
	require.Equal(t, key, intent.Key)
	return cookie, intent
}

func TestRedemptionLinkPreviewDoesNotRedeemAndConfirmationCreditsOnce(t *testing.T) {
	router, user, code := redemptionLinkTestRouter(t)
	cookie, intent := createTestRedemptionIntent(t, router, code.Key)
	for i := 0; i < 2; i++ {
		response := callRedemptionLink(t, router, http.MethodGet, "/link", nil, cookie)
		require.Contains(t, response.Body.String(), `"quota":500`)
		require.NotContains(t, response.Body.String(), code.Key)
		require.NotContains(t, response.Body.String(), code.Name)
		require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	}
	var saved model.User
	require.NoError(t, model.DB.First(&saved, user.Id).Error)
	require.Equal(t, 100, saved.Quota)
	require.NoError(t, model.DB.First(code, code.Id).Error)
	require.Equal(t, common.RedemptionCodeStatusEnabled, code.Status)

	body := gin.H{"user_id": user.Id, "intent_id": intent.ID}
	response := callRedemptionLink(t, router, http.MethodPost, "/redeem", body, cookie)
	require.Contains(t, response.Body.String(), `"success":true`)
	require.Contains(t, response.Body.String(), `"data":500`)
	require.NoError(t, model.DB.First(&saved, user.Id).Error)
	require.Equal(t, 600, saved.Quota)
	require.NoError(t, model.DB.First(code, code.Id).Error)
	require.Equal(t, common.RedemptionCodeStatusUsed, code.Status)
	require.Equal(t, user.Id, code.UsedUserId)

	response = callRedemptionLink(t, router, http.MethodPost, "/redeem", body, cookie)
	require.Contains(t, response.Body.String(), `"success":false`)
	require.NoError(t, model.DB.First(&saved, user.Id).Error)
	require.Equal(t, 600, saved.Quota)
	var logs int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeTopup).Count(&logs).Error)
	require.EqualValues(t, 1, logs)
}

func TestRedemptionLinkRejectsChangedAccountOrChangedIntent(t *testing.T) {
	router, user, code := redemptionLinkTestRouter(t)
	cookie, intent := createTestRedemptionIntent(t, router, code.Key)
	for _, body := range []gin.H{
		{"user_id": user.Id + 1, "intent_id": intent.ID},
		{"user_id": user.Id, "intent_id": "a different tab's intent"},
	} {
		response := callRedemptionLink(t, router, http.MethodPost, "/redeem", body, cookie)
		require.Contains(t, response.Body.String(), `"success":false`)
	}
	var saved model.User
	require.NoError(t, model.DB.First(&saved, user.Id).Error)
	require.Equal(t, 100, saved.Quota)
}

func TestRedemptionLinkUsedExpiredDisabledOrMissingCode(t *testing.T) {
	for _, kind := range []string{"used", "expired", "disabled", "missing"} {
		t.Run(kind, func(t *testing.T) {
			router, _, code := redemptionLinkTestRouter(t)
			switch kind {
			case "used":
				code.Status = common.RedemptionCodeStatusUsed
			case "disabled":
				code.Status = common.RedemptionCodeStatusDisabled
			case "expired":
				code.ExpiredTime = common.GetTimestamp() - 1
			case "missing":
				require.NoError(t, model.DB.Delete(code).Error)
			}
			if kind != "missing" {
				require.NoError(t, model.DB.Save(code).Error)
			}
			cookie, _ := createTestRedemptionIntent(t, router, code.Key)
			response := callRedemptionLink(t, router, http.MethodGet, "/link", nil, cookie)
			require.Contains(t, response.Body.String(), `"success":false`)
		})
	}
}

func TestRedemptionLinkEncryptionExpiryDismissalAndCompliance(t *testing.T) {
	router, user, code := redemptionLinkTestRouter(t)
	cookie, intent := createTestRedemptionIntent(t, router, code.Key)
	_, err := decodeRedemptionLink(cookie.Value[:len(cookie.Value)-4] + "AAAA")
	require.Error(t, err)
	intent.ExpiresAt = time.Now().Add(-time.Second).Unix()
	expired, err := encodeRedemptionLink(intent)
	require.NoError(t, err)
	_, err = decodeRedemptionLink(expired)
	require.Error(t, err)
	response := callRedemptionLink(t, router, http.MethodDelete, "/link", nil, cookie)
	for _, cookie := range response.Result().Cookies() {
		require.Less(t, cookie.MaxAge, 0)
	}
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		path := "/link"
		if method == http.MethodPost {
			path = "/redeem"
		}
		response = callRedemptionLink(t, router, method, path, gin.H{"user_id": user.Id, "intent_id": intent.ID}, cookie)
		require.Contains(t, response.Body.String(), `"success":false`)
	}
	request := httptest.NewRequest(http.MethodPost, "/link", strings.NewReader(`{"key":"demo"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-Proto", "https")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	for _, cookie := range response.Result().Cookies() {
		require.True(t, cookie.Secure)
	}
	request = httptest.NewRequest(http.MethodPost, "/link", strings.NewReader(`{"key":"demo"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Contains(t, response.Body.String(), `"success":false`)
	require.Empty(t, response.Result().Cookies(), "a cross-site request must not replace a pending intent")
}
