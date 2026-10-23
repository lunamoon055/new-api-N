package model

import (
	"sort"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetVideoTaskPerfMetrics derives one outcome per persisted terminal task.
// Submission acknowledgements, poll requests and media transfers still in
// progress are not outcomes. Reading task state also covers historical tasks
// without replaying charges or recording a second sample on every poll.
func GetVideoTaskPerfMetrics(modelName, group string, groups []string, startTs, endTs, bucketSeconds int64) ([]PerfMetric, error) {
	if groups != nil && len(groups) == 0 {
		return nil, nil
	}
	if bucketSeconds <= 0 {
		bucketSeconds = 3600
	}
	// Keep finish_time indexable. Older terminal tasks can lack finish_time;
	// use their last persisted timestamp with portable comparisons.
	query := DB.Model(&Task{}).
		Select("id", "platform", "action", "status", "submit_time", "start_time", "finish_time", "created_at", "updated_at", "properties", "group").
		Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Where("platform NOT IN ?", []constant.TaskPlatform{constant.TaskPlatformSuno, constant.TaskPlatformMidjourney}).
		Where("action <> ?", constant.TaskActionImageGenerate).
		Where(`(finish_time > 0 AND finish_time >= ? AND finish_time <= ?)
			OR (finish_time <= 0 AND updated_at > 0 AND updated_at >= ? AND updated_at <= ?)
			OR (finish_time <= 0 AND updated_at <= 0 AND submit_time >= ? AND submit_time <= ?)`,
			startTs, endTs, startTs, endTs, startTs, endTs)
	query = applyTaskModelNameFilter(query, modelName)
	if group != "" {
		query = query.Where(clause.IN{Column: "group", Values: taskPerfGroupValues([]string{group})})
	}
	if groups != nil {
		query = query.Where(clause.IN{Column: "group", Values: taskPerfGroupValues(groups)})
	}

	type metricKey struct {
		model, group string
		bucket       int64
	}
	buckets := map[metricKey]*PerfMetric{}
	var tasks []Task
	err := query.FindInBatches(&tasks, 500, func(_ *gorm.DB, _ int) error {
		for _, task := range tasks {
			name := task.Properties.ModelName()
			if name == "" {
				continue
			}
			finished := task.FinishTime
			if finished <= 0 {
				finished = task.UpdatedAt
			}
			if finished <= 0 {
				finished = task.SubmitTime
			}
			started := task.SubmitTime
			if started <= 0 {
				started = task.StartTime
			}
			if started <= 0 {
				started = task.CreatedAt
			}
			latencyMs := int64(0)
			if started > 0 && finished >= started {
				latencyMs = (finished - started) * 1000
			}
			usingGroup := task.Group
			if usingGroup == "" {
				usingGroup = "default"
			}
			key := metricKey{name, usingGroup, finished - finished%bucketSeconds}
			metric := buckets[key]
			if metric == nil {
				metric = &PerfMetric{ModelName: name, Group: usingGroup, BucketTs: key.bucket}
				buckets[key] = metric
			}
			metric.RequestCount++
			metric.TotalLatencyMs += latencyMs
			if task.Status == TaskStatusSuccess {
				metric.SuccessCount++
			}
		}
		return nil
	}).Error
	if err != nil {
		return nil, err
	}
	metrics := make([]PerfMetric, 0, len(buckets))
	for _, metric := range buckets {
		metrics = append(metrics, *metric)
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].ModelName != metrics[j].ModelName {
			return metrics[i].ModelName < metrics[j].ModelName
		}
		if metrics[i].Group != metrics[j].Group {
			return metrics[i].Group < metrics[j].Group
		}
		return metrics[i].BucketTs < metrics[j].BucketTs
	})
	return metrics, nil
}

func taskPerfGroupValues(values []string) []interface{} {
	result := make([]interface{}, 0, len(values)+1)
	for _, value := range values {
		result = append(result, value)
		if value == "default" {
			result = append(result, "")
		}
	}
	return result
}
