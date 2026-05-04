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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	LauncherConfigMapName = "kfp-launcher"
	LauncherConfigKey     = "plugins.mlflow"
)

// EnvMLflowConfig is the single environment variable injected into Argo
// Workflow templates by the API server.
const EnvMLflowConfig = "KFP_MLFLOW_CONFIG"

// MLflow tag on nested runs (parent linkage).
const TagNestedRunParentRunID = "mlflow.parentRunId"

// MLflowRuntimeConfig is the JSON payload marshalled into KFP_MLFLOW_CONFIG.
type MLflowRuntimeConfig struct {
	Endpoint           string `json:"endpoint"`
	Workspace          string `json:"workspace,omitempty"`
	ParentRunID        string `json:"parentRunId"`
	ExperimentID       string `json:"experimentId"`
	AuthType           string `json:"authType"`
	Timeout            string `json:"timeout,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	InjectUserEnvVars  bool   `json:"injectUserEnvVars,omitempty"`
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

// MLflowCredentials holds the resolved authentication credentials for an MLflow endpoint.
type MLflowCredentials struct {
	AuthType    string
	BearerToken string
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
	WorkspacesEnabled     *bool   `json:"workspacesEnabled,omitempty"`
	ExperimentDescription *string `json:"experimentDescription,omitempty"`
	DefaultExperimentName string  `json:"defaultExperimentName,omitempty"`
	KFPBaseURL            string  `json:"kfpBaseURL,omitempty"`
	InjectUserEnvVars     *bool   `json:"injectUserEnvVars,omitempty"`
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
	if namespace.ExperimentDescription != nil {
		merged.ExperimentDescription = namespace.ExperimentDescription
	}
	if namespace.DefaultExperimentName != "" {
		merged.DefaultExperimentName = namespace.DefaultExperimentName
	}
	if namespace.KFPBaseURL != "" {
		merged.KFPBaseURL = namespace.KFPBaseURL
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
