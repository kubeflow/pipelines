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
type Handler struct {
	input     *MLflowPluginInput
	namespace string

	// RunStartEnv is populated by OnBeforeRunCreation with the single
	// KFP_MLFLOW_CONFIG env var for the driver and launcher.
	RunStartEnv map[string]string
}

// NewHandler creates a new MLflow plugin handler.
func NewHandler(input *MLflowPluginInput, namespace string) *Handler {
	return &Handler{
		input:     input,
		namespace: namespace,
	}
}

// OnBeforeRunCreation creates the MLflow experiment and parent run, tags it
// with KFP metadata, and populates RunStartEnv with tracking env vars.
func (h *Handler) OnBeforeRunCreation(ctx context.Context, run *apiserverPlugins.PendingRun, config interface{}) (*apiv2beta1.PluginOutput, error) {
	if h == nil || run == nil || h.input == nil || h.input.Disabled {
		return nil, nil
	}
	pluginConfig, ok := config.(*commonmlflow.PluginConfig)
	if !ok || pluginConfig == nil {
		return nil, nil
	}

	endpoint := pluginConfig.Endpoint

	settings := ApplySettingsDefaults(pluginConfig.Settings)

	experimentID, experimentName := SelectMLflowExperiment(h.input, settings)

	resolvedCfg := &ResolvedConfig{Config: pluginConfig, Settings: settings}
	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, resolvedCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, fmt.Sprintf("failed to build MLflow request context: %v", err)), err
	}

	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		settings.ExperimentDescription,
	)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, err.Error()), err
	}

	tags := BuildKFPTags(run, settings.KFPBaseURL)
	parentRunID, err := mlflowRequestCtx.Client.CreateRun(ctx, mlflowExperiment.ID, run.DisplayName, tags)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", endpoint, err.Error()), err
	}

	insecureSkipVerify := false
	if pluginConfig.TLS != nil {
		insecureSkipVerify = pluginConfig.TLS.InsecureSkipVerify
	}
	workspace := ""
	if settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled {
		workspace = h.namespace
	}
	mlflowRuntimeConfig := commonmlflow.MLflowRuntimeConfig{
		Endpoint:           mlflowRequestCtx.BaseURL.String(),
		Workspace:          workspace,
		WorkspacesEnabled:  settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:        parentRunID,
		ExperimentID:       mlflowExperiment.ID,
		AuthType:           commonmlflow.AuthTypeKubernetes,
		Timeout:            pluginConfig.Timeout,
		InsecureSkipVerify: insecureSkipVerify,
		InjectUserEnvVars:  settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", endpoint, fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), err
	}

	h.RunStartEnv = map[string]string{
		commonmlflow.EnvMLflowConfig: string(mlflowConfigJSON),
	}

	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID)
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL, endpoint), nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, config interface{}) error {
	if h == nil || run == nil {
		return nil
	}
	pluginConfig, _ := config.(*commonmlflow.PluginConfig)
	return h.syncOnRunTerminal(ctx, run, pluginConfig)
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, config *commonmlflow.PluginConfig) error {
	endTimeMs := int64(0)
	endTimeRef := (*int64)(nil)
	if run.FinishedAt != nil {
		endTimeMs = run.FinishedAt.UnixMilli()
		endTimeRef = &endTimeMs
	}
	terminalStatus := ToMLflowTerminalStatus(run.State)
	h.syncMLflowRuns(ctx, run, config, RunSyncModeTerminal, terminalStatus, endTimeRef, "terminal")
	return nil
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, config *commonmlflow.PluginConfig) {
	h.syncMLflowRuns(ctx, run, config, RunSyncModeRetry, "", nil, "retry")
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, config *commonmlflow.PluginConfig, mode RunSyncMode, terminalStatus string, endTimeRef *int64, label string) {
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

	settings := ApplySettingsDefaults(config.Settings)

	resolvedCfg := &ResolvedConfig{Config: config, Settings: settings}
	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, resolvedCfg)
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
