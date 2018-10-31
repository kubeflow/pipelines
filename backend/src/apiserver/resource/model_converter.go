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
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

func ToModelRunMetric(metric *api.RunMetric, runUUID string) *model.RunMetric {
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
func ToModelRunDetail(run *api.Run, workflow *util.Workflow, workflowSpecManifest string) (*model.RunDetail, error) {
	params, err := toModelParameters(run.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse the parameter.")
	}
	resourceReferences, err := toModelResourceReferences(string(workflow.UID), common.Run, run.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Unable to convert resource references.")
	}

	return &model.RunDetail{
		Run: model.Run{
			UUID:               string(workflow.UID),
			DisplayName:        run.Name,
			Name:               workflow.Name,
			Namespace:          workflow.Namespace,
			Conditions:         workflow.Condition(),
			Description:        run.Description,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           run.PipelineSpec.GetPipelineId(),
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           params,
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
		},
	}, nil
}

func ToModelJob(job *api.Job, swf *util.ScheduledWorkflow, workflowSpecManifest string) (*model.Job, error) {
	params, err := toModelParameters(job.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Error parsing the input job.")
	}
	resourceReferences, err := toModelResourceReferences(string(swf.UID), common.Job, job.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Error to convert resource references.")
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

func toModelResourceReferences(
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
		modelRef := &model.ResourceReference{
			ResourceUUID:  resourceId,
			ResourceType:  resourceType,
			ReferenceUUID: apiRef.Key.Id,
			ReferenceType: modelReferenceType,
			Relationship:  modelRelationship,
		}
		modelRefs = append(modelRefs, modelRef)
	}
	return modelRefs, nil
}
