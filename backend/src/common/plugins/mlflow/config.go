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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
)

// EnvMLflowConfig is the single environment variable injected into Argo
// Workflow templates by the API server.
const EnvMLflowConfig = "KFP_MLFLOW_CONFIG"

const (
	EnvMLflowTrackingAuth     = "MLFLOW_TRACKING_AUTH"
	EnvMLflowTrackingToken    = "MLFLOW_TRACKING_TOKEN"
	EnvMLflowTrackingUsername = "MLFLOW_TRACKING_USERNAME"
	EnvMLflowTrackingPassword = "MLFLOW_TRACKING_PASSWORD"
)

// CredentialSecretName is the fixed Kubernetes Secret name for secret-based MLflow auth.
const CredentialSecretName = "kfp-mlflow-credentials"

// TagNestedRunParentRunID is the MLflow tag used for nested run parent linkage.
const TagNestedRunParentRunID = "mlflow.parentRunId"

// MLflowRuntimeConfig is the JSON payload marshaled into KFP_MLFLOW_CONFIG.
type MLflowRuntimeConfig struct {
	Endpoint            string               `json:"endpoint"`
	WorkspacesEnabled   bool                 `json:"workspacesEnabled,omitempty"`
	Workspace           string               `json:"workspace,omitempty"`
	ParentRunID         string               `json:"parentRunId"`
	ExperimentID        string               `json:"experimentId"`
	AuthType            string               `json:"authType"`
	CredentialSecretRef *CredentialSecretRef `json:"credentialSecretRef,omitempty"`
	Timeout             string               `json:"timeout,omitempty"`
	InsecureSkipVerify  bool                 `json:"insecureSkipVerify,omitempty"`
	InjectUserEnvVars   bool                 `json:"injectUserEnvVars,omitempty"`
	TLS                 *TLSConfig           `json:"tls,omitempty" mapstructure:"tls"`
}

// TLSConfig holds TLS settings for the MLflow endpoint.
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" mapstructure:"insecureSkipVerify"`
	CABundlePath       string `json:"caBundlePath,omitempty" mapstructure:"caBundlePath"`
}

// PluginConfig represents the global or namespace-level plugin configuration.
type PluginConfig struct {
	Endpoint string                `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Timeout  string                `json:"timeout,omitempty" mapstructure:"timeout"`
	TLS      *TLSConfig            `json:"tls,omitempty" mapstructure:"tls"`
	Settings *MLflowPluginSettings `json:"settings,omitempty" mapstructure:"settings"`
}

// CredentialSecretRef identifies the Secret keys used for MLflow auth.
type CredentialSecretRef struct {
	TokenKey    string `json:"tokenKey,omitempty"`
	UsernameKey string `json:"usernameKey,omitempty"`
	PasswordKey string `json:"passwordKey,omitempty"`
}

func buildSecretEnvVar(name, secretKey string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: CredentialSecretName},
				Key:                  secretKey,
			},
		},
	}
}

// BuildCredentialEnvVars converts secret-based MLflow auth configuration into
// Kubernetes env vars backed by SecretKeyRef entries.
func BuildCredentialEnvVars(ref *CredentialSecretRef, authType string) ([]corev1.EnvVar, error) {
	if ref == nil {
		switch authType {
		case AuthTypeBearer, AuthTypeBasicAuth:
			return nil, fmt.Errorf("credentialSecretRef is required for auth type %q", authType)
		default:
			return nil, nil
		}
	}
	switch authType {
	case AuthTypeBearer:
		if ref.TokenKey == "" {
			return nil, fmt.Errorf("credentialSecretRef.tokenKey is required for auth type %q", authType)
		}
		return []corev1.EnvVar{buildSecretEnvVar(EnvMLflowTrackingToken, ref.TokenKey)}, nil
	case AuthTypeBasicAuth:
		if ref.UsernameKey == "" {
			return nil, fmt.Errorf("credentialSecretRef.usernameKey is required for auth type %q", authType)
		}
		if ref.PasswordKey == "" {
			return nil, fmt.Errorf("credentialSecretRef.passwordKey is required for auth type %q", authType)
		}
		return []corev1.EnvVar{
			buildSecretEnvVar(EnvMLflowTrackingUsername, ref.UsernameKey),
			buildSecretEnvVar(EnvMLflowTrackingPassword, ref.PasswordKey),
		}, nil
	default:
		return nil, nil
	}
}

func (r *CredentialSecretRef) UnmarshalJSON(data []byte) error {
	type credentialSecretRef CredentialSecretRef
	var raw struct {
		credentialSecretRef
		Name string `json:"name,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Name != "" {
		return fmt.Errorf("credentialSecretRef.name is not supported; Secret name is fixed to %q", CredentialSecretName)
	}
	*r = CredentialSecretRef(raw.credentialSecretRef)
	return nil
}

// MLflowCredentials holds the resolved authentication credentials for an MLflow endpoint.
type MLflowCredentials struct {
	AuthType    string
	BearerToken string
	Username    string
	Password    string
}

// RequestContext holds a fully resolved MLflow connection: the parsed
// endpoint URL, the shared HTTP client, and workspace settings.
type RequestContext struct {
	BaseURL           *url.URL
	Client            *Client
	Workspace         string
	WorkspacesEnabled bool
}

// MLflowPluginSettings contains MLflow-specific settings parsed from
// PluginConfig.Settings.
type MLflowPluginSettings struct {
	WorkspacesEnabled     *bool                `json:"workspacesEnabled,omitempty"`
	AuthType              string               `json:"authType,omitempty"`
	CredentialSecretRef   *CredentialSecretRef `json:"credentialSecretRef,omitempty"`
	ExperimentDescription *string              `json:"experimentDescription,omitempty"`
	DefaultExperimentName string               `json:"defaultExperimentName,omitempty"`
	KFPBaseURL            string               `json:"kfpBaseURL,omitempty"`
	KFPRunURLPathTemplate string               `json:"kfpRunURLPathTemplate,omitempty"`
	MLflowBaseURL         string               `json:"mlflowBaseURL,omitempty"`
	MLflowUIPathPrefix    string               `json:"mlflowUIPathPrefix,omitempty"`
	InjectUserEnvVars     *bool                `json:"injectUserEnvVars,omitempty"`
}

// MergePluginConfig merges namespace-level overrides into the global config.
// The namespace config takes precedence on non-zero fields.
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
	merged.Settings = mergeSettings(merged.Settings, namespaceCfg.Settings)
	return merged
}

// mergeSettings performs a field-level merge of two MLflowPluginSettings.
// Namespace values override global values for non-zero fields.
func mergeSettings(global, namespace *MLflowPluginSettings) *MLflowPluginSettings {
	if namespace == nil {
		return global
	}
	if global == nil {
		return namespace
	}
	merged := *global // shallow copy
	if namespace.WorkspacesEnabled != nil {
		merged.WorkspacesEnabled = namespace.WorkspacesEnabled
	}
	if namespace.AuthType != "" {
		merged.AuthType = namespace.AuthType
	}
	if namespace.CredentialSecretRef != nil {
		merged.CredentialSecretRef = namespace.CredentialSecretRef
	}
	if namespace.ExperimentDescription != nil {
		merged.ExperimentDescription = namespace.ExperimentDescription
	}
	if namespace.DefaultExperimentName != "" {
		merged.DefaultExperimentName = namespace.DefaultExperimentName
	}
	if namespace.KFPBaseURL != "" {
		merged.KFPBaseURL = namespace.KFPBaseURL
	}
	if namespace.KFPRunURLPathTemplate != "" {
		merged.KFPRunURLPathTemplate = namespace.KFPRunURLPathTemplate
	}
	if namespace.MLflowBaseURL != "" {
		merged.MLflowBaseURL = namespace.MLflowBaseURL
	}
	if namespace.MLflowUIPathPrefix != "" {
		merged.MLflowUIPathPrefix = namespace.MLflowUIPathPrefix
	}
	if namespace.InjectUserEnvVars != nil {
		merged.InjectUserEnvVars = namespace.InjectUserEnvVars
	}
	return &merged
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
				return nil, fmt.Errorf("failed to read plugins.mlflow.tls.caBundlePath %q: %w", tlsCfg.CABundlePath, err)
			}
			certPool, err := x509.SystemCertPool()
			if err != nil {
				certPool = x509.NewCertPool()
			}
			if !certPool.AppendCertsFromPEM(caBundle) {
				return nil, fmt.Errorf("plugins.mlflow.tls.caBundlePath %q did not contain valid PEM certificates", tlsCfg.CABundlePath)
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

// ResolveMLflowCredentials resolves the Kubernetes service account token used
// to authenticate with the MLflow endpoint.
func ResolveMLflowCredentials() (MLflowCredentials, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return MLflowCredentials{}, util.NewInternalServerError(err, "failed to get Kubernetes config for MLflow auth")
	}
	token := restConfig.BearerToken
	if token == "" && restConfig.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			return MLflowCredentials{}, util.NewInternalServerError(err, "failed to read bearer token file %q for MLflow auth", restConfig.BearerTokenFile)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}
	if token == "" {
		return MLflowCredentials{}, util.NewInvalidInputError("Kubernetes bearer token is empty for MLflow auth")
	}
	return MLflowCredentials{
		AuthType:    AuthTypeKubernetes,
		BearerToken: token,
	}, nil
}

// ResolveRuntimeMLflowCredentials resolves the runtime credentials available to
// the driver and launcher based on the configured auth type.
func ResolveRuntimeMLflowCredentials(authType string) (MLflowCredentials, error) {
	switch authType {
	case AuthTypeKubernetes:
		return ResolveMLflowCredentials()
	case AuthTypeBearer:
		token := strings.TrimSpace(os.Getenv(EnvMLflowTrackingToken))
		if token == "" {
			return MLflowCredentials{}, util.NewInvalidInputError("%s is empty for MLflow bearer auth", EnvMLflowTrackingToken)
		}
		return MLflowCredentials{
			AuthType:    AuthTypeBearer,
			BearerToken: token,
		}, nil
	case AuthTypeBasicAuth:
		username := strings.TrimSpace(os.Getenv(EnvMLflowTrackingUsername))
		password := strings.TrimSpace(os.Getenv(EnvMLflowTrackingPassword))
		if username == "" {
			return MLflowCredentials{}, util.NewInvalidInputError("%s is empty for MLflow basic auth", EnvMLflowTrackingUsername)
		}
		if password == "" {
			return MLflowCredentials{}, util.NewInvalidInputError("%s is empty for MLflow basic auth", EnvMLflowTrackingPassword)
		}
		return MLflowCredentials{
			AuthType: AuthTypeBasicAuth,
			Username: username,
			Password: password,
		}, nil
	case AuthTypeNone:
		return MLflowCredentials{AuthType: AuthTypeNone}, nil
	default:
		return MLflowCredentials{}, util.NewInvalidInputError("unsupported MLflow auth type %q", authType)
	}
}

// BuildMLflowRequestContext is the shared core that validates the PluginConfig,
// resolves credentials, builds the HTTP client and MLflow client, and returns
// a ready-to-use RequestContext. The workspace and workspacesEnabled values
// are caller-specific and passed in directly.
func BuildMLflowRequestContext(
	pluginCfg PluginConfig,
	authMaterial MLflowCredentials,
	workspace string,
	workspacesEnabled bool,
) (*RequestContext, error) {
	baseURL, err := url.Parse(pluginCfg.Endpoint)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow endpoint %q", pluginCfg.Endpoint)
	}
	timeout, err := time.ParseDuration(pluginCfg.Timeout)
	if err != nil {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow timeout %q: %v", pluginCfg.Timeout, err)
	}
	if timeout <= 0 {
		return nil, util.NewInvalidInputError("plugins.mlflow timeout must be > 0")
	}
	httpClient, err := BuildHTTPClient(timeout, pluginCfg.TLS)
	if err != nil {
		return nil, err
	}
	retrySettings := RetryPolicy{
		InitialInterval: DefaultRetryInitial,
		MaxInterval:     DefaultRetryMax,
		MaxElapsedTime:  DefaultRetryElapsed,
		Multiplier:      2.0,
	}
	sharedClient, err := NewClient(Config{
		Endpoint:          pluginCfg.Endpoint,
		HTTPClient:        httpClient,
		BearerToken:       authMaterial.BearerToken,
		Username:          authMaterial.Username,
		Password:          authMaterial.Password,
		WorkspacesEnabled: workspacesEnabled,
		Workspace:         workspace,
		Retry:             retrySettings,
	})
	if err != nil {
		return nil, util.NewInvalidInputError("failed to build MLflow client: %v", err)
	}
	return &RequestContext{
		BaseURL:           baseURL,
		Workspace:         workspace,
		WorkspacesEnabled: workspacesEnabled,
		Client:            sharedClient,
	}, nil
}
