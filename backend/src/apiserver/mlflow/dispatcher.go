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
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var _ apiserverPlugins.PluginDispatcher = (*Dispatcher)(nil)

// Dispatcher implements PluginDispatcher for MLflow.
type Dispatcher struct {
	kubeClients    KubeClientProvider
	runOutputStore apiserverPlugins.RunPluginOutputStore
}

// NewDispatcher creates a new MLflow plugin dispatcher.
func NewDispatcher(kubeClients KubeClientProvider, runOutputStore apiserverPlugins.RunPluginOutputStore) *Dispatcher {
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
		return util.NewInternalServerError(err, "MLflow config resolution failed for run %q", run.RunID)
	}
	if resolvedCfg == nil {
		return nil
	}

	handler := NewHandler(mlflowInput, run.Namespace)
	pluginOutput, pluginErr := handler.OnBeforeRunCreation(ctx, run, resolvedCfg.Config)
	if pluginErr != nil {
		glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (run creation will continue): %v", run.RunID, pluginErr)
	}
	if pluginOutput == nil {
		return nil
	}
	if err := SetPendingRunPluginOutput(run, PluginName, pluginOutput); err != nil {
		glog.Warningf("Failed to persist MLflow plugin output for run %q: %v", run.RunID, err)
	}
	if len(handler.RunStartEnv) != 0 {
		if err := InjectMLflowRuntimeEnv(executionSpec, handler.RunStartEnv); err != nil {
			glog.Warningf("Failed to inject MLflow runtime env for run %q: %v", run.RunID, err)
		}
	}
	return nil
}

// OnRunEnd syncs the MLflow parent and nested runs at terminal state.
// Returns true if the sync succeeded or there is nothing to retry.
func (d *Dispatcher) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun) bool {
	mlflowOutput := run.PluginsOutput[PluginName]
	hasParentRun := mlflowOutput != nil && GetParentRunID(mlflowOutput) != ""

	syncOK := d.executePostAction(ctx, run, "OnRunEnd", func(h *Handler, r *apiserverPlugins.PersistedRun, cfg *commonmlflow.PluginConfig) {
		if err := h.OnRunEnd(ctx, r, cfg); err != nil {
			glog.Warningf("MLflow OnRunEnd failed for run %q: %v", run.RunID, err)
		}
	})

	// Only signal retry-needed when there's a parent run that failed to close.
	if hasParentRun && !syncOK {
		return false
	}
	return true
}

// OnRunRetry reopens the MLflow parent run and any failed/killed nested runs.
func (d *Dispatcher) OnRunRetry(ctx context.Context, run *apiserverPlugins.PersistedRun) {
	d.executePostAction(ctx, run, "HandleRetry", func(h *Handler, r *apiserverPlugins.PersistedRun, cfg *commonmlflow.PluginConfig) {
		h.HandleRetry(ctx, r, cfg)
	})
}

// executePostAction handles the common setup for OnRunEnd and HandleRetry.
// Returns true if the MLflow sync succeeded.
func (d *Dispatcher) executePostAction(
	ctx context.Context,
	run *apiserverPlugins.PersistedRun,
	hookName string,
	invoke func(*Handler, *apiserverPlugins.PersistedRun, *commonmlflow.PluginConfig),
) bool {
	resolvedCfg, err := ResolveMLflowRequestConfig(ctx, d.kubeClients, run.Namespace)
	if err != nil {
		glog.Warningf("MLflow %s: failed to resolve config for namespace %q: %v", hookName, run.Namespace, err)
	}

	handler := NewHandler(nil, run.Namespace)

	var config *commonmlflow.PluginConfig
	if resolvedCfg != nil {
		config = resolvedCfg.Config
	}
	invoke(handler, run, config)

	mlflowSyncOK := false
	if po := run.PluginsOutput[PluginName]; po != nil {
		mlflowSyncOK = po.State == apiv2beta1.PluginState_PLUGIN_SUCCEEDED
	}

	if err := PersistPluginsOutput(run, d.runOutputStore); err != nil {
		glog.Warningf("MLflow %s: failed to persist plugin output for run %q: %v", hookName, run.RunID, err)
		return false
	}
	return mlflowSyncOK
}
