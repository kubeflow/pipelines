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
	"os"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func runtimeFixture(t *testing.T) (*sql.DB, *TaskStore, RuntimeStoreInterface, *model.Task) {
	t.Helper()
	db, d, _ := initializeRunStore()
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	tasks := NewTaskStore(db, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), d)
	task, err := tasks.CreateTask(&model.Task{
		RunUUID: "1", Namespace: "n1", Name: "runtime-node", Fingerprint: "cache-fingerprint",
		State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{},
		TypeAttrs: model.JSONData{}, InputParameters: model.JSONSlice{},
	})
	require.NoError(t, err)
	return db, tasks, tasks.RuntimeStore(), task
}

func runtimeAttemptFixture(task *model.Task, attempt string) *model.RuntimeTask {
	return &model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: task.RunUUID, AttemptUUID: &attempt,
		State: model.RuntimeTaskRunning, ExecutionRequest: model.JSONData{"version": float64(1), "task": task.UUID},
		ExecutionReference: model.JSONData{"version": float64(1), "name": attempt},
	}
}

func runtimeOutputUpdate(t *testing.T, task *model.Task) *model.Task {
	t.Helper()
	parameters, err := model.ProtoSliceToJSONSlice([]*apiv2.PipelineTask_InputOutputs_IOParameter{{
		ParameterKey: "answer", Value: structpb.NewNumberValue(42), Type: apiv2.IOType_OUTPUT,
	}})
	require.NoError(t, err)
	uri := "file:///artifacts/result"
	return &model.Task{
		UUID: task.UUID, RunUUID: task.RunUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		OutputParameters: parameters, StatusMetadata: model.JSONData{"diagnostic": "worker result"},
		Pods: model.JSONSlice{map[string]any{"name": "attempt-log"}},
		OutputArtifactsHydrated: []model.TaskArtifactHydrated{{
			Key: "result", Type: apiv2.IOType_OUTPUT,
			Value: &model.Artifact{Name: "result", URI: &uri, Metadata: model.JSONData{"version": float64(1)}},
		}},
	}
}

func TestRuntimeStoreCRUDAndAttemptFences(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	waiting := &model.RuntimeTask{TaskUUID: task.UUID, RunUUID: "1", State: model.RuntimeTaskWaiting}
	require.NoError(t, runtime.Create(waiting))
	require.Error(t, runtime.Create(waiting), "duplicate create must not overwrite current state")
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Nil(t, current.AttemptUUID)
	assert.Nil(t, current.ExecutionRequest)
	assert.Nil(t, current.ExecutionReference)

	second, err := tasks.CreateTask(&model.Task{
		RunUUID: "1", Namespace: "n1", Name: "second", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
	})
	require.NoError(t, err)
	secondRuntime := &model.RuntimeTask{TaskUUID: second.UUID, RunUUID: "1", State: model.RuntimeTaskReady}
	require.NoError(t, runtime.Create(secondRuntime), "unique nullable attempts allow multiple waiting tasks")
	entries, err := runtime.ListByRun("1")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	entries, err = runtime.ListByRun("2")
	require.NoError(t, err)
	assert.Empty(t, entries)

	attempt := runtimeAttemptFixture(task, "attempt-1")
	require.NoError(t, runtime.Update(attempt, ""))
	require.ErrorIs(t, runtime.Update(attempt, ""), ErrRuntimeTaskConflict)
	require.NoError(t, runtime.Update(attempt, "attempt-1"), "same-attempt recovery is idempotent")
	next := *attempt
	id := "attempt-2"
	next.AttemptUUID = &id
	require.ErrorContains(t, runtime.Update(&next, "attempt-1"), "only BeginRetry")
	changedRequest := *attempt
	changedRequest.ExecutionRequest = model.JSONData{"changed": true}
	require.ErrorContains(t, runtime.Update(&changedRequest, "attempt-1"), "immutable")
	secondRuntime.AttemptUUID = attempt.AttemptUUID
	require.Error(t, runtime.Update(secondRuntime, ""), "attempt UUID must be globally unique")
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM runtime_tasks WHERE AttemptUUID IS NULL").Scan(new(int)))

	invalid := *attempt
	invalid.State = "UNKNOWN"
	require.ErrorContains(t, runtime.Update(&invalid, "attempt-1"), "phase")
	invalid = *attempt
	invalid.ExecutionReference = model.JSONData{"bad": make(chan bool)}
	require.ErrorContains(t, runtime.Update(&invalid, "attempt-1"), "encode runtime execution reference")
	require.ErrorIs(t, func() error { _, err := runtime.Get("missing"); return err }(), sql.ErrNoRows)
}

func TestRuntimeRequestFirstAssignmentAndImmutability(t *testing.T) {
	_, _, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	attempt.ExecutionRequest = nil
	attempt.State = model.RuntimeTaskReady
	require.NoError(t, runtime.Create(attempt))
	attempt.ExecutionRequest = model.JSONData{"version": json.Number("1.0"), "task": task.UUID}
	require.NoError(t, runtime.Update(attempt, "attempt"))
	require.NoError(t, runtime.Update(attempt, "attempt"), "equivalent numeric JSON encodings must have the same immutable request identity")
	reloaded, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	require.NoError(t, runtime.Update(reloaded, "attempt"), "JSON numeric normalization must not change request identity")
	reloaded.ExecutionRequest = nil
	require.ErrorContains(t, runtime.Update(reloaded, "attempt"), "immutable")
	reloaded.ExecutionRequest = model.JSONData{"version": 2}
	require.ErrorContains(t, runtime.Update(reloaded, "attempt"), "immutable")
}

func TestRuntimeCleanupPhasesPersistAcrossRecovery(t *testing.T) {
	_, _, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	for _, state := range []string{model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing} {
		attempt.State = state
		require.NoError(t, runtime.Update(attempt, "attempt"))
		recovered, err := runtime.Get(task.UUID)
		require.NoError(t, err)
		assert.Equal(t, state, recovered.State)
		assert.Zero(t, recovered.RetryCount)
		assert.Equal(t, attempt.ExecutionReference, recovered.ExecutionReference)
	}
}

func TestRuntimeInfrastructureBackoffPersists(t *testing.T) {
	_, _, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	attempt.State = model.RuntimeTaskInfrastructureBackoff
	eligible := int64(123)
	attempt.NextEligibleAt = &eligible
	require.NoError(t, runtime.Update(attempt, "attempt"))
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, attempt, current)
}

func TestRuntimeCreateDoesNotResurrectTerminalTask(t *testing.T) {
	_, tasks, runtime, task := runtimeFixture(t)
	_, err := tasks.UpdateTask(&model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)})
	require.NoError(t, err)
	require.ErrorIs(t, runtime.Create(&model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: task.RunUUID, State: model.RuntimeTaskWaiting,
	}), ErrRuntimeTaskConflict)
	_, err = runtime.Get(task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRuntimeFinalizeRunPreservesAcceptedCancellation(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStateCancelling, model.RuntimeStateCanceled} {
		t.Run(string(state), func(t *testing.T) {
			db, _, runtime, _ := runtimeFixture(t)
			_, err := db.Exec("UPDATE run_details SET State = ? WHERE UUID = '1'", state)
			require.NoError(t, err)
			require.ErrorIs(t, runtime.FinalizeRun("1", model.RuntimeStateSucceeded, 123), ErrRuntimeRunCanceled)
			require.ErrorIs(t, runtime.FinalizeRun("1", model.RuntimeStateFailed, 123), ErrRuntimeRunCanceled)
			require.NoError(t, runtime.FinalizeRun("1", model.RuntimeStateCanceled, 123))
			var stored string
			require.NoError(t, db.QueryRow("SELECT State FROM run_details WHERE UUID = '1'").Scan(&stored))
			assert.Equal(t, string(model.RuntimeStateCanceled), stored)
		})
	}
	t.Run("successful run", func(t *testing.T) {
		db, _, runtime, _ := runtimeFixture(t)
		require.NoError(t, runtime.FinalizeRun("1", model.RuntimeStateSucceeded, 123))
		var state string
		var finished int64
		require.NoError(t, db.QueryRow("SELECT State, FinishedAtInSec FROM run_details WHERE UUID = '1'").Scan(&state, &finished))
		assert.Equal(t, string(model.RuntimeStateSucceeded), state)
		assert.EqualValues(t, 123, finished)
	})
}

func TestRuntimeCheckpointDurablePublicationAndFinalize(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "publication-attempt")
	require.NoError(t, runtime.Create(attempt))
	attempt.State = model.RuntimeTaskPublishing
	attempt.ExecutionReference["completion"] = map[string]any{"count": float64(3)}
	update := runtimeOutputUpdate(t, task)
	require.NoError(t, runtime.Checkpoint(attempt, update))
	require.NoError(t, runtime.Checkpoint(attempt, update), "replayed delivery must not create another output artifact")

	// A new store reconstructs publication using only durable rows.
	recovered := NewTaskStore(db, tasks.time, tasks.uuid, tasks.dbDialect).RuntimeStore()
	current, err := recovered.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskPublishing, current.State)
	assert.Equal(t, attempt.ExecutionRequest, current.ExecutionRequest)
	assert.Equal(t, attempt.ExecutionReference, current.ExecutionReference)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
	assert.Equal(t, update.OutputParameters, public.OutputParameters)
	assert.Equal(t, update.StatusMetadata, public.StatusMetadata)
	for _, table := range []string{"artifacts", "artifact_tasks"} {
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
		assert.Equal(t, 1, count, table)
	}
	cached, err := tasks.FindLatestCachedTask(task.Namespace, task.Fingerprint)
	require.NoError(t, err)
	assert.Nil(t, cached)

	bad := *update
	bad.OutputArtifactsHydrated = append(append([]model.TaskArtifactHydrated{}, update.OutputArtifactsHydrated...), model.TaskArtifactHydrated{Key: "invalid"})
	attempt.State = model.RuntimeTaskFailed
	require.Error(t, runtime.Checkpoint(attempt, &bad))
	current, err = runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskPublishing, current.State, "failed checkpoint must roll back runtime phase")

	final := &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), FinishedInSec: 123}
	require.ErrorIs(t, runtime.Finalize(task.UUID, "stale-attempt", final), ErrRuntimeTaskConflict)
	require.NoError(t, recovered.Finalize(task.UUID, "publication-attempt", final))
	require.ErrorIs(t, runtime.Finalize(task.UUID, "publication-attempt", final), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.Checkpoint(attempt, update), ErrRuntimeTaskConflict)
	_, err = runtime.Get(task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	public, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), public.State)
	assert.EqualValues(t, 123, public.FinishedInSec)
	cached, err = tasks.FindLatestCachedTask(task.Namespace, task.Fingerprint)
	require.NoError(t, err)
	require.NotNil(t, cached)
}

func TestRuntimeBeginRetryReplacesIdentityAndPreservesDiagnostics(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	old := runtimeAttemptFixture(task, "old-attempt")
	require.NoError(t, runtime.Create(old))
	old.State = model.RuntimeTaskFailed
	old.ExecutionReference["completion"] = map[string]any{"count": float64(4)}
	update := runtimeOutputUpdate(t, task)
	require.NoError(t, runtime.Checkpoint(old, update))
	_, err := db.Exec(`INSERT INTO artifact_tasks (UUID, ArtifactID, TaskID, Type, Iteration, RunUUID, ArtifactKey)
		SELECT 'input-link', ArtifactID, TaskID, ?, Iteration, RunUUID, 'input' FROM artifact_tasks`, apiv2.IOType_COMPONENT_INPUT)
	require.NoError(t, err)
	publicBefore, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)

	next := runtimeAttemptFixture(task, "next-attempt")
	next.State = model.RuntimeTaskBackoff
	eligible := int64(456)
	next.NextEligibleAt = &eligible
	require.Error(t, runtime.BeginRetry(task.UUID, "old-attempt", old), "retry identity cannot be reused")
	require.NoError(t, runtime.BeginRetry(task.UUID, "old-attempt", next))
	assert.Equal(t, 1, next.RetryCount)
	require.ErrorIs(t, runtime.BeginRetry(task.UUID, "old-attempt", next), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.Finalize(task.UUID, "old-attempt", &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	}), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.Checkpoint(old, update), ErrRuntimeTaskConflict)
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, "next-attempt", *current.AttemptUUID)
	assert.Equal(t, 1, current.RetryCount)
	assert.NotContains(t, current.ExecutionReference, "completion", "new attempts must not inherit executor checkpoints")
	assert.Equal(t, &eligible, current.NextEligibleAt)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, task.UUID, public.UUID)
	assert.Equal(t, task.Fingerprint, public.Fingerprint)
	assert.Empty(t, public.OutputParameters)
	assert.Empty(t, public.StatusMetadata)
	assert.Empty(t, public.Pods)
	assert.EqualValues(t, 0, public.FinishedInSec)
	assert.Greater(t, len(public.StateHistory), len(publicBefore.StateHistory))
	require.Equal(t, publicBefore.StateHistory, public.StateHistory[:len(publicBefore.StateHistory)])
	_, err = model.JSONSliceToProtoSlice(public.StateHistory, func() *apiv2.PipelineTask_TaskStatus { return &apiv2.PipelineTask_TaskStatus{} })
	require.NoError(t, err, "retained diagnostics must remain API-readable")
	history, err := json.Marshal(public.StateHistory)
	require.NoError(t, err)
	assert.Contains(t, string(history), "worker result")
	assert.Contains(t, string(history), "attempt-log")
	assert.Contains(t, string(history), "old-attempt")
	var outputs, inputs, artifacts int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE Type = ?", apiv2.IOType_OUTPUT).Scan(&outputs))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE Type = ?", apiv2.IOType_COMPONENT_INPUT).Scan(&inputs))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&artifacts))
	assert.Equal(t, 0, outputs)
	assert.Equal(t, 1, inputs)
	assert.Equal(t, 1, artifacts)
}

func TestRuntimeForeignKeysAndCascade(t *testing.T) {
	for _, deletion := range []string{"task", "run"} {
		t.Run(deletion, func(t *testing.T) {
			db, tasks, runtime, task := runtimeFixture(t)
			var ddl string
			require.NoError(t, db.QueryRow("SELECT sql FROM sqlite_master WHERE name = 'runtime_tasks'").Scan(&ddl))
			assert.Contains(t, ddl, "FOREIGN KEY (`RunUUID`) REFERENCES `run_details`(`UUID`) ON DELETE CASCADE")
			assert.Contains(t, ddl, "FOREIGN KEY (`TaskUUID`,`RunUUID`) REFERENCES `tasks`(`UUID`,`RunUUID`) ON DELETE CASCADE")
			attempt := runtimeAttemptFixture(task, "attempt")
			mismatch := *attempt
			mismatch.RunUUID = "2"
			require.ErrorContains(t, runtime.Create(&mismatch), "owning public task")
			_, err := db.Exec("INSERT INTO runtime_tasks (TaskUUID, RunUUID, State, RetryCount) VALUES (?, '2', 'WAITING', 0)", task.UUID)
			require.ErrorContains(t, err, "FOREIGN KEY", "composite FK must independently reject a mismatched existing run")
			missing := *attempt
			missing.TaskUUID = "missing"
			require.ErrorIs(t, runtime.Create(&missing), ErrRuntimeTaskConflict)
			require.NoError(t, runtime.Create(attempt))
			if deletion == "task" {
				_, err := db.Exec("DELETE FROM tasks WHERE UUID = ?", task.UUID)
				require.NoError(t, err)
			} else {
				require.NoError(t, NewRunStore(db, util.NewFakeTimeForEpoch(), tasks.dbDialect).DeleteRun("1"))
			}
			_, err = runtime.Get(task.UUID)
			require.ErrorIs(t, err, sql.ErrNoRows)
			require.ErrorIs(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
				UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
			}), ErrRuntimeTaskConflict)
		})
	}
}

func TestRuntimeCancellationFences(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStateCancelling, model.RuntimeStateCanceled} {
		t.Run(string(state), func(t *testing.T) {
			db, _, runtime, task := runtimeFixture(t)
			attempt := runtimeAttemptFixture(task, "attempt")
			require.NoError(t, runtime.Create(attempt))
			_, err := db.Exec("UPDATE run_details SET State = ? WHERE UUID = '1'", string(state))
			require.NoError(t, err)
			require.ErrorIs(t, runtime.Update(attempt, "attempt"), ErrRuntimeRunCanceled)
			require.ErrorIs(t, runtime.Checkpoint(attempt, runtimeOutputUpdate(t, task)), ErrRuntimeRunCanceled)
			require.ErrorIs(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
				UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
			}), ErrRuntimeRunCanceled)
			require.ErrorIs(t, runtime.BeginRetry(task.UUID, "attempt", runtimeAttemptFixture(task, "new")), ErrRuntimeRunCanceled)
			attempt.State = model.RuntimeTaskCanceling
			require.NoError(t, runtime.Update(attempt, "attempt"))
			require.NoError(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
				UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
			}))
		})
	}
	t.Run("task cancellation remains authoritative before run state changes", func(t *testing.T) {
		_, _, runtime, task := runtimeFixture(t)
		attempt := runtimeAttemptFixture(task, "attempt")
		attempt.State = model.RuntimeTaskCanceling
		require.NoError(t, runtime.Create(attempt))
		attempt.State = model.RuntimeTaskRunning
		require.ErrorIs(t, runtime.Update(attempt, "attempt"), ErrRuntimeRunCanceled)
		require.ErrorIs(t, runtime.Checkpoint(attempt, runtimeOutputUpdate(t, task)), ErrRuntimeRunCanceled)
		require.ErrorIs(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
			UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
		}), ErrRuntimeRunCanceled)
		require.NoError(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
			UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}))
	})
}

func TestRuntimeLauncherReportsRemainProvisional(t *testing.T) {
	_, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	launcher := runtimeOutputUpdate(t, task)
	launcher.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
	launcher.FinishedInSec = 123
	public, err := tasks.UpdateTask(launcher)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
	assert.EqualValues(t, 0, public.FinishedInSec)
	assert.Equal(t, launcher.OutputParameters, public.OutputParameters)
	cached, err := tasks.FindLatestCachedTask(task.Namespace, task.Fingerprint)
	require.NoError(t, err)
	assert.Nil(t, cached, "launcher outputs must not become cacheable before publication")
	require.NoError(t, runtime.Finalize(task.UUID, "attempt", launcher))
	public, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), public.State)

	// Unmanaged Argo tasks retain the original UpdateTask behavior.
	unmanaged, err := tasks.CreateTask(&model.Task{RunUUID: "1", Namespace: "n1", Name: "argo", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
	require.NoError(t, err)
	public, err = tasks.UpdateTask(&model.Task{UUID: unmanaged.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)})
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), public.State)
}

func TestRuntimeCheckpointArtifactsAtomicAndIdempotent(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	attempt.State = model.RuntimeTaskPublishing
	update := runtimeOutputUpdate(t, task)
	update.OutputArtifactsHydrated = nil
	artifact := &model.Artifact{
		UUID: util.NewDeterministicUUID("artifact:" + task.UUID + ":attempt"), Name: "output",
		Metadata: model.JSONData{"version": float64(1)},
	}
	link := &model.ArtifactTask{
		UUID: util.NewDeterministicUUID("link:" + artifact.UUID), ArtifactID: artifact.UUID,
		TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT),
		Iteration: model.ArtifactTaskNoIteration, ArtifactKey: "output",
	}
	checkpoint := func() error {
		return runtime.CheckpointArtifacts(attempt, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link})
	}
	// Force a late relationship failure after artifact and task writes.
	_, err := db.Exec(`CREATE TRIGGER reject_runtime_link BEFORE INSERT ON artifact_tasks
		BEGIN SELECT RAISE(FAIL, 'relationship unavailable'); END`)
	require.NoError(t, err)
	require.ErrorContains(t, checkpoint(), "relationship unavailable")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Zero(t, count)
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskRunning, current.State)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Empty(t, public.OutputParameters)
	_, err = db.Exec("DROP TRIGGER reject_runtime_link")
	require.NoError(t, err)
	require.NoError(t, checkpoint())
	require.NoError(t, checkpoint())
	attempt.State = model.RuntimeTaskFailed
	require.NoError(t, checkpoint(), "failed workload logs must checkpoint before cleanup or retry")
	current, err = runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskFailed, current.State)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts WHERE UUID = ?", artifact.UUID).Scan(&count))
	assert.Equal(t, 1, count)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE UUID = ?", link.UUID).Scan(&count))
	assert.Equal(t, 1, count)
	artifact.Name = "different-output"
	require.ErrorContains(t, checkpoint(), "different metadata")
	artifact.Name = "output"
	link.ArtifactKey = "different-key"
	require.ErrorContains(t, checkpoint(), "different output")
	link.ArtifactKey = "output"
	stale := "stale"
	attempt.AttemptUUID = &stale
	require.ErrorIs(t, checkpoint(), ErrRuntimeTaskConflict)
	id := "attempt"
	attempt.AttemptUUID = &id
	_, err = db.Exec("UPDATE run_details SET State = 'CANCELING' WHERE UUID = '1'")
	require.NoError(t, err)
	require.ErrorIs(t, checkpoint(), ErrRuntimeRunCanceled)
}

func TestRuntimeExecutorArtifactCheckpointsRemainProvisional(t *testing.T) {
	for _, phase := range []string{
		model.RuntimeTaskRunning, model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing,
	} {
		t.Run(phase, func(t *testing.T) {
			testRuntimeArtifactCheckpointRemainsProvisional(t, phase)
		})
	}
}

func testRuntimeArtifactCheckpointRemainsProvisional(t *testing.T, phase string) {
	t.Helper()
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	attempt.ExecutionReference["completion"] = map[string]any{"count": float64(1), "terminal": false}
	require.NoError(t, runtime.Create(attempt))
	attempt.State = phase
	update := runtimeOutputUpdate(t, task)
	checkpoint := func() error { return runtime.CheckpointArtifacts(attempt, update, nil, nil) }
	require.NoError(t, checkpoint())
	require.NoError(t, checkpoint(), "executor-owned publication may replay before cleanup and public finalization")
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, attempt, current)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
	assert.Equal(t, update.OutputParameters, public.OutputParameters)
	require.Len(t, public.OutputArtifactsHydrated, 1)
	cached, err := tasks.FindLatestSucceededTaskByFingerprint(task.Namespace, task.Fingerprint)
	require.NoError(t, err)
	assert.Nil(t, cached, "provisional outputs must not become cache eligible")
	update.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
	require.ErrorContains(t, checkpoint(), "provisional")
	update.State = model.TaskStatus(apiv2.PipelineTask_RUNNING)
	stale := "old-attempt"
	attempt.AttemptUUID = &stale
	require.ErrorIs(t, checkpoint(), ErrRuntimeTaskConflict)
	active := "attempt"
	attempt.AttemptUUID = &active
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", model.RuntimeStateCancelling, task.RunUUID)
	require.NoError(t, err)
	require.ErrorIs(t, checkpoint(), ErrRuntimeRunCanceled)
}

func TestRuntimeBeginRetryRollsBackOnAttemptCollision(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	old := runtimeAttemptFixture(task, "old")
	require.NoError(t, runtime.Create(old))
	old.State = model.RuntimeTaskFailed
	require.NoError(t, runtime.Checkpoint(old, runtimeOutputUpdate(t, task)))
	second, err := tasks.CreateTask(&model.Task{
		RunUUID: "1", Namespace: "n1", Name: "other", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
	})
	require.NoError(t, err)
	require.NoError(t, runtime.Create(runtimeAttemptFixture(second, "claimed")))
	next := runtimeAttemptFixture(task, "claimed")
	require.Error(t, runtime.BeginRetry(task.UUID, "old", next))
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, "old", *current.AttemptUUID)
	assert.Zero(t, current.RetryCount)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.NotEmpty(t, public.OutputParameters)
	assert.NotEmpty(t, public.StatusMetadata)
	var outputs int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE TaskID = ?", task.UUID).Scan(&outputs))
	assert.Equal(t, 1, outputs)
}

func TestRuntimeManualRetryInvalidatesDeletedAttempt(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "old")
	require.NoError(t, runtime.Create(attempt))
	attempt.State = model.RuntimeTaskPublishing
	update := runtimeOutputUpdate(t, task)
	require.NoError(t, runtime.Checkpoint(attempt, update))
	require.NoError(t, tasks.ResetTasksForRetry([]string{task.UUID}))
	waiting, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, &model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: task.RunUUID, State: model.RuntimeTaskWaiting,
	}, waiting)
	require.ErrorIs(t, runtime.Update(attempt, "old"), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.Checkpoint(attempt, update), ErrRuntimeTaskConflict)
	require.ErrorIs(t, runtime.Finalize(task.UUID, "old", &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	}), ErrRuntimeTaskConflict)
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
	assert.Empty(t, public.OutputParameters)
	assert.Equal(t, task.Fingerprint, public.Fingerprint)
	history, err := json.Marshal(public.StateHistory)
	require.NoError(t, err)
	assert.Contains(t, string(history), "worker result")
	assert.Contains(t, string(history), "attempt-log")
	var outputs int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE TaskID = ?", task.UUID).Scan(&outputs))
	assert.Zero(t, outputs)
}

func TestRuntimePublicationSurvivesDatabaseReopen(t *testing.T) {
	id, err := util.NewUUIDGenerator().NewRandom()
	require.NoError(t, err)
	path := ".runtime-checkpoint-" + id.String() + ".sqlite"
	t.Cleanup(func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			err := os.Remove(path + suffix)
			if !os.IsNotExist(err) {
				require.NoError(t, err)
			}
		}
	})
	orm, err := gorm.Open(sqlite.Open(path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))
	require.NoError(t, orm.Create(&model.Experiment{UUID: "experiment", Name: "experiment"}).Error)
	db, err := orm.DB()
	require.NoError(t, err)
	runs := NewRunStore(db, util.NewFakeTimeForEpoch(), dialect.NewDBDialect("sqlite"))
	_, err = runs.CreateRun(&model.Run{UUID: "run", ExperimentId: "experiment", RunDetails: model.RunDetails{State: model.RuntimeStateRunning}})
	require.NoError(t, err)
	tasks := NewTaskStore(db, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), dialect.NewDBDialect("sqlite"))
	task, err := tasks.CreateTask(&model.Task{RunUUID: "run", Name: "node", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
	require.NoError(t, err)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, tasks.RuntimeStore().Create(attempt))
	attempt.State = model.RuntimeTaskPublishing
	attempt.ExecutionReference["completion"] = map[string]any{"count": float64(2)}
	require.NoError(t, tasks.RuntimeStore().Checkpoint(attempt, runtimeOutputUpdate(t, task)))
	require.NoError(t, db.Close())

	reopened, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_busy_timeout=5000")
	require.NoError(t, err)
	defer reopened.Close()
	recoveredTasks := NewTaskStore(reopened, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), dialect.NewDBDialect("sqlite"))
	current, err := recoveredTasks.RuntimeStore().Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskPublishing, current.State)
	assert.Equal(t, attempt.ExecutionReference, current.ExecutionReference)
	assert.Equal(t, "attempt", *current.AttemptUUID)
	assert.Equal(t, attempt.ExecutionRequest, current.ExecutionRequest)
	public, err := recoveredTasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.NotEmpty(t, public.OutputParameters)
	var count int
	require.NoError(t, reopened.QueryRow("SELECT COUNT(*) FROM artifact_tasks WHERE TaskID = ?", task.UUID).Scan(&count))
	assert.Equal(t, 1, count)
	require.NoError(t, recoveredTasks.RuntimeStore().Finalize(task.UUID, "attempt", &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	}))
}

func TestRunStoreRuntimeMetadataRoundTrip(t *testing.T) {
	db, d, runs := initializeRunStore()
	defer db.Close()
	executor, provisioner := "subprocess", "directory"
	reference := model.JSONData{"version": float64(1), "path": "/workspace/run"}
	created, err := runs.CreateRun(&model.Run{
		UUID: "runtime-run", ExperimentId: defaultFakeExpId, DisplayName: "runtime",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateRunning, ExecutorType: &executor,
			WorkspaceProvisioner: &provisioner, WorkspaceReference: &reference,
		},
	})
	require.NoError(t, err)
	loaded, err := runs.GetRun(created.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, &executor, loaded.ExecutorType)
	assert.Equal(t, &provisioner, loaded.WorkspaceProvisioner)
	assert.Equal(t, &reference, loaded.WorkspaceReference)
	updatedReference := model.JSONData{"version": float64(1), "path": "/workspace/updated"}
	loaded.WorkspaceReference = &updatedReference
	require.NoError(t, runs.UpdateRun(loaded))
	opts, err := list.NewOptions(&model.Run{}, 20, "", nil)
	require.NoError(t, err)
	page, _, _, err := runs.ListRuns(&model.FilterContext{}, opts, false)
	require.NoError(t, err)
	found := false
	for _, run := range page {
		if run.UUID == created.UUID {
			found = true
			assert.Equal(t, &executor, run.ExecutorType)
			assert.Equal(t, &provisioner, run.WorkspaceProvisioner)
			assert.Equal(t, &updatedReference, run.WorkspaceReference)
		}
	}
	require.True(t, found)
	argo, err := NewRunStore(db, util.NewFakeTimeForEpoch(), d).GetRun("1", false)
	require.NoError(t, err)
	assert.Nil(t, argo.ExecutorType)
	assert.Nil(t, argo.WorkspaceProvisioner)
	assert.Nil(t, argo.WorkspaceReference)
}
