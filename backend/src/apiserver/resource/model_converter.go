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
func ToModelRun(run *api.Run, workflowSpecManifest string) (*model.Run, error) {
	params, err := toModelParameters(run.PipelineSpec.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Error parsing the parameter.")
	}

	// Only convert input field. Any output specific field will be ignored.
	return &model.Run{
		DisplayName: run.Name,
		Description: run.Description,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           run.PipelineSpec.GetPipelineId(),
			WorkflowSpecManifest: workflowSpecManifest,
			Parameters:           params,
		},
	}, nil
}

func ToModelJob(job *api.Job) (*model.Job, error) {
	params, err := toModelParameters(job.Parameters)
	if err != nil {
		return nil, util.Wrap(err, "Error parsing the input job.")
	}

	// Only convert input field. Any output specific field will be ignored.
	return &model.Job{
		DisplayName:    job.Name,
		Description:    job.Description,
		Enabled:        job.Enabled,
		Trigger:        toModelTrigger(job.Trigger),
		MaxConcurrency: job.MaxConcurrency,
		PipelineSpec: model.PipelineSpec{
			PipelineId: job.PipelineId,
			Parameters: params,
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
	return string(paramsBytes), nil
}
