package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPaymentWebhookTestContext(method string, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	return context, recorder
}

func TestReadPaymentWebhookBodyRejectsOversizedContentLength(t *testing.T) {
	context, _ := newPaymentWebhookTestContext(
		http.MethodPost,
		"/api/stripe/webhook",
		make([]byte, paymentWebhookMaxBodyBytes+1),
	)

	_, err := readPaymentWebhookBody(context)
	require.Error(t, err)
	assert.True(t, common.IsRequestBodyTooLargeError(err))
}

func TestReadPaymentWebhookBodyRejectsOversizedChunkedBody(t *testing.T) {
	context, _ := newPaymentWebhookTestContext(
		http.MethodPost,
		"/api/creem/webhook",
		make([]byte, paymentWebhookMaxBodyBytes+1),
	)
	context.Request.ContentLength = -1

	_, err := readPaymentWebhookBody(context)
	require.Error(t, err)
	assert.True(t, common.IsRequestBodyTooLargeError(err))
}

func TestPaymentCallbackParamsLimitsFormBody(t *testing.T) {
	body := []byte("value=" + strings.Repeat("x", int(paymentWebhookMaxBodyBytes)))
	context, _ := newPaymentWebhookTestContext(
		http.MethodPost,
		"/api/user/epay/notify",
		body,
	)
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := paymentCallbackParams(context)
	require.Error(t, err)
	assert.True(t, common.IsRequestBodyTooLargeError(err))
}

func TestPaymentWebhookPathExcludesQuerySecrets(t *testing.T) {
	context, _ := newPaymentWebhookTestContext(
		http.MethodGet,
		"/api/user/epay/notify?sign=secret&trade_no=123",
		nil,
	)

	assert.Equal(t, "/api/user/epay/notify", paymentWebhookPath(context))
}
