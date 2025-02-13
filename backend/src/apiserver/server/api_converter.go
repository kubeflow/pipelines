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
	"sort"
	"strconv"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Converts API experiment to its internal representation.
// Supports both v1beta1 abd v2beta1 API.
func toModelExperiment(e interface{}) (*model.Experiment, error) {
	var namespace, name, description string
	switch apiExperiment := e.(type) {
	case *apiv1beta1.Experiment:
		name = apiExperiment.GetName()
		namespace = getNamespaceFromResourceReferenceV1(apiExperiment.GetResourceReferences())
		description = apiExperiment.GetDescription()
	case *apiv2beta1.Experiment:
		name = apiExperiment.GetDisplayName()
		namespace = apiExperiment.GetNamespace()
		description = apiExperiment.GetDescription()
	default:
		return nil, util.NewUnknownApiVersionError("Experiment", e)
	}
	if name == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Experiment must have a non-empty name"), "Failed to convert API experiment to model experiment")
	}
	return &model.Experiment{
		Name:         name,
		Description:  description,
		Namespace:    namespace,
		StorageState: model.StorageStateAvailable,
	}, nil
}

// Converts internal experiment representation to its API counterpart.
// Supports v1beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiExperimentV1(experiment *model.Experiment) *apiv1beta1.Experiment {
	if experiment == nil {
		return &apiv1beta1.Experiment{}
	}
	resourceReferences := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_NAMESPACE,
				Id:   experiment.Namespace,
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	var storageState apiv1beta1.Experiment_StorageState
	switch experiment.StorageState {
	case "AVAILABLE", "STORAGESTATE_AVAILABLE":
		storageState = apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"])
	case "ARCHIVED", "STORAGESTATE_ARCHIVED":
		storageState = apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"])
	default:
		storageState = apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_UNSPECIFIED"])
	}
	return &apiv1beta1.Experiment{
		Id:                 experiment.UUID,
		Name:               experiment.Name,
		Description:        experiment.Description,
		CreatedAt:          &timestamp.Timestamp{Seconds: experiment.CreatedAtInSec},
		ResourceReferences: resourceReferences,
		StorageState:       storageState,
	}
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
		CreatedAt:        &timestamp.Timestamp{Seconds: experiment.CreatedAtInSec},
		LastRunCreatedAt: &timestamp.Timestamp{Seconds: experiment.LastRunCreatedAtInSec},
		Namespace:        experiment.Namespace,
		StorageState:     storageState,
	}
}

// Converts an array of internal experiment representations to an array of API experiments.
// Supports v1beta1 API.
func toApiExperimentsV1(experiments []*model.Experiment) []*apiv1beta1.Experiment {
	apiExperiments := make([]*apiv1beta1.Experiment, 0)
	for _, experiment := range experiments {
		apiExperiments = append(apiExperiments, toApiExperimentV1(experiment))
	}
	return apiExperiments
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
// Supports both v1beta1 abd v2beta1 API.
func toModelPipeline(p interface{}) (*model.Pipeline, error) {
	var name, namespace, description string
	switch apiPipeline := p.(type) {
	case *apiv1beta1.Pipeline:
		namespace = getNamespaceFromResourceReferenceV1(apiPipeline.GetResourceReferences())
		name = apiPipeline.GetName()
		description = apiPipeline.GetDescription()
	case *apiv2beta1.Pipeline:
		namespace = apiPipeline.GetNamespace()
		name = apiPipeline.GetDisplayName()
		description = apiPipeline.GetDescription()
	default:
		return nil, util.NewUnknownApiVersionError("Pipeline", p)
	}
	return &model.Pipeline{
		Name:        name,
		Namespace:   namespace,
		Description: description,
		Status:      model.PipelineCreating,
	}, nil
}

// Converts internal pipeline and pipeline version representation to an API pipeline.
// Supports v1beta1 API.
// Note: stores details inside the message if a parsing error occurs.
func toApiPipelineV1(pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) *apiv1beta1.Pipeline {
	if pipeline == nil {
		return &apiv1beta1.Pipeline{
			Id: "",
			Error: util.NewInternalServerError(
				util.NewInvalidInputError("Pipeline cannot be nil"),
				"Failed to convert a model pipeline to v1beta1 API pipeline",
			).Error(),
		}
	}

	params := toApiParametersV1(pipelineVersion.Parameters)
	if params == nil {
		return &apiv1beta1.Pipeline{
			Id:    pipeline.UUID,
			Error: util.NewInternalServerError(util.NewInvalidInputError(fmt.Sprintf("Failed to convert parameters: %s", pipelineVersion.Parameters)), "Failed to convert a model pipeline to v1beta1 API pipeline").Error(),
		}
	}
	if len(params) == 0 {
		params = nil
	}

	defaultVersion := toApiPipelineVersionV1(pipelineVersion)
	var resourceRefs []*apiv1beta1.ResourceReference
	if pipeline.Namespace != "" {
		resourceRefs = []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   pipeline.Namespace,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		}
	}
	apiPipeline := &apiv1beta1.Pipeline{
		Id:                 pipeline.UUID,
		CreatedAt:          &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Name:               pipeline.Name,
		Description:        pipeline.Description,
		Parameters:         params,
		DefaultVersion:     defaultVersion,
		ResourceReferences: resourceRefs,
	}
	if defaultVersion.GetPackageUrl() != nil && defaultVersion.GetPackageUrl().GetPipelineUrl() != "" {
		apiPipeline.Url = defaultVersion.GetPackageUrl()
	} else if defaultVersion.GetCodeSourceUrl() != "" {
		apiPipeline.Url = &apiv1beta1.Url{PipelineUrl: defaultVersion.GetCodeSourceUrl()}
	}
	return apiPipeline
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
		DisplayName: pipeline.Name,
		Description: pipeline.Description,
		CreatedAt:   &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Namespace:   pipeline.Namespace,
	}
}

// Converts arrays of internal pipeline representations and pipeline version representations
// to an array of API pipelines.
// Supports v1beta1 API.
func toApiPipelinesV1(pipelines []*model.Pipeline, pipelineVersion []*model.PipelineVersion) []*apiv1beta1.Pipeline {
	apiPipelines := make([]*apiv1beta1.Pipeline, 0)
	for i, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, toApiPipelineV1(pipeline, pipelineVersion[i]))
	}
	return apiPipelines
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
// Supports both v1beta1 abd v2beta1 API.
// Note: supports v1beta1 API pipeline's conversion based on default pipeline version.
func toModelPipelineVersion(p interface{}) (*model.PipelineVersion, error) {
	var name, description, pipelineId, pipelineUrl, codeUrl string
	switch p := p.(type) {
	case *apiv1beta1.PipelineVersion:
		apiPipelineVersionV1 := p
		if apiPipelineVersionV1.GetPackageUrl() == nil || len(apiPipelineVersionV1.GetPackageUrl().GetPipelineUrl()) == 0 {
			return nil, util.NewInvalidInputError("Failed to convert v1beta1 API pipeline version to its internal representation due to missing pipeline URL")
		}
		pipelineUrl = apiPipelineVersionV1.GetPackageUrl().GetPipelineUrl()
		codeUrl = apiPipelineVersionV1.GetCodeSourceUrl()
		name = apiPipelineVersionV1.GetName()
		pipelineId = getPipelineIdFromResourceReferencesV1(apiPipelineVersionV1.GetResourceReferences())
		description = apiPipelineVersionV1.GetDescription()
	case *apiv2beta1.PipelineVersion:
		apiPipelineVersionV2 := p
		if apiPipelineVersionV2.GetPackageUrl() == nil || len(apiPipelineVersionV2.GetPackageUrl().GetPipelineUrl()) == 0 {
			return nil, util.NewInvalidInputError("Failed to convert v2beta1 API pipeline version to its internal representation due to missing pipeline URL")
		}
		name = apiPipelineVersionV2.GetDisplayName()
		pipelineId = apiPipelineVersionV2.GetPipelineId()
		pipelineUrl = apiPipelineVersionV2.GetPackageUrl().GetPipelineUrl()
		codeUrl = apiPipelineVersionV2.GetCodeSourceUrl()
		description = apiPipelineVersionV2.GetDescription()
	default:
		return nil, util.NewUnknownApiVersionError("PipelineVersion", p)
	}
	return &model.PipelineVersion{
		Name:            name,
		PipelineId:      pipelineId,
		PipelineSpecURI: pipelineUrl,
		CodeSourceUrl:   codeUrl,
		Description:     description,
		Status:          model.PipelineVersionCreating,
	}, nil
}

// Converts internal pipeline version representation to its API counterpart.
// Supports v1beta1 API.
// Note: does not return an error along with the result anymore. Check if the result is nil instead.
func toApiPipelineVersionV1(pv *model.PipelineVersion) *apiv1beta1.PipelineVersion {
	apiPipelineVersion := &apiv1beta1.PipelineVersion{}
	if pv == nil {
		return apiPipelineVersion
	}
	apiPipelineVersion.Id = pv.UUID
	apiPipelineVersion.Name = pv.Name
	apiPipelineVersion.CreatedAt = &timestamp.Timestamp{Seconds: pv.CreatedAtInSec}
	if p := toApiParametersV1(pv.Parameters); p == nil {
		return nil
	} else if len(p) > 0 {
		apiPipelineVersion.Parameters = p
	}
	apiPipelineVersion.Description = pv.Description
	if pv.CodeSourceUrl != "" {
		apiPipelineVersion.CodeSourceUrl = pv.CodeSourceUrl
		apiPipelineVersion.PackageUrl = &apiv1beta1.Url{PipelineUrl: pv.CodeSourceUrl}
	}
	if pv.PipelineId != "" {
		apiPipelineVersion.ResourceReferences = []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Id:   pv.PipelineId,
					Type: apiv1beta1.ResourceType_PIPELINE,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		}
	}
	return apiPipelineVersion
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
		DisplayName:       pv.Name,
		Description:       pv.Description,
		CreatedAt:         &timestamp.Timestamp{Seconds: pv.CreatedAtInSec},
	}

	// Infer pipeline url
	if pv.CodeSourceUrl != "" {
		apiPipelineVersion.PackageUrl = &apiv2beta1.Url{
			PipelineUrl: pv.PipelineSpecURI,
		}
	} else if pv.PipelineSpecURI != "" {
		apiPipelineVersion.PackageUrl = &apiv2beta1.Url{
			PipelineUrl: pv.PipelineSpecURI,
		}
	}

	// Convert pipeline spec
	spec, err := yamlStringToPipelineSpecStruct(pv.PipelineSpec)
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
// Supports v1beta1 API.
func toApiPipelineVersionsV1(pv []*model.PipelineVersion) []*apiv1beta1.PipelineVersion {
	apiVersions := make([]*apiv1beta1.PipelineVersion, 0)
	for _, version := range pv {
		v := toApiPipelineVersionV1(version)
		if v == nil {
			return nil
		}
		apiVersions = append(apiVersions, v)
	}
	return apiVersions
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

// Converts API resource type to its internal representation.
// Supports v1beta1 API.
func toModelResourceTypeV1(rt apiv1beta1.ResourceType) (model.ResourceType, error) {
	switch rt {
	case apiv1beta1.ResourceType_NAMESPACE:
		return model.NamespaceResourceType, nil
	case apiv1beta1.ResourceType_EXPERIMENT:
		return model.ExperimentResourceType, nil
	case apiv1beta1.ResourceType_PIPELINE:
		return model.PipelineResourceType, nil
	case apiv1beta1.ResourceType_PIPELINE_VERSION:
		return model.PipelineVersionResourceType, nil
	case apiv1beta1.ResourceType_JOB:
		return model.JobResourceType, nil
	default:
		return "", util.NewInvalidInputError("Failed to convert unsupported v1beta1 API resource type %s", apiv1beta1.ResourceType_name[int32(rt)])
	}
}

// Converts internal resource type representations to its API counterpart.
// Supports v1beta1 API.
func toApiResourceTypeV1(rt model.ResourceType) apiv1beta1.ResourceType {
	switch rt {
	case model.NamespaceResourceType:
		return apiv1beta1.ResourceType_NAMESPACE
	case model.ExperimentResourceType:
		return apiv1beta1.ResourceType_EXPERIMENT
	case model.PipelineResourceType:
		return apiv1beta1.ResourceType_PIPELINE
	case model.PipelineVersionResourceType:
		return apiv1beta1.ResourceType_PIPELINE_VERSION
	case model.JobResourceType:
		return apiv1beta1.ResourceType_JOB
	default:
		return apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE
	}
}

// Converts API resource relationship to its internal representation.
// Supports v1beta1 API.
func toModelRelationshipV1(r apiv1beta1.Relationship) (model.Relationship, error) {
	switch r {
	case apiv1beta1.Relationship_CREATOR:
		return model.CreatorRelationship, nil
	case apiv1beta1.Relationship_OWNER:
		return model.OwnerRelationship, nil
	default:
		return "", util.NewInvalidInputError("Failed to convert unsupported v1beta1 API resource relationship type: %s", apiv1beta1.Relationship_name[int32(r)])
	}
}

// Converts internal representation of a resource relationship to it API counterpart.
// Supports v1beta1 API.
func toApiRelationshipV1(r model.Relationship) apiv1beta1.Relationship {
	switch r {
	case model.CreatorRelationship:
		return apiv1beta1.Relationship_CREATOR
	case model.OwnerRelationship:
		return apiv1beta1.Relationship_OWNER
	default:
		return apiv1beta1.Relationship_UNKNOWN_RELATIONSHIP
	}
}

// Converts API runtime config to internal representation of parameters.
// Supports both v1beta1 and v2beta1 API.
// Supports conversion of an array of v1beta1 parameters, map[string]*structpb.Value,
// and v1beta1 or v2beta1 runtime configs.
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
	case []*apiv1beta1.Parameter:
		apiParams := obj
		var params util.SpecParameters
		for _, apiParam := range apiParams {
			if common.GetBoolConfigWithDefault(common.HasDefaultBucketEnvVar, false) {
				pVal, err := common.PatchPipelineDefaultParameter(apiParam.Value)
				if err == nil {
					apiParam.Value = pVal
				}
			}
			param := util.SpecParameter{
				Name:  apiParam.Name,
				Value: util.StringPointer(apiParam.Value),
			}
			params = append(params, param)
		}
		return toModelParameters(params)
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
	case *apiv1beta1.PipelineSpec_RuntimeConfig:
		runtimeConfig := obj
		protoParams := runtimeConfig.GetParameters()
		if protoParams == nil {
			return "", util.NewInternalServerError(util.NewInvalidInputError("Parameters cannot be nil"), "Failed to convert v1beta1 API runtime config to internal parameters representation")
		}
		return toModelParameters(protoParams)
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

// Converts internal parameters representation to their API counterpart.
// Supports v1beta1 API.
// Note: does not return an error anymore. Check of the result is nil instead.
func toApiParametersV1(p string) []*apiv1beta1.Parameter {
	apiParams := make([]*apiv1beta1.Parameter, 0)
	if p == "" || p == "null" || p == "[]" {
		return apiParams
	}
	params, err := util.UnmarshalParameters(util.CurrentExecutionType(), p)
	if err != nil {
		return nil
	}
	for _, param := range params {
		var value string
		if param.Value != nil {
			value = *param.Value
		}
		apiParam := apiv1beta1.Parameter{
			Name:  param.Name,
			Value: value,
		}
		apiParams = append(apiParams, &apiParam)
	}
	return apiParams
}

// Converts internal runtime parameters to (name, value) pairs as map[string]*structpb.Value.
// Supports v1beta1 and v2beta1 API.
// Note: returns nil if a parsing error occurs.
func toMapProtoStructParameters(p string) map[string]*structpb.Value {
	protoParams := make(map[string]*structpb.Value, 0)
	if p == "" || p == "null" || p == "[]" {
		return protoParams
	}
	err := json.Unmarshal([]byte(p), &protoParams)
	if err != nil {
		if paramsV1 := toApiParametersV1(p); paramsV1 == nil {
			return nil
		} else {
			for _, paramV1 := range paramsV1 {
				protoParams[paramV1.Name] = structpb.NewStringValue(paramV1.Value)
			}
		}
	}
	return protoParams
}

// Converts API trigger to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelTrigger(t interface{}) (*model.Trigger, error) {
	modelTrigger := model.Trigger{}
	if t == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("API Trigger cannot be nil"), "Failed to convert API trigger to its internal representation")
	}
	switch apiTrigger := t.(type) {
	case *apiv2beta1.Trigger:
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
	case *apiv1beta1.Trigger:
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
	default:
		return nil, util.NewUnknownApiVersionError("Trigger", t)
	}
	return &modelTrigger, nil
}

// Converts internal trigger representation to its API counterpart.
// Supports v1beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiTriggerV1(trigger *model.Trigger) *apiv1beta1.Trigger {
	if trigger == nil {
		return &apiv1beta1.Trigger{}
	}
	if trigger.Cron != nil && *trigger.Cron != "" {
		var cronSchedule apiv1beta1.CronSchedule
		cronSchedule.Cron = *trigger.Cron
		if trigger.CronScheduleStartTimeInSec != nil {
			cronSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleStartTimeInSec,
			}
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleEndTimeInSec,
			}
		}
		return &apiv1beta1.Trigger{Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}
	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
		var periodicSchedule apiv1beta1.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleStartTimeInSec,
			}
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleEndTimeInSec,
			}
		}
		return &apiv1beta1.Trigger{Trigger: &apiv1beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &periodicSchedule}}
	}
	if trigger.IntervalSecond == nil && trigger.Cron == nil {
		return &apiv1beta1.Trigger{}
	}
	return nil
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
			cronSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleStartTimeInSec,
			}
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleEndTimeInSec,
			}
		}
		return &apiv2beta1.Trigger{Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}
	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
		var periodicSchedule apiv2beta1.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleStartTimeInSec,
			}
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleEndTimeInSec,
			}
		}
		return &apiv2beta1.Trigger{Trigger: &apiv2beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &periodicSchedule}}
	}
	if trigger.IntervalSecond == nil && trigger.Cron == nil {
		return &apiv2beta1.Trigger{}
	}
	return nil
}

// Converts an array of API resource references to an array of their internal representations.
// Supports v1beta1 API.
// Note: avoid using reference resource name. Use resource's UUID instead.
// Possible relationship types:
// NAMESPACE:
//
//	OWNER of EXPERIMENT
//	OWNER of JOB
//	OWNER of RECURRING_RUN
//	OWNER of RUN
//	OWNER of PIPELINE
//	OWNER of PIPELINE_VERSION
//
// EXPERIMENT:
//
//	OWNER of JOB
//	OWNER of RECURRING_RUN
//	OWNER of RUN
//
// JOB:
//
//	CREATOR of RUN
//
// RECURRING_RUN:
//
//	CREATOR of RUN
//
// PIPELINE:
//
//	OWNER of PIPELINE_VERSION
//	CREATOR of JOB
//	CREATOR of RECURRING_RUN
//	CREATOR of RUN
//
// PIPELINE_VERSION:
//
//	CREATOR of JOB
//	CREATOR of RECURRING_RUN
//	CREATOR of RUN
func toModelResourceReferencesV1(apiRefs []*apiv1beta1.ResourceReference, resourceId string, resourceType apiv1beta1.ResourceType) ([]*model.ResourceReference, error) {
	modelRefs := make([]*model.ResourceReference, 0)
	for _, apiRef := range apiRefs {
		modelReferenceType, err := toModelResourceTypeV1(apiRef.Key.Type)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API resource references to their internal representation due to an error in reference type")
		}
		modelResourceType, err := toModelResourceTypeV1(resourceType)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API resource references to their internal representation due to an error in resource type")
		}
		modelRelationship, err := toModelRelationshipV1(apiRef.Relationship)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API resource references to their internal representation due to an error in reference relationship")
		}
		if !model.ValidateResourceReferenceRelationship(modelResourceType, modelReferenceType, modelRelationship) {
			return nil, util.Wrapf(errors.New("Invalid resource-reference relationship"), "Failed to convert v1beta1 API resource references to their internal representation due to invalid relationship: resource %T, reference %T, relationship %T", modelResourceType, modelReferenceType, modelRelationship)
		}

		modelRef := &model.ResourceReference{
			ResourceUUID:  resourceId,
			ResourceType:  modelResourceType,
			ReferenceUUID: apiRef.Key.Id,
			ReferenceName: apiRef.Name,
			ReferenceType: modelReferenceType,
			Relationship:  modelRelationship,
		}
		modelRefs = append(modelRefs, modelRef)
	}
	return modelRefs, nil
}

// Converts an array of resource references to an array of their API counterparts.
// Supports v1beta1 API.
func toApiResourceReferencesV1(references []*model.ResourceReference) []*apiv1beta1.ResourceReference {
	apiReferences := make([]*apiv1beta1.ResourceReference, 0)
	for _, ref := range references {
		apiReferences = append(apiReferences, &apiv1beta1.ResourceReference{
			Key: &apiv1beta1.ResourceKey{
				Type: toApiResourceTypeV1(ref.ReferenceType),
				Id:   ref.ReferenceUUID,
			},
			Name:         ref.ReferenceName,
			Relationship: toApiRelationshipV1(ref.Relationship),
		})
	}
	return apiReferences
}

// Converts API runtime config to its internal representations.
// Supports both v1beta1 and v2beta1 API.
func toModelRuntimeConfig(obj interface{}) (*model.RuntimeConfig, error) {
	if obj == nil {
		return nil, util.NewInvalidInputError("Failed to convert API runtime config to its internal representation. Input cannot be nil")
	}
	var params, root string
	switch obj := obj.(type) {
	case *apiv1beta1.PipelineSpec_RuntimeConfig:
		apiRuntimeConfigV1 := obj
		p, err := toModelParameters(apiRuntimeConfigV1.GetParameters())
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to convert v1beta1 API runtime config to its internal representation due to parameters conversion error")
		}
		params = p
		root = apiRuntimeConfigV1.GetPipelineRoot()
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
		Parameters:   params,
		PipelineRoot: root,
	}, nil
}

// Converts internal runtime config representation to its API counterpart.
// Supports v1beta1 API.
// Note: does not return error anymore. Check is the result is nil instead.
func toApiRuntimeConfigV1(modelRuntime model.RuntimeConfig) *apiv1beta1.PipelineSpec_RuntimeConfig {
	apiRuntimeConfig := apiv1beta1.PipelineSpec_RuntimeConfig{}
	if modelRuntime.Parameters == "" && modelRuntime.PipelineRoot == "" {
		return &apiRuntimeConfig
	}
	runtimeParams := toMapProtoStructParameters(modelRuntime.Parameters)
	if runtimeParams == nil {
		return nil
	}
	apiRuntimeConfig.Parameters = runtimeParams
	apiRuntimeConfig.PipelineRoot = modelRuntime.PipelineRoot
	return &apiRuntimeConfig
}

// Converts internal runtime config representation to its API counterpart.
// Supports v2beta1 API.
// Note: returns nil if a parsing error occurs.
func toApiRuntimeConfig(modelRuntime model.RuntimeConfig) *apiv2beta1.RuntimeConfig {
	apiRuntimeConfig := apiv2beta1.RuntimeConfig{}
	if modelRuntime.Parameters == "" && modelRuntime.PipelineRoot == "" {
		return &apiRuntimeConfig
	}
	runtimeParams := toMapProtoStructParameters(modelRuntime.Parameters)
	if runtimeParams == nil {
		return nil
	}
	apiRuntimeConfig.Parameters = runtimeParams
	apiRuntimeConfig.PipelineRoot = modelRuntime.PipelineRoot
	return &apiRuntimeConfig
}

// Converts internal runtime config representation to PipelineSpec's runtime config.
// Note: returns nil if a parsing error occurs.
func toPipelineSpecRuntimeConfig(cfg *model.RuntimeConfig) *pipelinespec.PipelineJob_RuntimeConfig {
	if cfg == nil {
		return &pipelinespec.PipelineJob_RuntimeConfig{}
	}
	runtimeParams := toMapProtoStructParameters(cfg.Parameters)
	if runtimeParams == nil {
		return nil
	}
	return &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues:    runtimeParams,
		GcsOutputDirectory: cfg.PipelineRoot,
	}
}

// Converts API run metric to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelRunMetric(m interface{}, runId string) (*model.RunMetric, error) {
	var name, nodeId, format string
	var val float64
	switch apiRunMetric := m.(type) {
	case *apiv1beta1.RunMetric:
		name = apiRunMetric.GetName()
		nodeId = apiRunMetric.GetNodeId()
		val = apiRunMetric.GetNumberValue()
		format = apiRunMetric.GetFormat().String()
	default:
		return nil, util.NewUnknownApiVersionError("RunMetric", m)
	}

	modelMetric := &model.RunMetric{
		RunUUID:     runId,
		Name:        name,
		NodeID:      nodeId,
		NumberValue: val,
		Format:      format,
	}
	return modelMetric, nil
}

// Converts internal run metric representation to its API counterpart.
// Supports v1beta1 API.
func toApiRunMetricV1(metric *model.RunMetric) *apiv1beta1.RunMetric {
	return &apiv1beta1.RunMetric{
		Name:   metric.Name,
		NodeId: metric.NodeID,
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: metric.NumberValue,
		},
		Format: apiv1beta1.RunMetric_Format(apiv1beta1.RunMetric_Format_value[metric.Format]),
	}
}

// Converts an array of internal run metric representations to an array of their API counterparts.
// Supports v1beta1 API.
func toApiRunMetricsV1(m []*model.RunMetric) []*apiv1beta1.RunMetric {
	apiMetrics := make([]*apiv1beta1.RunMetric, 0)
	for _, metric := range m {
		apiMetrics = append(apiMetrics, toApiRunMetricV1(metric))
	}
	return apiMetrics
}

// Convert results of run metrics creation to API response.
// Supports v1beta1 API.
// Return nil if a parsing error occurs.
func toApiReportMetricsResultV1(metricName string, nodeId string, status string, message string) *apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult {
	apiResultV1 := &apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
		MetricName:   metricName,
		MetricNodeId: nodeId,
		Message:      message,
	}
	switch status {
	case "ok":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
	case "internal":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR
	case "invalid":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT
	case "duplicate":
		apiResultV1.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_DUPLICATE_REPORTING
	default:
		return nil
	}
	return apiResultV1
}

// Converts API run or run details to internal run details representation.
// Supports both v1beta1 and v2beta1 API.
// TODO(gkcalat): update this to extend run details.
func toModelRunDetails(r interface{}) (*model.RunDetails, error) {
	switch r := r.(type) {
	case *apiv2beta1.Run:
		apiRunV2 := r
		modelRunDetails := &model.RunDetails{
			CreatedAtInSec:       apiRunV2.GetCreatedAt().GetSeconds(),
			ScheduledAtInSec:     apiRunV2.GetScheduledAt().GetSeconds(),
			FinishedAtInSec:      apiRunV2.GetFinishedAt().GetSeconds(),
			State:                model.RuntimeState(apiRunV2.GetState().String()),
			PipelineContextId:    apiRunV2.GetRunDetails().GetPipelineContextId(),
			PipelineRunContextId: apiRunV2.GetRunDetails().GetPipelineRunContextId(),
		}
		if apiRunV2.GetPipelineSpec() != nil {
			spec, err := pipelineSpecStructToYamlString(apiRunV2.GetPipelineSpec())
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to convert a API run to internal run details representation due to pipeline spec parsing error")
			}
			modelRunDetails.PipelineRuntimeManifest = spec
		}
		return modelRunDetails, nil
	case *apiv2beta1.RunDetails:
		return toModelRunDetails(apiv2beta1.Run{RunDetails: r})
	case *apiv1beta1.RunDetail:
		apiRunV1 := r.GetRun()
		modelRunDetails, err := toModelRunDetails(apiRunV1)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API run detail to its internal representation")
		}
		apiRuntimeV1 := r.GetPipelineRuntime()
		modelRunDetails.PipelineRuntimeManifest = apiRuntimeV1.GetPipelineManifest()
		modelRunDetails.WorkflowRuntimeManifest = apiRuntimeV1.GetWorkflowManifest()
		return modelRunDetails, nil
	default:
		return nil, util.NewUnknownApiVersionError("RunDetails", r)
	}
}

// Converts API run to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelRun(r interface{}) (*model.Run, error) {
	if r == nil {
		return &model.Run{}, nil
	}
	var namespace, experimentId, pipelineName, pipelineId, pipelineVersionId string
	var recRunId, runName, runDesc, runId, specParams, cfgParams string
	var pipelineSpec, workflowSpec, runtimePipelineSpec, runtimeWorkflowSpec string
	var pipelineRoot, storageState, serviceAcc string
	var createTime, scheduleTime, finishTime int64
	var modelMetrics []*model.RunMetric
	var state model.RuntimeState
	var stateHistory []*model.RuntimeStatus
	var tasks []*model.Task
	switch r := r.(type) {
	case *apiv1beta1.Run:
		return toModelRun(&apiv1beta1.RunDetail{Run: r})
	case *apiv1beta1.RunDetail:
		apiRunV1 := r.GetRun()
		if s, err := toModelRuntimeState(apiRunV1.GetStatus()); err == nil {
			state = s
		}
		apiPipelineRuntimeV1 := r.GetPipelineRuntime()
		// TODO(gkcalat): deserialize these two fields into runtime details of a run.
		runtimePipelineSpec = apiPipelineRuntimeV1.GetPipelineManifest()
		runtimeWorkflowSpec = apiPipelineRuntimeV1.GetWorkflowManifest()
		pipelineId = apiRunV1.GetPipelineSpec().GetPipelineId()
		if pipelineId == "" {
			pipelineId = getPipelineIdFromResourceReferencesV1(apiRunV1.GetResourceReferences())
		}
		pipelineVersionId = getPipelineVersionFromResourceReferencesV1(apiRunV1.GetResourceReferences())

		runName = apiRunV1.GetName()
		if runName == "" {
			runName = apiRunV1.GetPipelineSpec().GetPipelineName()
		}
		if runName == "" {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("Run name cannot be empty"), "Failed to convert a v1beta1 API run detail to its internal representation")
		}
		namespace = getNamespaceFromResourceReferenceV1(apiRunV1.GetResourceReferences())
		experimentId = getExperimentIdFromResourceReferencesV1(apiRunV1.GetResourceReferences())
		recRunId = getJobIdFromResourceReferencesV1(apiRunV1.GetResourceReferences())
		runId = apiRunV1.GetId()
		runDesc = apiRunV1.GetDescription()
		if temp, err := toModelStorageState(apiRunV1.GetStorageState()); err == nil {
			storageState = temp.ToString()
		}
		createTime = apiRunV1.GetCreatedAt().GetSeconds()
		scheduleTime = apiRunV1.GetScheduledAt().GetSeconds()
		finishTime = apiRunV1.GetFinishedAt().GetSeconds()
		if len(apiRunV1.GetMetrics()) > 0 {
			modelMetrics = make([]*model.RunMetric, 0)
			for _, metric := range apiRunV1.GetMetrics() {
				modelMetric, err := toModelRunMetric(metric, runId)
				if err == nil {
					modelMetrics = append(modelMetrics, modelMetric)
				}
			}
		}

		params, err := toModelParameters(apiRunV1.GetPipelineSpec().GetParameters())
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API run to its internal representation due to parameters parsing error")
		}
		specParams = params

		cfg, err := toModelRuntimeConfig(apiRunV1.GetPipelineSpec().GetRuntimeConfig())
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API run to its internal representation due to runtime config conversion error")
		}
		cfgParams = cfg.Parameters
		pipelineRoot = cfg.PipelineRoot

		pipelineSpec = apiRunV1.GetPipelineSpec().GetPipelineManifest()
		workflowSpec = apiRunV1.GetPipelineSpec().GetWorkflowManifest()
		serviceAcc = apiRunV1.GetServiceAccount()
	case *apiv2beta1.Run:
		apiRunV2 := r
		if temp, err := toModelTasks(apiRunV2.GetRunDetails().GetTaskDetails()); err == nil {
			tasks = temp
		} else {
			return nil, util.NewInternalServerError(err, "Failed to convert a API run detail to its internal representation due to error converting tasks")
		}
		if temp, err := toModelRuntimeState(apiRunV2.GetState()); err == nil {
			state = temp
		} else {
			return nil, util.NewInternalServerError(err, "Failed to convert a API run detail to its internal representation due to error converting runtime state")
		}
		if temp, err := toModelRuntimeStatuses(apiRunV2.GetStateHistory()); err == nil {
			stateHistory = temp
		} else {
			return nil, util.NewInternalServerError(err, "Failed to convert a API run detail to its internal representation due to error converting runtime state history")
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
					pipelineId = resources["PipelineId"]
				}
				if pipelineVersionId == "" {
					pipelineVersionId = resources["PipelineVersionId"]
				}
				if runId == "" {
					runId = resources["RunId"]
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
		cfgParams = cfg.Parameters
		pipelineRoot = cfg.PipelineRoot

		if apiRunV2.GetPipelineSpec() == nil {
			pipelineSpec = ""
		} else if spec, err := pipelineSpecStructToYamlString(apiRunV2.GetPipelineSpec()); err == nil {
			pipelineSpec = spec
		} else {
			pipelineSpec = ""
		}
		specParams = ""
	default:
		return nil, util.NewUnknownApiVersionError("Run", r)
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
		Metrics:        modelMetrics,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipelineId,
			PipelineVersionId:    pipelineVersionId,
			PipelineName:         pipelineName,
			PipelineSpecManifest: pipelineSpec,
			WorkflowSpecManifest: workflowSpec,
			Parameters:           specParams,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   cfgParams,
				PipelineRoot: pipelineRoot,
			},
		},
		RunDetails: model.RunDetails{
			State:                   state.ToV2(),
			StateHistory:            stateHistory,
			CreatedAtInSec:          createTime,
			ScheduledAtInSec:        scheduleTime,
			FinishedAtInSec:         finishTime,
			PipelineRuntimeManifest: runtimePipelineSpec,
			WorkflowRuntimeManifest: runtimeWorkflowSpec,
			TaskDetails:             tasks,
		},
	}
	return &modelRun, nil
}

// Converts internal representation of a run to its API counterpart.
// Supports v1beta1 API.
// Note: adds error details to the message if a parsing error occurs.
func toApiRunV1(r *model.Run) *apiv1beta1.Run {
	r = r.ToV1()
	// v1 parameters
	specParams := toApiParametersV1(r.PipelineSpec.Parameters)
	if specParams == nil {
		return &apiv1beta1.Run{
			Id:    r.UUID,
			Error: util.Wrap(errors.New("Failed to parse pipeline spec parameters"), "Failed to convert internal run representation to its v1beta1 API counterpart").Error(),
		}
	}
	var runtimeConfig *apiv1beta1.PipelineSpec_RuntimeConfig
	if len(specParams) == 0 {
		specParams = nil
		runtimeConfig = toApiRuntimeConfigV1(r.PipelineSpec.RuntimeConfig)
		if runtimeConfig == nil {
			return &apiv1beta1.Run{
				Id:    r.UUID,
				Error: util.Wrap(errors.New("Failed to parse runtime config"), "Failed to convert internal run representation to its v1beta1 API counterpart").Error(),
			}
		}
		if len(runtimeConfig.GetParameters()) == 0 && len(runtimeConfig.GetPipelineRoot()) == 0 {
			runtimeConfig = nil
		}
	}
	var metrics []*apiv1beta1.RunMetric
	if r.Metrics != nil {
		metrics = toApiRunMetricsV1(r.Metrics)
	}
	if len(metrics) == 0 {
		metrics = nil
	}

	resRefs := toApiResourceReferencesV1(r.ResourceReferences)
	if resRefs == nil {
		return &apiv1beta1.Run{
			Id:    r.UUID,
			Error: util.Wrap(errors.New("Failed to parse resource references"), "Failed to convert internal run representation to its v1beta1 API counterpart").Error(),
		}
	}
	if rrNamespace := getNamespaceFromResourceReferenceV1(resRefs); rrNamespace == "" && r.Namespace != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   r.Namespace,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		)
	}
	if rrExperimentId := getExperimentIdFromResourceReferencesV1(resRefs); rrExperimentId == "" && r.ExperimentId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   r.ExperimentId,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		)
	}
	if rrJobId := getJobIdFromResourceReferencesV1(resRefs); rrJobId == "" && r.RecurringRunId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_JOB,
					Id:   r.RecurringRunId,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		)
	} else if rrPipelineVersionId := getPipelineVersionFromResourceReferencesV1(resRefs); rrPipelineVersionId == "" && r.PipelineSpec.PipelineVersionId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE_VERSION,
					Id:   r.PipelineSpec.PipelineVersionId,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		)
	} else if rrPipelineId := getPipelineIdFromResourceReferencesV1(resRefs); rrPipelineId == "" && r.PipelineSpec.PipelineId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE,
					Id:   r.PipelineSpec.PipelineId,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		)
	}
	if len(resRefs) == 0 {
		resRefs = nil
	}
	specManifest := r.PipelineSpec.PipelineSpecManifest
	wfManifest := r.PipelineSpec.WorkflowSpecManifest
	return &apiv1beta1.Run{
		CreatedAt:      &timestamp.Timestamp{Seconds: r.RunDetails.CreatedAtInSec},
		Id:             r.UUID,
		Metrics:        metrics,
		Name:           r.DisplayName,
		ServiceAccount: r.ServiceAccount,
		StorageState:   apiv1beta1.Run_StorageState(apiv1beta1.Run_StorageState_value[string(r.StorageState.ToV1())]),
		Description:    r.Description,
		ScheduledAt:    &timestamp.Timestamp{Seconds: r.RunDetails.ScheduledAtInSec},
		FinishedAt:     &timestamp.Timestamp{Seconds: r.RunDetails.FinishedAtInSec},
		Status:         string(r.RunDetails.State.ToV1()),
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       r.PipelineSpec.PipelineId,
			PipelineName:     r.PipelineSpec.PipelineName,
			WorkflowManifest: wfManifest,
			PipelineManifest: specManifest,
			Parameters:       specParams,
			RuntimeConfig:    runtimeConfig,
		},
		ResourceReferences: resRefs,
	}
}

// Converts internal representation of a run to its API counterpart.
// Supports v2beta1 API.
// Note: adds error details to the message if a parsing error occurs.
func toApiRun(r *model.Run) *apiv2beta1.Run {
	r = r.ToV2()
	runtimeConfig := toApiRuntimeConfig(r.PipelineSpec.RuntimeConfig)
	if runtimeConfig == nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(util.Wrap(errors.New("Failed to parse runtime config"), "Failed to convert internal run representation to its API counterpart")),
		}
	}
	if len(runtimeConfig.GetParameters()) == 0 && len(runtimeConfig.GetPipelineRoot()) == 0 {
		if params := toMapProtoStructParameters(r.PipelineSpec.Parameters); len(params) > 0 {
			runtimeConfig.Parameters = params
		} else {
			runtimeConfig = nil
		}
	}
	apiRd := &apiv2beta1.RunDetails{
		PipelineContextId:    r.RunDetails.PipelineContextId,
		PipelineRunContextId: r.RunDetails.PipelineRunContextId,
		TaskDetails:          toApiPipelineTaskDetails(r.RunDetails.TaskDetails),
	}
	if apiRd.PipelineContextId == 0 && apiRd.PipelineRunContextId == 0 && apiRd.TaskDetails == nil {
		apiRd = nil
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
		CreatedAt:      &timestamp.Timestamp{Seconds: r.RunDetails.CreatedAtInSec},
		ScheduledAt:    &timestamp.Timestamp{Seconds: r.RunDetails.ScheduledAtInSec},
		FinishedAt:     &timestamp.Timestamp{Seconds: r.RunDetails.FinishedAtInSec},
		RunDetails:     apiRd,
	}
	err := util.NewInvalidInputError("Failed to parse the pipeline source")
	if r.PipelineSpec.PipelineVersionId != "" {
		apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId:        r.PipelineSpec.PipelineId,
				PipelineVersionId: r.PipelineSpec.PipelineVersionId,
			},
		}
		return apiRunV2
	} else if r.PipelineSpec.PipelineSpecManifest != "" {
		spec, err1 := yamlStringToPipelineSpecStruct(r.PipelineSpec.PipelineSpecManifest)
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
			return apiRunV2
		}
		err = util.Wrap(err1, err.Error()).(*util.UserError)
	} else if r.PipelineSpec.WorkflowSpecManifest != "" {
		spec, err1 := yamlStringToPipelineSpecStruct(r.PipelineSpec.WorkflowSpecManifest)
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
			return apiRunV2
		}
		err = util.Wrap(err1, err.Error()).(*util.UserError)
	}
	return &apiv2beta1.Run{
		RunId:        r.UUID,
		ExperimentId: r.ExperimentId,
		Error:        util.ToRpcStatus(util.Wrap(err, "Failed to convert internal run representation to its API counterpart due to missing pipeline source")),
	}
}

// Converts an array of internal pipeline version representations to an array of API pipeline versions.
// Supports v1beta1 API.
func toApiRunsV1(runs []*model.Run) []*apiv1beta1.Run {
	apiRuns := make([]*apiv1beta1.Run, 0)
	for _, run := range runs {
		apiRuns = append(apiRuns, toApiRunV1(run))
	}
	return apiRuns
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

// Converts internal representation of a run to v1beta1 API run detail.
// Supports v1beta1 API.
// Note: adds error details to the nested run message if a parsing error occurs.
func toApiRunDetailV1(r *model.Run) *apiv1beta1.RunDetail {
	apiRunV1 := toApiRunV1(r)
	apiRunDetails := &apiv1beta1.RunDetail{
		Run: apiRunV1,
	}
	if r.RunDetails.WorkflowRuntimeManifest == "" {
		apiRunDetails.PipelineRuntime = &apiv1beta1.PipelineRuntime{
			PipelineManifest: r.RunDetails.PipelineRuntimeManifest,
		}
	} else if r.RunDetails.PipelineRuntimeManifest == "" {
		apiRunDetails.PipelineRuntime = &apiv1beta1.PipelineRuntime{
			WorkflowManifest: r.RunDetails.WorkflowRuntimeManifest,
		}
	} else {
		apiRunDetails.PipelineRuntime = &apiv1beta1.PipelineRuntime{
			PipelineManifest: r.RunDetails.PipelineRuntimeManifest,
			WorkflowManifest: r.RunDetails.WorkflowRuntimeManifest,
		}
	}
	return apiRunDetails
}

// Converts API task to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelTask(t interface{}) (*model.Task, error) {
	if t == nil {
		return &model.Task{}, nil
	}
	var taskId, nodeId, namespace, pipelineName, runId, mlmdExecId, fingerprint string
	var name, parentTaskId, state, inputs, outputs string
	var createTime, startTime, finishTime int64
	var stateHistory []*model.RuntimeStatus
	var children []string
	switch t := t.(type) {
	case *apiv1beta1.Task:
		apiTaskV1 := t
		namespace = apiTaskV1.GetNamespace()
		taskId = apiTaskV1.GetId()
		pipelineName = apiTaskV1.GetPipelineName()
		runId = apiTaskV1.GetRunId()
		mlmdExecId = apiTaskV1.GetMlmdExecutionID()
		fingerprint = apiTaskV1.GetFingerprint()
		createTime = apiTaskV1.GetCreatedAt().GetSeconds()
		startTime = createTime
		finishTime = apiTaskV1.GetFinishedAt().GetSeconds()
		name = ""
		parentTaskId = ""
		state = ""
		inputs = ""
		outputs = ""
	case *apiv2beta1.PipelineTaskDetail:
		apiTaskDetailV2 := t
		namespace = ""
		taskId = apiTaskDetailV2.GetTaskId()
		pipelineName = ""
		runId = apiTaskDetailV2.GetRunId()
		mlmdExecId = fmt.Sprint(apiTaskDetailV2.GetExecutionId())
		fingerprint = ""
		createTime = apiTaskDetailV2.GetCreateTime().GetSeconds()
		startTime = apiTaskDetailV2.GetStartTime().GetSeconds()
		finishTime = apiTaskDetailV2.GetEndTime().GetSeconds()
		name = apiTaskDetailV2.GetDisplayName()
		parentTaskId = apiTaskDetailV2.GetParentTaskId()
		state = apiTaskDetailV2.GetState().String()
		if hist, err := toModelRuntimeStatuses(apiTaskDetailV2.GetStateHistory()); err == nil {
			stateHistory = hist
		}
		if inpBytes, err := json.Marshal(apiTaskDetailV2.GetInputs()); err == nil {
			inputs = string(inpBytes)
		}
		if outBytes, err := json.Marshal(apiTaskDetailV2.GetOutputs()); err == nil {
			outputs = string(outBytes)
		}
		for _, c := range apiTaskDetailV2.GetChildTasks() {
			if c.GetTaskId() != "" {
				children = append(children, c.GetTaskId())
			} else {
				children = append(children, c.GetPodName())
			}
		}
	case util.NodeStatus:
		// TODO(gkcalat): parse input and output artifacts
		wfStatus := t
		nodeId = wfStatus.ID
		name = wfStatus.DisplayName
		state = wfStatus.State
		startTime = wfStatus.StartTime
		createTime = wfStatus.CreateTime
		finishTime = wfStatus.FinishTime
		children = wfStatus.Children
	default:
		return nil, util.NewUnknownApiVersionError("Task", t)
	}
	return &model.Task{
		UUID:              taskId,
		PodName:           nodeId,
		Namespace:         namespace,
		PipelineName:      pipelineName,
		RunId:             runId,
		MLMDExecutionID:   mlmdExecId,
		CreatedTimestamp:  createTime,
		StartedTimestamp:  startTime,
		FinishedTimestamp: finishTime,
		Fingerprint:       fingerprint,
		Name:              name,
		ParentTaskId:      parentTaskId,
		State:             model.RuntimeState(state).ToV2(),
		StateHistory:      stateHistory,
		MLMDInputs:        inputs,
		MLMDOutputs:       outputs,
		ChildrenPods:      children,
	}, nil
}

// Converts API tasks details into their internal representations.
// Supports both v1beta1 and v2beta1 API.
func toModelTasks(t interface{}) ([]*model.Task, error) {
	if t == nil {
		return nil, nil
	}
	switch t := t.(type) {
	case []*apiv2beta1.PipelineTaskDetail:
		apiTasks := t
		modelTasks := make([]*model.Task, 0)
		for _, apiTask := range apiTasks {
			modelTask, err := toModelTask(apiTask)
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert API tasks to their internal representations")
			}
			modelTasks = append(modelTasks, modelTask)
		}
		return modelTasks, nil
	case util.ExecutionSpec:
		execSpec := t
		runId := execSpec.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowRunId]
		namespace := execSpec.ExecutionNamespace()
		createdAt := execSpec.ExecutionObjectMeta().GetCreationTimestamp().Unix()
		// Get sorted node names to make the results repeatable
		nodes := execSpec.ExecutionStatus().NodeStatuses()
		nodeNames := make([]string, 0, len(nodes))
		for nodeName := range nodes {
			nodeNames = append(nodeNames, nodeName)
		}
		sort.Strings(nodeNames)
		modelTasks := make([]*model.Task, 0)
		for _, nodeName := range nodeNames {
			node := nodes[nodeName]
			modelTask, err := toModelTask(node)
			if err != nil {
				return nil, util.Wrap(err, "Failed to convert Argo workflow to tasks details")
			}
			modelTask.RunId = runId
			modelTask.Namespace = namespace
			modelTask.CreatedTimestamp = createdAt
			modelTasks = append(modelTasks, modelTask)
		}
		return modelTasks, nil
	default:
		return nil, util.NewUnknownApiVersionError("[]Task", t)
	}
}

// Converts internal task representation to its API counterpart.
// Supports v1beta1 API.
func toApiTaskV1(task *model.Task) *apiv1beta1.Task {
	return &apiv1beta1.Task{
		Id:              task.UUID,
		Namespace:       task.Namespace,
		PipelineName:    task.PipelineName,
		RunId:           task.RunId,
		MlmdExecutionID: task.MLMDExecutionID,
		CreatedAt:       &timestamp.Timestamp{Seconds: task.CreatedTimestamp},
		FinishedAt:      &timestamp.Timestamp{Seconds: task.FinishedTimestamp},
		Fingerprint:     task.Fingerprint,
	}
}

// Converts internal task representation to its API counterpart.
// Supports v2beta1 API.
// TODO(gkcalat): implement runtime details of a task.
func toApiPipelineTaskDetail(t *model.Task) *apiv2beta1.PipelineTaskDetail {
	execId, err := strconv.ParseInt(t.MLMDExecutionID, 10, 64)
	if err != nil {
		execId = 0
	}
	var inputArtifacts map[string]*apiv2beta1.ArtifactList
	if t.MLMDInputs != "" {
		err = json.Unmarshal([]byte(t.MLMDInputs), &inputArtifacts)
		if err != nil {
			return &apiv2beta1.PipelineTaskDetail{
				RunId:  t.RunId,
				TaskId: t.UUID,
				Error:  util.ToRpcStatus(util.NewInternalServerError(err, "Failed to convert task's internal representation to its API counterpart due to error parsing inputs")),
			}
		}
	}
	var outputArtifacts map[string]*apiv2beta1.ArtifactList
	if t.MLMDOutputs != "" {
		err = json.Unmarshal([]byte(t.MLMDOutputs), &outputArtifacts)
		if err != nil {
			return &apiv2beta1.PipelineTaskDetail{
				RunId:  t.RunId,
				TaskId: t.UUID,
				Error:  util.ToRpcStatus(util.NewInternalServerError(err, "Failed to convert task's internal representation to its API counterpart due to error parsing outputs")),
			}
		}
	}
	var children []*apiv2beta1.PipelineTaskDetail_ChildTask
	for _, c := range t.ChildrenPods {
		children = append(children, &apiv2beta1.PipelineTaskDetail_ChildTask{
			ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: c},
		})
	}
	return &apiv2beta1.PipelineTaskDetail{
		RunId:        t.RunId,
		TaskId:       t.UUID,
		DisplayName:  t.Name,
		CreateTime:   &timestamp.Timestamp{Seconds: t.CreatedTimestamp},
		StartTime:    &timestamp.Timestamp{Seconds: t.StartedTimestamp},
		EndTime:      &timestamp.Timestamp{Seconds: t.FinishedTimestamp},
		State:        apiv2beta1.RuntimeState(apiv2beta1.RuntimeState_value[t.State.ToString()]),
		ExecutionId:  execId,
		Inputs:       inputArtifacts,
		Outputs:      outputArtifacts,
		ParentTaskId: t.ParentTaskId,
		StateHistory: toApiRuntimeStatuses(t.StateHistory),
		ChildTasks:   children,
	}
}

// Converts and array of internal task representations to its API counterpart.
// Supports v1beta1 API.
func toApiTasksV1(tasks []*model.Task) []*apiv1beta1.Task {
	if len(tasks) == 0 {
		return nil
	}
	apiTasks := make([]*apiv1beta1.Task, 0)
	for _, task := range tasks {
		apiTasks = append(apiTasks, toApiTaskV1(task))
	}
	return apiTasks
}

// Converts and array of internal task representations to its API counterpart.
// Supports v2beta1 API.
func toApiPipelineTaskDetails(tasks []*model.Task) []*apiv2beta1.PipelineTaskDetail {
	if len(tasks) == 0 {
		return nil
	}
	apiTasks := make([]*apiv2beta1.PipelineTaskDetail, 0)
	for _, task := range tasks {
		apiTasks = append(apiTasks, toApiPipelineTaskDetail(task))
	}
	return apiTasks
}

// Converts API recurring run to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelJob(j interface{}) (*model.Job, error) {
	if j == nil {
		return &model.Job{}, nil
	}
	var jobId, jobName, k8sName, namespace, serviceAcc, desc, experimentId, pipelineName string
	var pipelineId, pipelineVersionId, pipelineSpec, workflowSpec, specParams, cfgParams, pipelineRoot string
	var maxConcur, createTime, updateTime int64
	var noCatchup, isEnabled bool
	var trigger *model.Trigger
	resRefs := make([]*model.ResourceReference, 0)
	switch apiJob := j.(type) {
	case *apiv1beta1.Job:
		pipelineId = apiJob.GetPipelineSpec().GetPipelineId()
		if pipelineId == "" {
			pipelineId = getPipelineIdFromResourceReferencesV1(apiJob.GetResourceReferences())
		}
		pipelineVersionId = getPipelineVersionFromResourceReferencesV1(apiJob.GetResourceReferences())

		jobName = apiJob.GetName()
		if jobName == "" {
			jobName = apiJob.GetPipelineSpec().GetPipelineName()
		}
		if jobName == "" {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("Job name cannot be empty"), "Failed to convert a v1beta1 API recurring run to its internal representation")
		}

		if t, err := toModelTrigger(apiJob.GetTrigger()); err == nil {
			trigger = t
		} else {
			return nil, util.Wrap(err, "Failed to convert a v1beta1 API recurring run to its internal representation due to trigger parsing error")
		}

		// TODO(gkcalat): consider deprecating Enabled field in v1beta1/job.proto
		isEnabled = apiJob.GetEnabled()
		if !isEnabled {
			if flag, err := toModelJobEnabled(apiJob.GetMode()); err != nil {
				return nil, util.Wrap(err, "Failed to convert a v1beta1 API recurring run to its internal representation due to mode parsing error")
			} else {
				isEnabled = flag
			}
		}

		jobId = apiJob.GetId()
		desc = apiJob.GetDescription()
		namespace = getNamespaceFromResourceReferenceV1(apiJob.GetResourceReferences())
		experimentId = getExperimentIdFromResourceReferencesV1(apiJob.GetResourceReferences())
		serviceAcc = apiJob.GetServiceAccount()
		noCatchup = apiJob.GetNoCatchup()
		maxConcur = apiJob.GetMaxConcurrency()
		createTime = apiJob.GetCreatedAt().GetSeconds()
		updateTime = apiJob.GetUpdatedAt().GetSeconds()

		if params, err := toModelParameters(apiJob.GetPipelineSpec().GetParameters()); err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API recurring run to its internal representation due to parameters parsing error")
		} else {
			specParams = params
		}

		cfg, err := toModelRuntimeConfig(apiJob.GetPipelineSpec().GetRuntimeConfig())
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert v1beta1 API recurring run to its internal representation due to runtime config conversion error")
		}
		cfgParams = cfg.Parameters
		pipelineRoot = cfg.PipelineRoot

		pipelineSpec = apiJob.GetPipelineSpec().GetPipelineManifest()
		workflowSpec = apiJob.GetPipelineSpec().GetWorkflowManifest()
		k8sName = jobName
	case *apiv2beta1.RecurringRun:
		pipelineId = apiJob.GetPipelineVersionReference().GetPipelineId()
		pipelineVersionId = apiJob.GetPipelineVersionReference().GetPipelineVersionId()
		if spec, err := pipelineSpecStructToYamlString(apiJob.GetPipelineSpec()); err == nil {
			pipelineSpec = spec
		} else {
			return nil, util.Wrap(err, "Failed to convert API recurring run to its internal representation due to pipeline spec conversion error")
		}

		cfg, err := toModelRuntimeConfig(apiJob.GetRuntimeConfig())
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert API recurring run to its internal representation due to runtime config conversion error")
		}
		cfgParams = cfg.Parameters
		pipelineRoot = cfg.PipelineRoot

		jobName = apiJob.GetDisplayName()
		if jobName == "" {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("Recurring run's name cannot be empty"), "Failed to convert a API recurring run to its internal representation")
		}

		if t, err := toModelTrigger(apiJob.GetTrigger()); err == nil {
			trigger = t
		} else {
			return nil, util.Wrap(err, "Failed to convert a API recurring run to its internal representation due to trigger parsing error")
		}
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
	default:
		return nil, util.NewUnknownApiVersionError("RecurringRun", j)
	}
	if maxConcur > 10 || maxConcur < 1 {
		return nil, util.NewInvalidInputError("Max concurrency of a recurring run must be at leas 1 and at most 10. Received %v", maxConcur)
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
		ResourceReferences: resRefs,
		Trigger:            *trigger,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           pipelineId,
			PipelineName:         pipelineName,
			PipelineVersionId:    pipelineVersionId,
			PipelineSpecManifest: pipelineSpec,
			WorkflowSpecManifest: workflowSpec,
			Parameters:           specParams,
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   cfgParams,
				PipelineRoot: pipelineRoot,
			},
		},
	}, nil
}

// Converts API recurring run's mode to its internal representation.
// Supports both v1beta and v2beta1 API.
func toModelJobEnabled(m interface{}) (bool, error) {
	if m == nil {
		return false, nil
	}
	switch mode := m.(type) {
	case apiv2beta1.RecurringRun_Mode:
		switch mode {
		case apiv2beta1.RecurringRun_ENABLE:
			return true, nil
		case apiv2beta1.RecurringRun_MODE_UNSPECIFIED, apiv2beta1.RecurringRun_DISABLE:
			return false, nil
		default:
			return false, util.NewInternalServerError(util.NewInvalidInputError("Recurring run's mode is invalid: %v", mode), "Failed to convert API recurring run's mode to its internal representation")
		}
	case apiv1beta1.Job_Mode:
		switch mode {
		case apiv1beta1.Job_ENABLED:
			return true, nil
		case apiv1beta1.Job_UNKNOWN_MODE, apiv1beta1.Job_DISABLED:
			return false, nil
		default:
			return false, util.NewInternalServerError(util.NewInvalidInputError("Recurring run's mode is invalid: %v", mode), "Failed to convert v1beta1 API recurring run's mode to its internal representation")
		}
	default:
		return false, util.NewUnknownApiVersionError("RecurringRun.Mode", m)
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

// Converts internal recurring run's status to API counterpart.
// Supports v1beta1 API.
// Note: returns STATUS_UNSPECIFIED by default.
// Note: the returned values are now consistent with v2beta1 and will differ
// from Argo's conditions: [Enabled, Disabled, Running, Succeeded, Error].
// The mapping from Argo to v2beta:
// Enabled, Running, Succeeded -> ENABLED
// Disabled -> DISABLED
// Error -> STATUS_UNSPECIFIED.
func toApiJobStatus(s string) string {
	switch s {
	case string(model.StatusStateEnabled), string(swapi.ScheduledWorkflowSucceeded), string(swapi.ScheduledWorkflowRunning), string(swapi.ScheduledWorkflowEnabled):
		return string(model.StatusStateEnabled)
	case string(model.StatusStateDisabled), string(swapi.ScheduledWorkflowDisabled):
		return string(model.StatusStateDisabled)
	case string(model.StatusStateUnspecified), string(model.StatusStateUnspecifiedV1), string(swapi.ScheduledWorkflowError):
		return string(model.StatusStateUnspecified)
	default:
		return string(model.StatusStateUnspecified)
	}
}

// Converts recurring run's internal representation to its API counterpart.
// Supports v1beta1 API.
func toApiJobV1(j *model.Job) *apiv1beta1.Job {
	j = j.ToV1()
	specParams := toApiParametersV1(j.PipelineSpec.Parameters)
	if specParams == nil {
		return &apiv1beta1.Job{
			Id:    j.UUID,
			Error: util.NewInternalServerError(util.NewInvalidInputError("Pipeline v1 parameters were not parsed correctly"), "Failed to convert recurring run's internal representation to its v1beta1 API counterpart").Error(),
		}
	}
	var runtimeConfig *apiv1beta1.PipelineSpec_RuntimeConfig
	if len(specParams) == 0 {
		specParams = nil
		runtimeConfig = toApiRuntimeConfigV1(j.PipelineSpec.RuntimeConfig)
		if runtimeConfig == nil {
			return &apiv1beta1.Job{
				Id:    j.UUID,
				Error: util.NewInternalServerError(util.NewInvalidInputError("Runtime config was not parsed correctly"), "Failed to convert recurring run's internal representation to its v1beta1 API counterpart").Error(),
			}
		}
		if len(runtimeConfig.GetParameters()) == 0 && len(runtimeConfig.GetPipelineRoot()) == 0 {
			runtimeConfig = nil
		}
	}
	resRefs := toApiResourceReferencesV1(j.ResourceReferences)
	if resRefs == nil {
		return &apiv1beta1.Job{
			Id:    j.UUID,
			Error: util.NewInternalServerError(util.NewInvalidInputError("Resource references were not parsed correctly"), "Failed to convert recurring run's internal representation to its v1beta1 API counterpart").Error(),
		}
	}
	if rrNamespace := getNamespaceFromResourceReferenceV1(resRefs); rrNamespace == "" && j.Namespace != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   j.Namespace,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		)
	}
	if rrExperimentId := getExperimentIdFromResourceReferencesV1(resRefs); rrExperimentId == "" && j.ExperimentId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   j.ExperimentId,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		)
	}
	if rrPipelineVersionId := getPipelineVersionFromResourceReferencesV1(resRefs); rrPipelineVersionId == "" && j.PipelineSpec.PipelineVersionId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE_VERSION,
					Id:   j.PipelineSpec.PipelineVersionId,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		)
	} else if rrPipelineId := getPipelineIdFromResourceReferencesV1(resRefs); rrPipelineId == "" && j.PipelineSpec.PipelineId != "" {
		resRefs = append(
			resRefs,
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_PIPELINE,
					Id:   j.PipelineSpec.PipelineId,
				},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		)
	}
	if len(resRefs) == 0 {
		resRefs = nil
	}
	trigger := toApiTriggerV1(&j.Trigger)
	if trigger.GetTrigger() == nil {
		trigger = nil
	}

	specManifest := j.PipelineSpec.PipelineSpecManifest
	wfManifest := j.PipelineSpec.WorkflowSpecManifest
	return &apiv1beta1.Job{
		Id:             j.UUID,
		Name:           j.DisplayName,
		ServiceAccount: j.ServiceAccount,
		Description:    j.Description,
		Enabled:        j.Enabled,
		CreatedAt:      &timestamp.Timestamp{Seconds: j.CreatedAtInSec},
		Status:         toApiJobStatus(j.Conditions),
		UpdatedAt:      &timestamp.Timestamp{Seconds: j.UpdatedAtInSec},
		MaxConcurrency: j.MaxConcurrency,
		NoCatchup:      j.NoCatchup,
		Trigger:        trigger,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       j.PipelineSpec.PipelineId,
			PipelineName:     j.PipelineSpec.PipelineName,
			WorkflowManifest: wfManifest,
			PipelineManifest: specManifest,
			Parameters:       specParams,
			RuntimeConfig:    runtimeConfig,
		},
		ResourceReferences: resRefs,
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
		if params := toMapProtoStructParameters(j.PipelineSpec.Parameters); len(params) > 0 {
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
		CreatedAt:      &timestamp.Timestamp{Seconds: j.CreatedAtInSec},
		UpdatedAt:      &timestamp.Timestamp{Seconds: j.UpdatedAtInSec},
		MaxConcurrency: j.MaxConcurrency,
		NoCatchup:      j.NoCatchup,
		Trigger:        toApiTrigger(&j.Trigger),
		RuntimeConfig:  runtimeConfig,
		Namespace:      j.Namespace,
		ExperimentId:   j.ExperimentId,
	}

	if j.PipelineSpec.PipelineVersionId == "" {
		spec, err := yamlStringToPipelineSpecStruct(j.PipelineSpec.PipelineSpecManifest)
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
// Supports v1beta1 API.
func toApiJobsV1(jobs []*model.Job) []*apiv1beta1.Job {
	apiJobs := make([]*apiv1beta1.Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, toApiJobV1(job))
	}
	return apiJobs
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
// Supports both v1beta1 and v2beta1 API.
func toModelStorageState(s interface{}) (model.StorageState, error) {
	if s == nil {
		return model.StorageStateUnspecified, nil
	}
	switch s.(type) {
	case string, *string:
		state := s.(string)
		switch state {
		case string(model.StorageStateArchived), string(model.StorageStateArchived.ToV1()):
			return model.StorageStateArchived, nil
		case string(model.StorageStateAvailable), string(model.StorageStateAvailable.ToV1()):
			return model.StorageStateAvailable, nil
		case string(model.StorageStateUnspecified), string(model.StorageStateUnspecified.ToV1()):
			return model.StorageStateUnspecified, nil
		default:
			return "", util.NewInternalServerError(util.NewInvalidInputError("Storage state cannot be equal to %v", s), "Failed to convert API storage state to its internal representation")
		}
	case apiv1beta1.Run_StorageState, *apiv1beta1.Run_StorageState:
		return toModelStorageState(apiv1beta1.Run_StorageState_name[int32(s.(apiv1beta1.Run_StorageState))])
	case apiv1beta1.Experiment_StorageState, *apiv1beta1.Experiment_StorageState:
		return toModelStorageState(apiv1beta1.Experiment_StorageState_name[int32(s.(apiv1beta1.Experiment_StorageState))])
	case apiv2beta1.Run_StorageState, *apiv2beta1.Run_StorageState:
		return toModelStorageState(apiv2beta1.Run_StorageState_name[int32(s.(apiv2beta1.Run_StorageState))])
	case apiv2beta1.Experiment_StorageState, *apiv2beta1.Experiment_StorageState:
		return toModelStorageState(apiv2beta1.Experiment_StorageState_name[int32(s.(apiv2beta1.Experiment_StorageState))])
	default:
		return "", util.NewUnknownApiVersionError("StorageState", s)
	}
}

// Converts internal storage state representation to its API run's counterpart.
// Support v2beta1 API.
func toApiRunStorageState(s *model.StorageState) apiv2beta1.Run_StorageState {
	if string(*s) == "" {
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchived.ToV1()):
		return apiv2beta1.Run_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailable.ToV1()):
		return apiv2beta1.Run_AVAILABLE
	case string(model.StorageStateUnspecified), string(model.StorageStateUnspecified.ToV1()):
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	default:
		return apiv2beta1.Run_STORAGE_STATE_UNSPECIFIED
	}
}

// Converts internal storage state representation to its API run's counterpart.
// Support v1beta1 API.
// Note, default to STORAGESTATE_AVAILABLE.
func toApiRunStorageStateV1(s *model.StorageState) apiv1beta1.Run_StorageState {
	if string(*s) == "" {
		return apiv1beta1.Run_STORAGESTATE_AVAILABLE
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchived.ToV1()):
		return apiv1beta1.Run_STORAGESTATE_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailable.ToV1()):
		return apiv1beta1.Run_STORAGESTATE_AVAILABLE
	default:
		return apiv1beta1.Run_STORAGESTATE_AVAILABLE
	}
}

// Converts internal storage state representation to its API experiment's counterpart.
// Support v2beta1 API.
func toApiExperimentStorageState(s *model.StorageState) apiv2beta1.Experiment_StorageState {
	if string(*s) == "" {
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchived.ToV1()):
		return apiv2beta1.Experiment_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailable.ToV1()):
		return apiv2beta1.Experiment_AVAILABLE
	case string(model.StorageStateUnspecified), string(model.StorageStateUnspecified.ToV1()):
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	default:
		return apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED
	}
}

// Converts internal storage state representation to its API experiment's counterpart.
// Support v1beta1 API.
func toApiExperimentStorageStateV1(s *model.StorageState) apiv1beta1.Experiment_StorageState {
	if string(*s) == "" {
		return apiv1beta1.Experiment_STORAGESTATE_UNSPECIFIED
	}
	switch string(*s) {
	case string(model.StorageStateArchived), string(model.StorageStateArchived.ToV1()):
		return apiv1beta1.Experiment_STORAGESTATE_ARCHIVED
	case string(model.StorageStateAvailable), string(model.StorageStateAvailable.ToV1()):
		return apiv1beta1.Experiment_STORAGESTATE_AVAILABLE
	case string(model.StorageStateUnspecified), string(model.StorageStateUnspecified.ToV1()):
		return apiv1beta1.Experiment_STORAGESTATE_UNSPECIFIED
	default:
		return apiv1beta1.Experiment_STORAGESTATE_UNSPECIFIED
	}
}

// Converts API runtime state to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelRuntimeState(s interface{}) (model.RuntimeState, error) {
	if s == nil {
		return model.RuntimeStateUnspecified, nil
	}
	switch s := s.(type) {
	case string, *string:
		return model.RuntimeState(s.(string)), nil
	case apiv2beta1.RuntimeState, *apiv2beta1.RuntimeState:
		return toModelRuntimeState(apiv2beta1.RuntimeState_name[int32(s.(apiv2beta1.RuntimeState))])
	default:
		return "", util.NewUnknownApiVersionError("RuntimeState", s)
	}
}

// Converts internal runtime state representation to its API counterpart.
// Support v2beta1 API.
func toApiRuntimeState(s *model.RuntimeState) apiv2beta1.RuntimeState {
	return apiv2beta1.RuntimeState(apiv2beta1.RuntimeState_value[s.ToString()])
}

// Converts internal runtime state representation to its API counterpart.
// Support v1beta1 API by mapping v1beta1 API runtime states names.
func toApiRuntimeStateV1(s *model.RuntimeState) string {
	return string(s.ToV1())
}

// Converts API runtime status to its internal representation.
// Supports v2beta1 API.
func toModelRuntimeStatus(s *apiv2beta1.RuntimeStatus) (*model.RuntimeStatus, error) {
	if s == nil {
		return &model.RuntimeStatus{}, nil
	}
	state, err := toModelRuntimeState(s.GetState())
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert runtime status to its internal representation")
	}
	modelStatus := &model.RuntimeStatus{
		UpdateTimeInSec: s.GetUpdateTime().GetSeconds(),
		State:           state.ToV2(),
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
