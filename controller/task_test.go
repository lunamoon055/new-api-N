package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCanViewTaskPrivateDetailsOnlyAllowsRoot(t *testing.T) {
	require.False(t, canViewTaskPrivateDetails(common.RoleCommonUser))
	require.False(t, canViewTaskPrivateDetails(common.RoleAdminUser))
	require.True(t, canViewTaskPrivateDetails(common.RoleRootUser))
}

func TestTaskPageWithStatsUsesCurrentFilters(t *testing.T) {
	page := &common.PageInfo{Page: 2, PageSize: 10, Total: 23, Items: []string{"current page"}}
	params := model.SyncTaskQueryParams{ModelName: "video-model", ChannelID: "17", StartTimestamp: 100}
	var statuses []string
	count := func(filtered model.SyncTaskQueryParams) int64 {
		require.Equal(t, params.ModelName, filtered.ModelName)
		require.Equal(t, params.ChannelID, filtered.ChannelID)
		require.Equal(t, params.StartTimestamp, filtered.StartTimestamp)
		statuses = append(statuses, filtered.Status)
		if filtered.Status == string(model.TaskStatusSuccess) {
			return 12
		}
		return 3
	}

	result := taskPageWithStats(page, params, count)
	require.Equal(t, 23, result["total"])
	require.Equal(t, page.Items, result["items"])
	require.Equal(t, []string{"SUCCESS", "FAILURE"}, statuses)
	require.Equal(t, int64(12), result["stats"].(gin.H)["success"])
	require.Equal(t, int64(3), result["stats"].(gin.H)["failure"])

	statuses = nil
	params.Status = string(model.TaskStatusFailure)
	result = taskPageWithStats(page, params, count)
	require.Equal(t, []string{"FAILURE"}, statuses)
	require.Equal(t, int64(0), result["stats"].(gin.H)["success"])
	require.Equal(t, int64(3), result["stats"].(gin.H)["failure"])

	statuses = nil
	params.Status = string(model.TaskStatusInProgress)
	result = taskPageWithStats(page, params, count)
	require.Empty(t, statuses)
	require.Equal(t, int64(0), result["stats"].(gin.H)["success"])
	require.Equal(t, int64(0), result["stats"].(gin.H)["failure"])
}

func TestTasksToDtoUsesRootOnlyMaterialVisibility(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input:       "prompt remains visible",
			InputImages: []string{"https://cdn.example.com/reference.png"},
		},
	}

	adminDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskPrivateDetails(common.RoleAdminUser),
	)[0]
	adminProperties, ok := adminDto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, "prompt remains visible", adminProperties.Input)
	require.Empty(t, adminProperties.InputImages)

	rootDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskPrivateDetails(common.RoleRootUser),
	)[0]
	rootProperties, ok := rootDto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, task.Properties.InputImages, rootProperties.InputImages)
}

func TestTasksToDtoOnlyExposesRawFailureToRoot(t *testing.T) {
	rawReason := `{"error":{"message":"invalid image_urls[0]: image url returned 404"}}`
	task := &model.Task{
		Status:     model.TaskStatusFailure,
		FailReason: "参考图片链接不存在或已失效，请重新上传图片后再试。",
		PrivateData: model.TaskPrivateData{
			UpstreamError: rawReason,
		},
	}

	adminDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskPrivateDetails(common.RoleAdminUser),
	)[0]
	require.Equal(t, task.FailReason, adminDto.FailReason)
	require.Empty(t, adminDto.RawFailReason)
	adminPayload, err := common.Marshal(adminDto)
	require.NoError(t, err)
	require.NotContains(t, string(adminPayload), "image url returned 404")

	rootDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskPrivateDetails(common.RoleRootUser),
	)[0]
	require.Equal(t, task.FailReason, rootDto.FailReason)
	require.Equal(t, rawReason, rootDto.RawFailReason)
}

func TestRespondTaskErrorTranslatesVideoErrorWithoutReturningRawBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/async-generations", nil)
	rawReason := `{"error":{"message":"invalid image_urls[0]: image url returned 404"}}`

	respondTaskError(ctx, service.TaskErrorWrapper(
		errors.New(rawReason),
		"fail_to_fetch_task",
		http.StatusBadRequest,
	))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "参考图片链接不存在或已失效，请重新上传图片后再试。", response["message"])
	require.NotContains(t, recorder.Body.String(), "image url returned 404")
	require.Equal(t, "fail_to_fetch_task", response["code"])
}

func TestRespondTaskErrorPassesThroughDetailedVideoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/async-generations", nil)
	rawReason := `{"error":{"message":"minimax-h3 video reference duration must be between 1 and 15 seconds","type":"invalid_request_error"}}`
	expected := "minimax-h3 video reference duration must be between 1 and 15 seconds"

	respondTaskError(ctx, service.TaskErrorWrapper(
		errors.New(rawReason),
		"fail_to_fetch_task",
		http.StatusBadRequest,
	))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, expected, response["message"])
	require.NotContains(t, recorder.Body.String(), "请求参数无效")
	require.Equal(t, "fail_to_fetch_task", response["code"])
}
