// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"fmt"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
)

type orderingMockAPI struct {
	*kfpapi.MockAPI
	order []string
}

type flakyArtifactTaskMockAPI struct {
	*kfpapi.MockAPI
	createArtifactsBulkCalls int
	createArtifactTaskCalls  int
	updateTasksBulkCalls     int
}

func (m *orderingMockAPI) CreateArtifactsBulk(ctx context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	m.order = append(m.order, "artifacts")
	return m.MockAPI.CreateArtifactsBulk(ctx, req)
}

func (m *orderingMockAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	m.order = append(m.order, "artifact-tasks")
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func (m *orderingMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.order = append(m.order, "tasks")
	return m.MockAPI.UpdateTasksBulk(ctx, req)
}

func (m *flakyArtifactTaskMockAPI) CreateArtifactsBulk(ctx context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	m.createArtifactsBulkCalls++
	return m.MockAPI.CreateArtifactsBulk(ctx, req)
}

func (m *flakyArtifactTaskMockAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	m.createArtifactTaskCalls++
	if m.createArtifactTaskCalls == 1 {
		return nil, fmt.Errorf("transient artifact-task failure")
	}
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func (m *flakyArtifactTaskMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.updateTasksBulkCalls++
	return m.MockAPI.UpdateTasksBulk(ctx, req)
}

func TestBatchUpdater_FlushDeduplicatesArtifactTasks(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	updater := NewBatchUpdater()

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_OUTPUT,
	}

	updater.QueueArtifactTask(artifactTask)
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_OUTPUT,
	})

	require.NoError(t, updater.Flush(context.Background(), mockAPI))

	resp, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, resp.ArtifactTasks, 1)
}

func TestBatchUpdater_FlushKeepsDistinctArtifactTasksForDifferentKeys(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	updater := NewBatchUpdater()

	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "input-a",
		Type:       apiv2beta1.IOType_COMPONENT_INPUT,
	})
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "input-b",
		Type:       apiv2beta1.IOType_COMPONENT_INPUT,
	})

	require.NoError(t, updater.Flush(context.Background(), mockAPI))

	resp, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, resp.ArtifactTasks, 2)
}

func TestBatchUpdater_QueueTaskUpdateDoesNotSelfDuplicateOutputs(t *testing.T) {
	updater := NewBatchUpdater()
	task := &apiv2beta1.PipelineTask{
		TaskId: "task-1",
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
				{ParameterKey: "result"},
			},
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
				{ArtifactKey: "model"},
			},
		},
	}

	updater.QueueTaskUpdate(task)
	updater.QueueTaskUpdate(task)

	require.Len(t, updater.taskUpdates["task-1"].GetOutputs().GetParameters(), 1)
	require.Len(t, updater.taskUpdates["task-1"].GetOutputs().GetArtifacts(), 1)
}

func TestBatchUpdater_FlushCreatesArtifactLinksBeforeTerminalTaskUpdate(t *testing.T) {
	mockAPI := &orderingMockAPI{MockAPI: kfpapi.NewMockAPI()}
	updater := NewBatchUpdater()
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			TaskId: "task-1",
			RunId:  "run-1",
			Name:   "task-1",
			State:  apiv2beta1.PipelineTask_RUNNING,
		},
	})
	require.NoError(t, err)

	updater.QueueArtifact(&apiv2beta1.CreateArtifactRequest{
		RunId:       "run-1",
		TaskId:      "task-1",
		ProducerKey: "model",
		Artifact: &apiv2beta1.Artifact{
			Name: "model",
		},
	})
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		TaskId: "task-1",
		RunId:  "run-1",
		Key:    "model",
		Type:   apiv2beta1.IOType_OUTPUT,
	})
	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: "task-1",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_SUCCEEDED,
	})

	require.NoError(t, updater.Flush(context.Background(), mockAPI))
	require.Equal(t, []string{"artifacts", "artifact-tasks", "tasks"}, mockAPI.order)
}

func TestBatchUpdater_FlushRetriesOnlyFailedTail(t *testing.T) {
	mockAPI := &flakyArtifactTaskMockAPI{MockAPI: kfpapi.NewMockAPI()}
	updater := NewBatchUpdater()
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			TaskId: "task-parent",
			RunId:  "run-1",
			Name:   "task-parent",
			State:  apiv2beta1.PipelineTask_RUNNING,
		},
	})
	require.NoError(t, err)

	updater.QueueArtifact(&apiv2beta1.CreateArtifactRequest{
		RunId:       "run-1",
		TaskId:      "task-output",
		ProducerKey: "model",
		Artifact: &apiv2beta1.Artifact{
			Name: "model",
		},
	})
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		TaskId: "task-parent",
		RunId:  "run-1",
		Key:    "model",
		Type:   apiv2beta1.IOType_OUTPUT,
	})
	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: "task-parent",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_SUCCEEDED,
	})

	err = updater.Flush(context.Background(), mockAPI)
	require.Error(t, err)
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 1, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 0, mockAPI.updateTasksBulkCalls)
	require.Empty(t, updater.artifacts)
	require.Len(t, updater.artifactTasks, 1)
	require.Len(t, updater.taskUpdates, 1)

	require.NoError(t, updater.Flush(context.Background(), mockAPI))
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 2, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 1, mockAPI.updateTasksBulkCalls)
	require.Empty(t, updater.artifacts)
	require.Empty(t, updater.artifactTasks)
	require.Empty(t, updater.taskUpdates)
}

func TestBatchUpdater_QueueTaskUpdateClonesAndMergesStatusMetadata(t *testing.T) {
	updater := NewBatchUpdater()
	originalTask := &apiv2beta1.PipelineTask{
		TaskId: "task-1",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_RUNNING,
	}

	updater.QueueTaskUpdate(originalTask)
	originalTask.State = apiv2beta1.PipelineTask_FAILED
	originalTask.StatusMetadata = &apiv2beta1.PipelineTask_StatusMetadata{Message: "mutated-after-queue"}

	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: "task-1",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_FAILED,
		StatusMetadata: &apiv2beta1.PipelineTask_StatusMetadata{
			Message: "final failure",
		},
	})

	queuedTask := updater.taskUpdates["task-1"]
	require.NotNil(t, queuedTask)
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, queuedTask.GetState())
	require.NotNil(t, queuedTask.GetStatusMetadata())
	require.Equal(t, "final failure", queuedTask.GetStatusMetadata().GetMessage())
}

func TestBatchUpdater_GetMetricsAccumulatesAcrossFlushes(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	updater := NewBatchUpdater()
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			TaskId: "task-1",
			RunId:  "run-1",
			Name:   "task-1",
			State:  apiv2beta1.PipelineTask_RUNNING,
		},
	})
	require.NoError(t, err)

	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: "task-1",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_SUCCEEDED,
	})
	require.NoError(t, updater.Flush(context.Background(), mockAPI))

	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: "task-1",
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_FAILED,
	})
	require.NoError(t, updater.Flush(context.Background(), mockAPI))

	metrics := updater.GetMetrics()
	require.Equal(t, 2, metrics["actual_task_update_calls"])
}
