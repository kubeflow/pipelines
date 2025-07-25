// Copyright 2018 The Kubeflow Authors
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
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

var (
	commonApiJob = &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
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
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt: timestamppb.New(time.Unix(2, 0)),
		UpdatedAt: timestamppb.New(time.Unix(2, 0)),
		Status:    "STATUS_UNSPECIFIED",
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

	commonApiRecurringRun = &apiv2beta1.RecurringRun{
		DisplayName:    "job1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: &structpb.Struct{}},
		ExperimentId:   "123e4567-e89b-12d3-a456-426655440000",
	}
)

func createJobServerV1(resourceManager *resource.ResourceManager) *JobServerV1 {
	return &JobServerV1{
		BaseJobServer: &BaseJobServer{
			resourceManager: resourceManager,
			options: &JobServerOptions{
				CollectMetrics: false,
			},
		},
	}
}

func createJobServer(resourceManager *resource.ResourceManager) *JobServer {
	return &JobServer{
		BaseJobServer: &BaseJobServer{
			resourceManager: resourceManager,
			options: &JobServerOptions{
				CollectMetrics: false,
			},
		},
	}
}

func TestCreateJob_WrongInput(t *testing.T) {
	clients, manager, experiment, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createJobServerV1(manager)
	tests := []struct {
		name   string
		arg    *apiv1beta1.Job
		errMsg string
	}{
		{
			"invalid pipeline version reference",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 1,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}}},
				ResourceReferences: []*api.ResourceReference{
					{
						Key: &apiv1beta1.ResourceKey{
							Type: apiv1beta1.ResourceType_EXPERIMENT,
							Id:   DefaultFakeUUID,
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: invalidPipelineVersionId},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			"ResourceNotFoundError: PipelineVersion not_exist_pipeline_version not found",
		},
		{
			"missing pipeline spec",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 1,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}},
				},
				ResourceReferences: validReference,
			},
			"Failed to create a recurring run: Cannot create a job with an empty pipeline ID",
		},
		{
			"invalid pipeline spec",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 1,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}}},
				PipelineSpec: &apiv1beta1.PipelineSpec{
					PipelineId: "not_exist_pipeline",
					Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
				},
			},
			"Failed to fetch a pipeline version from pipeline not_exist_pipeline: Failed to get the latest " +
				"pipeline version as pipeline was not found: ResourceNotFoundError: Pipeline not_exist_pipeline not found",
		},
		{
			"invalid cron",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 1,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * ",
					}}},
				PipelineSpec: &apiv1beta1.PipelineSpec{
					WorkflowManifest: testWorkflow.ToStringForStore(),
					Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
				},
			},
			"Schedule cron is not a supported format(https://godoc.org/github.com/robfig/cron). Error: Expected 5 to 6 fields, found 3: 1 * * ",
		},
		{
			"max concur out of range",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 0,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}}},
				PipelineSpec: &apiv1beta1.PipelineSpec{
					WorkflowManifest: testWorkflow.ToStringForStore(),
					Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
				},
			},
			"Max concurrency of a recurring run must be at least 1 and at most 10. Received 0",
		},
		{
			"negative interval seconds",
			&apiv1beta1.Job{
				Name:           "job1",
				Enabled:        true,
				MaxConcurrency: 5,
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
			},
			"Found invalid period schedule interval -1. Set at interval to least 1 second",
		},
	}
	for _, tt := range tests {
		got, err := server.CreateJob(context.Background(), &apiv1beta1.CreateJobRequest{Job: tt.arg})
		assert.NotNil(t, err)
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		assert.Contains(t, errMsg, tt.errMsg)
		assert.Nil(t, got)
	}
}

func TestCreateJob_pipelineVersion(t *testing.T) {
	clients, manager, exp, pipelineVersion := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	server := createJobServerV1(manager)
	rr := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT,
				Id:   exp.UUID,
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_PIPELINE_VERSION,
				Id:   pipelineVersion.UUID,
			},
			Relationship: apiv1beta1.Relationship_CREATOR,
		},
	}
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: rr,
	}
	job, err := server.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)

	expectedJob := &apiv1beta1.Job{
		Id:             "123e4567-e89b-12d3-a456-426655440000",
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: rr,
		ServiceAccount:     "pipeline-runner",
		Status:             "STATUS_UNSPECIFIED",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:       "123e4567-e89b-12d3-a456-426655440000",
			PipelineName:     "p1",
			WorkflowManifest: testWorkflow.ToStringForStore(),
		},
	}
	matched := 0
	for _, resRef := range expectedJob.GetResourceReferences() {
		for _, resRef2 := range job.GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(rr), matched)
	expectedJob.ResourceReferences = job.GetResourceReferences()
	expectedJob.CreatedAt = job.GetCreatedAt()
	expectedJob.UpdatedAt = job.GetUpdatedAt()
	assert.Equal(t, expectedJob, job)
}

func TestCreateJob_NoResRefs(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdTwo, nil))
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := createJobServerV1(manager)
	apiJob := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
		// This job has no ResourceReferences, no experiment
	}
	job, err := server.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: apiJob})
	assert.Nil(t, err)
	rr := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT,
				Id:   DefaultFakeIdTwo,
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	expectedJob := &apiv1beta1.Job{
		Id:             DefaultFakeIdOne,
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: rr,
		ServiceAccount:     "pipeline-runner",
		Status:             "STATUS_UNSPECIFIED",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: testWorkflow.ToStringForStore(),
			Parameters:       []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
		},
	}
	matched := 0
	for _, resRef := range expectedJob.GetResourceReferences() {
		for _, resRef2 := range job.GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(rr), matched)
	expectedJob.ResourceReferences = job.GetResourceReferences()
	expectedJob.CreatedAt = job.GetCreatedAt()
	expectedJob.UpdatedAt = job.GetUpdatedAt()
	assert.Equal(t, expectedJob, job)
}

func TestCreateJob(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)
	job, err := server.CreateJob(nil, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)
	matched := 0
	for _, resRef := range commonExpectedJob.GetResourceReferences() {
		for _, resRef2 := range job.GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(commonExpectedJob.GetResourceReferences()), matched)
	commonExpectedJob.ResourceReferences = job.GetResourceReferences()

	commonExpectedJob.PipelineSpec.PipelineId = job.GetPipelineSpec().GetPipelineId()
	commonExpectedJob.PipelineSpec.PipelineName = job.GetPipelineSpec().GetPipelineName()
	commonExpectedJob.PipelineSpec.PipelineManifest = job.GetPipelineSpec().GetPipelineManifest()
	commonExpectedJob.CreatedAt = job.GetCreatedAt()
	commonExpectedJob.UpdatedAt = job.GetUpdatedAt()
	assert.Equal(t, commonExpectedJob, job)
}

func TestCreateJob_V2(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)
	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param2": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param3": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param4": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param5": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	apiJob_V2 := &apiv1beta1.Job{
		Name:           "job1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			WorkflowManifest: v2SpecHelloWorldParams,
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
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt: timestamppb.New(time.Unix(2, 0)),
		UpdatedAt: timestamppb.New(time.Unix(2, 0)),
		Status:    "STATUS_UNSPECIFIED",
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineManifest: v2SpecHelloWorldParams,
			WorkflowManifest: v2SpecHelloWorldParams,
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

	matched := 0
	for _, resRef := range expectedJob_V2.GetResourceReferences() {
		for _, resRef2 := range job.GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(expectedJob_V2.GetResourceReferences()), matched)
	expectedJob_V2.ResourceReferences = job.GetResourceReferences()

	expectedJob_V2.PipelineSpec.PipelineId = job.GetPipelineSpec().GetPipelineId()
	expectedJob_V2.PipelineSpec.PipelineName = job.GetPipelineSpec().GetPipelineName()
	expectedJob_V2.CreatedAt = job.GetCreatedAt()
	expectedJob_V2.UpdatedAt = job.GetUpdatedAt()
	job.PipelineSpec.RuntimeConfig.Parameters = v2RuntimeParams

	assert.Equal(t, expectedJob_V2, job)
}

func TestListRecurringRuns_MultiUser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: experiment.UUID,
	}

	_, err := server.CreateRecurringRun(ctx, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		Status:       apiv2beta1.RecurringRun_ENABLED,
		ExperimentId: experiment.UUID,
	}

	expectedRecurringRunsList := []*apiv2beta1.RecurringRun{expectedRecurringRun}

	// List API should fail in multi-user mode for empty requests
	actualRecurringRunsList, err := server.ListRecurringRuns(ctx, &apiv2beta1.ListRecurringRunsRequest{})
	assert.NotNil(t, err)
	assert.Nil(t, actualRecurringRunsList)

	actualRecurringRunsList2, err := server.ListRecurringRuns(ctx, &apiv2beta1.ListRecurringRunsRequest{
		ExperimentId: experiment.UUID,
	})
	actualRecurringRunsList2.RecurringRuns[0].RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Nil(t, err)
	assert.Equal(t, 1, len(actualRecurringRunsList2.RecurringRuns))
	assert.Equal(t, expectedRecurringRunsList, actualRecurringRunsList2.RecurringRuns)
}

func TestCreateJob_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()

	server := createJobServerV1(manager)
	_, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason",
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

	server := createJobServerV1(manager)
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server = createJobServerV1(manager)

	_, err = server.GetJob(ctx, &apiv1beta1.GetJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason",
	)
}

func TestGetJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)
	createdJob, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	job, err := server.GetJob(ctx, &apiv1beta1.GetJobRequest{Id: createdJob.Id})
	assert.Nil(t, err)
	matched := 0
	for _, resRef := range commonExpectedJob.GetResourceReferences() {
		for _, resRef2 := range job.GetResourceReferences() {
			if resRef.Key.Type == resRef2.Key.Type && resRef.Key.Id == resRef2.Key.Id && resRef.Relationship == resRef2.Relationship {
				matched++
			}
		}
	}
	assert.Equal(t, len(commonExpectedJob.GetResourceReferences()), matched)
	commonExpectedJob.ResourceReferences = job.GetResourceReferences()

	commonExpectedJob.PipelineSpec.PipelineId = job.GetPipelineSpec().GetPipelineId()
	commonExpectedJob.PipelineSpec.PipelineName = job.GetPipelineSpec().GetPipelineName()
	commonExpectedJob.PipelineSpec.PipelineManifest = job.GetPipelineSpec().GetPipelineManifest()
	commonExpectedJob.CreatedAt = job.GetCreatedAt()
	commonExpectedJob.UpdatedAt = job.GetUpdatedAt()
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

	server := createJobServerV1(manager)
	_, err := server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason",
	)

	_, err = server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"PermissionDenied: User 'user@google.com' is not authorized with reason",
	)
}

func TestListJobs_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)
	_, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	var expectedJobs []*apiv1beta1.Job
	commonExpectedJob.CreatedAt = timestamppb.New(time.Unix(2, 0))
	commonExpectedJob.UpdatedAt = timestamppb.New(time.Unix(2, 0))
	commonExpectedJob.ResourceReferences = []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"}, Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: DefaultFakeIdOne}, Relationship: apiv1beta1.Relationship_OWNER},
	}
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
			"Invalid - empty request",
			&apiv1beta1.ListJobsRequest{},
			true,
			"a recurring run cannot have an empty namespace in multi-user mode",
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
	server := createJobServerV1(manager)
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server = createJobServerV1(manager)

	_, err = server.EnableJob(ctx, &apiv1beta1.EnableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		" PermissionDenied: User 'user@google.com' is not authorized with reason",
	)
}

func TestEnableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)

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
	server := createJobServerV1(manager)
	job, err := server.CreateJob(ctx, &apiv1beta1.CreateJobRequest{Job: commonApiJob})
	assert.Nil(t, err)

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server = createJobServerV1(manager)

	_, err = server.DisableJob(ctx, &apiv1beta1.DisableJobRequest{Id: job.Id})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		" PermissionDenied: User 'user@google.com' is not authorized with reason",
	)
}

func TestDisableJob_Multiuser(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServerV1(manager)

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

	server := createJobServerV1(manager)
	_, err := server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_EXPERIMENT,
			Id:   experiment.UUID,
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"User identity is empty in the request header",
	)

	_, err = server.ListJobs(ctx, &apiv1beta1.ListJobsRequest{
		ResourceReferenceKey: &apiv1beta1.ResourceKey{
			Type: apiv1beta1.ResourceType_NAMESPACE,
			Id:   "ns1",
		},
	})
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"User identity is empty in the request header",
	)
}

func TestCreateRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	recurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		Status:         apiv2beta1.RecurringRun_ENABLED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}
	recurringRun.RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Equal(t, expectedRecurringRun, recurringRun)
}

func TestGetRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		Status:         apiv2beta1.RecurringRun_ENABLED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	recurringRun, err := server.GetRecurringRun(nil, &apiv2beta1.GetRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
	recurringRun.RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Equal(t, expectedRecurringRun, recurringRun)
}

func TestListRecurringRuns(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: experiment.UUID,
	}

	_, err := server.CreateRecurringRun(context.Background(), &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		Status:       apiv2beta1.RecurringRun_ENABLED,
		ExperimentId: experiment.UUID,
	}

	expectedRecurringRunsList := []*apiv2beta1.RecurringRun{expectedRecurringRun}

	actualRecurringRunsList, err := server.ListRecurringRuns(context.Background(), &apiv2beta1.ListRecurringRunsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(actualRecurringRunsList.RecurringRuns))
	actualRecurringRunsList.RecurringRuns[0].RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Equal(t, expectedRecurringRunsList, actualRecurringRunsList.RecurringRuns)

	actualRecurringRunsList2, err := server.ListRecurringRuns(context.Background(), &apiv2beta1.ListRecurringRunsRequest{
		ExperimentId: experiment.UUID,
	})
	actualRecurringRunsList2.RecurringRuns[0].RuntimeConfig.Parameters = map[string]*structpb.Value{
		"param1": structpb.NewStringValue("world"),
	}
	assert.Nil(t, err)
	assert.Equal(t, 1, len(actualRecurringRunsList2.RecurringRuns))
	assert.Equal(t, expectedRecurringRunsList, actualRecurringRunsList2.RecurringRuns)
}

func TestEnableRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	_, err = server.EnableRecurringRun(nil, &apiv2beta1.EnableRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
}

func TestDisableRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createJobServer(manager)

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * * * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
			Parameters: map[string]*structpb.Value{
				"param1": structpb.NewStringValue("world"),
			},
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	_, err = server.DisableRecurringRun(nil, &apiv2beta1.DisableRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
}
