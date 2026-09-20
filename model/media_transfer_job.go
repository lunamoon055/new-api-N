package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MediaTransferStatus string

const (
	MediaTransferPending    MediaTransferStatus = "PENDING"
	MediaTransferProcessing MediaTransferStatus = "PROCESSING"
	MediaTransferRetry      MediaTransferStatus = "RETRY"
	MediaTransferReady      MediaTransferStatus = "READY"
	MediaTransferFailed     MediaTransferStatus = "FAILED"
)

// MediaTransferJob is the durable owner of an already-generated video. URLs,
// credentials and the original task response are encrypted before insertion.
// TaskRecordID is unique so polling cannot enqueue a second copy of a task.
type MediaTransferJob struct {
	ID                     int64               `gorm:"primaryKey;autoIncrement"`
	TaskRecordID           int64               `gorm:"not null;uniqueIndex"`
	SourceCiphertext       string              `gorm:"type:text;not null"`
	OptionsCiphertext      string              `gorm:"type:text;not null"`
	ResponseCiphertext     string              `gorm:"type:text"`
	Status                 MediaTransferStatus `gorm:"type:varchar(16);not null;index:idx_media_transfer_due,priority:1"`
	Attempts               int                 `gorm:"not null;default:0"`
	NextRetryAt            int64               `gorm:"not null;default:0;index:idx_media_transfer_due,priority:2"`
	LeaseUntil             int64               `gorm:"not null;default:0;index"`
	LeaseToken             string              `gorm:"type:varchar(64);not null;default:''"`
	CandidateURLCiphertext string              `gorm:"type:text"`
	CandidateProvider      string              `gorm:"type:varchar(191);not null;default:''"`
	StoredURL              string              `gorm:"type:text"`
	ByteSize               int64               `gorm:"not null;default:0"`
	SHA256                 string              `gorm:"type:varchar(64);not null;default:''"`
	FileName               string              `gorm:"type:varchar(255);not null;default:''"`
	ContentType            string              `gorm:"type:varchar(128);not null;default:''"`
	LastError              string              `gorm:"type:varchar(512);not null;default:''"`
	CreatedAt              int64               `gorm:"not null;index"`
	UpdatedAt              int64               `gorm:"not null"`
	CompletedAt            int64               `gorm:"not null;default:0;index"`
}

// MediaTransferJobView is the administrator-safe representation of a transfer
// job. It deliberately excludes encrypted source/options and lease tokens.
type MediaTransferJobView struct {
	ID                int64               `json:"id"`
	TaskRecordID      int64               `json:"task_record_id"`
	TaskID            string              `json:"task_id"`
	Status            MediaTransferStatus `json:"status"`
	Attempts          int                 `json:"attempts"`
	NextRetryAt       int64               `json:"next_retry_at"`
	StoredURL         string              `json:"stored_url,omitempty"`
	CandidateProvider string              `json:"candidate_provider,omitempty"`
	ByteSize          int64               `json:"byte_size"`
	FileName          string              `json:"file_name,omitempty"`
	ContentType       string              `json:"content_type,omitempty"`
	LastError         string              `json:"last_error,omitempty"`
	CreatedAt         int64               `json:"created_at"`
	UpdatedAt         int64               `json:"updated_at"`
	CompletedAt       int64               `json:"completed_at,omitempty"`
}

type MediaTransferSummary struct {
	Total                  int     `json:"total"`
	Pending                int     `json:"pending"`
	Processing             int     `json:"processing"`
	Retry                  int     `json:"retry"`
	Ready                  int     `json:"ready"`
	Failed                 int     `json:"failed"`
	QueueDepth             int     `json:"queue_depth"`
	OldestPendingAt        int64   `json:"oldest_pending_at,omitempty"`
	RecentFailureRate      float64 `json:"recent_failure_rate"`
	AverageTransferSeconds float64 `json:"average_transfer_seconds"`
}

type MediaTransferDashboard struct {
	Summary MediaTransferSummary   `json:"summary"`
	Items   []MediaTransferJobView `json:"items"`
	Total   int                    `json:"total"`
}

// EnqueueMediaTransferAndTask commits the hidden, still-processing task and
// its transfer job together. A failed transaction cannot leave a public task
// pending without a recoverable job, or a job with a completed task.
func EnqueueMediaTransferAndTask(task, previous *Task, job *MediaTransferJob) (bool, error) {
	if task == nil || previous == nil || job == nil || task.ID <= 0 {
		return false, gorm.ErrInvalidData
	}
	now := common.GetTimestamp()
	job.TaskRecordID = task.ID
	job.Status = MediaTransferPending
	job.NextRetryAt = now
	job.CreatedAt = now
	job.UpdatedAt = now
	var won bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Task{}).
			Where("id = ? AND status = ? AND billing_status = ? AND quota = ?", task.ID, previous.Status, previous.BillingStatus, previous.Quota).
			Updates(map[string]any{
				"status":               task.Status,
				"progress":             task.Progress,
				"finish_time":          task.FinishTime,
				"fail_reason":          task.FailReason,
				"private_data":         task.PrivateData,
				"data":                 task.Data,
				"media_transfer_state": task.MediaTransferState,
				"billing_status":       task.BillingStatus,
				"billing_target_quota": task.BillingTargetQuota,
				"updated_at":           now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		result = tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "task_record_id"}}, DoNothing: true}).Create(job)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			// The task update and job insert must be one atomic claim. Returning an
			// error rolls back the task update when another worker won the insert.
			return gorm.ErrDuplicatedKey
		}
		won = true
		return nil
	})
	return won, err
}

func ClaimDueMediaTransfer(now int64, leaseToken string) (*MediaTransferJob, error) {
	var candidates []MediaTransferJob
	err := DB.Where("(status IN ? AND next_retry_at <= ?) OR (status = ? AND lease_until <= ?)",
		[]MediaTransferStatus{MediaTransferPending, MediaTransferRetry}, now, MediaTransferProcessing, now).
		Order("next_retry_at, id").Limit(10).Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		result := DB.Model(&MediaTransferJob{}).
			Where("id = ? AND ((status IN ? AND next_retry_at <= ?) OR (status = ? AND lease_until <= ?))",
				candidate.ID, []MediaTransferStatus{MediaTransferPending, MediaTransferRetry}, now, MediaTransferProcessing, now).
			Updates(map[string]any{
				"status":      MediaTransferProcessing,
				"lease_token": leaseToken,
				"lease_until": now + 120,
				"updated_at":  now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			candidate.Status = MediaTransferProcessing
			candidate.LeaseToken = leaseToken
			candidate.LeaseUntil = now + 120
			return &candidate, nil
		}
	}
	return nil, nil
}

func RenewMediaTransferLease(id int64, token string) error {
	now := common.GetTimestamp()
	result := DB.Model(&MediaTransferJob{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, MediaTransferProcessing, token).
		Updates(map[string]any{"lease_until": now + 120, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SaveMediaTransferFile(id int64, token string, size int64, sha256, filename, contentType string) error {
	result := DB.Model(&MediaTransferJob{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, MediaTransferProcessing, token).
		Updates(map[string]any{
			"byte_size": size, "sha256": sha256, "file_name": filename,
			"content_type": contentType, "updated_at": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SaveMediaTransferCandidate(id int64, token, candidateCiphertext, providerID string) error {
	result := DB.Model(&MediaTransferJob{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, MediaTransferProcessing, token).
		Updates(map[string]any{
			"candidate_url_ciphertext": candidateCiphertext, "candidate_provider": providerID,
			"updated_at": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ScheduleMediaTransferRetry(job *MediaTransferJob, nextRetryAt int64, lastError string, failed bool) error {
	if len(lastError) > 512 {
		lastError = lastError[:512]
	}
	status := MediaTransferRetry
	if failed {
		status = MediaTransferFailed
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&MediaTransferJob{}).
			Where("id = ? AND status = ? AND lease_token = ?", job.ID, MediaTransferProcessing, job.LeaseToken).
			Updates(map[string]any{
				"status": status, "attempts": gorm.Expr("attempts + 1"),
				"next_retry_at": nextRetryAt, "lease_until": 0, "lease_token": "",
				"last_error": lastError, "updated_at": common.GetTimestamp(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		if failed {
			result = tx.Model(&Task{}).
				Where("id = ? AND media_transfer_state = ?", job.TaskRecordID, string(MediaTransferPending)).
				Updates(map[string]any{
					"media_transfer_state": string(MediaTransferFailed),
					"updated_at":           common.GetTimestamp(),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return gorm.ErrInvalidTransaction
			}
		}
		return nil
	})
}

// RetryMediaTransferJob requeues a failed transfer without creating a new
// upstream task or changing billing. It is intentionally CAS guarded so two
// administrators cannot reset the same job concurrently.
func RetryMediaTransferJob(id int64) (bool, error) {
	now := common.GetTimestamp()
	var won bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var job MediaTransferJob
		if err := tx.First(&job, id).Error; err != nil {
			return err
		}
		if job.Status != MediaTransferFailed && job.Status != MediaTransferRetry {
			return nil
		}
		result := tx.Model(&MediaTransferJob{}).
			Where("id = ? AND status IN ?", id, []MediaTransferStatus{MediaTransferFailed, MediaTransferRetry}).
			Updates(map[string]any{
				"status": MediaTransferPending, "attempts": 0, "next_retry_at": now,
				"lease_until": 0, "lease_token": "", "last_error": "", "updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		expectedTaskState := string(MediaTransferPending)
		if job.Status == MediaTransferFailed {
			expectedTaskState = string(MediaTransferFailed)
		}
		result = tx.Model(&Task{}).
			Where("id = ? AND media_transfer_state = ? AND status = ?", job.TaskRecordID, expectedTaskState, TaskStatusInProgress).
			Updates(map[string]any{"media_transfer_state": string(MediaTransferPending), "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidTransaction
		}
		won = true
		return nil
	})
	return won, err
}

func GetMediaTransferDashboard(status string, offset, limit int) (MediaTransferDashboard, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := DB.Model(&MediaTransferJob{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return MediaTransferDashboard{}, err
	}
	var jobs []MediaTransferJob
	if err := query.Order("updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		return MediaTransferDashboard{}, err
	}
	views := make([]MediaTransferJobView, 0, len(jobs))
	for _, job := range jobs {
		view := MediaTransferJobView{
			ID: job.ID, TaskRecordID: job.TaskRecordID, Status: job.Status,
			Attempts: job.Attempts, NextRetryAt: job.NextRetryAt, StoredURL: job.StoredURL,
			CandidateProvider: job.CandidateProvider, ByteSize: job.ByteSize, FileName: job.FileName,
			ContentType: job.ContentType, LastError: job.LastError, CreatedAt: job.CreatedAt,
			UpdatedAt: job.UpdatedAt, CompletedAt: job.CompletedAt,
		}
		var task Task
		if err := DB.Select("task_id").First(&task, job.TaskRecordID).Error; err == nil {
			view.TaskID = task.TaskID
		}
		views = append(views, view)
	}

	counts := make(map[MediaTransferStatus]int)
	for _, state := range []MediaTransferStatus{MediaTransferPending, MediaTransferProcessing, MediaTransferRetry, MediaTransferReady, MediaTransferFailed} {
		var count int64
		if err := DB.Model(&MediaTransferJob{}).Where("status = ?", state).Count(&count).Error; err != nil {
			return MediaTransferDashboard{}, err
		}
		counts[state] = int(count)
	}
	allTotal := counts[MediaTransferPending] + counts[MediaTransferProcessing] + counts[MediaTransferRetry] + counts[MediaTransferReady] + counts[MediaTransferFailed]
	var oldest MediaTransferJob
	oldestPendingAt := int64(0)
	if err := DB.Where("status IN ?", []MediaTransferStatus{MediaTransferPending, MediaTransferRetry, MediaTransferProcessing}).Order("created_at ASC").First(&oldest).Error; err == nil {
		oldestPendingAt = oldest.CreatedAt
	}
	var recent []MediaTransferJob
	if err := DB.Where("status IN ?", []MediaTransferStatus{MediaTransferReady, MediaTransferFailed}).Order("updated_at DESC").Limit(200).Find(&recent).Error; err != nil {
		return MediaTransferDashboard{}, err
	}
	failedRecent := 0
	var durationTotal int64
	durationCount := 0
	for _, job := range recent {
		if job.Status == MediaTransferFailed {
			failedRecent++
		}
		if job.Status == MediaTransferReady && job.CompletedAt > job.CreatedAt {
			durationTotal += job.CompletedAt - job.CreatedAt
			durationCount++
		}
	}
	failureRate := 0.0
	if len(recent) > 0 {
		failureRate = float64(failedRecent) / float64(len(recent))
	}
	averageDuration := 0.0
	if durationCount > 0 {
		averageDuration = float64(durationTotal) / float64(durationCount)
	}
	return MediaTransferDashboard{
		Summary: MediaTransferSummary{
			Total: allTotal, Pending: counts[MediaTransferPending], Processing: counts[MediaTransferProcessing],
			Retry: counts[MediaTransferRetry], Ready: counts[MediaTransferReady], Failed: counts[MediaTransferFailed],
			QueueDepth:      counts[MediaTransferPending] + counts[MediaTransferRetry] + counts[MediaTransferProcessing],
			OldestPendingAt: oldestPendingAt, RecentFailureRate: failureRate,
			AverageTransferSeconds: averageDuration,
		}, Items: views, Total: int(total),
	}, nil
}

// CompleteMediaTransfer publishes the verified URL only if the lease is still
// ours and the public task is still awaiting media. The job and task commit in
// one transaction so a restart cannot expose a URL without recording READY.
func CompleteMediaTransfer(job *MediaTransferJob, publicURL string, responseData []byte) (*Task, bool, error) {
	var task Task
	var won bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var current MediaTransferJob
		if err := tx.Where("id = ? AND status = ? AND lease_token = ?", job.ID, MediaTransferProcessing, job.LeaseToken).
			First(&current).Error; err != nil {
			return err
		}
		if err := tx.First(&task, job.TaskRecordID).Error; err != nil {
			return err
		}
		if task.Status != TaskStatusInProgress || task.MediaTransferState != string(MediaTransferPending) {
			return nil
		}
		now := time.Now().Unix()
		task.Status = TaskStatusSuccess
		task.MediaTransferState = string(MediaTransferReady)
		task.Progress = "100%"
		task.FinishTime = now
		task.PrivateData.ResultURL = publicURL
		task.Data = responseData
		result := tx.Model(&Task{}).
			Where("id = ? AND status = ? AND media_transfer_state = ?", task.ID, TaskStatusInProgress, MediaTransferPending).
			Updates(map[string]any{
				"status": task.Status, "media_transfer_state": task.MediaTransferState,
				"progress": task.Progress, "finish_time": now, "private_data": task.PrivateData,
				"data": task.Data, "updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		result = tx.Model(&MediaTransferJob{}).
			Where("id = ? AND status = ? AND lease_token = ?", job.ID, MediaTransferProcessing, job.LeaseToken).
			Updates(map[string]any{
				"status": MediaTransferReady, "stored_url": publicURL,
				"candidate_url_ciphertext": "", "candidate_provider": "", "lease_until": 0,
				"lease_token": "", "last_error": "", "completed_at": now, "updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		won = result.RowsAffected == 1
		if !won {
			return gorm.ErrInvalidTransaction
		}
		return nil
	})
	return &task, won, err
}
