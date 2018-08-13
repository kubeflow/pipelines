package main

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestToApiPipeline(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
		Parameters:     "[]",
	}
	apiPipeline := ToApiPipeline(modelPipeline)
	expectedApiPipeline := &api.Pipeline{
		Id:         "pipeline1",
		CreatedAt:  &timestamp.Timestamp{Seconds: 1},
		Parameters: []*api.Parameter{},
	}
	assert.Equal(t, expectedApiPipeline, apiPipeline)
}

func TestToApiPipeline_ErrorParsingField(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
		Parameters:     "[invalid parameter",
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
			Namespace:        "ns123",
			JobID:            "job123",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		Workflow: "workflow123",
	}
	apiRun := ToApiRunDetail(modelRun)
	expectedApiRun := &api.RunDetail{
		Run: &api.Run{
			Id:          "run123",
			Name:        "name123",
			Namespace:   "ns123",
			CreatedAt:   &timestamp.Timestamp{Seconds: 1},
			ScheduledAt: &timestamp.Timestamp{Seconds: 1},
			Status:      "running",
		},
		Workflow: "workflow123",
	}
	assert.Equal(t, expectedApiRun, apiRun)
}

func TestToApiRuns(t *testing.T) {
	modelRun1 := model.Run{
		UUID:             "run1",
		Name:             "name1",
		Namespace:        "ns1",
		JobID:            "job1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
	}
	modelRun2 := model.Run{
		UUID:             "run2",
		Name:             "name2",
		Namespace:        "ns2",
		JobID:            "job2",
		CreatedAtInSec:   2,
		ScheduledAtInSec: 2,
		Conditions:       "done",
	}
	apiRuns := ToApiRuns([]model.Run{modelRun1, modelRun2})
	expectedApiRun := []*api.Run{
		{
			Id:          "run1",
			Name:        "name1",
			Namespace:   "ns1",
			CreatedAt:   &timestamp.Timestamp{Seconds: 1},
			ScheduledAt: &timestamp.Timestamp{Seconds: 1},
			Status:      "running",
		},
		{
			Id:          "run2",
			Name:        "name2",
			Namespace:   "ns2",
			CreatedAt:   &timestamp.Timestamp{Seconds: 2},
			ScheduledAt: &timestamp.Timestamp{Seconds: 2},
			Status:      "done",
		},
	}
	assert.Equal(t, expectedApiRun, apiRuns)
}

func TestCronScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		PipelineId:  "1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name 1",
		PipelineId:     "1",
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
	assert.Equal(t, expectedJob, apiJob)
}

func TestPeriodicScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		PipelineId:  "1",
		Enabled:     true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name 1",
		PipelineId:     "1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_PeriodicSchedule{PeriodicSchedule: &api.PeriodicSchedule{
				StartTime:      &timestamp.Timestamp{Seconds: 1},
				IntervalSecond: 3,
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestNonScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		PipelineId:     "1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger:        &api.Trigger{},
		Parameters:     []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestToApiJob_ErrorParsingField(t *testing.T) {
	modelJob := &model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		PipelineId:     "1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		Parameters:     `invalid parameter format`,
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
		PipelineId:  "1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	modeljob2 := model.Job{
		UUID:        "job2",
		DisplayName: "name 2",
		Name:        "name2",
		PipelineId:  "2",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(2),
				Cron:                       util.StringPointer("2 * *"),
			},
		},
		MaxConcurrency: 2,
		Parameters:     `[{"name":"param2","value":"world"}]`,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	apiJobs, err := ToApiJobs([]model.Job{modelJob1, modeljob2})
	assert.Nil(t, err)
	expectedJobs := []*api.Job{
		{
			Id:             "job1",
			Name:           "name 1",
			PipelineId:     "1",
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
			Id:             "job2",
			Name:           "name 2",
			PipelineId:     "2",
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
	assert.Equal(t, expectedJobs, apiJobs)
}

func TestToModelJob(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * *",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	modelJob, err := ToModelJob(apiJob)
	assert.Nil(t, err)

	expectedModelJob := &model.Job{
		UUID:        "job1",
		DisplayName: "name1",
		PipelineId:  "1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		Parameters:     `[{"name":"param2","value":"world"}]`,
	}
	assert.Equal(t, expectedModelJob, modelJob)
}

func TestToModelJob_ParameterTooLong(t *testing.T) {
	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Parameters:     params,
	}
	_, err := ToModelJob(apiJob)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}
