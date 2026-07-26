package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
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
// For secret-based auth, the driver executor plugin resolves credentials from
// Secret/kfp-mlflow-credentials in its own namespace using the key names carried
// in KFP_MLFLOW_CONFIG; secret values are not sent through runtime args.
func BuildMLflowTaskRequestContext(ctx context.Context, runtimeCfg commonmlflow.MLflowRuntimeConfig) (*commonmlflow.RequestContext, error) {
	credentials, err := resolveRuntimeCredentials(ctx, runtimeCfg)
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

func resolveRuntimeCredentials(ctx context.Context, runtimeCfg commonmlflow.MLflowRuntimeConfig) (commonmlflow.MLflowCredentials, error) {
	switch runtimeCfg.AuthType {
	case commonmlflow.AuthTypeBearer, commonmlflow.AuthTypeBasicAuth:
		// The API server cannot inject per-run SecretKeyRef env vars into the
		// Argo executor plugin sidecar, so the driver reads the namespace Secret
		// directly for bearer/basic auth.
		namespace, err := config.InPodNamespace()
		if err != nil {
			return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to resolve pod namespace for MLflow credentials: %w", err)
		}
		restConfig, err := util.GetKubernetesConfig()
		if err != nil {
			return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to initialize Kubernetes config for MLflow credentials: %w", err)
		}
		clientSet, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to initialize Kubernetes clientset for MLflow credentials: %w", err)
		}
		return commonmlflow.ResolveSecretMLflowCredentials(ctx, clientSet, namespace, runtimeCfg.CredentialSecretRef, runtimeCfg.AuthType)
	default:
		return commonmlflow.ResolveRuntimeMLflowCredentials(runtimeCfg.AuthType)
	}
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
