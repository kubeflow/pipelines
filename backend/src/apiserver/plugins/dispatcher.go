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
	"sort"
	"time"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// RunPluginOutputStore abstracts the DB write needed to persist plugin output.
type RunPluginOutputStore interface {
	UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error
}

// RunPluginDispatcher orchestrates plugin lifecycle hooks.
type RunPluginDispatcher interface {
	// OnBeforeRunCreation is called before the workflow is created.
	// The dispatcher reads run.PluginsInput and may write run.PluginsOutput.
	// Returns an error only for validation failures that should block
	// run creation.
	OnBeforeRunCreation(ctx context.Context, run *PendingRun, executionSpec util.ExecutionSpec) error

	// OnRunEnd is called when a run reaches a terminal state. Returns
	// true if all plugin syncs succeeded.
	OnRunEnd(ctx context.Context, run *PersistedRun) bool

	// OnRunRetry is called when a run is retried.
	OnRunRetry(ctx context.Context, run *PersistedRun) error
}

// NoOpDispatcher is a RunPluginDispatcher that does nothing.
type NoOpDispatcher struct{}

func (NoOpDispatcher) OnBeforeRunCreation(context.Context, *PendingRun, util.ExecutionSpec) error {
	return nil
}
func (NoOpDispatcher) OnRunEnd(context.Context, *PersistedRun) bool { return true }
func (NoOpDispatcher) OnRunRetry(context.Context, *PersistedRun) error {
	return nil
}

var _ RunPluginDispatcher = NoOpDispatcher{}

// NewRunPluginDispatcherImpl initializes a new RunPluginDispatcherImpl with the given handlers, client provider, and output store.
// Returns an error if handlers are nil or empty.
func NewRunPluginDispatcherImpl(handlers []RunPluginHandler, kubeClients KubeClientProvider, runOutputStore RunPluginOutputStore) (*RunPluginDispatcherImpl, error) {
	if handlers == nil || len(handlers) == 0 {
		return nil, fmt.Errorf("NewRunPluginDispatcherImpl requires non-nil slice containing minimum one handler")
	}
	sorted := make([]RunPluginHandler, len(handlers))
	copy(sorted, handlers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name() < sorted[j].Name()
	})
	return &RunPluginDispatcherImpl{
		handlers:       sorted,
		kubeClients:    kubeClients,
		runOutputStore: runOutputStore,
	}, nil
}

var _ RunPluginDispatcher = (*RunPluginDispatcherImpl)(nil)

// RunPluginDispatcherImpl implements RunPluginDispatcher.
type RunPluginDispatcherImpl struct {
	handlers       []RunPluginHandler
	kubeClients    KubeClientProvider
	runOutputStore RunPluginOutputStore
}

const preRunPluginTimeout = 3 * time.Second

// OnBeforeRunCreation parses plugin input, resolves namespace-level plugin config,
// and creates the parent run and injects env vars
// into the execution for each registered handler. On success it writes run.PluginsOutput.
func (d *RunPluginDispatcherImpl) OnBeforeRunCreation(ctx context.Context, run *PendingRun, executionSpec util.ExecutionSpec) error {
	if d == nil || run == nil || executionSpec == nil {
		return fmt.Errorf("dispatcher, run, and executionSpec must be non-nil")
	}

	// Limit plugin pre-run calls to a short timeout budget while still
	// honoring parent request cancellation.
	pluginCtx, cancel := context.WithTimeout(ctx, preRunPluginTimeout)
	defer cancel()
	for _, handler := range d.handlers {
		runPluginCfg, err := ResolvePluginRequestConfig(pluginCtx, d.kubeClients.GetClientSet(), handler, run.Namespace)
		if err != nil {
			glog.Warningf("Failed to resolve plugin config for %s on run %q (run creation will continue): %v", handler.Name(), run.RunID, err)
			continue
		}
		pluginOutput, pluginRuntimeEnv, pluginErr := handler.OnBeforeRunCreation(pluginCtx, run, runPluginCfg)
		if pluginErr != nil {
			glog.Warningf("%s OnBeforeRunCreation failed for run %q (run creation will continue): %v", handler.Name(), run.RunID, pluginErr)
		}
		if pluginOutput == nil {
			continue
		}
		if err := SetPendingRunPluginOutput(run, handler.Name(), pluginOutput); err != nil {
			glog.Warningf("Failed to persist %s plugin output for run %q: %v", handler.Name(), run.RunID, err)
		}
		if len(pluginRuntimeEnv) != 0 {
			if err := InjectPluginRuntimeEnv(executionSpec, pluginRuntimeEnv); err != nil {
				glog.Warningf("Failed to inject %s runtime env for run %q: %v", handler.Name(), run.RunID, err)
			}
		}
	}

	return nil
}

// OnRunEnd syncs the plugin parent and nested runs at terminal state.
// Returns true if the sync succeeded or there is nothing to retry.
func (d *RunPluginDispatcherImpl) OnRunEnd(ctx context.Context, run *PersistedRun) bool {
	for _, handler := range d.handlers {
		pluginOutput := run.PluginsOutput[handler.Name()]
		if pluginOutput == nil {
			continue
		}

		runPluginCfg, err := ResolvePluginRequestConfig(ctx, d.kubeClients.GetClientSet(), handler, run.Namespace)
		if err != nil {
			glog.Warningf("Failed to resolve plugin config for %s on run %q (run end sync will be skipped): %v", handler.Name(), run.RunID, err)
			continue
		}

		hasParentRun := GetParentRunID(pluginOutput) != ""

		syncOK := d.executePostAction(run, "OnRunEnd", runPluginCfg, handler.Name(), func(r *PersistedRun, cfg *PluginConfig) {
			if err := handler.OnRunEnd(ctx, r, cfg); err != nil {
				glog.Warningf("%s OnRunEnd failed for run %q: %v", handler.Name(), run.RunID, err)
			}
		})

		if hasParentRun && !syncOK {
			return false
		}
	}
	return true
}

// OnRunRetry reopens the parent run and any failed/killed nested runs.
func (d *RunPluginDispatcherImpl) OnRunRetry(ctx context.Context, run *PersistedRun) error {
	if d == nil || run == nil {
		return fmt.Errorf("dispatcher and run must be non-nil")
	}

	for _, handler := range d.handlers {
		if run.PluginsOutput[handler.Name()] == nil {
			continue
		}
		runPluginCfg, err := ResolvePluginRequestConfig(ctx, d.kubeClients.GetClientSet(), handler, run.Namespace)
		if err != nil {
			glog.Warningf("Failed to resolve plugin config for %s on run %q (retry will be skipped): %v", handler.Name(), run.RunID, err)
			continue
		}
		d.executePostAction(run, "HandleRetry", runPluginCfg, handler.Name(), func(r *PersistedRun, c *PluginConfig) {
			handler.HandleRetry(ctx, r, c)
		})
	}
	return nil
}

// executePostAction handles the common setup for OnRunEnd and HandleRetry.
// Returns true if the plugin sync succeeded.
func (d *RunPluginDispatcherImpl) executePostAction(
	run *PersistedRun,
	hookName string,
	runCfg *PluginConfig,
	pluginName string,
	invoke func(*PersistedRun, *PluginConfig),
) bool {
	if runCfg == nil {
		glog.Warningf("%s %s skipped: resolved plugin config is nil for run %q", pluginName, hookName, run.RunID)
		if po := run.PluginsOutput[pluginName]; po != nil {
			po.State = apiv2beta1.PluginState_PLUGIN_FAILED
			po.StateMessage = fmt.Sprintf("%s %s sync failed: config unavailable", pluginName, hookName)
		}
		if err := PersistPluginsOutput(run, d.runOutputStore); err != nil {
			glog.Warningf("%s %s: failed to persist plugin output for run %q: %v", pluginName, hookName, run.RunID, err)
		}
		return false
	}

	invoke(run, runCfg)

	pluginSyncOK := false
	if po := run.PluginsOutput[pluginName]; po != nil {
		pluginSyncOK = po.State == apiv2beta1.PluginState_PLUGIN_SUCCEEDED
	}

	if err := PersistPluginsOutput(run, d.runOutputStore); err != nil {
		glog.Warningf("%s %s: failed to persist plugin output for run %q: %v", pluginName, hookName, run.RunID, err)
		return false
	}
	return pluginSyncOK
}
