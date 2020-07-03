// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestToApiPipeline(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
		Parameters:     "[]",
		DefaultVersion: &model.PipelineVersion{
			UUID:           "pipelineversion1",
			CreatedAtInSec: 1,
			Parameters:     "[]",
			PipelineId:     "pipeline1",
			CodeSourceUrl:  "http://repo/22222",
		},
	}
	apiPipeline := ToApiPipeline(modelPipeline)
	expectedApiPipeline := &api.Pipeline{
		Id:         "pipeline1",
		CreatedAt:  &timestamp.Timestamp{Seconds: 1},
		Parameters: []*api.Parameter{},
		DefaultVersion: &api.PipelineVersion{
			Id:            "pipelineversion1",
			CreatedAt:     &timestamp.Timestamp{Seconds: 1},
			Parameters:    []*api.Parameter{},
			CodeSourceUrl: "http://repo/22222",
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   "pipeline1",
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		},
	}
	assert.Equal(t, expectedApiPipeline, apiPipeline)
}

func TestToApiPipeline_ErrorParsingField(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
		Parameters:     "[invalid parameter",
		DefaultVersion: &model.PipelineVersion{},
	}
	apiPipeline := ToApiPipeline(modelPipeline)
	expectedApiPipeline := &api.Pipeline{
		Id:    "pipeline1",
		Error: "InternalServerError: Parameter with wrong format is stored: invalid character 'i' looking for beginning of value",
	}
	assert.Equal(t, expectedApiPipeline, apiPipeline)
}

func TestToApiRunDetail(t *testing.T) {
	modelRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "run123",
			Name:             "name123",
			StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
			DisplayName:      "displayName123",
			Namespace:        "ns123",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			FinishedAtInSec:  1,
			Conditions:       "running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "manifest",
			},
			ResourceReferences: []*model.ResourceReference{
				{ResourceUUID: "run123", ResourceType: common.Run, ReferenceUUID: "job123",
					ReferenceName: "j123", ReferenceType: common.Job, Relationship: common.Creator},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow123"},
	}
	apiRun := ToApiRunDetail(modelRun)
	expectedApiRun := &api.RunDetail{
		Run: &api.Run{
			Id:           "run123",
			Name:         "displayName123",
			StorageState: api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 1},
			FinishedAt:   &timestamp.Timestamp{Seconds: 1},
			Status:       "running",
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			ResourceReferences: []*api.ResourceReference{
				{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "job123"},
					Name: "j123", Relationship: api.Relationship_CREATOR},
			},
		},
		PipelineRuntime: &api.PipelineRuntime{
			WorkflowManifest: "workflow123",
		},
	}
	assert.Equal(t, expectedApiRun, apiRun)
}

func TestToApiRuns(t *testing.T) {
	metric1 := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}
	metric2 := &model.RunMetric{
		Name:        "metric-2",
		NodeID:      "node-2",
		NumberValue: 0.99,
		Format:      "PERCENTAGE",
	}
	apiMetric1 := &api.RunMetric{
		Name:   metric1.Name,
		NodeId: metric1.NodeID,
		Value:  &api.RunMetric_NumberValue{NumberValue: metric1.NumberValue},
		Format: api.RunMetric_RAW,
	}
	apiMetric2 := &api.RunMetric{
		Name:   metric2.Name,
		NodeId: metric2.NodeID,
		Value:  &api.RunMetric_NumberValue{NumberValue: metric2.NumberValue},
		Format: api.RunMetric_PERCENTAGE,
	}
	modelRun1 := model.Run{
		UUID:             "run1",
		Name:             "name1",
		StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
		DisplayName:      "displayName1",
		Namespace:        "ns1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "run1", ResourceType: common.Run, ReferenceUUID: "job1",
				ReferenceName: "j1", ReferenceType: common.Job, Relationship: common.Creator},
		},
		Metrics: []*model.RunMetric{metric1, metric2},
	}
	modelRun2 := model.Run{
		UUID:             "run2",
		Name:             "name2",
		StorageState:     api.Run_STORAGESTATE_AVAILABLE.String(),
		DisplayName:      "displayName2",
		Namespace:        "ns2",
		CreatedAtInSec:   2,
		ScheduledAtInSec: 2,
		Conditions:       "done",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "run2", ResourceType: common.Run, ReferenceUUID: "job2",
				ReferenceName: "j2", ReferenceType: common.Job, Relationship: common.Creator},
		},
		Metrics: []*model.RunMetric{metric2},
	}
	apiRuns := ToApiRuns([]*model.Run{&modelRun1, &modelRun2})
	expectedApiRun := []*api.Run{
		{
			Id:           "run1",
			Name:         "displayName1",
			StorageState: api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 1},
			FinishedAt:   &timestamp.Timestamp{},
			Status:       "running",
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			ResourceReferences: []*api.ResourceReference{
				{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "job1"},
					Name: "j1", Relationship: api.Relationship_CREATOR},
			},
			Metrics: []*api.RunMetric{apiMetric1, apiMetric2},
		},
		{
			Id:           "run2",
			Name:         "displayName2",
			StorageState: api.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 2},
			FinishedAt:   &timestamp.Timestamp{},
			Status:       "done",
			ResourceReferences: []*api.ResourceReference{
				{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "job2"},
					Name: "j2", Relationship: api.Relationship_CREATOR},
			},
			PipelineSpec: &api.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			Metrics: []*api.RunMetric{apiMetric2},
		},
	}
	assert.Equal(t, expectedApiRun, apiRuns)
}

func TestCronScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "job1", ResourceType: common.Job, ReferenceUUID: "experiment1", ReferenceName: "e1",
				ReferenceType: common.Experiment, Relationship: common.Owner},
		},
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			Parameters:   []*api.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "experiment1"},
				Name: "e1", Relationship: api.Relationship_OWNER},
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestPeriodicScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_PeriodicSchedule{PeriodicSchedule: &api.PeriodicSchedule{
				StartTime:      &timestamp.Timestamp{Seconds: 1},
				IntervalSecond: 3,
			}}},
		PipelineSpec: &api.PipelineSpec{
			Parameters:   []*api.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestNonScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger:        &api.Trigger{},
		PipelineSpec: &api.PipelineSpec{
			Parameters:   []*api.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestToApiJob_ErrorParsingField(t *testing.T) {
	modelJob := &model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `invalid parameter format`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}

	apiJob := ToApiJob(modelJob)
	expectedApiJob := &api.Job{
		Id:    "job1",
		Error: "InternalServerError: Parameter with wrong format is stored: invalid character 'i' looking for beginning of value",
	}
	assert.Equal(t, expectedApiJob, apiJob)
}

func TestToApiJobs(t *testing.T) {
	modelJob1 := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	modeljob2 := model.Job{
		UUID:        "job2",
		DisplayName: "name 2",
		Name:        "name2",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(2),
				Cron:                       util.StringPointer("2 * *"),
			},
		},
		MaxConcurrency: 2,
		NoCatchup:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "2",
			PipelineName: "p2",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	apiJobs := ToApiJobs([]*model.Job{&modelJob1, &modeljob2})
	expectedJobs := []*api.Job{
		{
			Id:             "job1",
			Name:           "name 1",
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 1},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
			MaxConcurrency: 1,
			Trigger: &api.Trigger{
				Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 1},
					Cron:      "1 * *",
				}}},
			PipelineSpec: &api.PipelineSpec{
				Parameters:   []*api.Parameter{{Name: "param2", Value: "world"}},
				PipelineId:   "1",
				PipelineName: "p1",
			},
		},
		{
			Id:             "job2",
			Name:           "name 2",
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
			MaxConcurrency: 2,
			NoCatchup:      true,
			Trigger: &api.Trigger{
				Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 2},
					Cron:      "2 * *",
				}}},
			PipelineSpec: &api.PipelineSpec{
				Parameters:   []*api.Parameter{{Name: "param2", Value: "world"}},
				PipelineId:   "2",
				PipelineName: "p2",
			},
		},
	}
	assert.Equal(t, expectedJobs, apiJobs)
}

func TestToApiRunMetric(t *testing.T) {
	modelRunMetric := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}

	actualAPIRunMetric := ToApiRunMetric(modelRunMetric)

	expectedAPIRunMetric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
	}
	assert.Equal(t, expectedAPIRunMetric, actualAPIRunMetric)
}

func TestToApiRunMetric_UnknownFormat(t *testing.T) {
	// This can happen if we accidentally remove an existing format value from proto.
	modelRunMetric := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "NotExistValue",
	}

	actualAPIRunMetric := ToApiRunMetric(modelRunMetric)

	expectedAPIRunMetric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		// Expect return UNSPECIFIED for unknown format
		Format: api.RunMetric_UNSPECIFIED,
	}
	assert.Equal(t, expectedAPIRunMetric, actualAPIRunMetric)
}

func TestToApiResourceReferences(t *testing.T) {
	resourceReferences := []*model.ResourceReference{
		{ResourceUUID: "run1", ResourceType: common.Run, ReferenceUUID: "experiment1",
			ReferenceName: "e1", ReferenceType: common.Experiment, Relationship: common.Owner},
		{ResourceUUID: "run1", ResourceType: common.Run, ReferenceUUID: "job1",
			ReferenceName: "j1", ReferenceType: common.Job, Relationship: common.Owner},
		{ResourceUUID: "run1", ResourceType: common.Run, ReferenceUUID: "pipelineversion1",
			ReferenceName: "k1", ReferenceType: common.PipelineVersion, Relationship: common.Owner},
	}
	expectedApiResourceReferences := []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "experiment1"},
			Name: "e1", Relationship: api.Relationship_OWNER},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "job1"},
			Name: "j1", Relationship: api.Relationship_OWNER},
		{Key: &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: "pipelineversion1"},
			Name: "k1", Relationship: api.Relationship_OWNER},
	}
	assert.Equal(t, expectedApiResourceReferences, toApiResourceReferences(resourceReferences))
}

func TestToApiExperiments(t *testing.T) {
	exp1 := &model.Experiment{
		UUID:           "exp1",
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	exp2 := &model.Experiment{
		UUID:           "exp2",
		CreatedAtInSec: 2,
		Name:           "experiment2",
		Description:    "My name is experiment2",
		StorageState:   "STORAGESTATE_ARCHIVED",
	}
	apiExps := ToApiExperiments([]*model.Experiment{exp1, exp2})
	expectedApiExps := []*api.Experiment{
		{
			Id:           "exp1",
			Name:         "experiment1",
			Description:  "My name is experiment1",
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			StorageState: api.Experiment_StorageState(api.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"]),
		},
		{
			Id:           "exp2",
			Name:         "experiment2",
			Description:  "My name is experiment2",
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			StorageState: api.Experiment_StorageState(api.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"]),
		},
	}
	assert.Equal(t, expectedApiExps, apiExps)
}
