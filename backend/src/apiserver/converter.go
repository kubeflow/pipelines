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

package main

import (
	"encoding/json"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

func ToApiPipeline(pipeline *model.Pipeline) *api.Pipeline {
	params, err := toApiParameters(pipeline.Parameters)
	if err != nil {
		return &api.Pipeline{
			Id:    pipeline.UUID,
			Error: err.Error(),
		}
	}
	return &api.Pipeline{
		Id:          pipeline.UUID,
		CreatedAt:   &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Name:        pipeline.Name,
		Description: pipeline.Description,
		Parameters:  params,
	}
}

func ToApiPipelines(pipelines []model.Pipeline) []*api.Pipeline {
	apiPipelines := make([]*api.Pipeline, 0)
	for _, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, ToApiPipeline(&pipeline))
	}
	return apiPipelines
}

func toApiParameters(paramsString string) ([]*api.Parameter, error) {
	apiParams := make([]*api.Parameter, 0)
	var params []v1alpha1.Parameter
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Parameter with wrong format is stored")
	}
	for _, param := range params {
		var value string
		if param.Value != nil {
			value = *param.Value
		}
		apiParam := api.Parameter{
			Name:  param.Name,
			Value: value,
		}
		apiParams = append(apiParams, &apiParam)
	}
	return apiParams, nil
}

func toModelParameters(apiParams []*api.Parameter) (string, error) {
	params := make([]v1alpha1.Parameter, 0)
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
	if len(paramsBytes) > util.MaxParameterBytes {
		return "", util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
	}
	return string(paramsBytes), nil
}

func toApiRun(run *model.Run) *api.Run {
	return &api.Run{
		Id:          run.UUID,
		Name:        run.Name,
		Namespace:   run.Namespace,
		CreatedAt:   &timestamp.Timestamp{Seconds: run.CreatedAtInSec},
		ScheduledAt: &timestamp.Timestamp{Seconds: run.ScheduledAtInSec},
		Status:      run.Conditions,
	}
}

func ToApiRuns(runs []model.Run) []*api.Run {
	apiRuns := make([]*api.Run, 0)
	for _, run := range runs {
		apiRuns = append(apiRuns, toApiRun(&run))
	}
	return apiRuns
}

func ToApiRunDetail(run *model.RunDetail) *api.RunDetail {
	return &api.RunDetail{
		Run:      toApiRun(&run.Run),
		Workflow: run.Workflow,
	}
}

func ToApiJob(job *model.Job) *api.Job {
	params, err := toApiParameters(job.Parameters)
	if err != nil {
		return &api.Job{
			Id:    job.UUID,
			Error: err.Error(),
		}
	}
	return &api.Job{
		Id:             job.UUID,
		Name:           job.DisplayName,
		Description:    job.Description,
		PipelineId:     job.PipelineId,
		Enabled:        job.Enabled,
		CreatedAt:      &timestamp.Timestamp{Seconds: job.CreatedAtInSec},
		UpdatedAt:      &timestamp.Timestamp{Seconds: job.UpdatedAtInSec},
		Status:         job.Conditions,
		MaxConcurrency: job.MaxConcurrency,
		Trigger:        toApiTrigger(job.Trigger),
		Parameters:     params,
	}
}

func ToApiJobs(jobs []model.Job) ([]*api.Job, error) {
	apiJobs := make([]*api.Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, ToApiJob(&job))
	}
	return apiJobs, nil
}

func ToModelJob(job *api.Job) (*model.Job, error) {
	params, err := toModelParameters(job.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Error convert job API model to DB model.")
	}
	return &model.Job{
		UUID:           job.Id,
		DisplayName:    job.Name,
		Description:    job.Description,
		PipelineId:     job.PipelineId,
		Enabled:        job.Enabled,
		Trigger:        toModelTrigger(job.Trigger),
		MaxConcurrency: job.MaxConcurrency,
		Parameters:     params,
	}, nil
}

func toModelTrigger(trigger *api.Trigger) model.Trigger {
	modelTrigger := model.Trigger{}
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

func toApiTrigger(trigger model.Trigger) *api.Trigger {
	if trigger.Cron != nil {
		var cronSchedule api.CronSchedule
		cronSchedule.Cron = *trigger.Cron
		if trigger.CronScheduleStartTimeInSec != nil {
			cronSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleStartTimeInSec}
		}
		if trigger.CronScheduleEndTimeInSec != nil {
			cronSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.CronScheduleEndTimeInSec}
		}
		return &api.Trigger{Trigger: &api.Trigger_CronSchedule{CronSchedule: &cronSchedule}}
	}

	if trigger.IntervalSecond != nil {
		var periodicSchedule api.PeriodicSchedule
		periodicSchedule.IntervalSecond = *trigger.IntervalSecond
		if trigger.PeriodicScheduleStartTimeInSec != nil {
			periodicSchedule.StartTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleStartTimeInSec}
		}
		if trigger.PeriodicScheduleEndTimeInSec != nil {
			periodicSchedule.EndTime = &timestamp.Timestamp{
				Seconds: *trigger.PeriodicScheduleEndTimeInSec}
		}
		return &api.Trigger{Trigger: &api.Trigger_PeriodicSchedule{PeriodicSchedule: &periodicSchedule}}
	}
	return &api.Trigger{}
}
