package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	mlflowRunID     = "MLFLOW_RUN_ID"
	kfpMLflowConfig = "KFP_MLFLOW_CONFIG"
)

// GetStringConfig returns a string from the runtime Viper configuration.
func GetStringConfig(configName string) string {
	return viper.GetString(configName)
}

// GetMLflowRunID returns the configured task-level MLflow run ID.
func GetMLflowRunID() string {
	return GetStringConfig(mlflowRunID)
}

// ParseKfpMLflowRuntimeConfig parses the KFP_MLFLOW_CONFIG environment variable into an MLflowRuntimeConfig struct.
// Returns an error if the variable is not set, malformed, or contains an unsupported auth type.
func ParseKfpMLflowRuntimeConfig() (*commonmlflow.MLflowRuntimeConfig, error) {
	runtimeCfg := GetStringConfig(kfpMLflowConfig)
	return ParseKfpMLflowRuntimeConfigValue(runtimeCfg)
}

// ParseKfpMLflowRuntimeConfigValue parses and validates a KFP_MLFLOW_CONFIG JSON value.
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
// Bearer/basic credentials come from launcher env vars when present, otherwise
// from Secret/kfp-mlflow-credentials in the executor plugin's namespace.
// KFP_MLFLOW_CONFIG carries key names, never secret values.
func BuildMLflowTaskRequestContext(ctx context.Context, runtimeCfg commonmlflow.MLflowRuntimeConfig) (*commonmlflow.RequestContext, error) {
	credentials, err := resolveRuntimeCredentials(ctx, runtimeCfg, "/var/run/secrets/kubernetes.io/serviceaccount/namespace", util.GetKubernetesConfig)
	if err != nil {
		return nil, err
	}
	pluginCfg := commonmlflow.MLflowPluginConfig{
		Endpoint: runtimeCfg.Endpoint,
		Timeout:  runtimeCfg.Timeout,
		TLS:      runtimeCfg.TLS,
	}
	return commonmlflow.BuildMLflowRequestContext(
		ctx,
		pluginCfg,
		credentials,
		runtimeCfg.Workspace,
		runtimeCfg.WorkspacesEnabled,
	)
}

func resolveRuntimeCredentials(ctx context.Context, runtimeCfg commonmlflow.MLflowRuntimeConfig, namespaceFile string, getKubernetesConfig func() (*rest.Config, error)) (commonmlflow.MLflowCredentials, error) {
	// Presence, not validity, selects launcher env credentials. Invalid or partial
	// env credentials must not silently fall back to a different identity.
	switch runtimeCfg.AuthType {
	case commonmlflow.AuthTypeBearer:
		if _, present := os.LookupEnv(commonmlflow.EnvMLflowTrackingToken); present {
			return commonmlflow.ResolveRuntimeMLflowCredentials(runtimeCfg.AuthType)
		}
	case commonmlflow.AuthTypeBasicAuth:
		_, usernamePresent := os.LookupEnv(commonmlflow.EnvMLflowTrackingUsername)
		_, passwordPresent := os.LookupEnv(commonmlflow.EnvMLflowTrackingPassword)
		if usernamePresent || passwordPresent {
			return commonmlflow.ResolveRuntimeMLflowCredentials(runtimeCfg.AuthType)
		}
	default:
		return commonmlflow.ResolveRuntimeMLflowCredentials(runtimeCfg.AuthType)
	}

	// The executor plugin cannot receive per-run SecretKeyRef env vars, so it
	// reads the namespace Secret directly when launcher env credentials are absent.
	namespaceBytes, err := os.ReadFile(namespaceFile)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to resolve pod namespace for MLflow credentials; mount the service account namespace file: %w", err)
	}
	namespace := strings.TrimSpace(string(namespaceBytes))
	if namespace == "" {
		return commonmlflow.MLflowCredentials{}, fmt.Errorf("pod namespace for MLflow credentials is empty; mount a service account namespace file containing the pod namespace")
	}
	restConfig, err := getKubernetesConfig()
	if err != nil {
		return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to initialize Kubernetes config for MLflow credentials; provide service account credentials or a valid kubeconfig: %w", err)
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, fmt.Errorf("failed to initialize Kubernetes clientset for MLflow credentials; check the Kubernetes API endpoint and TLS configuration: %w", err)
	}
	return commonmlflow.ResolveSecretMLflowCredentials(ctx, clientSet, namespace, runtimeCfg.CredentialSecretRef, runtimeCfg.AuthType)
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
