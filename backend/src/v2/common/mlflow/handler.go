package mlflow

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
)

var _ plugins.TaskPluginHandler = (*Handler)(nil)

// Handler implements PluginHandler for the MLflow integration.
type Handler struct {
}

// NewHandler creates a new MLflow plugin handler with the given dependencies
// and plugin input.
func NewHandler() *Handler {
	return &Handler{}
}

// OnTaskStart creates a nested MLflow run for the given task.
func (h *Handler) OnTaskStart(ctx context.Context, taskName string, info interface{}, config interface{}) (*plugins.TaskStartResult, error) {
	if h == nil || config == nil {
		return nil, nil
	}
	taskInfo, ok := info.(TaskInfo)
	if !ok {
		return nil, fmt.Errorf("invalid task info: %v", info)
	}
	pluginConfig, ok := config.(commonmlflow.PluginConfig)
	if !ok {
		return nil, fmt.Errorf("invalid plugin config: %v", config)
	}
	if taskInfo.ParentRunID == "" || taskInfo.ExperimentID == "" {
		return nil, fmt.Errorf("ParentRunID and ExperimentID are both required to create nested MLflow run")
	}
	mlflowRequestCtx, err := BuildMLflowTaskRequestContext(taskInfo, pluginConfig)
	if err != nil {
		return nil, err
	}
	if mlflowRequestCtx == nil {
		return nil, fmt.Errorf("failed to build MLflow request context: %v", err)
	}
	nestedRunID, err := CreateNestedRun(ctx, mlflowRequestCtx, taskName, taskInfo.ExperimentID, taskInfo.ParentRunID)
	if err != nil {
		return nil, err
	}

	return &plugins.TaskStartResult{RunID: nestedRunID}, nil
}

// OnTaskEnd updates the nested MLflow run for the given task, along with its corresponding metrics and parameters.
func (h *Handler) OnTaskEnd(ctx context.Context, info interface{}, metrics map[string]float64, params map[string]string, config interface{}) error {
	taskInfo, ok := info.(TaskInfo)
	if !ok {
		return fmt.Errorf("invalid task info: %v", info)
	}
	pluginConfig, ok := config.(commonmlflow.PluginConfig)
	if !ok {
		return fmt.Errorf("invalid plugin config: %v", config)
	}
	if taskInfo.RunID == "" || taskInfo.ExperimentID == "" {
		return fmt.Errorf("RunID and ExperimentID are both required to update nested MLflow run")
	}
	mlflowRequestCtx, err := BuildMLflowTaskRequestContext(taskInfo, pluginConfig)
	if err != nil {
		return fmt.Errorf("failed to build MLflow request context: %v", err)
	}

	// Record run data before updating MLflow run status.
	err = LogBatch(ctx, mlflowRequestCtx, taskInfo.RunID, mapToMetrics(metrics), mapToParams(params), []commonmlflow.Tag{})
	if err != nil {
		glog.Errorf("failed to log metrics and params to MLflow: %v", err)
	}

	endTime := taskInfo.RunEndTime
	err = UpdateRun(ctx, mlflowRequestCtx, taskInfo.RunID, taskInfo.RunStatus, endTime)
	if err != nil {
		return fmt.Errorf("failed to update MLflow run: %v", err)
	}

	return nil
}

// TaskPluginHandler defines the generic task-level plugin lifecycle hooks
type TaskPluginHandler interface {
	OnTaskStart(ctx context.Context, taskInfo TaskInfo, config *commonmlflow.PluginConfig) (string, error)
	OnTaskEnd(ctx context.Context, taskInfo TaskInfo, metrics []commonmlflow.Metric, params []commonmlflow.Param, config *commonmlflow.PluginConfig) error
}

// mapToMetrics converts a map of string to float64 into a slice of MLflow Metric structs.
func mapToMetrics(metrics map[string]float64) []commonmlflow.Metric {
	metricsFmt := make([]commonmlflow.Metric, 0, len(metrics))
	for key, value := range metrics {
		metricsFmt = append(metricsFmt, commonmlflow.Metric{Key: key, Value: value})
	}
	return metricsFmt
}

// mapToParams converts a map of string to string into a slice of MLflow Param structs.
func mapToParams(params map[string]string) []commonmlflow.Param {
	paramsFmt := make([]commonmlflow.Param, 0, len(params))
	for key, value := range params {
		paramsFmt = append(paramsFmt, commonmlflow.Param{Key: key, Value: value})
	}
	return paramsFmt
}
