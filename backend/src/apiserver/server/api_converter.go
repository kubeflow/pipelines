// Copyright 2018-2022 The Kubeflow Authors
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

	"github.com/golang/protobuf/ptypes/timestamp"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

func toApiRelationship(r model.Relationship) apiv1beta1.Relationship {
	switch r {
	case model.CreatorRelationship:
		return apiv1beta1.Relationship_CREATOR
	case model.OwnerRelationship:
		return apiv1beta1.Relationship_OWNER
	default:
		return apiv1beta1.Relationship_UNKNOWN_RELATIONSHIP
	}
}

func toApiTrigger(trigger model.Trigger) *apiv1beta1.Trigger {
	if trigger.Cron != nil && *trigger.Cron != "" {
		var cronSchedule apiv1beta1.CronSchedule
		cronSchedule.Cron = *trigger.Cron
		if trigger.CronScheduleStartTimeInSec != nil {
			cronSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleStartTimeInSec}
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleEndTimeInSec}
		}
		return &apiv1beta1.Trigger{Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}

	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
		var periodicSchedule apiv1beta1.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleStartTimeInSec}
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleEndTimeInSec}
		}
		return &apiv1beta1.Trigger{Trigger: &apiv1beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &periodicSchedule}}
	}
	return &apiv1beta1.Trigger{}
}

// Converts API Pipeline message into a Pipeline
func ToModelPipeline(p interface{}) (model.Pipeline, error) {
	var modelPipeline model.Pipeline
	switch p.(type) {
	case *apiv1beta1.Pipeline:
		p1 := p.(*apiv1beta1.Pipeline)
		namespace := GetNamespaceFromAPIResourceReferences(p1.GetResourceReferences())
		modelPipeline = model.Pipeline{
			UUID:           p1.GetId(),
			CreatedAtInSec: p1.GetCreatedAt().GetSeconds(),
			Name:           p1.GetName(),
			Description:    p1.GetDescription(),
			Namespace:      namespace,
		}
	case *apiv2beta1.Pipeline:
		p2 := p.(*apiv2beta1.Pipeline)
		modelPipeline = model.Pipeline{
			UUID:           p2.GetPipelineId(),
			CreatedAtInSec: p2.GetCreatedAt().GetSeconds(),
			Name:           p2.GetDisplayName(),
			Description:    p2.GetDescription(),
			Namespace:      p2.GetNamespace(),
		}
	default:
		return modelPipeline, util.NewUnknownApiVersionError("Pipeline", fmt.Sprintf("%v", p))
	}
	return modelPipeline, nil
}

// Converts API PipelineVersion message into PipelineVersion.
// Note: supports v1beta1 Pipeline to model's PipelineVersion conversion.
func ToModelPipelineVersion(p interface{}) (model.PipelineVersion, error) {
	var modelPipelineVersion model.PipelineVersion
	switch p.(type) {
	case *apiv1beta1.PipelineVersion:
		pv1 := p.(*apiv1beta1.PipelineVersion)
		modelPipelineVersion = model.PipelineVersion{
			UUID:           pv1.GetId(),
			CreatedAtInSec: pv1.GetCreatedAt().GetSeconds(),
			Name:           pv1.GetName(),
			Parameters:     ParametersToString(pv1.GetParameters()),
			PipelineId:     GetPipelineIdFromAPIResourceReferences(pv1.GetResourceReferences()),
			CodeSourceUrl:  pv1.GetCodeSourceUrl(),
			Description:    pv1.GetDescription(),
			PipelineSpec:   "",
		}
	case *apiv1beta1.Pipeline:
		p1 := p.(*apiv1beta1.Pipeline)
		pv1 := p1.DefaultVersion
		pipelineId := p1.GetId()
		if pipelineId == "" {
			pipelineId = GetPipelineIdFromAPIResourceReferences(pv1.GetResourceReferences())
		}
		params := ParametersToString(pv1.GetParameters())
		if params == "" {
			params = ParametersToString(p1.GetParameters())
		}
		modelPipelineVersion = model.PipelineVersion{
			UUID:           pv1.GetId(),
			CreatedAtInSec: pv1.GetCreatedAt().GetSeconds(),
			Name:           pv1.GetName(),
			Parameters:     params,
			PipelineId:     pipelineId,
			CodeSourceUrl:  pv1.GetCodeSourceUrl(),
			Description:    pv1.GetDescription(),
			PipelineSpec:   "",
		}
	case *apiv2beta1.PipelineVersion:
		pv2 := p.(*apiv2beta1.PipelineVersion)
		modelPipelineVersion = model.PipelineVersion{
			UUID:           pv2.GetPipelineVersionId(),
			CreatedAtInSec: pv2.GetCreatedAt().GetSeconds(),
			Name:           pv2.GetDisplayName(),
			PipelineId:     pv2.GetPipelineId(),
			CodeSourceUrl:  pv2.GetPackageUrl().GetPipelineUrl(),
			Description:    pv2.GetDescription(),
			PipelineSpec:   pv2.GetPipelineSpec().String(),
		}
	default:
		return modelPipelineVersion, util.NewUnknownApiVersionError("PipelineVersion", fmt.Sprintf("%v", p))
	}
	return modelPipelineVersion, nil
}

func ToModelResourceType(apiType apiv1beta1.ResourceType) (model.ResourceType, error) {
	switch apiType {
	case apiv1beta1.ResourceType_EXPERIMENT:
		return model.ExperimentResourceType, nil
	case apiv1beta1.ResourceType_JOB:
		return model.JobResourceType, nil
	case apiv1beta1.ResourceType_PIPELINE_VERSION:
		return model.PipelineVersionResourceType, nil
	case apiv1beta1.ResourceType_NAMESPACE:
		return model.NamespaceResourceType, nil
	default:
		return "", util.NewInvalidInputError("Unsupported resource type: %s", apiv1beta1.ResourceType_name[int32(apiType)])
	}
}

func ToModelRelationship(r apiv1beta1.Relationship) (model.Relationship, error) {
	switch r {
	case apiv1beta1.Relationship_CREATOR:
		return model.CreatorRelationship, nil
	case apiv1beta1.Relationship_OWNER:
		return model.OwnerRelationship, nil
	default:
		return "", util.NewInvalidInputError("Unsupported resource relationship: %s", apiv1beta1.Relationship_name[int32(r)])
	}
}

func GetNamespaceFromAPIResourceReferences(resourceRefs []*apiv1beta1.ResourceReference) string {
	namespace := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == apiv1beta1.ResourceType_NAMESPACE {
			namespace = resourceRef.Key.Id
			break
		}
	}
	return namespace
}

func GetPipelineIdFromAPIResourceReferences(resourceRefs []*apiv1beta1.ResourceReference) string {
	pipelineId := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == apiv1beta1.ResourceType_PIPELINE {
			pipelineId = resourceRef.Key.Id
			break
		}
	}
	return pipelineId
}

func GetExperimentIDFromAPIResourceReferences(resourceRefs []*apiv1beta1.ResourceReference) string {
	experimentID := ""
	for _, resourceRef := range resourceRefs {
		if resourceRef.Key.Type == apiv1beta1.ResourceType_EXPERIMENT {
			experimentID = resourceRef.Key.Id
			break
		}
	}
	return experimentID
}

func ToApiExperimentV1(experiment *model.Experiment) *apiv1beta1.Experiment {
	resourceReferences := []*apiv1beta1.ResourceReference(nil)
	if common.IsMultiUserMode() {
		resourceReferences = []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   experiment.Namespace,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		}
	}
	storageState := apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_UNSPECIFIED"])
	switch experiment.StorageState {
	case "AVAILABLE", "STORAGESTATE_AVAILABLE":
		storageState = apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"])
	case "ARCHIVED", "STORAGESTATE_ARCHIVED":
		storageState = apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"])
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

func ToApiExperimentsV1(experiments []*model.Experiment) []*apiv1beta1.Experiment {
	apiExperiments := make([]*apiv1beta1.Experiment, 0)
	for _, experiment := range experiments {
		apiExperiments = append(apiExperiments, ToApiExperimentV1(experiment))
	}
	return apiExperiments
}

func ToApiExperiment(experiment *model.Experiment) *apiv2beta1.Experiment {
	storageState := apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["STORAGESTATE_UNSPECIFIED"])
	switch experiment.StorageState {
	case "AVAILABLE", "STORAGESTATE_AVAILABLE":
		storageState = apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"])
	case "ARCHIVED", "STORAGESTATE_ARCHIVED":
		storageState = apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"])
	}

	return &apiv2beta1.Experiment{
		ExperimentId: experiment.UUID,
		DisplayName:  experiment.Name,
		Description:  experiment.Description,
		CreatedAt:    &timestamp.Timestamp{Seconds: experiment.CreatedAtInSec},
		Namespace:    experiment.Namespace,
		StorageState: storageState,
	}
}

func ToApiExperiments(experiments []*model.Experiment) []*apiv2beta1.Experiment {
	apiExperiments := make([]*apiv2beta1.Experiment, 0)
	for _, experiment := range experiments {
		apiExperiments = append(apiExperiments, ToApiExperiment(experiment))
	}
	return apiExperiments
}

// Coverts a pipeline into API Pipeline message
func ToApiPipelineV1(pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) *apiv1beta1.Pipeline {
	params, err := toApiParameters(pipelineVersion.Parameters)
	if err != nil {
		return &apiv1beta1.Pipeline{
			Id:    pipeline.UUID,
			Error: err.Error(),
		}
	}

	defaultVersion, err := ToApiPipelineVersionV1(pipelineVersion)
	if err != nil {
		return &apiv1beta1.Pipeline{
			Id:    pipeline.UUID,
			Error: err.Error(),
		}
	}

	var resourceRefs []*apiv1beta1.ResourceReference
	if len(pipeline.Namespace) > 0 {
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

	return &apiv1beta1.Pipeline{
		Id:                 pipeline.UUID,
		CreatedAt:          &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Name:               pipeline.Name,
		Description:        pipeline.Description,
		Parameters:         params,
		DefaultVersion:     defaultVersion,
		ResourceReferences: resourceRefs,
	}
}

// Converts a pipeline into API Pipeline message.
// Require the input pipeline to have UUID and CreateAt set to non-default values.
func ToApiPipeline(pipeline *model.Pipeline) *apiv2beta1.Pipeline {
	if pipeline.UUID == "" {
		return &apiv2beta1.Pipeline{
			PipelineId: "",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Empty pipeline id."),
					"Failed to convert a model pipeline to API Pipeline.",
				),
			),
		}
	}
	if pipeline.CreatedAtInSec == 0 {
		return &apiv2beta1.Pipeline{
			PipelineId: pipeline.UUID,
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Create time is missing."),
					"Failed to convert a model pipeline to API Pipeline.",
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

func ToApiPipelineVersionV1(version *model.PipelineVersion) (*apiv1beta1.PipelineVersion, error) {
	if version == nil {
		return nil, nil
	}
	params, err := toApiParameters(version.Parameters)
	if err != nil {
		return nil, err
	}

	return &apiv1beta1.PipelineVersion{
		Id:            version.UUID,
		Name:          version.Name,
		CreatedAt:     &timestamp.Timestamp{Seconds: version.CreatedAtInSec},
		Parameters:    params,
		Description:   version.Description,
		CodeSourceUrl: version.CodeSourceUrl,
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key: &apiv1beta1.ResourceKey{
					Id:   version.PipelineId,
					Type: apiv1beta1.ResourceType_PIPELINE,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}, nil
}

func ToApiPipelineVersionsV1(versions []*model.PipelineVersion) ([]*apiv1beta1.PipelineVersion, error) {
	apiVersions := make([]*apiv1beta1.PipelineVersion, 0)
	for _, version := range versions {
		v, _ := ToApiPipelineVersionV1(version)
		apiVersions = append(apiVersions, v)
	}
	return apiVersions, nil
}

func ToApiPipelinesV1(pipelines []*model.Pipeline, pipelineVersion []*model.PipelineVersion) []*apiv1beta1.Pipeline {
	apiPipelines := make([]*apiv1beta1.Pipeline, 0)
	for i, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, ToApiPipelineV1(pipeline, pipelineVersion[i]))
	}
	return apiPipelines
}

func toApiParameters(paramsString string) ([]*apiv1beta1.Parameter, error) {
	if paramsString == "" {
		return nil, nil
	}
	params, err := util.UnmarshalParameters(util.ArgoWorkflow, paramsString)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Parameter with wrong format is stored")
	}
	apiParams := make([]*apiv1beta1.Parameter, 0)
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
	return apiParams, nil
}

func toApiRuntimeConfig(modelRuntime model.RuntimeConfig) (*apiv1beta1.PipelineSpec_RuntimeConfig, error) {
	if modelRuntime.Parameters == "" && modelRuntime.PipelineRoot == "" {
		return nil, nil
	}
	var runtimeParams map[string]*structpb.Value
	if modelRuntime.Parameters != "" {
		err := json.Unmarshal([]byte(modelRuntime.Parameters), &runtimeParams)
		if err != nil {
			return nil, util.NewInternalServerError(err, fmt.Sprintf("Cannot unmarshal RuntimeConfig Parameter to map[string]*structpb.Value, string value: %+v", modelRuntime.Parameters))
		}
	}
	apiRuntimeConfig := &apiv1beta1.PipelineSpec_RuntimeConfig{
		Parameters:   runtimeParams,
		PipelineRoot: modelRuntime.PipelineRoot,
	}
	return apiRuntimeConfig, nil
}

func toApiRun(run *model.Run) *apiv1beta1.Run {
	// v1 parameters
	params, err := toApiParameters(run.Parameters)
	if err != nil {
		return &apiv1beta1.Run{
			Id:    run.UUID,
			Error: err.Error(),
		}
	}
	// v2 RuntimeConfig
	runtimeConfig, err := toApiRuntimeConfig(run.PipelineSpec.RuntimeConfig)
	if err != nil {
		return &apiv1beta1.Run{
			Id:    run.UUID,
			Error: err.Error(),
		}
	}
	var metrics []*apiv1beta1.RunMetric
	if run.Metrics != nil {
		for _, metric := range run.Metrics {
			metrics = append(metrics, ToApiRunMetric(metric))
		}
	}
	return &apiv1beta1.Run{
		CreatedAt:      &timestamp.Timestamp{Seconds: run.CreatedAtInSec},
		Id:             run.UUID,
		Metrics:        metrics,
		Name:           run.DisplayName,
		ServiceAccount: run.ServiceAccount,
		StorageState:   apiv1beta1.Run_StorageState(apiv1beta1.Run_StorageState_value[run.StorageState]),
		Description:    run.Description,
		ScheduledAt:    &timestamp.Timestamp{Seconds: run.ScheduledAtInSec},
		FinishedAt:     &timestamp.Timestamp{Seconds: run.FinishedAtInSec},
		Status:         run.Conditions,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       run.PipelineId,
			PipelineName:     run.PipelineName,
			WorkflowManifest: run.WorkflowSpecManifest,
			PipelineManifest: run.PipelineSpecManifest,
			Parameters:       params,
			RuntimeConfig:    runtimeConfig,
		},
		ResourceReferences: toApiResourceReferences(run.ResourceReferences),
	}
}

func ToApiRuns(runs []*model.Run) []*apiv1beta1.Run {
	apiRuns := make([]*apiv1beta1.Run, 0)
	for _, run := range runs {
		apiRuns = append(apiRuns, toApiRun(run))
	}
	return apiRuns
}

func ToApiRunDetail(run *model.RunDetail) *apiv1beta1.RunDetail {
	return &apiv1beta1.RunDetail{
		Run: toApiRun(&run.Run),
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: run.WorkflowRuntimeManifest,
			PipelineManifest: run.PipelineRuntimeManifest,
		},
	}
}

func ToApiTask(task *model.Task) *apiv1beta1.Task {
	return &apiv1beta1.Task{
		Id:              task.UUID,
		Namespace:       task.Namespace,
		PipelineName:    task.PipelineName,
		RunId:           task.RunUUID,
		MlmdExecutionID: task.MLMDExecutionID,
		CreatedAt:       &timestamp.Timestamp{Seconds: task.CreatedTimestamp},
		FinishedAt:      &timestamp.Timestamp{Seconds: task.FinishedTimestamp},
		Fingerprint:     task.Fingerprint,
	}
}

func ToApiTasks(tasks []*model.Task) []*apiv1beta1.Task {
	apiTasks := make([]*apiv1beta1.Task, 0)
	for _, task := range tasks {
		apiTasks = append(apiTasks, ToApiTask(task))
	}
	return apiTasks
}

func ToApiJob(job *model.Job) *apiv1beta1.Job {
	// v1 parameters
	params, err := toApiParameters(job.Parameters)
	if err != nil {
		return &apiv1beta1.Job{
			Id:    job.UUID,
			Error: err.Error(),
		}
	}
	// v2 RuntimeConfig
	runtimeConfig, err := toApiRuntimeConfig(job.PipelineSpec.RuntimeConfig)
	if err != nil {
		return &apiv1beta1.Job{
			Id:    job.UUID,
			Error: err.Error(),
		}
	}
	return &apiv1beta1.Job{
		Id:             job.UUID,
		Name:           job.DisplayName,
		ServiceAccount: job.ServiceAccount,
		Description:    job.Description,
		Enabled:        job.Enabled,
		CreatedAt:      &timestamp.Timestamp{Seconds: job.CreatedAtInSec},
		UpdatedAt:      &timestamp.Timestamp{Seconds: job.UpdatedAtInSec},
		Status:         job.Conditions,
		MaxConcurrency: job.MaxConcurrency,
		NoCatchup:      job.NoCatchup,
		Trigger:        toApiTrigger(job.Trigger),
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       job.PipelineId,
			PipelineName:     job.PipelineName,
			WorkflowManifest: job.WorkflowSpecManifest,
			PipelineManifest: job.PipelineSpecManifest,
			Parameters:       params,
			RuntimeConfig:    runtimeConfig,
		},
		ResourceReferences: toApiResourceReferences(job.ResourceReferences),
	}
}

func ToApiJobs(jobs []*model.Job) []*apiv1beta1.Job {
	apiJobs := make([]*apiv1beta1.Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, ToApiJob(job))
	}
	return apiJobs
}

func ToApiRunMetric(metric *model.RunMetric) *apiv1beta1.RunMetric {
	return &apiv1beta1.RunMetric{
		Name:   metric.Name,
		NodeId: metric.NodeID,
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: metric.NumberValue,
		},
		Format: apiv1beta1.RunMetric_Format(apiv1beta1.RunMetric_Format_value[metric.Format]),
	}
}

func toApiResourceReferences(references []*model.ResourceReference) []*apiv1beta1.ResourceReference {
	var apiReferences []*apiv1beta1.ResourceReference
	for _, ref := range references {
		apiReferences = append(apiReferences, &apiv1beta1.ResourceReference{
			Key: &apiv1beta1.ResourceKey{
				Type: toApiResourceType(ref.ReferenceType),
				Id:   ref.ReferenceUUID,
			},
			Name:         ref.ReferenceName,
			Relationship: toApiRelationship(ref.Relationship),
		})
	}
	return apiReferences
}

func toApiResourceType(modelType model.ResourceType) apiv1beta1.ResourceType {
	switch modelType {
	case model.ExperimentResourceType:
		return apiv1beta1.ResourceType_EXPERIMENT
	case model.JobResourceType:
		return apiv1beta1.ResourceType_JOB
	case model.PipelineVersionResourceType:
		return apiv1beta1.ResourceType_PIPELINE_VERSION
	case model.NamespaceResourceType:
		return apiv1beta1.ResourceType_NAMESPACE
	default:
		return apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE
	}
}

// Converts v1beta1 Parameters into a string
func ParametersToString(p []*apiv1beta1.Parameter) string {
	newParameters := ""
	newParameterBytes, err := json.Marshal(p)
	if err == nil {
		newParameters = string(newParameterBytes)
	}
	return newParameters
}

func ToModelExperiment(inputExperiment interface{}) (*model.Experiment, error) {
	namespace := ""
	name := ""
	description := ""
	switch inputExperiment.(type) {
	case *apiv1beta1.Experiment:
		v1Experiment := inputExperiment.(*apiv1beta1.Experiment)
		name = v1Experiment.GetName()
		description = v1Experiment.GetDescription()
		resourceReferences := v1Experiment.GetResourceReferences()
		if resourceReferences != nil {
			if len(resourceReferences) != 1 ||
				resourceReferences[0].Key.Type != apiv1beta1.ResourceType_NAMESPACE ||
				resourceReferences[0].Relationship != apiv1beta1.Relationship_OWNER {
				return nil, util.NewInternalServerError(errors.New("Invalid resource references for experiment"), "Unable to convert to model experiment.")
			}
			namespace = resourceReferences[0].Key.Id
		}
	case *apiv2beta1.Experiment:
		v2Experiment := inputExperiment.(*apiv2beta1.Experiment)
		name = v2Experiment.GetDisplayName()
		namespace = v2Experiment.GetNamespace()
		description = v2Experiment.GetDescription()
	default:
		return nil, util.NewInternalServerError(errors.New("Invalid experiment type"), "Unable to convert to model experiment.")
	}
	return &model.Experiment{
		Name:        name,
		Description: description,
		Namespace:   namespace,
	}, nil
}

func ToModelRunMetric(metric *apiv1beta1.RunMetric, runUUID string) *model.RunMetric {
	return &model.RunMetric{
		RunUUID:     runUUID,
		Name:        metric.GetName(),
		NodeID:      metric.GetNodeId(),
		NumberValue: metric.GetNumberValue(),
		Format:      metric.GetFormat().String(),
	}
}

func modeToModelEnabled(v2APIMode apiv2beta1.RecurringRun_Mode) bool {
	// Returns false if status is disable or unspecified.
	if v2APIMode == apiv2beta1.RecurringRun_ENABLE {
		return true
	} else {
		return false
	}
}

func toModelTriggerV2(v2APITrigger *apiv2beta1.Trigger) (model.Trigger, error) {
	modelTrigger := model.Trigger{}
	if v2APITrigger == nil {
		return modelTrigger, nil
	}
	if v2APITrigger.GetCronSchedule() != nil {
		cronSchedule := v2APITrigger.GetCronSchedule()
		modelTrigger.CronSchedule = model.CronSchedule{Cron: &cronSchedule.Cron}
		if cronSchedule.StartTime != nil {
			modelTrigger.CronScheduleStartTimeInSec = &cronSchedule.StartTime.Seconds
		}
		if cronSchedule.EndTime != nil {
			modelTrigger.CronScheduleEndTimeInSec = &cronSchedule.EndTime.Seconds
		}
	}

	if v2APITrigger.GetPeriodicSchedule() != nil {
		periodicSchedule := v2APITrigger.GetPeriodicSchedule()
		modelTrigger.PeriodicSchedule = model.PeriodicSchedule{
			IntervalSecond: &periodicSchedule.IntervalSecond}
		if v2APITrigger.GetPeriodicSchedule().StartTime != nil {
			modelTrigger.PeriodicScheduleStartTimeInSec = &periodicSchedule.StartTime.Seconds
		}
		if v2APITrigger.GetPeriodicSchedule().EndTime != nil {
			modelTrigger.PeriodicScheduleEndTimeInSec = &periodicSchedule.EndTime.Seconds
		}
	}
	return modelTrigger, nil
}

func toModelTriggerV1(trigger *apiv1beta1.Trigger) model.Trigger {
	modelTrigger := model.Trigger{}
	if trigger == nil {
		return modelTrigger
	}
	if trigger.GetCronSchedule() != nil {
		cronSchedule := trigger.GetCronSchedule()
		modelTrigger.CronSchedule = model.CronSchedule{Cron: &cronSchedule.Cron}
		if cronSchedule.StartTime != nil {
			modelTrigger.CronScheduleStartTimeInSec = &cronSchedule.StartTime.Seconds
		}
		if cronSchedule.EndTime != nil {
			modelTrigger.CronScheduleEndTimeInSec = &cronSchedule.EndTime.Seconds
		}
	}

	if trigger.GetPeriodicSchedule() != nil {
		periodicSchedule := trigger.GetPeriodicSchedule()
		modelTrigger.PeriodicSchedule = model.PeriodicSchedule{
			IntervalSecond: &periodicSchedule.IntervalSecond}
		if trigger.GetPeriodicSchedule().StartTime != nil {
			modelTrigger.PeriodicScheduleStartTimeInSec = &periodicSchedule.StartTime.Seconds
		}
		if trigger.GetPeriodicSchedule().EndTime != nil {
			modelTrigger.PeriodicScheduleEndTimeInSec = &periodicSchedule.EndTime.Seconds
		}
	}
	return modelTrigger
}

func apiParametersToModelParameters(apiParams []*apiv1beta1.Parameter) (string, error) {
	if apiParams == nil || len(apiParams) == 0 {
		return "", nil
	}
	var params util.SpecParameters
	for _, apiParam := range apiParams {
		param := util.SpecParameter{
			Name:  apiParam.Name,
			Value: util.StringPointer(apiParam.Value),
		}
		params = append(params, param)
	}
	paramsBytes, err := util.MarshalParameters(util.ArgoWorkflow, params)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to stream API parameter as string.")
	}
	return string(paramsBytes), nil
}

func runtimeConfigToModelParametersV1(runtimeConfig *apiv1beta1.PipelineSpec_RuntimeConfig) (string, error) {
	if runtimeConfig == nil {
		return "", nil
	}
	paramsBytes, err := json.Marshal(runtimeConfig.GetParameters())
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to marshal RuntimeConfig API parameters as string.")
	}
	return string(paramsBytes), nil
}

func runtimeConfigToModelParameters(runtimeConfig *apiv2beta1.RuntimeConfig) (string, error) {
	if runtimeConfig == nil {
		return "", nil
	}
	paramsBytes, err := json.Marshal(runtimeConfig.GetParameters())
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to marshal RuntimeConfig API parameters as string.")
	}
	return string(paramsBytes), nil
}
