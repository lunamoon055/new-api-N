package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func requestBatchModelTags(t *testing.T, ids []int, tags []string) bool {
	t.Helper()
	return requestModelTagOperation(t, ids, tags, BatchAddModelTags)
}

func requestModelTagOperation(t *testing.T, ids []int, tags []string, handler gin.HandlerFunc) bool {
	t.Helper()
	body, err := common.Marshal(map[string]any{"ids": ids, "tags": tags})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/models/batch/tags", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	return response.Success
}

func TestBatchModelTagsClearSelectedOnly(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	t.Cleanup(model.InvalidatePricingCache)
	items := []model.Model{
		{ModelName: "clear-first", Tags: "视频模型,推荐", Description: "keep note", Status: 0, SyncOfficial: 1, Icon: "icon", VendorID: 42, Endpoints: `{"openai-video":{}}`, NameRule: model.NameRulePrefix},
		{ModelName: "clear-empty", Status: 1},
		{ModelName: "clear-unselected", Tags: "untouched", Status: 1},
	}
	for i := range items {
		require.NoError(t, items[i].Insert())
		require.NoError(t, db.First(&items[i], items[i].Id).Error)
	}
	require.True(t, requestModelTagOperation(t, []int{items[0].Id, items[1].Id, items[0].Id}, nil, BatchClearModelTags))
	for i := range items {
		var actual model.Model
		require.NoError(t, db.First(&actual, items[i].Id).Error)
		if i < 2 {
			require.Empty(t, actual.Tags)
			actual.Tags, actual.UpdatedTime = items[i].Tags, items[i].UpdatedTime
		}
		require.Equal(t, items[i], actual)
	}
	require.True(t, requestModelTagOperation(t, []int{items[0].Id}, nil, BatchClearModelTags), "clearing an empty model is safe")
}

func TestBatchModelTagsClearRejectsInvalidBatchAtomically(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	item := model.Model{ModelName: "clear-atomic", Tags: "keep", Status: 1}
	require.NoError(t, item.Insert())
	for _, ids := range [][]int{nil, {0}, {item.Id, item.Id + 1}} {
		require.False(t, requestModelTagOperation(t, ids, nil, BatchClearModelTags))
		var actual model.Model
		require.NoError(t, db.First(&actual, item.Id).Error)
		require.Equal(t, "keep", actual.Tags)
	}
}

func TestBatchModelTagsAppendAndPreserveMetadata(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	t.Cleanup(model.InvalidatePricingCache)
	items := []model.Model{
		{ModelName: "tag-first", Tags: "原有, 视频模型 ", Description: "keep description", Icon: "icon", VendorID: 42, Endpoints: `{"openai-video":{}}`, Status: 0, SyncOfficial: 1, NameRule: model.NameRulePrefix},
		{ModelName: "tag-second", Tags: "other", Status: 1},
		{ModelName: "tag-unselected", Tags: "untouched", Status: 1},
	}
	for i := range items {
		require.NoError(t, items[i].Insert())
		require.NoError(t, db.First(&items[i], items[i].Id).Error)
	}
	require.True(t, requestBatchModelTags(t, []int{items[0].Id, items[1].Id, items[0].Id}, []string{" 视频模型 ", "新标签", "新标签"}))
	var updated model.Model
	require.NoError(t, db.First(&updated, items[0].Id).Error)
	require.Equal(t, "原有,视频模型,新标签", updated.Tags)
	updated.Tags, updated.UpdatedTime = items[0].Tags, items[0].UpdatedTime
	require.Equal(t, items[0], updated, "all fields except tags and updated_time must be preserved")
	var second, untouched model.Model
	require.NoError(t, db.First(&second, items[1].Id).Error)
	require.Equal(t, "other,视频模型,新标签", second.Tags)
	require.NoError(t, db.First(&untouched, items[2].Id).Error)
	require.Equal(t, items[2], untouched)
	tags, err := model.GetModelTags()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"原有", "视频模型", "新标签", "other", "untouched"}, tags)

	require.True(t, requestBatchModelTags(t, []int{items[0].Id}, []string{"新标签"}))
	require.NoError(t, db.First(&updated, items[0].Id).Error)
	require.Equal(t, "原有,视频模型,新标签", updated.Tags, "repeated requests must not duplicate tags")
}

func TestBatchModelTagsRejectInvalidBatchAtomically(t *testing.T) {
	for _, scenario := range []string{"missing model", "overflow", "empty tags", "invalid delimiter", "invalid id"} {
		t.Run(scenario, func(t *testing.T) {
			db := setupModelListControllerTestDB(t)
			first := model.Model{ModelName: "tag-valid", Tags: "原有", Status: 1}
			second := model.Model{ModelName: "tag-overflow", Tags: strings.Repeat("字", 254), Status: 1}
			require.NoError(t, first.Insert())
			require.NoError(t, second.Insert())
			require.NoError(t, db.First(&first, first.Id).Error)
			ids, tags := []int{first.Id, second.Id}, []string{"新标签"}
			switch scenario {
			case "missing model":
				ids = []int{first.Id, second.Id + 1}
			case "empty tags":
				tags = []string{" "}
			case "invalid delimiter":
				tags = []string{"one,two"}
			case "invalid id":
				ids = []int{first.Id, 0}
			}
			require.False(t, requestBatchModelTags(t, ids, tags))
			var unchanged model.Model
			require.NoError(t, db.First(&unchanged, first.Id).Error)
			require.Equal(t, first, unchanged)
		})
	}
}
