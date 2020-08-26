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

package server

import (
	"encoding/json"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func ToApiExperiment(experiment *model.Experiment) *api.Experiment {
	resourceReferences := []*api.ResourceReference(nil)
	if common.IsMultiUserMode() {
		resourceReferences = []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   experiment.Namespace,
				},
				Relationship: api.Relationship_OWNER,
			},
		}
	}
	return &api.Experiment{
		Id:                 experiment.UUID,
		Name:               experiment.Name,
		Description:        experiment.Description,
		CreatedAt:          &timestamp.Timestamp{Seconds: experiment.CreatedAtInSec},
		ResourceReferences: resourceReferences,
		StorageState:       api.Experiment_StorageState(api.Experiment_StorageState_value[experiment.StorageState]),
	}
}

func ToApiExperiments(experiments []*model.Experiment) []*api.Experiment {
	apiExperiments := make([]*api.Experiment, 0)
	for _, experiment := range experiments {
		apiExperiments = append(apiExperiments, ToApiExperiment(experiment))
	}
	return apiExperiments
}

func ToApiPipeline(pipeline *model.Pipeline) *api.Pipeline {
	params, err := toApiParameters(pipeline.Parameters)
	if err != nil {
		return &api.Pipeline{
			Id:    pipeline.UUID,
			Error: err.Error(),
		}
	}

	defaultVersion, err := ToApiPipelineVersion(pipeline.DefaultVersion)
	if err != nil {
		return &api.Pipeline{
			Id:    pipeline.UUID,
			Error: err.Error(),
		}
	}

	return &api.Pipeline{
		Id:             pipeline.UUID,
		CreatedAt:      &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Name:           pipeline.Name,
		Description:    pipeline.Description,
		Parameters:     params,
		DefaultVersion: defaultVersion,
	}
}

func ToApiPipelineVersion(version *model.PipelineVersion) (*api.PipelineVersion, error) {
	if version == nil {
		return nil, nil
	}
	params, err := toApiParameters(version.Parameters)
	if err != nil {
		return nil, err
	}

	return &api.PipelineVersion{
		Id:            version.UUID,
		Name:          version.Name,
		CreatedAt:     &timestamp.Timestamp{Seconds: version.CreatedAtInSec},
		Parameters:    params,
		CodeSourceUrl: version.CodeSourceUrl,
		ResourceReferences: []*api.ResourceReference{
			&api.ResourceReference{
				Key: &api.ResourceKey{
					Id:   version.PipelineId,
					Type: api.ResourceType_PIPELINE,
				},
				Relationship: api.Relationship_OWNER,
			},
		},
	}, nil
}

func ToApiPipelineVersions(versions []*model.PipelineVersion) ([]*api.PipelineVersion, error) {
	apiVersions := make([]*api.PipelineVersion, 0)
	for _, version := range versions {
		v, _ := ToApiPipelineVersion(version)
		apiVersions = append(apiVersions, v)
	}
	return apiVersions, nil
}

func ToApiPipelines(pipelines []*model.Pipeline) []*api.Pipeline {
	apiPipelines := make([]*api.Pipeline, 0)
	for _, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, ToApiPipeline(pipeline))
	}
	return apiPipelines
}

func toApiParameters(paramsString string) ([]*api.Parameter, error) {
	if paramsString == "" {
		return nil, nil
	}
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

func toApiRun(run *model.Run) *api.Run {
	params, err := toApiParameters(run.Parameters)
	if err != nil {
		return &api.Run{
			Id:    run.UUID,
			Error: err.Error(),
		}
	}
	var metrics []*api.RunMetric
	if run.Metrics != nil {
		for _, metric := range run.Metrics {
			metrics = append(metrics, ToApiRunMetric(metric))
		}
	}
	return &api.Run{
		CreatedAt:      &timestamp.Timestamp{Seconds: run.CreatedAtInSec},
		Id:             run.UUID,
		Metrics:        metrics,
		Name:           run.DisplayName,
		ServiceAccount: run.ServiceAccount,
		StorageState:   api.Run_StorageState(api.Run_StorageState_value[run.StorageState]),
		Description:    run.Description,
		ScheduledAt:    &timestamp.Timestamp{Seconds: run.ScheduledAtInSec},
		FinishedAt:     &timestamp.Timestamp{Seconds: run.FinishedAtInSec},
		Status:         run.Conditions,
		PipelineSpec: &api.PipelineSpec{
			PipelineId:       run.PipelineId,
			PipelineName:     run.PipelineName,
			WorkflowManifest: run.WorkflowSpecManifest,
			PipelineManifest: run.PipelineSpecManifest,
			Parameters:       params,
		},
		ResourceReferences: toApiResourceReferences(run.ResourceReferences),
	}
}

func ToApiRuns(runs []*model.Run) []*api.Run {
	apiRuns := make([]*api.Run, 0)
	for _, run := range runs {
		apiRuns = append(apiRuns, toApiRun(run))
	}
	return apiRuns
}

func ToApiRunDetail(run *model.RunDetail) *api.RunDetail {
	return &api.RunDetail{
		Run: toApiRun(&run.Run),
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: run.WorkflowRuntimeManifest,
			PipelineManifest: run.PipelineRuntimeManifest,
		},
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
		ServiceAccount: job.ServiceAccount,
		Description:    job.Description,
		Enabled:        job.Enabled,
		CreatedAt:      &timestamp.Timestamp{Seconds: job.CreatedAtInSec},
		UpdatedAt:      &timestamp.Timestamp{Seconds: job.UpdatedAtInSec},
		Status:         job.Conditions,
		MaxConcurrency: job.MaxConcurrency,
		NoCatchup:      job.NoCatchup,
		Trigger:        toApiTrigger(job.Trigger),
		PipelineSpec: &api.PipelineSpec{
			PipelineId:       job.PipelineId,
			PipelineName:     job.PipelineName,
			WorkflowManifest: job.WorkflowSpecManifest,
			PipelineManifest: job.PipelineSpecManifest,
			Parameters:       params,
		},
		ResourceReferences: toApiResourceReferences(job.ResourceReferences),
	}
}

func ToApiJobs(jobs []*model.Job) []*api.Job {
	apiJobs := make([]*api.Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, ToApiJob(job))
	}
	return apiJobs
}

func ToApiRunMetric(metric *model.RunMetric) *api.RunMetric {
	return &api.RunMetric{
		Name:   metric.Name,
		NodeId: metric.NodeID,
		Value: &api.RunMetric_NumberValue{
			NumberValue: metric.NumberValue,
		},
		Format: api.RunMetric_Format(api.RunMetric_Format_value[metric.Format]),
	}
}

func toApiResourceReferences(references []*model.ResourceReference) []*api.ResourceReference {
	var apiReferences []*api.ResourceReference
	for _, ref := range references {
		apiReferences = append(apiReferences, &api.ResourceReference{
			Key: &api.ResourceKey{
				Type: toApiResourceType(ref.ReferenceType),
				Id:   ref.ReferenceUUID,
			},
			Name:         ref.ReferenceName,
			Relationship: toApiRelationship(ref.Relationship),
		})
	}
	return apiReferences
}

func toApiResourceType(modelType common.ResourceType) api.ResourceType {
	switch modelType {
	case common.Experiment:
		return api.ResourceType_EXPERIMENT
	case common.Job:
		return api.ResourceType_JOB
	case common.PipelineVersion:
		return api.ResourceType_PIPELINE_VERSION
	case common.Namespace:
		return api.ResourceType_NAMESPACE
	default:
		return api.ResourceType_UNKNOWN_RESOURCE_TYPE
	}
}

func toApiRelationship(r common.Relationship) api.Relationship {
	switch r {
	case common.Creator:
		return api.Relationship_CREATOR
	case common.Owner:
		return api.Relationship_OWNER
	default:
		return api.Relationship_UNKNOWN_RELATIONSHIP
	}
}

func toApiTrigger(trigger model.Trigger) *api.Trigger {
	if trigger.Cron != nil && *trigger.Cron != "" {
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

	if trigger.IntervalSecond != nil && *trigger.IntervalSecond != 0 {
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
