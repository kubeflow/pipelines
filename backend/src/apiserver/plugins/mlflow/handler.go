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
	"time"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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

// ResolveRunPluginInput parses the plugins_input.mlflow JSON from a run model
// and validates it against the MLflowPluginInput schema.
func (h *Handler) ResolveRunPluginInput(pluginsInputString *string) (interface{}, bool, error) {
	if resolvedInput, err := ResolveMLflowPluginInput(pluginsInputString); err != nil {
		return nil, false, util.NewInvalidInputError("failed to resolve MLflow plugin input: %v", err)
	} else {
		return resolvedInput, !resolvedInput.Disabled, nil
	}
}

// ResolveRunPluginConfig resolves the MLflow plugin config for the given run.
func (h *Handler) ResolveRunPluginConfig(ctx context.Context, clientSet kubernetes.Interface, launcherNamespaceCfg string, namespace string) (interface{}, error) {
	var runCfg interface{}
	var err error
	if runCfg, err = ResolveMLflowRequestConfig(ctx, clientSet, launcherNamespaceCfg, namespace); err != nil {
		return nil, fmt.Errorf("MLflow run canceled due to error resolving run-level plugin config")
	}
	return runCfg, nil
}

// GetPluginOperationTimeout calculates and returns the operation budget duration for MLflow plugin based on run configuration.
func (h *Handler) GetPluginOperationTimeout(runCfg interface{}) time.Duration {
	defaultTimeout := 30 * time.Second

	cfg, ok := runCfg.(*ResolvedMLflowConfig)
	if ok && cfg != nil && cfg.Config != nil && cfg.Config.Timeout != "" {
		if timeout, err := time.ParseDuration(cfg.Config.Timeout); err == nil && timeout > 0 {
			return mlflowOperationBudget(timeout)
		}
	}
	return mlflowOperationBudget(defaultTimeout)
}

// OnBeforeRunCreation creates the MLflow experiment and parent run, tags it
// with KFP metadata, and returns runtime env vars.
func (h *Handler) OnBeforeRunCreation(ctx context.Context, run *apiserverPlugins.PendingRun, runCfg interface{}, resolvedPluginInput interface{}) (*apiv2beta1.PluginOutput, []corev1.EnvVar, error) {
	if h == nil || run == nil || runCfg == nil {
		return nil, nil, nil
	}
	mlflowPluginInput, ok := resolvedPluginInput.(*MLflowPluginInput)
	if !ok || mlflowPluginInput == nil || mlflowPluginInput.Disabled {
		return nil, nil, nil
	}
	resolvedRunCfg, ok := runCfg.(*ResolvedMLflowConfig)
	if !ok || resolvedRunCfg == nil || resolvedRunCfg.Config == nil {
		return nil, nil, nil
	}

	experimentID, experimentName := SelectMLflowExperiment(mlflowPluginInput, resolvedRunCfg.Config.Settings)
	if experimentID != "" {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_id=%q (create-by-name skipped)", experimentID)
	} else {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_name=%q", experimentName)
	}

	settings := resolvedRunCfg.Config.Settings
	if settings == nil {
		err := fmt.Errorf("resolved MLflow settings are missing")
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), nil, err
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(run.Namespace, resolvedRunCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", fmt.Sprintf("failed to build MLflow request context: %v", err)), nil, err
	}

	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		settings.ExperimentDescription,
	)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), nil, err
	}

	tags := BuildKFPTags(run, settings.KFPBaseURL, settings.KFPRunURLPathTemplate)
	parentRunID, err := mlflowRequestCtx.Client.CreateRun(ctx, mlflowExperiment.ID, run.DisplayName, tags)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", err.Error()), nil, err
	}

	insecureSkipVerify := false
	if resolvedRunCfg.Config.TLS != nil {
		insecureSkipVerify = resolvedRunCfg.Config.TLS.InsecureSkipVerify
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
		Endpoint:            mlflowRequestCtx.BaseURL.String(),
		Workspace:           workspace,
		WorkspacesEnabled:   settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:         parentRunID,
		ExperimentID:        mlflowExperiment.ID,
		AuthType:            settings.AuthType,
		CredentialSecretRef: runtimeCredentialSecretRef(settings),
		Timeout:             resolvedRunCfg.Config.Timeout,
		InsecureSkipVerify:  insecureSkipVerify,
		InjectUserEnvVars:   settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), nil, err
	}

	runStartEnv := []corev1.EnvVar{
		{
			Name:  commonmlflow.EnvMLflowConfig,
			Value: string(mlflowConfigJSON),
		},
	}
	credentialEnvVars, err := commonmlflow.BuildCredentialEnvVars(settings.CredentialSecretRef, settings.AuthType)
	if err != nil {
		return FailedPluginOutput(
			mlflowExperiment.ID,
			mlflowExperiment.Name,
			parentRunID,
			"",
			fmt.Sprintf("failed to build MLflow credential env vars: %v", err),
		), nil, err
	}
	runStartEnv = append(runStartEnv, credentialEnvVars...)

	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID, settings)
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL), runStartEnv, nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
// bool reports whether a failed sync is worth retrying: transient MLflow
// call failures request a retry, while permanent problems (missing parent
// run id, unavailable or invalid config) are recorded in the plugin output
// and must not block run finalization.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg interface{}) (bool, error) {
	if h == nil || run == nil {
		return false, nil
	}
	return h.syncOnRunTerminal(ctx, run, resolveHandlerConfig(runCfg)), nil
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
// It returns true when the sync failed transiently and should be retried.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *ResolvedMLflowConfig) bool {
	endTimeMs := int64(0)
	endTimeRef := (*int64)(nil)
	if run.FinishedAt != nil {
		endTimeMs = run.FinishedAt.UnixMilli()
		endTimeRef = &endTimeMs
	}
	terminalStatus := ToMLflowTerminalStatus(run.State)
	return h.syncMLflowRuns(ctx, run, runCfg, apiserverPlugins.RunSyncModeTerminal, terminalStatus, endTimeRef, "terminal")
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg interface{}) error {
	if h == nil || run == nil {
		return fmt.Errorf("handler and run must be non-nil")
	}

	h.syncMLflowRuns(ctx, run, resolveHandlerConfig(runCfg), apiserverPlugins.RunSyncModeRetry, "", nil, "retry")
	return nil
}

// GetGenericFailedPluginOutput generates a failure plugin output for a given runID and message if pluginInput is valid.
// Returns nil if pluginInput cannot be resolved to MLflowPluginInput.
func (h *Handler) GetGenericFailedPluginOutput(runID string, message string, pluginInput interface{}) *apiv2beta1.PluginOutput {
	resolvedPluginInput, ok := pluginInput.(*MLflowPluginInput)
	if !ok || resolvedPluginInput == nil {
		return nil
	}
	return FailedPluginOutput(resolvedPluginInput.ExperimentID, resolvedPluginInput.ExperimentName, runID, "", message)
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state. The returned bool reports whether the failure is
// transient and worth retrying; permanent failures (missing parent run id, unavailable
// or invalid config, unresolvable credentials) return false so callers do not retry
// a sync that cannot succeed until an operator fixes the configuration.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, cfg *ResolvedMLflowConfig, mode apiserverPlugins.RunSyncMode, terminalStatus string, endTimeRef *int64, label string) bool {
	pluginOutput := run.PluginsOutput[h.Name()]
	if pluginOutput == nil {
		return false
	}

	parentRunID := apiserverPlugins.GetParentRunID(pluginOutput)
	experimentID := apiserverPlugins.GetStringEntry(pluginOutput, EntryExperimentID)
	if parentRunID == "" {
		msg := fmt.Sprintf("MLflow %s sync skipped: missing parent root_run_id in plugins_output.mlflow", label)
		glog.Warning(msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}

	localConfig := cloneResolvedConfig(cfg)
	if localConfig == nil || localConfig.Config == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: config unavailable", label)
		glog.Warning(msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}
	if localConfig.Config.Settings == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: resolved MLflow settings are missing", label)
		glog.Warning(msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(run.Namespace, localConfig)
	if err != nil {
		msg := fmt.Sprintf("MLflow %s sync failed: %v", label, err)
		glog.Warning(msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}

	syncErrors := SyncParentAndNestedRuns(ctx, mlflowRequestCtx, parentRunID, experimentID, mode, terminalStatus, endTimeRef)
	if len(syncErrors) > 0 {
		msg := strings.Join(syncErrors, "; ")
		glog.Warningf("MLflow %s sync encountered errors for run %s: %s", label, run.RunID, msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		// The MLflow calls themselves failed (network, availability, or
		// server-side errors); a later retry can succeed.
		return true
	}
	apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
	return false
}

func resolveHandlerConfig(config interface{}) *ResolvedMLflowConfig {
	typedConfig, _ := config.(*ResolvedMLflowConfig)
	return typedConfig
}

func runtimeCredentialSecretRef(settings *commonmlflow.MLflowPluginSettings) *commonplugins.CredentialSecretRef {
	if settings == nil || settings.CredentialSecretRef == nil {
		return nil
	}
	switch settings.AuthType {
	case commonplugins.AuthTypeBearer, commonplugins.AuthTypeBasicAuth:
		credentialSecretRef := *settings.CredentialSecretRef
		return &credentialSecretRef
	default:
		return nil
	}
}

func cloneResolvedConfig(config *ResolvedMLflowConfig) *ResolvedMLflowConfig {
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
