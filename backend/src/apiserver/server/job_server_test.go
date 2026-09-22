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
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	authorizationv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/yaml"
)

var (
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

func TestCreateJob_PrivatePipelineVersionUnauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	privatePipeline, err := manager.CreatePipeline(&model.Pipeline{Name: "private-pipeline", Namespace: "ns2"})
	require.NoError(t, err)
	pipelineStore, ok := clients.PipelineStore().(*storage.PipelineStore)
	require.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(NonDefaultFakeUUID, nil))
	privateVersion, err := manager.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "private-version",
		PipelineId:   privatePipeline.UUID,
		PipelineSpec: model.LargeText(testIRPipeline),
	})
	require.NoError(t, err)
	require.NotEqual(t, privatePipeline.UUID, privateVersion.UUID)

	reviewClient := &recordingSubjectAccessReviewClient{
		authorize: func(attributes authorizationv1.ResourceAttributes) bool {
			return attributes.Resource != common.RbacResourceTypePipelines
		},
	}
	clients.SubjectAccessReviewClientFake = reviewClient
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	server := createJobServer(manager)
	_, err = server.createJob(ctx, &model.Job{
		DisplayName:  "private-pipeline-job",
		ExperimentId: experiment.UUID,
		Namespace:    experiment.Namespace,
		PipelineSpec: model.PipelineSpec{PipelineVersionId: privateVersion.UUID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PermissionDenied")
	require.Len(t, reviewClient.requests, 2)
	assert.Equal(t, authorizationv1.ResourceAttributes{
		Namespace: "ns2",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypePipelines,
		Name:      privatePipeline.Name,
	}, reviewClient.requests[1])
}

func TestCreateJob_PrivatePipelineVersionMustMatchDestinationNamespace(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	privatePipeline, err := manager.CreatePipeline(&model.Pipeline{Name: "private-pipeline", Namespace: "ns2"})
	require.NoError(t, err)
	pipelineStore, ok := clients.PipelineStore().(*storage.PipelineStore)
	require.True(t, ok)
	pipelineStore.SetUUIDGenerator(util.NewFakeUUIDGeneratorOrFatal(NonDefaultFakeUUID, nil))
	privateVersion, err := manager.CreatePipelineVersion(&model.PipelineVersion{
		Name:         "private-version",
		PipelineId:   privatePipeline.UUID,
		PipelineSpec: model.LargeText(testIRPipeline),
	})
	require.NoError(t, err)
	require.NotEqual(t, privatePipeline.UUID, privateVersion.UUID)

	// Model the scheduled-workflow controller, whose cluster-wide pipeline read
	// permission must not let a namespaced recurring-run resource reference a
	// different tenant's private pipeline.
	reviewClient := &recordingSubjectAccessReviewClient{allowed: true}
	clients.SubjectAccessReviewClientFake = reviewClient
	manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	server := createJobServer(manager)
	_, err = server.createJob(ctx, &model.Job{
		DisplayName:  "cross-namespace-pipeline-job",
		ExperimentId: experiment.UUID,
		Namespace:    experiment.Namespace,
		PipelineSpec: model.PipelineSpec{PipelineVersionId: privateVersion.UUID},
	})
	require.Error(t, err)
	assert.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
	assert.Contains(t, err.Error(), "private pipeline can only be referenced")
	require.Len(t, reviewClient.requests, 2)
	assert.Equal(t, common.RbacResourceTypeJobs, reviewClient.requests[0].Resource)
	assert.Equal(t, common.RbacResourceTypePipelines, reviewClient.requests[1].Resource)
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

func TestDeleteRecurringRun(t *testing.T) {
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

	createdRecurringRun, err := server.CreateRecurringRun(context.Background(), &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	_, err = server.DeleteRecurringRun(context.Background(), &apiv2beta1.DeleteRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)

	// Verify the recurring run is gone.
	_, err = server.GetRecurringRun(context.Background(), &apiv2beta1.GetRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}
