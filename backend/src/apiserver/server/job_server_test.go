package server

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestValidateApiJob(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_ValidateNoExperimentResourceReferenceSucceeds(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		// This job has no ResourceReferences, no experiment
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_ValidatePipelineSpecFailed(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			PipelineId: "not_exist_pipeline",
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline not_exist_pipeline not found")
}

func TestValidateApiJob_NoValidPipelineSpecOrPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReference,
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Neither pipeline spec nor pipeline version is valid")
}

func TestValidateApiJob_InvalidCron(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * ",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Schedule cron is not a supported format")
}

func TestValidateApiJob_MaxConcurrencyOutOfRange(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 0,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "max concurrency of the job is out of range")
}

func TestValidateApiJob_NegativeIntervalSecond(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager)
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 0,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_PeriodicSchedule{PeriodicSchedule: &api.PeriodicSchedule{
				IntervalSecond: -1,
			}}},
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: api.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "The max concurrency of the job is out of range")
}
