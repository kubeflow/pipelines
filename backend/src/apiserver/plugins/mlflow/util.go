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
	"net/url"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	DefaultExperimentDescription = "Created by Kubeflow Pipelines"
	TagKFPRunID                  = "kfp.pipeline_run_id"
	TagKFPRunURL                 = "kfp.pipeline_run_url"
	TagKFPPipelineID             = "kfp.pipeline_id"
	TagKFPPipelineVersionID      = "kfp.pipeline_version_id"
	EntryExperimentName          = "experiment_name"
	EntryExperimentID            = "experiment_id"
	EntryRootRunID               = "root_run_id"
	EntryRunURL                  = "run_url"
)

type Experiment struct {
	ID   string
	Name string
}

// EnsureExperimentExists looks up the MLflow experiment by ID or name, and creates it
// if it does not already exist. ExperimentID will take precedence over ExperimentName.
func EnsureExperimentExists(ctx context.Context, requestCtx *commonmlflow.RequestContext, experimentID, experimentName string, description *string) (*Experiment, error) {
	if requestCtx == nil || requestCtx.Client == nil {
		return nil, util.NewInvalidInputError("MLflow request context is required")
	}
	if experimentID != "" {
		exp, err := requestCtx.Client.GetExperiment(ctx, experimentID)
		if err != nil {
			return nil, fmt.Errorf("experiment ID %q not found in MLflow: %w", experimentID, err)
		}
		return &Experiment{ID: exp.ID, Name: exp.Name}, nil
	}
	existing, err := requestCtx.Client.GetExperimentByName(ctx, experimentName)
	if err == nil {
		return &Experiment{ID: existing.ID, Name: existing.Name}, nil
	}
	if !commonmlflow.IsNotFoundError(err) {
		return nil, err
	}
	return CreateExperiment(ctx, requestCtx, experimentName, description)
}

// CreateExperiment creates an MLflow experiment and handles the race condition
// where another request may have created the same experiment concurrently.
func CreateExperiment(ctx context.Context, requestCtx *commonmlflow.RequestContext, experimentName string, description *string) (*Experiment, error) {
	createdID, createErr := requestCtx.Client.CreateExperiment(ctx, experimentName, description)
	if createErr == nil {
		return &Experiment{ID: createdID, Name: experimentName}, nil
	}
	if commonmlflow.IsAlreadyExistsError(createErr) {
		// Race-safe fallback: another request created it between get-by-name and create.
		existing, err := requestCtx.Client.GetExperimentByName(ctx, experimentName)
		if err != nil {
			return nil, err
		}
		return &Experiment{ID: existing.ID, Name: existing.Name}, nil
	}
	return nil, createErr
}

// BuildKFPTags builds MLflow tags containing KFP metadata for a pipeline run.
func BuildKFPTags(run *apiserverPlugins.PendingRun, kfpBaseURL, kfpRunURLPathTemplate string) []commonmlflow.Tag {
	if run == nil {
		return nil
	}
	tags := []commonmlflow.Tag{
		{Key: TagKFPRunID, Value: run.RunID},
		{Key: TagKFPRunURL, Value: apiserverPlugins.BuildKFPRunURL(run.RunID, run.Namespace, kfpBaseURL, kfpRunURLPathTemplate)},
	}
	if run.PipelineID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: run.PipelineID})
	}
	if run.PipelineVersionID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: run.PipelineVersionID})
	}
	// Idempotency key for CreateRun retries: the stable KFP run ID lets the
	// client find and reuse a parent run created by a timed-out attempt instead
	// of creating a duplicate. Appended last so it does not shift the other tag
	// positions.
	tags = append(tags, commonmlflow.Tag{Key: commonmlflow.IdempotencyTagKey, Value: run.RunID})
	return tags
}

// mlflowTrackingUIMountBase resolves the UI link prefix for MLflow Tracking UI links.
func mlflowTrackingUIMountBase(requestCtx *commonmlflow.RequestContext, settings *commonmlflow.MLflowPluginSettings) string {
	if settings != nil {
		if b := strings.TrimSpace(settings.MLflowBaseURL); b != "" {
			return strings.TrimRight(b, "/")
		}
	}
	if requestCtx != nil && requestCtx.BaseURL != nil {
		return strings.TrimRight(requestCtx.BaseURL.String(), "/")
	}
	return ""
}

func normalizeMlflowUIPathPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

// BuildRunURL returns the MLflow Tracking UI URL for a run.
func BuildRunURL(requestCtx *commonmlflow.RequestContext, experimentID, runID string, settings *commonmlflow.MLflowPluginSettings) string {
	if experimentID == "" || runID == "" {
		glog.V(4).Infof(
			"BuildRunURL returned empty URL due to missing input(s): experimentID_empty=%t runID_empty=%t",
			experimentID == "",
			runID == "",
		)
		return ""
	}
	trackingUIBase := mlflowTrackingUIMountBase(requestCtx, settings)
	if trackingUIBase == "" {
		glog.V(4).Infof(
			"BuildRunURL returned empty URL: no mlflowBaseURL and requestCtx.BaseURL is unavailable",
		)
		return ""
	}
	uiPathPrefix := ""
	if settings != nil {
		uiPathPrefix = normalizeMlflowUIPathPrefix(settings.MLflowUIPathPrefix)
	}

	trackingMlflowRunPath := fmt.Sprintf(
		"/experiments/%s/runs/%s",
		url.PathEscape(experimentID),
		url.PathEscape(runID),
	)
	if requestCtx != nil && requestCtx.WorkspacesEnabled && requestCtx.Workspace != "" {
		trackingMlflowRunPath = fmt.Sprintf("%s?workspace=%s", trackingMlflowRunPath, url.QueryEscape(strings.ToLower(requestCtx.Workspace)))
	}
	return trackingUIBase + uiPathPrefix + "/#" + trackingMlflowRunPath
}

func SuccessfulPluginOutput(experimentID, experimentName, runID, runURL string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
}

func FailedPluginOutput(experimentID, experimentName, runID, runURL, stateMessage string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, apiv2beta1.PluginState_PLUGIN_FAILED, stateMessage)
}

// maxSearchPages caps SearchRuns pagination in reopenNestedRuns (retry path).
const maxSearchPages = 10

// maxNestingDepth caps recursive nested run traversal.
const maxNestingDepth = 6

func SyncParentAndNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode apiserverPlugins.RunSyncMode, terminalStatus string, endTimeMs *int64) []string {
	if requestCtx == nil || requestCtx.Client == nil {
		return []string{"MLflow request context is required"}
	}
	if parentRunID == "" {
		return []string{"MLflow parent run_id is required"}
	}
	targetStatus := terminalStatus
	parentAction := "update parent run status"
	switch mode {
	case apiserverPlugins.RunSyncModeRetry:
		targetStatus = "RUNNING"
		parentAction = "reopen parent run"
	case apiserverPlugins.RunSyncModeTerminal:
		// keep caller-provided terminal status
	default:
		return []string{fmt.Sprintf("unsupported MLflow run sync mode %q", mode)}
	}
	var syncErrors []string
	if err := requestCtx.Client.UpdateRun(ctx, parentRunID, targetStatus, endTimeMs); err != nil {
		syncErrors = append(syncErrors, fmt.Sprintf("failed to %s: %v", parentAction, err))
	}
	if experimentID == "" {
		return syncErrors
	}
	// Recursively update all nested runs
	nestedErrors := syncNestedRuns(ctx, requestCtx, parentRunID, experimentID, mode, targetStatus, endTimeMs, 0)
	syncErrors = append(syncErrors, nestedErrors...)
	return syncErrors
}

// syncNestedRuns enforces the recursion depth budget, then dispatches to
// closeNestedRuns (terminal) or reopenNestedRuns (retry).
func syncNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode apiserverPlugins.RunSyncMode, targetStatus string, endTimeMs *int64, depth int) []string {
	if depth >= maxNestingDepth {
		return []string{fmt.Sprintf("max nesting depth (%d) reached when syncing children of run %s", maxNestingDepth, parentRunID)}
	}
	if mode == apiserverPlugins.RunSyncModeRetry {
		return reopenNestedRuns(ctx, requestCtx, parentRunID, experimentID, targetStatus, endTimeMs, depth)
	}
	return closeNestedRuns(ctx, requestCtx, parentRunID, experimentID, targetStatus, endTimeMs, depth)
}

// closeNestedRuns closes open direct children of parentRunID and their descendants, bottom-up.
func closeNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID, targetStatus string, endTimeMs *int64, depth int) []string {
	var syncErrors []string
	childCloseErrors := commonmlflow.CloseOpenChildRuns(ctx, requestCtx.Client, experimentID, parentRunID, targetStatus, endTimeMs, func(childRunID string) {
		syncErrors = append(syncErrors, syncNestedRuns(ctx, requestCtx, childRunID, experimentID, apiserverPlugins.RunSyncModeTerminal, targetStatus, endTimeMs, depth+1)...)
	})
	syncErrors = append(syncErrors, childCloseErrors...)
	return syncErrors
}

// reopenNestedRuns reopens FAILED/KILLED nested runs of parentRunID on KFP
// run retry.
func reopenNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID, targetStatus string, endTimeMs *int64, depth int) []string {
	var syncErrors []string
	filter := fmt.Sprintf(`tags.%q = '%s'`, commonmlflow.ParentRunTagKey, parentRunID)
	pageToken := ""
	for page := 0; page < maxSearchPages; page++ {
		searchResp, err := requestCtx.Client.SearchRuns(ctx, []string{experimentID}, filter, 1000, pageToken)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("failed to search nested runs of %s: %v", parentRunID, err))
			break
		}
		for _, runPayload := range searchResp.Runs {
			mlflowRun := &searchRunPayload{}
			if err := json.Unmarshal(runPayload, mlflowRun); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("failed to decode nested run payload: %v", err))
				continue
			}
			nestedRunID := mlflowRun.Info.RunID
			if nestedRunID == "" {
				nestedRunID = mlflowRun.Info.RunUUID
			}
			if nestedRunID == "" || nestedRunID == parentRunID || !shouldReopenNestedRun(mlflowRun.Info.Status) {
				continue
			}
			childErrors := syncNestedRuns(ctx, requestCtx, nestedRunID, experimentID, apiserverPlugins.RunSyncModeRetry, targetStatus, endTimeMs, depth+1)
			syncErrors = append(syncErrors, childErrors...)
			if err := requestCtx.Client.UpdateRun(ctx, nestedRunID, targetStatus, endTimeMs); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("failed to reopen nested run %s: %v", nestedRunID, err))
			}
		}
		if searchResp.NextPageToken == "" {
			break
		}
		if page == maxSearchPages-1 {
			syncErrors = append(syncErrors, fmt.Sprintf(
				"pagination limit (%d pages) reached while reopening nested runs of %s; some runs may have been skipped",
				maxSearchPages, parentRunID))
			break
		}
		pageToken = searchResp.NextPageToken
	}
	return syncErrors
}

func buildPluginOutput(experimentID, experimentName, runID, runURL string, state apiv2beta1.PluginState, stateMessage string) *apiv2beta1.PluginOutput {
	entries := map[string]*apiv2beta1.MetadataValue{}
	if experimentName != "" {
		entries[EntryExperimentName] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentName)}
	}
	if experimentID != "" {
		entries[EntryExperimentID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentID)}
	}
	if runID != "" {
		entries[EntryRootRunID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(runID)}
	}
	if runURL != "" {
		entries[EntryRunURL] = &apiv2beta1.MetadataValue{
			Value:      structpb.NewStringValue(runURL),
			RenderType: apiv2beta1.MetadataValue_URL.Enum(),
		}
	}
	return &apiv2beta1.PluginOutput{
		Entries:      entries,
		State:        state,
		StateMessage: stateMessage,
	}
}

// shouldReopenNestedRun reports whether a nested run is eligible to be
// reopened on retry, i.e. it previously ended in a failed/killed state.
func shouldReopenNestedRun(status string) bool {
	upperStatus := strings.ToUpper(status)
	return upperStatus == "FAILED" || upperStatus == "KILLED"
}

type searchRunPayload struct {
	Info struct {
		RunID   string `json:"run_id"`
		RunUUID string `json:"run_uuid"`
		Status  string `json:"status"`
	} `json:"info"`
}
