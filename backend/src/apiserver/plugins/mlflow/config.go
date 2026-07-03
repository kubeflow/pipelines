// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultExperimentName is the MLflow experiment name used when the user
	// and admin configuration do not specify one.
	DefaultExperimentName = "KFP-Default"
	// DefaultTimeout is the default HTTP request timeout for the MLflow client.
	DefaultTimeout = "30s"
)

const (
	LauncherConfigMapName = "kfp-launcher"
	LauncherConfigKey     = "plugins.mlflow"
)

// LauncherNamespaceConfig is the restricted MLflow override shape allowed in the
// namespace-scoped kfp-launcher ConfigMap.
type LauncherNamespaceConfig struct {
	Settings *LauncherNamespaceSettings `json:"settings,omitempty"`
}

// LauncherNamespaceSettings lists the only MLflow settings that a namespace may
// override through the kfp-launcher ConfigMap.
type LauncherNamespaceSettings struct {
	ExperimentDescription *string                           `json:"experimentDescription,omitempty"`
	DefaultExperimentName string                            `json:"defaultExperimentName,omitempty"`
	InjectUserEnvVars     *bool                             `json:"injectUserEnvVars,omitempty"`
	CredentialSecretRef   *commonmlflow.CredentialSecretRef `json:"credentialSecretRef,omitempty"`
}

// ApplySettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplySettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.AuthType == "" {
		settings.AuthType = commonmlflow.AuthTypeKubernetes
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := settings.AuthType == commonmlflow.AuthTypeKubernetes
		settings.WorkspacesEnabled = &defaultEnabled
	}
	if settings.DefaultExperimentName == "" {
		settings.DefaultExperimentName = DefaultExperimentName
	}
	if settings.ExperimentDescription == nil {
		d := DefaultExperimentDescription
		settings.ExperimentDescription = &d
	}
	return settings
}

// ResolvedConfig bundles the merged, defaulted plugin configuration and its
// resolved credentials.
type ResolvedConfig struct {
	Config      *commonmlflow.PluginConfig
	Credentials commonmlflow.MLflowCredentials
}

func newResolvedConfig(config *commonmlflow.PluginConfig, credentials commonmlflow.MLflowCredentials) (*ResolvedConfig, error) {
	if config == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "resolved MLflow config requires plugin config")
	}
	if config.Settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "resolved MLflow config requires plugin settings")
	}
	if credentials.AuthType == "" {
		return nil, util.NewInternalServerError(
			fmt.Errorf("missing resolved credentials for auth type %q", config.Settings.AuthType),
			"resolved MLflow config requires credentials",
		)
	}
	return &ResolvedConfig{
		Config:      config,
		Credentials: credentials,
	}, nil
}

// MLflowPluginInput represents the user-facing plugins_input.mlflow schema.
type MLflowPluginInput struct {
	ExperimentName string `json:"experiment_name,omitempty"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
}

// KubeClientProvider abstracts Kubernetes clientset access.
type KubeClientProvider interface {
	GetClientSet() kubernetes.Interface
}

// IsEnabled reports whether the global plugins.mlflow configuration is present,
// indicating the API server has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet("plugins.mlflow")
}

// GetGlobalMLflowConfig reads the global plugins.mlflow configuration
func GetGlobalMLflowConfig() (commonmlflow.PluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return commonmlflow.PluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return commonmlflow.PluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg commonmlflow.PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return commonmlflow.PluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

// GetServerSideNamespaceMLflowConfig reads an optional per-namespace MLflow
// override from the API server's plugins.mlflow.namespaces config.
func GetServerSideNamespaceMLflowConfig(namespace string) (*commonmlflow.PluginConfig, error) {
	if namespace == "" || !viper.IsSet("plugins.mlflow.namespaces") {
		return nil, nil
	}
	raw := viper.Get("plugins.mlflow.namespaces")
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, util.NewInvalidInputError("failed to marshal global plugins.mlflow.namespaces config: %v", err)
	}
	var namespaceCfgs map[string]json.RawMessage
	if err := json.Unmarshal(data, &namespaceCfgs); err != nil {
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces config: %v", err)
	}
	namespaceRaw, ok := namespaceCfgs[namespace]
	if !ok {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(namespaceRaw))
	decoder.DisallowUnknownFields()
	var cfg commonmlflow.PluginConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces[%q] config: %v", namespace, err)
	}
	var trailing json.RawMessage
	if trailingErr := decoder.Decode(&trailing); trailingErr != io.EOF {
		if trailingErr == nil {
			trailingErr = fmt.Errorf("unexpected trailing JSON content")
		}
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces[%q] config: %v", namespace, trailingErr)
	}
	return &cfg, nil
}

// GetLauncherNamespaceMLflowConfig reads the namespace-level MLflow launcher
// fragment from the kfp-launcher ConfigMap. Returns nil (no error) when the
// ConfigMap or key is absent.
func GetLauncherNamespaceMLflowConfig(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*LauncherNamespaceConfig, error) {
	if namespace == "" {
		return nil, util.NewInternalServerError(fmt.Errorf("namespace is empty"), "namespace must be specified when reading MLflow config")
	}
	if clientSet == nil {
		return nil, util.NewInternalServerError(fmt.Errorf("clientSet is nil"), "Kubernetes clientset must be provided when reading MLflow namespace config")
	}
	cm, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, LauncherConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, util.NewInternalServerError(err, "failed to read MLflow namespace config from configmap %q in namespace %q", LauncherConfigMapName, namespace)
	}
	raw, ok := cm.Data[LauncherConfigKey]
	if !ok || raw == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.DisallowUnknownFields()
	var cfg LauncherNamespaceConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, util.NewInternalServerError(err, "failed to parse MLflow config from key %q in configmap %q/%q", LauncherConfigKey, namespace, LauncherConfigMapName)
	}
	var trailing json.RawMessage
	if trailingErr := decoder.Decode(&trailing); trailingErr != io.EOF {
		if trailingErr == nil {
			trailingErr = fmt.Errorf("unexpected trailing JSON content")
		}
		return nil, util.NewInternalServerError(trailingErr, "failed to parse MLflow config from key %q in configmap %q/%q", LauncherConfigKey, namespace, LauncherConfigMapName)
	}
	return &cfg, nil
}

func applyLauncherNamespaceOverrides(base commonmlflow.PluginConfig, launcherCfg *LauncherNamespaceConfig) commonmlflow.PluginConfig {
	if launcherCfg == nil {
		return base
	}
	base.Settings = mergeLauncherNamespaceSettings(base.Settings, launcherCfg.Settings)
	return base
}

func mergeLauncherNamespaceSettings(base *commonmlflow.MLflowPluginSettings, overrides *LauncherNamespaceSettings) *commonmlflow.MLflowPluginSettings {
	if overrides == nil {
		return base
	}
	if base == nil {
		base = &commonmlflow.MLflowPluginSettings{}
	}
	merged := *base
	if overrides.ExperimentDescription != nil {
		merged.ExperimentDescription = overrides.ExperimentDescription
	}
	if overrides.DefaultExperimentName != "" {
		merged.DefaultExperimentName = overrides.DefaultExperimentName
	}
	if overrides.InjectUserEnvVars != nil {
		merged.InjectUserEnvVars = overrides.InjectUserEnvVars
	}
	if overrides.CredentialSecretRef != nil {
		merged.CredentialSecretRef = overrides.CredentialSecretRef
	}
	return &merged
}

// ResolveMLflowRequestConfig builds a merged and validated ResolvedConfig for the
// given namespace, combining global config, optional server-side namespace
// overrides, and the restricted launcher fragment.
func ResolveMLflowRequestConfig(ctx context.Context, kubeClients KubeClientProvider, namespace string) (*ResolvedConfig, error) {
	globalCfg, hasGlobal, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !hasGlobal {
		return nil, nil
	}

	var clientSet kubernetes.Interface
	if kubeClients != nil {
		clientSet = kubeClients.GetClientSet()
	}

	mergedCfg := globalCfg
	var launcherNamespaceCfg *LauncherNamespaceConfig
	if common.IsMultiUserMode() {
		serverSideNamespaceCfg, err := GetServerSideNamespaceMLflowConfig(namespace)
		if err != nil {
			return nil, err
		}
		mergedCfg = commonmlflow.MergePluginConfig(mergedCfg, serverSideNamespaceCfg)
		if mergedCfg.Settings != nil {
			// In multi-user mode, secret refs are namespace-owned: clear inherited refs so
			// only the namespace launcher ConfigMap can opt back in.
			mergedCfg.Settings.CredentialSecretRef = nil
		}

		if kubeClients == nil {
			return nil, util.NewInternalServerError(fmt.Errorf("kubeClients is nil"), "Kubernetes clients must be provided when reading MLflow namespace config")
		}
		launcherNamespaceCfg, err = GetLauncherNamespaceMLflowConfig(ctx, clientSet, namespace)
		if err != nil {
			return nil, err
		}
		mergedCfg = applyLauncherNamespaceOverrides(mergedCfg, launcherNamespaceCfg)
	}
	if mergedCfg.Timeout == "" {
		mergedCfg.Timeout = DefaultTimeout
	}
	settings := ApplySettingsDefaults(mergedCfg.Settings)
	mergedCfg.Settings = settings
	credentials, err := resolveConfiguredCredentials(ctx, clientSet, namespace, settings)
	if err != nil {
		return nil, err
	}
	return newResolvedConfig(&mergedCfg, credentials)
}

// BuildMLflowRunRequestContext constructs a fully initialized RequestContext by
// performing API-server-specific validation and then delegating to the common
// BuildRequestContext.
func BuildMLflowRunRequestContext(ctx context.Context, namespace string, requestCfg *ResolvedConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil || requestCfg.Config == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "cannot build MLflow request context without a resolved config")
	}
	if requestCfg.Config.Endpoint == "" {
		return nil, util.NewInvalidInputError("plugins.mlflow endpoint must be set")
	}
	settings := requestCfg.Config.Settings
	if settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "BuildMLflowRequestContext requires resolved settings")
	}
	workspacesEnabled := settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled
	return commonmlflow.BuildMLflowRequestContext(*requestCfg.Config, requestCfg.Credentials, namespace, workspacesEnabled)
}

// ResolveMLflowPluginInput parses the plugins_input.mlflow JSON from a run model,
// and validates it against the MLflowPluginInput schema.
func ResolveMLflowPluginInput(pluginsInputString *string) (*MLflowPluginInput, error) {
	if pluginsInputString == nil || *pluginsInputString == "" {
		return &MLflowPluginInput{}, nil
	}

	var pluginInputs apiserverPlugins.PluginsInputMap
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	mlflowRaw, ok := pluginInputs["mlflow"]
	if !ok || len(mlflowRaw) == 0 {
		return &MLflowPluginInput{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(mlflowRaw))
	decoder.DisallowUnknownFields()
	input := &MLflowPluginInput{}
	if err := decoder.Decode(input); err != nil {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must follow schema {experiment_name?: string, experiment_id?: string, disabled?: bool}: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must be a single JSON object")
	}

	return input, nil
}

// SelectMLflowExperiment chooses the selector used for MLflow experiment resolution.
// Priority: user-provided experiment_id > user-provided experiment_name >
// admin-configured defaultExperimentName > hardcoded "KFP-Default".
func SelectMLflowExperiment(input *MLflowPluginInput, settings *commonmlflow.MLflowPluginSettings) (experimentID string, experimentName string) {
	if input != nil {
		if input.ExperimentID != "" {
			return input.ExperimentID, ""
		}
		if input.ExperimentName != "" {
			return "", input.ExperimentName
		}
	}
	if settings != nil && settings.DefaultExperimentName != "" {
		return "", settings.DefaultExperimentName
	}
	return "", DefaultExperimentName
}

func resolveConfiguredCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	settings *commonmlflow.MLflowPluginSettings,
) (commonmlflow.MLflowCredentials, error) {
	if settings == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInternalServerError(
			fmt.Errorf("settings are nil"),
			"MLflow settings must be provided when resolving credentials",
		)
	}
	switch settings.AuthType {
	case commonmlflow.AuthTypeKubernetes:
		return commonmlflow.ResolveMLflowCredentials()
	case commonmlflow.AuthTypeBearer:
		if settings.CredentialSecretRef == nil {
			return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
				"plugins.mlflow.settings.credentialSecretRef is required for authType %q",
				commonmlflow.AuthTypeBearer,
			)
		}
		return resolveBearerSecretCredentials(ctx, clientSet, namespace, settings.CredentialSecretRef)
	case commonmlflow.AuthTypeBasicAuth:
		if settings.CredentialSecretRef == nil {
			return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
				"plugins.mlflow.settings.credentialSecretRef is required for authType %q",
				commonmlflow.AuthTypeBasicAuth,
			)
		}
		return resolveBasicAuthSecretCredentials(ctx, clientSet, namespace, settings.CredentialSecretRef)
	case commonmlflow.AuthTypeNone:
		return commonmlflow.MLflowCredentials{AuthType: commonmlflow.AuthTypeNone}, nil
	default:
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"unsupported plugins.mlflow.settings.authType %q",
			settings.AuthType,
		)
	}
}

func resolveBearerSecretCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	ref *commonmlflow.CredentialSecretRef,
) (commonmlflow.MLflowCredentials, error) {
	if ref == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError("MLflow bearer auth requires credentialSecretRef")
	}
	if ref.TokenKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.tokenKey is required for authType %q",
			commonmlflow.AuthTypeBearer,
		)
	}
	secret, err := getMLflowCredentialSecret(ctx, clientSet, namespace)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	token, err := readRequiredSecretKey(secret, namespace, ref.TokenKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	return commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeBearer,
		BearerToken: token,
	}, nil
}

func resolveBasicAuthSecretCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	ref *commonmlflow.CredentialSecretRef,
) (commonmlflow.MLflowCredentials, error) {
	if ref == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError("MLflow basic auth requires credentialSecretRef")
	}
	if ref.UsernameKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.usernameKey is required for authType %q",
			commonmlflow.AuthTypeBasicAuth,
		)
	}
	if ref.PasswordKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.passwordKey is required for authType %q",
			commonmlflow.AuthTypeBasicAuth,
		)
	}
	secret, err := getMLflowCredentialSecret(ctx, clientSet, namespace)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	username, err := readRequiredSecretKey(secret, namespace, ref.UsernameKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	password, err := readRequiredSecretKey(secret, namespace, ref.PasswordKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	return commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeBasicAuth,
		Username: username,
		Password: password,
	}, nil
}

// getMLflowCredentialSecret reads the fixed MLflow credentials Secret from the
// given namespace.
func getMLflowCredentialSecret(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*corev1.Secret, error) {
	if clientSet == nil {
		return nil, util.NewInternalServerError(
			fmt.Errorf("clientSet is nil"),
			"Kubernetes clientset must be provided when reading MLflow credentials secret",
		)
	}
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, commonmlflow.CredentialSecretName, v1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"failed to read MLflow credentials secret %q in namespace %q",
			commonmlflow.CredentialSecretName,
			namespace,
		)
	}
	return secret, nil
}

// readRequiredSecretKey returns the trimmed value for key from secret, returning
// an error if the key is missing or resolves to an empty value.
func readRequiredSecretKey(secret *corev1.Secret, namespace, key string) (string, error) {
	valueBytes, ok := secret.Data[key]
	if !ok {
		return "", util.NewInvalidInputError(
			"secret %q in namespace %q does not contain key %q",
			commonmlflow.CredentialSecretName,
			namespace,
			key,
		)
	}
	value := strings.TrimSpace(string(valueBytes))
	if value == "" {
		return "", util.NewInvalidInputError(
			"secret %q in namespace %q has an empty value for key %q",
			commonmlflow.CredentialSecretName,
			namespace,
			key,
		)
	}
	return value, nil
}

// InjectMLflowRuntimeEnv sets KFP_MLFLOW_CONFIG on driver and launcher
// containers.
func InjectMLflowRuntimeEnv(executionSpec util.ExecutionSpec, envVars []corev1.EnvVar) error {
	if len(envVars) == 0 || executionSpec == nil {
		return nil
	}
	return executionSpec.UpsertRuntimeEnvVars(envVars,
		util.ExecutionRuntimeRoleDriver,
		util.ExecutionRuntimeRoleLauncher,
	)
}

// ToMLflowTerminalStatus converts a KFP RuntimeState string to an MLflow
// terminal status.
func ToMLflowTerminalStatus(stateV2 string) string {
	switch stateV2 {
	case "SUCCEEDED":
		return "FINISHED"
	case "CANCELED", "CANCELING":
		return "KILLED"
	default:
		return "FAILED"
	}
}

func resolvedMLflowTimeout(pluginConfig *commonmlflow.PluginConfig) time.Duration {
	defaultTimeout := 30 * time.Second
	if pluginConfig == nil || pluginConfig.Timeout == "" {
		return defaultTimeout
	}
	timeout, err := time.ParseDuration(pluginConfig.Timeout)
	if err != nil || timeout <= 0 {
		return defaultTimeout
	}
	return timeout
}
