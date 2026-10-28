package relay

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPersistentTaskRequestSnapshotIsDurableAndRootOnly(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/task-request.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Task{}))
	previous := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previous })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(`{"model":"public-video","prompt":"demo","duration":30,"ratio":"9:16","resolution":"720p","referenceAudios":["https://cdn.example/voice.mp3"],"metadata":{"generate_audio":false,"seed":0}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "public-video", Prompt: "demo", Duration: 30})
	info := &relaycommon.RelayInfo{UserId: 41, OriginModelName: "public-video", TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_snapshot"}, ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 7, UpstreamModelName: "private-provider-model"}}
	task, err := ensurePersistentTaskSubmission(c, info, constant.TaskPlatform("1"))
	require.NoError(t, err)
	stored, err := model.GetTaskByID(task.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.PrivateData.RequestBodySnapshot)
	root := TaskModel2DtoWithInputMaterials(stored)
	require.Contains(t, root.RequestBodySnapshot.BodyJSON, `"duration": 30`)
	require.NotContains(t, root.RequestBodySnapshot.BodyJSON, "private-provider-model")
	require.Nil(t, root.RequestBodySnapshot.Body)
	require.NotEmpty(t, stored.PrivateData.RequestBodySnapshot.Body)
	public := TaskModel2Dto(stored)
	require.Nil(t, public.RequestBodySnapshot)
	encoded, err := common.Marshal(public)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "request_body_snapshot")
	require.NotContains(t, string(encoded), "voice.mp3")
	// Retries reuse the first submission snapshot rather than replacing it.
	common.CleanupBodyStorage(c)
	c.Request.Body = io.NopCloser(strings.NewReader(`{"prompt":"changed"}`))
	_, err = ensurePersistentTaskSubmission(c, info, constant.TaskPlatform("1"))
	require.NoError(t, err)
	stored, err = model.GetTaskByID(task.ID)
	require.NoError(t, err)
	require.Contains(t, string(stored.PrivateData.RequestBodySnapshot.Body), `"prompt":"demo"`)
}
