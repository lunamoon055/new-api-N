package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestMarkVideoMediaTransferPendingHidesUpstreamResult(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_media_pending",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		FinishTime: 123,
		FailReason: "https://upstream.example/video.mp4",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/video.mp4",
		},
	}

	MarkVideoMediaTransferPending(task)

	require.Equal(t, model.TaskStatus(model.TaskStatusInProgress), task.Status)
	require.Equal(t, "99%", task.Progress)
	require.Zero(t, task.FinishTime)
	require.Empty(t, task.PrivateData.ResultURL)
	require.Empty(t, task.FailReason)
	require.JSONEq(t, `{"media_transfer":"pending"}`, string(task.Data))
}
