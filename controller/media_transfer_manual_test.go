package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestManualMediaTransferRejectsInvalidMultipartInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/transfers/:id/manual", ManualMediaTransfer)
	for _, id := range []string{"invalid", "405"} {
		t.Run(id, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/transfers/"+id+"/manual", bytes.NewBufferString(`{"source_url":"https://example.com/video.mp4"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestManualMediaTransferRejectsMultipleFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/transfers/:id/manual", ManualMediaTransfer)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for range 2 {
		file, err := writer.CreateFormFile("file", "video.mp4")
		require.NoError(t, err)
		_, _ = file.Write([]byte("video"))
	}
	require.NoError(t, writer.Close())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/transfers/405/manual", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
