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
	"context"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var _ apiserverPlugins.PluginDispatcher = (*Dispatcher)(nil)

// Dispatcher implements PluginDispatcher for MLflow.
type Dispatcher struct {
	kubeClients    KubeClientProvider
	runOutputStore apiserverPlugins.RunPluginOutputStore
}

// NewRunPluginDispatcher creates a new MLflow plugin dispatcher.
func NewRunPluginDispatcher(kubeClients KubeClientProvider, runOutputStore apiserverPlugins.RunPluginOutputStore) *Dispatcher {
	return &Dispatcher{
		kubeClients:    kubeClients,
		runOutputStore: runOutputStore,
	}
}

// OnBeforeRunCreation parses plugin input, resolves MLflow config,
// creates the MLflow experiment and parent run, and injects env vars
// into the execution spec. On success it writes run.PluginsOutput.
func (d *Dispatcher) OnBeforeRunCreation(ctx context.Context, run *apiserverPlugins.PendingRun, executionSpec util.ExecutionSpec) error {
	mlflowInput, err := ResolveMLflowPluginInput(run.PluginsInput)
	if err != nil {
		return util.NewBadRequestError(err, "Failed to create a run due to invalid MLflow plugin input")
	}
	if mlflowInput.Disabled {
		glog.V(4).Infof("MLflow is disabled for this run; skipping")
		return nil
	}

	selectedID, selectedName := SelectMLflowExperiment(mlflowInput, nil)
	if selectedID != "" {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_id=%q (create-by-name skipped)", selectedID)
	} else {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_name=%q", selectedName)
	}

	resolvedCfg, err := ResolveMLflowRequestConfig(ctx, d.kubeClients, run.Namespace)
	if err != nil {
		message := "MLflow config resolution failed; run creation will continue: " + err.Error()
		glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (%s)", run.RunID, message)
		pluginOutput := FailedPluginOutput(selectedID, selectedName, "", "", message)
		if outputErr := SetPendingRunPluginOutput(run, PluginName, pluginOutput); outputErr != nil {
			glog.Warningf("Failed to persist MLflow plugin output for run %q: %v", run.RunID, outputErr)
		}
		return nil
	}
	if resolvedCfg == nil {
		return nil
	}

	handler := NewHandler(mlflowInput, run.Namespace)
	// Keep the pre-run MLflow calls within the configured MLflow timeout while
	// still honoring parent request cancellation.
	mlflowCtx, cancel := context.WithTimeout(ctx, resolvedMLflowTimeout(resolvedCfg.Config))
	defer cancel()
	pluginOutput, pluginErr := handler.OnBeforeRunCreation(mlflowCtx, run, resolvedCfg)
	if pluginErr != nil {
		glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (run creation will continue): %v", run.RunID, pluginErr)
	}
	if pluginOutput == nil {
		return nil
	}
	if err := SetPendingRunPluginOutput(run, PluginName, pluginOutput); err != nil {
		glog.Warningf("Failed to persist MLflow plugin output for run %q: %v", run.RunID, err)
	}
	if err := InjectMLflowRuntimeEnv(executionSpec, handler.RunStartEnvVars); err != nil {
		glog.Warningf("Failed to inject MLflow runtime env for run %q: %v", run.RunID, err)
	}
	return nil
}

// OnRunEnd syncs the MLflow parent and nested runs at terminal state.
// Returns true if the sync succeeded or there is nothing to retry.
// Permanent failures (for example missing or invalid MLflow config) are
// recorded in the plugin output and reported as "nothing to retry" so a
// broken configuration cannot strand completed runs in terminal-report
// retry; only transient sync failures request a retry.
func (d *Dispatcher) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun) bool {
	mlflowOutput := run.PluginsOutput[PluginName]
	hasParentRun := mlflowOutput != nil && GetParentRunID(mlflowOutput) != ""

	syncOK, retryRequested, persisted := d.executePostAction(ctx, run, "OnRunEnd", func(mlflowCtx context.Context, h *Handler, r *apiserverPlugins.PersistedRun, cfg *ResolvedConfig) bool {
		retryable, err := h.OnRunEnd(mlflowCtx, r, cfg)
		if err != nil {
			glog.Warningf("MLflow OnRunEnd failed for run %q: %v", run.RunID, err)
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
}

// OnRunRetry reopens the MLflow parent run and any failed/killed nested runs.
func (d *Dispatcher) OnRunRetry(ctx context.Context, run *apiserverPlugins.PersistedRun) {
	d.executePostAction(ctx, run, "HandleRetry", func(mlflowCtx context.Context, h *Handler, r *apiserverPlugins.PersistedRun, cfg *ResolvedConfig) bool {
		h.HandleRetry(mlflowCtx, r, cfg)
		return false
	})
}

// executePostAction handles the common setup for OnRunEnd and HandleRetry.
// It reports whether the MLflow sync succeeded, whether the hook requested a
// retry for a transient failure, and whether the plugin output was persisted.
func (d *Dispatcher) executePostAction(
	ctx context.Context,
	run *apiserverPlugins.PersistedRun,
	hookName string,
	invoke func(context.Context, *Handler, *apiserverPlugins.PersistedRun, *ResolvedConfig) bool,
) (syncOK, retryRequested, persisted bool) {
	resolvedCfg, err := ResolveMLflowRequestConfig(ctx, d.kubeClients, run.Namespace)
	if err != nil {
		glog.Warningf("MLflow %s: failed to resolve config for namespace %q: %v", hookName, run.Namespace, err)
	}
	timeoutCfg := (*commonmlflow.PluginConfig)(nil)
	if resolvedCfg != nil {
		timeoutCfg = resolvedCfg.Config
	}
	mlflowCtx, cancel := context.WithTimeout(ctx, resolvedMLflowTimeout(timeoutCfg))
	defer cancel()

	handler := NewHandler(nil, run.Namespace)

	retryRequested = invoke(mlflowCtx, handler, run, resolvedCfg)

	syncOK = false
	if po := run.PluginsOutput[PluginName]; po != nil {
		syncOK = po.State == apiv2beta1.PluginState_PLUGIN_SUCCEEDED
	}

	if err := PersistPluginsOutput(run, d.runOutputStore); err != nil {
		glog.Warningf("MLflow %s: failed to persist plugin output for run %q: %v", hookName, run.RunID, err)
		return syncOK, retryRequested, false
	}
	return syncOK, retryRequested, true
}
