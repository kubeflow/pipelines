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
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
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
		CreatedAt:          timestamppb.New(time.Unix(experiment.CreatedAtInSec, 0)),
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
		CreatedAt:        timestamppb.New(time.Unix(experiment.CreatedAtInSec, 0)),
		LastRunCreatedAt: timestamppb.New(time.Unix(experiment.LastRunCreatedAtInSec, 0)),
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
	var name, displayName, namespace, description string

	switch apiPipeline := p.(type) {
	case *apiv1beta1.Pipeline:
		namespace = getNamespaceFromResourceReferenceV1(apiPipeline.GetResourceReferences())
		name = apiPipeline.GetName()
		displayName = name
		description = apiPipeline.GetDescription()
	case *apiv2beta1.Pipeline:
		namespace = apiPipeline.GetNamespace()
		name = apiPipeline.GetName()
		displayName = apiPipeline.GetDisplayName()
		description = apiPipeline.GetDescription()
	default:
		return nil, util.NewUnknownApiVersionError("Pipeline", p)
	}

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
	}

	if err := validation.ValidateModel(pipeline); err != nil {
		return nil, util.NewInternalServerError(
			err,
			"Failed to convert API pipeline to model pipeline",
		)
	}

	return pipeline, nil

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

	params := toApiParametersV1(string(pipelineVersion.Parameters))
	if params == nil {
		return &apiv1beta1.Pipeline{
			Id:    pipeline.UUID,
			Error: util.NewInternalServerError(util.NewInvalidInputError("%s", fmt.Sprintf("Failed to convert parameters: %s", pipelineVersion.Parameters)), "Failed to convert a model pipeline to v1beta1 API pipeline").Error(),
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
		CreatedAt:          timestamppb.New(time.Unix(pipeline.CreatedAtInSec, 0)),
		Name:               pipeline.Name,
		Description:        string(pipeline.Description),
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
		Name:        pipeline.Name,
		DisplayName: pipeline.DisplayName,
		Description: string(pipeline.Description),
		CreatedAt:   timestamppb.New(time.Unix(pipeline.CreatedAtInSec, 0)),
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
	var name, displayName, description, pipelineId, pipelineUrl, codeUrl string
	switch p := p.(type) {
	case *apiv1beta1.PipelineVersion:
		apiPipelineVersionV1 := p
		if apiPipelineVersionV1.GetPackageUrl() == nil || len(apiPipelineVersionV1.GetPackageUrl().GetPipelineUrl()) == 0 {
			return nil, util.NewInvalidInputError("Failed to convert v1beta1 API pipeline version to its internal representation due to missing pipeline URL")
		}
		pipelineUrl = apiPipelineVersionV1.GetPackageUrl().GetPipelineUrl()
		codeUrl = apiPipelineVersionV1.GetCodeSourceUrl()
		name = apiPipelineVersionV1.GetName()
		displayName = name
		pipelineId = getPipelineIdFromResourceReferencesV1(apiPipelineVersionV1.GetResourceReferences())
		description = apiPipelineVersionV1.GetDescription()
	case *apiv2beta1.PipelineVersion:
		apiPipelineVersionV2 := p
		if apiPipelineVersionV2.GetPackageUrl() == nil || len(apiPipelineVersionV2.GetPackageUrl().GetPipelineUrl()) == 0 {
			return nil, util.NewInvalidInputError("Failed to convert v2beta1 API pipeline version to its internal representation due to missing pipeline URL")
		}
		name = apiPipelineVersionV2.GetName()
		displayName = apiPipelineVersionV2.GetDisplayName()
		pipelineId = apiPipelineVersionV2.GetPipelineId()
		pipelineUrl = apiPipelineVersionV2.GetPackageUrl().GetPipelineUrl()
		codeUrl = apiPipelineVersionV2.GetCodeSourceUrl()
		description = apiPipelineVersionV2.GetDescription()
	default:
		return nil, util.NewUnknownApiVersionError("PipelineVersion", p)
	}

	// Previously display_name was the required API field and name didn't exist. Name is now the required API field
	// but if display_name is provided, we use it as the name.
	if name == "" {
		name = displayName
	}
	// If display_name is not provided, we use the name as the display_name for backward compatibility and convenience.
	if displayName == "" {
		displayName = name
	}
	pv := &model.PipelineVersion{
		Name:            name,
		DisplayName:     displayName,
		PipelineId:      pipelineId,
		PipelineSpecURI: model.LargeText(pipelineUrl),
		CodeSourceUrl:   codeUrl,
		Description:     model.LargeText(description),
		Status:          model.PipelineVersionCreating,
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
// Supports v1beta1 API.
// Note: does not return an error along with the result anymore. Check if the result is nil instead.
func toApiPipelineVersionV1(pv *model.PipelineVersion) *apiv1beta1.PipelineVersion {
	apiPipelineVersion := &apiv1beta1.PipelineVersion{}
	if pv == nil {
		return apiPipelineVersion
	}
	apiPipelineVersion.Id = pv.UUID
	apiPipelineVersion.Name = pv.Name
	apiPipelineVersion.CreatedAt = timestamppb.New(time.Unix(pv.CreatedAtInSec, 0))
	if p := toApiParametersV1(string(pv.Parameters)); p == nil {
		return nil
	} else if len(p) > 0 {
		apiPipelineVersion.Parameters = p
	}
	apiPipelineVersion.Description = string(pv.Description)
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
		Name:              pv.Name,
		DisplayName:       pv.DisplayName,
		Description:       string(pv.Description),
		CreatedAt:         timestamppb.New(time.Unix(pv.CreatedAtInSec, 0)),
	}

	// Infer pipeline url
	if pv.CodeSourceUrl != "" {
		apiPipelineVersion.PackageUrl = &apiv2beta1.Url{
			PipelineUrl: string(pv.PipelineSpecURI),
		}
	} else if pv.PipelineSpecURI != "" {
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
			cronSchedule.StartTime = timestamppb.New(time.Unix(*trigger.CronScheduleStartTimeInSec, 0))
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = timestamppb.New(time.Unix(*trigger.CronScheduleEndTimeInSec, 0))
		}
		return &apiv1beta1.Trigger{Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}
	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
		var periodicSchedule apiv1beta1.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = timestamppb.New(time.Unix(*trigger.PeriodicScheduleStartTimeInSec, 0))
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = timestamppb.New(time.Unix(*trigger.PeriodicScheduleEndTimeInSec, 0))
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
		Parameters:   model.LargeText(params),
		PipelineRoot: model.LargeText(root),
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
	runtimeParams := toMapProtoStructParameters(string(modelRuntime.Parameters))
	if runtimeParams == nil {
		return nil
	}
	apiRuntimeConfig.Parameters = runtimeParams
	apiRuntimeConfig.PipelineRoot = string(modelRuntime.PipelineRoot)
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
	runtimeParams := toMapProtoStructParameters(string(modelRuntime.Parameters))
	if runtimeParams == nil {
		return nil
	}
	apiRuntimeConfig.Parameters = runtimeParams
	apiRuntimeConfig.PipelineRoot = string(modelRuntime.PipelineRoot)
	return &apiRuntimeConfig
}

// Converts API run metric to its internal representation.
// Supports both v1beta1 and v2beta1 API.
func toModelRunMetricV1(m interface{}, runID string) (*model.RunMetricV1, error) {
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
	modelMetric := &model.RunMetricV1{
		RunUUID:     runID,
		Name:        name,
		NodeID:      nodeId,
		NumberValue: val,
		Format:      format,
	}
	if err := validation.ValidateModel(modelMetric); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to convert API run metric to internal representation")
	}
	return modelMetric, nil

}

// Converts internal run metric representation to its API counterpart.
// Supports v1beta1 API.
func toAPIRunMetricV1(metric *model.RunMetricV1) *apiv1beta1.RunMetric {
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
func toAPIRunMetricsV1(m []*model.RunMetricV1) []*apiv1beta1.RunMetric {
	apiMetrics := make([]*apiv1beta1.RunMetric, 0)
	for _, metric := range m {
		apiMetrics = append(apiMetrics, toAPIRunMetricV1(metric))
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
	var modelMetrics []*model.RunMetricV1
	var modelTasks []*model.Task
	var state model.RuntimeState
	var stateHistory []*model.RuntimeStatus
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
			modelMetrics = make([]*model.RunMetricV1, 0)
			for _, metric := range apiRunV1.GetMetrics() {
				modelMetric, err := toModelRunMetricV1(metric, runId)
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
		cfgParams = string(cfg.Parameters)
		pipelineRoot = string(cfg.PipelineRoot)

		pipelineSpec = apiRunV1.GetPipelineSpec().GetPipelineManifest()
		workflowSpec = apiRunV1.GetPipelineSpec().GetWorkflowManifest()
		serviceAcc = apiRunV1.GetServiceAccount()
	case *apiv2beta1.Run:
		apiRunV2 := r
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
// Supports v1beta1 API.
// Note: adds error details to the message if a parsing error occurs.
func toApiRunV1(r *model.Run) *apiv1beta1.Run {
	r = r.ToV1()
	// v1 parameters
	specParams := toApiParametersV1(string(r.Parameters))
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
		metrics = toAPIRunMetricsV1(r.Metrics)
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
		CreatedAt:      timestamppb.New(time.Unix(r.CreatedAtInSec, 0)),
		Id:             r.UUID,
		Metrics:        metrics,
		Name:           r.DisplayName,
		ServiceAccount: r.ServiceAccount,
		StorageState:   apiv1beta1.Run_StorageState(apiv1beta1.Run_StorageState_value[string(r.StorageState.ToV1())]),
		Description:    r.Description,
		ScheduledAt:    timestamppb.New(time.Unix(r.ScheduledAtInSec, 0)),
		FinishedAt:     timestamppb.New(time.Unix(r.FinishedAtInSec, 0)),
		Status:         string(r.RunDetails.State.ToV1()),
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       r.PipelineSpec.PipelineId,
			PipelineName:     r.PipelineSpec.PipelineName,
			WorkflowManifest: string(wfManifest),
			PipelineManifest: string(specManifest),
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
		if params := toMapProtoStructParameters(string(r.Parameters)); len(params) > 0 {
			runtimeConfig.Parameters = params
		} else {
			runtimeConfig = nil
		}
	}
	apiTasks, err := generateAPITasks(r.Tasks)
	if err != nil {
		return &apiv2beta1.Run{
			RunId:        r.UUID,
			ExperimentId: r.ExperimentId,
			Error:        util.ToRpcStatus(err),
		}
	}
	if len(apiTasks) == 0 {
		apiTasks = nil
	}

	apiRd := &apiv2beta1.RunDetails{
		PipelineContextId:    r.RunDetails.PipelineContextId,
		PipelineRunContextId: r.RunDetails.PipelineRunContextId,
	}
	if apiRd.PipelineContextId == 0 && apiRd.PipelineRunContextId == 0 && apiRd.TaskDetails == nil {
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

	err = util.NewInvalidInputError("Failed to parse the pipeline source")
	if r.PipelineSpec.PipelineVersionId != "" {
		apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId:        r.PipelineSpec.PipelineId,
				PipelineVersionId: r.PipelineSpec.PipelineVersionId,
			},
		}
		return apiRunV2
	} else if r.PipelineSpec.PipelineSpecManifest != "" {
		spec, err1 := YamlStringToPipelineSpecStruct(string(r.PipelineSpecManifest))
		if err1 == nil {
			apiRunV2.PipelineSource = &apiv2beta1.Run_PipelineSpec{
				PipelineSpec: spec,
			}
			return apiRunV2
		}
		err = util.Wrap(err1, err.Error()).(*util.UserError)
	} else if r.PipelineSpec.WorkflowSpecManifest != "" {
		spec, err1 := YamlStringToPipelineSpecStruct(string(r.WorkflowSpecManifest))
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

func generateAPITasks(tasks []*model.Task) ([]*apiv2beta1.PipelineTaskDetail, error) {
	// Create map to store parent->children relationships
	taskMap := make(map[string]*model.Task)
	childrenMap := make(map[string][]*model.Task)

	// Build maps of tasks and parent->children relationships
	for _, task := range tasks {
		taskMap[task.UUID] = task
		if task.ParentTaskUUID != nil {
			childrenMap[*task.ParentTaskUUID] = append(childrenMap[*task.ParentTaskUUID], task)
		}
	}

	// Convert each task to API format, building child task info if it has children
	apiTasks := make([]*apiv2beta1.PipelineTaskDetail, 0)
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
			PipelineManifest: string(r.PipelineRuntimeManifest),
		}
	} else if r.RunDetails.PipelineRuntimeManifest == "" {
		apiRunDetails.PipelineRuntime = &apiv1beta1.PipelineRuntime{
			WorkflowManifest: string(r.WorkflowRuntimeManifest),
		}
	} else {
		apiRunDetails.PipelineRuntime = &apiv1beta1.PipelineRuntime{
			PipelineManifest: string(r.PipelineRuntimeManifest),
			WorkflowManifest: string(r.WorkflowRuntimeManifest),
		}
	}
	return apiRunDetails
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
		cfgParams = string(cfg.Parameters)
		pipelineRoot = string(cfg.PipelineRoot)

		pipelineSpec = apiJob.GetPipelineSpec().GetPipelineManifest()
		workflowSpec = apiJob.GetPipelineSpec().GetWorkflowManifest()
		k8sName = jobName
	case *apiv2beta1.RecurringRun:
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
		ResourceReferences: resRefs,
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
	specParams := toApiParametersV1(string(j.Parameters))
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
		CreatedAt:      timestamppb.New(time.Unix(j.CreatedAtInSec, 0)),
		Status:         toApiJobStatus(j.Conditions),
		UpdatedAt:      timestamppb.New(time.Unix(j.UpdatedAtInSec, 0)),
		MaxConcurrency: j.MaxConcurrency,
		NoCatchup:      j.NoCatchup,
		Trigger:        trigger,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       j.PipelineSpec.PipelineId,
			PipelineName:     j.PipelineSpec.PipelineName,
			WorkflowManifest: string(wfManifest),
			PipelineManifest: string(specManifest),
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

	return modelAT, nil
}

// Converts API PipelineTaskDetail to its internal representation.
// Supports v2beta1 API.
// Note that InputArtifactsHydrated and OutputArtifactsHydrated are not converted as these
// are not stored in DB, and to fill them out would require additional DB queries to fetch Artifacts values.
func toModelTask(apiTask *apiv2beta1.PipelineTaskDetail) (*model.Task, error) {
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

	// Convert inputs: store full InputOutputs in InputParameters and artifacts subset in InputArtifacts
	if apiTask.GetInputs() != nil {
		if apiTask.GetInputs().GetParameters() != nil {
			parameters, err := model.ProtoSliceToJSONSlice(apiTask.GetInputs().GetParameters())
			if err != nil {
				return nil, err
			}
			task.InputParameters = parameters
		}
	}

	// Convert outputs: store full InputOutputs in OutputParameters and artifacts subset in OutputArtifacts
	if apiTask.GetOutputs() != nil {
		if apiTask.GetOutputs().GetParameters() != nil {
			artifacts, err := model.ProtoSliceToJSONSlice(apiTask.GetOutputs().GetParameters())
			if err != nil {
				return nil, err
			}
			task.OutputParameters = artifacts
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
// they must be provided as an argument.
func toAPITask(modelTask *model.Task, childTasks []*model.Task) (*apiv2beta1.PipelineTaskDetail, error) {
	if modelTask == nil {
		return nil, util.NewInvalidInputError("Task cannot be nil")
	}

	apiTask := &apiv2beta1.PipelineTaskDetail{
		TaskId:           modelTask.UUID,
		RunId:            modelTask.RunUUID,
		ParentTaskId:     modelTask.ParentTaskUUID,
		Name:             modelTask.Name,
		DisplayName:      modelTask.DisplayName,
		CacheFingerprint: modelTask.Fingerprint,
		Inputs:           &apiv2beta1.PipelineTaskDetail_InputOutputs{},
		Outputs:          &apiv2beta1.PipelineTaskDetail_InputOutputs{},
	}

	// Convert timestamps
	if modelTask.CreatedAtInSec > 0 {
		apiTask.CreateTime = &timestamppb.Timestamp{Seconds: modelTask.CreatedAtInSec}
	}
	if modelTask.FinishedInSec > 0 {
		apiTask.EndTime = &timestamppb.Timestamp{Seconds: modelTask.FinishedInSec}
	}

	// Convert status
	apiTask.State = apiv2beta1.PipelineTaskDetail_TaskState(modelTask.State)

	// Convert task type
	apiTask.Type = apiv2beta1.PipelineTaskDetail_TaskType(modelTask.Type)

	// Set pod name from the first pod in PodNames array
	if modelTask.Pods != nil {
		apiPods, err := model.JSONSliceToProtoSlice(
			modelTask.Pods,
			func() *apiv2beta1.PipelineTaskDetail_TaskPod {
				return &apiv2beta1.PipelineTaskDetail_TaskPod{}
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
			func() *apiv2beta1.PipelineTaskDetail_StatusMetadata {
				return &apiv2beta1.PipelineTaskDetail_StatusMetadata{}
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
			func() *apiv2beta1.PipelineTaskDetail_TaskStatus {
				return &apiv2beta1.PipelineTaskDetail_TaskStatus{}
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
			func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
				return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
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
			func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
				return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.Outputs.Parameters = apiOutputParams
	}

	// Populate artifacts from hydrated fields on the model task with shared converter
	convertHydrated := func(in []model.TaskArtifactHydrated) ([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
		if len(in) == 0 {
			return nil, nil
		}

		// Group artifacts by (ArtifactKey, Type, Producer) to consolidate metrics
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
				if h.Producer.Iteration != nil {
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

		// Convert grouped artifacts to IOArtifacts
		out := make([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, 0, len(grouped))
		for _, hydratedGroup := range grouped {
			// Check if all artifacts in this group are metrics
			allMetrics := true
			for _, h := range hydratedGroup {
				if h.Value == nil || h.Value.Type != model.ArtifactType(apiv2beta1.Artifact_Metric) {
					allMetrics = false
					break
				}
			}

			if allMetrics && len(hydratedGroup) > 1 {
				// Multiple metrics with same key - consolidate into ONE IOArtifact with multiple artifacts
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

				// Use first hydrated entry for common fields
				firstHydrated := hydratedGroup[0]
				ioArtifact := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
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
			} else {
				// Non-metrics or single artifact - one IOArtifact per artifact
				for _, h := range hydratedGroup {
					var apiArt *apiv2beta1.Artifact
					if h.Value != nil {
						apiArtConv, err := toAPIArtifact(h.Value)
						if err != nil {
							return nil, err
						}
						apiArt = apiArtConv
					}
					ioArtifact := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
						Artifacts:   []*apiv2beta1.Artifact{apiArt},
						ArtifactKey: h.Key,
						Type:        h.Type,
					}
					if h.Producer != nil {
						ioArtifact.Producer = &apiv2beta1.IOProducer{
							TaskName:  h.Producer.TaskName,
							Iteration: h.Producer.Iteration,
						}
					}
					out = append(out, ioArtifact)
				}
			}
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
			func() *apiv2beta1.PipelineTaskDetail_TypeAttributes {
				return &apiv2beta1.PipelineTaskDetail_TypeAttributes{}
			})
		if err != nil {
			return nil, err
		}
		apiTask.TypeAttributes = apiTypeAttrs
	}

	// Convert child tasks
	apiChildTasks := make([]*apiv2beta1.PipelineTaskDetail_ChildTask, 0)
	for _, childTask := range childTasks {
		apiChildTask := &apiv2beta1.PipelineTaskDetail_ChildTask{
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
