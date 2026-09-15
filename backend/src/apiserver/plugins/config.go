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

package plugins

import (
	"context"
	"fmt"
	"strings"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	LauncherConfigMapName   = "kfp-launcher"
	LauncherConfigKeyPrefix = "plugins."
)

// KubeClientProvider abstracts Kubernetes clientset access.
type KubeClientProvider interface {
	GetClientSet() kubernetes.Interface
}

type PluginConfig struct {
	Endpoint string                   `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Timeout  string                   `json:"timeout,omitempty" mapstructure:"timeout"`
	TLS      *commonplugins.TLSConfig `json:"tls,omitempty" mapstructure:"tls"`
	Settings map[string]interface{}   `json:"settings,omitempty" mapstructure:"settings"`
}

// InjectPluginRuntimeEnv upserts plugin-provided environment variables into the
// driver and launcher containers of the execution spec.
func InjectPluginRuntimeEnv(executionSpec util.ExecutionSpec, envVars []corev1.EnvVar) error {
	if len(envVars) == 0 || executionSpec == nil {
		return nil
	}
	return executionSpec.UpsertRuntimeEnvVars(envVars,
		util.ExecutionRuntimeRoleDriver,
		util.ExecutionRuntimeRoleLauncher,
	)
}

// GetLauncherNamespacePluginConfigsMap retrieves a map of plugin configurations from a Kubernetes ConfigMap in a given namespace.
// It requires a non-empty namespace and a valid Kubernetes clientset to access the ConfigMap.
// Returns a map of plugin configurations or an error if the ConfigMap retrieval or parsing fails.
func GetLauncherNamespacePluginConfigsMap(ctx context.Context, clientSet kubernetes.Interface, namespace string) (map[string]string, error) {
	if namespace == "" {
		return nil, util.NewInternalServerError(fmt.Errorf("namespace is empty"), "namespace must be specified when reading Plugin config")
	}
	if clientSet == nil {
		return nil, util.NewInternalServerError(fmt.Errorf("clientSet is nil"), "Kubernetes clientset must be provided when reading Plugin namespace config")
	}
	cm, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, LauncherConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, util.NewInternalServerError(err, "failed to read MLflow namespace config from configmap %q in namespace %q", LauncherConfigMapName, namespace)
	}

	cfgMap := make(map[string]string)
	for key, value := range cm.Data {
		if strings.HasPrefix(key, LauncherConfigKeyPrefix) {
			cfgMap[strings.TrimPrefix(key, LauncherConfigKeyPrefix)] = value
		}
	}
	return cfgMap, nil
}

// MergePluginConfig merges namespace-level overrides into the global config.
// The namespace config takes precedence on non-zero fields.
// Returns (nil, nil) when both inputs are nil (plugin unconfigured).
func MergePluginConfig(namespaceCfg *PluginConfig, globalCfg *PluginConfig) (*PluginConfig, error) {
	if globalCfg == nil && namespaceCfg == nil {
		return nil, nil
	}
	if globalCfg == nil {
		return namespaceCfg, nil
	}
	merged := &PluginConfig{
		Endpoint: globalCfg.Endpoint,
		Timeout:  globalCfg.Timeout,
		TLS:      globalCfg.TLS,
		Settings: globalCfg.Settings,
	}
	if namespaceCfg == nil {
		return merged, nil
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
	merged.Settings = mergeSettings(namespaceCfg.Settings, merged.Settings)
	return merged, nil
}

func mergeSettings(ns, global map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(global)+len(ns))
	for key, value := range global {
		merged[key] = value
	}
	for key, value := range ns {
		merged[key] = value
	}
	return merged
}

func SetPluginOutputState(output *apiv2beta1.PluginOutput, state apiv2beta1.PluginState, stateMessage string) {
	if output == nil {
		return
	}
	output.State = state
	output.StateMessage = stateMessage
}
