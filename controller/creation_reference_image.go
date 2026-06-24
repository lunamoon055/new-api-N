package controller

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	creationReferenceImageFormField = "image"
	creationReferenceImageMaxBytes  = 20 << 20
	creationReferenceImageDir       = "new-api-creation-reference-images"
)

var creationReferenceImageExtensions = map[string]string{
	"image/avif": ".avif",
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func UploadCreationReferenceImage(c *gin.Context) {
	fileHeader, err := c.FormFile(creationReferenceImageFormField)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "image is required",
		})
		return
	}
	if fileHeader.Size > creationReferenceImageMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "image must not exceed 20 MB",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "unable to read image",
		})
		return
	}
	defer file.Close()

	mimeType, data, err := readCreationReferenceImage(file, fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	token, err := randomCreationReferenceImageToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "unable to prepare image",
		})
		return
	}
	extension := creationReferenceImageExtensions[mimeType]
	filename := token + extension
	dir := creationReferenceImageStorageDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "unable to store image",
		})
		return
	}
	cleanupExpiredCreationReferenceImages(dir)
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "unable to store image",
		})
		return
	}

	common.ApiSuccess(c, gin.H{
		"url": buildCreationReferenceImageURL(c, filename),
	})
}

func GetCreationReferenceImage(c *gin.Context) {
	filename := c.Param("filename")
	if !isSafeCreationReferenceImageName(filename) {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(creationReferenceImageStorageDir(), filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	extension := strings.ToLower(filepath.Ext(filename))
	mimeType := "application/octet-stream"
	for candidate, candidateExtension := range creationReferenceImageExtensions {
		if candidateExtension == extension {
			mimeType = candidate
			break
		}
	}
	file, err := os.Open(path)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()
	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(http.StatusOK, info.Size(), mimeType, file, nil)
}

func readCreationReferenceImage(
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (string, []byte, error) {
	reader := io.LimitReader(file, creationReferenceImageMaxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read image")
	}
	if len(data) == 0 {
		return "", nil, fmt.Errorf("image is empty")
	}
	if len(data) > creationReferenceImageMaxBytes {
		return "", nil, fmt.Errorf("image must not exceed 20 MB")
	}

	mimeType := http.DetectContentType(data)
	if _, ok := creationReferenceImageExtensions[mimeType]; ok {
		return mimeType, data, nil
	}
	if fallback := creationReferenceImageMimeFromName(fileHeader.Filename); fallback != "" {
		if _, ok := creationReferenceImageExtensions[fallback]; ok {
			return fallback, data, nil
		}
	}
	return "", nil, fmt.Errorf("image must be PNG, JPEG, WebP, GIF, or AVIF")
}

func creationReferenceImageMimeFromName(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func randomCreationReferenceImageToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(bytes)), nil
}

func creationReferenceImageStorageDir() string {
	return filepath.Join(os.TempDir(), creationReferenceImageDir)
}

func cleanupExpiredCreationReferenceImages(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() || !isSafeCreationReferenceImageName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

func isSafeCreationReferenceImageName(filename string) bool {
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return false
	}
	extension := strings.ToLower(filepath.Ext(filename))
	if extension == "" {
		return false
	}
	for _, allowed := range creationReferenceImageExtensions {
		if extension == allowed {
			return true
		}
	}
	return false
}

func buildCreationReferenceImageURL(c *gin.Context, filename string) string {
	return fmt.Sprintf("%s/api/creation/reference-images/%s", buildCreationReferencePublicBaseURL(c), filename)
}

func cleanupCreationReferenceImageForTest(filename string) error {
	if !isSafeCreationReferenceImageName(filename) {
		return errors.New("invalid filename")
	}
	return os.Remove(filepath.Join(creationReferenceImageStorageDir(), filename))
}
