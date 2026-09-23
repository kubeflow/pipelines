package mlflow

import (
	"encoding/json"
	"fmt"
	"strings"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
)

const (
	mlflowRunID     = "MLFLOW_RUN_ID"
	kfpMLflowConfig = "KFP_MLFLOW_CONFIG"
)

func GetStringConfig(configName string) string {
	return viper.GetString(configName)
}

func GetMLflowRunID() string {
	return GetStringConfig(mlflowRunID)
}

// ParseKfpMLflowRuntimeConfig parses the KFP_MLFLOW_CONFIG environment variable into an MLflowRuntimeConfig struct.
// Returns an error if the variable is not set, malformed, or contains an unsupported auth type.
func ParseKfpMLflowRuntimeConfig() (*commonmlflow.MLflowRuntimeConfig, error) {
	var cfg commonmlflow.MLflowRuntimeConfig
	runtimeCfg := GetStringConfig(kfpMLflowConfig)
	if runtimeCfg == "" {
		return nil, fmt.Errorf("KFP_MLFLOW_CONFIG env var not set")
	}
	if err := json.Unmarshal([]byte(runtimeCfg), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KFP_MLFLOW_CONFIG: %v", err)
	}
	if cfg.Workspace != "" {
		cfg.WorkspacesEnabled = true
	}
	var missingFields []string
	if cfg.Endpoint == "" {
		missingFields = append(missingFields, "Endpoint")
	}
	if cfg.ParentRunID == "" {
		missingFields = append(missingFields, "ParentRunID")
	}
	if cfg.ExperimentID == "" {
		missingFields = append(missingFields, "ExperimentID")
	}
	if cfg.AuthType == "" {
		missingFields = append(missingFields, "AuthType")
	}
	if cfg.Timeout == "" {
		missingFields = append(missingFields, "Timeout")
	}
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing one or more of the following required fields in KFP_MLFLOW_CONFIG: %s", strings.Join(missingFields, ", "))
	}
	if !commonmlflow.IsSupportedAuthType(cfg.AuthType) {
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}
	// Only InsecureSkipVerify is propagated from the API server. Driver/launcher CA trust is configured
	// separately (e.g., cluster-wide trusted CA injection).
	cfg.TLS = &commonplugins.TLSConfig{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	return &cfg, nil
}

// IsEnabled reports whether the env var for the MLflow runtime config is present,
// indicating the driver/launcher has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet(commonmlflow.EnvMLflowConfig)
}

// BuildMLflowTaskRequestContext constructs a fully initialized RequestContext
// by delegating to the common BuildMLflowRequestContext with task-specific parameters.
func BuildMLflowTaskRequestContext(runtimeCfg commonmlflow.MLflowRuntimeConfig) (*commonmlflow.RequestContext, error) {
	credentials, err := commonmlflow.ResolveRuntimeMLflowCredentials(runtimeCfg.AuthType)
	if err != nil {
		return nil, err
	}
	pluginCfg := commonmlflow.MLflowPluginConfig{
		Endpoint: runtimeCfg.Endpoint,
		Timeout:  runtimeCfg.Timeout,
		TLS:      runtimeCfg.TLS,
	}
	return commonmlflow.BuildMLflowRequestContext(
		pluginCfg,
		credentials,
		runtimeCfg.Workspace,
		runtimeCfg.WorkspacesEnabled,
	)
}

// TaskStateToMLflowTerminalStatus converts a PipelineTask_TaskState to an MLflow
// terminal status string. Returns an error for unrecognized states.
func TaskStateToMLflowTerminalStatus(state apiV2beta1.PipelineTask_TaskState) (string, error) {
	switch state {
	case apiV2beta1.PipelineTask_SUCCEEDED, apiV2beta1.PipelineTask_CACHED, apiV2beta1.PipelineTask_SKIPPED:
		return "FINISHED", nil
	case apiV2beta1.PipelineTask_FAILED:
		return "FAILED", nil
	default:
		return "", fmt.Errorf("unsupported task state for MLflow terminal status: %v", state)
	}
}
