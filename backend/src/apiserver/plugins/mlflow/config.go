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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// DefaultExperimentName is the MLflow experiment name used when the user
	// and admin configuration do not specify one.
	DefaultExperimentName        = "KFP-Default"
	DefaultExperimentDescription = "Created by Kubeflow Pipelines"
	PluginName                   = "MLflow"
	TagKFPRunID                  = "kfp.pipeline_run_id"
	TagKFPRunURL                 = "kfp.pipeline_run_url"
	TagKFPPipelineID             = "kfp.pipeline_id"
	TagKFPPipelineVersionID      = "kfp.pipeline_version_id"
)

// MLflowPluginInput represents the user-facing plugins_input.mlflow schema.
type MLflowPluginInput struct {
	ExperimentName string `json:"experiment_name,omitempty"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
}

// IsEnabled reports whether the global plugins.mlflow configuration is present,
// indicating the API server has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet("plugins.mlflow")
}

type Experiment struct {
	ID   string
	Name string
}

// ResolveMLflowPluginConfig builds an MLflowPluginConfig for the given input generic PluginConfig.
func ResolveMLflowPluginConfig(runPluginConfig *apiserverPlugins.PluginConfig, resolvedMLflowSettings *commonmlflow.MLflowPluginSettings) (*commonmlflow.MLflowPluginConfig, error) {
	if runPluginConfig == nil || resolvedMLflowSettings == nil {
		return nil, fmt.Errorf("runPluginConfig and resolvedMLflowSettings must be non-nil")
	}

	resolvedTimeout := runPluginConfig.Timeout
	if resolvedTimeout == "" {
		resolvedTimeout = apiserverPlugins.DefaultTimeout
	}

	resolvedMLflowCfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: runPluginConfig.Endpoint,
		Timeout:  resolvedTimeout,
		TLS:      runPluginConfig.TLS,
		Settings: resolvedMLflowSettings,
	}
	return resolvedMLflowCfg, nil
}

// BuildMLflowRunRequestContext constructs a fully initialized RequestContext by
// performing API-server-specific validation and then delegating to the common
// BuildRequestContext.
func BuildMLflowRunRequestContext(namespace string, requestCfg *commonmlflow.MLflowPluginConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "cannot build MLflow request context without a resolved config")
	}
	if requestCfg.Endpoint == "" {
		return nil, util.NewInvalidInputError("plugins.mlflow endpoint must be set")
	}
	settings := requestCfg.Settings
	if settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "BuildMLflowRequestContext requires resolved settings")
	}
	workspacesEnabled := settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled
	return commonmlflow.BuildMLflowRequestContext(*requestCfg, namespace, workspacesEnabled)
}

// ResolveMLflowPluginInput parses the plugins_input.mlflow JSON from a run model,
// validates it against the MLflowPluginInput schema, and applies defaults.
func ResolveMLflowPluginInput(pluginsInputString *string) (*MLflowPluginInput, error) {
	defaultInput := &MLflowPluginInput{ExperimentName: DefaultExperimentName}

	if pluginsInputString == nil || *pluginsInputString == "" {
		return defaultInput, nil
	}

	var pluginInputs apiserverPlugins.PluginsInputMap
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	var mlflowRaw json.RawMessage
	for key, value := range pluginInputs {
		if strings.EqualFold(key, PluginName) {
			mlflowRaw = value
			break
		}
	}
	if len(mlflowRaw) == 0 {
		return defaultInput, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(mlflowRaw))
	decoder.DisallowUnknownFields()
	input := &MLflowPluginInput{}
	if err := decoder.Decode(input); err != nil {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must follow schema {experiment_name?: string, experiment_id?: string, disabled?: bool}: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must be a single JSON object")
	}

	if input.Disabled {
		return input, nil
	}
	if input.ExperimentID != "" {
		return input, nil
	}
	if input.ExperimentName == "" {
		input.ExperimentName = DefaultExperimentName
	}
	return input, nil
}

// ResolvePluginSettings parses and validates MLflow plugin settings from raw map, and applies default values where
// necessary.
func ResolvePluginSettings(rawSettings map[string]interface{}) *commonmlflow.MLflowPluginSettings {
	var settings commonmlflow.MLflowPluginSettings
	for key, value := range rawSettings {
		switch strings.ToLower(key) {
		case "workspacesenabled":
			settings.WorkspacesEnabled = asBoolPointer(key, value)
		case "experimentdescription":
			settings.ExperimentDescription = asStringPointer(key, value)
		case "defaultexperimentname":
			if s, ok := asString(key, value); ok {
				settings.DefaultExperimentName = s
			}
		case "kfpbaseurl":
			if s, ok := asString(key, value); ok {
				settings.KFPBaseURL = s
			}
		case "kfprunurlpathtemplate":
			if s, ok := asString(key, value); ok {
				settings.KFPRunURLPathTemplate = s
			}
		case "mlflowbaseurl":
			if s, ok := asString(key, value); ok {
				settings.MLflowBaseURL = s
			}
		case "mlflowuipathprefix":
			if s, ok := asString(key, value); ok {
				settings.MLflowUIPathPrefix = s
			}
		case "injectuserenvvars":
			settings.InjectUserEnvVars = asBoolPointer(key, value)
		default:
			glog.Warningf("unrecognized MLflow plugin setting: %s", key)
		}
	}
	return ApplySettingsDefaults(&settings)
}

func asBoolPointer(key string, value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case *bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			glog.Errorf("failed to parse %s as bool from MLflow plugin settings: %v", key, err)
			return nil
		}
		return &parsed
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return nil
	}
}

func asStringPointer(key string, value interface{}) *string {
	if s, ok := asString(key, value); ok {
		return &s
	}
	return nil
}

func asString(key string, value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
		return "", false
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return "", false
	}
}

// GetGlobalMLflowConfig reads the global plugins.mlflow configuration
func GetGlobalMLflowConfig() (apiserverPlugins.PluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return apiserverPlugins.PluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg apiserverPlugins.PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

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

// BuildKFPRunURL builds a link from kfpBaseURL to the pipeline run details page.
func BuildKFPRunURL(runID, namespace, kfpBaseURL, pathTemplate string) string {
	if runID == "" || kfpBaseURL == "" {
		glog.V(4).Infof(
			"BuildKFPRunURL returned empty URL due to missing input(s): runID_empty=%t kfpBaseURL_empty=%t",
			runID == "",
			kfpBaseURL == "",
		)
		return ""
	}
	pathTemplate = strings.TrimSpace(pathTemplate)
	if pathTemplate == "" {
		base := strings.TrimRight(kfpBaseURL, "/")
		return fmt.Sprintf("%s/#/runs/details/%s", base, url.PathEscape(runID))
	}
	if namespace == "" && strings.Contains(pathTemplate, "{namespace}") {
		glog.V(4).Infof("BuildKFPRunURL returned empty URL: namespace required when template contains {namespace}")
		return ""
	}
	base := strings.TrimRight(kfpBaseURL, "/")
	rendered := strings.ReplaceAll(pathTemplate, "{run_id}", url.PathEscape(runID))
	rendered = strings.ReplaceAll(rendered, "{namespace}", url.PathEscape(namespace))
	if !strings.HasPrefix(rendered, "/") && !strings.HasPrefix(rendered, "#") {
		rendered = "/" + rendered
	}
	return base + rendered
}

// BuildKFPTags builds MLflow tags containing KFP metadata for a pipeline run.
func BuildKFPTags(run *apiserverPlugins.PendingRun, kfpBaseURL, kfpRunURLPathTemplate string) []commonmlflow.Tag {
	if run == nil {
		return nil
	}
	tags := []commonmlflow.Tag{
		{Key: TagKFPRunID, Value: run.RunID},
		{Key: TagKFPRunURL, Value: BuildKFPRunURL(run.RunID, run.Namespace, kfpBaseURL, kfpRunURLPathTemplate)},
	}
	if run.PipelineID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: run.PipelineID})
	}
	if run.PipelineVersionID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: run.PipelineVersionID})
	}
	return tags
}

// ApplySettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplySettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := true
		settings.WorkspacesEnabled = &defaultEnabled
	}
	if settings.DefaultExperimentName == "" {
		settings.DefaultExperimentName = DefaultExperimentName
	}
	if settings.ExperimentDescription == nil {
		d := DefaultExperimentDescription
		settings.ExperimentDescription = &d
	}
	return settings
}

// SelectMLflowExperiment chooses the selector used for MLflow experiment resolution.
// Priority: user-provided experiment_id > user-provided experiment_name >
// admin-configured defaultExperimentName > hardcoded "KFP-Default".
func SelectMLflowExperiment(input *MLflowPluginInput, settings *commonmlflow.MLflowPluginSettings) (experimentID string, experimentName string) {
	if input != nil {
		if input.ExperimentID != "" {
			return input.ExperimentID, ""
		}
		if input.ExperimentName != "" {
			return "", input.ExperimentName
		}
	}
	if settings != nil && settings.DefaultExperimentName != "" {
		return "", settings.DefaultExperimentName
	}
	return "", DefaultExperimentName
}

// ToMLflowTerminalStatus converts a KFP RuntimeState string to an MLflow
// terminal status.
func ToMLflowTerminalStatus(stateV2 string) string {
	switch stateV2 {
	case "SUCCEEDED":
		return "FINISHED"
	case "CANCELED", "CANCELING":
		return "KILLED"
	default:
		return "FAILED"
	}
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

func SuccessfulPluginOutput(experimentID, experimentName, runID, runURL, endpoint string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
}

func FailedPluginOutput(experimentID, experimentName, runID, runURL, endpoint, stateMessage string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_FAILED, stateMessage)
}

// maxSearchPages caps SearchRuns pagination to prevent infinite loops.
const maxSearchPages = 100

// maxNestingDepth caps recursive nested run traversal.
const maxNestingDepth = 4

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

// syncNestedRuns searches for MLflow runs tagged with the given parentRunID and
// updates their status. It recurses into each found run to handle deeper nesting
// (e.g., parent → loop nested run → iteration nested run).
func syncNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode apiserverPlugins.RunSyncMode, targetStatus string, endTimeMs *int64, depth int) []string {
	if depth >= maxNestingDepth {
		return []string{fmt.Sprintf("max nesting depth (%d) reached when syncing children of run %s", maxNestingDepth, parentRunID)}
	}
	action := "close nested run"
	if mode == apiserverPlugins.RunSyncModeRetry {
		action = "reopen nested run"
	}
	var syncErrors []string
	filter := fmt.Sprintf(`tags.%q = '%s'`, commonmlflow.TagNestedRunParentRunID, parentRunID)
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
		entries[apiserverPlugins.EntryExperimentName] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentName)}
	}
	if experimentID != "" {
		entries[apiserverPlugins.EntryExperimentID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentID)}
	}
	if runID != "" {
		entries[apiserverPlugins.EntryRootRunID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(runID)}
	}
	if runURL != "" {
		entries[apiserverPlugins.EntryRunURL] = &apiv2beta1.MetadataValue{
			Value:      structpb.NewStringValue(runURL),
			RenderType: apiv2beta1.MetadataValue_URL.Enum(),
		}
	}
	if endpoint != "" {
		entries[apiserverPlugins.EntryEndpoint] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(endpoint)}
	}
	return &apiv2beta1.PluginOutput{
		Entries:      entries,
		State:        state,
		StateMessage: stateMessage,
	}
}

func shouldSyncNestedRun(mode apiserverPlugins.RunSyncMode, status string) bool {
	upperStatus := strings.ToUpper(status)
	switch mode {
	case apiserverPlugins.RunSyncModeTerminal:
		return upperStatus != "FINISHED" && upperStatus != "FAILED" && upperStatus != "KILLED"
	case apiserverPlugins.RunSyncModeRetry:
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
