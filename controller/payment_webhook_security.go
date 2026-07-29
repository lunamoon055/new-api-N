package controller

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const paymentWebhookMaxBodyBytes int64 = 1 << 20

func paymentWebhookPath(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return c.Request.URL.Path
}

func limitPaymentWebhookBody(c *gin.Context) error {
	if c == nil || c.Request == nil {
		return fmt.Errorf("invalid webhook request")
	}
	if c.Request.ContentLength > paymentWebhookMaxBodyBytes {
		return fmt.Errorf("%w: maximum is %d bytes", common.ErrRequestBodyTooLarge, paymentWebhookMaxBodyBytes)
	}
	if c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, paymentWebhookMaxBodyBytes)
	}
	return nil
}

func readPaymentWebhookBody(c *gin.Context) ([]byte, error) {
	if err := limitPaymentWebhookBody(c); err != nil {
		return nil, err
	}
	if c.Request.Body == nil {
		return []byte{}, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func paymentWebhookReadErrorStatus(err error) int {
	if common.IsRequestBodyTooLargeError(err) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func paymentCallbackParams(c *gin.Context) (map[string]string, error) {
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("invalid payment callback request")
	}
	values := c.Request.URL.Query()
	if c.Request.Method == http.MethodPost {
		if err := limitPaymentWebhookBody(c); err != nil {
			return nil, err
		}
		if err := c.Request.ParseForm(); err != nil {
			return nil, err
		}
		values = c.Request.PostForm
	}

	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	return params, nil
}
