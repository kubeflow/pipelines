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
	"fmt"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
)

// PluginHandler encapsulates plugin lifecycle hooks used by core paths.
type PluginHandler interface {
	OnRunStart(ctx context.Context, run *model.Run, namespace string) (*apiv2beta1.PluginOutput, error)
	OnRunEnd(ctx context.Context, run *model.Run, namespace string) error
	OnTaskStart(ctx context.Context, taskInfo interface{}, namespace string) (interface{}, error)
	OnTaskEnd(ctx context.Context, taskInfo interface{}, metrics map[string]float64, params map[string]string, namespace string) error
}

// HandlerDeps bundles the external dependencies required by Handler.
type HandlerDeps struct {
	KubeClients               KubeClientProvider
	SubjectAccessReviewClient client.SubjectAccessReviewInterface
	IsAuthorized              func(context.Context, *authorizationv1.ResourceAttributes) error
	RunStoreUpdater           RunStoreUpdater
}

// RunStoreUpdater is the minimal interface for persisting run plugin output,
// so the handler does not depend on the full RunStore.
type RunStoreUpdater interface {
	// UpdateRunPluginsOutput updates only the PluginsOutput column for the
	// given run, leaving core run fields untouched.
	UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error
}

// Handler implements PluginHandler for the MLflow integration.
type Handler struct {
	deps  HandlerDeps
	input *PluginInput

	// RunStartEnv is populated by OnRunStart and consumed by the caller to
	// inject environment variables into the Argo Workflow templates.
	RunStartEnv map[string]string
}

// NewHandler creates a new MLflow plugin handler with the given dependencies
// and plugin input.
func NewHandler(deps HandlerDeps, input *PluginInput) *Handler {
	return &Handler{
		deps:  deps,
		input: input,
	}
}

func (h *Handler) clientSet() kubernetes.Interface {
	if h.deps.KubeClients == nil {
		return nil
	}
	return h.deps.KubeClients.GetClientSet()
}

// OnRunStart creates the MLflow experiment and parent run, tags it with KFP
// metadata, and populates RunStartEnv with tracking env vars.
func (h *Handler) OnRunStart(ctx context.Context, run *model.Run, namespace string) (*apiv2beta1.PluginOutput, error) {
	if h == nil || run == nil || h.input == nil || h.input.Disabled {
		return nil, nil
	}
	experimentID, experimentName := SelectExperiment(h.input)
	requestCfg, err := ResolveRequestConfig(ctx, h.deps.KubeClients, namespace)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", fmt.Sprintf("failed to resolve MLflow request configuration: %v", err)), err
	}
	if requestCfg == nil {
		return nil, nil
	}
	if err := AuthorizeExperimentAction(ctx, namespace, h.deps.SubjectAccessReviewClient, h.deps.IsAuthorized); err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
	}
	mlflowRequestCtx, err := BuildRequestContext(ctx, h.clientSet(), namespace, requestCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", fmt.Sprintf("failed to build MLflow request context: %v", err)), err
	}
	if mlflowRequestCtx == nil {
		return nil, nil
	}
	var description *string
	if requestCfg.Settings != nil {
		description = requestCfg.Settings.ExperimentDescription
	}
	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		ResolveExperimentDescription(description),
	)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
	}
	parentRunID, err := CreateParentRun(ctx, mlflowRequestCtx, mlflowExperiment.ID, run.DisplayName)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", err.Error()), err
	}
	authType := DefaultAuthType
	if requestCfg.Settings != nil && requestCfg.Settings.AuthType != "" {
		authType = requestCfg.Settings.AuthType
	}
	h.RunStartEnv = map[string]string{
		EnvTrackingURI: mlflowRequestCtx.BaseURL.String(),
		EnvWorkspace:   namespace,
		EnvParentRunID: parentRunID,
		EnvAuthType:    authType,
	}
	if err := TagRunWithKFPMetadata(ctx, mlflowRequestCtx, parentRunID, run); err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", err.Error()), err
	}
	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID)
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL), nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
// The namespace is the resolved Kubernetes namespace for the run.
func (h *Handler) OnRunEnd(ctx context.Context, run *model.Run, namespace string) error {
	if h == nil || run == nil {
		return nil
	}
	return h.syncOnRunTerminal(ctx, run, namespace)
}

// OnTaskStart is a placeholder for future task-level MLflow integration.
func (h *Handler) OnTaskStart(ctx context.Context, taskInfo interface{}, namespace string) (interface{}, error) {
	return nil, nil
}

// OnTaskEnd is a placeholder for future task-level MLflow integration.
func (h *Handler) OnTaskEnd(ctx context.Context, taskInfo interface{}, metrics map[string]float64, params map[string]string, namespace string) error {
	return nil
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
// The namespace parameter is the resolved Kubernetes namespace (the caller is
// responsible for falling back to the pod namespace in standalone mode).
//
// This method only persists PluginsOutput via UpdateRunPluginsOutput, avoiding
// redundant writes of core run fields (State, Conditions, etc.) that have
// already been committed by ReportWorkflowResource.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *model.Run, namespace string) error {
	pluginOutput, err := GetRunPluginOutput(run, PluginName)
	if err != nil || pluginOutput == nil {
		return err
	}

	parentRunID := GetParentRunID(pluginOutput)
	experimentID := GetStringEntry(pluginOutput, EntryExperimentID)
	if parentRunID == "" {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, "MLflow terminal sync skipped: missing parent root_run_id in plugins_output.mlflow")
		if upsertErr := UpsertRunPluginOutput(run, PluginName, pluginOutput); upsertErr != nil {
			return upsertErr
		}
		return h.deps.RunStoreUpdater.UpdateRunPluginsOutput(run.UUID, run.PluginsOutputString)
	}

	requestCfg, cfgErr := ResolveRequestConfig(ctx, h.deps.KubeClients, namespace)
	if cfgErr != nil || requestCfg == nil {
		message := "MLflow terminal sync failed: MLflow config unavailable"
		if cfgErr != nil {
			message = fmt.Sprintf("MLflow terminal sync failed: %v", cfgErr)
		}
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, message)
		if upsertErr := UpsertRunPluginOutput(run, PluginName, pluginOutput); upsertErr != nil {
			return upsertErr
		}
		return h.deps.RunStoreUpdater.UpdateRunPluginsOutput(run.UUID, run.PluginsOutputString)
	}
	mlflowRequestCtx, ctxErr := BuildRequestContext(ctx, h.clientSet(), namespace, requestCfg)
	if ctxErr != nil || mlflowRequestCtx == nil || mlflowRequestCtx.Client == nil {
		message := "MLflow terminal sync failed: unable to build MLflow request context"
		if ctxErr != nil {
			message = fmt.Sprintf("MLflow terminal sync failed: %v", ctxErr)
		}
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, message)
		if upsertErr := UpsertRunPluginOutput(run, PluginName, pluginOutput); upsertErr != nil {
			return upsertErr
		}
		return h.deps.RunStoreUpdater.UpdateRunPluginsOutput(run.UUID, run.PluginsOutputString)
	}

	endTimeMs := int64(0)
	endTimeRef := (*int64)(nil)
	if run.FinishedAtInSec > 0 {
		endTimeMs = run.FinishedAtInSec * 1000
		endTimeRef = &endTimeMs
	}
	terminalStatus := ToMLflowTerminalStatus(string(run.State.ToV2()))
	syncErrors := SyncParentAndNestedRuns(
		ctx,
		mlflowRequestCtx,
		parentRunID,
		experimentID,
		RunSyncModeTerminal,
		terminalStatus,
		endTimeRef,
	)

	if len(syncErrors) > 0 {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, strings.Join(syncErrors, "; "))
	} else {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
	}
	if err := UpsertRunPluginOutput(run, PluginName, pluginOutput); err != nil {
		return err
	}
	return h.deps.RunStoreUpdater.UpdateRunPluginsOutput(run.UUID, run.PluginsOutputString)
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
//
// Like syncOnRunTerminal, this method is self-contained: it persists
// PluginsOutput via UpdateRunPluginsOutput on every exit path so that
// callers do not need to include PluginsOutput in their own UpdateRun calls.
func (h *Handler) HandleRetry(ctx context.Context, run *model.Run, namespace string) {
	pluginOutput, err := GetRunPluginOutput(run, PluginName)
	if err != nil || pluginOutput == nil {
		if err != nil {
			glog.Warningf("MLflow HandleRetry: failed to read plugin output for run %q: %v", run.UUID, err)
		}
		return
	}
	parentRunID := GetParentRunID(pluginOutput)
	experimentID := GetStringEntry(pluginOutput, EntryExperimentID)
	if parentRunID == "" {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, "MLflow retry sync skipped: missing parent root_run_id in plugins_output.mlflow")
		h.persistRetryPluginOutput(run, pluginOutput)
		return
	}
	requestCfg, cfgErr := ResolveRequestConfig(ctx, h.deps.KubeClients, namespace)
	if cfgErr != nil || requestCfg == nil {
		message := "MLflow retry sync failed: MLflow config unavailable"
		if cfgErr != nil {
			message = fmt.Sprintf("MLflow retry sync failed: %v", cfgErr)
		}
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, message)
		h.persistRetryPluginOutput(run, pluginOutput)
		return
	}
	mlflowRequestCtx, ctxErr := BuildRequestContext(ctx, h.clientSet(), namespace, requestCfg)
	if ctxErr != nil || mlflowRequestCtx == nil || mlflowRequestCtx.Client == nil {
		message := "MLflow retry sync failed: unable to build MLflow request context"
		if ctxErr != nil {
			message = fmt.Sprintf("MLflow retry sync failed: %v", ctxErr)
		}
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, message)
		h.persistRetryPluginOutput(run, pluginOutput)
		return
	}
	syncErrors := SyncParentAndNestedRuns(
		ctx,
		mlflowRequestCtx,
		parentRunID,
		experimentID,
		RunSyncModeRetry,
		"",
		nil,
	)
	if len(syncErrors) > 0 {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, strings.Join(syncErrors, "; "))
	} else {
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
	}
	h.persistRetryPluginOutput(run, pluginOutput)
}

// persistRetryPluginOutput is a helper that upserts plugin output and persists
// it to the store, logging warnings on failure. HandleRetry is fire-and-forget
// so errors are not returned to the caller.
func (h *Handler) persistRetryPluginOutput(run *model.Run, pluginOutput *apiv2beta1.PluginOutput) {
	if upsertErr := UpsertRunPluginOutput(run, PluginName, pluginOutput); upsertErr != nil {
		glog.Warningf("MLflow HandleRetry: failed to marshal plugin output for run %q: %v", run.UUID, upsertErr)
		return
	}
	if storeErr := h.deps.RunStoreUpdater.UpdateRunPluginsOutput(run.UUID, run.PluginsOutputString); storeErr != nil {
		glog.Warningf("MLflow HandleRetry: failed to persist plugin output for run %q: %v", run.UUID, storeErr)
	}
}
