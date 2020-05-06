// Copyright 2018 Google LLC
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

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

func (r *ResourceManager) ToModelExperiment(apiExperiment *api.Experiment) (*model.Experiment, error) {
	namespace := ""
	resourceReferences := apiExperiment.GetResourceReferences()
	if resourceReferences != nil {
		if len(resourceReferences) != 1 ||
			resourceReferences[0].Key.Type != api.ResourceType_NAMESPACE ||
			resourceReferences[0].Relationship != api.Relationship_OWNER {
			return nil, util.NewInternalServerError(errors.New("Invalid resource references for experiment"), "Unable to convert to model experiment.")
		}
		namespace = resourceReferences[0].Key.Id
	}
	return &model.Experiment{
		Name:        apiExperiment.Name,
		Description: apiExperiment.Description,
		Namespace:   namespace,
	}, nil
}

func (r *ResourceManager) ToModelRunMetric(metric *api.RunMetric, runUUID string) *model.RunMetric {
	return &model.RunMetric{
		RunUUID:     runUUID,
		Name:        metric.GetName(),
		NodeID:      metric.GetNodeId(),
		NumberValue: metric.GetNumberValue(),
		Format:      metric.GetFormat().String(),
	}
}

// The input run might not contain workflowSpecManifest, but instead a pipeline ID.
// The caller would retrieve workflowSpecManifest and pass in.
func (r *ResourceManager) ToModelRunDetail(run *api.Run, runId string, workflow *util.Workflow, workflowSpecManifest string) (*model.RunDetail, error) {
	params, err := toModelParameters(run.GetPipelineSpec().GetParameters())
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse the parameter.")
	}
	resourceReferences, err := r.toModelResourceReferences(runId, common.Run, run.GetResourceReferences())
	if err != nil {
		return nil, util.Wrap(err, "Unable to convert resource references.")
	}
	var pipelineName string
	if run.GetPipelineSpec().GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, run.GetPipelineSpec().GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}

	experimentUUID, err := r.getOwningExperimentUUID(run.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Error getting the experiment UUID")
	}

	return &model.RunDetail{
		Run: model.Run{
			UUID:               runId,
			ExperimentUUID:     experimentUUID,
			DisplayName:        run.Name,
			Name:               workflow.Name,
			Namespace:          workflow.Namespace,
			ServiceAccount:     workflow.Spec.ServiceAccountName,
			Conditions:         workflow.Condition(),
			Description:        run.Description,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           run.GetPipelineSpec().GetPipelineId(),
				PipelineName:         pipelineName,
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           params,
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
		},
	}, nil
}

func (r *ResourceManager) ToModelJob(job *api.Job, swf *util.ScheduledWorkflow, workflowSpecManifest string) (*model.Job, error) {
	params, err := toModelParameters(job.GetPipelineSpec().GetParameters())
	if err != nil {
		return nil, util.Wrap(err, "Error parsing the input job.")
	}
	resourceReferences, err := r.toModelResourceReferences(string(swf.UID), common.Job, job.GetResourceReferences())
	if err != nil {
		return nil, util.Wrap(err, "Error to convert resource references.")
	}
	var pipelineName string
	if job.GetPipelineSpec().GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, job.GetPipelineSpec().GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}
	serviceAccount := ""
	if swf.Spec.Workflow != nil {
		serviceAccount = swf.Spec.Workflow.Spec.ServiceAccountName
	}
	return &model.Job{
		UUID:               string(swf.UID),
		DisplayName:        job.Name,
		Name:               swf.Name,
		Namespace:          swf.Namespace,
		ServiceAccount:     serviceAccount,
		Description:        job.Description,
		Conditions:         swf.ConditionSummary(),
		Enabled:            job.Enabled,
		Trigger:            toModelTrigger(job.Trigger),
		MaxConcurrency:     job.MaxConcurrency,
		NoCatchup:          job.NoCatchup,
		ResourceReferences: resourceReferences,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           job.GetPipelineSpec().GetPipelineId(),
			PipelineName:         pipelineName,
			WorkflowSpecManifest: workflowSpecManifest,
			Parameters:           params,
		},
	}, nil
}

func (r *ResourceManager) ToModelPipelineVersion(version *api.PipelineVersion) (*model.PipelineVersion, error) {
	paramStr, err := toModelParameters(version.Parameters)
	if err != nil {
		return nil, err
	}

	var pipelineId string
	for _, resourceReference := range version.ResourceReferences {
		if resourceReference.Key.Type == api.ResourceType_PIPELINE {
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

func toModelTrigger(trigger *api.Trigger) model.Trigger {
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

func toModelParameters(apiParams []*api.Parameter) (string, error) {
	if apiParams == nil || len(apiParams) == 0 {
		return "", nil
	}
	var params []v1alpha1.Parameter
	for _, apiParam := range apiParams {
		param := v1alpha1.Parameter{
			Name:  apiParam.Name,
			Value: &apiParam.Value,
		}
		params = append(params, param)
	}
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to stream API parameter as string.")
	}
	return string(paramsBytes), nil
}

func (r *ResourceManager) toModelResourceReferences(
	resourceId string, resourceType common.ResourceType, apiRefs []*api.ResourceReference) ([]*model.ResourceReference, error) {
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

func (r *ResourceManager) getResourceName(resourceType common.ResourceType, resourceId string) (string, error) {
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
			return "", util.Wrap(err, "Referred job not found.")
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

func (r *ResourceManager) getOwningExperimentUUID(references []*api.ResourceReference) (string, error) {
	var experimentUUID string
	for _, ref := range references {
		if ref.Key.Type == api.ResourceType_EXPERIMENT && ref.Relationship == api.Relationship_OWNER {
			experimentUUID = ref.Key.Id
			break
		}
	}

	if experimentUUID == "" {
		return "", util.NewInternalServerError(nil, "Missing owning experiment UUID")
	}
	return experimentUUID, nil
}
