package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sunoPollingTestAdaptor struct {
	responseBody string
}

func (a *sunoPollingTestAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *sunoPollingTestAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(a.responseBody)),
	}, nil
}

func (a *sunoPollingTestAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return nil, nil
}

func (a *sunoPollingTestAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return 0
}

func installSunoPollingTestAdaptor(t *testing.T, responseBody string) {
	t.Helper()
	previous := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(platform constant.TaskPlatform) TaskPollingAdaptor {
		require.Equal(t, constant.TaskPlatformSuno, platform)
		return &sunoPollingTestAdaptor{responseBody: responseBody}
	}
	t.Cleanup(func() {
		GetTaskAdaptorFunc = previous
	})
}

func disablePollingTestMemoryCache(t *testing.T) {
	t.Helper()
	previous := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previous
	})
}

func setPollingTestChannelBaseURL(t *testing.T, channelID int) {
	t.Helper()
	require.NoError(t, model.DB.Model(&model.Channel{}).
		Where("id = ?", channelID).
		Update("base_url", "https://example.invalid").Error)
}

func TestBuildVideoTaskFetchBodyIncludesModel(t *testing.T) {
	task := &model.Task{
		Action: constant.TaskActionTextGenerate,
		Properties: model.Properties{
			UpstreamModelName: "sora2",
			OriginModelName:   "sora2",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_upstream",
		},
	}

	body := buildVideoTaskFetchBody(task)

	require.Equal(t, "task_upstream", body["task_id"])
	require.Equal(t, constant.TaskActionTextGenerate, body["action"])
	require.Equal(t, "sora2", body["model"])
	require.Equal(t, "sora2", body["origin_model"])
}

func TestSweepTimedOutTasks_LegacyTaskDoesNotRefund(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	previousTimeout := constant.TaskTimeoutMinutes
	constant.TaskTimeoutMinutes = 1
	t.Cleanup(func() {
		constant.TaskTimeoutMinutes = previousTimeout
	})

	const userID, tokenID, channelID = 50, 50, 50
	const userQuota, tokenQuota, chargedQuota = 2_000, 1_500, 600
	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-legacy-timeout", tokenQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, chargedQuota, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatusInProgress
	task.Progress = "50%"
	task.SubmitTime = time.Now().Add(-2 * time.Minute).Unix()
	persistBillingTask(t, task, "task_legacy_timeout", model.TaskBillingStatusLegacy, 0)

	sweepTimedOutTasks(ctx)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), persisted.Status)
	assert.Equal(t, "100%", persisted.Progress)
	assert.Contains(t, persisted.FailReason, "旧系统遗留任务")
	assert.Equal(t, model.TaskBillingStatusLegacy, persisted.BillingStatus)
	assert.Equal(t, chargedQuota, persisted.Quota)
	assert.Equal(t, userQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenQuota, getTokenRemainQuota(t, tokenID))
	assert.Zero(t, countLogs(t))
}

func TestUpdateSunoTasks_FailReasonWithEmptyStatusRefunds(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	disablePollingTestMemoryCache(t)

	const userID, tokenID, channelID = 51, 51, 51
	const userQuota, tokenQuota, chargedQuota = 2_000, 1_500, 600
	const upstreamID = "upstream_suno_failure"

	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-suno-empty-status", tokenQuota)
	seedChannel(t, channelID)
	setPollingTestChannelBaseURL(t, channelID)

	task := makeTask(userID, channelID, chargedQuota, tokenID, BillingSourceWallet, 0)
	task.Platform = constant.TaskPlatformSuno
	task.Status = model.TaskStatusInProgress
	task.Progress = "50%"
	task.PrivateData.UpstreamTaskID = upstreamID
	persistBillingTask(t, task, "task_suno_empty_status", model.TaskBillingStatusPending, chargedQuota)

	installSunoPollingTestAdaptor(t, `{
		"code":"success",
		"data":[{
			"task_id":"upstream_suno_failure",
			"status":"",
			"fail_reason":"upstream generation failed"
		}]
	}`)

	err := updateSunoTasks(
		ctx,
		channelID,
		[]string{upstreamID},
		map[string]*model.Task{upstreamID: task},
	)
	require.NoError(t, err)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), persisted.Status)
	assert.Equal(t, "100%", persisted.Progress)
	assert.Equal(t, "upstream generation failed", persisted.FailReason)
	assert.Equal(t, model.TaskBillingStatusRefunded, persisted.BillingStatus)
	assert.Zero(t, persisted.BillingTargetQuota)
	assert.Zero(t, persisted.Quota)
	assert.Equal(t, userQuota+chargedQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenQuota+chargedQuota, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, int64(1), countLogs(t))
}

func TestUpdateSunoTasks_UnknownUpstreamIDDoesNotPanic(t *testing.T) {
	truncate(t)
	disablePollingTestMemoryCache(t)

	const channelID = 52
	seedChannel(t, channelID)
	setPollingTestChannelBaseURL(t, channelID)
	installSunoPollingTestAdaptor(t, `{
		"code":"success",
		"data":[{
			"task_id":"unknown_upstream_id",
			"status":"SUCCESS"
		}]
	}`)

	require.NotPanics(t, func() {
		err := updateSunoTasks(
			context.Background(),
			channelID,
			[]string{"requested_upstream_id"},
			map[string]*model.Task{},
		)
		require.NoError(t, err)
	})
}

func TestSubmittingTaskSkippedByPollingAndRecoveredWhenStale(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 53, 53, 53
	const userQuota, tokenQuota, chargedQuota = 2_000, 1_500, 600
	seedUser(t, userID, userQuota)
	seedToken(t, tokenID, userID, "sk-stale-submitting", tokenQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, chargedQuota, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatusInProgress
	task.Progress = "10%"
	task.UpdatedAt = time.Now().Unix()
	persistBillingTask(t, task, "task_stale_submitting", model.TaskBillingStatusSubmitting, chargedQuota)

	for _, pending := range model.GetAllUnFinishSyncTasks(100) {
		assert.NotEqual(t, task.ID, pending.ID, "SUBMITTING task entered ordinary polling")
	}

	recoverStaleTaskSubmissions(ctx, 100)
	fresh := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskBillingStatusSubmitting, fresh.BillingStatus)

	require.NoError(t, model.DB.Model(&model.Task{}).
		Where("id = ?", task.ID).
		Update("updated_at", time.Now().Add(-taskSubmissionRecoveryDelay-time.Minute).Unix()).Error)

	recoverStaleTaskSubmissions(ctx, 100)

	persisted := reloadBillingTask(t, task.ID)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), persisted.Status)
	assert.Equal(t, "100%", persisted.Progress)
	assert.Contains(t, persisted.FailReason, "提交进程中断")
	assert.Equal(t, model.TaskBillingStatusRefunded, persisted.BillingStatus)
	assert.Zero(t, persisted.Quota)
	assert.Equal(t, userQuota+chargedQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenQuota+chargedQuota, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, int64(1), countLogs(t))
}
