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
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
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
