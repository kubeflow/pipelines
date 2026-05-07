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

// Package mlflow implements MLflow API server plugin handlers.
package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
)

var _ apiserverPlugins.RunPluginHandler = (*Handler)(nil)

// Handler implements PluginHandler for the MLflow integration.
type Handler struct{}

// NewMLflowRunHandler creates a new MLflow plugin handler.
func NewMLflowRunHandler() *Handler {
	return &Handler{}
}

// Name returns the name of the MLflow plugin handler.
func (h *Handler) Name() string {
	return "mlflow"
}

// GetGlobalPluginConfig re-reads the global plugin configuration from Viper so
// that runtime config changes (e.g. admin removing the MLflow config) are
// reflected without restarting the API server.
func (h *Handler) GetGlobalPluginConfig() (*apiserverPlugins.PluginConfig, error) {
	cfg, ok, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &cfg, nil
}

// OnBeforeRunCreation creates the MLflow experiment and parent run, tags it
// with KFP metadata, and returns runtime env vars for the driver and launcher.
func (h *Handler) OnBeforeRunCreation(ctx context.Context, run *apiserverPlugins.PendingRun, runCfg *apiserverPlugins.PluginConfig) (*apiv2beta1.PluginOutput, map[string]string, error) {
	if h == nil || run == nil || runCfg == nil {
		return nil, nil, nil
	}
	mlflowPluginInput, err := ResolveMLflowPluginInput(run.PluginsInput)
	if err != nil {
		return nil, nil, fmt.Errorf("MLflow run cancelled due to error retrieving run-level plugin input: %s", err)
	}
	if mlflowPluginInput == nil || mlflowPluginInput.Disabled {
		return nil, nil, nil
	}

	endpoint := runCfg.Endpoint

	settings := ResolvePluginSettings(runCfg.Settings)

	experimentID, experimentName := SelectMLflowExperiment(mlflowPluginInput, settings)
	if experimentID != "" {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_id=%q (create-by-name skipped)", experimentID)
	} else {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_name=%q", experimentName)
	}

	resolvedCfg, err := ResolveMLflowPluginConfig(runCfg, settings)
	if err != nil {
		message := "MLflow config resolution failed; run creation will continue: " + err.Error()
		glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (%s)", run.RunID, message)
		return FailedPluginOutput(experimentID, experimentName, "", "", "", message), nil, err
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(run.Namespace, resolvedCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, fmt.Sprintf("failed to build MLflow request context: %v", err)), nil, err
	}

	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		settings.ExperimentDescription,
	)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, err.Error()), nil, err
	}

	tags := BuildKFPTags(run, settings.KFPBaseURL, settings.KFPRunURLPathTemplate)
	parentRunID, err := mlflowRequestCtx.Client.CreateRun(ctx, mlflowExperiment.ID, run.DisplayName, tags)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", endpoint, err.Error()), nil, err
	}

	insecureSkipVerify := false
	if resolvedCfg.TLS != nil {
		insecureSkipVerify = resolvedCfg.TLS.InsecureSkipVerify
	}
	workspace := ""
	if settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled {
		workspace = run.Namespace
	}
	// TLS.CABundlePath is intentionally omitted: driver/launcher pods do not
	// have access to the API server's CA bundle file. CA trust for those pods
	// is configured via cluster-wide trusted CA injection or volume mounts
	// managed outside the plugin config. Only InsecureSkipVerify is carried
	// over because it is a boolean flag independent of filesystem context.
	mlflowRuntimeConfig := commonmlflow.MLflowRuntimeConfig{
		Endpoint:           mlflowRequestCtx.BaseURL.String(),
		Workspace:          workspace,
		WorkspacesEnabled:  settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:        parentRunID,
		ExperimentID:       mlflowExperiment.ID,
		AuthType:           commonmlflow.AuthTypeKubernetes,
		Timeout:            resolvedCfg.Timeout,
		InsecureSkipVerify: insecureSkipVerify,
		InjectUserEnvVars:  settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", endpoint, fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), nil, err
	}

	runStartEnv := map[string]string{
		commonmlflow.EnvMLflowConfig: string(mlflowConfigJSON),
	}

	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID, settings)
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL, endpoint), runStartEnv, nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *apiserverPlugins.PluginConfig) error {
	if h == nil || run == nil || runCfg == nil {
		return nil
	}
	resolvedSettings := ResolvePluginSettings(runCfg.Settings)
	resolvedMLflowCfg, err := ResolveMLflowPluginConfig(runCfg, resolvedSettings)
	if err != nil {
		return err
	}
	return h.syncOnRunTerminal(ctx, run, resolvedMLflowCfg, run.Namespace)
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *commonmlflow.MLflowPluginConfig, namespace string) error {
	endTimeMs := int64(0)
	endTimeRef := (*int64)(nil)
	if run.FinishedAt != nil {
		endTimeMs = run.FinishedAt.UnixMilli()
		endTimeRef = &endTimeMs
	}
	terminalStatus := ToMLflowTerminalStatus(run.State)
	h.syncMLflowRuns(ctx, run, runCfg, RunSyncModeTerminal, terminalStatus, endTimeRef, "terminal", namespace)
	return nil
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *apiserverPlugins.PluginConfig) {
	if h == nil || run == nil || runCfg == nil {
		return
	}

	resolvedSettings := ResolvePluginSettings(runCfg.Settings)
	resolvedMLflowCfg, err := ResolveMLflowPluginConfig(runCfg, resolvedSettings)
	if err != nil {
		glog.Errorf("failed to resolve MLflow plugin config: %v", err)
		return
	}

	h.syncMLflowRuns(ctx, run, resolvedMLflowCfg, RunSyncModeRetry, "", nil, "retry", run.Namespace)
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, config *commonmlflow.MLflowPluginConfig, mode RunSyncMode, terminalStatus string, endTimeRef *int64, label string, namespace string) {
	pluginOutput := run.PluginsOutput[PluginName]
	if pluginOutput == nil {
		return
	}

	parentRunID := GetParentRunID(pluginOutput)
	experimentID := GetStringEntry(pluginOutput, EntryExperimentID)
	if parentRunID == "" {
		msg := fmt.Sprintf("MLflow %s sync skipped: missing parent root_run_id in plugins_output.mlflow", label)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}

	if config == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: config unavailable", label)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}

	// Use the endpoint stored at run-start time so that in-flight runs
	// always talk to the MLflow server where their parent run was created,
	// even if the admin changes the endpoint while the run is in progress.
	storedEndpoint := GetStringEntry(pluginOutput, EntryEndpoint)
	if storedEndpoint != "" {
		config.Endpoint = storedEndpoint
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(namespace, config)
	if err != nil {
		msg := fmt.Sprintf("MLflow %s sync failed: %v", label, err)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}

	syncErrors := SyncParentAndNestedRuns(ctx, mlflowRequestCtx, parentRunID, experimentID, mode, terminalStatus, endTimeRef)
	if len(syncErrors) > 0 {
		msg := strings.Join(syncErrors, "; ")
		glog.Warningf("MLflow %s sync encountered errors for run %s: %s", label, run.RunID, msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
	} else {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
	}
}
