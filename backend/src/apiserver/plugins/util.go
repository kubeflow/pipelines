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
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/encoding/protojson"
)

type PluginsOutputEnvelope struct {
	Plugins map[string]json.RawMessage
}

func (e *PluginsOutputEnvelope) UnmarshalJSON(data []byte) error {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	if len(all) > 0 {
		e.Plugins = all
	}
	return nil
}

func (e PluginsOutputEnvelope) MarshalJSON() ([]byte, error) {
	if len(e.Plugins) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(e.Plugins)
}

// set stores a plugin entry by name.
func (e *PluginsOutputEnvelope) set(name string, data json.RawMessage) {
	if e.Plugins == nil {
		e.Plugins = make(map[string]json.RawMessage)
	}
	e.Plugins[name] = data
}

func (e *PluginsOutputEnvelope) forEachEntry(fn func(name string, payload json.RawMessage)) {
	for name, payload := range e.Plugins {
		fn(name, payload)
	}
}

const (
	EntryRootRunID = "root_run_id"
	EntryRunURL    = "run_url"
)

type RunSyncMode string

const (
	RunSyncModeTerminal RunSyncMode = "terminal"
	RunSyncModeRetry    RunSyncMode = "retry"
)

// BuildKFPRunURL builds a link from kfpBaseURL to the pipeline run details page.
func BuildKFPRunURL(runID, namespace, kfpBaseURL, pathTemplate string) string {
	if runID == "" || kfpBaseURL == "" {
		glog.V(4).Infof(
			"BuildKFPRunURL returned empty URL due to missing input(s): runID_empty=%t kfpBaseURL_empty=%t",
			runID == "",
			kfpBaseURL == "",
		)
		return ""
	}
	pathTemplate = strings.TrimSpace(pathTemplate)
	if pathTemplate == "" {
		base := strings.TrimRight(kfpBaseURL, "/")
		return fmt.Sprintf("%s/#/runs/details/%s", base, url.PathEscape(runID))
	}
	if namespace == "" && strings.Contains(pathTemplate, "{namespace}") {
		glog.V(4).Infof("BuildKFPRunURL returned empty URL: namespace required when template contains {namespace}")
		return ""
	}
	base := strings.TrimRight(kfpBaseURL, "/")
	rendered := strings.ReplaceAll(pathTemplate, "{run_id}", url.PathEscape(runID))
	rendered = strings.ReplaceAll(rendered, "{namespace}", url.PathEscape(namespace))
	if !strings.HasPrefix(rendered, "/") && !strings.HasPrefix(rendered, "#") {
		rendered = "/" + rendered
	}
	return base + rendered
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

// ModelToPersistedRun converts a model.Run to a PersistedRun for the
// post-run plugin hooks (OnRunEnd, OnRunRetry).
func ModelToPersistedRun(m *model.Run, namespace string) (*PersistedRun, error) {
	if m == nil {
		return nil, fmt.Errorf("model.Run is nil")
	}
	pluginsOutput, err := DeserializePluginsOutput(m.PluginsOutputString)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize plugins_output for run %q: %w", m.UUID, err)
	}
	pr := &PersistedRun{
		RunID:         m.UUID,
		Namespace:     namespace,
		State:         string(m.RunDetails.State), //nolint:staticcheck // QF1008
		PluginsOutput: pluginsOutput,
	}
	if m.RunDetails.FinishedAtInSec > 0 { //nolint:staticcheck // QF1008
		t := time.Unix(m.RunDetails.FinishedAtInSec, 0) //nolint:staticcheck // QF1008
		pr.FinishedAt = &t
	}
	return pr, nil
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

func DeserializePluginsOutput(raw *model.LargeText) (map[string]*apiv2beta1.PluginOutput, error) {
	result := make(map[string]*apiv2beta1.PluginOutput)
	if raw == nil || *raw == "" {
		return result, nil
	}
	var envelope PluginsOutputEnvelope
	if err := json.Unmarshal([]byte(*raw), &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugins_output: %w", err)
	}
	envelope.forEachEntry(func(name string, payload json.RawMessage) {
		output := &apiv2beta1.PluginOutput{}
		if err := protojson.Unmarshal(payload, output); err == nil {
			result[name] = output
		}
	})
	return result, nil
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

// PersistPluginsOutput serializes the PersistedRun's PluginsOutput and writes
// it to the database via the given store.
func PersistPluginsOutput(run *PersistedRun, store RunPluginOutputStore) error {
	lt, err := SerializePluginsOutput(run.PluginsOutput)
	if err != nil {
		return fmt.Errorf("failed to serialize plugins_output for run %q: %w", run.RunID, err)
	}
	return store.UpdateRunPluginsOutput(run.RunID, lt)
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

func GetParentRunID(output *apiv2beta1.PluginOutput) string {
	return GetStringEntry(output, EntryRootRunID)
}
