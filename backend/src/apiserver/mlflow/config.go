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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultExperimentName = "Default"
	DefaultTimeout        = "30s"
	LauncherConfigMapName = "kfp-launcher"
	LauncherConfigKey     = "plugins.mlflow"
	DefaultAuthType       = commonmlflow.DefaultAuthType
	AuthTypeNone          = commonmlflow.AuthTypeNone
	AuthTypeBearer        = commonmlflow.AuthTypeBearer
	AuthTypeBasicAuth     = commonmlflow.AuthTypeBasicAuth
	DefaultTokenKey       = "MLFLOW_TRACKING_TOKEN"
	DefaultUsernameKey    = "MLFLOW_TRACKING_USERNAME"
	DefaultPasswordKey    = "MLFLOW_TRACKING_PASSWORD"

	EnvTrackingURI = "KFP_MLFLOW_TRACKING_URI"
	EnvWorkspace   = "KFP_MLFLOW_WORKSPACE"
	EnvParentRunID = "KFP_MLFLOW_PARENT_RUN_ID"
	EnvAuthType    = "KFP_MLFLOW_AUTH_TYPE"
)

var ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// TLSConfig holds TLS settings for the MLflow endpoint.
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" mapstructure:"insecureSkipVerify"`
	CABundlePath       string `json:"caBundlePath,omitempty" mapstructure:"caBundlePath"`
}

// PluginConfig represents the global or namespace-level plugin configuration.
type PluginConfig struct {
	Endpoint string          `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Timeout  string          `json:"timeout,omitempty" mapstructure:"timeout"`
	TLS      *TLSConfig      `json:"tls,omitempty" mapstructure:"tls"`
	Settings json.RawMessage `json:"settings,omitempty" mapstructure:"settings"`
}

// CredentialSecretRef identifies the Kubernetes Secret and keys that hold credentials.
type CredentialSecretRef struct {
	Name        string `json:"name,omitempty"`
	TokenKey    string `json:"tokenKey,omitempty"`
	UsernameKey string `json:"usernameKey,omitempty"`
	PasswordKey string `json:"passwordKey,omitempty"`
}

// PluginSettings contains MLflow-specific settings parsed from PluginConfig.Settings.
type PluginSettings struct {
	WorkspacesEnabled     *bool                `json:"workspacesEnabled,omitempty"`
	AuthType              string               `json:"authType,omitempty"`
	CredentialSecretRef   *CredentialSecretRef `json:"credentialSecretRef,omitempty"`
	ExperimentDescription *string              `json:"experimentDescription,omitempty"`
}

// ResolvedConfig bundles the merged plugin configuration and its parsed settings.
type ResolvedConfig struct {
	Settings *PluginSettings
	Config   *PluginConfig
}

// AuthMaterial holds the resolved authentication credentials for an MLflow endpoint.
type AuthMaterial struct {
	AuthType      string
	BearerToken   string
	BasicUsername string
	BasicPassword string
}

// PluginInput represents the user-facing plugins_input.mlflow schema.
type PluginInput struct {
	ExperimentName string `json:"experiment_name,omitempty"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
}

// KubeClientProvider abstracts the Kubernetes clientset access so the mlflow
// package does not depend on the resource package's concrete types.
type KubeClientProvider interface {
	GetClientSet() kubernetes.Interface
}

// GetGlobalPluginConfig reads the global plugins.mlflow configuration
func GetGlobalPluginConfig() (PluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return PluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return PluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return PluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

// GetNamespacePluginConfig reads the namespace-level MLflow configuration
// from the kfp-launcher ConfigMap.
func GetNamespacePluginConfig(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*PluginConfig, bool, error) {
	if namespace == "" || clientSet == nil {
		return nil, false, nil
	}
	cm, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, LauncherConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, util.NewInternalServerError(err, "failed to read namespace configmap %q in namespace %q", LauncherConfigMapName, namespace)
	}
	raw, ok := cm.Data[LauncherConfigKey]
	if !ok || raw == "" {
		return nil, false, nil
	}
	var cfg PluginConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, false, util.NewInvalidInputError("failed to parse namespace %q %s as JSON: %v", namespace, LauncherConfigKey, err)
	}
	return &cfg, true, nil
}

// MergePluginConfig merges namespace-level overrides into the global config.
func MergePluginConfig(globalCfg PluginConfig, namespaceCfg *PluginConfig) PluginConfig {
	merged := globalCfg
	if namespaceCfg == nil {
		return merged
	}
	if namespaceCfg.Endpoint != "" {
		merged.Endpoint = namespaceCfg.Endpoint
	}
	if namespaceCfg.Timeout != "" {
		merged.Timeout = namespaceCfg.Timeout
	}
	if namespaceCfg.TLS != nil {
		merged.TLS = namespaceCfg.TLS
	}
	if len(namespaceCfg.Settings) != 0 {
		merged.Settings = namespaceCfg.Settings
	}
	return merged
}

// ParsePluginSettings deserializes the raw settings JSON into PluginSettings,
// applying defaults for AuthType and WorkspacesEnabled.
func ParsePluginSettings(raw json.RawMessage) (*PluginSettings, error) {
	settings := &PluginSettings{
		AuthType: DefaultAuthType,
	}
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, settings); err != nil {
			return nil, util.NewInvalidInputError("failed to parse plugins.mlflow.settings: %v", err)
		}
	}
	if settings.AuthType == "" {
		settings.AuthType = DefaultAuthType
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := settings.AuthType == DefaultAuthType
		settings.WorkspacesEnabled = &defaultEnabled
	}
	return settings, nil
}

// ResolveRequestConfig builds a merged and validated ResolvedConfig for the
// given namespace, combining global and namespace-level configuration.
func ResolveRequestConfig(ctx context.Context, kubeClients KubeClientProvider, namespace string) (*ResolvedConfig, error) {
	globalCfg, hasGlobal, err := GetGlobalPluginConfig()
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
	namespaceCfg, hasNamespace, err := GetNamespacePluginConfig(ctx, clientSet, namespace)
	if err != nil {
		return nil, err
	}
	mergedCfg := MergePluginConfig(globalCfg, namespaceCfg)
	if mergedCfg.Timeout == "" {
		mergedCfg.Timeout = DefaultTimeout
	}
	settings, err := ParsePluginSettings(mergedCfg.Settings)
	if err != nil {
		return nil, err
	}
	if hasNamespace && namespaceCfg != nil && namespaceCfg.Endpoint != "" && globalCfg.Endpoint != namespaceCfg.Endpoint {
		if settings.CredentialSecretRef == nil || settings.CredentialSecretRef.Name == "" {
			return nil, util.NewInvalidInputError("namespace plugins.mlflow endpoint override requires namespace credentialSecretRef")
		}
	}
	return &ResolvedConfig{
		Settings: settings,
		Config:   &mergedCfg,
	}, nil
}

// ResolveAuthMaterial resolves the authentication credentials for the MLflow
// endpoint based on the configured authType.
func ResolveAuthMaterial(ctx context.Context, clientSet kubernetes.Interface, namespace string, settings *PluginSettings) (AuthMaterial, error) {
	authType := DefaultAuthType
	if settings != nil && settings.AuthType != "" {
		authType = strings.ToLower(settings.AuthType)
	}
	switch authType {
	case AuthTypeNone:
		return AuthMaterial{AuthType: authType}, nil
	case DefaultAuthType:
		tokenBytes, err := os.ReadFile(ServiceAccountTokenPath)
		if err != nil {
			return AuthMaterial{}, util.NewInternalServerError(err, "failed to read API server service account token for MLflow auth")
		}
		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			return AuthMaterial{}, util.NewInvalidInputError("API server service account token is empty for MLflow auth")
		}
		return AuthMaterial{
			AuthType:    authType,
			BearerToken: token,
		}, nil
	case AuthTypeBearer:
		secretData, secretRef, err := GetCredentialSecretData(ctx, clientSet, namespace, settings, authType)
		if err != nil {
			return AuthMaterial{}, err
		}
		tokenKey := secretRef.TokenKey
		if tokenKey == "" {
			tokenKey = DefaultTokenKey
		}
		token := strings.TrimSpace(string(secretData[tokenKey]))
		if token == "" {
			return AuthMaterial{}, util.NewInvalidInputError("plugins.mlflow credential secret %q key %q must be non-empty for bearer auth", secretRef.Name, tokenKey)
		}
		return AuthMaterial{
			AuthType:    authType,
			BearerToken: token,
		}, nil
	case AuthTypeBasicAuth:
		secretData, secretRef, err := GetCredentialSecretData(ctx, clientSet, namespace, settings, authType)
		if err != nil {
			return AuthMaterial{}, err
		}
		usernameKey := secretRef.UsernameKey
		if usernameKey == "" {
			usernameKey = DefaultUsernameKey
		}
		passwordKey := secretRef.PasswordKey
		if passwordKey == "" {
			passwordKey = DefaultPasswordKey
		}
		username := strings.TrimSpace(string(secretData[usernameKey]))
		password := strings.TrimSpace(string(secretData[passwordKey]))
		if username == "" {
			return AuthMaterial{}, util.NewInvalidInputError("plugins.mlflow credential secret %q key %q must be non-empty for basic-auth", secretRef.Name, usernameKey)
		}
		if password == "" {
			return AuthMaterial{}, util.NewInvalidInputError("plugins.mlflow credential secret %q key %q must be non-empty for basic-auth", secretRef.Name, passwordKey)
		}
		return AuthMaterial{
			AuthType:      authType,
			BasicUsername: username,
			BasicPassword: password,
		}, nil
	default:
		return AuthMaterial{}, util.NewInvalidInputError("unsupported plugins.mlflow.settings.authType %q (expected one of %q, %q, %q, %q)", authType, AuthTypeNone, DefaultAuthType, AuthTypeBearer, AuthTypeBasicAuth)
	}
}

// GetCredentialSecretData reads and returns the data from the credential Secret
// referenced in settings.
func GetCredentialSecretData(ctx context.Context, clientSet kubernetes.Interface, namespace string, settings *PluginSettings, authType string) (map[string][]byte, *CredentialSecretRef, error) {
	if namespace == "" {
		return nil, nil, util.NewInvalidInputError("namespace must be set for MLflow %s auth", authType)
	}
	if settings == nil || settings.CredentialSecretRef == nil || settings.CredentialSecretRef.Name == "" {
		return nil, nil, util.NewInvalidInputError("plugins.mlflow.settings.credentialSecretRef.name is required for authType %q", authType)
	}
	if clientSet == nil {
		return nil, nil, util.NewInternalServerError(errors.New("kubernetes clientset is nil"), "failed to read MLflow credential secret")
	}
	secretRef := settings.CredentialSecretRef
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, secretRef.Name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil, util.NewInvalidInputError("plugins.mlflow credential secret %q not found in namespace %q", secretRef.Name, namespace)
		}
		return nil, nil, util.NewInternalServerError(err, "failed to read MLflow credential secret %q in namespace %q", secretRef.Name, namespace)
	}
	return secret.Data, secretRef, nil
}

// BuildHTTPClient configures an http.Client with the given timeout and TLS settings.
func BuildHTTPClient(timeout time.Duration, tlsCfg *TLSConfig) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
		}
		if tlsCfg.CABundlePath != "" {
			caBundle, err := os.ReadFile(tlsCfg.CABundlePath)
			if err != nil {
				return nil, util.NewInvalidInputError("failed to read plugins.mlflow.tls.caBundlePath %q: %v", tlsCfg.CABundlePath, err)
			}
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caBundle) {
				return nil, util.NewInvalidInputError("plugins.mlflow.tls.caBundlePath %q did not contain valid PEM certificates", tlsCfg.CABundlePath)
			}
			tlsConfig.RootCAs = certPool
		}
		transport.TLSClientConfig = tlsConfig
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// BuildRequestContext constructs a fully initialized RequestContext including
// the underlying MLflow HTTP client with authentication.
func BuildRequestContext(ctx context.Context, clientSet kubernetes.Interface, namespace string, requestCfg *ResolvedConfig) (*RequestContext, error) {
	if requestCfg == nil || requestCfg.Config == nil || requestCfg.Config.Endpoint == "" {
		return nil, nil
	}
	baseURL, err := url.Parse(requestCfg.Config.Endpoint)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow endpoint %q", requestCfg.Config.Endpoint)
	}
	settings := requestCfg.Settings
	if settings == nil {
		settings, err = ParsePluginSettings(requestCfg.Config.Settings)
		if err != nil {
			return nil, err
		}
	}
	timeout, err := time.ParseDuration(requestCfg.Config.Timeout)
	if err != nil {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow timeout %q: %v", requestCfg.Config.Timeout, err)
	}
	if timeout <= 0 {
		return nil, util.NewInvalidInputError("plugins.mlflow timeout must be > 0")
	}
	authMaterial, err := ResolveAuthMaterial(ctx, clientSet, namespace, settings)
	if err != nil {
		return nil, err
	}
	httpClient, err := BuildHTTPClient(timeout, requestCfg.Config.TLS)
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
		AuthType:          authMaterial.AuthType,
		BearerToken:       authMaterial.BearerToken,
		BasicAuthUsername: authMaterial.BasicUsername,
		BasicAuthPassword: authMaterial.BasicPassword,
		WorkspacesEnabled: workspacesEnabled,
		Workspace:         namespace,
		Retry:             retrySettings,
	})
	if err != nil {
		return nil, util.NewInvalidInputError("failed to build MLflow client: %v", err)
	}
	return &RequestContext{
		BaseURL:           baseURL,
		Workspace:         namespace,
		WorkspacesEnabled: workspacesEnabled,
		Client:            sharedClient,
	}, nil
}

// ResolvePluginInput parses the plugins_input.mlflow JSON from a run model,
// validates it against the PluginInput schema, and applies defaults.
func ResolvePluginInput(pluginsInputString *string) (*PluginInput, error) {
	if pluginsInputString == nil || *pluginsInputString == "" {
		return &PluginInput{ExperimentName: DefaultExperimentName}, nil
	}

	pluginInputs := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	mlflowRaw, ok := pluginInputs["mlflow"]
	if !ok || len(mlflowRaw) == 0 {
		return &PluginInput{ExperimentName: DefaultExperimentName}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(mlflowRaw))
	decoder.DisallowUnknownFields()
	input := &PluginInput{}
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

// SelectExperiment chooses the selector used for MLflow experiment resolution.
// experiment_id takes precedence over experiment_name.
func SelectExperiment(input *PluginInput) (experimentID string, experimentName string) {
	if input == nil {
		return "", DefaultExperimentName
	}
	if input.ExperimentID != "" {
		return input.ExperimentID, ""
	}
	if input.ExperimentName == "" {
		return "", DefaultExperimentName
	}
	return "", input.ExperimentName
}

// ToMLflowTerminalStatus converts a KFP RuntimeState to an MLflow terminal
// status string.
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

// InjectRuntimeEnv injects MLflow environment variables into driver and
// launcher Argo Workflow container templates.
func InjectRuntimeEnv(executionSpec util.ExecutionSpec, env map[string]string) error {
	if len(env) == 0 || executionSpec == nil {
		return nil
	}
	workflow, ok := executionSpec.(*util.Workflow)
	if !ok {
		return util.NewInternalServerError(errors.New("unsupported execution spec type"), "failed to inject MLflow runtime env: unsupported execution spec")
	}
	workflow.SetEnvVarsToDriverAndLauncherTemplates(env)
	return nil
}
