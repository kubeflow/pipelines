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

package resource

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/template"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

func (r *ResourceManager) ToModelExperiment(inputExperiment interface{}) (*model.Experiment, error) {
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

func (r *ResourceManager) ToModelRunMetric(metric *apiv1beta1.RunMetric, runUUID string) *model.RunMetric {
	return &model.RunMetric{
		RunUUID:     runUUID,
		Name:        metric.GetName(),
		NodeID:      metric.GetNodeId(),
		NumberValue: metric.GetNumberValue(),
		Format:      metric.GetFormat().String(),
	}
}

func (r *ResourceManager) ToModelRunDetail(runInterface interface{}, runId string, runAt int64, manifest string, templateType template.TemplateType) (*model.RunDetail, error) {
	switch runInterface.(type) {
	case *apiv1beta1.Run:
		return r.ToModelRunDetailV1(runInterface, runId, runAt, manifest, templateType)
	case *apiv2beta1.Run:
		return r.ToModelRunDetailV2(runInterface, runId, runAt, manifest, templateType)
	default:
		return nil, util.NewInvalidInputError("Invalid Api run type.")
	}
}

// The input run might not contain workflowSpecManifest and pipelineSpecManifest, but instead a pipeline ID.
// The caller would retrieve manifest and pass in.
func (r *ResourceManager) ToModelRunDetailV1(runInterface interface{}, runId string, runAt int64, manifest string, templateType template.TemplateType) (*model.RunDetail, error) {
	run := runInterface.(*apiv1beta1.Run)
	var pipelineName string
	var err error
	if run.GetPipelineSpec().GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, run.GetPipelineSpec().GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}

	// Add a reference to the default experiment if run does not already have a containing experiment
	ref, err := r.getDefaultExperimentIfNoExperiment(run.GetResourceReferences())
	if err != nil {
		return nil, err
	}
	if ref != nil {
		run.ResourceReferences = append(run.GetResourceReferences(), ref)
	}

	// Convert api ResourceReferences to Model ResourceReferences.
	resourceReferences, err := r.toModelResourceReferences(runId, common.Run, run.GetResourceReferences())
	if err != nil {
		return nil, util.Wrap(err, "Unable to convert resource references.")
	}

	experimentUUID, err := r.getOwningExperimentUUID(run.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Error getting the experiment UUID")
	}

	namespace, err := r.getNamespaceFromExperiment(run.GetResourceReferences())
	if err != nil {
		return nil, err
	}

	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:               runId,
			ExperimentUUID:     experimentUUID,
			Namespace:          namespace,
			Name:               run.Name,
			DisplayName:        run.Name,
			ServiceAccount:     run.ServiceAccount,
			Description:        run.Description,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:   run.GetPipelineSpec().GetPipelineId(),
				PipelineName: pipelineName,
			},
		},
	}

	// Assign the scheduled at time
	if !run.ScheduledAt.AsTime().IsZero() {
		// if there is no scheduled time, then we assume this run is scheduled at the same time it is created
		runDetail.ScheduledAtInSec = runAt
	} else {
		runDetail.ScheduledAtInSec = run.ScheduledAt.AsTime().Unix()
	}

	if templateType == template.V1 {
		// Input template if of V1 type (argo)
		params, err := apiParametersToModelParameters(run.GetPipelineSpec().GetParameters())
		if err != nil {
			return nil, util.Wrap(err, "Unable to parse the V1 parameter.")
		}
		runDetail.Parameters = params
		runDetail.WorkflowSpecManifest = manifest
		return runDetail, nil

	} else if templateType == template.V2 {
		// Input template if of V2 type (IR)
		params, err := runtimeConfigToModelParametersV1(run.GetPipelineSpec().GetRuntimeConfig())
		if err != nil {
			return nil, util.Wrap(err, "Unable to parse the V2 parameter.")
		}
		runDetail.PipelineSpecManifest = manifest
		runDetail.PipelineSpec.RuntimeConfig.Parameters = params
		runDetail.PipelineSpec.RuntimeConfig.PipelineRoot = run.GetPipelineSpec().GetRuntimeConfig().GetPipelineRoot()
		return runDetail, nil

	} else {
		return nil, fmt.Errorf("failed to generate RunDetail with templateType %s.", templateType)
	}
}

// The input run might not contain workflowSpecManifest and pipelineSpecManifest, but instead a pipeline ID.
// The caller would retrieve manifest and pass in.
func (r *ResourceManager) ToModelRunDetailV2(runInterface interface{}, runId string, runAt int64, manifest string, templateType template.TemplateType) (*model.RunDetail, error) {
	modelRunDetail := &model.RunDetail{}
	run := runInterface.(*apiv2beta1.Run)

	// Retrieve Pipeline name
	var pipelineName string
	var err error
	if run.GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, run.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}

	modelRunDetail.UUID = runId
	modelRunDetail.ExperimentUUID = run.ExperimentId
	modelRunDetail.DisplayName = run.DisplayName
	modelRunDetail.Name = run.DisplayName
	namespace, err := r.GetNamespaceFromExperimentID(run.ExperimentId)
	if err != nil {
		return nil, util.Wrap(err, "Unable to retrieve namespace from experiment id.")
	}
	modelRunDetail.Namespace = namespace
	modelRunDetail.Description = run.Description
	modelRunDetail.ServiceAccount = run.ServiceAccount
	modelRunDetail.PipelineSpec = model.PipelineSpec{
		PipelineId:   run.GetPipelineId(),
		PipelineName: pipelineName,
	}

	params, err := runtimeConfigToModelParametersV2(run.GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse the V2 parameter.")
	}
	modelRunDetail.PipelineSpecManifest = manifest
	modelRunDetail.RuntimeConfig.Parameters = params
	modelRunDetail.RuntimeConfig.PipelineRoot = run.GetRuntimeConfig().GetPipelineRoot()

	// Assign the scheduled at time
	if !run.ScheduledAt.AsTime().IsZero() {
		// if there is no scheduled time, then we assume this run is scheduled at the same time it is created
		modelRunDetail.ScheduledAtInSec = runAt
	} else {
		modelRunDetail.ScheduledAtInSec = run.ScheduledAt.AsTime().Unix()
	}

	return modelRunDetail, nil
}

func (r *ResourceManager) updateModelRunWithNewScheduledWorkflow(modelRunDetail *model.RunDetail, workflow util.ExecutionSpec, templateType template.TemplateType) {
	modelRunDetail.Name = workflow.ExecutionName()
	modelRunDetail.Namespace = workflow.ExecutionNamespace()
	modelRunDetail.ServiceAccount = workflow.ServiceAccount()
	modelRunDetail.Conditions = string(workflow.ExecutionStatus().Condition())
	if templateType == template.V1 {
		modelRunDetail.WorkflowRuntimeManifest = workflow.ToStringForStore()
	}
}

func (r *ResourceManager) ToModelJob(jobInterface interface{}, manifest string, templateType template.TemplateType) (*model.Job, error) {
	switch jobInterface.(type) {
	case *apiv1beta1.Job:
		return r.toModelJobV1(jobInterface, manifest, templateType)
	case *apiv2beta1.RecurringRun:
		return r.toModelJobV2(jobInterface, manifest, templateType)
	default:
		return nil, util.NewInvalidInputError("Invalid Api Job type.")
	}
}

func (r *ResourceManager) toModelJobV1(jobInterface interface{}, manifest string, templateType template.TemplateType) (*model.Job, error) {
	modelJob := &model.Job{}
	apiJob := jobInterface.(*apiv1beta1.Job)
	// Add a reference to the default experiment if run does not already have a containing experiment
	ref, err := r.getDefaultExperimentIfNoExperiment(apiJob.GetResourceReferences())
	if err != nil {
		return nil, err
	}
	if ref != nil {
		apiJob.ResourceReferences = append(apiJob.GetResourceReferences(), ref)
	}

	namespace, err := r.getNamespaceFromExperiment(apiJob.GetResourceReferences())
	if err != nil {
		return nil, err
	}

	// Create model Resource References
	resourceReferences, err := r.toModelResourceReferences("", common.Job, apiJob.GetResourceReferences())
	if err != nil {
		return nil, util.Wrap(err, "Error converting resource references.")
	}
	// Retrieve Pipeline name
	var pipelineName string
	if apiJob.GetPipelineSpec().GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, apiJob.GetPipelineSpec().GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name.")
		}
	}
	modelJob.Name = apiJob.Name
	modelJob.DisplayName = apiJob.Name
	modelJob.Description = apiJob.Description
	modelJob.Enabled = apiJob.Enabled
	modelJob.Namespace = namespace
	modelJob.Trigger = toModelTriggerV1(apiJob.Trigger)
	modelJob.MaxConcurrency = apiJob.MaxConcurrency
	modelJob.NoCatchup = apiJob.NoCatchup
	modelJob.ResourceReferences = resourceReferences
	modelJob.ServiceAccount = apiJob.ServiceAccount
	modelJob.PipelineSpec = model.PipelineSpec{
		PipelineId:   apiJob.GetPipelineSpec().GetPipelineId(),
		PipelineName: pipelineName,
	}
	if templateType == template.V1 {
		// Input template if of V1 type (argo)
		params, err := apiParametersToModelParameters(apiJob.GetPipelineSpec().GetParameters())
		if err != nil {
			return nil, util.Wrap(err, "Unable to parse the parameters.")
		}
		modelJob.Parameters = params
		modelJob.WorkflowSpecManifest = manifest
	} else if templateType == template.V2 {
		// Input template if of V2 type (IR)
		params, err := runtimeConfigToModelParametersV1(apiJob.GetPipelineSpec().GetRuntimeConfig())
		if err != nil {
			return nil, util.Wrap(err, "Unable to parse the parameters inside runtimeConfig.")
		}
		modelJob.PipelineSpecManifest = manifest
		modelJob.PipelineSpec.RuntimeConfig.Parameters = params
		modelJob.PipelineSpec.RuntimeConfig.PipelineRoot = apiJob.GetPipelineSpec().GetRuntimeConfig().GetPipelineRoot()
	} else {
		return nil, fmt.Errorf("failed to generate ModelJob with templateType %s", templateType)
	}
	return modelJob, nil
}

func (r *ResourceManager) toModelJobV2(jobInterface interface{}, manifest string, templateType template.TemplateType) (*model.Job, error) {
	modelJob := &model.Job{}
	apiRecurringRun := jobInterface.(*apiv2beta1.RecurringRun)
	// Retrieve Pipeline name
	var pipelineName string
	var err error
	if apiRecurringRun.GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, apiRecurringRun.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name.")
		}
	}
	modelJob.DisplayName = apiRecurringRun.DisplayName
	modelJob.Name = apiRecurringRun.DisplayName
	modelJob.Description = apiRecurringRun.Description
	modelJob.Enabled = modeToModelEnabled(apiRecurringRun.Mode)
	modelJob.Trigger, err = toModelTriggerV2(apiRecurringRun.Trigger)
	if err != nil {
		return nil, util.Wrap(err, "Cannot convert RecurringRun Trigger.")
	}
	modelJob.MaxConcurrency = apiRecurringRun.MaxConcurrency
	modelJob.NoCatchup = apiRecurringRun.NoCatchup
	modelJob.ServiceAccount = apiRecurringRun.ServiceAccount
	namespace, err := r.GetNamespaceFromExperimentID(apiRecurringRun.ExperimentId)
	if err != nil {
		return nil, util.Wrap(err, "Unable to retrieve namespace from experiment id.")
	}
	modelJob.Namespace = namespace
	modelJob.PipelineSpec = model.PipelineSpec{
		PipelineId:   apiRecurringRun.GetPipelineId(),
		PipelineName: pipelineName,
	}
	params, err := runtimeConfigToModelParameters(apiRecurringRun.GetRuntimeConfig())
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse the parameters inside runtimeConfig.")
	}
	modelJob.PipelineSpecManifest = manifest
	modelJob.PipelineSpec.RuntimeConfig.Parameters = params
	modelJob.PipelineSpec.RuntimeConfig.PipelineRoot = apiRecurringRun.GetRuntimeConfig().GetPipelineRoot()

	return modelJob, nil
}

func (r *ResourceManager) updateModelJobWithNewScheduledWorkflow(modelJob *model.Job, swf *util.ScheduledWorkflow) error {
	modelJob.UUID = string(swf.UID)
	modelJob.Name = swf.Name
	modelJob.Namespace = swf.Namespace
	modelJob.Conditions = swf.ConditionSummary()
	r.updateJobResourceReferences(string(swf.UID), modelJob)

	serviceAccount := ""
	if swf.Spec.Workflow != nil {
		execSpec, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, swf.Spec.Workflow)
		if err == nil {
			serviceAccount = execSpec.ServiceAccount()
		}
	}
	modelJob.ServiceAccount = serviceAccount
	return nil
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

func (r *ResourceManager) ToModelPipelineVersion(version *apiv1beta1.PipelineVersion) (*model.PipelineVersion, error) {
	paramStr, err := apiParametersToModelParameters(version.Parameters)
	if err != nil {
		return nil, err
	}

	var pipelineId string
	for _, resourceReference := range version.ResourceReferences {
		if resourceReference.Key.Type == apiv1beta1.ResourceType_PIPELINE {
			pipelineId = resourceReference.Key.Id
		}
	}

	return &model.PipelineVersion{
		UUID:           string(version.Id),
		Name:           version.Name,
		CreatedAtInSec: version.CreatedAt.Seconds,
		Parameters:     paramStr,
		PipelineId:     pipelineId,
		CodeSourceUrl:  version.CodeSourceUrl,
	}, nil
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

func runtimeConfigToModelParametersV2(runtimeConfig *apiv2beta1.RuntimeConfig) (string, error) {
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

func (r *ResourceManager) toModelResourceReferences(
	resourceId string, resourceType model.ResourceType, apiRefs []*apiv1beta1.ResourceReference) ([]*model.ResourceReference, error) {
	var modelRefs []*model.ResourceReference
	for _, apiRef := range apiRefs {
		modelReferenceType, err := common.ToModelResourceType(apiRef.Key.Type)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert reference type")
		}
		modelRelationship, err := common.ToModelRelationship(apiRef.Relationship)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert relationship")
		}
		referenceName, err := r.getResourceName(modelReferenceType, apiRef.Key.Id)
		if err != nil {
			return nil, util.Wrap(err, "Failed to find the referred resource")
		}

		//TODO(gaoning777) further investigation: Is the plain namespace a good option?  maybe uuid for distinctness even with namespace deletion/recreation.
		modelRef := &model.ResourceReference{
			ResourceUUID:  resourceId,
			ResourceType:  resourceType,
			ReferenceUUID: apiRef.Key.Id,
			ReferenceName: referenceName,
			ReferenceType: modelReferenceType,
			Relationship:  modelRelationship,
		}
		modelRefs = append(modelRefs, modelRef)
	}
	return modelRefs, nil
}

func (r *ResourceManager) getResourceName(resourceType model.ResourceType, resourceId string) (string, error) {
	switch resourceType {
	case common.Experiment:
		experiment, err := r.GetExperiment(resourceId)
		if err != nil {
			return "", util.Wrap(err, "Referred experiment not found.")
		}
		return experiment.Name, nil
	case common.Pipeline:
		pipeline, err := r.GetPipeline(resourceId)
		if err != nil {
			return "", util.Wrap(err, "Referred pipeline not found.")
		}
		return pipeline.Name, nil
	case common.Job:
		job, err := r.GetJob(resourceId)
		if err != nil {
			return "", util.NewInvalidInputError("Referred job not found.")
		}
		return job.DisplayName, nil
	case common.Run:
		run, err := r.GetRun(resourceId)
		if err != nil {
			return "", util.Wrap(err, "Referred run not found.")
		}
		return run.DisplayName, nil
	case common.PipelineVersion:
		version, err := r.GetPipelineVersion(resourceId)
		if err != nil {
			return "", util.Wrap(err, "Referred pipeline version not found.")
		}
		return version.Name, nil
	case common.Namespace:
		return resourceId, nil
	default:
		return "", util.NewInvalidInputError("Unsupported resource type: %s", string(resourceType))
	}
}

func (r *ResourceManager) getOwningExperimentUUID(references []*apiv1beta1.ResourceReference) (string, error) {
	var experimentUUID string
	for _, ref := range references {
		if ref.Key.Type == apiv1beta1.ResourceType_EXPERIMENT && ref.Relationship == apiv1beta1.Relationship_OWNER {
			experimentUUID = ref.Key.Id
			break
		}
	}

	if experimentUUID == "" {
		return "", util.NewInternalServerError(nil, "Missing owning experiment UUID")
	}
	return experimentUUID, nil
}
