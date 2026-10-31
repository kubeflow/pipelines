// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"sort"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/require"
)

func createRunningTaskForBulkTest(t *testing.T, runSrv *RunServer, runID, name string) string {
	t.Helper()
	task, err := runSrv.CreateTask(context.Background(), &apiv2.CreateTaskRequest{
		RunId: runID,
		Task: &apiv2.PipelineTask{
			RunId: runID,
			Name:  name,
			State: apiv2.PipelineTask_RUNNING,
		},
	})
	require.NoError(t, err)
	return task.GetTaskId()
}

func getTaskForBulkTest(t *testing.T, runSrv *RunServer, runID, taskID string) *apiv2.PipelineTask {
	t.Helper()
	task, err := runSrv.GetTask(context.Background(), &apiv2.GetTaskRequest{RunId: runID, TaskId: taskID})
	require.NoError(t, err)
	return task
}

// UpdateTasksBulk validates the whole request first, then applies one task
// update at a time in sorted task-ID order, each in its own transaction. It is
// therefore not atomic: a failure after the first update leaves the earlier
// updates committed and returns no response. These tests pin that contract and
// show that replaying the corrected request converges, because task updates
// merge by field and only append state history when the state changes.
func TestUpdateTasksBulk_FailureAfterFirstUpdateIsPartialAndReplaySafe(t *testing.T) {
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()
	runSrv := createRunServer(manager)

	idA := createRunningTaskForBulkTest(t, runSrv, runID, "bulk-partial-a")
	idB := createRunningTaskForBulkTest(t, runSrv, runID, "bulk-partial-b")

	// Updates are applied in sorted task-ID order, so the task with the larger ID
	// is always second and is the one that fails.
	names := map[string]string{idA: "bulk-partial-a", idB: "bulk-partial-b"}
	ids := []string{idA, idB}
	sort.Strings(ids)
	firstID, secondID := ids[0], ids[1]

	failingRequest := &apiv2.UpdateTasksBulkRequest{
		RunId: runID,
		Tasks: map[string]*apiv2.PipelineTask{
			firstID: {
				TaskId: firstID,
				Name:   names[firstID],
				State:  apiv2.PipelineTask_SUCCEEDED,
			},
			secondID: {
				TaskId: secondID,
				Name:   names[secondID],
				State:  apiv2.PipelineTask_SUCCEEDED,
				// Passes the up-front validation but fails inside the apply loop,
				// after the first task has already been committed.
				ParentTaskId: strPTR("task-that-does-not-exist"),
			},
		},
	}

	resp, err := runSrv.UpdateTasksBulk(context.Background(), failingRequest)
	require.Error(t, err)
	require.Nil(t, resp)

	// Partial completion: the first update committed, the second did not.
	require.Equal(t, apiv2.PipelineTask_SUCCEEDED, getTaskForBulkTest(t, runSrv, runID, firstID).GetState())
	require.Equal(t, apiv2.PipelineTask_RUNNING, getTaskForBulkTest(t, runSrv, runID, secondID).GetState())
	historyAfterFailure := len(getTaskForBulkTest(t, runSrv, runID, firstID).GetStateHistory())

	// Replaying the corrected request converges: the already-applied update is
	// accepted again and the remaining update is applied.
	replayRequest := &apiv2.UpdateTasksBulkRequest{
		RunId: runID,
		Tasks: map[string]*apiv2.PipelineTask{
			firstID: {
				TaskId: firstID,
				Name:   names[firstID],
				State:  apiv2.PipelineTask_SUCCEEDED,
			},
			secondID: {
				TaskId: secondID,
				Name:   names[secondID],
				State:  apiv2.PipelineTask_SUCCEEDED,
			},
		},
	}
	resp, err = runSrv.UpdateTasksBulk(context.Background(), replayRequest)
	require.NoError(t, err)
	require.Len(t, resp.GetTasks(), 2)

	require.Equal(t, apiv2.PipelineTask_SUCCEEDED, getTaskForBulkTest(t, runSrv, runID, firstID).GetState())
	require.Equal(t, apiv2.PipelineTask_SUCCEEDED, getTaskForBulkTest(t, runSrv, runID, secondID).GetState())

	// Re-applying an identical state does not append duplicate history.
	require.Len(t, getTaskForBulkTest(t, runSrv, runID, firstID).GetStateHistory(), historyAfterFailure)
}

func TestUpdateTasksBulk_ValidationFailureLeavesEveryTaskUnchanged(t *testing.T) {
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()
	runSrv := createRunServer(manager)

	idA := createRunningTaskForBulkTest(t, runSrv, runID, "bulk-validation-a")
	idB := createRunningTaskForBulkTest(t, runSrv, runID, "bulk-validation-b")

	// A mismatched run ID is rejected during up-front validation, before any
	// task is updated.
	resp, err := runSrv.UpdateTasksBulk(context.Background(), &apiv2.UpdateTasksBulkRequest{
		RunId: runID,
		Tasks: map[string]*apiv2.PipelineTask{
			idA: {TaskId: idA, Name: "bulk-validation-a", State: apiv2.PipelineTask_SUCCEEDED},
			idB: {TaskId: idB, Name: "bulk-validation-b", State: apiv2.PipelineTask_SUCCEEDED, RunId: "some-other-run"},
		},
	})
	require.Error(t, err)
	require.Nil(t, resp)

	require.Equal(t, apiv2.PipelineTask_RUNNING, getTaskForBulkTest(t, runSrv, runID, idA).GetState())
	require.Equal(t, apiv2.PipelineTask_RUNNING, getTaskForBulkTest(t, runSrv, runID, idB).GetState())
}
