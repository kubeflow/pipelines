package mlflow

import (
	"encoding/json"
	"fmt"
	"strings"

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
	runtimeCfg := GetStringConfig(kfpMLflowConfig)
	return ParseKfpMLflowRuntimeConfigValue(runtimeCfg)
}

func ParseKfpMLflowRuntimeConfigValue(runtimeCfg string) (*commonmlflow.MLflowRuntimeConfig, error) {
	var cfg commonmlflow.MLflowRuntimeConfig
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
	if cfg.AuthType != "kubernetes" {
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}
	// Only InsecureSkipVerify is propagated from the API server. Driver/launcher CA trust is configured
	// separately (e.g., cluster-wide trusted CA injection).
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
