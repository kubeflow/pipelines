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
	"net/url"
	"os"
	"strings"
	"time"

	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
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

// ApplySettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplySettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := true
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

// ResolvedConfig bundles the merged plugin configuration and its parsed settings.
type ResolvedConfig struct {
	Settings *commonmlflow.MLflowPluginSettings
	Config   *commonmlflow.PluginConfig
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

// GetNamespaceMLflowConfig reads the namespace-level MLflow configuration
// from the kfp-launcher ConfigMap.  Returns nil (no error) when the ConfigMap
// or key is absent.
func GetNamespaceMLflowConfig(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*commonmlflow.PluginConfig, error) {
	if namespace == "" {
		return nil, util.NewInternalServerError(fmt.Errorf("namespace is empty"), "namespace must be specified when reading MLflow config")
	}
	if clientSet == nil {
		return nil, util.NewInternalServerError(fmt.Errorf("clientSet is nil"), "Kubernetes clientset must be provided when reading MLflow namespace config")
	}
	cm, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, commonmlflow.LauncherConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, util.NewInternalServerError(err, "failed to read MLflow namespace config from configmap %q in namespace %q", commonmlflow.LauncherConfigMapName, namespace)
	}
	raw, ok := cm.Data[commonmlflow.LauncherConfigKey]
	if !ok || raw == "" {
		return nil, nil
	}
	var cfg commonmlflow.PluginConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, util.NewInternalServerError(err, "failed to parse MLflow config from key %q in configmap %q/%q", commonmlflow.LauncherConfigKey, namespace, commonmlflow.LauncherConfigMapName)
	}
	return &cfg, nil
}

// ResolveMLflowRequestConfig builds a merged and validated ResolvedConfig for the
// given namespace, combining global and namespace-level configuration.
func ResolveMLflowRequestConfig(ctx context.Context, kubeClients KubeClientProvider, namespace string) (*ResolvedConfig, error) {
	globalCfg, hasGlobal, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !hasGlobal {
		return nil, nil
	}

	namespaceCfg, err := GetNamespaceMLflowConfig(ctx, kubeClients.GetClientSet(), namespace)
	if err != nil {
		return nil, err
	}
	mergedCfg := commonmlflow.MergePluginConfig(globalCfg, namespaceCfg)
	if mergedCfg.Timeout == "" {
		mergedCfg.Timeout = DefaultTimeout
	}
	settings := ApplySettingsDefaults(mergedCfg.Settings)
	return &ResolvedConfig{
		Settings: settings,
		Config:   &mergedCfg,
	}, nil
}

// ResolveMLflowCredentials resolves the Kubernetes service account token used
// to authenticate with the MLflow endpoint.
func ResolveMLflowCredentials() (commonmlflow.MLflowCredentials, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return commonmlflow.MLflowCredentials{}, util.NewInternalServerError(err, "failed to get Kubernetes config for MLflow auth")
	}
	token := restConfig.BearerToken
	if token == "" && restConfig.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			return commonmlflow.MLflowCredentials{}, util.NewInternalServerError(err, "failed to read bearer token file %q for MLflow auth", restConfig.BearerTokenFile)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}
	if token == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError("Kubernetes bearer token is empty for MLflow auth")
	}
	return commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: token,
	}, nil
}

// BuildMLflowRequestContext constructs a fully initialized RequestContext including
// the underlying MLflow HTTP client with authentication.
func BuildMLflowRequestContext(ctx context.Context, namespace string, requestCfg *ResolvedConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil || requestCfg.Config == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "cannot build MLflow request context without a resolved config")
	}
	if requestCfg.Config.Endpoint == "" {
		return nil, util.NewInvalidInputError("plugins.mlflow endpoint must be set")
	}
	baseURL, err := url.Parse(requestCfg.Config.Endpoint)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow endpoint %q", requestCfg.Config.Endpoint)
	}
	settings := requestCfg.Settings
	if settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "BuildMLflowRequestContext requires resolved settings")
	}
	timeout, err := time.ParseDuration(requestCfg.Config.Timeout)
	if err != nil {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow timeout %q: %v", requestCfg.Config.Timeout, err)
	}
	if timeout <= 0 {
		return nil, util.NewInvalidInputError("plugins.mlflow timeout must be > 0")
	}
	authMaterial, err := ResolveMLflowCredentials()
	if err != nil {
		return nil, err
	}
	httpClient, err := commonmlflow.BuildHTTPClient(timeout, requestCfg.Config.TLS)
	if err != nil {
		return nil, err
	}
	workspacesEnabled := settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled
	retrySettings := commonmlflow.RetryPolicy{
		InitialInterval: commonmlflow.DefaultRetryInitial,
		MaxInterval:     commonmlflow.DefaultRetryMax,
		MaxElapsedTime:  commonmlflow.DefaultRetryElapsed,
		Multiplier:      2.0,
	}
	sharedClient, err := commonmlflow.NewClient(commonmlflow.Config{
		Endpoint:          requestCfg.Config.Endpoint,
		HTTPClient:        httpClient,
		BearerToken:       authMaterial.BearerToken,
		WorkspacesEnabled: workspacesEnabled,
		Workspace:         namespace,
		Retry:             retrySettings,
	})
	if err != nil {
		return nil, util.NewInvalidInputError("failed to build MLflow client: %v", err)
	}
	return &commonmlflow.RequestContext{
		BaseURL:           baseURL,
		Workspace:         namespace,
		WorkspacesEnabled: workspacesEnabled,
		Client:            sharedClient,
	}, nil
}

// ResolveMLflowPluginInput parses the plugins_input.mlflow JSON from a run model,
// validates it against the MLflowPluginInput schema, and applies defaults.
func ResolveMLflowPluginInput(pluginsInputString *string) (*MLflowPluginInput, error) {
	if pluginsInputString == nil || *pluginsInputString == "" {
		return &MLflowPluginInput{ExperimentName: DefaultExperimentName}, nil
	}

	var pluginInputs apiserverPlugins.PluginsInputMap
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	mlflowRaw, ok := pluginInputs["mlflow"]
	if !ok || len(mlflowRaw) == 0 {
		return &MLflowPluginInput{ExperimentName: DefaultExperimentName}, nil
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

	if input.Disabled {
		return input, nil
	}
	if input.ExperimentID != "" {
		return input, nil
	}
	if input.ExperimentName == "" {
		input.ExperimentName = DefaultExperimentName
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

// InjectMLflowRuntimeEnv sets KFP_MLFLOW_CONFIG on driver and launcher
// containers.
func InjectMLflowRuntimeEnv(executionSpec util.ExecutionSpec, env map[string]string) error {
	if len(env) == 0 || executionSpec == nil {
		return nil
	}
	return executionSpec.UpsertRuntimeEnvVars(env,
		util.ExecutionRuntimeRoleDriver,
		util.ExecutionRuntimeRoleLauncher,
	)
}
