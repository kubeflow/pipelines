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
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/stretchr/testify/require"
)

type orderingMockAPI struct {
	*kfpapi.MockAPI
	order []string
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
