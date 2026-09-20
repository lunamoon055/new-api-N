package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestMediaTransferEncryptionRoundTrip(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "stage-two-test-secret")
	previous := common.CryptoSecret
	common.CryptoSecret = "stage-two-test-secret"
	t.Cleanup(func() { common.CryptoSecret = previous })

	sealed, err := encryptMediaTransferValue("https://upstream.example/video.mp4?sig=private")
	require.NoError(t, err)
	require.NotContains(t, sealed, "upstream.example")
	plain, err := decryptMediaTransferValue(sealed)
	require.NoError(t, err)
	require.Equal(t, "https://upstream.example/video.mp4?sig=private", plain)
}

func TestMediaTransferBackoffIsBounded(t *testing.T) {
	require.Equal(t, 5*time.Second, mediaTransferBackoff(0))
	require.Equal(t, 5*time.Second, mediaTransferBackoff(1))
	require.Equal(t, 2*time.Hour, mediaTransferBackoff(8))
	require.Equal(t, 2*time.Hour, mediaTransferBackoff(100))
}

func TestRetryMediaTransferJobRequeuesWithoutCreatingUpstreamTask(t *testing.T) {
	task := &model.Task{
		TaskID:             "task_transfer_retry_test",
		Status:             model.TaskStatusInProgress,
		MediaTransferState: string(model.MediaTransferFailed),
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}
	require.NoError(t, model.DB.Create(task).Error)
	job := &model.MediaTransferJob{
		TaskRecordID: task.ID, SourceCiphertext: "sealed-source", OptionsCiphertext: "sealed-options",
		Status: model.MediaTransferFailed, Attempts: 8, NextRetryAt: time.Now().Unix(),
		CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(), LastError: "timeout",
	}
	require.NoError(t, model.DB.Create(job).Error)
	t.Cleanup(func() {
		model.DB.Delete(&model.MediaTransferJob{}, job.ID)
		model.DB.Delete(&model.Task{}, task.ID)
	})

	won, err := model.RetryMediaTransferJob(job.ID)
	require.NoError(t, err)
	require.True(t, won)
	var reloadedJob model.MediaTransferJob
	require.NoError(t, model.DB.First(&reloadedJob, job.ID).Error)
	require.Equal(t, model.MediaTransferPending, reloadedJob.Status)
	require.Zero(t, reloadedJob.Attempts)
	var reloadedTask model.Task
	require.NoError(t, model.DB.First(&reloadedTask, task.ID).Error)
	require.Equal(t, string(model.MediaTransferPending), reloadedTask.MediaTransferState)
	require.Equal(t, model.TaskStatus(model.TaskStatusInProgress), reloadedTask.Status)
}

func TestMediaTransferDashboardExcludesEncryptedPayloads(t *testing.T) {
	now := time.Now().Unix()
	task := &model.Task{TaskID: "task_transfer_dashboard_test", Status: model.TaskStatusInProgress, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, model.DB.Create(task).Error)
	job := &model.MediaTransferJob{
		TaskRecordID: task.ID, SourceCiphertext: "private-source", OptionsCiphertext: "private-options",
		Status: model.MediaTransferReady, StoredURL: "https://media.example/ready.mp4", ByteSize: 2048,
		CreatedAt: now - 10, UpdatedAt: now, CompletedAt: now,
	}
	require.NoError(t, model.DB.Create(job).Error)
	t.Cleanup(func() {
		model.DB.Delete(&model.MediaTransferJob{}, job.ID)
		model.DB.Delete(&model.Task{}, task.ID)
	})

	dashboard, err := model.GetMediaTransferDashboard("READY", 0, 10)
	require.NoError(t, err)
	require.NotEmpty(t, dashboard.Items)
	require.Equal(t, "task_transfer_dashboard_test", dashboard.Items[0].TaskID)
	require.NotContains(t, dashboard.Items[0].StoredURL, "private-source")
	require.NotContains(t, dashboard.Items[0].StoredURL, "private-options")
	require.GreaterOrEqual(t, dashboard.Summary.Ready, 1)
}

func TestMediaStorageCircuitOpensAndResets(t *testing.T) {
	providerID := "circuit-test-provider"
	recordMediaProviderSuccess(providerID)
	t.Cleanup(func() { recordMediaProviderSuccess(providerID) })

	for range mediaTransferCircuitFailures {
		recordMediaProviderFailure(providerID, context.DeadlineExceeded)
	}
	require.False(t, mediaProviderAvailable(providerID, time.Now()))

	recordMediaProviderSuccess(providerID)
	require.True(t, mediaProviderAvailable(providerID, time.Now()))
}

func TestRetryScheduledMediaTransferMakesItImmediatelyDue(t *testing.T) {
	now := time.Now().Unix()
	task := &model.Task{
		TaskID: "task_transfer_retry_scheduled", Status: model.TaskStatusInProgress,
		MediaTransferState: string(model.MediaTransferPending), CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(task).Error)
	job := &model.MediaTransferJob{
		TaskRecordID: task.ID, SourceCiphertext: "sealed-source", OptionsCiphertext: "sealed-options",
		Status: model.MediaTransferRetry, Attempts: 2, NextRetryAt: now + 3600, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(job).Error)
	t.Cleanup(func() {
		model.DB.Delete(&model.MediaTransferJob{}, job.ID)
		model.DB.Delete(&model.Task{}, task.ID)
	})

	won, err := model.RetryMediaTransferJob(job.ID)
	require.NoError(t, err)
	require.True(t, won)
	var reloaded model.MediaTransferJob
	require.NoError(t, model.DB.First(&reloaded, job.ID).Error)
	require.Equal(t, model.MediaTransferPending, reloaded.Status)
	require.LessOrEqual(t, reloaded.NextRetryAt, time.Now().Unix())
}
