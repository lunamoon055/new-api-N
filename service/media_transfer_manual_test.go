package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func manualTestVideo() []byte {
	body := make([]byte, 32)
	binary.BigEndian.PutUint32(body, 24)
	copy(body[4:], "ftypisom")
	copy(body[16:], "isommp42")
	copy(body[28:], "mdat")
	return body
}

func seedManualTransfer(t *testing.T) (*model.Task, *model.MediaTransferJob) {
	t.Helper()
	task := &model.Task{
		TaskID: "task_manual_" + t.Name(), Status: model.TaskStatusInProgress,
		MediaTransferState: string(model.MediaTransferFailed),
		BillingStatus:      model.TaskBillingStatusSettled, Quota: 12345,
		CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, model.DB.Create(task).Error)
	job := &model.MediaTransferJob{
		TaskRecordID: task.ID, SourceCiphertext: "old-broken-source", OptionsCiphertext: "old-private-options",
		ResponseCiphertext: "old-response", CandidateURLCiphertext: "old-candidate", CandidateProvider: "old",
		Status: model.MediaTransferFailed, Attempts: 8, LastError: "transfer_failed",
		CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, model.DB.Create(job).Error)
	t.Cleanup(func() { model.DB.Delete(job); model.DB.Delete(task) })
	return task, job
}

func manualTestSettings(t *testing.T, providers []MediaStorageProvider) {
	t.Helper()
	data, err := common.Marshal(providers)
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[MediaStorageOptionKey]
	common.OptionMap[MediaStorageOptionKey] = string(data)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[MediaStorageOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})
	t.Setenv("MEDIA_SPOOL_DIR", t.TempDir())
	t.Setenv("CRYPTO_SECRET", "manual-transfer-test")
	previousSecret := common.CryptoSecret
	common.CryptoSecret = "manual-transfer-test"
	t.Cleanup(func() { common.CryptoSecret = previousSecret })
}

func TestManualVideoUploadCompletesSameTaskWithoutBillingOrSourceDownload(t *testing.T) {
	video := manualTestVideo()
	uploads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			uploads++
			require.Equal(t, "Bearer storage-secret", r.Header.Get("Authorization"))
			require.NoError(t, r.ParseMultipartForm(1024))
			defer r.MultipartForm.RemoveAll()
			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			data, err := io.ReadAll(file)
			require.NoError(t, err)
			require.Equal(t, video, data)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"src":"/video.mp4"}]`)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write(video)
	}))
	defer server.Close()
	fetch := system_setting.GetFetchSetting()
	previous := *fetch
	fetch.EnableSSRFProtection = false
	t.Cleanup(func() { *fetch = previous })
	manualTestSettings(t, []MediaStorageProvider{
		{ID: "not-selected", Enabled: true, UploadURL: "https://must-not-be-called.invalid/upload"},
		{ID: "selected", Enabled: true, UploadURL: server.URL + "/upload", Token: "storage-secret", AuthPrefix: "Bearer "},
	})
	task, job := seedManualTransfer(t)
	won, err := QueueManualMediaTransfer(job.ID, "selected", "", bytes.NewReader(video), "cloud.mp4")
	require.NoError(t, err)
	require.True(t, won)
	var queued model.MediaTransferJob
	require.NoError(t, model.DB.First(&queued, job.ID).Error)
	require.True(t, queued.ManualFile)
	require.Empty(t, queued.ResponseCiphertext)
	require.Empty(t, queued.CandidateURLCiphertext)
	require.Zero(t, queued.Attempts)
	claimed, err := model.ClaimDueMediaTransfer(time.Now().Unix(), "manual-test-lease")
	require.NoError(t, err)
	require.Equal(t, job.ID, claimed.ID)
	require.NoError(t, transferMediaJob(context.Background(), claimed))
	require.Equal(t, 1, uploads)
	var ready model.Task
	require.NoError(t, model.DB.First(&ready, task.ID).Error)
	require.Equal(t, model.TaskStatus(model.TaskStatusSuccess), ready.Status)
	require.Equal(t, model.TaskBillingStatusSettled, ready.BillingStatus)
	require.Equal(t, task.Quota, ready.Quota)
	require.Equal(t, server.URL+"/video.mp4", ready.PrivateData.ResultURL)
	require.NotContains(t, string(ready.Data), "old-")
	var count int64
	require.NoError(t, model.DB.Model(&model.Task{}).Where("task_id = ?", task.TaskID).Count(&count).Error)
	require.EqualValues(t, 1, count)
	_, err = os.Stat(mediaTransferJobFilePath(claimed))
	require.True(t, os.IsNotExist(err))
}

func TestManualReplacementURLClearsUpstreamCredentialsAndOldDownload(t *testing.T) {
	manualTestSettings(t, []MediaStorageProvider{{ID: "selected", Enabled: true, UploadURL: "https://media.example/upload"}})
	_, job := seedManualTransfer(t)
	require.NoError(t, os.WriteFile(mediaTransferFilePath(job.ID), []byte("old-video"), 0600))
	const source = "https://replacement.example/video.mp4?signature=private"
	won, err := QueueManualMediaTransfer(job.ID, "selected", source, nil, "")
	require.NoError(t, err)
	require.True(t, won)
	var queued model.MediaTransferJob
	require.NoError(t, model.DB.First(&queued, job.ID).Error)
	require.False(t, queued.ManualFile)
	decrypted, err := decryptMediaTransferValue(queued.SourceCiphertext)
	require.NoError(t, err)
	require.Equal(t, source, decrypted)
	options, err := decryptMediaTransferValue(queued.OptionsCiphertext)
	require.NoError(t, err)
	require.Equal(t, "{}", options)
	require.NotEqual(t, mediaTransferFilePath(job.ID), mediaTransferJobFilePath(&queued))
	require.NotContains(t, queued.SourceCiphertext, "signature")
	won, err = QueueManualMediaTransfer(job.ID, "selected", source, nil, "")
	require.NoError(t, err)
	require.False(t, won, "a second administrator must not replace an active transfer")
}

func TestManualMediaRejectsHTMLAndKeepsFailedJob(t *testing.T) {
	manualTestSettings(t, []MediaStorageProvider{{ID: "selected", Enabled: true, UploadURL: "https://media.example/upload"}})
	_, job := seedManualTransfer(t)
	won, err := QueueManualMediaTransfer(job.ID, "selected", "", bytes.NewBufferString("<html>not a video</html>"), "fake.mp4")
	require.ErrorIs(t, err, ErrInvalidManualMediaInput)
	require.False(t, won)
	var unchanged model.MediaTransferJob
	require.NoError(t, model.DB.First(&unchanged, job.ID).Error)
	require.Equal(t, model.MediaTransferFailed, unchanged.Status)
	files, err := filepath.Glob(filepath.Join(mediaTransferSpoolDir(), "manual-*"))
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestManualUploadMissingFileNeverFallsBackToOldSource(t *testing.T) {
	t.Setenv("MEDIA_SPOOL_DIR", t.TempDir())
	_, _, _, _, _, err := ensureMediaTransferFile(context.Background(), &model.MediaTransferJob{ManualFile: true, SpoolFile: "missing.media"}, "https://old.invalid/video.mp4", MediaDownloadOptions{})
	require.ErrorContains(t, err, "manual upload file is missing")
}

func TestManualTransferRollsBackWhenTaskIsAlreadyCompleted(t *testing.T) {
	manualTestSettings(t, []MediaStorageProvider{{ID: "selected", Enabled: true, UploadURL: "https://media.example/upload"}})
	task, job := seedManualTransfer(t)
	require.NoError(t, model.DB.Model(task).Update("status", model.TaskStatusSuccess).Error)
	won, err := QueueManualMediaTransfer(job.ID, "selected", "", bytes.NewReader(manualTestVideo()), "video.mp4")
	require.Error(t, err)
	require.False(t, won)
	var unchanged model.MediaTransferJob
	require.NoError(t, model.DB.First(&unchanged, job.ID).Error)
	require.Equal(t, model.MediaTransferFailed, unchanged.Status)
	require.Equal(t, "old-broken-source", unchanged.SourceCiphertext)
	files, err := filepath.Glob(filepath.Join(mediaTransferSpoolDir(), "manual-*"))
	require.NoError(t, err)
	require.Empty(t, files)
}
