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
	corev1 "k8s.io/api/core/v1"
)

var _ apiserverPlugins.RunPluginHandler = (*Handler)(nil)

// Handler implements PluginHandler for the MLflow integration.
type Handler struct {
	input     *MLflowPluginInput
	namespace string

	// RunStartEnvVars is populated by OnBeforeRunCreation with runtime env vars
	// for the driver and launcher.
	RunStartEnvVars []corev1.EnvVar
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
	resolvedCfg := resolveHandlerConfig(config)
	if resolvedCfg == nil || resolvedCfg.Config == nil {
		return nil, nil
	}
	pluginConfig := resolvedCfg.Config

	experimentID, experimentName := SelectMLflowExperiment(h.input, pluginConfig.Settings)

	settings := pluginConfig.Settings
	if settings == nil {
		err := fmt.Errorf("resolved MLflow settings are missing")
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, resolvedCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", fmt.Sprintf("failed to build MLflow request context: %v", err)), err
	}

	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		settings.ExperimentDescription,
	)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
	}

	tags := BuildKFPTags(run, settings.KFPBaseURL, settings.KFPRunURLPathTemplate)
	parentRunID, err := mlflowRequestCtx.Client.CreateRun(ctx, mlflowExperiment.ID, run.DisplayName, tags)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", err.Error()), err
	}

	insecureSkipVerify := false
	if pluginConfig.TLS != nil {
		insecureSkipVerify = pluginConfig.TLS.InsecureSkipVerify
	}
	workspace := ""
	if settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled {
		workspace = h.namespace
	}
	// TLS.CABundlePath is intentionally omitted: driver/launcher pods do not
	// have access to the API server's CA bundle file. CA trust for those pods
	// is configured via cluster-wide trusted CA injection or volume mounts
	// managed outside the plugin config. Only InsecureSkipVerify is carried
	// over because it is a boolean flag independent of filesystem context.
	mlflowRuntimeConfig := commonmlflow.MLflowRuntimeConfig{
		Endpoint:            mlflowRequestCtx.BaseURL.String(),
		Workspace:           workspace,
		WorkspacesEnabled:   settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:         parentRunID,
		ExperimentID:        mlflowExperiment.ID,
		AuthType:            settings.AuthType,
		CredentialSecretRef: runtimeCredentialSecretRef(settings),
		Timeout:             pluginConfig.Timeout,
		InsecureSkipVerify:  insecureSkipVerify,
		InjectUserEnvVars:   settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), err
	}

	h.RunStartEnvVars = []corev1.EnvVar{{
		Name:  commonmlflow.EnvMLflowConfig,
		Value: string(mlflowConfigJSON),
	}}
	credentialEnvVars, err := commonmlflow.BuildCredentialEnvVars(settings.CredentialSecretRef, settings.AuthType)
	if err != nil {
		return FailedPluginOutput(
			mlflowExperiment.ID,
			mlflowExperiment.Name,
			parentRunID,
			"",
			fmt.Sprintf("failed to build MLflow credential env vars: %v", err),
		), err
	}
	h.RunStartEnvVars = append(h.RunStartEnvVars, credentialEnvVars...)

	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID, settings)
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL), nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, config interface{}) error {
	if h == nil || run == nil {
		return nil
	}
	return h.syncOnRunTerminal(ctx, run, resolveHandlerConfig(config))
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig) error {
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
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig) {
	h.syncMLflowRuns(ctx, run, config, RunSyncModeRetry, "", nil, "retry")
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig, mode RunSyncMode, terminalStatus string, endTimeRef *int64, label string) {
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

	localConfig := cloneResolvedConfig(config)
	if localConfig == nil || localConfig.Config == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: config unavailable", label)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}
	if localConfig.Config.Settings == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: resolved MLflow settings are missing", label)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, localConfig)
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

func resolveHandlerConfig(config interface{}) *ResolvedConfig {
	typedConfig, _ := config.(*ResolvedConfig)
	return typedConfig
}

func runtimeCredentialSecretRef(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.CredentialSecretRef {
	if settings == nil || settings.CredentialSecretRef == nil {
		return nil
	}
	switch settings.AuthType {
	case commonmlflow.AuthTypeBearer, commonmlflow.AuthTypeBasicAuth:
		credentialSecretRef := *settings.CredentialSecretRef
		return &credentialSecretRef
	default:
		return nil
	}
}

func cloneResolvedConfig(config *ResolvedConfig) *ResolvedConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	if config.Config != nil {
		configCopy := *config.Config
		cloned.Config = &configCopy
	}
	if config.Config != nil && config.Config.Settings != nil {
		settingsCopy := *config.Config.Settings
		if settingsCopy.WorkspacesEnabled != nil {
			workspacesEnabled := *settingsCopy.WorkspacesEnabled
			settingsCopy.WorkspacesEnabled = &workspacesEnabled
		}
		cloned.Config.Settings = &settingsCopy
	}
	return &cloned
}
