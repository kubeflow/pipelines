package plugins

// TaskInfo contains Task-level information
type TaskInfo struct {
	Name            string           `json:"name"`
	TaskStartResult *TaskStartResult `json:"taskStartResult"`
	RunEndTime      int64            `json:"runEndTime"`
	RunStatus       string           `json:"runStatus"`
	ScalarMetrics   map[string]float64
	Parameters      map[string]string
	Tags            map[string]string
}

// UpdateTaskInfoWithMetadata updates the task's scalar metrics and parameters with the provided metadata maps.
func (t *TaskInfo) UpdateTaskInfoWithMetadata(kfpRunStatus string, metrics map[string]float64, params map[string]string) {
	t.RunStatus = kfpRunStatus
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
