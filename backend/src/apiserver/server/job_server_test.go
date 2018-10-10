package server

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestValidateApiJob(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	err := ValidateCreateJobRequest(apiJob)
	assert.Nil(t, err)
}

func TestValidateApiJob_InvalidCron(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				Cron: "1 * * ",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	err := ValidateCreateJobRequest(apiJob)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Schedule cron is not a supported format")
}

func TestValidateApiJob_ParameterTooLong(t *testing.T) {
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
	err := ValidateCreateJobRequest(apiJob)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "parameter length exceed maximum size")
}

func TestValidateApiJob_MaxConcurrencyOutOfRange(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: -1,
		Parameters:     []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	err := ValidateCreateJobRequest(apiJob)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "max concurrency of the job is out of range")
}

func TestValidateApiJob_NegativeIntervalSecond(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_PeriodicSchedule{PeriodicSchedule: &api.PeriodicSchedule{
				IntervalSecond: -1,
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	err := ValidateCreateJobRequest(apiJob)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "invalid period schedule interval")
}
