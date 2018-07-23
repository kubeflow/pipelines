package main

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestToApiJobDetail(t *testing.T) {
	modelJob := &model.JobDetail{
		Job: model.Job{
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
	apiJob := ToApiJobDetail(modelJob)
	expectedApiJob := &api.JobDetail{
		Job: &api.Job{
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

func TestToApiJobs(t *testing.T) {
	modelJob1 := model.Job{
		UUID:             "job1",
		Name:             "name1",
		Namespace:        "ns1",
		PipelineID:       "pipeline1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
	}
	modelJob2 := model.Job{
		UUID:             "job2",
		Name:             "name2",
		Namespace:        "ns2",
		PipelineID:       "pipeline2",
		CreatedAtInSec:   2,
		ScheduledAtInSec: 2,
		Conditions:       "done",
	}
	apiJobs := ToApiJobs([]model.Job{modelJob1, modelJob2})
	expectedApiJob := []*api.Job{
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

func TestToApiPipeline(t *testing.T) {
	modelPipeline := model.Pipeline{
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
	apiPipeline, err := ToApiPipeline(&modelPipeline)
	assert.Nil(t, err)
	expectedPipeline := &api.Pipeline{
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

func TestNonScheduledPipelineToApiPipeline(t *testing.T) {
	modelPipeline := model.Pipeline{
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
	apiPipeline, err := ToApiPipeline(&modelPipeline)
	assert.Nil(t, err)
	expectedPipeline := &api.Pipeline{
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

func TestToApiPipelines(t *testing.T) {
	modelPipeline1 := model.Pipeline{
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
	modelpipeline2 := model.Pipeline{
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
	apiPipelines, err := ToApiPipelines([]model.Pipeline{modelPipeline1, modelpipeline2})
	assert.Nil(t, err)
	expectedPipelines := []*api.Pipeline{
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

func TestToModelPipeline(t *testing.T) {
	apiPipeline := &api.Pipeline{
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
	modelPipeline, err := ToModelPipeline(apiPipeline)
	assert.Nil(t, err)

	expectedModelPipeline := &model.Pipeline{
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
