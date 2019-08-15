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
)

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
	params, err := toModelParameters(run.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse the parameter.")
	}
	resourceReferences, err := r.toModelResourceReferences(runId, common.Run, run.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Unable to convert resource references.")
	}
	var pipelineName string
	if run.PipelineSpec.GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, run.PipelineSpec.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}

	return &model.RunDetail{
		Run: model.Run{
			UUID:               runId,
			DisplayName:        run.Name,
			Name:               workflow.Name,
			Namespace:          workflow.Namespace,
			Conditions:         workflow.Condition(),
			Description:        run.Description,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           run.PipelineSpec.GetPipelineId(),
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
	params, err := toModelParameters(job.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Error parsing the input job.")
	}
	resourceReferences, err := r.toModelResourceReferences(string(swf.UID), common.Job, job.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Error to convert resource references.")
	}
	var pipelineName string
	if job.PipelineSpec.GetPipelineId() != "" {
		pipelineName, err = r.getResourceName(common.Pipeline, job.PipelineSpec.GetPipelineId())
		if err != nil {
			return nil, util.Wrap(err, "Error getting the pipeline name")
		}
	}
	return &model.Job{
		UUID:               string(swf.UID),
		DisplayName:        job.Name,
		Name:               swf.Name,
		Namespace:          swf.Namespace,
		Description:        job.Description,
		Conditions:         swf.ConditionSummary(),
		Enabled:            job.Enabled,
		Trigger:            toModelTrigger(job.Trigger),
		MaxConcurrency:     job.MaxConcurrency,
		ResourceReferences: resourceReferences,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           job.PipelineSpec.GetPipelineId(),
			PipelineName:         pipelineName,
			WorkflowSpecManifest: workflowSpecManifest,
			Parameters:           params,
		},
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
	default:
		return "", util.NewInvalidInputError("Unsupported resource type: %s", string(resourceType))
	}
}
