package plugins

import (
	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
)

// TaskInfo contains Task-level information
type TaskInfo struct {
	Name          string                       `json:"name"`
	RunEndTime    int64                        `json:"runEndTime"`
	RunStatus     apiV2.PipelineTask_TaskState `json:"runStatus"`
	ScalarMetrics map[string]float64
	Parameters    map[string]interface{}
	Tags          map[string]string
}

// UpdateTaskInfoWithMetadata updates the task's scalar metrics and parameters with the provided metadata maps.
func (t *TaskInfo) UpdateTaskInfoWithMetadata(state apiV2.PipelineTask_TaskState, metrics map[string]float64, params map[string]interface{}) {
	t.RunStatus = state
	if metrics != nil {
		t.ScalarMetrics = metrics
	}
	if params != nil {
		t.Parameters = params
	}
}

func (t *TaskInfo) UpdateTaskInfoWithRunEndTime(runEndTime int64) {
	t.RunEndTime = runEndTime
}
