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

package storage

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeManualRetryPreservesReusableTasksAndRestoresWaiting(t *testing.T) {
	db, tasks, runtime, source := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = '1'")
	require.NoError(t, err)
	var ids []string
	reused := make(map[string]*model.Task)
	for _, state := range []apiv2.PipelineTask_TaskState{
		apiv2.PipelineTask_FAILED, apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_SKIPPED,
	} {
		task, err := runtime.CreateTask(pendingTaskFixture(state.String()))
		require.NoError(t, err)
		var links []*model.ArtifactTask
		if state == apiv2.PipelineTask_CACHED {
			_, link := cachedOutputFixture(t, tasks, source, task)
			links = []*model.ArtifactTask{link}
		}
		require.NoError(t, runtime.FinalizePending(task.UUID, &model.Task{
			UUID: task.UUID, State: model.TaskStatus(state), FinishedInSec: 123,
			StatusMetadata: model.JSONData{"diagnostic": state.String()},
		}, links))
		ids = append(ids, task.UUID)
		if state != apiv2.PipelineTask_FAILED {
			reused[task.UUID], err = tasks.GetTask(task.UUID)
			require.NoError(t, err)
		}
	}
	active, err := runtime.CreateTask(pendingTaskFixture("active"))
	require.NoError(t, err)
	row := runtimeAttemptFixture(active, "old-attempt")
	row.RetryCount = 3
	eligible := int64(123)
	row.NextEligibleAt = &eligible
	// Seed prior retry progress without consuming three attempts in the fixture.
	require.NoError(t, runtime.Update(runtimeAttemptFixture(active, "old-attempt"), ""))
	_, err = db.Exec("UPDATE runtime_tasks SET RetryCount = ?, NextEligibleAt = ? WHERE TaskUUID = ?", row.RetryCount, eligible, active.UUID)
	require.NoError(t, err)
	ids = append(ids, active.UUID)
	require.NoError(t, tasks.ResetTasksForRetry(ids))
	for _, id := range ids {
		public, err := tasks.GetTask(id)
		require.NoError(t, err)
		current, err := runtime.Get(id)
		if original := reused[id]; original != nil {
			require.ErrorIs(t, err, sql.ErrNoRows)
			assert.Equal(t, original, public)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
		assert.Zero(t, public.FinishedInSec)
		assert.Equal(t, &model.RuntimeTask{TaskUUID: id, RunUUID: "1", State: model.RuntimeTaskWaiting}, current)
	}
	require.ErrorIs(t, runtime.Update(row, "old-attempt"), ErrRuntimeTaskConflict)
	require.NoError(t, runtime.Update(runtimeAttemptFixture(active, "new-attempt"), ""),
		"a manual retry's fresh NULL attempt must be ready for first-attempt assignment")
}

func TestRuntimeManualRetryClearsFinalizedAttemptOutputs(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = '1'")
	require.NoError(t, err)
	attempt := runtimeAttemptFixture(task, "old")
	require.NoError(t, runtime.Create(attempt))
	require.NoError(t, runtime.Checkpoint(attempt, runtimeOutputUpdate(t, task)))
	require.NoError(t, runtime.Finalize(task.UUID, "old", &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
	}))
	require.NoError(t, tasks.ResetTasksForRetry([]string{task.UUID}))
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Empty(t, public.OutputParameters)
	assert.Empty(t, public.OutputArtifactsHydrated, "finalized failures must not retain old attempt output links")
	history, err := json.Marshal(public.StateHistory)
	require.NoError(t, err)
	assert.Contains(t, string(history), "worker result", "finalization must not make retry discard previous task diagnostics")
	assert.Contains(t, string(history), "attempt-log")
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, &model.RuntimeTask{TaskUUID: task.UUID, RunUUID: task.RunUUID, State: model.RuntimeTaskWaiting}, current)
}

func TestRuntimeManualRetryRollback(t *testing.T) {
	for _, failure := range []string{"runtime-insert", "runtime-update", "public-update"} {
		t.Run(failure, func(t *testing.T) {
			db, tasks, runtime, task := runtimeFixture(t)
			_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = '1'")
			require.NoError(t, err)
			attempt := runtimeAttemptFixture(task, "old")
			require.NoError(t, runtime.Create(attempt))
			require.NoError(t, runtime.Checkpoint(attempt, runtimeOutputUpdate(t, task)))
			finalized, err := runtime.CreateTask(pendingTaskFixture("finalized-failure"))
			require.NoError(t, err)
			require.NoError(t, runtime.FinalizePending(finalized.UUID, &model.Task{
				UUID: finalized.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
			}, nil))
			_, err = db.Exec("UPDATE tasks SET CreatedAtInSec = 1 WHERE UUID = ?", task.UUID)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE tasks SET CreatedAtInSec = 2 WHERE UUID = ?", finalized.UUID)
			require.NoError(t, err)
			before, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			finalizedBefore, err := tasks.GetTask(finalized.UUID)
			require.NoError(t, err)
			trigger := map[string]string{
				"runtime-insert": "BEFORE INSERT ON runtime_tasks",
				"runtime-update": "BEFORE UPDATE ON runtime_tasks",
				"public-update":  "BEFORE UPDATE ON tasks",
			}[failure]
			_, err = db.Exec("CREATE TRIGGER reject_retry " + trigger + " BEGIN SELECT RAISE(ABORT, 'retry unavailable'); END")
			require.NoError(t, err)
			require.ErrorContains(t, tasks.ResetTasksForRetry([]string{task.UUID, finalized.UUID}), "retry unavailable")
			current, err := runtime.Get(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, attempt, current)
			_, err = runtime.Get(finalized.UUID)
			require.ErrorIs(t, err, sql.ErrNoRows)
			after, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after, "attempt diagnostics and artifact links must roll back with runtime reset")
			finalizedAfter, err := tasks.GetTask(finalized.UUID)
			require.NoError(t, err)
			assert.Equal(t, finalizedBefore, finalizedAfter, "all tasks in a batch must roll back together")
		})
	}
}

func TestRuntimeManualRetryCancellationAndLeaseFences(t *testing.T) {
	db, tasks, runtime, _ := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = '1'")
	require.NoError(t, err)
	public, err := runtime.CreateTask(pendingTaskFixture("retry-fenced"))
	require.NoError(t, err)
	token, err := tasks.RunLeaseStore().Acquire("1", "worker", time.Minute, false)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	foreign, err := tasks.CreateTask(&model.Task{RunUUID: "2", Name: "foreign"})
	require.NoError(t, err)
	require.ErrorIs(t, bound.Tasks.ResetTasksForRetry([]string{foreign.UUID}), ErrRunLeaseScope)
	for _, state := range []model.RuntimeState{model.RuntimeStateCancelling, model.RuntimeStateCanceled} {
		_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = '1'", state)
		require.NoError(t, err)
		require.ErrorIs(t, bound.Tasks.ResetTasksForRetry([]string{public.UUID}), ErrRuntimeRunCanceled)
		require.ErrorIs(t, tasks.ResetTasksForRetry([]string{public.UUID}), ErrRuntimeRunCanceled)
	}
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = '1'", model.RuntimeStateRunning)
	require.NoError(t, err)
	require.NoError(t, bound.Tasks.ResetTasksForRetry([]string{public.UUID}))
	require.NoError(t, tasks.RunLeaseStore().Release(*token))
	require.ErrorIs(t, bound.Tasks.ResetTasksForRetry([]string{public.UUID}), ErrRunLeaseConflict)
}

func TestRuntimeTerminalTaskCleanupPreservesOutcome(t *testing.T) {
	for _, mode := range []string{"attempt", "pending"} {
		for _, runState := range []model.RuntimeState{
			model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateCancelling,
		} {
			for _, taskState := range []apiv2.PipelineTask_TaskState{
				apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_FAILED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_SKIPPED,
			} {
				t.Run(mode+"/"+string(runState)+"/"+taskState.String(), func(t *testing.T) {
					db, tasks, runtime, task := runtimeFixture(t)
					current := &model.RuntimeTask{TaskUUID: task.UUID, RunUUID: task.RunUUID, State: model.RuntimeTaskWaiting}
					if mode == "attempt" {
						current = runtimeAttemptFixture(task, "attempt")
					}
					require.NoError(t, runtime.Create(current))
					initial := &model.Task{UUID: task.UUID, StatusMetadata: model.JSONData{"diagnostic": "original"}}
					if mode == "attempt" {
						initial = runtimeOutputUpdate(t, task)
					}
					require.NoError(t, runtime.Checkpoint(current, initial))
					_, err := db.Exec("UPDATE tasks SET State = ?, FinishedInSec = 123 WHERE UUID = ?", taskState, task.UUID)
					require.NoError(t, err)
					_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", runState, task.RunUUID)
					require.NoError(t, err)
					before, err := tasks.GetTask(task.UUID)
					require.NoError(t, err)
					update := &model.Task{
						UUID: task.UUID, RunUUID: task.RunUUID, State: model.TaskStatus(taskState), FinishedInSec: 999,
						StatusMetadata: model.JSONData{"diagnostic": "must not overwrite"},
					}
					finalize := func(update *model.Task) error {
						if mode == "attempt" {
							return runtime.Finalize(task.UUID, "attempt", update)
						}
						return runtime.FinalizePending(task.UUID, update, nil)
					}
					conflicting := *update
					conflicting.State = model.TaskStatus(apiv2.PipelineTask_FAILED)
					if conflicting.State == update.State {
						conflicting.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
					}
					require.ErrorIs(t, finalize(&conflicting), ErrRuntimeTaskConflict)
					_, err = runtime.Get(task.UUID)
					require.NoError(t, err, "conflicting outcomes must not delete coordination state")
					require.NoError(t, finalize(update))
					after, err := tasks.GetTask(task.UUID)
					require.NoError(t, err)
					assert.Equal(t, before, after, "terminal cleanup must not rewrite public results, timestamps, or diagnostics")
					_, err = runtime.Get(task.UUID)
					require.ErrorIs(t, err, sql.ErrNoRows)
					require.ErrorIs(t, finalize(update), ErrRuntimeTaskConflict)
				})
			}
		}
	}
}
