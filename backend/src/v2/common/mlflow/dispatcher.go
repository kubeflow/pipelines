package mlflow

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	k8score "k8s.io/api/core/v1"
)

var _ plugins.TaskPluginDispatcher = (*TaskPluginDispatcher)(nil)

// TaskPluginDispatcher implements TaskPluginDispatcher for MLflow.
type TaskPluginDispatcher struct {
	pluginCfg commonmlflow.PluginConfig
	taskInfo  TaskInfo
}

// NewDispatcher creates a new MLflow plugin dispatcher.
func NewDispatcher() (*TaskPluginDispatcher, error) {
	cfg, err := config.FormatKfpMLflowRuntimeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to format MLflow runtime config: %s", err)
	}

	info, err := getTaskInfoFromRuntimeConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to format MLflow TaskInfo: %s", err)
	}
	pluginCfg := getPluginConfigFromRuntimeConfig(cfg)
	return &TaskPluginDispatcher{
		taskInfo:  *info,
		pluginCfg: pluginCfg,
	}, nil
}

// OnTaskStart creates a Handler and invokes its OnTaskStart method with dispatcher's TaskInfo and PluginConfig.
func (d *TaskPluginDispatcher) OnTaskStart(ctx context.Context, taskName string) (*plugins.TaskStartResult, error) {
	handler := NewHandler()
	taskStartResult, err := handler.OnTaskStart(ctx, taskName, d.taskInfo, d.pluginCfg)
	if err != nil {
		glog.Errorf("failed to launch task: %s", err)
	} else {
		d.taskInfo.RunID = taskStartResult.RunID
	}
	return taskStartResult, err
}

// OnTaskEnd parses input taskExecution for task parameters and input outputArtifacts for task scalar metrics, and then
// creates a Handler and invokes its OnTaskEnd method with the parameters and metrics, as well as the dispatcher's TaskInfo and PluginConfig.
func (d *TaskPluginDispatcher) OnTaskEnd(ctx context.Context, taskExecution *metadata.Execution, outputArtifacts []*metadata.OutputArtifact) error {
	metrics := map[string]float64{}
	params := map[string]string{}

	exec := taskExecution.GetExecution()
	d.taskInfo.RunEndTime = exec.GetLastUpdateTimeSinceEpoch()
	d.taskInfo.RunStatus = exec.LastKnownState.String()

	inputParams, _, err := taskExecution.GetParameters()
	if err != nil {
		return err
	}
	for key, value := range inputParams {
		valueFormatted, err := metadata.PbValueToText(value)
		if err != nil {
			glog.Errorf("Failed to format parameter value for key %s: %v", key, err)
			continue
		}
		params[key] = valueFormatted
	}

	for _, artifact := range outputArtifacts {
		if artifact.Artifact != nil && artifact.Artifact.GetType() == "system.Metrics" {
			for customKey, customValue := range artifact.Artifact.CustomProperties {
				// retrieve scalar metric artifact values. do not retrieve display_name or store_session_info.
				if customKey == "display_name" || customKey == "store_session_info" {
					continue
				}
				metrics[customKey] = customValue.GetDoubleValue()
			}
		}
	}

	handler := NewHandler()
	err = handler.OnTaskEnd(ctx, d.taskInfo, metrics, params, d.pluginCfg)
	if err != nil {
		return fmt.Errorf("failed to complete task: %s", err)
	}

	return nil
}

// RetrieveUserContainerEnvVars retrieves the env vars that MLflow wants to pass into the user container.
func (d *TaskPluginDispatcher) RetrieveUserContainerEnvVars() (injectVars []k8score.EnvVar) {

	if d.taskInfo.RunID != "" {
		injectVars = append(injectVars, k8score.EnvVar{Name: "MLFLOW_RUN_ID", Value: d.taskInfo.RunID})
	} else {
		glog.Warningf("MLflow run ID is empty, skipping MLFLOW_RUN_ID env var injection.")
	}

	// Only inject env vars if injection is enabled
	if d.taskInfo.InjectUserEnvVars {
		injectVars = append(injectVars,
			k8score.EnvVar{
				Name:  "MLFLOW_TRACKING_URI",
				Value: d.pluginCfg.Endpoint,
			},
			k8score.EnvVar{
				Name:  "MLFLOW_EXPERIMENT_ID",
				Value: d.taskInfo.ExperimentID,
			})

		if d.taskInfo.WorkspacesEnabled {
			injectVars = append(injectVars,
				k8score.EnvVar{Name: "MLFLOW_WORKSPACE", Value: d.taskInfo.Workspace})
		}
		var auth string
		if d.taskInfo.AuthType == "kubernetes" {
			// Set MLFLOW_TRACKING_AUTH only if auth type is "kubernetes". If MLflow workspaces are enabled, use "kubernetes-namespaced"
			auth = "kubernetes"
			if d.taskInfo.WorkspacesEnabled {
				auth = "kubernetes-namespaced"
			}
			injectVars = append(injectVars, k8score.EnvVar{Name: "MLFLOW_TRACKING_AUTH", Value: auth})
		} else {
			glog.Warningf("MLflow auth type %s is not supported", d.taskInfo.AuthType)
		}
	}
	return injectVars
}

// getTaskInfoFromRuntimeConfig converts an MLflowRuntimeConfig into a TaskInfo for the MLflow plugin dispatcher.
func getTaskInfoFromRuntimeConfig(runtimeCfg commonmlflow.MLflowRuntimeConfig) (*TaskInfo, error) {
	if runtimeCfg.ParentRunID == "" {
		return nil, fmt.Errorf("ParentRunID is required to create MLflow task")
	}
	if runtimeCfg.ExperimentID == "" {
		return nil, fmt.Errorf("ExperimentID is required to create MLflow task")
	}
	if runtimeCfg.AuthType == "" {
		return nil, fmt.Errorf("AuthType is required to create MLflow task")
	}

	info := TaskInfo{
		Workspace:         runtimeCfg.Workspace,
		WorkspacesEnabled: runtimeCfg.Workspace != "",
		RunID:             config.GetMLflowRunID(),
		ParentRunID:       runtimeCfg.ParentRunID,
		ExperimentID:      runtimeCfg.ExperimentID,
		AuthType:          runtimeCfg.AuthType,
		InjectUserEnvVars: runtimeCfg.InjectUserEnvVars,
	}
	return &info, nil
}

// getPluginConfigFromRuntimeConfig converts an MLflowRuntimeConfig into a PluginConfig for the MLflow plugin dispatcher.
func getPluginConfigFromRuntimeConfig(runtimeCfg commonmlflow.MLflowRuntimeConfig) commonmlflow.PluginConfig {
	tlsCfg := commonmlflow.TLSConfig{
		InsecureSkipVerify: runtimeCfg.InsecureSkipVerify,
	}
	return commonmlflow.PluginConfig{
		Endpoint: runtimeCfg.Endpoint,
		Timeout:  runtimeCfg.Timeout,
		TLS:      &tlsCfg,
	}
}
