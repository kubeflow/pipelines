package mlflow

import (
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/spf13/viper"
)

type TaskInfo struct {
	Workspace         string `json:"workspace"`
	WorkspacesEnabled bool   `json:"workspacesEnabled"`
	ParentRunID       string `json:"parentRunId"`
	RunID             string `json:"runID"`
	ExperimentID      string `json:"experimentId"`
	AuthType          string `json:"authType"`
	RunEndTime        int64  `json:"runEndTime"`
	RunStatus         string `json:"runStatus"`
	InjectUserEnvVars bool   `json:"injectUserEnvVars,omitempty"`
}

// IsEnabled reports whether the env var for the MLflow runtime config is present,
// indicating the driver/launcher has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet(commonmlflow.EnvMLflowConfig)
}

// BuildMLflowTaskRequestContext constructs a fully initialized RequestContext
// by delegating to the common BuildRequestContext with task-specific parameters.
func BuildMLflowTaskRequestContext(taskInfo TaskInfo, pluginCfg commonmlflow.PluginConfig) (*commonmlflow.RequestContext, error) {
	return commonmlflow.BuildMLflowRequestContext(pluginCfg, taskInfo.Workspace, taskInfo.WorkspacesEnabled)
}

// ExecutionStateToMLflowTerminalStatus converts a string representing an MLMD Execution_State to an MLflow
// terminal status.
func ExecutionStateToMLflowTerminalStatus(state string) string {
	switch state {
	case "COMPLETE", "CACHED":
		return "FINISHED"
	case "CANCELED":
		return "KILLED"
	default:
		return "FAILED"
	}
}
