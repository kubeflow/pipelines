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
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedManagedRunDeletion(t *testing.T, tasks *TaskStore, source *model.Task) managedRunRetryFixture {
	t.Helper()
	fixture := newManagedRunRetryFixture(t, tasks, source)
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Insert(q("run_metrics")).SetMap(sq.Eq{
		q("RunUUID"): fixture.run.UUID, q("NodeID"): "metrics-node", q("Name"): "accuracy",
		q("NumberValue"): 0.95, q("Format"): "RAW", q("Payload"): "{}",
	}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
	seedLegacyResourceReferences(t, tasks.db, tasks.dbDialect,
		&model.ResourceReference{
			ResourceUUID: fixture.run.UUID, ResourceType: model.RunResourceType,
			ReferenceUUID: fixture.run.ExperimentId, ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
		})
	// Historical incoming references are read/deleted, but the deprecated
	// reference writer no longer permits creating Run targets.
	query, args, err = tasks.dbDialect.QueryBuilder().Insert(q("resource_references")).SetMap(sq.Eq{
		q("ResourceUUID"): fixture.sibling.UUID, q("ResourceType"): model.TaskResourceType,
		q("ReferenceUUID"): fixture.run.UUID, q("ReferenceType"): model.RunResourceType,
		q("ReferenceName"): "managed run", q("Relationship"): model.OwnerRelationship, q("Payload"): "{}",
	}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
	return fixture
}

func managedRunDeletionCounts(t *testing.T, tasks *TaskStore, runID string) map[string]int {
	t.Helper()
	q := tasks.dbDialect.QuoteIdentifier
	counts := make(map[string]int)
	for _, table := range []string{"run_details", "tasks", "artifact_tasks", "runtime_tasks", "run_metrics", "run_leases", "resource_references"} {
		var where sq.Sqlizer = sq.Eq{q("RunUUID"): runID}
		switch table {
		case "run_details":
			where = sq.Eq{q("UUID"): runID}
		case "resource_references":
			where = sq.Or{
				sq.Eq{q("ResourceUUID"): runID, q("ResourceType"): model.RunResourceType},
				sq.Eq{q("ReferenceUUID"): runID, q("ReferenceType"): model.RunResourceType},
			}
		}
		query, args, err := tasks.dbDialect.QueryBuilder().Select("COUNT(*)").From(q(table)).Where(where).ToSql()
		require.NoError(t, err)
		var count int
		require.NoError(t, tasks.db.QueryRow(query, args...).Scan(&count))
		counts[table] = count
	}
	return counts
}

func TestRuntimeDeleteRunCleansAllRelatedTables(t *testing.T) {
	_, tasks, _, source := runtimeFixture(t)
	assertManagedRunDeleteContracts(t, tasks, source)
}

func assertManagedRunDelete(t *testing.T, tasks *TaskStore, source *model.Task, state model.RuntimeState) {
	t.Helper()
	fixture := seedManagedRunDeletion(t, tasks, source)
	q := tasks.dbDialect.QuoteIdentifier
	updateRuntimeContractRun(t, tasks, fixture.run.UUID, sq.Eq{q("State"): state, q("Conditions"): state.ToExecutionPhase()})
	token, err := tasks.RunLeaseStore().Acquire(fixture.run.UUID, "delete-owner", time.Minute, true)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	before := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
	assert.Positive(t, before["tasks"])
	assert.Positive(t, before["artifact_tasks"])
	assert.Equal(t, 1, before["run_metrics"])
	assert.Equal(t, 2, before["resource_references"])
	assert.Equal(t, 1, before["run_leases"])
	require.NoError(t, bound.Runtime.DeleteRun(fixture.run.UUID, 7))
	for table, count := range managedRunDeletionCounts(t, tasks, fixture.run.UUID) {
		assert.Zero(t, count, table)
	}
	artifact, err := NewArtifactStore(tasks.db, tasks.time, tasks.uuid, tasks.dbDialect).GetArtifact(fixture.log.ArtifactID)
	require.NoError(t, err)
	assert.Equal(t, fixture.log.ArtifactID, artifact.UUID, "ordinary run deletion retains standalone artifact metadata")
	require.ErrorIs(t, bound.Runtime.DeleteRun(fixture.run.UUID, 7), ErrRunLeaseConflict)
	require.ErrorIs(t, tasks.RuntimeStore().DeleteRun(fixture.run.UUID, 7), ErrRuntimeTaskConflict)
	_, err = tasks.GetTask(source.UUID)
	require.NoError(t, err, "unrelated runs and tasks must remain intact")
}

func assertManagedRunDeleteContracts(t *testing.T, tasks *TaskStore, source *model.Task) {
	t.Helper()
	for _, state := range []model.RuntimeState{model.RuntimeStateFailed, model.RuntimeStateCanceled} {
		for _, phase := range []string{model.RuntimeTaskWaiting, model.RuntimeTaskReady} {
			t.Run("never-started-cleanup/"+string(state)+"/"+phase, func(t *testing.T) {
				assertNeverStartedTerminalRunCleanup(t, tasks, source, state, phase)
			})
		}
	}
	for _, state := range []model.RuntimeState{
		model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateSucceeded, model.RuntimeStateSkipped,
	} {
		t.Run(string(state), func(t *testing.T) {
			assertManagedRunDelete(t, tasks, source, state)
		})
	}
	for _, invalid := range []string{
		"stale-generation", "future-generation", "unmanaged", "running", "pending", "paused", "canceling",
		"unknown-state", "waiting-cleanup", "active-cleanup", "missing-run", "empty-run-id", "negative-generation",
	} {
		t.Run("reject-"+invalid, func(t *testing.T) {
			fixture := seedManagedRunDeletion(t, tasks, source)
			q := tasks.dbDialect.QuoteIdentifier
			runID, generation := fixture.run.UUID, int64(7)
			switch invalid {
			case "stale-generation":
				generation = 6
			case "future-generation":
				generation = 8
			case "unmanaged":
				updateRuntimeContractRun(t, tasks, runID, sq.Eq{q("ExecutorType"): nil})
			case "running", "pending", "paused", "canceling", "unknown-state":
				updateRuntimeContractRun(t, tasks, runID, sq.Eq{q("State"): strings.ToUpper(invalid)})
			case "waiting-cleanup", "active-cleanup":
				values := sq.Eq{q("TaskUUID"): fixture.selected[0].UUID, q("RunUUID"): runID, q("State"): model.RuntimeTaskWaiting, q("RetryCount"): 0}
				if invalid == "active-cleanup" {
					values[q("State")], values[q("AttemptUUID")] = model.RuntimeTaskCancelFinalizing, "unfinished-delete-attempt"
				}
				query, args, err := tasks.dbDialect.QueryBuilder().Insert(q("runtime_tasks")).SetMap(values).ToSql()
				require.NoError(t, err)
				_, err = tasks.db.Exec(query, args...)
				require.NoError(t, err)
			case "missing-run":
				runID = "missing"
			case "empty-run-id":
				runID = ""
			case "negative-generation":
				generation = -1
			}
			before := snapshotManagedRetry(t, tasks, fixture)
			counts := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
			err := tasks.RuntimeStore().DeleteRun(runID, generation)
			require.Error(t, err)
			if invalid != "empty-run-id" && invalid != "negative-generation" {
				require.ErrorIs(t, err, ErrRuntimeTaskConflict)
			}
			assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
			assert.Equal(t, counts, managedRunDeletionCounts(t, tasks, fixture.run.UUID))
		})
	}
	t.Run("retry-reopen-rejects-old-delete", func(t *testing.T) {
		fixture := seedManagedRunDeletion(t, tasks, source)
		runtime := tasks.RuntimeStore()
		require.NoError(t, runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "reopened-manifest", false))
		before := snapshotManagedRetry(t, tasks, fixture)
		counts := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
		require.ErrorIs(t, runtime.DeleteRun(fixture.run.UUID, 7), ErrRuntimeTaskConflict)
		require.ErrorIs(t, runtime.DeleteRun(fixture.run.UUID, 8), ErrRuntimeTaskConflict, "a known new generation is still nonterminal")
		assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
		assert.Equal(t, counts, managedRunDeletionCounts(t, tasks, fixture.run.UUID))
	})
	t.Run("delete-prevents-old-retry", func(t *testing.T) {
		fixture := seedManagedRunDeletion(t, tasks, source)
		require.NoError(t, tasks.RuntimeStore().DeleteRun(fixture.run.UUID, 7))
		require.ErrorIs(t, tasks.RuntimeStore().RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "stale-manifest", false), ErrRuntimeTaskConflict)
	})
	for _, table := range []string{"run_metrics", "tasks", "resource_references", "run_details"} {
		t.Run("rollback-"+table, func(t *testing.T) {
			fixture := seedManagedRunDeletion(t, tasks, source)
			before := snapshotManagedRetry(t, tasks, fixture)
			counts := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
			rejectRuntimeTableMutation(t, tasks, table, "DELETE")
			require.ErrorContains(t, tasks.RuntimeStore().DeleteRun(fixture.run.UUID, 7), "runtime contract mutation rejected")
			assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
			assert.Equal(t, counts, managedRunDeletionCounts(t, tasks, fixture.run.UUID))
		})
	}
	t.Run("scope-and-stale-lease", func(t *testing.T) {
		fixture := seedManagedRunDeletion(t, tasks, source)
		before := snapshotManagedRetry(t, tasks, fixture)
		leases := tasks.RunLeaseStore()
		oldToken, err := leases.Acquire(fixture.run.UUID, "old-delete-owner", time.Minute, true)
		require.NoError(t, err)
		old, err := tasks.LeaseScopedStores(*oldToken)
		require.NoError(t, err)
		require.ErrorIs(t, old.Runtime.DeleteRun(source.RunUUID, 0), ErrRunLeaseScope)
		require.NoError(t, leases.Release(*oldToken))
		token, err := leases.Acquire(fixture.run.UUID, "new-retry-owner", time.Minute, true)
		require.NoError(t, err)
		current, err := tasks.LeaseScopedStores(*token)
		require.NoError(t, err)
		require.ErrorIs(t, old.Runtime.DeleteRun(fixture.run.UUID, 7), ErrRunLeaseConflict)
		assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
		require.NoError(t, current.Runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "reopened-manifest", false))
		require.ErrorIs(t, old.Runtime.DeleteRun(fixture.run.UUID, 7), ErrRunLeaseConflict)
		require.ErrorIs(t, current.Runtime.DeleteRun(fixture.run.UUID, 7), ErrRuntimeTaskConflict)
		require.NoError(t, leases.Release(*token))
	})
	t.Run("expiry-after-cascaded-lease-delete-rolls-back", func(t *testing.T) {
		fixture := seedManagedRunDeletion(t, tasks, source)
		before := snapshotManagedRetry(t, tasks, fixture)
		counts := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
		delayManagedRunDeletion(t, tasks)
		token, err := tasks.RunLeaseStore().Acquire(fixture.run.UUID, "expiring-delete-owner", 300*time.Millisecond, true)
		require.NoError(t, err)
		counts["run_leases"] = 1
		bound, err := tasks.LeaseScopedStores(*token)
		require.NoError(t, err)
		start := time.Now()
		require.ErrorIs(t, bound.Runtime.DeleteRun(fixture.run.UUID, 7), ErrRunLeaseConflict)
		require.GreaterOrEqual(t, time.Since(start), 400*time.Millisecond, "the lease must expire inside the actual deletion transaction")
		assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
		assert.Equal(t, counts, managedRunDeletionCounts(t, tasks, fixture.run.UUID), "rollback must restore even the cascaded lease row")
	})
}

func assertNeverStartedTerminalRunCleanup(t *testing.T, tasks *TaskStore, source *model.Task, state model.RuntimeState, phase string) {
	t.Helper()
	fixture := seedManagedRunDeletion(t, tasks, source)
	input := runtimeContractTaskInput(source, "never-started")
	input.RunUUID = fixture.run.UUID
	public, err := tasks.RuntimeStore().CreateTask(input)
	require.NoError(t, err)
	row, err := tasks.RuntimeStore().Get(public.UUID)
	require.NoError(t, err)
	row.State = phase
	if phase == model.RuntimeTaskReady {
		row.ExecutionRequest, row.ExecutionReference = model.JSONData{}, model.JSONData{}
	}
	require.NoError(t, tasks.RuntimeStore().Update(row, ""))
	q := tasks.dbDialect.QuoteIdentifier
	updateRuntimeContractRun(t, tasks, fixture.run.UUID, sq.Eq{q("State"): state, q("Conditions"): state.ToExecutionPhase()})
	runs := NewRunStore(tasks.db, tasks.time, tasks.dbDialect)
	before, err := runs.GetRun(fixture.run.UUID, false)
	require.NoError(t, err)
	token, err := tasks.RunLeaseStore().Acquire(fixture.run.UUID, "never-started-cleanup-owner", time.Minute, true)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	diagnostic := model.JSONData{"diagnostic": "pipeline rejected before dispatch: executor mismatch"}
	update := &model.Task{
		UUID: public.UUID, RunUUID: fixture.run.UUID, Namespace: source.Namespace,
		State: model.TaskStatus(apiv2.PipelineTask_SKIPPED), StatusMetadata: diagnostic, FinishedInSec: 456,
	}
	_, err = tasks.ReportTasks(&model.TaskReportIdentity{TaskUUID: public.UUID, AttemptUUID: "never-assigned"}, []*model.Task{update})
	require.Error(t, err, "normal task reports cannot bypass the trusted never-started cleanup fence")
	require.NoError(t, bound.Runtime.FinalizePending(public.UUID, update, nil))
	public, err = tasks.GetTask(public.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), public.State)
	assert.Equal(t, diagnostic, public.StatusMetadata)
	_, err = tasks.RuntimeStore().Get(public.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	after, err := runs.GetRun(fixture.run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, before, after, "never-started task cleanup cannot change the run outcome, manifest, or generation")
	require.NoError(t, bound.Runtime.DeleteRun(fixture.run.UUID, 7), "terminal run deletion becomes safe after pruning never-started tasks")
}

func TestRuntimeNeverStartedTerminalRunCleanup(t *testing.T) {
	_, tasks, _, source := runtimeFixture(t)
	assertNeverStartedTerminalRunCleanup(t, tasks, source, model.RuntimeStateCanceled, model.RuntimeTaskWaiting)
}

func TestRuntimeNeverStartedCleanupRejectsUnknownWork(t *testing.T) {
	for _, invalid := range []string{"prepared-request", "prepared-request-terminal-task", "execution-reference", "submitting", "running", "real-attempt", "canceling-run", "namespace", "normal-outputs"} {
		t.Run(invalid, func(t *testing.T) {
			db, tasks, runtime, public := runtimeFixture(t)
			row := &model.RuntimeTask{TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskWaiting}
			state := model.RuntimeStateCanceled
			update := &model.Task{UUID: public.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED), StatusMetadata: model.JSONData{"diagnostic": "rejected before dispatch"}}
			switch invalid {
			case "prepared-request", "prepared-request-terminal-task":
				row.ExecutionRequest = model.JSONData{"version": float64(1)}
			case "execution-reference":
				row.ExecutionReference = model.JSONData{"backend": "unknown"}
			case "submitting":
				row.State = model.RuntimeTaskSubmitting
			case "running":
				row.State = model.RuntimeTaskRunning
			case "real-attempt":
				attempt := "current-attempt"
				row.AttemptUUID = &attempt
			case "canceling-run":
				state = model.RuntimeStateCancelling
			case "namespace":
				update.Namespace = "other-namespace"
			case "normal-outputs":
				update.OutputParameters = runtimeOutputUpdate(t, public).OutputParameters
			}
			require.NoError(t, runtime.Create(row))
			if invalid == "prepared-request-terminal-task" {
				_, err := db.Exec("UPDATE tasks SET State = ? WHERE UUID = ?", apiv2.PipelineTask_SKIPPED, public.UUID)
				require.NoError(t, err)
			}
			_, err := db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", state, public.RunUUID)
			require.NoError(t, err)
			before, err := tasks.GetTask(public.UUID)
			require.NoError(t, err)
			require.Error(t, runtime.FinalizePending(public.UUID, update, nil))
			after, err := tasks.GetTask(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			current, err := runtime.Get(public.UUID)
			require.NoError(t, err)
			assert.Equal(t, row, current)
		})
	}
}

func delayManagedRunDeletion(t *testing.T, tasks *TaskStore) {
	t.Helper()
	q := tasks.dbDialect.QuoteIdentifier
	name := q("delay_managed_run_delete")
	table := q("run_details")
	drop := "DROP TRIGGER IF EXISTS " + name
	if tasks.dbDialect.Name() == "pgx" {
		drop += " ON " + table
	}
	t.Cleanup(func() {
		_, err := tasks.db.Exec(drop)
		require.NoError(t, err)
		if tasks.dbDialect.Name() == "pgx" {
			_, err = tasks.db.Exec("DROP FUNCTION IF EXISTS " + name + "()")
			require.NoError(t, err)
		}
	})
	create := "CREATE TRIGGER " + name + " BEFORE DELETE ON " + table
	switch tasks.dbDialect.Name() {
	case "sqlite":
		conn, err := tasks.db.Conn(context.Background())
		require.NoError(t, err)
		err = conn.Raw(func(driverConn any) error {
			return driverConn.(*sqlite3.SQLiteConn).RegisterFunc("delay_managed_delete", func() int {
				time.Sleep(400 * time.Millisecond)
				return 1
			}, false)
		})
		require.NoError(t, err)
		require.NoError(t, conn.Close())
		create += " BEGIN SELECT delay_managed_delete(); END"
	case "mysql":
		create += " FOR EACH ROW DO SLEEP(0.4)"
	case "pgx":
		_, err := tasks.db.Exec("CREATE FUNCTION " + name + "() RETURNS trigger LANGUAGE plpgsql AS " +
			"$$ BEGIN PERFORM pg_sleep(0.4); RETURN OLD; END; $$")
		require.NoError(t, err)
		create += " FOR EACH ROW EXECUTE FUNCTION " + name + "()"
	default:
		t.Fatalf("unsupported managed deletion fixture dialect %q", tasks.dbDialect.Name())
	}
	_, err := tasks.db.Exec(create)
	require.NoError(t, err)
}

func assertConcurrentManagedRetryAndDelete(t *testing.T, dbs []*sql.DB, tasks *TaskStore, source *model.Task) {
	t.Helper()
	fixture := seedManagedRunDeletion(t, tasks, source)
	results := recurringConcurrent(2, func(i int) error {
		runtime := NewTaskStore(dbs[i%len(dbs)], tasks.time, tasks.uuid, tasks.dbDialect).RuntimeStore()
		if i == 0 {
			return runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "racing-retry-manifest", false)
		}
		return runtime.DeleteRun(fixture.run.UUID, 7)
	})
	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, ErrRuntimeTaskConflict)
		}
	}
	assert.Equal(t, 1, successes, "retry and deletion at the same generation cannot both commit")
	counts := managedRunDeletionCounts(t, tasks, fixture.run.UUID)
	if results[0] == nil {
		run, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).GetRun(fixture.run.UUID, false)
		require.NoError(t, err)
		assert.Equal(t, model.RuntimeStatePending, run.State)
		assert.EqualValues(t, 8, run.RetryGeneration)
		assert.Equal(t, len(fixture.selected), counts["runtime_tasks"])
		assert.Equal(t, 1, counts["run_metrics"])
		assert.Equal(t, 2, counts["resource_references"])
	} else {
		for table, count := range counts {
			assert.Zero(t, count, table)
		}
	}
}
