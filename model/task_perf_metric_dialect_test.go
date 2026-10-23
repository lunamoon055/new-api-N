package model

import (
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVideoTaskPerfMetricsAcrossConfiguredDialect(t *testing.T) {
	dialect := os.Getenv("PERF_METRICS_TEST_DIALECT")
	if dialect == "" {
		t.Skip("set PERF_METRICS_TEST_DIALECT to test an external database")
	}
	var dialector gorm.Dialector
	switch dialect {
	case "mysql":
		dialector = mysql.Open(os.Getenv("PERF_METRICS_MYSQL_DSN"))
	case "postgres":
		dialector = postgres.Open(os.Getenv("PERF_METRICS_POSTGRES_DSN"))
	default:
		t.Fatalf("unsupported dialect %q", dialect)
	}
	database, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&Task{}))
	sqlDB, err := database.DB()
	require.NoError(t, err)
	previousDB := DB
	previousSQLite, previousMySQL, previousPostgres := common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL
	DB = database
	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = false, dialect == "mysql", dialect == "postgres"
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = previousSQLite, previousMySQL, previousPostgres
		initCol()
		_ = sqlDB.Close()
	})
	name := "video-metric-" + time.Now().Format("150405.000000000")
	tasks := []Task{
		{Platform: "openai", Group: "vip", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1120, Properties: Properties{OriginModelName: name, UpstreamModelName: "provider-model"}},
		{Platform: "openai", Group: "vip", Status: TaskStatusFailure, SubmitTime: 1000, UpdatedAt: 1180, Properties: Properties{OriginModelName: name}},
		{Platform: "openai", Group: "other", Status: TaskStatusSuccess, SubmitTime: 1000, FinishTime: 1120, Properties: Properties{OriginModelName: name}},
	}
	require.NoError(t, DB.Create(&tasks).Error)
	metrics, err := GetVideoTaskPerfMetrics(name, "vip", []string{"vip"}, 1000, 1200, 3600)
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	require.EqualValues(t, 2, metrics[0].RequestCount)
	require.EqualValues(t, 1, metrics[0].SuccessCount)
	require.EqualValues(t, 300000, metrics[0].TotalLatencyMs)
}
