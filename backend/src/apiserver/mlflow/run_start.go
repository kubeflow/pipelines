package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/encoding/protojson"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type pluginsOutputEnvelope struct {
	MLflow json.RawMessage
	others map[string]json.RawMessage
}

func (e *pluginsOutputEnvelope) UnmarshalJSON(data []byte) error {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	e.MLflow = all[PluginName]
	delete(all, PluginName)
	if len(all) > 0 {
		e.others = all
	}
	return nil
}

func (e pluginsOutputEnvelope) MarshalJSON() ([]byte, error) {
	all := make(map[string]json.RawMessage, len(e.others)+1)
	for k, v := range e.others {
		all[k] = v
	}
	if len(e.MLflow) > 0 {
		all[PluginName] = e.MLflow
	}
	if len(all) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(all)
}

// set stores a plugin entry by name.
func (e *pluginsOutputEnvelope) set(name string, data json.RawMessage) {
	switch name {
	case PluginName:
		e.MLflow = data
	default:
		if e.others == nil {
			e.others = make(map[string]json.RawMessage)
		}
		e.others[name] = data
	}
}

func (e *pluginsOutputEnvelope) forEachEntry(fn func(name string, payload json.RawMessage)) {
	if len(e.MLflow) > 0 {
		fn(PluginName, e.MLflow)
	}
	for name, payload := range e.others {
		fn(name, payload)
	}
}

const (
	DefaultExperimentDescription = "Created by Kubeflow Pipelines"
	PluginName                   = "mlflow"
	TagKFPRunID                  = "kfp.pipeline_run_id"
	TagKFPRunURL                 = "kfp.pipeline_run_url"
	TagKFPPipelineID             = "kfp.pipeline_id"
	TagKFPPipelineVersionID      = "kfp.pipeline_version_id"
	EntryExperimentName          = "experiment_name"
	EntryExperimentID            = "experiment_id"
	EntryRootRunID               = "root_run_id"
	EntryRunURL                  = "run_url"
	EntryEndpoint                = "endpoint"
)

type Experiment struct {
	ID   string
	Name string
}

type RunSyncMode string

const (
	RunSyncModeTerminal RunSyncMode = "terminal"
	RunSyncModeRetry    RunSyncMode = "retry"
)

// EnsureExperimentExists looks up the MLflow experiment by ID or name, and creates it
// if it does not already exist.
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

// BuildKFPRunURL constructs a KFP pipeline run URL.
func BuildKFPRunURL(runID, kfpBaseURL string) string {
	if runID == "" {
		return ""
	}
	path := fmt.Sprintf("/#/runs/details/%s", runID)
	if kfpBaseURL != "" {
		return strings.TrimRight(kfpBaseURL, "/") + path
	}
	return path
}

// BuildKFPTags builds MLflow tags containing KFP metadata for a pipeline run.
func BuildKFPTags(run *apiserverPlugins.PendingRun, kfpBaseURL string) []commonmlflow.Tag {
	if run == nil {
		return nil
	}
	tags := []commonmlflow.Tag{
		{Key: TagKFPRunID, Value: run.RunID},
		{Key: TagKFPRunURL, Value: BuildKFPRunURL(run.RunID, kfpBaseURL)},
	}
	if run.PipelineID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: run.PipelineID})
	}
	if run.PipelineVersionID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: run.PipelineVersionID})
	}
	return tags
}

func BuildRunURL(requestCtx *commonmlflow.RequestContext, experimentID, runID string) string {
	if requestCtx == nil || requestCtx.BaseURL == nil || experimentID == "" || runID == "" {
		return ""
	}
	u := *requestCtx.BaseURL
	basePath := stringsTrimRightSlash(u.Path)
	u.Path = fmt.Sprintf("%s/experiments/%s/runs/%s", basePath, url.PathEscape(experimentID), url.PathEscape(runID))
	if requestCtx.WorkspacesEnabled && requestCtx.Workspace != "" {
		q := u.Query()
		q.Set("workspace", requestCtx.Workspace)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func SuccessfulPluginOutput(experimentID, experimentName, runID, runURL, endpoint string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
}

func FailedPluginOutput(experimentID, experimentName, runID, runURL, endpoint, stateMessage string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_FAILED, stateMessage)
}

// upsertPluginOutput merges a single plugin's output into an existing
// plugins_output JSON string, returning the updated JSON.
func upsertPluginOutput(existing *string, pluginName string, output *apiv2beta1.PluginOutput) (string, error) {
	marshaledOutput, err := protojson.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugin output for %q: %w", pluginName, err)
	}
	var envelope pluginsOutputEnvelope
	if existing != nil && *existing != "" {
		if err := json.Unmarshal([]byte(*existing), &envelope); err != nil {
			return "", fmt.Errorf("failed to unmarshal existing plugins_output: %w", err)
		}
	}
	envelope.set(pluginName, marshaledOutput)
	marshaledMap, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugins_output map: %w", err)
	}
	return string(marshaledMap), nil
}

// ModelToPersistedRun converts a model.Run to a PersistedRun for the
// post-run plugin hooks (OnRunEnd, OnRunRetry).
func ModelToPersistedRun(m *model.Run, namespace string) (*apiserverPlugins.PersistedRun, error) {
	if m == nil {
		return nil, fmt.Errorf("model.Run is nil")
	}
	pluginsOutput, err := DeserializePluginsOutput(m.PluginsOutputString)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize plugins_output for run %q: %w", m.UUID, err)
	}
	pr := &apiserverPlugins.PersistedRun{
		RunID:         m.UUID,
		Namespace:     namespace,
		State:         string(m.RunDetails.State),
		PluginsOutput: pluginsOutput,
	}
	if m.RunDetails.FinishedAtInSec > 0 {
		t := time.Unix(m.RunDetails.FinishedAtInSec, 0)
		pr.FinishedAt = &t
	}
	return pr, nil
}

// SetPendingRunPluginOutput serializes the given PluginOutput into PendingRun.PluginsOutput.
func SetPendingRunPluginOutput(run *apiserverPlugins.PendingRun, pluginName string, output *apiv2beta1.PluginOutput) error {
	if run == nil || output == nil || pluginName == "" {
		return nil
	}
	result, err := upsertPluginOutput(run.PluginsOutput, pluginName, output)
	if err != nil {
		return err
	}
	run.PluginsOutput = &result
	return nil
}

func DeserializePluginsOutput(raw *model.LargeText) (map[string]*apiv2beta1.PluginOutput, error) {
	result := make(map[string]*apiv2beta1.PluginOutput)
	if raw == nil || *raw == "" {
		return result, nil
	}
	var envelope pluginsOutputEnvelope
	if err := json.Unmarshal([]byte(*raw), &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugins_output: %w", err)
	}
	envelope.forEachEntry(func(name string, payload json.RawMessage) {
		output := &apiv2beta1.PluginOutput{}
		if err := protojson.Unmarshal(payload, output); err == nil {
			result[name] = output
		}
	})
	return result, nil
}

func SerializePluginsOutput(outputs map[string]*apiv2beta1.PluginOutput) (*model.LargeText, error) {
	if len(outputs) == 0 {
		return nil, nil
	}
	var envelope pluginsOutputEnvelope
	for key, output := range outputs {
		marshaledOutput, err := protojson.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal plugin output for %q: %w", key, err)
		}
		envelope.set(key, marshaledOutput)
	}
	marshaledMap, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugins_output map: %w", err)
	}
	lt := model.LargeText(string(marshaledMap))
	return &lt, nil
}

// PersistPluginsOutput serializes the PersistedRun's PluginsOutput and writes
// it to the database via the given store.
func PersistPluginsOutput(run *apiserverPlugins.PersistedRun, store apiserverPlugins.RunPluginOutputStore) error {
	lt, err := SerializePluginsOutput(run.PluginsOutput)
	if err != nil {
		return fmt.Errorf("failed to serialize plugins_output for run %q: %w", run.RunID, err)
	}
	return store.UpdateRunPluginsOutput(run.RunID, lt)
}

func GetStringEntry(output *apiv2beta1.PluginOutput, key string) string {
	if output == nil || output.Entries == nil || key == "" {
		return ""
	}
	entry, ok := output.Entries[key]
	if !ok || entry == nil || entry.Value == nil {
		return ""
	}
	return entry.Value.GetStringValue()
}

func GetParentRunID(output *apiv2beta1.PluginOutput) string {
	return GetStringEntry(output, EntryRootRunID)
}

func SetPluginOutputState(output *apiv2beta1.PluginOutput, state apiv2beta1.PluginState, stateMessage string) {
	if output == nil {
		return
	}
	output.State = state
	output.StateMessage = stateMessage
}

// maxSearchPages caps SearchRuns pagination to prevent infinite loops.
const maxSearchPages = 100

// maxNestingDepth caps recursive nested run traversal.
const maxNestingDepth = 4

func SyncParentAndNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode RunSyncMode, terminalStatus string, endTimeMs *int64) []string {
	if requestCtx == nil || requestCtx.Client == nil {
		return []string{"MLflow request context is required"}
	}
	if parentRunID == "" {
		return []string{"MLflow parent run_id is required"}
	}
	targetStatus := terminalStatus
	parentAction := "update parent run status"
	switch mode {
	case RunSyncModeRetry:
		targetStatus = "RUNNING"
		parentAction = "reopen parent run"
	case RunSyncModeTerminal:
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

// syncNestedRuns searches for MLflow runs tagged with the given parentRunID and
// updates their status. It recurses into each found run to handle deeper nesting
// (e.g., parent → loop nested run → iteration nested run).
func syncNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode RunSyncMode, targetStatus string, endTimeMs *int64, depth int) []string {
	if depth >= maxNestingDepth {
		return []string{fmt.Sprintf("max nesting depth (%d) reached when syncing children of run %s", maxNestingDepth, parentRunID)}
	}
	action := "close nested run"
	if mode == RunSyncModeRetry {
		action = "reopen nested run"
	}
	var syncErrors []string
	filter := fmt.Sprintf("tags.mlflow.parentRunId = '%s'", parentRunID)
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
			if nestedRunID == "" || nestedRunID == parentRunID || !shouldSyncNestedRun(mode, mlflowRun.Info.Status) {
				continue
			}
			// Recurse into children before updating this run .
			childErrors := syncNestedRuns(ctx, requestCtx, nestedRunID, experimentID, mode, targetStatus, endTimeMs, depth+1)
			syncErrors = append(syncErrors, childErrors...)
			if err := requestCtx.Client.UpdateRun(ctx, nestedRunID, targetStatus, endTimeMs); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("failed to %s %s: %v", action, nestedRunID, err))
			}
		}
		if searchResp.NextPageToken == "" {
			break
		}
		pageToken = searchResp.NextPageToken
	}
	return syncErrors
}

func buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint string, state apiv2beta1.PluginState, stateMessage string) *apiv2beta1.PluginOutput {
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
	if endpoint != "" {
		entries[EntryEndpoint] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(endpoint)}
	}
	return &apiv2beta1.PluginOutput{
		Entries:      entries,
		State:        state,
		StateMessage: stateMessage,
	}
}

func stringsTrimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func shouldSyncNestedRun(mode RunSyncMode, status string) bool {
	upperStatus := strings.ToUpper(status)
	switch mode {
	case RunSyncModeTerminal:
		return upperStatus == "RUNNING" || upperStatus == "SCHEDULED"
	case RunSyncModeRetry:
		return upperStatus == "FAILED" || upperStatus == "KILLED"
	default:
		return false
	}
}

type searchRunPayload struct {
	Info struct {
		RunID   string `json:"run_id"`
		RunUUID string `json:"run_uuid"`
		Status  string `json:"status"`
	} `json:"info"`
}
