// Copyright 2018 The Kubeflow Authors
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

package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	urlSchemeJavaScript = "javascript:"
	urlSchemeData       = "data:"
	urlSchemeVBScript   = "vbscript:"

	pluginErrInvalidLimitsConfig      = "invalid plugin limits configuration"
	pluginErrPluginsInputTooManyKeys  = "number of plugins_input entries"
	pluginErrPluginsInputNilEntry     = "plugins_input[%q] must not be nil"
	pluginErrPluginsInputInvalidValue = "plugins_input[%q] contains invalid nested value"
	pluginErrPluginsInputNestingDepth = "plugins_input[%q] nesting depth exceeds maximum"
	pluginErrPluginsInputEntrySize    = "plugins_input[%q] size"
	pluginErrPluginsInputTotalSize    = "plugins_input total size"
	pluginErrPluginsInputMarshalEntry = "marshal plugins_input[%q]: %w"
	pluginErrPluginsInputMarshalMap   = "marshal plugins_input map: %w"

	pluginErrPluginsOutputTooManyKeys  = "number of plugins_output entries"
	pluginErrPluginsOutputEntrySize    = "plugins_output[%q] size"
	pluginErrPluginsOutputTotalSize    = "plugins_output total size"
	pluginErrPluginsOutputMarshalEntry = "marshal plugins_output[%q]: %w"
	pluginErrPluginsOutputMarshalMap   = "marshal plugins_output map: %w"
	pluginErrPluginsOutputNilMetadata  = "plugins_output[%q].entries[%q] metadata must not be nil"
	pluginErrPluginsOutputNilValue     = "plugins_output[%q].entries[%q].value must not be nil"
	pluginErrPluginsOutputInvalidValue = "plugins_output[%q].entries[%q] contains invalid nested value"
	pluginErrPluginsOutputNestingDepth = "plugins_output[%q].entries[%q] nesting depth exceeds maximum"

	pluginErrStructValueNil = "struct value must not be nil"
	pluginErrStructFieldNil = "struct field %q must not be nil"
	pluginErrValueNil       = "value must not be nil"
	pluginErrValueKindUnset = "unsupported or unset value kind"

	pluginErrExceedsMaxBytes = " (%d bytes) exceeds maximum %d bytes"
)

// Converts API experiment to its internal representation.
func toModelExperiment(apiExperiment *apiv2beta1.Experiment) (*model.Experiment, error) {
	name := apiExperiment.GetDisplayName()
	namespace := apiExperiment.GetNamespace()
	description := apiExperiment.GetDescription()
	if name == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Experiment must have a non-empty name"), "Failed to convert API experiment to model experiment")
	}
	// Namespace validation is handled in the server layer as it depends on multi-user mode.
	exp := &model.Experiment{
		Name:         name,
		Description:  description,
		Namespace:    namespace,
		StorageState: model.StorageStateAvailable,
	}
	if err := validation.ValidateModel(exp); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to convert API experiment to model experiment",
		)
	}
	return exp, nil
}

// Converts internal experiment representation to its API counterpart.
// Supports v2beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiExperiment(experiment *model.Experiment) *apiv2beta1.Experiment {
	if experiment == nil {
		return &apiv2beta1.Experiment{}
	}
	var storageState apiv2beta1.Experiment_StorageState
	switch experiment.StorageState {
	case "AVAILABLE", "STORAGESTATE_AVAILABLE":
		storageState = apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"])
	case "ARCHIVED", "STORAGESTATE_ARCHIVED":
		storageState = apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"])
	default:
		storageState = apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["STORAGE_STATE_UNSPECIFIED"])
	}
	return &apiv2beta1.Experiment{
		ExperimentId:     experiment.UUID,
		DisplayName:      experiment.Name,
		Description:      experiment.Description,
		CreatedAt:        timestamppb.New(time.Unix(experiment.CreatedAtInSec, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(experiment.LastRunCreatedAtInSec, 0)),
		Namespace:        experiment.Namespace,
		StorageState:     storageState,
	}
}

// Converts an array of internal experiment representations to an array of API experiments.
// Supports v2beta1 API.
func toApiExperiments(experiments []*model.Experiment) []*apiv2beta1.Experiment {
	apiExperiments := make([]*apiv2beta1.Experiment, 0)
	for _, experiment := range experiments {
		apiExperiments = append(apiExperiments, toApiExperiment(experiment))
	}
	return apiExperiments
}

// Converts API pipeline to its internal representation.
func toModelPipeline(apiPipeline *apiv2beta1.Pipeline) (*model.Pipeline, error) {
	namespace := apiPipeline.GetNamespace()
	name := apiPipeline.GetName()
	displayName := apiPipeline.GetDisplayName()
	description := apiPipeline.GetDescription()
	tags := apiPipeline.GetTags()

	// Previously display_name was the required API field and name didn't exist. Name is now the required API field
	// but if display_name is provided, we use it as the name.
	if name == "" {
		name = displayName
	}
	// If display_name is not provided, we use the name as the display_name for backward compatibility and convenience.
	if displayName == "" {
		displayName = name
	}

	// Build the full model first, then validate the actual values on the struct.
	pipeline := &model.Pipeline{
		Name:        name,
		DisplayName: displayName,
		Namespace:   namespace,
		Description: model.LargeText(description),
		Status:      model.PipelineCreating,
		Tags:        tags,
	}

	if err := validation.ValidateModel(pipeline); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to convert API pipeline to model pipeline",
		)
	}

	return pipeline, nil

}

// Converts internal pipeline representation to its API counterpart.
// Supports v2beta1 API.
// Input pipeline must have UUID, Name, Namespace, and CreateAt set to non-default values.
// Note: stores details inside the message if a parsing error occurs.
func toApiPipeline(pipeline *model.Pipeline) *apiv2beta1.Pipeline {
	if pipeline == nil {
		return &apiv2beta1.Pipeline{
			PipelineId: "",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline cannot be nil"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		}
	}

	if pipeline.UUID == "" {
		return &apiv2beta1.Pipeline{
			PipelineId: "",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline id cannot be empty"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		}
	}

	if pipeline.CreatedAtInSec == 0 {
		return &apiv2beta1.Pipeline{
			PipelineId: pipeline.UUID,
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline create time cannot be 0"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		}
	}

	if pipeline.Name == "" {
		return &apiv2beta1.Pipeline{
			PipelineId: pipeline.UUID,
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline name cannot be empty"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		}
	}

	return &apiv2beta1.Pipeline{
		PipelineId:  pipeline.UUID,
		Name:        pipeline.Name,
		DisplayName: pipeline.DisplayName,
		Description: string(pipeline.Description),
		CreatedAt:   timestamppb.New(time.Unix(pipeline.CreatedAtInSec, 0)),
		Namespace:   pipeline.Namespace,
		Tags:        pipeline.Tags,
	}
}

// Converts arrays of internal pipeline representations and pipeline version representations
// to an array of API pipelines.
// Supports v2beta1 API.
func toApiPipelines(pipelines []*model.Pipeline) []*apiv2beta1.Pipeline {
	apiPipelines := make([]*apiv2beta1.Pipeline, 0)
	for _, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, toApiPipeline(pipeline))
	}
	return apiPipelines
}

// Converts API pipeline to its internal representation.
func toModelPipelineVersion(p *apiv2beta1.PipelineVersion) (*model.PipelineVersion, error) {
	if p.GetPackageUrl() == nil || len(p.GetPackageUrl().GetPipelineUrl()) == 0 {
		return nil, util.NewInvalidInputError("Failed to convert v2beta1 API pipeline version to its internal representation due to missing pipeline URL")
	}
	name := p.GetName()
	displayName := p.GetDisplayName()
	pipelineID := p.GetPipelineId()
	pipelineURL := p.GetPackageUrl().GetPipelineUrl()
	codeURL := p.GetCodeSourceUrl()
	description := p.GetDescription()

	// Previously display_name was the required API field and name didn't exist. Name is now the required API field
	// but if display_name is provided, we use it as the name.
	if name == "" {
		name = displayName
	}
	// If display_name is not provided, we use the name as the display_name for backward compatibility and convenience.
	if displayName == "" {
		displayName = name
	}
	// Extract tags if present (v2beta1 only)
	tags := p.GetTags()

	pv := &model.PipelineVersion{
		Name:            name,
		DisplayName:     displayName,
		PipelineId:      pipelineID,
		PipelineSpecURI: model.LargeText(pipelineURL),
		CodeSourceUrl:   codeURL,
		Description:     model.LargeText(description),
		Status:          model.PipelineVersionCreating,
		Tags:            tags,
	}
	if err := validation.ValidateModel(pv); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to convert API pipeline version to model pipeline version",
		)
	}
	return pv, nil
}

// Converts internal pipeline version representation to its API counterpart.
// Supports v2beta1 API.
// Note: stores details inside the message if a parsing error occurs.
func toApiPipelineVersion(pv *model.PipelineVersion) *apiv2beta1.PipelineVersion {
	if pv == nil {
		return &apiv2beta1.PipelineVersion{
			PipelineVersionId: "",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline version cannot be nil"),
					"Failed to convert a pipeline version to API pipeline version",
				),
			),
		}
	}
	// Validate pipeline version id
	if pv.UUID == "" {
		return &apiv2beta1.PipelineVersion{
			PipelineVersionId: "",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline version id cannot be empty"),
					"Failed to convert a pipeline version to API pipeline version",
				),
			),
		}
	}
	// Validate creation time
	if pv.CreatedAtInSec == 0 {
		return &apiv2beta1.PipelineVersion{
			PipelineVersionId: pv.UUID,
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Create time can not be 0"),
					"Failed to convert a pipeline versions to API pipeline version",
				),
			),
		}
	}

	apiPipelineVersion := &apiv2beta1.PipelineVersion{
		PipelineId:        pv.PipelineId,
		PipelineVersionId: pv.UUID,
		Name:              pv.Name,
		DisplayName:       pv.DisplayName,
		Description:       string(pv.Description),
		CreatedAt:         timestamppb.New(time.Unix(pv.CreatedAtInSec, 0)),
		Tags:              pv.Tags,
	}

	// Set code source url
	if pv.CodeSourceUrl != "" {
		apiPipelineVersion.CodeSourceUrl = pv.CodeSourceUrl
	}

	// Set package url from pipeline spec URI
	if pv.PipelineSpecURI != "" {
		apiPipelineVersion.PackageUrl = &apiv2beta1.Url{
			PipelineUrl: string(pv.PipelineSpecURI),
		}
	}

	// Convert pipeline spec
	spec, err := YamlStringToPipelineSpecStruct(string(pv.PipelineSpec))
	if err != nil {
		return &apiv2beta1.PipelineVersion{
			PipelineVersionId: pv.UUID,
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					err,
					"Failed to convert a pipeline versions to API pipeline version due to error in parsing its pipeline spec yaml",
				),
			),
		}
	}
	if len(spec.GetFields()) > 0 {
		apiPipelineVersion.PipelineSpec = spec
	}
	return apiPipelineVersion
}

// Converts an array of internal pipeline version representations to an array of API pipeline versions.
// Supports v2beta1 API.
func toApiPipelineVersions(pv []*model.PipelineVersion) []*apiv2beta1.PipelineVersion {
	apiVersions := make([]*apiv2beta1.PipelineVersion, 0)
	for _, version := range pv {
		apiVersions = append(apiVersions, toApiPipelineVersion(version))
	}
	return apiVersions
}

// Converts API runtime config to internal representation of parameters.
// Converts engine parameters or v2 runtime parameters into stored JSON.
// Runtime config's parameters stored as map[string]*structpb.Value are translated to
// a string representation of a map object, while an array of parameters is translated
// to a string representation of an array of maps.
// For example:
//
//	{"param1": "value1"} -> `{"param1": "value1"}`
//	[{"param1": "value1"}] -> `[{"param1": "value1"}]`
func toModelParameters(obj interface{}) (string, error) {
	if obj == nil {
		return "", nil
	}
	switch obj := obj.(type) {
	case util.SpecParameters:
		// This will translate to an array of parameters
		specParams := obj
		paramsString, err := util.MarshalParameters(util.ArgoWorkflow, specParams)
		if err != nil {
			return "", util.NewInternalServerError(err, "Failed to convert an array of SpecParameters to their internal representation")
		}
		if len(paramsString) > util.MaxParameterBytes {
			return "", util.NewInvalidInputError("The input parameter length exceed maximum size of %v", util.MaxParameterBytes)
		}
		if paramsString == "[]" {
			paramsString = ""
		}
		return paramsString, nil

	case map[string]*structpb.Value:
		// This will translate to a map of parameters
		protoStructParams := obj
		paramsBytes, err := json.Marshal(protoStructParams)
		if err != nil {
			return "", util.NewInternalServerError(err, "Failed to marshal RuntimeConfig API parameters as string")
		}
		paramsString := string(paramsBytes)
		if paramsString == "null" {
			paramsString = ""
		}
		return paramsString, nil

	case *apiv2beta1.RuntimeConfig:
		runtimeConfig := obj
		protoParams := runtimeConfig.GetParameters()
		if protoParams == nil {
			return "", util.NewInternalServerError(util.NewInvalidInputError("Parameters cannot be nil"), "Failed to convert API runtime config to internal parameters representation")
		}
		return toModelParameters(protoParams)
	default:
		return "", util.NewUnknownApiVersionError("Parameters", obj)
	}
}

// Converts internal runtime parameters to (name, value) pairs as map[string]*structpb.Value.
// Note: returns nil if a parsing error occurs.
func toMapProtoStructParameters(p string) map[string]*structpb.Value {
	protoParams := make(map[string]*structpb.Value, 0)
	if p == "" || p == "null" || p == "[]" {
		return protoParams
	}
	if err := json.Unmarshal([]byte(p), &protoParams); err == nil {
		return protoParams
	}
	// Historical records stored string parameters as an array, not an IR map.
	var legacyParams []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(p), &legacyParams); err != nil {
		return nil
	}
	protoParams = make(map[string]*structpb.Value, len(legacyParams))
	for _, param := range legacyParams {
		protoParams[param.Name] = structpb.NewStringValue(param.Value)
	}
	return protoParams
}

// Converts API trigger to its internal representation.
// A nil API trigger converts to an empty model trigger, matching recurring
// runs that do not declare a schedule.
func toModelTrigger(apiTrigger *apiv2beta1.Trigger) *model.Trigger {
	modelTrigger := model.Trigger{}
	if apiTrigger.GetCronSchedule() != nil {
		cronSchedule := apiTrigger.GetCronSchedule()
		modelTrigger.CronSchedule = model.CronSchedule{Cron: &cronSchedule.Cron}
		if cronSchedule.StartTime != nil {
			modelTrigger.CronScheduleStartTimeInSec = &cronSchedule.StartTime.Seconds
		}
		if cronSchedule.EndTime != nil {
			modelTrigger.CronScheduleEndTimeInSec = &cronSchedule.EndTime.Seconds
		}
	}
	if apiTrigger.GetPeriodicSchedule() != nil {
		periodicSchedule := apiTrigger.GetPeriodicSchedule()
		modelTrigger.PeriodicSchedule = model.PeriodicSchedule{
			IntervalSecond: &periodicSchedule.IntervalSecond,
		}
		if apiTrigger.GetPeriodicSchedule().StartTime != nil {
			modelTrigger.PeriodicScheduleStartTimeInSec = &periodicSchedule.StartTime.Seconds
		}
		if apiTrigger.GetPeriodicSchedule().EndTime != nil {
			modelTrigger.PeriodicScheduleEndTimeInSec = &periodicSchedule.EndTime.Seconds
		}
	}
	return &modelTrigger
}

// Converts internal trigger representation to its API counterpart.
// Supports v2beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiTrigger(trigger *model.Trigger) *apiv2beta1.Trigger {
	if trigger == nil {
		return &apiv2beta1.Trigger{}
	}
	if trigger.Cron != nil && *trigger.Cron != "" {
		var cronSchedule apiv2beta1.CronSchedule
		cronSchedule.Cron = *trigger.Cron
		if trigger.CronScheduleStartTimeInSec != nil {
			cronSchedule.StartTime = timestamppb.New(time.Unix(*trigger.CronScheduleStartTimeInSec, 0))
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = timestamppb.New(time.Unix(*trigger.CronScheduleEndTimeInSec, 0))
		}
		return &apiv2beta1.Trigger{Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}
	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
		var periodicSchedule apiv2beta1.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = timestamppb.New(time.Unix(*trigger.PeriodicScheduleStartTimeInSec, 0))
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = timestamppb.New(time.Unix(*trigger.PeriodicScheduleEndTimeInSec, 0))
		}
		return &apiv2beta1.Trigger{Trigger: &apiv2beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &periodicSchedule}}
	}
	if trigger.IntervalSecond == nil && trigger.Cron == nil {
		return &apiv2beta1.Trigger{}
	}
	return nil
}

// Converts API runtime config to its internal representations.
func toModelRuntimeConfig(obj interface{}) (*model.RuntimeConfig, error) {
	if obj == nil {
		return nil, util.NewInvalidInputError("Failed to convert API runtime config to its internal representation. Input cannot be nil")
	}
	var params, root string
	switch obj := obj.(type) {
	case *apiv2beta1.RuntimeConfig:
		apiRuntimeConfigV2 := obj
		p, err := toModelParameters(apiRuntimeConfigV2.GetParameters())
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to convert API runtime config to its internal representation due to parameters conversion error")
		}
		params = p
		root = apiRuntimeConfigV2.GetPipelineRoot()
	case *pipelinespec.PipelineJob_RuntimeConfig:
		specRuntimeConfig := obj
		p, err := toModelParameters(specRuntimeConfig.GetParameterValues())
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to convert PipelineSpec's runtime config to its internal representation due to parameters conversion error")
		}
		params = p
		root = specRuntimeConfig.GetGcsOutputDirectory()
	default:
		return nil, util.NewUnknownApiVersionError("RuntimeConfig", obj)
	}
	return &model.RuntimeConfig{
		Parameters:   model.LargeText(params),
		PipelineRoot: model.LargeText(root),
	}, nil
}

// Converts internal runtime config representation to its API counterpart.
// Supports v2beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiRuntimeConfig(modelRuntime model.RuntimeConfig) *apiv2beta1.RuntimeConfig {
	apiRuntimeConfig := apiv2beta1.RuntimeConfig{}
	if modelRuntime.Parameters == "" && modelRuntime.PipelineRoot == "" {
		return &apiRuntimeConfig
	}
	runtimeParams := toMapProtoStructParameters(string(modelRuntime.Parameters))
	if runtimeParams == nil {
		return nil
	}
	apiRuntimeConfig.Parameters = runtimeParams
	apiRuntimeConfig.PipelineRoot = string(modelRuntime.PipelineRoot)
	return &apiRuntimeConfig
}

// Converts internal runtime config representation to PipelineSpec's runtime config.
// Note: returns nil if a parsing error occurs.
func toPipelineSpecRuntimeConfig(cfg *model.RuntimeConfig) *pipelinespec.PipelineJob_RuntimeConfig {
	if cfg == nil {
		return &pipelinespec.PipelineJob_RuntimeConfig{}
	}
	runtimeParams := toMapProtoStructParameters(string(cfg.Parameters))
	if runtimeParams == nil {
		return nil
	}
	return &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues:    runtimeParams,
		GcsOutputDirectory: string(cfg.PipelineRoot),
	}
}

// Converts API run to its internal representation.
func toModelRun(apiRunV2 *apiv2beta1.Run) (*model.Run, error) {
	var namespace, experimentId, pipelineName, pipelineId, pipelineVersionId string
	var recRunId, runName, runDesc, runId, specParams, cfgParams string
	var pipelineSpec, workflowSpec, runtimePipelineSpec, runtimeWorkflowSpec string
	var pipelineRoot, storageState, serviceAcc string
	var createTime, scheduleTime, finishTime int64
	var modelTasks []*model.Task
	var stateHistory []*model.RuntimeStatus
	var pluginsInputStr, pluginsOutputStr *string
	var err error
	state := toModelRuntimeState(apiRunV2.GetState())
	if temp, err := toModelRuntimeStatuses(apiRunV2.GetStateHistory()); err == nil {
		stateHistory = temp
	} else {
		return nil, util.NewInternalServerError(err, "Failed to convert a API run detail to its internal representation due to error converting runtime state history")
	}
	if pluginsInputStr, err = pluginsInputToJSON(apiRunV2.GetPluginsInput()); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert plugins_input to JSON")
	}
	pluginLimitsConfig, err := common.GetPluginLimitsConfig()
	if err != nil {
		return nil, util.NewInvalidInputError("Invalid plugins limits configuration: %v", err)
	}
	if err = validatePluginsInputLimits(apiRunV2.GetPluginsInput(), pluginLimitsConfig); err != nil {
		return nil, util.NewInvalidInputError("Invalid plugins_input: %v", err)
	}
	if err = validatePluginsOutputWithLimits(apiRunV2.GetPluginsOutput(), pluginLimitsConfig); err != nil {
		return nil, util.NewInvalidInputError("Invalid plugins_output: %v", err)
	}
	if pluginsOutputStr, err = pluginsOutputToJSON(apiRunV2.GetPluginsOutput()); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert plugins_output to JSON")
	}

	namespace = ""
	workflowSpec = ""
	// TODO(gkcalat): implement runtime details of a run logic based on the apiRunV2.RuDetails().
	runtimePipelineSpec = ""
	runtimeWorkflowSpec = ""
	runName = apiRunV2.GetDisplayName()
	if runName == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Run name cannot be empty"), "Failed to convert a API run detail to its internal representation")
	}
	pipelineId = apiRunV2.GetPipelineVersionReference().GetPipelineId()
	pipelineVersionId = apiRunV2.GetPipelineVersionReference().GetPipelineVersionId()
	experimentId = apiRunV2.GetExperimentId()
	runId = apiRunV2.GetRunId()
	recRunId = apiRunV2.GetRecurringRunId()

	specMap := apiRunV2.GetPipelineSpec().AsMap()
	if pv, ok := specMap["PipelineInfo"]; ok {
		if pName, ok := pv.(map[string]interface{})["Name"]; ok {
			resources := common.ParseResourceIdsFromFullName(pName.(string))
			if namespace == "" {
				namespace = resources["Namespace"]
			}
			if experimentId == "" {
				experimentId = resources["ExperimentId"]
			}
			if pipelineId == "" {
				pipelineId = resources[common.PipelineIDResourceNameKey]
			}
			if pipelineVersionId == "" {
				pipelineVersionId = resources[common.PipelineVersionIDResourceNameKey]
			}
			if runId == "" {
				runId = resources["RunID"]
			}
			if recRunId == "" {
				recRunId = resources["RecurringRunId"]
			}
		}
	}
	runDesc = apiRunV2.GetDescription()
	serviceAcc = apiRunV2.GetServiceAccount()
	if temp, err := toModelStorageState(apiRunV2.GetStorageState()); err == nil {
		storageState = temp.ToString()
	}

	createTime = apiRunV2.GetCreatedAt().GetSeconds()
	scheduleTime = apiRunV2.GetScheduledAt().GetSeconds()
	finishTime = apiRunV2.GetFinishedAt().GetSeconds()

	cfg, err := toModelRuntimeConfig(apiRunV2.GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert API run to its internal representation due to runtime config conversion error")
	}
	cfgParams = string(cfg.Parameters)
	pipelineRoot = string(cfg.PipelineRoot)

	if apiRunV2.GetPipelineSpec() == nil {
		pipelineSpec = ""
	} else if spec, err := pipelineSpecStructToYamlString(apiRunV2.GetPipelineSpec()); err == nil {
		pipelineSpec = spec
	} else {
		pipelineSpec = ""
	}
	specParams = ""

	if len(apiRunV2.Tasks) > 0 {
		for _, apiTask := range apiRunV2.Tasks {
			modelTask, err := toModelTask(apiTask)
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert API run to its internal representation due to task conversion error")
			}
			modelTasks = append(modelTasks, modelTask)
		}
	}
	if namespace != "" && pipelineVersionId != "" {
		pipelineName = fmt.Sprintf("namespaces/%v/pipelines/%v", namespace, pipelineVersionId)
	} else if pipelineVersionId != "" {
		pipelineName = fmt.Sprintf("pipelines/%v", pipelineVersionId)
	}
	modelRun := model.Run{
		UUID:           runId,
		DisplayName:    runName,
		Description:    runDesc,
		Namespace:      namespace,
		ExperimentId:   experimentId,
		RecurringRunId: recRunId,
		StorageState:   model.StorageState(storageState),
		ServiceAccount: serviceAcc,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipelineId,
			PipelineVersionId:    pipelineVersionId,
			PipelineName:         pipelineName,
			PipelineSpecManifest: model.LargeText(pipelineSpec),
			WorkflowSpecManifest: model.LargeText(workflowSpec),
			Parameters:           model.LargeText(specParams),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   model.LargeText(cfgParams),
				PipelineRoot: model.LargeText(pipelineRoot),
			},
		},
		RunDetails: model.RunDetails{
			State:                   state.ToV2(),
			StateHistory:            stateHistory,
			CreatedAtInSec:          createTime,
			ScheduledAtInSec:        scheduleTime,
			FinishedAtInSec:         finishTime,
			PipelineRuntimeManifest: model.LargeText(runtimePipelineSpec),
			WorkflowRuntimeManifest: model.LargeText(runtimeWorkflowSpec),
			TaskDetails:             modelTasks,
			PluginsInputString:      stringToLargeText(pluginsInputStr),
			PluginsOutputString:     stringToLargeText(pluginsOutputStr),
		},
		Tasks: modelTasks,
	}

	if err := validation.ValidateModel(&modelRun); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to convert API run to its internal representation",
		)
	}
	return &modelRun, nil
}

// Converts internal representation of a run to its API counterpart.
// Supports v2beta1 API.
// Note: adds error details to the message if a parsing error occurs.
func toApiRun(r *model.Run) *apiv2beta1.Run {
	return toApiRunWithPipelineSourcePreference(r, false)
}

// toApiRunWithPipelineSourcePreference converts a run to its API form.
// When preferEmbeddedPipelineSpec is true and the run stores a pipeline
// manifest, the response embeds pipeline_spec even if a pipeline version
// reference is also present. Runtime clients authenticate with run-scoped
// tokens and cannot call GetPipelineVersion.
//
//nolint:staticcheck // ST1003: matches existing toApi* naming in this package
func toApiRunWithPipelineSourcePreference(r *model.Run, preferEmbeddedPipelineSpec bool) *apiv2beta1.Run {
	r = r.ToV2()
	runtimeConfig := toApiRuntimeConfig(r.PipelineSpec.RuntimeConfig)
	var apiRunErr error
	if runtimeConfig == nil {
		apiRunErr = util.Wrap(errors.New("Failed to parse runtime config"), "Failed to convert internal run representation to its API counterpart")
	}
	if runtimeConfig != nil && len(runtimeConfig.GetParameters()) == 0 && len(runtimeConfig.GetPipelineRoot()) == 0 {
		if params := toMapProtoStructParameters(string(r.Parameters)); len(params) > 0 {
			runtimeConfig.Parameters = params
		} else {
			runtimeConfig = nil
		}
	}
	apiTasks, err := generateAPITasks(r.Tasks)
	if err != nil {
		if apiRunErr == nil {
			apiRunErr = err
		}
		apiTasks = nil
	}
	if len(apiTasks) == 0 {
		apiTasks = nil
	}

	apiRd := &apiv2beta1.RunDetails{
		PipelineContextId:    r.RunDetails.PipelineContextId,
		PipelineRunContextId: r.RunDetails.PipelineRunContextId,
	}
	if apiRd.PipelineContextId == 0 && apiRd.PipelineRunContextId == 0 {
		apiRd = nil
	}
	// Populate task count from either the TaskCount field or the length of Tasks
	taskCount := int32(r.TaskCount)
	if taskCount == 0 && len(apiTasks) > 0 {
		// If TaskCount wasn't populated but we have tasks, use the task slice length
		taskCount = int32(len(apiTasks))
	}

	apiRunV2 := &apiv2beta1.Run{
		RunId:          r.UUID,
		ExperimentId:   r.ExperimentId,
		RecurringRunId: r.RecurringRunId,
		DisplayName:    r.DisplayName,
		Description:    r.Description,
		ServiceAccount: r.ServiceAccount,
		RuntimeConfig:  runtimeConfig,
		StorageState:   toApiRunStorageState(&r.StorageState),
		State:          toApiRuntimeState(&r.RunDetails.State),
		StateHistory:   toApiRuntimeStatuses(r.RunDetails.StateHistory),
		CreatedAt:      timestamppb.New(time.Unix(r.CreatedAtInSec, 0)),
		ScheduledAt:    timestamppb.New(time.Unix(r.ScheduledAtInSec, 0)),
		FinishedAt:     timestamppb.New(time.Unix(r.FinishedAtInSec, 0)),
		RunDetails:     apiRd,
		TaskCount:      taskCount,
		Tasks:          apiTasks,
	}
	apiRunV2.PluginsInput, err = jsonToPluginsInput(largeTextToString(r.PluginsInputString))
	if err != nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(util.Wrap(err, "Failed to convert internal run representation to its API counterpart: invalid plugins_input")),
		}
	}
	apiRunV2.PluginsOutput, err = jsonToPluginsOutput(largeTextToString(r.PluginsOutputString))
	if err != nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(util.Wrap(err, "Failed to convert internal run representation to its API counterpart: invalid plugins_output")),
		}
	}
	pluginLimitsConfig, err := common.GetPluginLimitsConfig()
	if err != nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(util.Wrap(err, "Failed to convert internal run representation to its API counterpart: invalid plugins_output")),
		}
	}
	if err = validatePluginsOutputWithLimits(apiRunV2.PluginsOutput, pluginLimitsConfig); err != nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(util.Wrap(err, "Failed to convert internal run representation to its API counterpart: invalid plugins_output")),
		}
	}

	pipelineSourceErr := util.NewInvalidInputError("Failed to parse the pipeline source")
	if preferEmbeddedPipelineSpec && r.PipelineSpecManifest != "" {
		spec, err1 := YamlStringToPipelineSpecStruct(string(r.PipelineSpecManifest))
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
		} else if apiRunErr == nil {
			pipelineSourceErr = util.Wrap(err1, pipelineSourceErr.Error()).(*util.UserError)
		}
	}
	switch {
	case apiRunV2.PipelineSource == nil && r.PipelineVersionId != "":
		apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId:        r.PipelineId,
				PipelineVersionId: r.PipelineVersionId,
			},
		}
	case apiRunV2.PipelineSource == nil && r.PipelineSpecManifest != "":
		spec, err1 := YamlStringToPipelineSpecStruct(string(r.PipelineSpecManifest))
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
		} else if apiRunErr == nil {
			pipelineSourceErr = util.Wrap(err1, pipelineSourceErr.Error()).(*util.UserError)
		}
	case apiRunV2.PipelineSource == nil && r.WorkflowSpecManifest != "":
		spec, err1 := YamlStringToPipelineSpecStruct(string(r.WorkflowSpecManifest))
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
		} else if apiRunErr == nil {
			pipelineSourceErr = util.Wrap(err1, pipelineSourceErr.Error()).(*util.UserError)
		}
	}

	if apiRunV2.GetPipelineSource() == nil && apiRunErr == nil {
		apiRunErr = util.Wrap(pipelineSourceErr, "Failed to convert internal run representation to its API counterpart due to missing pipeline source")
	}

	if apiRunErr != nil {
		apiRunV2.Error = util.ToRpcStatus(apiRunErr)
	}
	return apiRunV2
}

func generateAPITasks(tasks []*model.Task) ([]*apiv2beta1.PipelineTask, error) {
	// Create map to store parent->children relationships
	childrenMap := make(map[string][]*model.Task)

	// Build maps of tasks and parent->children relationships
	for _, task := range tasks {
		if task.ParentTaskUUID != nil {
			childrenMap[*task.ParentTaskUUID] = append(childrenMap[*task.ParentTaskUUID], task)
		}
	}

	// Convert each task to API format, building child task info if it has children
	apiTasks := make([]*apiv2beta1.PipelineTask, 0)
	for _, task := range tasks {
		childTasks := childrenMap[task.UUID]

		apiTask, err := toAPITask(task, childTasks)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert task to API format")
		}
		apiTasks = append(apiTasks, apiTask)
	}

	return apiTasks, nil
}

// Converts an array of internal pipeline version representations to an array of API pipeline versions.
// Supports v2beta1 API.
func toApiRuns(runs []*model.Run) []*apiv2beta1.Run {
	apiRuns := make([]*apiv2beta1.Run, 0)
	for _, run := range runs {
		apiRuns = append(apiRuns, toApiRun(run))
	}
	return apiRuns
}

// Converts API recurring run to its internal representation.
func toModelJob(apiJob *apiv2beta1.RecurringRun) (*model.Job, error) {
	var jobId, jobName, k8sName, namespace, serviceAcc, desc, experimentId, pipelineName string
	var pipelineId, pipelineVersionId, pipelineSpec, workflowSpec, specParams, cfgParams, pipelineRoot string
	var maxConcur, createTime, updateTime int64
	var noCatchup, isEnabled bool
	var trigger *model.Trigger
	var jobPluginsInputStr *string
	pipelineId = apiJob.GetPipelineVersionReference().GetPipelineId()
	pipelineVersionId = apiJob.GetPipelineVersionReference().GetPipelineVersionId()

	if apiJob.GetPipelineSpec() != nil {
		if spec, err := pipelineSpecStructToYamlString(apiJob.GetPipelineSpec()); err == nil {
			pipelineSpec = spec
		} else {
			return nil, util.Wrap(err, "Failed to convert API recurring run to its internal representation due to pipeline spec conversion error")
		}
	}

	cfg, err := toModelRuntimeConfig(apiJob.GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert API recurring run to its internal representation due to runtime config conversion error")
	}
	cfgParams = string(cfg.Parameters)
	pipelineRoot = string(cfg.PipelineRoot)

	jobName = apiJob.GetDisplayName()
	if jobName == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Recurring run's name cannot be empty"), "Failed to convert a API recurring run to its internal representation")
	}

	trigger = toModelTrigger(apiJob.GetTrigger())
	isEnabled, err = toModelJobEnabled(apiJob.GetMode())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert a API recurring run to its internal representation due to parsing error occurred in its mode field")
	}

	jobId = apiJob.GetRecurringRunId()
	desc = apiJob.GetDescription()
	namespace = apiJob.GetNamespace()
	experimentId = apiJob.GetExperimentId()
	serviceAcc = apiJob.GetServiceAccount()
	noCatchup = apiJob.GetNoCatchup()
	maxConcur = apiJob.GetMaxConcurrency()
	createTime = apiJob.GetCreatedAt().GetSeconds()
	updateTime = apiJob.GetUpdatedAt().GetSeconds()

	k8sName = jobName
	specParams = ""
	workflowSpec = ""

	jobPluginsInputStr, err = pluginsInputToJSON(apiJob.GetPluginsInput())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert plugins_input to JSON")
	}
	pluginLimitsConfig, err := common.GetPluginLimitsConfig()
	if err != nil {
		return nil, util.NewInvalidInputError("Invalid plugins limits configuration: %v", err)
	}
	if err = validatePluginsInputLimits(apiJob.GetPluginsInput(), pluginLimitsConfig); err != nil {
		return nil, util.NewInvalidInputError("Invalid plugins_input: %v", err)
	}
	if maxConcur > 10 || maxConcur < 1 {
		return nil, util.NewInvalidInputError("Max concurrency of a recurring run must be at least 1 and at most 10. Received %v", maxConcur)
	}
	if trigger != nil && trigger.CronSchedule.Cron != nil {
		if _, err := cron.Parse(*trigger.CronSchedule.Cron); err != nil {
			return nil, util.NewInvalidInputError(
				"Schedule cron is not a supported format(https://godoc.org/github.com/robfig/cron). Error: %v", err)
		}
	}
	if trigger != nil && trigger.PeriodicSchedule.IntervalSecond != nil {
		if *trigger.PeriodicSchedule.IntervalSecond < 1 {
			return nil, util.NewInvalidInputError(
				"Found invalid period schedule interval %v. Set at interval to least 1 second", *trigger.PeriodicSchedule.IntervalSecond)
		}
	}
	if namespace != "" && pipelineVersionId != "" {
		pipelineName = fmt.Sprintf("namespaces/%v/pipelines/%v", namespace, pipelineVersionId)
	} else if pipelineVersionId != "" {
		pipelineName = fmt.Sprintf("pipelines/%v", pipelineVersionId)
	}

	status := model.StatusStateUnspecified
	if isEnabled {
		status = model.StatusStateEnabled
	} else {
		status = model.StatusStateDisabled
	}
	return &model.Job{
		UUID:               jobId,
		DisplayName:        jobName,
		K8SName:            k8sName,
		Namespace:          namespace,
		ServiceAccount:     serviceAcc,
		Description:        desc,
		MaxConcurrency:     maxConcur,
		NoCatchup:          noCatchup,
		CreatedAtInSec:     createTime,
		UpdatedAtInSec:     updateTime,
		Enabled:            isEnabled,
		Conditions:         status.ToString(),
		ExperimentId:       experimentId,
		PluginsInputString: stringToLargeText(jobPluginsInputStr),
		Trigger:            *trigger,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipelineId,
			PipelineName:         pipelineName,
			PipelineVersionId:    pipelineVersionId,
			PipelineSpecManifest: model.LargeText(pipelineSpec),
			WorkflowSpecManifest: model.LargeText(workflowSpec),
			Parameters:           model.LargeText(specParams),
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   model.LargeText(cfgParams),
				PipelineRoot: model.LargeText(pipelineRoot),
			},
		},
	}, nil
}

// Converts API recurring run's mode to its internal representation.
func toModelJobEnabled(mode apiv2beta1.RecurringRun_Mode) (bool, error) {
	switch mode {
	case apiv2beta1.RecurringRun_ENABLE:
		return true, nil
	case apiv2beta1.RecurringRun_MODE_UNSPECIFIED, apiv2beta1.RecurringRun_DISABLE:
		return false, nil
	default:
		return false, util.NewInternalServerError(util.NewInvalidInputError("Recurring run's mode is invalid: %v", mode), "Failed to convert API recurring run's mode to its internal representation")
	}
}

// Converts internal recurring run's status to API counterpart.
// Supports v2beta1 API.
// Note: returns STATUS_UNSPECIFIED by default.
// The mapping from Argo to v2beta:
// Enabled, Running, Succeeded -> ENABLED
// Disabled -> DISABLED
// Error -> STATUS_UNSPECIFIED.
func toApiRecurringRunStatus(s string) apiv2beta1.RecurringRun_Status {
	switch s {
	case string(model.StatusStateEnabled), string(swapi.ScheduledWorkflowSucceeded), string(swapi.ScheduledWorkflowRunning), string(swapi.ScheduledWorkflowEnabled):
		return apiv2beta1.RecurringRun_ENABLED
	case string(model.StatusStateDisabled), string(swapi.ScheduledWorkflowDisabled):
		return apiv2beta1.RecurringRun_DISABLED
	case string(model.StatusStateUnspecified), string(model.StatusStateUnspecifiedV1), string(swapi.ScheduledWorkflowError):
		return apiv2beta1.RecurringRun_STATUS_UNSPECIFIED
	default:
		return apiv2beta1.RecurringRun_STATUS_UNSPECIFIED
	}
}

// Converts recurring run's internal representation to its API counterpart.
// Supports v2beta1 API.
func toApiRecurringRun(j *model.Job) *apiv2beta1.RecurringRun {
	j = j.ToV2()
	runtimeConfig := toApiRuntimeConfig(j.PipelineSpec.RuntimeConfig)
	if runtimeConfig == nil {
		return &apiv2beta1.RecurringRun{
			RecurringRunId: j.UUID,
			Error:          util.ToRpcStatus(util.NewInternalServerError(util.NewInvalidInputError("Runtime config was not parsed correctly"), "Failed to convert recurring run's internal representation to its API counterpart")),
		}
	}
	if runtimeConfig == nil || (len(runtimeConfig.GetParameters()) == 0 && len(runtimeConfig.GetPipelineRoot()) == 0) {
		if params := toMapProtoStructParameters(string(j.Parameters)); len(params) > 0 {
			runtimeConfig.Parameters = params
		} else {
			runtimeConfig = nil
		}
	}

	apiRecurringRunV2 := &apiv2beta1.RecurringRun{
		RecurringRunId: j.UUID,
		DisplayName:    j.DisplayName,
		ServiceAccount: j.ServiceAccount,
		Description:    j.Description,
		Status:         toApiRecurringRunStatus(j.Conditions),
		CreatedAt:      timestamppb.New(time.Unix(j.CreatedAtInSec, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(j.UpdatedAtInSec, 0)),
		MaxConcurrency: j.MaxConcurrency,
		NoCatchup:      j.NoCatchup,
		Trigger:        toApiTrigger(&j.Trigger),
		RuntimeConfig:  runtimeConfig,
		Namespace:      j.Namespace,
		ExperimentId:   j.ExperimentId,
	}
	var err error
	apiRecurringRunV2.PluginsInput, err = jsonToPluginsInput(largeTextToString(j.PluginsInputString))
	if err != nil {
		return &apiv2beta1.RecurringRun{
			RecurringRunId: j.UUID,
			Error:          util.ToRpcStatus(util.Wrap(err, "Failed to convert recurring run's internal representation to its API counterpart: invalid plugins_input")),
		}
	}

	if j.PipelineId == "" && j.PipelineVersionId == "" {
		spec, err := YamlStringToPipelineSpecStruct(string(j.PipelineSpecManifest))
		if err != nil {
			return &apiv2beta1.RecurringRun{
				RecurringRunId: j.UUID,
				Error:          util.ToRpcStatus(util.Wrap(err, "Failed to convert recurring run's internal representation to its API counterpart")),
			}
		}
		if len(spec.GetFields()) > 0 {
			apiRecurringRunV2.PipelineSource = &apiv2beta1.RecurringRun_PipelineSpec{
				PipelineSpec: spec,
			}
		}
	} else {
		apiRecurringRunV2.PipelineSource = &apiv2beta1.RecurringRun_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId:        j.PipelineSpec.PipelineId,
				PipelineVersionId: j.PipelineSpec.PipelineVersionId,
			},
		}
	}
	if j.Enabled {
		apiRecurringRunV2.Status = apiv2beta1.RecurringRun_ENABLED
		// TODO(gkcalat): consider removing this as Mode is input
		apiRecurringRunV2.Mode = apiv2beta1.RecurringRun_ENABLE
	} else {
		apiRecurringRunV2.Status = apiv2beta1.RecurringRun_DISABLED
		// TODO(gkcalat): consider removing this as Mode is input
		apiRecurringRunV2.Mode = apiv2beta1.RecurringRun_DISABLE
	}
	return apiRecurringRunV2
}

// Converts an array of recurring run internal representations to an array of their API counterparts.
// Supports v2beta1 API.
func toApiRecurringRuns(jobs []*model.Job) []*apiv2beta1.RecurringRun {
	apiRecurringRuns := make([]*apiv2beta1.RecurringRun, 0)
	for _, job := range jobs {
		apiRecurringRuns = append(apiRecurringRuns, toApiRecurringRun(job))
	}
	return apiRecurringRuns
}

// Converts API storage state to its internal representation.
func toModelStorageState(state apiv2beta1.Run_StorageState) (model.StorageState, error) {
	switch state {
	case apiv2beta1.Run_ARCHIVED:
		return model.StorageStateArchived, nil
	case apiv2beta1.Run_AVAILABLE:
		return model.StorageStateAvailable, nil
	case apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED:
		return model.StorageStateUnspecified, nil
	default:
		return "", util.NewInternalServerError(util.NewInvalidInputError("Storage state cannot be equal to %v", state), "Failed to convert API storage state to its internal representation")
	}
}

// Converts internal storage state representation to its API run's counterpart.
// Support v2beta1 API.
func toApiRunStorageState(s *model.StorageState) apiv2beta1.Run_StorageState {
	if string(*s) == "" {
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchivedV1):
		return apiv2beta1.Run_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailableV1):
		return apiv2beta1.Run_AVAILABLE
	case string(model.StorageStateUnspecified), string(model.StorageStateUnspecifiedV1):
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	default:
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	}
}

// Converts internal storage state representation to its API experiment's counterpart.
// Support v2beta1 API.
func toApiExperimentStorageState(s *model.StorageState) apiv2beta1.Experiment_StorageState {
	if string(*s) == "" {
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchivedV1):
		return apiv2beta1.Experiment_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailableV1):
		return apiv2beta1.Experiment_AVAILABLE
	case string(model.StorageStateUnspecified), string(model.StorageStateUnspecifiedV1):
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	default:
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	}
}

// Converts API runtime state to its internal representation.
func toModelRuntimeState(state apiv2beta1.RuntimeState) model.RuntimeState {
	return model.RuntimeState(apiv2beta1.RuntimeState_name[int32(state)]).ToV2()
}

// Converts internal runtime state representation to its API counterpart.
// Support v2beta1 API.
func toApiRuntimeState(s *model.RuntimeState) apiv2beta1.RuntimeState {
	return apiv2beta1.RuntimeState(apiv2beta1.RuntimeState_value[s.ToString()])
}

// Converts API runtime status to its internal representation.
// Supports v2beta1 API.
func toModelRuntimeStatus(s *apiv2beta1.RuntimeStatus) (*model.RuntimeStatus, error) {
	if s == nil {
		return &model.RuntimeStatus{}, nil
	}
	modelStatus := &model.RuntimeStatus{
		UpdateTimeInSec: s.GetUpdateTime().GetSeconds(),
		State:           toModelRuntimeState(s.GetState()),
	}
	if s.GetError() != nil {
		modelStatus.Error = util.ToError(s.GetError())
	}
	return modelStatus, nil
}

// Converts an array of API runtime statuses to an array of their internal representations.
// Support v2beta1 API.
func toModelRuntimeStatuses(s []*apiv2beta1.RuntimeStatus) ([]*model.RuntimeStatus, error) {
	statuses := make([]*model.RuntimeStatus, 0)
	if s == nil {
		return statuses, nil
	}
	for _, status := range s {
		modelStatus, err := toModelRuntimeStatus(status)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert an array of API runtime statuses to an array of their internal representations")
		}
		statuses = append(statuses, modelStatus)
	}
	return statuses, nil
}

// Converts internal representation of a runtime status to its API counterpart.
// Supports v2beta1 API.
func toApiRuntimeStatus(s *model.RuntimeStatus) *apiv2beta1.RuntimeStatus {
	if s == nil {
		return nil
	}
	apiStatus := &apiv2beta1.RuntimeStatus{
		State: toApiRuntimeState(&s.State),
	}
	if s.UpdateTimeInSec > 0 {
		apiStatus.UpdateTime = &timestamppb.Timestamp{Seconds: s.UpdateTimeInSec}
	}
	if s.Error != nil {
		apiStatus.Error = util.ToRpcStatus(s.Error)
	}
	return apiStatus
}

// Converts an array of API runtime statuses to an array of their internal representations.
// Support v2beta1 API.
func toApiRuntimeStatuses(s []*model.RuntimeStatus) []*apiv2beta1.RuntimeStatus {
	if len(s) == 0 {
		return nil
	}
	statuses := make([]*apiv2beta1.RuntimeStatus, 0)
	for _, status := range s {
		statuses = append(statuses, toApiRuntimeStatus(status))
	}
	return statuses
}

func largeTextToString(lt *model.LargeText) *string {
	if lt == nil {
		return nil
	}
	s := string(*lt)
	return &s
}

func stringToLargeText(s *string) *model.LargeText {
	if s == nil || *s == "" {
		return nil
	}
	lt := model.LargeText(*s)
	return &lt
}

func pluginsInputToJSON(pluginsInput map[string]*structpb.Struct) (*string, error) {
	if len(pluginsInput) == 0 {
		return nil, nil
	}
	raw := make(map[string]json.RawMessage, len(pluginsInput))
	for k, v := range pluginsInput {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal plugins_input[%q]: %w", k, err)
		}
		raw[k] = b
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal plugins_input map: %w", err)
	}
	s := string(out)
	return &s, nil
}

func jsonToPluginsInput(jsonStr *string) (map[string]*structpb.Struct, error) {
	if jsonStr == nil || *jsonStr == "" {
		return nil, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("unmarshal plugins_input: %w", err)
	}
	result := make(map[string]*structpb.Struct, len(raw))
	for k, v := range raw {
		st := &structpb.Struct{}
		if err := protojson.Unmarshal(v, st); err != nil {
			return nil, fmt.Errorf("unmarshal plugins_input[%q]: %w", k, err)
		}
		result[k] = st
	}
	return result, nil
}

func pluginsOutputToJSON(pluginsOutput map[string]*apiv2beta1.PluginOutput) (*string, error) {
	if len(pluginsOutput) == 0 {
		return nil, nil
	}
	raw := make(map[string]json.RawMessage, len(pluginsOutput))
	for k, v := range pluginsOutput {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal plugins_output[%q]: %w", k, err)
		}
		raw[k] = b
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal plugins_output map: %w", err)
	}
	s := string(out)
	return &s, nil
}

func validatePluginsOutput(pluginsOutput map[string]*apiv2beta1.PluginOutput) error {
	limits, err := common.GetPluginLimitsConfig()
	if err != nil {
		return fmt.Errorf("%s: %w", pluginErrInvalidLimitsConfig, err)
	}
	return validatePluginsOutputWithLimits(pluginsOutput, limits)
}

func validatePluginsOutputWithLimits(pluginsOutput map[string]*apiv2beta1.PluginOutput, limits common.PluginLimitsConfig) error {
	if err := validatePluginsOutputLimits(pluginsOutput, limits); err != nil {
		return err
	}
	for pluginKey, output := range pluginsOutput {
		if output == nil {
			continue
		}
		if err := validatePluginOutputEntries(pluginKey, output.Entries); err != nil {
			return err
		}
	}
	return nil
}

func validatePluginsInputLimits(pluginsInput map[string]*structpb.Struct, limits common.PluginLimitsConfig) error {
	if len(pluginsInput) > limits.MaxKeys {
		return fmt.Errorf("%s (%d) exceeds maximum %d", pluginErrPluginsInputTooManyKeys, len(pluginsInput), limits.MaxKeys)
	}
	raw := make(map[string]json.RawMessage, len(pluginsInput))
	for pluginKey, pluginStruct := range pluginsInput {
		if pluginStruct == nil {
			return fmt.Errorf(pluginErrPluginsInputNilEntry, pluginKey)
		}
		depth, err := structDepth(pluginStruct)
		if err != nil {
			return fmt.Errorf(pluginErrPluginsInputInvalidValue+": %w", pluginKey, err)
		}
		if depth > limits.MaxNestingDepth {
			return fmt.Errorf(pluginErrPluginsInputNestingDepth+" %d", pluginKey, limits.MaxNestingDepth)
		}
		pluginBytes, err := protojson.Marshal(pluginStruct)
		if err != nil {
			return fmt.Errorf(pluginErrPluginsInputMarshalEntry, pluginKey, err)
		}
		if len(pluginBytes) > limits.MaxPayloadBytes {
			return fmt.Errorf(pluginErrPluginsInputEntrySize+pluginErrExceedsMaxBytes, pluginKey, len(pluginBytes), limits.MaxPayloadBytes)
		}
		raw[pluginKey] = pluginBytes
	}
	serialized, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf(pluginErrPluginsInputMarshalMap, err)
	}
	if len(serialized) > limits.MaxTotalPayloadBytes {
		return fmt.Errorf(pluginErrPluginsInputTotalSize+pluginErrExceedsMaxBytes, len(serialized), limits.MaxTotalPayloadBytes)
	}
	return nil
}

func validatePluginsOutputLimits(pluginsOutput map[string]*apiv2beta1.PluginOutput, limits common.PluginLimitsConfig) error {
	if len(pluginsOutput) > limits.MaxKeys {
		return fmt.Errorf("%s (%d) exceeds maximum %d", pluginErrPluginsOutputTooManyKeys, len(pluginsOutput), limits.MaxKeys)
	}
	raw := make(map[string]json.RawMessage, len(pluginsOutput))
	for pluginKey, output := range pluginsOutput {
		if err := validateSinglePluginOutputLimit(pluginKey, output, limits); err != nil {
			return err
		}
		if output == nil {
			continue
		}
		pluginBytes, err := protojson.Marshal(output)
		if err != nil {
			return fmt.Errorf(pluginErrPluginsOutputMarshalEntry, pluginKey, err)
		}
		if len(pluginBytes) > limits.MaxPayloadBytes {
			return fmt.Errorf(
				pluginErrPluginsOutputEntrySize+pluginErrExceedsMaxBytes,
				pluginKey,
				len(pluginBytes),
				limits.MaxPayloadBytes,
			)
		}
		raw[pluginKey] = pluginBytes
	}
	serialized, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf(pluginErrPluginsOutputMarshalMap, err)
	}
	if len(serialized) > limits.MaxTotalPayloadBytes {
		return fmt.Errorf(pluginErrPluginsOutputTotalSize+pluginErrExceedsMaxBytes, len(serialized), limits.MaxTotalPayloadBytes)
	}
	return nil
}

func validateSinglePluginOutputLimit(
	pluginKey string,
	output *apiv2beta1.PluginOutput,
	limits common.PluginLimitsConfig,
) error {
	if output == nil {
		return nil
	}
	for entryKey, metadata := range output.Entries {
		if metadata == nil {
			return fmt.Errorf(pluginErrPluginsOutputNilMetadata, pluginKey, entryKey)
		}
		if metadata.Value == nil {
			return fmt.Errorf(pluginErrPluginsOutputNilValue, pluginKey, entryKey)
		}
		depth, err := valueDepth(metadata.Value)
		if err != nil {
			return fmt.Errorf(pluginErrPluginsOutputInvalidValue+": %w", pluginKey, entryKey, err)
		}
		if depth > limits.MaxNestingDepth {
			return fmt.Errorf(pluginErrPluginsOutputNestingDepth+" %d", pluginKey, entryKey, limits.MaxNestingDepth)
		}
	}
	return nil
}

func structDepth(s *structpb.Struct) (int, error) {
	if s == nil {
		return 0, errors.New(pluginErrStructValueNil)
	}
	maxDepth := 1
	for fieldKey, fieldValue := range s.Fields {
		if fieldValue == nil {
			return 0, fmt.Errorf(pluginErrStructFieldNil, fieldKey)
		}
		fieldDepth, err := valueDepth(fieldValue)
		if err != nil {
			return 0, err
		}
		currentDepth := 1 + fieldDepth
		if currentDepth > maxDepth {
			maxDepth = currentDepth
		}
	}
	return maxDepth, nil
}

func valueDepth(v *structpb.Value) (int, error) {
	if v == nil {
		return 0, errors.New(pluginErrValueNil)
	}
	switch kind := v.Kind.(type) {
	case *structpb.Value_StructValue:
		return structDepth(kind.StructValue)
	case *structpb.Value_ListValue:
		maxDepth := 1
		for _, item := range kind.ListValue.Values {
			itemDepth, err := valueDepth(item)
			if err != nil {
				return 0, err
			}
			currentDepth := 1 + itemDepth
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		}
		return maxDepth, nil
	case *structpb.Value_NullValue, *structpb.Value_NumberValue, *structpb.Value_StringValue, *structpb.Value_BoolValue:
		return 0, nil
	default:
		return 0, errors.New(pluginErrValueKindUnset)
	}
}

func validatePluginOutputEntries(pluginKey string, entries map[string]*apiv2beta1.MetadataValue) error {
	for entryKey, metadata := range entries {
		if metadata == nil || metadata.Value == nil {
			continue
		}
		if metadata.GetRenderType() != apiv2beta1.MetadataValue_URL {
			continue
		}
		if err := validateURLMetadataValue(pluginKey, entryKey, metadata); err != nil {
			return err
		}
	}
	return nil
}

func validateURLMetadataValue(pluginKey string, entryKey string, metadata *apiv2beta1.MetadataValue) error {
	urlValue, err := getURLMetadataString(pluginKey, entryKey, metadata)
	if err != nil {
		return err
	}
	lowerTrimmed := strings.ToLower(urlValue)
	if hasDisallowedURLSchemePrefix(lowerTrimmed) {
		return fmt.Errorf("plugins_output[%q].entries[%q] has disallowed URL scheme", pluginKey, entryKey)
	}
	parsed, err := url.Parse(urlValue)
	if err != nil {
		return fmt.Errorf("plugins_output[%q].entries[%q] has invalid URL: %w", pluginKey, entryKey, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("plugins_output[%q].entries[%q] has invalid URL: missing scheme or host", pluginKey, entryKey)
	}
	if !isAllowedURLScheme(parsed.Scheme) {
		return fmt.Errorf("plugins_output[%q].entries[%q] URL scheme must be http or https", pluginKey, entryKey)
	}
	return nil
}

func getURLMetadataString(pluginKey string, entryKey string, metadata *apiv2beta1.MetadataValue) (string, error) {
	stringValue, isStringValue := metadata.Value.Kind.(*structpb.Value_StringValue)
	if !isStringValue {
		return "", fmt.Errorf("plugins_output[%q].entries[%q] URL render_type requires string value", pluginKey, entryKey)
	}
	return strings.TrimSpace(stringValue.StringValue), nil
}

func hasDisallowedURLSchemePrefix(urlValueLower string) bool {
	for _, disallowedScheme := range []string{
		urlSchemeJavaScript,
		urlSchemeData,
		urlSchemeVBScript,
	} {
		if strings.HasPrefix(urlValueLower, disallowedScheme) {
			return true
		}
	}
	return false
}

func isAllowedURLScheme(urlScheme string) bool {
	lowerScheme := strings.ToLower(urlScheme)
	return lowerScheme == "http" || lowerScheme == "https"
}

func jsonToPluginsOutput(jsonStr *string) (map[string]*apiv2beta1.PluginOutput, error) {
	if jsonStr == nil || *jsonStr == "" {
		return nil, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("unmarshal plugins_output: %w", err)
	}
	result := make(map[string]*apiv2beta1.PluginOutput, len(raw))
	for k, v := range raw {
		po := &apiv2beta1.PluginOutput{}
		if err := protojson.Unmarshal(v, po); err != nil {
			return nil, fmt.Errorf("unmarshal plugins_output[%q]: %w", k, err)
		}
		result[k] = po
	}
	return result, nil
}

// Converts API v2beta1 artifact to its internal representation.
func toModelArtifact(a *apiv2beta1.Artifact) (*model.Artifact, error) {
	if a == nil {
		return nil, util.NewInvalidInputError("Artifact cannot be nil")
	}

	modelArtifact := &model.Artifact{
		UUID:        a.GetArtifactId(),
		Namespace:   a.GetNamespace(),
		Type:        model.ArtifactType(a.GetType()),
		URI:         a.Uri,
		Name:        a.GetName(),
		Description: a.GetDescription(),
		// NumberValue can be nil & nullable, so directly apply it
		// instead of using a.GetNumberValue() (which will return 0 if nil).
		NumberValue:     a.NumberValue,
		CreatedAtInSec:  time.Now().Unix(),
		LastUpdateInSec: time.Now().Unix(),
	}

	if a.GetMetadata() != nil {
		structValue := &structpb.Struct{Fields: a.GetMetadata()}
		jsonDataBytes, err := protojson.Marshal(structValue)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to marshal metadata to JSON")
		}
		var jsonData model.JSONData
		if err := json.Unmarshal(jsonDataBytes, &jsonData); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to unmarshal JSON into JSONData map")
		}
		modelArtifact.Metadata = jsonData
	}

	if err := validation.ValidateModel(modelArtifact); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert API artifact to internal representation")
	}
	return modelArtifact, nil
}

// Converts internal artifact representation to its API counterpart.
// Supports v2beta1 API.
func toAPIArtifact(artifact *model.Artifact) (*apiv2beta1.Artifact, error) {
	if artifact == nil {
		return nil, util.NewInvalidInputError("Artifact cannot be nil")
	}

	apiArtifact := &apiv2beta1.Artifact{
		ArtifactId:  artifact.UUID,
		Namespace:   artifact.Namespace,
		Type:        apiv2beta1.Artifact_ArtifactType(artifact.Type),
		Uri:         artifact.URI,
		Name:        artifact.Name,
		Description: artifact.Description,
		NumberValue: artifact.NumberValue,
		CreatedAt:   timestamppb.New(time.Unix(artifact.CreatedAtInSec, 0)),
	}

	if artifact.Metadata != nil {
		jsonDataBytes, err := json.Marshal(artifact.Metadata)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to marshal metadata to JSON")
		}
		var structValue structpb.Struct
		if err := protojson.Unmarshal(jsonDataBytes, &structValue); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to unmarshal JSON into structpb.Struct")
		}
		apiArtifact.Metadata = structValue.GetFields()
	}

	return apiArtifact, nil
}

// Converts an array of internal artifact representations to an array of their API counterparts.
// Supports v2beta1 API.
func toAPIArtifacts(artifacts []*model.Artifact) []*apiv2beta1.Artifact {
	apiArtifacts := make([]*apiv2beta1.Artifact, 0)
	for _, artifact := range artifacts {
		apiArtifact, err := toAPIArtifact(artifact)
		if err != nil {
			return nil
		}
		apiArtifacts = append(apiArtifacts, apiArtifact)
	}
	return apiArtifacts
}

// Converts internal artifact task representation to its API counterpart.
// Supports v2beta1 API.
func toAPIArtifactTask(artifactTask *model.ArtifactTask) *apiv2beta1.ArtifactTask {
	if artifactTask == nil {
		return &apiv2beta1.ArtifactTask{}
	}

	apiArtifactTask := &apiv2beta1.ArtifactTask{
		Id:         artifactTask.UUID,
		ArtifactId: artifactTask.ArtifactID,
		TaskId:     artifactTask.TaskID,
		Type:       apiv2beta1.IOType(artifactTask.Type),
		RunId:      artifactTask.RunUUID,
		Key:        artifactTask.ArtifactKey,
	}

	// Convert Producer from JSONData to IOProducer
	if artifactTask.Producer != nil {
		producer, err := model.JSONDataToProtoMessage(
			artifactTask.Producer,
			func() *apiv2beta1.IOProducer {
				return &apiv2beta1.IOProducer{}
			})
		if err == nil {
			apiArtifactTask.Producer = producer
		}
	}

	return apiArtifactTask
}

// Converts an array of internal artifact task representations to an array of their API counterparts.
// Supports v2beta1 API.
func toAPIArtifactTasks(artifactTasks []*model.ArtifactTask) []*apiv2beta1.ArtifactTask {
	apiArtifactTasks := make([]*apiv2beta1.ArtifactTask, 0)
	for _, artifactTask := range artifactTasks {
		apiArtifactTasks = append(apiArtifactTasks, toAPIArtifactTask(artifactTask))
	}
	return apiArtifactTasks
}

// Converts API v2beta1 ArtifactTask to its internal representation.
func toModelArtifactTask(apiAT *apiv2beta1.ArtifactTask) (*model.ArtifactTask, error) {
	if apiAT == nil {
		return nil, util.NewInvalidInputError("ArtifactTask cannot be nil")
	}

	if apiAT.GetType() == apiv2beta1.IOType_UNSPECIFIED {
		return nil, util.NewInvalidInputError("ArtifactTask's task id cannot be unspecified")
	}

	modelAT := &model.ArtifactTask{
		UUID:        apiAT.GetId(),
		RunUUID:     apiAT.GetRunId(),
		ArtifactID:  apiAT.GetArtifactId(),
		TaskID:      apiAT.GetTaskId(),
		Type:        model.IOType(apiAT.GetType()),
		ArtifactKey: apiAT.GetKey(),
	}

	// Convert Producer from IOProducer to JSONData
	if apiAT.GetProducer() != nil {
		producer, err := model.ProtoMessageToJSONData(apiAT.GetProducer())
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert producer to JSONData")
		}
		modelAT.Producer = producer
	}
	if err := modelAT.SyncIterationFromProducer(); err != nil {
		return nil, util.Wrap(err, "Failed to derive artifact-task iteration")
	}

	return modelAT, nil
}

// Converts API PipelineTask to its internal representation.
// Supports v2beta1 API.
// Note that InputArtifactsHydrated and OutputArtifactsHydrated are not converted.
// Those fields are transient hydration-only views, are not stored in DB, and
// callers must use the artifact APIs if they need to create or mutate artifact links.
func toModelTask(apiTask *apiv2beta1.PipelineTask) (*model.Task, error) {
	if apiTask == nil {
		return nil, util.NewInvalidInputError("Task cannot be nil")
	}

	task := &model.Task{
		UUID:           apiTask.GetTaskId(),
		RunUUID:        apiTask.GetRunId(),
		ParentTaskUUID: apiTask.ParentTaskId,
		Name:           apiTask.GetName(),
		DisplayName:    apiTask.GetDisplayName(),
		Fingerprint:    apiTask.GetCacheFingerprint(),
	}

	// Convert timestamps
	if apiTask.GetCreateTime() != nil {
		task.CreatedAtInSec = apiTask.GetCreateTime().GetSeconds()
	}
	if apiTask.GetEndTime() != nil {
		task.FinishedInSec = apiTask.GetEndTime().GetSeconds()
	}

	// Convert status
	task.State = model.TaskStatus(apiTask.GetState())

	// Convert task type
	task.Type = model.TaskType(apiTask.GetType())
	if apiTask.GetPods() != nil {
		pods, err := model.ProtoSliceToJSONSlice(apiTask.GetPods())
		if err != nil {
			return nil, err
		}
		task.Pods = pods
	}

	// Convert status metadata from new StatusMetadata struct
	if apiTask.GetStatusMetadata() != nil {
		sm, err := model.ProtoMessageToJSONData(apiTask.GetStatusMetadata())
		if err != nil {
			return nil, err
		}
		task.StatusMetadata = sm
	}

	// Convert state history using structured TaskStateHistoryEntry
	if len(apiTask.GetStateHistory()) > 0 {
		sh, err := model.ProtoSliceToJSONSlice(apiTask.GetStateHistory())
		if err != nil {
			return nil, err
		}
		task.StateHistory = sh
	}

	// Convert inputs: only parameter payloads are persisted here. Artifact links are
	// intentionally managed through artifact/artifact-task APIs.
	if apiTask.GetInputs() != nil {
		if apiTask.GetInputs().GetParameters() != nil {
			parameters, err := model.ProtoSliceToJSONSlice(apiTask.GetInputs().GetParameters())
			if err != nil {
				return nil, err
			}
			task.InputParameters = parameters
		}
	}

	// Convert outputs: only parameter payloads are persisted here. Artifact links are
	// intentionally managed through artifact/artifact-task APIs.
	if apiTask.GetOutputs() != nil {
		if apiTask.GetOutputs().GetParameters() != nil {
			parameters, err := model.ProtoSliceToJSONSlice(apiTask.GetOutputs().GetParameters())
			if err != nil {
				return nil, err
			}
			task.OutputParameters = parameters
		}
	}

	if apiTask.GetTypeAttributes() != nil {
		attrs, err := model.ProtoMessageToJSONData(apiTask.GetTypeAttributes())
		if err != nil {
			return nil, err
		}
		task.TypeAttrs = attrs
	}

	// Convert scope_path - validate it's not empty if provided
	if apiTask.GetScopePath() != "" {
		task.ScopePath = apiTask.GetScopePath()
	}

	return task, nil
}

// Converts internal task representation to its API counterpart.
// Supports v2beta1 API.
// Note that child tasks are not stored in the tasks table so
// they must be provided as an argument. Artifact payloads are exported only from
// InputArtifactsHydrated/OutputArtifactsHydrated, so callers that need
// Inputs.Artifacts or Outputs.Artifacts populated must hydrate artifact links first.
func toAPITask(modelTask *model.Task, childTasks []*model.Task) (*apiv2beta1.PipelineTask, error) {
	if modelTask == nil {
		return nil, util.NewInvalidInputError("Task cannot be nil")
	}

	apiTask := &apiv2beta1.PipelineTask{
		TaskId:           modelTask.UUID,
		RunId:            modelTask.RunUUID,
		ParentTaskId:     modelTask.ParentTaskUUID,
		Name:             modelTask.Name,
		DisplayName:      modelTask.DisplayName,
		CacheFingerprint: modelTask.Fingerprint,
		Inputs:           &apiv2beta1.PipelineTask_InputOutputs{},
		Outputs:          &apiv2beta1.PipelineTask_InputOutputs{},
	}

	// Convert timestamps
	if modelTask.CreatedAtInSec > 0 {
		apiTask.CreateTime = &timestamppb.Timestamp{Seconds: modelTask.CreatedAtInSec}
	}
	if modelTask.FinishedInSec > 0 {
		apiTask.EndTime = &timestamppb.Timestamp{Seconds: modelTask.FinishedInSec}
	}

	// Convert status
	apiTask.State = apiv2beta1.PipelineTask_TaskState(modelTask.State)

	// Convert task type
	apiTask.Type = apiv2beta1.PipelineTask_TaskType(modelTask.Type)

	// Set pod name from the first pod in PodNames array
	if modelTask.Pods != nil {
		apiPods, err := model.JSONSliceToProtoSlice(
			modelTask.Pods,
			func() *apiv2beta1.PipelineTask_TaskPod {
				return &apiv2beta1.PipelineTask_TaskPod{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.Pods = apiPods
	}

	// Convert status metadata to new StatusMetadata struct
	if modelTask.StatusMetadata != nil {
		statusMeta, err := model.JSONDataToProtoMessage(
			modelTask.StatusMetadata,
			func() *apiv2beta1.PipelineTask_StatusMetadata {
				return &apiv2beta1.PipelineTask_StatusMetadata{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.StatusMetadata = statusMeta
	}

	// Convert state history from JSONData back to RuntimeStatus slice using structured approach
	if modelTask.StateHistory != nil {
		apiSH, err := model.JSONSliceToProtoSlice(
			modelTask.StateHistory,
			func() *apiv2beta1.PipelineTask_TaskStatus {
				return &apiv2beta1.PipelineTask_TaskStatus{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.StateHistory = apiSH
	}

	// Convert InputParameters to API inputs field
	if modelTask.InputParameters != nil {
		apiInputParams, err := model.JSONSliceToProtoSlice(
			modelTask.InputParameters,
			func() *apiv2beta1.PipelineTask_InputOutputs_IOParameter {
				return &apiv2beta1.PipelineTask_InputOutputs_IOParameter{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.Inputs.Parameters = apiInputParams
	}

	// Convert OutputParameters to API outputs field
	if modelTask.OutputParameters != nil {
		apiOutputParams, err := model.JSONSliceToProtoSlice(
			modelTask.OutputParameters,
			func() *apiv2beta1.PipelineTask_InputOutputs_IOParameter {
				return &apiv2beta1.PipelineTask_InputOutputs_IOParameter{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.Outputs.Parameters = apiOutputParams
	}

	// Populate artifacts from hydrated fields on the model task with shared converter
	convertHydrated := func(in []model.TaskArtifactHydrated) ([]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact, error) {
		if len(in) == 0 {
			return nil, nil
		}

		// Group artifacts by (ArtifactKey, Type, Producer iteration for ITERATOR_OUTPUT).
		// For non-ITERATOR_OUTPUT types, all same-key artifacts are consolidated into one IOArtifact.
		// For ITERATOR_OUTPUT, each distinct iteration gets its own IOArtifact.
		type groupKey struct {
			artifactKey  string
			ioType       apiv2beta1.IOType
			producerTask string
			hasIteration bool
			iterationVal int64
		}

		makeKey := func(h model.TaskArtifactHydrated) groupKey {
			key := groupKey{
				artifactKey: h.Key,
				ioType:      h.Type,
			}
			if h.Producer != nil {
				key.producerTask = h.Producer.TaskName
				// Only split by iteration for ITERATOR_OUTPUT; ordinary outputs
				// consolidate all same-key artifacts into a single IOArtifact.
				if h.Type == apiv2beta1.IOType_ITERATOR_OUTPUT && h.Producer.Iteration != nil {
					key.hasIteration = true
					key.iterationVal = *h.Producer.Iteration
				}
			}
			return key
		}

		grouped := make(map[groupKey][]model.TaskArtifactHydrated)
		for _, h := range in {
			key := makeKey(h)
			grouped[key] = append(grouped[key], h)
		}

		keys := make([]groupKey, 0, len(grouped))
		for key := range grouped {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			left := keys[i]
			right := keys[j]
			if left.artifactKey != right.artifactKey {
				return left.artifactKey < right.artifactKey
			}
			if left.ioType != right.ioType {
				return left.ioType < right.ioType
			}
			if left.producerTask != right.producerTask {
				return left.producerTask < right.producerTask
			}
			if left.hasIteration != right.hasIteration {
				return !left.hasIteration && right.hasIteration
			}
			return left.iterationVal < right.iterationVal
		})

		// Convert grouped artifacts to IOArtifacts
		out := make([]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact, 0, len(grouped))
		for _, key := range keys {
			hydratedGroup := grouped[key]

			apiArtifacts := make([]*apiv2beta1.Artifact, 0, len(hydratedGroup))
			for _, h := range hydratedGroup {
				if h.Value != nil {
					apiArt, err := toAPIArtifact(h.Value)
					if err != nil {
						return nil, err
					}
					apiArtifacts = append(apiArtifacts, apiArt)
				}
			}

			firstHydrated := hydratedGroup[0]
			ioArtifact := &apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
				Artifacts:   apiArtifacts,
				ArtifactKey: firstHydrated.Key,
				Type:        firstHydrated.Type,
			}
			if firstHydrated.Producer != nil {
				ioArtifact.Producer = &apiv2beta1.IOProducer{
					TaskName:  firstHydrated.Producer.TaskName,
					Iteration: firstHydrated.Producer.Iteration,
				}
			}
			out = append(out, ioArtifact)
		}
		return out, nil
	}
	if arts, err := convertHydrated(modelTask.InputArtifactsHydrated); err != nil {
		return nil, err
	} else if len(arts) > 0 {
		apiTask.Inputs.Artifacts = arts
	}
	if arts, err := convertHydrated(modelTask.OutputArtifactsHydrated); err != nil {
		return nil, err
	} else if len(arts) > 0 {
		apiTask.Outputs.Artifacts = arts
	}

	// Extract additional fields from TypeAttrs
	if modelTask.TypeAttrs != nil {
		apiTypeAttrs, err := model.JSONDataToProtoMessage(
			modelTask.TypeAttrs,
			func() *apiv2beta1.PipelineTask_TypeAttributes {
				return &apiv2beta1.PipelineTask_TypeAttributes{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.TypeAttributes = apiTypeAttrs
	}

	// Convert child tasks
	apiChildTasks := make([]*apiv2beta1.PipelineTask_ChildTask, 0)
	for _, childTask := range childTasks {
		apiChildTask := &apiv2beta1.PipelineTask_ChildTask{
			TaskId: childTask.UUID,
			Name:   childTask.Name,
		}
		apiChildTasks = append(apiChildTasks, apiChildTask)
	}
	if len(apiChildTasks) > 0 {
		apiTask.ChildTasks = apiChildTasks
	}

	// Convert scope_path from model string to API string (both use dot notation)
	if modelTask.ScopePath != "" {
		apiTask.ScopePath = modelTask.ScopePath
	}

	return apiTask, nil
}
