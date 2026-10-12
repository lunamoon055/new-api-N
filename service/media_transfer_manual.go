package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

const MaxManualMediaUploadBytes = mediaStorageMaxBytes

var ErrInvalidManualMediaInput = errors.New("请选择已启用的存储渠道，并提供一个有效的 HTTP 下载链接或 MP4/WebM 视频（最大 200 MB）")

// QueueManualMediaTransfer changes only delivery of an already-generated task.
// Credentials from its original upstream are never reused for a replacement URL.
func QueueManualMediaTransfer(id int64, providerID, sourceURL string, file io.Reader, filename string) (bool, error) {
	providerID = strings.TrimSpace(providerID)
	sourceURL = strings.TrimSpace(sourceURL)
	providerValid := false
	for _, provider := range GetMediaStorageProviders() {
		if provider.ID == providerID && provider.Enabled {
			providerValid = true
			break
		}
	}
	if !providerValid || (file == nil) == (sourceURL == "") {
		return false, ErrInvalidManualMediaInput
	}
	var current model.MediaTransferJob
	if err := model.DB.First(&current, id).Error; err != nil {
		return false, err
	}
	if current.Status != model.MediaTransferRetry && current.Status != model.MediaTransferFailed {
		return false, nil
	}
	input := &model.MediaTransferJob{RequestedProvider: providerID}
	var stagedPath string
	if file != nil {
		staged, err := stageManualMediaFile(file, filename)
		if err != nil {
			return false, err
		}
		input.SpoolFile, input.ByteSize, input.SHA256 = staged.SpoolFile, staged.ByteSize, staged.SHA256
		input.FileName, input.ContentType, input.ManualFile = staged.FileName, staged.ContentType, true
		stagedPath = mediaTransferJobFilePath(input)
	} else {
		parsed, err := url.Parse(sourceURL)
		if err != nil || parsed.Hostname() == "" || parsed.User != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") || len(sourceURL) > 8192 {
			return false, ErrInvalidManualMediaInput
		}
		input.SourceCiphertext, err = encryptMediaTransferValue(sourceURL)
		if err != nil {
			return false, err
		}
		input.OptionsCiphertext, err = encryptMediaTransferValue("{}")
		if err != nil {
			return false, err
		}
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return false, err
		}
		input.SpoolFile = fmt.Sprintf("%d-manual-%s.media", id, hex.EncodeToString(nonce[:]))
	}
	won, err := model.ReplaceMediaTransferInput(id, input)
	if err != nil || !won {
		if stagedPath != "" {
			_ = os.Remove(stagedPath)
		}
		return false, err
	}
	WakeMediaTransferWorker()
	return true, nil
}

func stageManualMediaFile(reader io.Reader, filename string) (*model.MediaTransferJob, error) {
	var header [512]byte
	n, err := io.ReadFull(reader, header[:])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	contentType := http.DetectContentType(header[:n])
	if contentType != "video/mp4" && contentType != "video/webm" {
		return nil, ErrInvalidManualMediaInput
	}
	if err := os.MkdirAll(mediaTransferSpoolDir(), 0700); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(mediaTransferSpoolDir(), "manual-*.media")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(file.Name())
		}
	}()
	hash := sha256.New()
	if _, err := io.MultiWriter(file, hash).Write(header[:n]); err != nil {
		return nil, err
	}
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(reader, mediaStorageMaxBytes-int64(n)+1))
	if err != nil {
		return nil, err
	}
	size += int64(n)
	if size == 0 || size > mediaStorageMaxBytes {
		return nil, ErrInvalidManualMediaInput
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if filename == "" || filename == "." || len(filename) > 255 || strings.ContainsAny(filename, "\r\n") {
		filename = "manual-video" + mediaStorageExtension(contentType, "video/mp4")
	}
	keep = true
	return &model.MediaTransferJob{
		SpoolFile: filepath.Base(file.Name()), ByteSize: size, SHA256: hex.EncodeToString(hash.Sum(nil)),
		FileName: filename, ContentType: contentType, ManualFile: true,
	}, nil
}
