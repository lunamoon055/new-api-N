package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCreationIdempotencyRouter(t *testing.T, calls *atomic.Int32) *gin.Engine {
	t.Helper()
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/creation-idempotency.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CreationIdempotency{}))
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/v1/creation/images/generations", func(c *gin.Context) {
		c.Set("id", 11)
		c.Set("token_id", 22)
		c.Next()
	}, CreationIdempotency(), func(c *gin.Context) {
		calls.Add(1)
		c.JSON(http.StatusAccepted, gin.H{"task_id": "task_once"})
		// Simulate quota being known only after the provider body was produced.
		c.Set("creation_actual_quota", 321)
	})
	return engine
}

func performCreationIdempotencyRequest(engine http.Handler, key, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/creation/images/generations", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestCreationIdempotencyReplaysSameRequest(t *testing.T) {
	var calls atomic.Int32
	engine := setupCreationIdempotencyRouter(t, &calls)

	first := performCreationIdempotencyRequest(engine, "generation:same", `{"prompt":"one"}`)
	second := performCreationIdempotencyRequest(engine, "generation:same", `{"prompt":"one"}`)

	require.Equal(t, http.StatusAccepted, first.Code)
	require.Equal(t, http.StatusAccepted, second.Code)
	require.JSONEq(t, first.Body.String(), second.Body.String())
	require.Equal(t, "321", first.Header().Get("X-Oneapi-Actual-Quota"))
	require.Equal(t, "321", second.Header().Get("X-Oneapi-Actual-Quota"))
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	require.EqualValues(t, 1, calls.Load())
}

func TestCreationIdempotencyRejectsDifferentRequest(t *testing.T) {
	var calls atomic.Int32
	engine := setupCreationIdempotencyRouter(t, &calls)

	first := performCreationIdempotencyRequest(engine, "generation:conflict", `{"prompt":"one"}`)
	second := performCreationIdempotencyRequest(engine, "generation:conflict", `{"prompt":"two"}`)

	require.Equal(t, http.StatusAccepted, first.Code)
	require.Equal(t, http.StatusConflict, second.Code)
	require.Contains(t, second.Body.String(), "idempotency_conflict")
	require.EqualValues(t, 1, calls.Load())
}

func TestImageEditResponseCanBeReadWithoutRepeatingSubmission(t *testing.T) {
	var calls atomic.Int32
	engine := setupCreationIdempotencyRouter(t, &calls)
	const path = "/v1/images/edits"
	const key = "vw-edit-297510ea-805c-4b0c-a5d7-8f93a626219c"
	const result = `{"data":[{"b64_json":"aW1hZ2U="}]}`
	entered := make(chan struct{})
	release := make(chan struct{}, 1)
	defer close(release)
	auth := func(c *gin.Context) {
		tokenID := 22
		if c.GetHeader("Authorization") == "Bearer other" {
			tokenID = 23
		}
		c.Set("token_id", tokenID)
		c.Set("id", 11)
		c.Next()
	}
	engine.POST(path, auth, ImageEditIdempotency(), func(c *gin.Context) {
		calls.Add(1)
		close(entered)
		<-release
		c.Data(http.StatusOK, "application/json", []byte(result))
		c.Writer.Flush()
	})
	engine.GET(path+"/requests/:request_key", auth, GetImageEditResponse)
	query := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, path+"/requests/"+key, nil)
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		return response
	}
	require.Equal(t, http.StatusNotFound, query("Bearer own").Code)
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("reference-image"))
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(response, request)
		close(done)
	}()
	<-entered
	processingStatus := query("Bearer own").Code
	otherTokenStatus := query("Bearer other").Code
	release <- struct{}{}
	<-done
	require.Equal(t, http.StatusTooEarly, processingStatus)
	require.Equal(t, http.StatusNotFound, otherTokenStatus)
	recovered := query("Bearer own")
	require.Equal(t, http.StatusOK, recovered.Code)
	require.JSONEq(t, result, recovered.Body.String())
	require.Equal(t, "no-store", recovered.Header().Get("Cache-Control"))
	require.Equal(t, http.StatusNotFound, query("Bearer other").Code)
	require.EqualValues(t, 1, calls.Load())
	_, err := model.GetCreationIdempotency(22, key)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = model.GetCreationIdempotency(22, imageEditRequestNamespace+key)
	require.NoError(t, err)
}
