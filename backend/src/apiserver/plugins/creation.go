// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"errors"
	"fmt"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// AnnotationKeyPluginParents records parent IDs without plugin configuration or credentials.
const AnnotationKeyPluginParents = "pipelines.kubeflow.org/plugin-parents"

// SetExecutionPluginParents records ownership before Kubernetes arbitrates run creation.
// An empty map distinguishes a creator with no parents from an older execution
// whose plugin resources are unknown.
func SetExecutionPluginParents(run *PendingRun, execution util.ExecutionSpec) error {
	outputs, err := DeserializePluginsOutput((*model.LargeText)(run.PluginsOutput))
	if err != nil {
		return err
	}
	parents := map[string]string{}
	for name, output := range outputs {
		if parentID := GetParentRunID(output); parentID != "" {
			parents[name] = parentID
		}
	}
	raw, err := json.Marshal(parents)
	if err != nil {
		return err
	}
	execution.SetAnnotations(AnnotationKeyPluginParents, string(raw))
	return nil
}

// OnRunCreationDiscarded finalizes parents belonging only to this request.
// Cleanup is best effort: failures are returned for logging, never persisted
// over the winning run's plugin output.
func (d *RunPluginDispatcherImpl) OnRunCreationDiscarded(ctx context.Context, run *PendingRun, existing util.ExecutionSpec) error {
	if d == nil || run == nil || existing == nil {
		return fmt.Errorf("dispatcher, run, and existing execution must be non-nil")
	}
	outputs, err := DeserializePluginsOutput((*model.LargeText)(run.PluginsOutput))
	if err != nil || len(outputs) == 0 {
		return err
	}
	var parents map[string]string
	raw := existing.ExecutionObjectMeta().Annotations[AnnotationKeyPluginParents]
	if err := json.Unmarshal([]byte(raw), &parents); err != nil || parents == nil {
		return fmt.Errorf("cannot clean up discarded creation for run %q: existing execution plugin parents are unknown", run.RunID)
	}
	discarded := &PersistedRun{
		RunID: run.RunID, Namespace: run.Namespace, State: string(model.RuntimeStateCanceled),
		PluginsOutput: map[string]*apiv2.PluginOutput{},
	}
	for name, output := range outputs {
		if parentID := GetParentRunID(output); parentID != "" && parentID != parents[name] {
			discarded.PluginsOutput[name] = output
		}
	}
	return d.finalizeUnpersistedParents(ctx, discarded, "discarded creation")
}

// OnRunCreationRejected finalizes, as failed, the parents created by a request
// whose run creation failed before a Workflow existed. Cleanup is best effort
// and never persists plugin output: a concurrent request with the same run ID
// may own the run row.
func (d *RunPluginDispatcherImpl) OnRunCreationRejected(ctx context.Context, run *PendingRun) error {
	if d == nil || run == nil {
		return fmt.Errorf("dispatcher and run must be non-nil")
	}
	outputs, err := DeserializePluginsOutput((*model.LargeText)(run.PluginsOutput))
	if err != nil || len(outputs) == 0 {
		return err
	}
	rejected := &PersistedRun{
		RunID: run.RunID, Namespace: run.Namespace, State: string(model.RuntimeStateFailed),
		PluginsOutput: map[string]*apiv2.PluginOutput{},
	}
	for name, output := range outputs {
		if GetParentRunID(output) != "" {
			rejected.PluginsOutput[name] = output
		}
	}
	return d.finalizeUnpersistedParents(ctx, rejected, "rejected creation")
}

// finalizeUnpersistedParents runs each handler's OnRunEnd for the parents in run
// without persisting the resulting plugin output.
func (d *RunPluginDispatcherImpl) finalizeUnpersistedParents(ctx context.Context, run *PersistedRun, label string) error {
	if len(run.PluginsOutput) == 0 {
		return nil
	}
	configs, err := d.RetrieveMultiUserModeConfigOverrides(ctx, run.RunID, run.Namespace)
	if err != nil {
		return err
	}
	var cleanupErrors []error
	for _, handler := range d.handlers {
		if run.PluginsOutput[handler.Name()] == nil {
			continue
		}
		cfg, err := handler.ResolveRunPluginConfig(ctx, d.kubeClients.GetClientSet(), configs[handler.Name()], run.Namespace)
		if err != nil || cfg == nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cannot resolve %s config for %s: %v", handler.Name(), label, err))
			continue
		}
		pluginCtx, cancel := context.WithTimeout(ctx, handler.GetPluginOperationTimeout(cfg))
		retryable, err := handler.OnRunEnd(pluginCtx, run, cfg)
		cancel()
		if err != nil || retryable || run.PluginsOutput[handler.Name()].GetState() == apiv2.PluginState_PLUGIN_FAILED {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to finalize %s parent for %s: %v", handler.Name(), label, err))
		}
	}
	return errors.Join(cleanupErrors...)
}
