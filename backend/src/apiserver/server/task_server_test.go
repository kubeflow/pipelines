// Copyright 2025 The Kubeflow Authors
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

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func createTaskServer(resourceManager *resource.ResourceManager) *TaskServer {
	return &TaskServer{resourceManager: resourceManager}
}

func TestNewTaskServer(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewTaskServer(manager)
	assert.NotNil(t, server)
	assert.Equal(t, manager, server.resourceManager)
}

func TestCreateTaskV1_NilRequest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createTaskServer(manager)
	_, err := server.CreateTaskV1(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "CreateTaskRequest is nil")
}

func TestCreateTaskV1_IdSet(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createTaskServer(manager)
	_, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
		Task: &api.Task{
			Id:              "some-id",
			PipelineName:    "pipeline/my-pipeline",
			RunId:           "run-1",
			MlmdExecutionID: "exec-1",
			Fingerprint:     "abc123",
			CreatedAt:       timestamppb.New(time.Unix(1, 0)),
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Id should not be set")
}

func TestCreateTaskV1_MissingRequiredFields(t *testing.T) {
	createdAt := timestamppb.New(time.Unix(1, 0))
	tests := []struct {
		name          string
		task          *api.Task
		expectedError string
	}{
		{
			name: "missing PipelineName",
			task: &api.Task{
				RunId:           "run-1",
				MlmdExecutionID: "exec-1",
				Fingerprint:     "abc123",
				CreatedAt:       createdAt,
			},
			expectedError: "must specify PipelineName",
		},
		{
			name: "missing RunId",
			task: &api.Task{
				PipelineName:    "pipeline/my-pipeline",
				MlmdExecutionID: "exec-1",
				Fingerprint:     "abc123",
				CreatedAt:       createdAt,
			},
			expectedError: "must specify RunID",
		},
		{
			name: "missing MlmdExecutionID",
			task: &api.Task{
				PipelineName: "pipeline/my-pipeline",
				RunId:        "run-1",
				Fingerprint:  "abc123",
				CreatedAt:    createdAt,
			},
			expectedError: "must specify MlmdExecutionID",
		},
		{
			name: "missing Fingerprint",
			task: &api.Task{
				PipelineName:    "pipeline/my-pipeline",
				RunId:           "run-1",
				MlmdExecutionID: "exec-1",
				CreatedAt:       createdAt,
			},
			expectedError: "must specify FingerPrint",
		},
		{
			name: "missing CreatedAt",
			task: &api.Task{
				PipelineName:    "pipeline/my-pipeline",
				RunId:           "run-1",
				MlmdExecutionID: "exec-1",
				Fingerprint:     "abc123",
			},
			expectedError: "must specify CreatedAt",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			clients, manager, _ := initWithExperiment(t)
			defer clients.Close()
			server := createTaskServer(manager)
			_, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
				Task: testCase.task,
			})
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func TestCreateTaskV1_NamespacedPipeline_Invalid(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createTaskServer(manager)
	_, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
		Task: &api.Task{
			PipelineName:    "namespace/ns1",
			RunId:           "run-1",
			MlmdExecutionID: "exec-1",
			Fingerprint:     "abc123",
			CreatedAt:       timestamppb.New(time.Unix(1, 0)),
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid PipelineName for namespaced pipelines")
}

func TestCreateTaskV1_NamespacedPipeline_ConflictingNamespace(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createTaskServer(manager)
	_, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
		Task: &api.Task{
			Namespace:       "other-ns",
			PipelineName:    "namespace/ns1/pipeline/my-pipeline",
			RunId:           "run-1",
			MlmdExecutionID: "exec-1",
			Fingerprint:     "abc123",
			CreatedAt:       timestamppb.New(time.Unix(1, 0)),
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "namespace ns1 extracted from pipelineName is not equal to the namespace other-ns")
}

func TestCreateTaskV1(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := createTaskServer(manager)
	createdAt := timestamppb.New(time.Unix(1, 0))
	task, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
		Task: &api.Task{
			PipelineName:    "pipeline/my-pipeline",
			RunId:           run.UUID,
			MlmdExecutionID: "exec-1",
			Fingerprint:     "abc123",
			CreatedAt:       createdAt,
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, task)
	assert.NotEmpty(t, task.Id)
	assert.Equal(t, "pipeline/my-pipeline", task.PipelineName)
	assert.Equal(t, run.UUID, task.RunId)
	assert.Equal(t, "exec-1", task.MlmdExecutionID)
	assert.Equal(t, "abc123", task.Fingerprint)
}

func TestListTasksV1_Empty(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := createTaskServer(manager)
	response, err := server.ListTasksV1(context.Background(), &api.ListTasksRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Empty(t, response.Tasks)
	assert.Equal(t, int32(0), response.TotalSize)
}

func TestListTasksV1_AfterCreate(t *testing.T) {
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	// Reset UUID generator so the task gets a fresh UUID.
	clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdTwo, nil))
	server := createTaskServer(manager)
	createdAt := timestamppb.New(time.Unix(1, 0))
	_, err := server.CreateTaskV1(context.Background(), &api.CreateTaskRequest{
		Task: &api.Task{
			PipelineName:    "pipeline/my-pipeline",
			RunId:           run.UUID,
			MlmdExecutionID: "exec-1",
			Fingerprint:     "abc123",
			CreatedAt:       createdAt,
		},
	})
	assert.Nil(t, err)

	response, err := server.ListTasksV1(context.Background(), &api.ListTasksRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.Tasks))
	assert.Equal(t, int32(1), response.TotalSize)
	assert.Equal(t, "pipeline/my-pipeline", response.Tasks[0].PipelineName)
}

func TestTaskAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, namespace, pipeline       string
		authenticated, allowed, success bool
	}{
		{"owner namespace derived", "", "pipeline/p", true, true, true},
		{"explicit owner", "ns1", "namespace/ns1/pipeline/p", true, true, true},
		{"anonymous", "", "pipeline/p", false, true, false},
		{"unauthorized owner", "", "pipeline/p", true, false, false},
		{"forged namespace", "ns2", "pipeline/p", true, true, false},
		{"forged pipeline namespace", "", "namespace/ns2/pipeline/p", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set(common.MultiUserMode, true)
			defer viper.Set(common.MultiUserMode, false)
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			clients.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal(DefaultFakeIdTwo, nil))
			review := &recordingSubjectAccessReviewClient{allowed: tc.allowed}
			clients.SubjectAccessReviewClientFake = review
			manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
			ctx := context.Background()
			if tc.authenticated {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@example.com"))
			}
			created, err := NewTaskServer(manager).CreateTaskV1(ctx, &api.CreateTaskRequest{Task: &api.Task{
				RunId: run.UUID, Namespace: tc.namespace, PipelineName: tc.pipeline, MlmdExecutionID: "1", Fingerprint: "fingerprint", CreatedAt: timestamppb.Now(),
			}})
			if tc.success {
				require.NoError(t, err)
				require.Equal(t, "ns1", created.Namespace)
			} else {
				require.Error(t, err)
			}
			_, count, _, err := clients.TaskStore().ListTasks(model.EmptyFilterContext(), list.EmptyOptions())
			require.NoError(t, err)
			if tc.success {
				require.Equal(t, 1, count)
			} else {
				require.Zero(t, count)
			}
			if tc.authenticated {
				require.Len(t, review.requests, 1)
				require.Equal(t, authorizationv1.ResourceAttributes{Group: common.RbacPipelinesGroup, Version: common.RbacPipelinesVersion, Resource: common.RbacResourceTypeRuns, Verb: common.RbacResourceVerbCreateTask, Namespace: "ns1", Name: run.K8SName}, review.requests[0])
			}
		})
	}
}

func TestTaskListAuthorization(t *testing.T) {
	clients, _, _ := initWithOneTimeRun(t)
	defer clients.Close()
	review := &recordingSubjectAccessReviewClient{authorize: func(a authorizationv1.ResourceAttributes) bool { return a.Namespace == "ns1" }}
	clients.SubjectAccessReviewClientFake = review
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	viper.Set(common.MultiUserMode, true)
	defer viper.Set(common.MultiUserMode, false)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@example.com"))
	server := NewTaskServer(manager)
	for _, key := range []*api.ResourceKey{nil, {Type: api.ResourceType_NAMESPACE}, {Type: api.ResourceType_PIPELINE, Id: "p"}, {Type: api.ResourceType_NAMESPACE, Id: "ns2"}} {
		_, err := server.ListTasksV1(ctx, &api.ListTasksRequest{ResourceReferenceKey: key})
		require.Error(t, err)
	}
	for _, key := range []*api.ResourceKey{{Type: api.ResourceType_NAMESPACE, Id: "ns1"}} {
		_, err := server.ListTasksV1(context.Background(), &api.ListTasksRequest{ResourceReferenceKey: key})
		require.Error(t, err)
		_, err = server.ListTasksV1(ctx, &api.ListTasksRequest{ResourceReferenceKey: key})
		require.NoError(t, err)
	}
}

func TestTaskCreationLegacyRunNamespace(t *testing.T) {
	for _, tc := range []struct {
		name              string
		missingExperiment bool
	}{
		{"derive from experiment", false}, {"missing owner fails closed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set(common.MultiUserMode, true)
			defer viper.Set(common.MultiUserMode, false)
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			legacy := *run
			legacy.UUID = DefaultFakeIdTwo
			legacy.Namespace = ""
			if tc.missingExperiment {
				legacy.ExperimentId = ""
			}
			_, err := clients.RunStore().CreateRun(&legacy)
			require.NoError(t, err)
			review := &recordingSubjectAccessReviewClient{allowed: true}
			clients.SubjectAccessReviewClientFake = review
			manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@example.com"))
			created, err := NewTaskServer(manager).CreateTaskV1(ctx, &api.CreateTaskRequest{Task: &api.Task{
				RunId: legacy.UUID, PipelineName: "pipeline/p", MlmdExecutionID: "1", Fingerprint: "fp", CreatedAt: timestamppb.Now(),
			}})
			if tc.missingExperiment {
				require.Error(t, err)
				require.Empty(t, review.requests)
			} else {
				require.NoError(t, err)
				require.Equal(t, "ns1", created.Namespace)
				require.Len(t, review.requests, 1)
				require.Equal(t, "ns1", review.requests[0].Namespace)
			}
		})
	}
}
