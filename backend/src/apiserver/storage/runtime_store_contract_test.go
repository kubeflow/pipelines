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
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rejectRuntimeContractMutation injects a late transaction failure only in the
// disposable fixture's runtime table, never in the production migration path.
func rejectRuntimeContractMutation(t *testing.T, tasks *TaskStore, operation string) func() {
	t.Helper()
	return rejectRuntimeTableMutation(t, tasks, "runtime_tasks", operation)
}

func rejectRuntimeTableMutation(t *testing.T, tasks *TaskStore, tableName, operation string) func() {
	t.Helper()
	require.Contains(t, []string{"INSERT", "UPDATE", "DELETE"}, operation)
	require.Contains(t, []string{"runtime_tasks", "run_details", "tasks", "run_metrics", "resource_references"}, tableName)
	q := tasks.dbDialect.QuoteIdentifier
	name := q("reject_contract_" + tableName + "_" + strings.ToLower(operation))
	table := q(tableName)
	drop := "DROP TRIGGER IF EXISTS " + name
	if tasks.dbDialect.Name() == "pgx" {
		drop += " ON " + table
	}
	removed := false
	remove := func() {
		t.Helper()
		if removed {
			return
		}
		_, err := tasks.db.Exec(drop)
		require.NoError(t, err)
		if tasks.dbDialect.Name() == "pgx" {
			_, err = tasks.db.Exec("DROP FUNCTION IF EXISTS " + name + "()")
			require.NoError(t, err)
		}
		removed = true
	}
	t.Cleanup(remove)
	create := "CREATE TRIGGER " + name + " BEFORE " + operation + " ON " + table
	switch tasks.dbDialect.Name() {
	case "sqlite":
		create += " BEGIN SELECT RAISE(ABORT, 'runtime contract mutation rejected'); END"
	case "mysql":
		create += " FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'runtime contract mutation rejected'"
	case "pgx":
		_, err := tasks.db.Exec("CREATE FUNCTION " + name + "() RETURNS trigger LANGUAGE plpgsql AS " +
			"$$ BEGIN RAISE EXCEPTION 'runtime contract mutation rejected'; END; $$")
		require.NoError(t, err)
		create += " FOR EACH ROW EXECUTE FUNCTION " + name + "()"
	default:
		t.Fatalf("unsupported runtime contract fixture dialect %q", tasks.dbDialect.Name())
	}
	_, err := tasks.db.Exec(create)
	require.NoError(t, err)
	return remove
}

func runtimeContractTaskInput(source *model.Task, name string) *model.Task {
	input := pendingTaskFixture("contract-" + name)
	input.RunUUID, input.Namespace = source.RunUUID, source.Namespace
	return input
}

func updateRuntimeContractRun(t *testing.T, tasks *TaskStore, runID string, values sq.Eq) {
	t.Helper()
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Update(q("run_details")).
		SetMap(values).Where(sq.Eq{q("UUID"): runID}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
}

func assertRuntimeSchedulingStorageContracts(t *testing.T, tasks *TaskStore, source *model.Task) {
	t.Helper()
	runtime := tasks.RuntimeStore()
	q := tasks.dbDialect.QuoteIdentifier
	updateRuntimeContractRun(t, tasks, source.RunUUID, sq.Eq{q("ExecutorType"): "kubernetes"})

	t.Run("atomic-create-and-pending-finalize", func(t *testing.T) {
		input := runtimeContractTaskInput(source, "cached")
		restore := rejectRuntimeContractMutation(t, tasks, "INSERT")
		_, err := runtime.CreateTask(input)
		require.ErrorContains(t, err, "runtime contract mutation rejected")
		public, err := tasks.FindTaskByLogicalIdentity(input)
		require.NoError(t, err)
		assert.Nil(t, public, "failed runtime insertion must roll back public task creation")
		restore()
		public, err = runtime.CreateTask(input)
		require.NoError(t, err)
		replayed, err := runtime.CreateTask(input)
		require.NoError(t, err)
		assert.Equal(t, public.UUID, replayed.UUID)
		current, err := runtime.Get(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, &model.RuntimeTask{TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskWaiting}, current)
		assert.Empty(t, input.UUID)
		artifact, link := cachedOutputFixture(t, tasks, source, public)
		update := runtimeOutputUpdate(t, public)
		update.State, update.FinishedInSec, update.OutputArtifactsHydrated = model.TaskStatus(apiv2.PipelineTask_CACHED), 123, nil
		before, err := tasks.GetTask(public.UUID)
		require.NoError(t, err)
		restore = rejectRuntimeContractMutation(t, tasks, "DELETE")
		require.ErrorContains(t, runtime.FinalizePending(public.UUID, update, []*model.ArtifactTask{link}), "runtime contract mutation rejected")
		after, err := tasks.GetTask(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, before, after, "late delete failure must roll back public outputs and copied cache links")
		current, err = runtime.Get(public.UUID)
		require.NoError(t, err)
		assert.Nil(t, current.AttemptUUID)
		assert.Equal(t, model.RuntimeTaskWaiting, current.State)
		restore()
		require.NoError(t, runtime.FinalizePending(public.UUID, update, []*model.ArtifactTask{link, link}))
		after, err = tasks.GetTask(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, update.State, after.State)
		assert.Equal(t, update.OutputParameters, after.OutputParameters)
		require.Len(t, after.OutputArtifactsHydrated, 1)
		assert.Equal(t, artifact.UUID, after.OutputArtifactsHydrated[0].Value.UUID)
		_, err = runtime.Get(public.UUID)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.ErrorIs(t, runtime.FinalizePending(public.UUID, update, nil), ErrRuntimeTaskConflict)
		replayed, err = runtime.CreateTask(input)
		require.NoError(t, err)
		assert.Equal(t, public.UUID, replayed.UUID)
		assert.Equal(t, update.State, replayed.State, "creation replay must not resurrect terminal runtime state")

		skipped, err := runtime.CreateTask(runtimeContractTaskInput(source, "skipped"))
		require.NoError(t, err)
		require.NoError(t, runtime.FinalizePending(skipped.UUID, &model.Task{
			UUID: skipped.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}, nil))
		_, err = runtime.Get(skipped.UUID)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("atomic-manual-retry-waiting-reset", func(t *testing.T) {
		active, err := runtime.CreateTask(runtimeContractTaskInput(source, "retry-active"))
		require.NoError(t, err)
		initial := runtimeAttemptFixture(active, "contract-manual-initial")
		require.NoError(t, runtime.Update(initial, ""))
		attempt := runtimeAttemptFixture(active, "contract-manual-current")
		require.NoError(t, runtime.BeginRetry(active.UUID, "contract-manual-initial", attempt))
		require.NoError(t, runtime.CheckpointArtifacts(attempt, runtimeOutputUpdate(t, active), nil, nil))
		require.ErrorIs(t, runtime.FinalizePending(active.UUID, &model.Task{
			UUID: active.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}, nil), ErrRuntimeTaskConflict, "NULL-attempt finalization cannot replace a live attempt")
		failed, err := runtime.CreateTask(runtimeContractTaskInput(source, "retry-failed"))
		require.NoError(t, err)
		require.NoError(t, runtime.FinalizePending(failed.UUID, &model.Task{
			UUID: failed.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
		}, nil))
		taskIDs := []string{active.UUID, failed.UUID}
		for i, id := range taskIDs {
			query, args, err := tasks.dbDialect.QueryBuilder().Update(q("tasks")).
				SetMap(sq.Eq{q("CreatedAtInSec"): i + 1}).Where(sq.Eq{q("UUID"): id}).ToSql()
			require.NoError(t, err)
			_, err = tasks.db.Exec(query, args...)
			require.NoError(t, err)
		}
		reusable := map[string]*model.Task{}
		for _, state := range []apiv2.PipelineTask_TaskState{apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_SKIPPED} {
			task, err := runtime.CreateTask(runtimeContractTaskInput(source, "reuse-"+state.String()))
			require.NoError(t, err)
			require.NoError(t, runtime.FinalizePending(task.UUID, &model.Task{
				UUID: task.UUID, State: model.TaskStatus(state), FinishedInSec: 123,
			}, nil))
			reusable[task.UUID], err = tasks.GetTask(task.UUID)
			require.NoError(t, err)
			taskIDs = append(taskIDs, task.UUID)
		}
		before, err := tasks.GetTask(active.UUID)
		require.NoError(t, err)
		failedBefore, err := tasks.GetTask(failed.UUID)
		require.NoError(t, err)
		restore := rejectRuntimeContractMutation(t, tasks, "INSERT")
		require.ErrorContains(t, tasks.ResetTasksForRetry(taskIDs), "runtime contract mutation rejected")
		after, err := tasks.GetTask(active.UUID)
		require.NoError(t, err)
		assert.Equal(t, before, after, "second task insertion failure must roll back the first task's reset and artifact cleanup")
		current, err := runtime.Get(active.UUID)
		require.NoError(t, err)
		assert.Equal(t, attempt, current)
		_, err = runtime.Get(failed.UUID)
		require.ErrorIs(t, err, sql.ErrNoRows)
		failedAfter, err := tasks.GetTask(failed.UUID)
		require.NoError(t, err)
		assert.Equal(t, failedBefore, failedAfter, "failed WAITING insertion must also roll back its public task reset")
		restore()
		require.NoError(t, tasks.ResetTasksForRetry(taskIDs))
		for _, id := range []string{active.UUID, failed.UUID} {
			current, err := runtime.Get(id)
			require.NoError(t, err)
			assert.Equal(t, &model.RuntimeTask{TaskUUID: id, RunUUID: source.RunUUID, State: model.RuntimeTaskWaiting}, current)
			public, err := tasks.GetTask(id)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
			assert.Empty(t, public.OutputParameters)
			assert.Empty(t, public.OutputArtifactsHydrated)
		}
		for id, before := range reusable {
			after, err := tasks.GetTask(id)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			_, err = runtime.Get(id)
			require.ErrorIs(t, err, sql.ErrNoRows)
		}
		require.ErrorIs(t, runtime.Update(attempt, "contract-manual-current"), ErrRuntimeTaskConflict)
		require.NoError(t, runtime.Update(runtimeAttemptFixture(active, "contract-manual-new"), ""))
	})

	t.Run("atomic-cancellation-artifact-checkpoint", func(t *testing.T) {
		public, err := runtime.CreateTask(runtimeContractTaskInput(source, "cancel-logs"))
		require.NoError(t, err)
		attempt := runtimeAttemptFixture(public, "contract-cancel-attempt")
		require.NoError(t, runtime.Update(attempt, ""))
		attempt.State = model.RuntimeTaskCanceling
		require.NoError(t, runtime.Update(attempt, "contract-cancel-attempt"))
		token, err := tasks.RunLeaseStore().Acquire(source.RunUUID, "contract-worker", time.Minute, false)
		require.NoError(t, err)
		bound, err := tasks.LeaseScopedStores(*token)
		require.NoError(t, err)
		updateRuntimeContractRun(t, tasks, source.RunUUID, sq.Eq{q("State"): model.RuntimeStateCancelling})
		t.Cleanup(func() {
			updateRuntimeContractRun(t, tasks, source.RunUUID, sq.Eq{q("State"): model.RuntimeStateRunning})
		})
		before, err := runtime.Get(public.UUID)
		require.NoError(t, err)
		attempt.State = model.RuntimeTaskCancelFinalizing
		attempt.ExecutionReference["completion"] = map[string]any{"count": float64(1)}
		update, artifact, link := cancellationLogFixture(public, "contract-cancel-attempt")
		checkpoint := func() error {
			return bound.Runtime.CheckpointCancellationArtifacts(attempt, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link})
		}
		restore := rejectRuntimeContractMutation(t, tasks, "UPDATE")
		require.ErrorContains(t, checkpoint(), "runtime contract mutation rejected")
		current, err := runtime.Get(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, before, current)
		after, err := tasks.GetTask(public.UUID)
		require.NoError(t, err)
		assert.Empty(t, after.StatusMetadata)
		assert.Empty(t, after.OutputArtifactsHydrated)
		var count int
		query, args, err := tasks.dbDialect.QueryBuilder().Select("COUNT(*)").From(q("artifacts")).
			Where(sq.Eq{q("UUID"): artifact.UUID}).ToSql()
		require.NoError(t, err)
		require.NoError(t, tasks.db.QueryRow(query, args...).Scan(&count))
		assert.Zero(t, count, "artifact metadata must roll back when reference progress cannot commit")
		restore()
		require.NoError(t, checkpoint())
		require.NoError(t, checkpoint())
		after, err = tasks.GetTask(public.UUID)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), after.State)
		require.Len(t, after.OutputArtifactsHydrated, 1)
		assert.Equal(t, artifact.UUID, after.OutputArtifactsHydrated[0].Value.UUID)
		link.ArtifactKey = "normal-output"
		require.ErrorContains(t, checkpoint(), "executor-logs OUTPUT links")
		link.ArtifactKey = "executor-logs"
		normal := *attempt
		normal.State = model.RuntimeTaskRunning
		require.ErrorIs(t, bound.Runtime.CheckpointArtifacts(&normal, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link}), ErrRuntimeRunCanceled)
		require.NoError(t, bound.Runtime.Finalize(public.UUID, "contract-cancel-attempt", &model.Task{
			UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
		}))
		require.ErrorIs(t, checkpoint(), ErrRuntimeTaskConflict)
		require.NoError(t, tasks.RunLeaseStore().Release(*token))
		require.ErrorIs(t, checkpoint(), ErrRunLeaseConflict)
	})
	t.Run("atomic-managed-run-retry", func(t *testing.T) {
		assertManagedRunRetryContracts(t, tasks, source)
	})
	t.Run("atomic-managed-run-delete", func(t *testing.T) {
		assertManagedRunDeleteContracts(t, tasks, source)
	})
}
