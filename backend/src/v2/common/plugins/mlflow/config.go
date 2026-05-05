package mlflow

import (
	"encoding/json"
	"fmt"

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

func GetKfpMLflowRuntimeConfig() string {
	return GetStringConfig(kfpMLflowConfig)
}

// ParseKfpMLflowRuntimeConfig parses the KFP_MLFLOW_CONFIG environment variable into an MLflowRuntimeConfig struct.
// Returns an error if the variable is not set, malformed, or contains an unsupported auth type.
func ParseKfpMLflowRuntimeConfig() (*commonmlflow.MLflowRuntimeConfig, error) {
	var cfg commonmlflow.MLflowRuntimeConfig
	runtimeCfg := GetKfpMLflowRuntimeConfig()
	if runtimeCfg == "" {
		return nil, fmt.Errorf("KFP_MLFLOW_CONFIG env var not set")
	}
	if err := json.Unmarshal([]byte(runtimeCfg), &cfg); err != nil {
		return &cfg, err
	}
	if cfg.Workspace != "" {
		cfg.WorkspacesEnabled = true
	}
	if cfg.AuthType != "kubernetes" {
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}
	cfg.TLS = &commonmlflow.TLSConfig{
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
// by delegating to the common BuildRequestContext with task-specific parameters.
func BuildMLflowTaskRequestContext(runtimeCfg commonmlflow.MLflowRuntimeConfig) (*commonmlflow.RequestContext, error) {
	mlflowPluginSettings := &commonmlflow.MLflowPluginSettings{
		WorkspacesEnabled: &runtimeCfg.WorkspacesEnabled,
		KFPBaseURL:        runtimeCfg.Endpoint,
		InjectUserEnvVars: &runtimeCfg.InjectUserEnvVars,
	}

	pluginCfg := commonmlflow.PluginConfig{
		Endpoint: runtimeCfg.Endpoint,
		Timeout:  runtimeCfg.Timeout,
		TLS:      runtimeCfg.TLS,
		Settings: mlflowPluginSettings,
	}
	return commonmlflow.BuildMLflowRequestContext(pluginCfg, runtimeCfg.Workspace, runtimeCfg.WorkspacesEnabled)
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
