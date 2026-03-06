package plugins

import "context"

type TaskStartResult struct {
	RunID string
}

// TaskPluginHandler defines the generic task-level plugin lifecycle hooks
type TaskPluginHandler interface {
	OnTaskStart(ctx context.Context, taskName string, info interface{}, config interface{}) (*TaskStartResult, error)
	OnTaskEnd(ctx context.Context, info interface{}, metrics map[string]float64, params map[string]string, config interface{}) error
}
