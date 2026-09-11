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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
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
		PluginsOutput: map[string]*apiv2beta1.PluginOutput{},
	}
	for name, output := range outputs {
		if parentID := GetParentRunID(output); parentID != "" && parentID != parents[name] {
			discarded.PluginsOutput[name] = output
		}
	}
	if len(discarded.PluginsOutput) == 0 {
		return nil
	}
	configs, err := d.RetrieveMultiUserModeConfigOverrides(ctx, run.RunID, run.Namespace)
	if err != nil {
		return err
	}
	var cleanupErrors []error
	for _, handler := range d.handlers {
		if discarded.PluginsOutput[handler.Name()] == nil {
			continue
		}
		cfg, err := handler.ResolveRunPluginConfig(ctx, d.kubeClients.GetClientSet(), configs[handler.Name()], run.Namespace)
		if err != nil || cfg == nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cannot resolve %s config for discarded creation: %v", handler.Name(), err))
			continue
		}
		pluginCtx, cancel := context.WithTimeout(ctx, handler.GetPluginOperationTimeout(cfg))
		retryable, err := handler.OnRunEnd(pluginCtx, discarded, cfg)
		cancel()
		if err != nil || retryable || discarded.PluginsOutput[handler.Name()].GetState() == apiv2beta1.PluginState_PLUGIN_FAILED {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to finalize %s parent for discarded creation: %v", handler.Name(), err))
		}
	}
	return errors.Join(cleanupErrors...)
}
