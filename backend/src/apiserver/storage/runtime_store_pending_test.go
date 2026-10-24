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
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pendingTaskFixture(name string) *model.Task {
	return &model.Task{
		RunUUID: "1", Namespace: "n1", Name: name, ScopePath: "root." + name,
		Type:  model.TaskType(apiv2.PipelineTask_RUNTIME),
		State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		Pods:  model.JSONSlice{}, TypeAttrs: model.JSONData{}, InputParameters: model.JSONSlice{},
	}
}

func TestRuntimeCreateTaskAtomicRollback(t *testing.T) {
	db, tasks, runtime, _ := runtimeFixture(t)
	_, err := db.Exec(`CREATE TRIGGER reject_runtime_insert BEFORE INSERT ON runtime_tasks
		BEGIN SELECT RAISE(ABORT, 'runtime insert rejected'); END`)
	require.NoError(t, err)
	input := pendingTaskFixture("atomic-create")
	_, err = runtime.CreateTask(input)
	require.ErrorContains(t, err, "runtime insert rejected")
	public, err := tasks.FindTaskByLogicalIdentity(input)
	require.NoError(t, err)
	assert.Nil(t, public, "the public task must roll back when runtime insertion fails")
	_, err = db.Exec("DROP TRIGGER reject_runtime_insert")
	require.NoError(t, err)
	public, err = runtime.CreateTask(input)
	require.NoError(t, err)
	current, err := runtime.Get(public.UUID)
	require.NoError(t, err)
	assert.Equal(t, &model.RuntimeTask{
		TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskWaiting,
	}, current)
	assert.Empty(t, input.UUID, "creation must not mutate the caller's descriptor")
}

func TestRuntimeCreateTaskReplaysPreserveEveryPhase(t *testing.T) {
	for _, phase := range []string{
		model.RuntimeTaskWaiting, model.RuntimeTaskReady, model.RuntimeTaskSubmitting,
		model.RuntimeTaskRunning, model.RuntimeTaskPublishing, model.RuntimeTaskFailed,
		model.RuntimeTaskBackoff, model.RuntimeTaskInfrastructureBackoff, model.RuntimeTaskCanceling,
		model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing, model.RuntimeTaskCancelFinalizing,
	} {
		t.Run(phase, func(t *testing.T) {
			db, _, runtime, _ := runtimeFixture(t)
			input := pendingTaskFixture("replayed")
			public, err := runtime.CreateTask(input)
			require.NoError(t, err)
			current := &model.RuntimeTask{TaskUUID: public.UUID, RunUUID: public.RunUUID, State: phase}
			if phase != model.RuntimeTaskWaiting && phase != model.RuntimeTaskReady && !runtimeCancellationPhase(phase) {
				current = runtimeAttemptFixture(public, "existing-attempt")
				current.State = phase
			}
			require.NoError(t, runtime.Update(current, ""))
			replayed, err := runtime.CreateTask(input)
			require.NoError(t, err)
			assert.Equal(t, public.UUID, replayed.UUID)
			reloaded, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, current, reloaded, "logical replay must not reset attempt or scheduling progress")
			var publicCount, runtimeCount int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks WHERE LogicalKey = ?", public.LogicalKey).Scan(&publicCount))
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM runtime_tasks WHERE TaskUUID = ?", public.UUID).Scan(&runtimeCount))
			assert.Equal(t, 1, publicCount)
			assert.Equal(t, 1, runtimeCount)
			wrongNamespace := *input
			wrongNamespace.Namespace = "other"
			_, err = runtime.CreateTask(&wrongNamespace)
			require.ErrorContains(t, err, "namespace or logical identity")
		})
	}
}

func TestRuntimeCreateTaskValidationAndTerminalReplay(t *testing.T) {
	db, tasks, runtime, _ := runtimeFixture(t)
	_, err := runtime.CreateTask(nil)
	require.Error(t, err)
	_, err = runtime.CreateTask(&model.Task{})
	require.Error(t, err)
	terminal := pendingTaskFixture("terminal")
	terminal.State = model.TaskStatus(apiv2.PipelineTask_CACHED)
	_, err = runtime.CreateTask(terminal)
	require.ErrorContains(t, err, "non-terminal")

	input := pendingTaskFixture("skipped")
	public, err := runtime.CreateTask(input)
	require.NoError(t, err)
	update := &model.Task{UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED), FinishedInSec: 123}
	require.NoError(t, runtime.FinalizePending(public.UUID, update, nil))
	replayed, err := runtime.CreateTask(input)
	require.NoError(t, err)
	assert.Equal(t, public.UUID, replayed.UUID)
	assert.Equal(t, update.State, replayed.State)
	assert.Equal(t, update.FinishedInSec, replayed.FinishedInSec)
	_, err = runtime.Get(public.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	other, err := tasks.CreateTask(&model.Task{RunUUID: "2", Name: "foreign-parent"})
	require.NoError(t, err)
	child := pendingTaskFixture("child")
	child.ParentTaskUUID = &other.UUID
	_, err = runtime.CreateTask(child)
	require.ErrorContains(t, err, "same run")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks WHERE Name = 'child'").Scan(&count))
	assert.Zero(t, count)
}

func cachedOutputFixture(t *testing.T, tasks *TaskStore, source, target *model.Task) (*model.Artifact, *model.ArtifactTask) {
	t.Helper()
	uri := "file:///artifacts/cached-result"
	artifact, sourceLink, err := NewArtifactStore(tasks.db, tasks.time, tasks.uuid, tasks.dbDialect).
		CreateArtifactWithTask(&model.Artifact{
			Namespace: target.Namespace, Name: "cached-result", URI: &uri,
		}, &model.ArtifactTask{
			TaskID: source.UUID, RunUUID: source.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "result",
			Producer: model.JSONData{"taskName": "source", "iteration": "3"},
		})
	require.NoError(t, err)
	copied := *sourceLink
	copied.UUID, copied.TaskID, copied.RunUUID = "", target.UUID, target.RunUUID
	return artifact, &copied
}

func TestRuntimeFinalizePendingCachedOutputs(t *testing.T) {
	db, tasks, runtime, source := runtimeFixture(t)
	public, err := runtime.CreateTask(pendingTaskFixture("cache-hit"))
	require.NoError(t, err)
	artifact, link := cachedOutputFixture(t, tasks, source, public)
	update := runtimeOutputUpdate(t, public)
	update.State, update.FinishedInSec, update.OutputArtifactsHydrated = model.TaskStatus(apiv2.PipelineTask_CACHED), 123, nil
	require.NoError(t, runtime.FinalizePending(public.UUID, update, []*model.ArtifactTask{link, link}))
	stored, err := tasks.GetTask(public.UUID)
	require.NoError(t, err)
	assert.Equal(t, update.State, stored.State)
	assert.Equal(t, update.FinishedInSec, stored.FinishedInSec)
	assert.Equal(t, update.OutputParameters, stored.OutputParameters)
	assert.Equal(t, update.StatusMetadata, stored.StatusMetadata)
	require.Len(t, stored.OutputArtifactsHydrated, 1)
	assert.Equal(t, artifact.UUID, stored.OutputArtifactsHydrated[0].Value.UUID)
	assert.Equal(t, "result", stored.OutputArtifactsHydrated[0].Key)
	assert.EqualValues(t, 3, *stored.OutputArtifactsHydrated[0].Producer.Iteration)
	assert.Empty(t, link.UUID, "deterministic link creation must not mutate the caller's input")
	var artifacts, links int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&artifacts))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&links))
	assert.Equal(t, 1, artifacts, "cache hits must not duplicate artifact metadata")
	assert.Equal(t, 2, links, "the source and destination retain independent output links")
	_, err = runtime.Get(public.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.ErrorIs(t, runtime.FinalizePending(public.UUID, update, []*model.ArtifactTask{link}), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
		UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
	}, nil), ErrRuntimeTaskConflict, "a conflicting replay cannot change terminal state")
}

func TestRuntimeFinalizePendingRollback(t *testing.T) {
	for _, failure := range []string{"link-insert", "task-update", "runtime-delete"} {
		t.Run(failure, func(t *testing.T) {
			db, tasks, runtime, source := runtimeFixture(t)
			public, err := runtime.CreateTask(pendingTaskFixture("rollback-cache"))
			require.NoError(t, err)
			_, link := cachedOutputFixture(t, tasks, source, public)
			links := []*model.ArtifactTask{link}
			switch failure {
			case "link-insert":
				invalid := *link
				invalid.ArtifactID = "missing-artifact"
				links = append(links, &invalid)
			case "task-update":
				_, err = db.Exec(`CREATE TRIGGER reject_pending_update BEFORE UPDATE ON tasks
					BEGIN SELECT RAISE(ABORT, 'pending update rejected'); END`)
				require.NoError(t, err)
			case "runtime-delete":
				_, err = db.Exec(`CREATE TRIGGER reject_runtime_delete BEFORE DELETE ON runtime_tasks
					BEGIN SELECT RAISE(ABORT, 'runtime delete rejected'); END`)
				require.NoError(t, err)
			}
			require.Error(t, runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_CACHED), FinishedInSec: 123,
			}, links))
			stored, err := tasks.GetTask(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, public.State, stored.State)
			assert.Zero(t, stored.FinishedInSec)
			assert.Empty(t, stored.OutputArtifactsHydrated)
			assert.Equal(t, public.StateHistory, stored.StateHistory)
			current, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.RuntimeTaskWaiting, current.State)
			assert.Nil(t, current.AttemptUUID)
			var count int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE TaskID = ?", public.UUID).Scan(&count))
			assert.Zero(t, count, "links inserted before a later failure must roll back")
		})
	}
	t.Run("skip delete failure", func(t *testing.T) {
		db, tasks, runtime, _ := runtimeFixture(t)
		public, err := runtime.CreateTask(pendingTaskFixture("rollback-skip"))
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TRIGGER reject_runtime_delete BEFORE DELETE ON runtime_tasks
			BEGIN SELECT RAISE(ABORT, 'runtime delete rejected'); END`)
		require.NoError(t, err)
		require.ErrorContains(t, runtime.FinalizePending(public.UUID, &model.Task{
			UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}, nil), "runtime delete rejected")
		stored, err := tasks.GetTask(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, public.State, stored.State)
		_, err = runtime.Get(public.UUID)
		require.NoError(t, err)
	})
}

func TestRuntimeFinalizePendingAttemptAndPhaseFences(t *testing.T) {
	for _, phase := range []string{
		model.RuntimeTaskWaiting, model.RuntimeTaskReady, model.RuntimeTaskSubmitting, model.RuntimeTaskRunning,
		model.RuntimeTaskPublishing, model.RuntimeTaskBackoff, model.RuntimeTaskInfrastructureBackoff,
		model.RuntimeTaskCanceling, model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing, model.RuntimeTaskCancelFinalizing,
	} {
		t.Run(phase, func(t *testing.T) {
			_, _, runtime, public := runtimeFixture(t)
			current := &model.RuntimeTask{TaskUUID: public.UUID, RunUUID: public.RunUUID, State: phase}
			require.NoError(t, runtime.Create(current))
			update := &model.Task{UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)}
			if phase == model.RuntimeTaskWaiting || phase == model.RuntimeTaskReady || runtimeCancellationPhase(phase) {
				require.NoError(t, runtime.FinalizePending(public.UUID, update, nil))
			} else {
				require.ErrorIs(t, runtime.FinalizePending(public.UUID, update, nil), ErrRuntimeTaskConflict)
				current.AttemptUUID = new(string)
				*current.AttemptUUID = "attempt"
				require.ErrorContains(t, runtime.Update(current, ""), "first runtime attempt")
			}
		})
		t.Run("active "+phase, func(t *testing.T) {
			_, _, runtime, public := runtimeFixture(t)
			current := runtimeAttemptFixture(public, "active")
			current.State = phase
			require.NoError(t, runtime.Create(current))
			require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
			}, nil), ErrRuntimeTaskConflict)
			require.ErrorIs(t, runtime.Update(current, ""), ErrRuntimeTaskConflict)
			reloaded, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, current, reloaded)
		})
	}
}

func TestRuntimeFinalizePendingValidation(t *testing.T) {
	_, _, runtime, public := runtimeFixture(t)
	pending := &model.RuntimeTask{TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskReady}
	require.NoError(t, runtime.Create(pending))
	for _, update := range []*model.Task{
		nil, {UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)},
		{UUID: "other", State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)},
		{UUID: public.UUID, RunUUID: "2", State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)},
		{UUID: public.UUID, Name: "renamed", State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)},
		{UUID: public.UUID, TypeAttrs: model.JSONData{"iterationIndex": 1}, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)},
	} {
		require.Error(t, runtime.FinalizePending(public.UUID, update, nil))
	}
	require.ErrorContains(t, runtime.Finalize(public.UUID, "", &model.Task{
		UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	}), "FinalizePending", "attempt finalization must not bypass NULL-attempt fences")
	pending.ExecutionReference = model.JSONData{"workload": "unexpected"}
	require.NoError(t, runtime.Update(pending, ""))
	require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
		UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
	}, nil), ErrRuntimeTaskConflict)
	pending.AttemptUUID = new(string)
	*pending.AttemptUUID = "unsafe-attempt"
	require.ErrorContains(t, runtime.Update(pending, ""), "first runtime attempt")
}

func TestRuntimeFinalizePendingCancellationFences(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStateCancelling, model.RuntimeStateCanceled} {
		t.Run(string(state), func(t *testing.T) {
			db, _, runtime, _ := runtimeFixture(t)
			input := pendingTaskFixture("cancel-pending")
			public, err := runtime.CreateTask(input)
			require.NoError(t, err)
			prepared, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			prepared.ExecutionRequest = model.JSONData{"version": float64(1), "prepared": true}
			require.NoError(t, runtime.Update(prepared, ""))
			_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", state, public.RunUUID)
			require.NoError(t, err)
			_, err = runtime.CreateTask(input)
			require.ErrorIs(t, err, ErrRuntimeRunCanceled)
			for _, terminal := range []apiv2.PipelineTask_TaskState{
				apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_FAILED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_SKIPPED,
			} {
				require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
					UUID: public.UUID, State: model.TaskStatus(terminal),
				}, nil), ErrRuntimeRunCanceled)
			}
			current, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			current.State = model.RuntimeTaskCanceling
			active := *current
			id := "must-not-start"
			active.AttemptUUID = &id
			require.ErrorIs(t, runtime.Update(&active, ""), ErrRuntimeRunCanceled,
				"entering cancellation must not assign a new execution attempt")
			require.NoError(t, runtime.Update(current, ""))
			for _, terminal := range []apiv2.PipelineTask_TaskState{apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_FAILED} {
				require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
					UUID: public.UUID, State: model.TaskStatus(terminal),
				}, nil), ErrRuntimeRunCanceled)
			}
			active = *current
			active.AttemptUUID = &id
			require.ErrorIs(t, runtime.Update(&active, ""), ErrRuntimeRunCanceled)
			require.NoError(t, runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
			}, nil))
		})
	}
}

func TestRuntimePendingCachedLinkValidation(t *testing.T) {
	for _, invalid := range []string{
		"nil", "task", "run", "type", "key", "artifact", "namespace", "producer", "source-link-id", "changed-producer",
	} {
		t.Run(invalid, func(t *testing.T) {
			db, tasks, runtime, source := runtimeFixture(t)
			public, err := runtime.CreateTask(pendingTaskFixture("invalid-cache"))
			require.NoError(t, err)
			artifact, link := cachedOutputFixture(t, tasks, source, public)
			switch invalid {
			case "nil":
				link = nil
			case "task":
				link.TaskID = source.UUID
			case "run":
				link.RunUUID = "2"
			case "type":
				link.Type = model.IOType(apiv2.IOType_COMPONENT_INPUT)
			case "key":
				link.ArtifactKey = ""
			case "artifact":
				link.ArtifactID = ""
			case "namespace":
				_, err = db.Exec("UPDATE artifacts SET Namespace = 'other' WHERE UUID = ?", artifact.UUID)
				require.NoError(t, err)
			case "producer":
				link.Producer = model.JSONData{"iteration": "not-a-number"}
			case "source-link-id":
				require.NoError(t, db.QueryRow("SELECT UUID FROM artifact_tasks WHERE TaskID = ?", source.UUID).Scan(&link.UUID))
			case "changed-producer":
				existing, err := NewArtifactTaskStore(db, tasks.uuid, tasks.dbDialect).CreateArtifactTask(link)
				require.NoError(t, err)
				link.UUID = existing.UUID
				link.Producer = model.JSONData{"taskName": "different", "iteration": "3"}
			}
			require.Error(t, runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_CACHED),
			}, []*model.ArtifactTask{link}))
			stored, err := tasks.GetTask(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, public.State, stored.State)
			_, err = runtime.Get(public.UUID)
			require.NoError(t, err)
		})
	}
	t.Run("non-cache output attachments", func(t *testing.T) {
		_, tasks, runtime, source := runtimeFixture(t)
		public, err := runtime.CreateTask(pendingTaskFixture("no-cache-links"))
		require.NoError(t, err)
		_, link := cachedOutputFixture(t, tasks, source, public)
		require.ErrorContains(t, runtime.FinalizePending(public.UUID, &model.Task{
			UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}, []*model.ArtifactTask{link}), "only cached")
		require.ErrorContains(t, runtime.FinalizePending(public.UUID, runtimeOutputUpdate(t, public), nil), "terminal")
		update := runtimeOutputUpdate(t, public)
		update.State = model.TaskStatus(apiv2.PipelineTask_CACHED)
		require.ErrorContains(t, runtime.FinalizePending(public.UUID, update, nil), "not new attempt artifacts")
	})
	t.Run("empty string is not a NULL attempt", func(t *testing.T) {
		db, _, runtime, public := runtimeFixture(t)
		require.NoError(t, runtime.Create(&model.RuntimeTask{
			TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskReady,
		}))
		_, err := db.Exec("UPDATE runtime_tasks SET AttemptUUID = '' WHERE TaskUUID = ?", public.UUID)
		require.NoError(t, err)
		require.ErrorIs(t, runtime.FinalizePending(public.UUID, &model.Task{
			UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}, nil), ErrRuntimeTaskConflict)
		next := runtimeAttemptFixture(public, "attempt")
		require.ErrorIs(t, runtime.Update(next, ""), ErrRuntimeTaskConflict)
	})
}

func TestRuntimePendingLeaseScopeAndCancellation(t *testing.T) {
	db, tasks, _, _ := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = '1'")
	require.NoError(t, err)
	token, err := tasks.RunLeaseStore().Acquire("1", "worker", time.Minute, false)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	input := pendingTaskFixture("leased-create")
	public, err := bound.Runtime.CreateTask(input)
	require.NoError(t, err)
	foreign := pendingTaskFixture("foreign")
	foreign.RunUUID = "2"
	_, err = bound.Runtime.CreateTask(foreign)
	require.ErrorIs(t, err, ErrRunLeaseScope)
	foreignTask, err := tasks.CreateTask(foreign)
	require.NoError(t, err)
	require.ErrorIs(t, bound.Runtime.FinalizePending(foreignTask.UUID, &model.Task{
		UUID: foreignTask.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
	}, nil), ErrRunLeaseScope)
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = '1'", model.RuntimeStateCancelling)
	require.NoError(t, err)
	update := &model.Task{UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)}
	require.ErrorIs(t, bound.Runtime.FinalizePending(public.UUID, update, nil), ErrRuntimeRunCanceled)
	current, err := bound.Runtime.Get(public.UUID)
	require.NoError(t, err)
	current.State = model.RuntimeTaskCanceling
	require.NoError(t, bound.Runtime.Update(current, ""))
	require.NoError(t, bound.Runtime.FinalizePending(public.UUID, update, nil))
	require.NoError(t, tasks.RunLeaseStore().Release(*token))
	require.ErrorIs(t, bound.Runtime.FinalizePending(public.UUID, update, nil), ErrRunLeaseConflict,
		"even a finalized pending task must check the old worker's lease first")
}

func TestRuntimePendingStaleLeaseFences(t *testing.T) {
	for _, invalidation := range []string{"takeover", "expiry"} {
		t.Run(invalidation, func(t *testing.T) {
			db, tasks, runtime, _ := runtimeFixture(t)
			input := pendingTaskFixture("leased")
			public, err := runtime.CreateTask(input)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = ?", public.RunUUID)
			require.NoError(t, err)
			leases := tasks.RunLeaseStore()
			token, err := leases.Acquire(public.RunUUID, "old", time.Minute, false)
			require.NoError(t, err)
			old, err := tasks.LeaseScopedStores(*token)
			require.NoError(t, err)
			if invalidation == "takeover" {
				require.NoError(t, leases.Release(*token))
				_, err = leases.Acquire(public.RunUUID, "new", time.Minute, false)
				require.NoError(t, err)
			} else {
				_, err = db.Exec("UPDATE run_leases SET ExpiresAt = 0 WHERE RunUUID = ?", public.RunUUID)
				require.NoError(t, err)
			}
			_, err = old.Runtime.CreateTask(pendingTaskFixture("stale-new"))
			require.ErrorIs(t, err, ErrRunLeaseConflict)
			require.ErrorIs(t, old.Runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
			}, nil), ErrRunLeaseConflict)
			_, err = runtime.Get(public.UUID)
			require.NoError(t, err)
			stored, err := tasks.FindTaskByLogicalIdentity(pendingTaskFixture("stale-new"))
			require.NoError(t, err)
			assert.Nil(t, stored)
		})
	}
}
