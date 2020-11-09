package server

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var (
	commonApiJob = &api.Job{
		Name:           "job1",
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
			{
				Key:          &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Relationship: api.Relationship_OWNER,
			},
		},
	}

	commonExpectedJob = &api.Job{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "job1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt: &timestamp.Timestamp{Seconds: 2},
		UpdatedAt: &timestamp.Timestamp{Seconds: 2},
		Status:    "NO_STATUS",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*api.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{
				Key:  &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Name: "exp1", Relationship: api.Relationship_OWNER,
			},
		},
	}
)

func TestValidateApiJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	err := server.validateCreateJobRequest(&api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &api.Job{
		Name:           "job1",
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

func TestCreateJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(nil, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
	assert.Equal(t, commonExpectedJob, job)
}

func TestCreateJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_KFAM_Unauthorized(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")
}

func TestGetJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.KfamClientFake = client.NewFakeKFAMClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.GetJob(ctx, &api.GetJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")
}

func TestGetJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	createdJob, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	job, err := server.GetJob(ctx, &api.GetJobRequest{Id: createdJob.Id})
	assert.Nil(t, err)
	assert.Equal(t, commonExpectedJob, job)
}

func TestListJobs_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_KFAM_Unauthorized(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.ListJobs(ctx, &api.ListJobsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")

	_, err = server.ListJobs(ctx, &api.ListJobsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")
}

func TestListJobs_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	var expectedJobs []*api.Job
	expectedJobs = append(expectedJobs, commonExpectedJob)
	expectedJobsEmpty := []*api.Job{}

	tests := []struct {
		name         string
		request      *api.ListJobsRequest
		wantError    bool
		errorMessage string
		expectedJobs []*api.Job
	}{
		{
			"Valid - filter by experiment",
			&api.ListJobsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_EXPERIMENT,
					Id:   "123e4567-e89b-12d3-a456-426655440000",
				},
			},
			false,
			"",
			expectedJobs,
		},
		{
			"Valid - filter by namespace",
			&api.ListJobsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			expectedJobs,
		},
		{
			"Vailid - filter by namespace - no result",
			&api.ListJobsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_NAMESPACE,
					Id:   "no-such-ns",
				},
			},
			false,
			"",
			expectedJobsEmpty,
		},
		{
			"Invalid - no filter",
			&api.ListJobsRequest{},
			true,
			"ListJobs must filter by resource reference",
			nil,
		},
		{
			"Inalid - invalid filter type",
			&api.ListJobsRequest{
				ResourceReferenceKey: &api.ResourceKey{
					Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE,
					Id:   "unknown",
				},
			},
			true,
			"Unrecognized resource reference type",
			nil,
		},
	}

	for _, tc := range tests {
		response, err := server.ListJobs(ctx, tc.request)

		if tc.wantError {
			if err == nil {
				t.Errorf("TestListJobs_Multiuser(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestListJobs_Multiusert(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestListJobs_Multiuser(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedJobs, response.Jobs, cmpopts.IgnoreFields(api.Job{}, "Trigger"),
				cmpopts.IgnoreFields(api.Run{}, "CreatedAt"), cmpopts.IgnoreFields(api.Job{}, "UpdatedAt"),
				cmpopts.IgnoreFields(api.Job{}, "CreatedAt")) {
				t.Errorf("TestListJobs_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedJobs, response.Jobs)
			}
		}
	}
}

func TestEnableJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.KfamClientFake = client.NewFakeKFAMClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.EnableJob(ctx, &api.EnableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")
}

func TestEnableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	job, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	_, err = server.EnableJob(ctx, &api.EnableJobRequest{Id: job.Id})
	assert.Nil(t, err)
}

func TestDisableJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.KfamClientFake = client.NewFakeKFAMClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.DisableJob(ctx, &api.DisableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthorized access")
}

func TestDisableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	job, err := server.CreateJob(ctx, &api.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	_, err = server.DisableJob(ctx, &api.DisableJobRequest{Id: job.Id})
	assert.Nil(t, err)
}
