// Copyright 2018-2022 The Kubeflow Authors
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
	"context"
	"google.golang.org/protobuf/testing/protocmp"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	authorizationv1 "k8s.io/api/authorization/v1"
)

var (
	commonApiJob = &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}

	commonExpectedJob = &apiv1beta1.Job{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "job1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt: &timestamp.Timestamp{Seconds: 2},
		UpdatedAt: &timestamp.Timestamp{Seconds: 2},
		Status:    "NO_STATUS",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}

	commonApiRecurringRun = &apiv2beta1.RecurringRun{
		DisplayName:    "job1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: &structpb.Struct{}},
		ExperimentId:   "123e4567-e89b-12d3-a456-426655440000",
	}
)

func TestValidateApiJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_ValidateNoExperimentResourceReferenceSucceeds(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		// This job has no ResourceReferences, no experiment
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
}

func TestValidateApiJob_WithInvalidPipelineVersionReference(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: referencesOfExperimentAndInvalidPipelineVersion,
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed.")
}

func TestValidateApiJob_NoValidPipelineSpecOrPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: validReference,
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version).")
}

func TestValidateApiJob_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: validReferencesOfExperimentAndPipelineVersion,
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest.")
}

func TestValidateApiJob_ValidatePipelineSpecFailed(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId: "not_exist_pipeline",
			Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Pipeline not_exist_pipeline not found")
}

func TestValidateApiJob_InvalidCron(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * ",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Schedule cron is not a supported format")
}

func TestValidateApiJob_MaxConcurrencyOutOfRange(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 0,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "max concurrency of the job is out of range")
}

func TestValidateApiJob_NegativeIntervalSecond(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 0,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &apiv1beta1.PeriodicSchedule{
				IntervalSecond: -1,
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
		},
	}
	err := server.validateCreateJobRequest(&apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "The max concurrency of the job is out of range")
}

func TestCreateJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
	assert.Equal(t, commonExpectedJob, job)
}

func TestCreateJob_V2(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	apiJob_V2 := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters:   v2RuntimeParams,
				PipelineRoot: "model-pipeline-root",
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}

	expectedJob_V2 := &apiv1beta1.Job{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "job1",
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt: &timestamp.Timestamp{Seconds: 2},
		UpdatedAt: &timestamp.Timestamp{Seconds: 2},
		Status:    "NO_STATUS",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineManifest: v2SpecHelloWorld,
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters:   v2RuntimeParams,
				PipelineRoot: "model-pipeline-root",
			},
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123e4567-e89b-12d3-a456-426655440000"},
				Name: "exp1", Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}
	job, err := server.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: apiJob_V2})
	assert.Nil(t, err)
	assert.Equal(t, expectedJob_V2, job)
}

func TestCreateJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbCreate,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeJobs,
		Name:      commonApiJob.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestGetJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()

	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.GetJob(ctx, &apiv1beta1.GetJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeJobs,
		Name:      job.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestGetJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	createdJob, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	job, err := server.GetJob(ctx, &apiv1beta1.GetJobRequest{Id: createdJob.Id})
	assert.Nil(t, err)
	assert.Equal(t, commonExpectedJob, job)
}

func TestListJobs_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbList,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeJobs,
	}
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes)),
			"Failed to authorize with namespace in experiment resource reference.",
		).Error(),
	)

	_, err = server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes)),
			"Failed to authorize with namespace resource reference.",
		).Error(),
	)
}

func TestListJobs_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	var expectedJobs []*apiv1beta1.Job
	expectedJobs = append(expectedJobs, commonExpectedJob)
	expectedJobsEmpty := []*apiv1beta1.Job{}

	tests := []struct {
		name         string
		request      *apiv1beta1.ListJobsRequest
		wantError    bool
		errorMessage string
		expectedJobs []*apiv1beta1.Job
	}{
		{
			"Valid - filter by experiment",
			&apiv1beta1.ListJobsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_EXPERIMENT,
					Id:   "123e4567-e89b-12d3-a456-426655440000",
				},
			},
			false,
			"",
			expectedJobs,
		},
		{
			"Valid - filter by namespace",
			&apiv1beta1.ListJobsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "ns1",
				},
			},
			false,
			"",
			expectedJobs,
		},
		{
			"Vailid - filter by namespace - no result",
			&apiv1beta1.ListJobsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_NAMESPACE,
					Id:   "no-such-ns",
				},
			},
			false,
			"",
			expectedJobsEmpty,
		},
		{
			"Invalid - no filter",
			&apiv1beta1.ListJobsRequest{},
			true,
			"ListJobs must filter by resource reference",
			nil,
		},
		{
			"Inalid - invalid filter type",
			&apiv1beta1.ListJobsRequest{
				ResourceReferenceKey: &apiv1beta1.ResourceKey{
					Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE,
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
			} else if !cmp.Equal(tc.expectedJobs, response.Jobs, cmpopts.EquateEmpty(), protocmp.Transform(), cmpopts.IgnoreFields(apiv1beta1.Job{}, "Trigger", "UpdatedAt", "CreatedAt"),
				cmpopts.IgnoreFields(apiv1beta1.Run{}, "CreatedAt")) {
				t.Errorf("TestListJobs_Multiuser(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedJobs, response.Jobs)
			}
		}
	}
}

func TestEnableJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.EnableJob(ctx, &apiv1beta1.EnableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbEnable,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeJobs,
		Name:      commonApiJob.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestEnableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	_, err = server.EnableJob(ctx, &apiv1beta1.EnableJobRequest{Id: job.Id})
	assert.Nil(t, err)
}

func TestDisableJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients)
	server = NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	_, err = server.DisableJob(ctx, &apiv1beta1.DisableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbDisable,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeJobs,
		Name:      job.Name,
	}
	assert.EqualError(
		t,
		err,
		wrapFailedAuthzRequestError(wrapFailedAuthzApiResourcesError(getPermissionDeniedError(userIdentity, resourceAttributes))).Error(),
	)
}

func TestDisableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	_, err = server.DisableJob(ctx, &apiv1beta1.DisableJobRequest{Id: job.Id})
	assert.Nil(t, err)
}

func TestListJobs_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()

	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	_, err := server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError),
			"Failed to authorize with namespace in experiment resource reference.",
		).Error(),
	)

	_, err = server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.EqualError(
		t,
		err,
		util.Wrap(
			wrapFailedAuthzApiResourcesError(kfpauth.IdentityHeaderMissingError),
			"Failed to authorize with namespace resource reference.",
		).Error(),
	)
}

func TestValidateApiRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})
	err := server.validateCreateRecurringRunRequest(&apiv2beta1.CreateRecurringRunRequest{RecurringRun: commonApiRecurringRun})
	assert.Nil(t, err)
}

func TestCreateRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		Status:         apiv2beta1.RecurringRun_STATUS_UNSPECIFIED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	recurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRun, recurringRun)

}

func TestGetRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		Status:         apiv2beta1.RecurringRun_STATUS_UNSPECIFIED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	recurringRun, err := server.GetRecurringRun(nil, &apiv2beta1.GetRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRun, recurringRun)

}

func TestListRecurringRuns(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		Status:         apiv2beta1.RecurringRun_STATUS_UNSPECIFIED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRun, createdRecurringRun)

	expectedRecurringRunsList := []*apiv2beta1.RecurringRun{expectedRecurringRun}

	actualRecurringRunsList, err := server.ListRecurringRuns(nil, &apiv2beta1.ListRecurringRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRunsList, actualRecurringRunsList.RecurringRuns)
}

func TestEnableJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	_, err = server.EnableRecurringRun(nil, &apiv2beta1.EnableRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
}

func TestDisableJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	_, err = server.DisableRecurringRun(nil, &apiv2beta1.DisableRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
}
