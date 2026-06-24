package controller

import (
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

const (
	creationReferenceFileFormField = "file"
	creationReferenceFileKindField = "kind"
	creationReferenceFileDir       = "creation-reference-files"
)

type creationReferenceFileType struct {
	kind       string
	maxBytes   int64
	extensions map[string]string
}

var creationReferenceFileTypes = map[string]creationReferenceFileType{
	"image": {
		kind:       "image",
		maxBytes:   20 << 20,
		extensions: creationReferenceImageExtensions,
	},
	"video": {
		kind:     "video",
		maxBytes: 200 << 20,
		extensions: map[string]string{
			"video/mp4": ".mp4",
		},
	},
	"audio": {
		kind:     "audio",
		maxBytes: 15 << 20,
		extensions: map[string]string{
			"audio/mp3":   ".mp3",
			"audio/mpeg":  ".mp3",
			"audio/wav":   ".wav",
			"audio/wave":  ".wav",
			"audio/x-wav": ".wav",
		},
	},
}

func UploadCreationReferenceFile(c *gin.Context) {
	fileType, ok := creationReferenceFileTypes[strings.ToLower(strings.TrimSpace(c.PostForm(creationReferenceFileKindField)))]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "reference file kind must be image, video, or audio",
		})
		return
	}

	fileHeader, err := c.FormFile(creationReferenceFileFormField)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "reference file is required",
		})
		return
	}
	if fileHeader.Size > fileType.maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("%s reference file is too large", fileType.kind),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "unable to read reference file",
		})
		return
	}
	defer file.Close()

	mimeType, data, err := readCreationReferenceFile(file, fileHeader, fileType)
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
			"message": "unable to prepare reference file",
		})
		return
	}
	filename := token + fileType.extensions[mimeType]
	dir := creationReferenceFileStorageDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "unable to store reference file",
		})
		return
	}
	cleanupExpiredCreationReferenceFiles(dir)
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "unable to store reference file",
		})
		return
	}

	common.ApiSuccess(c, gin.H{
		"kind":      fileType.kind,
		"mime_type": mimeType,
		"url":       buildCreationReferenceFileURL(c, filename),
	})
}

func GetCreationReferenceFile(c *gin.Context) {
	filename := c.Param("filename")
	if !isSafeCreationReferenceFileName(filename) {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(creationReferenceFileStorageDir(), filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	mimeType := creationReferenceFileMimeFromName(filename)
	if mimeType == "" {
		c.Status(http.StatusNotFound)
		return
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

func readCreationReferenceFile(
	file multipart.File,
	fileHeader *multipart.FileHeader,
	fileType creationReferenceFileType,
) (string, []byte, error) {
	reader := io.LimitReader(file, fileType.maxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read reference file")
	}
	if len(data) == 0 {
		return "", nil, fmt.Errorf("%s reference file is empty", fileType.kind)
	}
	if int64(len(data)) > fileType.maxBytes {
		return "", nil, fmt.Errorf("%s reference file is too large", fileType.kind)
	}

	mimeType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if index := strings.Index(mimeType, ";"); index >= 0 {
		mimeType = strings.TrimSpace(mimeType[:index])
	}
	if _, ok := fileType.extensions[mimeType]; ok {
		return mimeType, data, nil
	}

	detected := http.DetectContentType(data)
	if _, ok := fileType.extensions[detected]; ok {
		return detected, data, nil
	}

	if fallback := creationReferenceFileMimeFromName(fileHeader.Filename); fallback != "" {
		if _, ok := fileType.extensions[fallback]; ok {
			return fallback, data, nil
		}
	}
	return "", nil, fmt.Errorf("%s reference file format is unsupported", fileType.kind)
}

func creationReferenceFileMimeFromName(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}

func creationReferenceFileStorageDir() string {
	return filepath.Join(os.TempDir(), creationReferenceFileDir)
}

func cleanupExpiredCreationReferenceFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() || !isSafeCreationReferenceFileName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

func isSafeCreationReferenceFileName(filename string) bool {
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return false
	}
	return creationReferenceFileMimeFromName(filename) != ""
}

func buildCreationReferenceFileURL(c *gin.Context, filename string) string {
	return fmt.Sprintf("%s/api/creation/reference-files/%s", buildCreationReferencePublicBaseURL(c), filename)
}

func buildCreationReferencePublicBaseURL(c *gin.Context) string {
	candidates := []string{
		strings.TrimSpace(os.Getenv("CREATION_REFERENCE_PUBLIC_BASE_URL")),
		creationReferenceOriginBase(c.GetHeader("Origin")),
		creationReferenceOriginBase(c.GetHeader("Referer")),
		creationReferenceForwardedBaseURL(c),
		creationReferenceRequestBaseURL(c),
		strings.TrimSpace(system_setting.ServerAddress),
	}
	for _, candidate := range candidates {
		if base := normalizeCreationReferenceBaseURL(candidate); base != "" && !isLocalCreationReferenceBaseURL(base) {
			return base
		}
	}
	for _, candidate := range candidates {
		if base := normalizeCreationReferenceBaseURL(candidate); base != "" {
			return base
		}
	}
	return "http://localhost:3000"
}

func creationReferenceRequestBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil || strings.TrimSpace(c.Request.Host) == "" {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := firstCreationReferenceHeaderValue(c.GetHeader("X-Forwarded-Proto")); forwardedProto == "http" || forwardedProto == "https" {
		scheme = forwardedProto
	}
	host := firstCreationReferenceHeaderValue(c.Request.Host)
	return fmt.Sprintf("%s://%s", scheme, host)
}

func creationReferenceOriginBase(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func creationReferenceForwardedBaseURL(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if base := creationReferenceForwardedHeaderBase(c.GetHeader("Forwarded")); base != "" {
		return base
	}
	host := firstCreationReferenceHeaderValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		return ""
	}
	proto := firstCreationReferenceHeaderValue(c.GetHeader("X-Forwarded-Proto"))
	if proto != "http" && proto != "https" {
		if strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Ssl")), "on") {
			proto = "https"
		} else {
			proto = "https"
		}
	}
	return fmt.Sprintf("%s://%s", proto, host)
}

func creationReferenceForwardedHeaderBase(value string) string {
	first := firstCreationReferenceHeaderValue(value)
	if first == "" {
		return ""
	}
	var proto, host string
	for _, part := range strings.Split(first, ";") {
		key, rawValue, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		normalizedValue := strings.Trim(strings.TrimSpace(rawValue), `"`)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "proto":
			proto = normalizedValue
		case "host":
			host = normalizedValue
		}
	}
	if (proto != "http" && proto != "https") || host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", proto, host)
}

func firstCreationReferenceHeaderValue(value string) string {
	first, _, _ := strings.Cut(value, ",")
	return strings.Trim(strings.TrimSpace(first), `"`)
}

func normalizeCreationReferenceBaseURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func isLocalCreationReferenceBaseURL(base string) bool {
	parsed, err := url.Parse(base)
	if err != nil {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate()
	}
	return false
}

func cleanupCreationReferenceFileForTest(filename string) error {
	if !isSafeCreationReferenceFileName(filename) {
		return fmt.Errorf("invalid filename")
	}
	return os.Remove(filepath.Join(creationReferenceFileStorageDir(), filename))
}
