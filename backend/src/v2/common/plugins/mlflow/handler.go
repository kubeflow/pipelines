// Package mlflow implements the MLflow v2 task plugin handler.
package mlflow

import (
	"context"
	"encoding/json"
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
	runtimeCfg  *commonmlflow.MLflowRuntimeConfig
	nestedRunID string
}

// Name returns the name of the MLflowHandler plugin, which is "MLflow".
func (h *MLflowHandler) Name() string {
	return "MLflow"
}

// NewMLflowTaskHandler creates a new MLflow plugin handler with the given dependencies
// and plugin input.
func NewMLflowTaskHandler(cfg *commonmlflow.MLflowRuntimeConfig) (*MLflowHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	if cfg.AuthType != commonmlflow.AuthTypeKubernetes {
		return nil, fmt.Errorf("failed to parse MLflow runtime config: unsupported auth type: %s", cfg.AuthType)
	}
	return &MLflowHandler{
		runtimeCfg: cfg,
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
	nestedRunID, err := requestCtx.Client.CreateRun(ctx, h.runtimeCfg.ExperimentID, taskInfo.Name, []commonmlflow.Tag{parentRunTag})
	if err != nil {
		return nil, fmt.Errorf("failed to create task-level MLflow run: %v", err)
	}
	h.nestedRunID = nestedRunID
	return &MLflowStartResult{RunID: nestedRunID}, nil
}

// OnTaskEnd updates the nested MLflow run for the given task, along with its corresponding metrics and parameters.
// The run ID is resolved from h.nestedRunID, which is set either by OnTaskStart
// (driver path) or by ApplyCustomProperties (launcher path, recovered from MLMD).
func (h *MLflowHandler) OnTaskEnd(ctx context.Context, info *plugins.TaskInfo) error {
	if info == nil {
		return fmt.Errorf("taskInfo is nil")
	}
	if h.runtimeCfg.ExperimentID == "" {
		return fmt.Errorf("experimentID is required to update nested MLflow run")
	}
	if h.nestedRunID == "" {
		return fmt.Errorf("runID is required to update nested MLflow run")
	}
	resolvedRunID := h.nestedRunID

	requestCtx, err := BuildMLflowTaskRequestContext(*h.runtimeCfg)
	if err != nil {
		return fmt.Errorf("failed to build MLflow request context: %v", err)
	}
	if requestCtx == nil || requestCtx.Client == nil {
		return fmt.Errorf("MLflow request context and client must be non-nil")
	}

	// Record run data, if applicable, before updating MLflow run status
	hasMetrics := len(info.ScalarMetrics) > 0
	hasParams := len(info.Parameters) > 0
	hasTags := len(info.Tags) > 0

	if hasMetrics || hasParams || hasTags {
		req := commonmlflow.LogBatchRequest{
			RunID:   resolvedRunID,
			Metrics: mapToMetrics(info.ScalarMetrics),
			Params:  mapToParams(info.Parameters),
			Tags:    mapToTags(info.Tags),
		}
		err = requestCtx.Client.LogBatch(ctx, req)
		if err != nil {
			glog.Errorf("failed to log metrics and params to MLflow: %v", err)
		}
	}

	resolvedStatus := ExecutionStateToMLflowTerminalStatus(info.RunStatus)
	err = requestCtx.Client.UpdateRun(ctx, resolvedRunID, resolvedStatus, new(info.RunEndTime))
	if err != nil {
		return fmt.Errorf("failed to update MLflow run: %v", err)
	}

	return nil
}

// RetrieveUserContainerEnvVars retrieves environment variables to inject into user containers based on the MLflow runtime config.
// The run ID is resolved from h.nestedRunID, set during OnTaskStart.
func (h *MLflowHandler) RetrieveUserContainerEnvVars() (injectVars map[string]string, err error) {
	if h == nil || h.runtimeCfg == nil {
		return nil, fmt.Errorf("MLflow plugin handler and runtime config must be non-nil")
	}
	injectVars = make(map[string]string)

	if h.runtimeCfg.InjectUserEnvVars {
		if h.nestedRunID != "" {
			injectVars["MLFLOW_RUN_ID"] = h.nestedRunID
		} else {
			return nil, fmt.Errorf("MLflow run ID is empty. Cannot inject MLFLOW_RUN_ID env var")
		}

		injectVars["MLFLOW_TRACKING_URI"] = h.runtimeCfg.Endpoint
		injectVars["MLFLOW_EXPERIMENT_ID"] = h.runtimeCfg.ExperimentID

		if h.runtimeCfg.WorkspacesEnabled {
			injectVars["MLFLOW_WORKSPACE"] = h.runtimeCfg.Workspace
		}
		var auth string
		if h.runtimeCfg.AuthType == "kubernetes" {
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

// GenerateCustomProperties returns MLMD execution custom properties for the
// MLflow plugin. The key matches the MLMD custom property key used by the
// metadata client.
func (h *MLflowHandler) GenerateCustomProperties(result plugins.TaskHandlerStartResult) map[string]string {
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
		h.runtimeCfg.ParentRunID = runID
		h.nestedRunID = runID
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

// mapToTags converts a map of string key-value pairs into a slice of MLflow Tag structs.
func mapToTags(tags map[string]string) []commonmlflow.Tag {
	tagsFmt := make([]commonmlflow.Tag, 0, len(tags))
	for key, value := range tags {
		tagsFmt = append(tagsFmt, commonmlflow.Tag{Key: key, Value: value})
	}
	return tagsFmt
}

// mapToParams converts a map of parameters into a slice of MLflow Param structs,
// serializing values as JSON for the MLflow API.
func mapToParams(params map[string]interface{}) []commonmlflow.Param {
	paramsFmt := make([]commonmlflow.Param, 0, len(params))
	for key, value := range params {
		serialized, err := json.Marshal(value)
		if err != nil {
			glog.Warningf("Failed to serialize param %q: %v", key, err)
			continue
		}
		paramsFmt = append(paramsFmt, commonmlflow.Param{Key: key, Value: string(serialized)})
	}
	return paramsFmt
}
