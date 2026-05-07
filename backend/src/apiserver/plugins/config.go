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
	"encoding/json"
	"fmt"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/encoding/protojson"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

const (
	EntryRootRunID = "root_run_id"
)

const (
	LauncherConfigMapName   = "kfp-launcher"
	LauncherConfigKeyPrefix = "plugins."
)

const (
	// DefaultTimeout is the default HTTP request timeout for the plugin client.
	DefaultTimeout = "30s"
)

// InjectPluginRuntimeEnv upserts plugin-provided environment variables into the
// driver and launcher containers of the execution spec.
func InjectPluginRuntimeEnv(executionSpec util.ExecutionSpec, env map[string]string) error {
	if len(env) == 0 || executionSpec == nil {
		return nil
	}
	return executionSpec.UpsertRuntimeEnvVars(env,
		util.ExecutionRuntimeRoleDriver,
		util.ExecutionRuntimeRoleLauncher,
	)
}

// GetNamespacePluginConfig reads the namespace-level Plugin configuration
// from the kfp-launcher ConfigMap.  Returns nil (no error) when the ConfigMap
// or key is absent.
func GetNamespacePluginConfig(ctx context.Context, clientSet kubernetes.Interface, pluginName, namespace string) (*PluginConfig, error) {
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
		return nil, util.NewInternalServerError(err, "failed to read %s plugin namespace config from configmap %q in namespace %q", pluginName, LauncherConfigMapName, namespace)
	}
	launcherConfigKey := LauncherConfigKeyPrefix + pluginName
	raw, ok := cm.Data[launcherConfigKey]
	if !ok || raw == "" {
		return nil, nil
	}
	var cfg PluginConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, util.NewInternalServerError(err, "failed to parse %s plugin config from key %q in configmap %q/%q", pluginName, launcherConfigKey, namespace, LauncherConfigMapName)
	}
	return &cfg, nil
}

type PluginsOutputEnvelope struct {
	others map[string]json.RawMessage
}

func (e *PluginsOutputEnvelope) UnmarshalJSON(data []byte) error {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	if len(all) > 0 {
		e.others = all
	}
	return nil
}

func (e PluginsOutputEnvelope) MarshalJSON() ([]byte, error) {
	if len(e.others) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(e.others)
}

// set stores a plugin entry by name.
func (e *PluginsOutputEnvelope) set(name string, data json.RawMessage) {
	if e.others == nil {
		e.others = make(map[string]json.RawMessage)
	}
	e.others[name] = data
}

// SetPendingRunPluginOutput serializes the given PluginOutput into PendingRun.PluginsOutput.
func SetPendingRunPluginOutput(run *PendingRun, pluginName string, output *apiv2beta1.PluginOutput) error {
	if run == nil || output == nil || pluginName == "" {
		return nil
	}
	result, err := upsertPluginOutput(run.PluginsOutput, pluginName, output)
	if err != nil {
		return err
	}
	run.PluginsOutput = &result
	return nil
}

// upsertPluginOutput merges a single plugin's output into an existing
// plugins_output JSON string, returning the updated JSON.
func upsertPluginOutput(existing *string, pluginName string, output *apiv2beta1.PluginOutput) (string, error) {
	marshaledOutput, err := protojson.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugin output for %q: %w", pluginName, err)
	}
	var envelope PluginsOutputEnvelope
	if existing != nil && *existing != "" {
		if err := json.Unmarshal([]byte(*existing), &envelope); err != nil {
			return "", fmt.Errorf("failed to unmarshal existing plugins_output: %w", err)
		}
	}
	envelope.set(pluginName, marshaledOutput)
	marshaledMap, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugins_output map: %w", err)
	}
	return string(marshaledMap), nil
}

func GetParentRunID(output *apiv2beta1.PluginOutput) string {
	return GetStringEntry(output, EntryRootRunID)
}

func GetStringEntry(output *apiv2beta1.PluginOutput, key string) string {
	if output == nil || output.Entries == nil || key == "" {
		return ""
	}
	entry, ok := output.Entries[key]
	if !ok || entry == nil || entry.Value == nil {
		return ""
	}
	return entry.Value.GetStringValue()
}

// PersistPluginsOutput serializes the PersistedRun's PluginsOutput and writes
// it to the database via the given store.
func PersistPluginsOutput(run *PersistedRun, store RunPluginOutputStore) error {
	lt, err := SerializePluginsOutput(run.PluginsOutput)
	if err != nil {
		return fmt.Errorf("failed to serialize plugins_output for run %q: %w", run.RunID, err)
	}
	return store.UpdateRunPluginsOutput(run.RunID, lt)
}

func SerializePluginsOutput(outputs map[string]*apiv2beta1.PluginOutput) (*model.LargeText, error) {
	if len(outputs) == 0 {
		return nil, nil
	}
	var envelope PluginsOutputEnvelope
	for key, output := range outputs {
		marshaledOutput, err := protojson.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal plugin output for %q: %w", key, err)
		}
		envelope.set(key, marshaledOutput)
	}
	marshaledMap, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugins_output map: %w", err)
	}
	lt := model.LargeText(string(marshaledMap))
	return &lt, nil
}

// ResolvePluginRequestConfig builds the effective PluginConfig for a single
// request by merging namespace-level overrides on top of the global config.
// Returns (nil, nil) when neither source provides configuration, signaling
// that the plugin is unconfigured rather than broken.
func ResolvePluginRequestConfig(ctx context.Context, clientSet kubernetes.Interface, handler RunPluginHandler, namespace string) (*PluginConfig, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler is nil")
	}

	globalCfg, err := handler.GetGlobalPluginConfig()
	if err != nil {
		return nil, err
	}

	namespaceCfg, err := GetNamespacePluginConfig(ctx, clientSet, handler.Name(), namespace)
	if err != nil {
		return nil, err
	}

	merged, err := MergePluginConfig(namespaceCfg, globalCfg)
	if err != nil {
		return nil, err
	}
	if merged == nil {
		return nil, nil
	}
	if merged.Timeout == "" {
		merged.Timeout = DefaultTimeout
	}
	return merged, nil
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
