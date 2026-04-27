package mlflow

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
)

var _ plugins.TaskPluginHandler = (*MLflowHandler)(nil)

// MLflowStartResult carries handler-specific state from OnTaskStart through to
// OnTaskEnd and RetrieveUserContainerEnvVars for the MLflow plugin.
type MLflowStartResult struct {
	RunID string
}

// MLflowHandler Handler implements PluginHandler for the MLflow integration.
type MLflowHandler struct {
	runtimeCfg *commonmlflow.MLflowRuntimeConfig
}

// Name returns the name of the MLflowHandler plugin, which is "MLflow".
func (h *MLflowHandler) Name() string {
	return "MLflow"
}

// NewMLflowTaskHandler creates a new MLflow plugin handler with the given dependencies
// and plugin input.
func NewMLflowTaskHandler() (*MLflowHandler, error) {
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse MLflow runtime config: %v", err)
	}
	return &MLflowHandler{
		runtimeCfg: runtimeCfg,
	}, nil
}

// OnTaskStart creates a nested MLflow run for the given task.
func (h *MLflowHandler) OnTaskStart(ctx context.Context, taskInfo *plugins.TaskInfo) (plugins.TaskHandlerStartResult, error) {
	if h == nil || taskInfo == nil {
		return nil, nil
	}
	if h.runtimeCfg == nil {
		return nil, fmt.Errorf("MLflow runtime config is not set")
	}
	if h.runtimeCfg.ParentRunID == "" || h.runtimeCfg.ExperimentID == "" {
		return nil, fmt.Errorf("ParentRunID and ExperimentID are both required to create nested MLflow run")
	}

	requestCtx, err := BuildMLflowTaskRequestContext(*h.runtimeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build MLflow request context: %v", err)
	}
	if requestCtx == nil || requestCtx.Client == nil {
		return nil, fmt.Errorf("MLflow request context and client must be non-nil")
	}

	parentRunTag := commonmlflow.Tag{Key: commonmlflow.ParentRunTagKey, Value: h.runtimeCfg.ParentRunID}
	parentRunTagUI := commonmlflow.Tag{Key: commonmlflow.ParentRunUIFormatTagKey, Value: h.runtimeCfg.ParentRunID}
	nestedRunID, err := requestCtx.Client.CreateRun(ctx, h.runtimeCfg.ExperimentID, taskInfo.Name, []commonmlflow.Tag{parentRunTagUI, parentRunTag})
	if err != nil {
		return nil, fmt.Errorf("failed to create task-level MLflow run: %v", err)
	}
	return &MLflowStartResult{RunID: nestedRunID}, nil
}

// OnTaskEnd updates the nested MLflow run for the given task, along with its corresponding metrics and parameters.
func (h *MLflowHandler) OnTaskEnd(ctx context.Context, startResult plugins.TaskHandlerStartResult, info *plugins.TaskInfo) error {
	if info == nil {
		return fmt.Errorf("taskInfo is nil")
	}
	if h.runtimeCfg.ExperimentID == "" {
		return fmt.Errorf("experimentID is required to update nested MLflow run")
	}
	var resolvedRunID string
	if startResult != nil {
		mlflowResult, ok := startResult.(*MLflowStartResult)
		if !ok {
			return fmt.Errorf("startResult is not type MLflowStartResult")
		}
		resolvedRunID = mlflowResult.RunID
	} else {
		// if input start result is nil, retrieve MLFLOW_RUN_ID env var.
		if GetMLflowRunID() != "" {
			resolvedRunID = GetMLflowRunID()
		} else {
			return fmt.Errorf("runID is required to update nested MLflow run")
		}
	}

	requestCtx, err := BuildMLflowTaskRequestContext(*h.runtimeCfg)
	if err != nil {
		return fmt.Errorf("failed to build MLflow request context: %v", err)
	}
	if requestCtx == nil || requestCtx.Client == nil {
		return fmt.Errorf("MLflow request context and client must be non-nil")
	}

	// Record run data before updating MLflow run status.
	if info.ScalarMetrics == nil && info.Parameters == nil && info.Tags == nil {
		return fmt.Errorf("at least one of metrics, params, or tags must be provided")
	}

	req := commonmlflow.LogBatchRequest{RunID: resolvedRunID, Metrics: mapToMetrics(info.ScalarMetrics), Params: mapToParams(info.Parameters), Tags: []commonmlflow.Tag{}}
	err = requestCtx.Client.LogBatch(ctx, req)
	if err != nil {
		glog.Errorf("failed to log metrics and params to MLflow: %v", err)
	}

	resolvedStatus := ExecutionStateToMLflowTerminalStatus(info.RunStatus)
	err = requestCtx.Client.UpdateRun(ctx, resolvedRunID, resolvedStatus, new(info.RunEndTime))
	if err != nil {
		return fmt.Errorf("failed to update MLflow run: %v", err)
	}

	return nil
}

// RetrieveUserContainerEnvVars retrieves environment variables to inject into user containers based on the MLflow runtime config.
func (h *MLflowHandler) RetrieveUserContainerEnvVars(result plugins.TaskHandlerStartResult) (injectVars map[string]string, err error) {
	if h == nil || h.runtimeCfg == nil {
		return nil, fmt.Errorf("MLflow plugin handler and runtime config must be non-nil")
	}
	injectVars = make(map[string]string)

	var resolvedRunID string
	if result != nil {
		mlflowResult, ok := result.(*MLflowStartResult)
		if !ok {
			return nil, fmt.Errorf("startResult is not an *MLflowStartResult")
		}
		resolvedRunID = mlflowResult.RunID
	}
	// if input runID is empty, retrieve MLFLOW_RUN_ID env var.
	if resolvedRunID == "" {
		resolvedRunID = GetMLflowRunID()
	}
	if resolvedRunID != "" {
		injectVars["MLFLOW_RUN_ID"] = resolvedRunID
	} else {
		glog.Warningf("MLflow run ID is empty, skipping MLFLOW_RUN_ID env var injection.")
	}

	// Only inject env vars if injection is enabled
	if h.runtimeCfg.InjectUserEnvVars {
		injectVars["MLFLOW_TRACKING_URI"] = h.runtimeCfg.Endpoint
		injectVars["MLFLOW_EXPERIMENT_ID"] = h.runtimeCfg.ExperimentID

		if h.runtimeCfg.WorkspacesEnabled {
			injectVars["MLFLOW_WORKSPACE"] = h.runtimeCfg.Workspace
		}
		var auth string
		if h.runtimeCfg.AuthType == "kubernetes" {
			// Set MLFLOW_TRACKING_AUTH only if auth type is "kubernetes". If MLflow workspaces are enabled, use "kubernetes-namespaced"
			auth = "kubernetes"
			if h.runtimeCfg.WorkspacesEnabled {
				auth = "kubernetes-namespaced"
			}
			injectVars["MLFLOW_TRACKING_AUTH"] = auth
		} else {
			return nil, fmt.Errorf("MLflow auth type %s is not supported", h.runtimeCfg.AuthType)
		}
	}
	return injectVars, nil
}

// RetrieveCustomProperties returns MLMD execution custom properties for the
// MLflow plugin. The key matches the MLMD custom property key used by the
// metadata client.
func (h *MLflowHandler) RetrieveCustomProperties(result plugins.TaskHandlerStartResult) map[string]string {
	if result == nil {
		return nil
	}
	mlflowResult, ok := result.(*MLflowStartResult)
	if !ok || mlflowResult.RunID == "" {
		return nil
	}
	return map[string]string{
		"plugins.mlflow.run_id": mlflowResult.RunID,
	}
}

// ApplyCustomProperties updates the MLflow handler runtime configuration with custom property values.
func (h *MLflowHandler) ApplyCustomProperties(props map[string]string) error {
	if h == nil || h.runtimeCfg == nil {
		return fmt.Errorf("MLflow plugin handler and runtime config must be non-nil")
	}
	if props == nil {
		return nil
	}
	runID, ok := props["plugins.mlflow.run_id"]
	if ok {
		// If runId is provided, value should override runtime config parentRunId
		h.runtimeCfg.ParentRunID = runID
	}
	return nil
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
