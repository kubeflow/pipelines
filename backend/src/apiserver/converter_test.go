package main

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestToApiJobDetailV2(t *testing.T) {
	modelJob := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "job123",
			Name:             "name123",
			Namespace:        "ns123",
			PipelineID:       "pipeline123",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		Workflow: "workflow123",
	}
	apiJob := ToApiJobDetailV2(modelJob)
	expectedApiJob := &api.JobDetailV2{
		Job: &api.JobV2{
			Id:          "job123",
			Name:        "name123",
			Namespace:   "ns123",
			CreatedAt:   &timestamp.Timestamp{Seconds: 1},
			ScheduledAt: &timestamp.Timestamp{Seconds: 1},
			Status:      "running",
		},
		Workflow: "workflow123",
	}
	assert.Equal(t, expectedApiJob, apiJob)
}

func TestToApiJobsV2(t *testing.T) {
	modelJob1 := model.JobV2{
		UUID:             "job1",
		Name:             "name1",
		Namespace:        "ns1",
		PipelineID:       "pipeline1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
	}
	modelJob2 := model.JobV2{
		UUID:             "job2",
		Name:             "name2",
		Namespace:        "ns2",
		PipelineID:       "pipeline2",
		CreatedAtInSec:   2,
		ScheduledAtInSec: 2,
		Conditions:       "done",
	}
	apiJobs := ToApiJobsV2([]model.JobV2{modelJob1, modelJob2})
	expectedApiJob := []*api.JobV2{
		{
			Id:          "job1",
			Name:        "name1",
			Namespace:   "ns1",
			CreatedAt:   &timestamp.Timestamp{Seconds: 1},
			ScheduledAt: &timestamp.Timestamp{Seconds: 1},
			Status:      "running",
		},
		{
			Id:          "job2",
			Name:        "name2",
			Namespace:   "ns2",
			CreatedAt:   &timestamp.Timestamp{Seconds: 2},
			ScheduledAt: &timestamp.Timestamp{Seconds: 2},
			Status:      "done",
		},
	}
	assert.Equal(t, expectedApiJob, apiJobs)
}

func TestToApiPipelineV2(t *testing.T) {
	modelPipeline := model.PipelineV2{
		UUID:      "pipeline1",
		Name:      "name1",
		PackageId: 1,
		Enabled:   true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron: util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiPipeline, err := ToApiPipelineV2(&modelPipeline)
	assert.Nil(t, err)
	expectedPipeline := &api.PipelineV2{
		Id:             "pipeline1",
		Name:           "name1",
		PackageId:      1,
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * *",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	assert.Equal(t, expectedPipeline, apiPipeline)
}

func TestNonScheduledPipelineToApiPipelineV2(t *testing.T) {
	modelPipeline := model.PipelineV2{
		UUID:           "pipeline1",
		Name:           "name1",
		PackageId:      1,
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiPipeline, err := ToApiPipelineV2(&modelPipeline)
	assert.Nil(t, err)
	expectedPipeline := &api.PipelineV2{
		Id:             "pipeline1",
		Name:           "name1",
		PackageId:      1,
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger:        &api.Trigger{},
		Parameters:     []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	assert.Equal(t, expectedPipeline, apiPipeline)
}

func TestToApiPipelinesV2(t *testing.T) {
	modelPipeline1 := model.PipelineV2{
		UUID:      "pipeline1",
		Name:      "name1",
		PackageId: 1,
		Enabled:   true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron: util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	modelpipeline2 := model.PipelineV2{
		UUID:      "pipeline2",
		Name:      "name2",
		PackageId: 2,
		Enabled:   true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(2),
				Cron: util.StringPointer("2 * *"),
			},
		},
		MaxConcurrency: 2,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	apiPipelines, err := ToApiPipelinesV2([]model.PipelineV2{modelPipeline1, modelpipeline2})
	assert.Nil(t, err)
	expectedPipelines := []*api.PipelineV2{
		{
			Id:             "pipeline1",
			Name:           "name1",
			PackageId:      1,
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 1},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
			MaxConcurrency: 1,
			Trigger: &api.Trigger{
				Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 1},
					Cron:      "1 * *",
				}}},
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
		{
			Id:             "pipeline2",
			Name:           "name2",
			PackageId:      2,
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
			MaxConcurrency: 2,
			Trigger: &api.Trigger{
				Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 2},
					Cron:      "2 * *",
				}}},
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
	}
	assert.Equal(t, expectedPipelines, apiPipelines)
}

func TestToModelPipelineV2(t *testing.T) {
	apiPipeline := &api.PipelineV2{
		Id:             "pipeline1",
		Name:           "name1",
		PackageId:      1,
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * *",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	modelPipeline, err := ToModelPipelineV2(apiPipeline)
	assert.Nil(t, err)

	expectedModelPipeline := &model.PipelineV2{
		UUID:      "pipeline1",
		Name:      "name1",
		PackageId: 1,
		Enabled:   true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron: util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
	}
	assert.Equal(t, expectedModelPipeline, modelPipeline)
}
