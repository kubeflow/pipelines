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
	"errors"
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

func TestBatchUpdater_FlushKeepsDistinctArtifactTasksForDifferentProducerIterations(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	updater := NewBatchUpdater()
	iteration0 := int64(0)
	iteration1 := int64(1)

	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "loop-body", Iteration: &iteration0},
	})
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "artifact-1",
		TaskId:     "task-1",
		RunId:      "run-1",
		Key:        "model",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "loop-body", Iteration: &iteration1},
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

// The mocks and tests below provide deterministic API-disruption coverage for
// the native runtime finalization path (issue #14424). They inject faults into
// the BatchUpdater.Flush staging (artifacts -> artifact-tasks -> terminal task
// update) using explicit failure controls rather than timing-dependent sleeps.
// They assert the updater's existing bounded behavior and intentionally do not
// introduce or assert new retry/idempotency contracts for append-style writes.

// failingTaskUpdateMockAPI fails the terminal task-state update on its first
// call without persisting it, then succeeds. It models a disruption that
// interrupts finalization after artifacts and links are durable but before the
// task is marked terminal.
type failingTaskUpdateMockAPI struct {
	*kfpapi.MockAPI
	createArtifactsBulkCalls int
	createArtifactTaskCalls  int
	updateTasksBulkCalls     int
}

func (m *failingTaskUpdateMockAPI) CreateArtifactsBulk(ctx context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	m.createArtifactsBulkCalls++
	return m.MockAPI.CreateArtifactsBulk(ctx, req)
}

func (m *failingTaskUpdateMockAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	m.createArtifactTaskCalls++
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func (m *failingTaskUpdateMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.updateTasksBulkCalls++
	if m.updateTasksBulkCalls == 1 {
		return nil, fmt.Errorf("transient task-update failure")
	}
	return m.MockAPI.UpdateTasksBulk(ctx, req)
}

// lostResponseTaskUpdateMockAPI commits the terminal task-state update to the
// backing store on its first call but then returns an error, modeling a write
// that succeeds server-side while its response is lost in transit.
type lostResponseTaskUpdateMockAPI struct {
	*kfpapi.MockAPI
	updateTasksBulkCalls int
}

func (m *lostResponseTaskUpdateMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.updateTasksBulkCalls++
	resp, err := m.MockAPI.UpdateTasksBulk(ctx, req)
	if m.updateTasksBulkCalls == 1 {
		// Write committed on the server, but the response never reaches the caller.
		return nil, fmt.Errorf("committed task-update whose response was lost")
	}
	return resp, err
}

// restartingArtifactMockAPI fails the first artifact write before persisting
// anything, then succeeds. It models the API server restarting during a write
// so the retry starts from a clean state.
type restartingArtifactMockAPI struct {
	*kfpapi.MockAPI
	createArtifactsBulkCalls int
	createArtifactTaskCalls  int
	updateTasksBulkCalls     int
}

func (m *restartingArtifactMockAPI) CreateArtifactsBulk(ctx context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	m.createArtifactsBulkCalls++
	if m.createArtifactsBulkCalls == 1 {
		return nil, fmt.Errorf("api server restarted before artifact write committed")
	}
	return m.MockAPI.CreateArtifactsBulk(ctx, req)
}

func (m *restartingArtifactMockAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	m.createArtifactTaskCalls++
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func (m *restartingArtifactMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.updateTasksBulkCalls++
	return m.MockAPI.UpdateTasksBulk(ctx, req)
}

// hangingArtifactMockAPI blocks the first artifact write until the caller's
// context is cancelled, then returns the context error. It models a call that
// is accepted but whose response hangs, so the caller must rely on
// cancellation/deadline to make progress. The started channel lets the test
// cancel deterministically without sleeping.
type hangingArtifactMockAPI struct {
	*kfpapi.MockAPI
	started                  chan struct{}
	createArtifactsBulkCalls int
	createArtifactTaskCalls  int
	updateTasksBulkCalls     int
}

func (m *hangingArtifactMockAPI) CreateArtifactsBulk(ctx context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	m.createArtifactsBulkCalls++
	close(m.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (m *hangingArtifactMockAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	m.createArtifactTaskCalls++
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func (m *hangingArtifactMockAPI) UpdateTasksBulk(ctx context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	m.updateTasksBulkCalls++
	return m.MockAPI.UpdateTasksBulk(ctx, req)
}

// seedFinalizationUpdater returns an updater queued with one artifact, one
// artifact-task linking it to taskID, and one terminal (SUCCEEDED) task update,
// with taskID pre-created in RUNNING state. This is the shape of a component
// launcher's end-of-execution finalization flush.
func seedFinalizationUpdater(t *testing.T, api kfpapi.API, taskID string) *BatchUpdater {
	t.Helper()
	_, err := api.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			TaskId: taskID,
			RunId:  "run-1",
			Name:   taskID,
			State:  apiv2beta1.PipelineTask_RUNNING,
		},
	})
	require.NoError(t, err)

	updater := NewBatchUpdater()
	updater.QueueArtifact(&apiv2beta1.CreateArtifactRequest{
		RunId:       "run-1",
		TaskId:      taskID,
		ProducerKey: "model",
		Artifact:    &apiv2beta1.Artifact{Name: "model"},
	})
	updater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		TaskId: taskID,
		RunId:  "run-1",
		Key:    "model",
		Type:   apiv2beta1.IOType_OUTPUT,
	})
	updater.QueueTaskUpdate(&apiv2beta1.PipelineTask{
		TaskId: taskID,
		RunId:  "run-1",
		State:  apiv2beta1.PipelineTask_SUCCEEDED,
	})
	return updater
}

// Scenario: finalization is interrupted between durable artifact/link writes and
// the terminal task-state update. The task must not be left silently successful
// with missing outputs, and the retry must touch only the remaining task update
// without re-creating the already-durable artifacts and links.
func TestBatchUpdater_FlushDoesNotMarkTaskTerminalWhenTaskUpdateFails(t *testing.T) {
	mockAPI := &failingTaskUpdateMockAPI{MockAPI: kfpapi.NewMockAPI()}
	updater := seedFinalizationUpdater(t, mockAPI, "task-1")

	err := updater.Flush(context.Background(), mockAPI)
	require.Error(t, err)
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 1, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 1, mockAPI.updateTasksBulkCalls)

	// Artifacts and links are durable; only the terminal task update is retained.
	require.Empty(t, updater.artifacts)
	require.Empty(t, updater.artifactTasks)
	require.Len(t, updater.taskUpdates, 1)

	// No silent success: the task is still RUNNING, not SUCCEEDED.
	task, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-1", RunId: "run-1"})
	require.NoError(t, err)
	require.Equal(t, apiv2beta1.PipelineTask_RUNNING, task.GetState())

	linksBefore, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)

	// Retry touches only the terminal task update.
	require.NoError(t, updater.Flush(context.Background(), mockAPI))
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 1, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 2, mockAPI.updateTasksBulkCalls)

	linksAfter, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, linksAfter.GetArtifactTasks(), len(linksBefore.GetArtifactTasks()))

	task, err = mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-1", RunId: "run-1"})
	require.NoError(t, err)
	require.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, task.GetState())
}

// Scenario: a replay-approved terminal task update commits server-side, but its
// response is lost. Because task updates are keyed by task ID, replaying the
// same logical update converges to the same terminal state without producing
// duplicate artifact or link records.
func TestBatchUpdater_FlushTaskUpdateReplayIsIdempotentAfterLostResponse(t *testing.T) {
	mockAPI := &lostResponseTaskUpdateMockAPI{MockAPI: kfpapi.NewMockAPI()}
	updater := seedFinalizationUpdater(t, mockAPI, "task-1")

	// First flush: the write commits but the caller observes an error.
	err := updater.Flush(context.Background(), mockAPI)
	require.Error(t, err)
	require.Equal(t, 1, mockAPI.updateTasksBulkCalls)

	// The committed terminal state is already visible in the store.
	task, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-1", RunId: "run-1"})
	require.NoError(t, err)
	require.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, task.GetState())

	// Only the task update remains queued for replay; durable stages are cleared.
	require.Empty(t, updater.artifacts)
	require.Empty(t, updater.artifactTasks)
	require.Len(t, updater.taskUpdates, 1)

	linksBefore, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)

	// Replay converges to the same logical result without duplicate records.
	require.NoError(t, updater.Flush(context.Background(), mockAPI))
	require.Equal(t, 2, mockAPI.updateTasksBulkCalls)

	linksAfter, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, linksAfter.GetArtifactTasks(), len(linksBefore.GetArtifactTasks()))

	tasks, err := mockAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{RunId: "run-1"})
	require.NoError(t, err)
	require.Len(t, tasks.GetTasks(), 1)
	require.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, tasks.GetTasks()[0].GetState())
}

// Scenario: the API server restarts during the artifact write and the call fails
// before anything is committed. The retry resumes from the uncommitted head,
// completes the full flush, and leaves no duplicate records.
func TestBatchUpdater_FlushRestartBeforeArtifactCommitRetriesCleanly(t *testing.T) {
	mockAPI := &restartingArtifactMockAPI{MockAPI: kfpapi.NewMockAPI()}
	updater := seedFinalizationUpdater(t, mockAPI, "task-1")

	// First flush fails at the artifact head; no later stages run.
	err := updater.Flush(context.Background(), mockAPI)
	require.Error(t, err)
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 0, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 0, mockAPI.updateTasksBulkCalls)

	// The whole batch is retained because nothing committed.
	require.Len(t, updater.artifacts, 1)
	require.Len(t, updater.artifactTasks, 1)
	require.Len(t, updater.taskUpdates, 1)

	// Retry completes the full finalization exactly once.
	require.NoError(t, updater.Flush(context.Background(), mockAPI))
	require.Equal(t, 2, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 1, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 1, mockAPI.updateTasksBulkCalls)
	require.Empty(t, updater.artifacts)
	require.Empty(t, updater.artifactTasks)
	require.Empty(t, updater.taskUpdates)

	// Exactly one committed artifact: its auto-generated link plus the explicit
	// link. A duplicated artifact write would leave additional records.
	links, err := mockAPI.ListArtifactTasks(context.Background(), &apiv2beta1.ListArtifactTasksRequest{})
	require.NoError(t, err)
	require.Len(t, links.GetArtifactTasks(), 2)

	task, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-1", RunId: "run-1"})
	require.NoError(t, err)
	require.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, task.GetState())
}

// Scenario: an accepted call hangs without responding. The caller's context
// cancellation must unblock Flush, which surfaces the cancellation and preserves
// the full batch so a later attempt can replay it.
func TestBatchUpdater_FlushHonorsContextCancellationOnHungCall(t *testing.T) {
	mockAPI := &hangingArtifactMockAPI{MockAPI: kfpapi.NewMockAPI(), started: make(chan struct{})}
	updater := seedFinalizationUpdater(t, mockAPI, "task-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- updater.Flush(ctx, mockAPI)
	}()

	// Wait until the hung call is in flight, then cancel deterministically.
	<-mockAPI.started
	cancel()

	err := <-errCh
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))

	// The hung stage was the only one attempted.
	require.Equal(t, 1, mockAPI.createArtifactsBulkCalls)
	require.Equal(t, 0, mockAPI.createArtifactTaskCalls)
	require.Equal(t, 0, mockAPI.updateTasksBulkCalls)

	// The full batch is preserved for a later replay; nothing was committed.
	require.Len(t, updater.artifacts, 1)
	require.Len(t, updater.artifactTasks, 1)
	require.Len(t, updater.taskUpdates, 1)

	task, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-1", RunId: "run-1"})
	require.NoError(t, err)
	require.Equal(t, apiv2beta1.PipelineTask_RUNNING, task.GetState())
}
