package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

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
