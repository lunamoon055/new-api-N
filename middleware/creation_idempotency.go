package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const creationIdempotencyResponseLimit = 16 << 20

var creationIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,128}$`)

type creationIdempotencyWriter struct {
	gin.ResponseWriter
	body      []byte
	truncated bool
	status    int
	written   bool
	streaming bool
}

func (w *creationIdempotencyWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	if w.streaming {
		return w.ResponseWriter.Write(data)
	}
	if len(w.body)+len(data) > creationIdempotencyResponseLimit {
		w.truncated = true
		w.flushBuffered()
		return w.ResponseWriter.Write(data)
	}
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *creationIdempotencyWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func (w *creationIdempotencyWriter) WriteHeader(status int) {
	if w.written {
		return
	}
	w.status = status
	w.written = true
}

func (w *creationIdempotencyWriter) WriteHeaderNow() {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
}

func (w *creationIdempotencyWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *creationIdempotencyWriter) Size() int {
	return len(w.body)
}

func (w *creationIdempotencyWriter) Written() bool {
	return w.written
}

func (w *creationIdempotencyWriter) Flush() {
	w.flushBuffered()
	w.ResponseWriter.Flush()
}

func (w *creationIdempotencyWriter) flushBuffered() {
	if w.streaming {
		return
	}
	w.streaming = true
	w.ResponseWriter.WriteHeader(w.Status())
	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
	}
}

func creationRequestHash(c *gin.Context) (string, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return "", err
	}
	body, err := storage.Bytes()
	if err != nil {
		return "", err
	}
	if _, err := storage.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	c.Request.Body = io.NopCloser(storage)
	hash := sha256.New()
	_, _ = hash.Write([]byte(c.Request.Method))
	_, _ = hash.Write([]byte("\n"))
	_, _ = hash.Write([]byte(c.Request.URL.EscapedPath()))
	_, _ = hash.Write([]byte("?"))
	_, _ = hash.Write([]byte(c.Request.URL.RawQuery))
	_, _ = hash.Write([]byte("\n"))
	_, _ = hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func creationIdempotencyError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"type":    "idempotency_error",
			"code":    code,
			"message": message,
		},
	})
}

func replayCreationIdempotency(c *gin.Context, entry *model.CreationIdempotency) {
	if entry.ResponseTruncated {
		creationIdempotencyError(c, http.StatusConflict, "idempotency_response_too_large", "原请求已完成，但响应过大，无法通过幂等缓存重放。")
		return
	}
	var headers map[string]string
	_ = common.Unmarshal(entry.ResponseHeaders, &headers)
	for key, value := range headers {
		if value != "" {
			c.Header(key, value)
		}
	}
	c.Header("X-Idempotency-Replayed", "true")
	status := entry.ResponseStatus
	if status < 100 {
		status = http.StatusOK
	}
	c.Status(status)
	if len(entry.ResponseBody) > 0 {
		_, _ = c.Writer.Write(entry.ResponseBody)
	}
	c.Abort()
}

func waitForCreationIdempotency(tokenID int, key string) (*model.CreationIdempotency, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-time.After(50 * time.Millisecond):
			entry, err := model.GetCreationIdempotency(tokenID, key)
			if err != nil {
				return nil, err
			}
			if entry.Status == model.CreationIdempotencyCompleted {
				return entry, nil
			}
		}
	}
	return nil, nil
}

// CreationIdempotency prevents duplicate provider submissions on the new
// /v1/creation compatibility endpoints. It is deliberately optional so the
// existing /v1 clients retain their current behavior.
func CreationIdempotency() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}
		if !creationIdempotencyKeyPattern.MatchString(key) {
			creationIdempotencyError(c, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key 必须为 8 到 128 位字母、数字、冒号、下划线或短横线。")
			return
		}
		requestHash, err := creationRequestHash(c)
		if err != nil {
			creationIdempotencyError(c, http.StatusBadRequest, "invalid_request_body", "无法读取请求体。")
			return
		}
		tokenID := c.GetInt("token_id")
		userID := c.GetInt("id")
		entry, created, err := model.ReserveCreationIdempotency(tokenID, userID, key, requestHash)
		if err != nil {
			creationIdempotencyError(c, http.StatusInternalServerError, "idempotency_unavailable", "幂等服务暂时不可用。")
			return
		}
		if !created {
			if entry.RequestHash != requestHash {
				creationIdempotencyError(c, http.StatusConflict, "idempotency_conflict", "同一 Idempotency-Key 不能用于不同请求。")
				return
			}
			if entry.Status == model.CreationIdempotencyCompleted {
				replayCreationIdempotency(c, entry)
				return
			}
			entry, err = waitForCreationIdempotency(tokenID, key)
			if err != nil {
				creationIdempotencyError(c, http.StatusInternalServerError, "idempotency_unavailable", "幂等服务暂时不可用。")
				return
			}
			if entry != nil {
				replayCreationIdempotency(c, entry)
				return
			}
			c.Header("Retry-After", "2")
			creationIdempotencyError(c, http.StatusTooEarly, "idempotency_in_progress", "相同请求正在处理中，请稍后使用同一 Idempotency-Key 重试。")
			return
		}

		writer := &creationIdempotencyWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()

		if actualQuota, ok := c.Get("creation_actual_quota"); ok {
			if value, ok := actualQuota.(int); ok && value >= 0 {
				c.Header("X-Oneapi-Actual-Quota", strconv.Itoa(value))
			}
		}
		headers := map[string]string{
			"Content-Type":           c.Writer.Header().Get("Content-Type"),
			common.RequestIdKey:      c.Writer.Header().Get(common.RequestIdKey),
			"X-Oneapi-Actual-Quota":  c.Writer.Header().Get("X-Oneapi-Actual-Quota"),
			"X-New-Api-Other-Ratios": c.Writer.Header().Get("X-New-Api-Other-Ratios"),
		}
		headerJSON, _ := common.Marshal(headers)
		if err := model.CompleteCreationIdempotency(entry.ID, requestHash, c.Writer.Status(), headerJSON, writer.body, writer.truncated); err != nil {
			common.SysError("complete creation idempotency error: " + err.Error())
		}
		writer.flushBuffered()
	}
}
