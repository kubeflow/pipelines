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

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// RunPluginOutputStore abstracts the DB write needed to persist plugin output.
type RunPluginOutputStore interface {
	UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error
}

// RunPluginDispatcher orchestrates plugin lifecycle hooks.
type RunPluginDispatcher interface {
	// PluginsRegistered returns true if one or more plugins are successfully registered.
	PluginsRegistered() bool

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

var _ RunPluginDispatcher = NoOpDispatcher{}

func (NoOpDispatcher) PluginsRegistered() bool { return false }
func (NoOpDispatcher) OnBeforeRunCreation(context.Context, *PendingRun, util.ExecutionSpec) error {
	return nil
}
func (NoOpDispatcher) OnRunEnd(context.Context, *PersistedRun) bool { return true }
func (NoOpDispatcher) OnRunRetry(context.Context, *PersistedRun) error {
	return nil
}

var _ RunPluginDispatcher = (*RunPluginDispatcherImpl)(nil)

// NewRunPluginDispatcherImpl initializes a new RunPluginDispatcherImpl with the given handlers, client provider, and output store.
// Returns an error if handlers are nil or empty.
func NewRunPluginDispatcherImpl(handlers []RunPluginHandler, kubeClients KubeClientProvider, runOutputStore RunPluginOutputStore) (*RunPluginDispatcherImpl, error) {
	if len(handlers) == 0 {
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

// RunPluginDispatcherImpl implements RunPluginDispatcher.
type RunPluginDispatcherImpl struct {
	handlers       []RunPluginHandler
	kubeClients    KubeClientProvider
	runOutputStore RunPluginOutputStore
}

// PluginsRegistered returns true if or more plugins are successfully registered.
func (d *RunPluginDispatcherImpl) PluginsRegistered() bool { return true }

// OnBeforeRunCreation parses plugin input, resolves namespace-level plugin config,
// and creates the parent run and injects env vars
// into the execution for each registered handler. On success it writes run.PluginsOutput.
func (d *RunPluginDispatcherImpl) OnBeforeRunCreation(ctx context.Context, run *PendingRun, executionSpec util.ExecutionSpec) error {
	if d == nil || run == nil || executionSpec == nil {
		return fmt.Errorf("dispatcher, run, and executionSpec must be non-nil")
	}

	// Retrieve run-level plugin inputs across all registered plugins
	pluginInputs := map[string]interface{}{}
	for _, handler := range d.handlers {
		input, enabled, err := handler.ResolveRunPluginInput(run.PluginsInput)
		if err != nil {
			return util.NewBadRequestError(err, "Failed to create a run due to invalid %s plugin input", handler.Name())
		}
		if !enabled {
			glog.V(4).Infof("%s is disabled for this run; skipping", handler.Name())
			continue
		}
		pluginInputs[handler.Name()] = input
	}

	// Retrieve launcher namespace plugin configs across all registered plugins, if multi-user mode is enabled.
	launcherNamespacePluginCfgs, err := d.RetrieveMultiUserModeConfigOverrides(ctx, run.RunID, run.Namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve launcher namespace plugin configs for run %q: %v", run.RunID, err)
	}
	for _, handler := range d.handlers {
		func(ctx context.Context, run *PendingRun, executionSpec util.ExecutionSpec, handler RunPluginHandler) {
			input := pluginInputs[handler.Name()]

			launcherNamespacePluginCfgStr := launcherNamespacePluginCfgs[handler.Name()]
			resolvedRunPluginCfg, err := handler.ResolveRunPluginConfig(ctx, d.kubeClients.GetClientSet(), launcherNamespacePluginCfgStr, run.Namespace)
			if err != nil {
				message := fmt.Sprintf("%s config resolution failed; run creation will continue: ", handler.Name()) + err.Error()
				glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (%s)", run.RunID, message)
				pluginOutput := handler.GetGenericFailedPluginOutput("", message, input)
				if outputErr := SetPendingRunPluginOutput(run, handler.Name(), pluginOutput); outputErr != nil {
					glog.Warningf("Failed to persist %s plugin output for run %q: %v", handler.Name(), run.RunID, outputErr)
				}
				return
			}
			if resolvedRunPluginCfg == nil {
				return
			}
			// Size the context to several per-call timeouts so the idempotent create
			// retries in the client can fit within the budget, while still honoring
			// parent request cancellation.
			pluginOperationBudget := handler.GetPluginOperationTimeout(resolvedRunPluginCfg)
			pluginCtx, cancel := context.WithTimeout(ctx, pluginOperationBudget)
			defer cancel()
			pluginOutput, pluginRuntimeEnv, pluginErr := handler.OnBeforeRunCreation(pluginCtx, run, resolvedRunPluginCfg, input)
			if pluginErr != nil {
				glog.Warningf("%s OnBeforeRunCreation failed for run %q (run creation will continue): %v", handler.Name(), run.RunID, pluginErr)
			}
			if pluginOutput == nil {
				return
			}
			if err := SetPendingRunPluginOutput(run, handler.Name(), pluginOutput); err != nil {
				glog.Warningf("failed to persist %s plugin output for run %q: %v", handler.Name(), run.RunID, err)
			}
			if len(pluginRuntimeEnv) != 0 {
				if err := InjectPluginRuntimeEnv(executionSpec, pluginRuntimeEnv); err != nil {
					glog.Warningf("Failed to inject %s runtime env for run %q: %v", handler.Name(), run.RunID, err)
				}
			}
		}(ctx, run, executionSpec, handler)
	}
	return nil
}

// OnRunEnd syncs the plugin parent and nested runs at terminal state.
// Returns true if the sync succeeded or there is nothing to retry.
// Permanent failures (for example missing or invalid plugin config) are
// recorded in the plugin output and reported as "nothing to retry" so a
// broken configuration cannot strand completed runs in terminal-report
// retry; only transient sync failures request a retry.
func (d *RunPluginDispatcherImpl) OnRunEnd(ctx context.Context, run *PersistedRun) bool {
	if d == nil || run == nil {
		glog.Errorf("dispatcher and input run must be non-nil")
		return false
	}

	launcherNamespacePluginCfgs, err := d.RetrieveMultiUserModeConfigOverrides(ctx, run.RunID, run.Namespace)
	if err != nil {
		glog.Errorf("failed to retrieve launcher namespace plugin configs for run %q: %v", run.RunID, err)
		return false
	}
	allSucceeded := true
	for _, handler := range d.handlers {
		succeeded := func(ctx context.Context, run *PersistedRun, handler RunPluginHandler) bool {
			pluginOutput := run.PluginsOutput[handler.Name()]
			if pluginOutput == nil {
				return true
			}

			launcherNamespacePluginCfgStr := launcherNamespacePluginCfgs[handler.Name()]
			runPluginCfg, err := handler.ResolveRunPluginConfig(ctx, d.kubeClients.GetClientSet(), launcherNamespacePluginCfgStr, run.Namespace)
			if err != nil {
				glog.Warningf("failed to resolve plugin config for %s on run %q (run creation will continue): %v", handler.Name(), run.RunID, err)
			}

			hasParentRun := GetParentRunID(pluginOutput) != ""
			pluginOperationBudget := handler.GetPluginOperationTimeout(runPluginCfg)
			pluginCtx, cancel := context.WithTimeout(ctx, pluginOperationBudget)
			defer cancel()
			syncOK, retryRequested, persisted := d.executePostAction(run, "OnRunEnd", runPluginCfg, handler.Name(), func(r *PersistedRun, cfg interface{}) bool {
				retryable, err := handler.OnRunEnd(pluginCtx, r, cfg)
				if err != nil {
					glog.Warningf("%s OnRunEnd failed for run %q: %v", handler.Name(), run.RunID, err)
				}
				return retryable
			})

			// The plugin outcome could not be recorded; retry so it is not lost.
			if !persisted {
				return false
			}
			// Only signal retry-needed when a transient failure left a parent run open.
			if hasParentRun && !syncOK && retryRequested {
				return false
			}
			return true
		}(ctx, run, handler)
		if !succeeded {
			allSucceeded = false
		}
	}
	return allSucceeded
}

// OnRunRetry reopens the parent run and any failed/killed nested runs.
func (d *RunPluginDispatcherImpl) OnRunRetry(ctx context.Context, run *PersistedRun) error {
	if d == nil || run == nil {
		return fmt.Errorf("dispatcher and run must be non-nil")
	}

	launcherNamespacePluginCfgs, err := d.RetrieveMultiUserModeConfigOverrides(ctx, run.RunID, run.Namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve launcher namespace plugin configs for run %q: %v", run.RunID, err)
	}
	for _, handler := range d.handlers {
		func(ctx context.Context, run *PersistedRun, handler RunPluginHandler) {
			if run.PluginsOutput[handler.Name()] == nil {
				glog.Warningf("Plugin output not found for %s plugin", handler.Name())
			}
			launcherNamespacePluginCfgStr := launcherNamespacePluginCfgs[handler.Name()]
			runPluginCfg, err := handler.ResolveRunPluginConfig(ctx, d.kubeClients.GetClientSet(), launcherNamespacePluginCfgStr, run.Namespace)
			if err != nil {
				glog.Warningf("failed to resolve plugin config for %s on run %q (run creation will continue): %v", handler.Name(), run.RunID, err)
			}
			pluginOperationBudget := handler.GetPluginOperationTimeout(runPluginCfg)
			pluginCtx, cancel := context.WithTimeout(ctx, pluginOperationBudget)
			defer cancel()
			d.executePostAction(run, "HandleRetry", runPluginCfg, handler.Name(), func(r *PersistedRun, cfg interface{}) bool {
				if err = handler.HandleRetry(pluginCtx, r, cfg); err != nil {
					glog.Warningf("%s OnRunRetry failed for run %q: %w", handler.Name(), run.RunID, err)
				}
				return false
			})
		}(ctx, run, handler)
	}

	return nil
}

// executePostAction handles the common setup for OnRunEnd and HandleRetry.
// It reports whether the plugin sync succeeded, whether the hook requested a
// retry for a transient failure, and whether the plugin output was persisted.
func (d *RunPluginDispatcherImpl) executePostAction(
	run *PersistedRun,
	hookName string,
	runCfg interface{},
	pluginName string,
	invoke func(*PersistedRun, interface{}) bool,
) (syncOK, retryRequested, persisted bool) {
	if runCfg == nil {
		glog.Warningf("%s %s skipped: resolved plugin config is nil for run %q", pluginName, hookName, run.RunID)
		if po := run.PluginsOutput[pluginName]; po != nil {
			po.State = apiv2beta1.PluginState_PLUGIN_FAILED
			po.StateMessage = fmt.Sprintf("%s %s sync failed: config unavailable", pluginName, hookName)
		}
		persistenceErr := PersistPluginsOutput(run, d.runOutputStore)
		if persistenceErr != nil {
			glog.Warningf("%s %s: failed to persist plugin output for run %q: %v", pluginName, hookName, run.RunID, persistenceErr)
		}
		return false, false, persistenceErr == nil
	}

	retryRequested = invoke(run, runCfg)

	syncOK = false
	if po := run.PluginsOutput[pluginName]; po != nil {
		syncOK = po.State == apiv2beta1.PluginState_PLUGIN_SUCCEEDED
	}

	if err := PersistPluginsOutput(run, d.runOutputStore); err != nil {
		glog.Warningf("%s %s: failed to persist plugin output for run %q: %v", pluginName, hookName, run.RunID, err)
		return syncOK, retryRequested, false
	}
	return syncOK, retryRequested, true
}

func (d *RunPluginDispatcherImpl) RetrieveMultiUserModeConfigOverrides(ctx context.Context, runID string, namespace string) (map[string]string, error) {
	if common.IsMultiUserMode() {
		launcherNamespacePluginCfgs, err := GetLauncherNamespacePluginConfigsMap(ctx, d.kubeClients.GetClientSet(), namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve launcher namespace plugin configs for run %q: %v", runID, err)
		}
		return launcherNamespacePluginCfgs, nil
	}
	return nil, nil
}
