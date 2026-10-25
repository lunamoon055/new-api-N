package controller

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const redemptionLinkCookie = "redemption_link"
const redemptionLinkMarker = "redemption_pending"
const redemptionLinkLifetime = 30 * time.Minute

type redemptionLinkIntent struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	ExpiresAt int64  `json:"expires_at"`
}

func redemptionLinkCipher() (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("redemption-link:v1:" + common.SessionSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encodeRedemptionLink(intent redemptionLinkIntent) (string, error) {
	aead, err := redemptionLinkCipher()
	if err != nil {
		return "", err
	}
	data, err := common.Marshal(intent)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, data, []byte(redemptionLinkCookie))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decodeRedemptionLink(value string) (redemptionLinkIntent, error) {
	var intent redemptionLinkIntent
	aead, err := redemptionLinkCipher()
	if err != nil {
		return intent, err
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(data) < aead.NonceSize()+aead.Overhead() {
		return intent, errors.New("兑换链接无效，请重新打开购买后收到的链接")
	}
	data, err = aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(redemptionLinkCookie))
	if err != nil {
		return intent, errors.New("兑换链接无效，请重新打开购买后收到的链接")
	}
	if err = common.Unmarshal(data, &intent); err != nil || intent.ID == "" || intent.Key == "" || intent.ExpiresAt <= common.GetTimestamp() {
		return redemptionLinkIntent{}, errors.New("兑换链接已超时，请重新打开购买后收到的链接")
	}
	return intent, nil
}

func setRedemptionLinkCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	forwardedProto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	secure := c.Request.TLS != nil || strings.EqualFold(forwardedProto, "https")
	c.SetCookie(name, value, maxAge, "/", "", secure, httpOnly)
}

func clearRedemptionLink(c *gin.Context) {
	setRedemptionLinkCookie(c, redemptionLinkCookie, "", -1, true)
	setRedemptionLinkCookie(c, redemptionLinkMarker, "", -1, false)
}

// CreateRedemptionLink keeps the code out of browser storage and auth redirect URLs.
// The authenticated redemption endpoint is the only operation that consumes it.
func CreateRedemptionLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
		common.ApiError(c, errors.New("请在本站打开购买后收到的兑换链接"))
		return
	}
	clearRedemptionLink(c)
	var request struct {
		Key string `json:"key" binding:"required,max=128"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Key) == "" {
		common.ApiError(c, errors.New("兑换链接无效，请检查购买后收到的完整链接"))
		return
	}
	value, err := encodeRedemptionLink(redemptionLinkIntent{
		ID: common.GetUUID(), Key: strings.TrimSpace(request.Key), ExpiresAt: time.Now().Add(redemptionLinkLifetime).Unix(),
	})
	if err != nil {
		common.ApiError(c, errors.New("暂时无法打开兑换链接，请稍后重试"))
		return
	}
	maxAge := int(redemptionLinkLifetime.Seconds())
	setRedemptionLinkCookie(c, redemptionLinkCookie, value, maxAge, true)
	// This non-secret marker lets any completed login resume at the wallet.
	setRedemptionLinkCookie(c, redemptionLinkMarker, "1", maxAge, false)
	common.ApiSuccess(c, nil)
}

func readRedemptionLink(c *gin.Context) (redemptionLinkIntent, error) {
	value, err := c.Cookie(redemptionLinkCookie)
	if err != nil {
		return redemptionLinkIntent{}, errors.New("兑换链接已超时，请重新打开购买后收到的链接")
	}
	return decodeRedemptionLink(value)
}

func PreviewRedemptionLink(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}
	intent, err := readRedemptionLink(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	quota, err := model.PreviewRedemption(intent.Key)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"quota": quota, "user_id": c.GetInt("id"), "intent_id": intent.ID})
}

func RedeemRedemptionLink(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}
	var request struct {
		UserID   int    `json:"user_id" binding:"required"`
		IntentID string `json:"intent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.UserID != c.GetInt("id") {
		common.ApiError(c, errors.New("当前登录账号已改变，请刷新后重新确认兑换账号"))
		return
	}
	intent, err := readRedemptionLink(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if request.IntentID != intent.ID {
		common.ApiError(c, errors.New("待兑换链接已改变，请刷新后重新确认兑换信息"))
		return
	}
	lock := getTopUpLock(request.UserID)
	if !lock.TryLock() {
		common.ApiErrorI18n(c, i18n.MsgUserTopUpProcessing)
		return
	}
	defer lock.Unlock()
	quota, err := model.Redeem(intent.Key, request.UserID)
	if err != nil {
		common.ApiError(c, errors.New("兑换未完成，请刷新核对兑换码是否已使用或过期"))
		return
	}
	clearRedemptionLink(c)
	common.ApiSuccess(c, quota)
}

func DismissRedemptionLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	clearRedemptionLink(c)
	common.ApiSuccess(c, nil)
}
