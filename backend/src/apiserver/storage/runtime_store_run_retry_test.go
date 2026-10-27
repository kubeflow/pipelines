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
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type managedRunRetryFixture struct {
	run      *model.Run
	selected []*model.Task
	sibling  *model.Task
	log      *model.ArtifactTask
}

func newManagedRunRetryFixture(t *testing.T, tasks *TaskStore, source *model.Task) managedRunRetryFixture {
	t.Helper()
	runs := NewRunStore(tasks.db, tasks.time, tasks.dbDialect)
	owner, err := runs.GetRun(source.RunUUID, false)
	require.NoError(t, err)
	executor, provisioner := "kubernetes", "directory"
	workspace := model.JSONData{"version": float64(1), "path": "workspace-before-retry"}
	run, err := runs.CreateRun(&model.Run{
		UUID: util.NewDeterministicUUID(t.Name()), ExperimentId: owner.ExperimentId, Namespace: source.Namespace,
		DisplayName: "managed retry", StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			ExecutorType: &executor, WorkspaceProvisioner: &provisioner, WorkspaceReference: &workspace,
			State: model.RuntimeStateRunning, RetryGeneration: 7,
			PipelineRuntimeManifest: "before-manifest", WorkflowRuntimeManifest: "unchanged-legacy-manifest",
		},
	})
	require.NoError(t, err)
	fixture := managedRunRetryFixture{run: run}
	runtime := tasks.RuntimeStore()
	for _, spec := range []struct {
		name  string
		kind  apiv2.PipelineTask_TaskType
		state apiv2.PipelineTask_TaskState
	}{
		{"composite", apiv2.PipelineTask_DAG, apiv2.PipelineTask_SUCCEEDED},
		{"exit-descendant", apiv2.PipelineTask_EXIT_HANDLER, apiv2.PipelineTask_SKIPPED},
		{"failed-workload", apiv2.PipelineTask_RUNTIME, apiv2.PipelineTask_FAILED},
		{"successful-sibling", apiv2.PipelineTask_RUNTIME, apiv2.PipelineTask_SUCCEEDED},
	} {
		input := pendingTaskFixture(spec.name)
		input.RunUUID, input.Namespace, input.Type = run.UUID, source.Namespace, model.TaskType(spec.kind)
		if len(fixture.selected) > 0 && spec.name != "successful-sibling" {
			input.ParentTaskUUID = &fixture.selected[0].UUID
		}
		public, err := runtime.CreateTask(input)
		require.NoError(t, err)
		if spec.state == apiv2.PipelineTask_FAILED {
			attempt := runtimeAttemptFixture(public, "managed-retry-old")
			require.NoError(t, runtime.Update(attempt, ""))
			update := runtimeOutputUpdate(t, public)
			logUpdate, artifact, link := cancellationLogFixture(public, "managed-retry-old")
			update.StatusMetadata["logs"] = logUpdate.StatusMetadata["logs"]
			require.NoError(t, runtime.CheckpointArtifacts(attempt, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link}))
			fixture.log = link
			outputs, err := tasks.GetTask(public.UUID)
			require.NoError(t, err)
			for _, output := range outputs.OutputArtifactsHydrated {
				if output.Key == "result" {
					_, err := NewArtifactTaskStore(tasks.db, tasks.uuid, tasks.dbDialect).CreateArtifactTask(&model.ArtifactTask{
						TaskID: public.UUID, RunUUID: run.UUID, ArtifactID: output.Value.UUID,
						Type: model.IOType(apiv2.IOType_ITERATOR_OUTPUT), ArtifactKey: "iterator-result",
					})
					require.NoError(t, err)
				}
			}
			require.NoError(t, runtime.Finalize(public.UUID, "managed-retry-old", &model.Task{
				UUID: public.UUID, State: model.TaskStatus(spec.state), FinishedInSec: 123,
			}))
		} else {
			require.NoError(t, runtime.FinalizePending(public.UUID, &model.Task{
				UUID: public.UUID, State: model.TaskStatus(spec.state), FinishedInSec: 123,
			}, nil))
		}
		public, err = tasks.GetTask(public.UUID)
		require.NoError(t, err)
		if spec.name == "successful-sibling" {
			fixture.sibling = public
		} else {
			fixture.selected = append(fixture.selected, public)
		}
	}
	q := tasks.dbDialect.QuoteIdentifier
	updateRuntimeContractRun(t, tasks, run.UUID, sq.Eq{
		q("State"): model.RuntimeStateFailed, q("Conditions"): model.RuntimeStateFailed.ToExecutionPhase(),
		q("FinishedAtInSec"): 900, q("RetryGeneration"): 7, q("RetryClaimedAtInSec"): 77,
	})
	fixture.run, err = runs.GetRun(run.UUID, false)
	require.NoError(t, err)
	return fixture
}

func (fixture managedRunRetryFixture) taskIDs() []string {
	ids := make([]string, len(fixture.selected))
	for i, task := range fixture.selected {
		ids[i] = task.UUID
	}
	return ids
}

func assertManagedRunRetryReopensSelectedTasks(t *testing.T, tasks *TaskStore, source *model.Task, state model.RuntimeState, resetWorkspace bool) {
	t.Helper()
	fixture := newManagedRunRetryFixture(t, tasks, source)
	q := tasks.dbDialect.QuoteIdentifier
	updateRuntimeContractRun(t, tasks, fixture.run.UUID, sq.Eq{q("State"): state, q("Conditions"): state.ToExecutionPhase()})
	manifest := model.LargeText(`{"version":1,"retry":"requested"}`)
	runtime := tasks.RuntimeStore()
	ids := fixture.taskIDs()
	require.NoError(t, runtime.RetryRun(fixture.run.UUID, 7, append(ids, ids[0]), manifest, resetWorkspace))
	run, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).GetRun(fixture.run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStatePending, run.State)
	assert.Equal(t, string(model.RuntimeStatePending.ToExecutionPhase()), run.Conditions)
	assert.Zero(t, run.FinishedAtInSec)
	assert.EqualValues(t, 8, run.RetryGeneration)
	assert.Zero(t, run.RetryClaimedAtInSec)
	assert.Equal(t, manifest, run.PipelineRuntimeManifest)
	assert.Equal(t, fixture.run.WorkflowRuntimeManifest, run.WorkflowRuntimeManifest)
	assert.Equal(t, fixture.run.ExecutorType, run.ExecutorType)
	assert.Equal(t, fixture.run.WorkspaceProvisioner, run.WorkspaceProvisioner)
	if resetWorkspace {
		assert.Nil(t, run.WorkspaceReference)
	} else {
		assert.Equal(t, fixture.run.WorkspaceReference, run.WorkspaceReference)
	}
	for _, before := range fixture.selected {
		public, err := tasks.GetTask(before.UUID)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
		assert.Zero(t, public.FinishedInSec)
		assert.Empty(t, public.Pods)
		assert.Empty(t, public.OutputParameters)
		assert.Nil(t, public.StatusMetadata)
		assert.Equal(t, before.LogicalKey, public.LogicalKey)
		assert.Equal(t, before.ParentTaskUUID, public.ParentTaskUUID)
		assert.Equal(t, before.Type, public.Type)
		require.GreaterOrEqual(t, len(public.StateHistory), len(before.StateHistory))
		assert.Equal(t, before.StateHistory, public.StateHistory[:len(before.StateHistory)])
		current, err := runtime.Get(before.UUID)
		require.NoError(t, err)
		assert.Equal(t, &model.RuntimeTask{TaskUUID: before.UUID, RunUUID: fixture.run.UUID, State: model.RuntimeTaskWaiting}, current)
		if before.Name == "failed-workload" {
			require.Len(t, public.OutputArtifactsHydrated, 1)
			assert.Equal(t, "executor-logs", public.OutputArtifactsHydrated[0].Key)
			assert.Equal(t, fixture.log.ArtifactID, public.OutputArtifactsHydrated[0].Value.UUID)
			history, err := json.Marshal(public.StateHistory)
			require.NoError(t, err)
			assert.Contains(t, string(history), "worker result")
			assert.Contains(t, string(history), "attempt-log")
		}
	}
	sibling, err := tasks.GetTask(fixture.sibling.UUID)
	require.NoError(t, err)
	assert.Equal(t, fixture.sibling, sibling)
	_, err = runtime.Get(fixture.sibling.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	log, err := NewArtifactTaskStore(tasks.db, tasks.uuid, tasks.dbDialect).GetArtifactTask(fixture.log.UUID)
	require.NoError(t, err)
	assert.Equal(t, fixture.log.ArtifactID, log.ArtifactID)
	rows, err := runtime.ListByRun(fixture.run.UUID)
	require.NoError(t, err)
	assert.Len(t, rows, len(fixture.selected))
	require.ErrorIs(t, runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), manifest, resetWorkspace), ErrRuntimeTaskConflict)
}

func TestRuntimeRetryRunReopensSelectedTerminalTasks(t *testing.T) {
	_, tasks, _, source := runtimeFixture(t)
	assertManagedRunRetryContracts(t, tasks, source)
}

type managedRetrySnapshot struct {
	run     *model.Run
	tasks   map[string]*model.Task
	runtime []*model.RuntimeTask
}

func snapshotManagedRetry(t *testing.T, tasks *TaskStore, fixture managedRunRetryFixture) managedRetrySnapshot {
	t.Helper()
	run, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).GetRun(fixture.run.UUID, false)
	require.NoError(t, err)
	public := make(map[string]*model.Task)
	for _, task := range append(append([]*model.Task{}, fixture.selected...), fixture.sibling) {
		public[task.UUID], err = tasks.GetTask(task.UUID)
		require.NoError(t, err)
	}
	rows, err := tasks.RuntimeStore().ListByRun(fixture.run.UUID)
	require.NoError(t, err)
	return managedRetrySnapshot{run: run, tasks: public, runtime: rows}
}

func assertManagedRunRetryContracts(t *testing.T, tasks *TaskStore, source *model.Task) {
	t.Helper()
	t.Run("first-retry-generation-zero", func(t *testing.T) {
		fixture := newManagedRunRetryFixture(t, tasks, source)
		q := tasks.dbDialect.QuoteIdentifier
		updateRuntimeContractRun(t, tasks, fixture.run.UUID, sq.Eq{q("RetryGeneration"): 0})
		require.NoError(t, tasks.RuntimeStore().RetryRun(fixture.run.UUID, 0, fixture.taskIDs(), "first-retry-manifest", false))
		run, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).GetRun(fixture.run.UUID, false)
		require.NoError(t, err)
		assert.EqualValues(t, 1, run.RetryGeneration)
	})
	for _, state := range []model.RuntimeState{model.RuntimeStateFailed, model.RuntimeStateCanceled} {
		for _, resetWorkspace := range []bool{false, true} {
			name := string(state) + "-keep-workspace"
			if resetWorkspace {
				name = string(state) + "-reset-workspace"
			}
			t.Run(name, func(t *testing.T) {
				assertManagedRunRetryReopensSelectedTasks(t, tasks, source, state, resetWorkspace)
			})
		}
	}
	t.Run("rejected-preconditions", func(t *testing.T) {
		assertManagedRunRetryPreconditions(t, tasks, source)
	})
	for _, failure := range []struct{ name, table, event string }{
		{"runtime-insertion", "runtime_tasks", "INSERT"}, {"public-reset", "tasks", "UPDATE"}, {"late-run-reopen", "run_details", "UPDATE"},
	} {
		t.Run("rollback-"+failure.name, func(t *testing.T) {
			fixture := newManagedRunRetryFixture(t, tasks, source)
			before := snapshotManagedRetry(t, tasks, fixture)
			rejectRuntimeTableMutation(t, tasks, failure.table, failure.event)
			err := tasks.RuntimeStore().RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "new-manifest", true)
			require.ErrorContains(t, err, "runtime contract mutation rejected")
			assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture), "run, tasks, outputs, logs, workspace, and runtime rows must roll back together")
		})
	}
	t.Run("lease-fences", func(t *testing.T) {
		assertManagedRunRetryLeaseFences(t, tasks, source)
	})
	t.Run("lease-expiry-rolls-back", func(t *testing.T) {
		fixture := newManagedRunRetryFixture(t, tasks, source)
		before := snapshotManagedRetry(t, tasks, fixture)
		token, err := tasks.RunLeaseStore().Acquire(fixture.run.UUID, "expiring-owner", 300*time.Millisecond, true)
		require.NoError(t, err)
		bound, err := tasks.LeaseScopedStores(*token)
		require.NoError(t, err)
		clock := &delayedRetryClock{TimeInterface: tasks.time}
		bound.Tasks.time = clock
		require.ErrorIs(t, bound.Runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "new-manifest", true), ErrRunLeaseConflict)
		require.True(t, clock.called, "the lease must expire inside the transaction, not before entering it")
		assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
	})
}

func assertManagedRunRetryPreconditions(t *testing.T, tasks *TaskStore, source *model.Task) {
	t.Helper()
	for _, invalid := range []string{
		"archived", "legacy-archived", "unmanaged", "running", "pending", "canceling", "succeeded", "unknown-state",
		"stale-generation", "future-generation", "waiting-cleanup", "active-cleanup",
		"nonterminal-selected", "nonterminal-sibling", "foreign-task", "missing-task", "missing-run",
		"empty-task-id", "negative-generation", "overflow-generation", "empty-manifest",
	} {
		t.Run(invalid, func(t *testing.T) {
			fixture := newManagedRunRetryFixture(t, tasks, source)
			q := tasks.dbDialect.QuoteIdentifier
			runID, generation, ids, manifest := fixture.run.UUID, int64(7), fixture.taskIDs(), model.LargeText("new-manifest")
			runChanges := sq.Eq{}
			switch invalid {
			case "archived":
				runChanges[q("StorageState")] = model.StorageStateArchived
			case "legacy-archived":
				runChanges[q("StorageState")] = model.StorageStateArchivedV1
			case "unmanaged":
				runChanges[q("ExecutorType")] = nil
			case "running", "pending", "canceling", "succeeded", "unknown-state":
				runChanges[q("State")] = strings.ToUpper(invalid)
			case "stale-generation":
				generation = 6
			case "future-generation":
				generation = 8
			case "waiting-cleanup", "active-cleanup":
				values := sq.Eq{q("TaskUUID"): ids[0], q("RunUUID"): runID, q("State"): model.RuntimeTaskWaiting, q("RetryCount"): 0}
				if invalid == "active-cleanup" {
					values[q("State")], values[q("AttemptUUID")] = model.RuntimeTaskCancelFinalizing, "unfinished-attempt"
				}
				query, args, err := tasks.dbDialect.QueryBuilder().Insert(q("runtime_tasks")).SetMap(values).ToSql()
				require.NoError(t, err)
				_, err = tasks.db.Exec(query, args...)
				require.NoError(t, err)
			case "nonterminal-selected", "nonterminal-sibling":
				id := ids[0]
				if invalid == "nonterminal-sibling" {
					id = fixture.sibling.UUID
				}
				query, args, err := tasks.dbDialect.QueryBuilder().Update(q("tasks")).
					SetMap(sq.Eq{q("State"): model.TaskStatus(apiv2.PipelineTask_RUNNING)}).Where(sq.Eq{q("UUID"): id}).ToSql()
				require.NoError(t, err)
				_, err = tasks.db.Exec(query, args...)
				require.NoError(t, err)
			case "foreign-task":
				ids = append(ids, source.UUID)
			case "missing-task":
				ids = append(ids, "missing")
			case "missing-run":
				runID = "missing"
			case "empty-task-id":
				ids = append(ids, "")
			case "negative-generation":
				generation = -1
			case "overflow-generation":
				generation = math.MaxInt64
			case "empty-manifest":
				manifest = ""
			}
			if len(runChanges) != 0 {
				updateRuntimeContractRun(t, tasks, fixture.run.UUID, runChanges)
			}
			before := snapshotManagedRetry(t, tasks, fixture)
			err := tasks.RuntimeStore().RetryRun(runID, generation, ids, manifest, true)
			require.Error(t, err)
			switch invalid {
			case "archived", "legacy-archived":
				require.ErrorContains(t, err, "archived")
			case "empty-task-id", "negative-generation", "overflow-generation", "empty-manifest":
			default:
				require.ErrorIs(t, err, ErrRuntimeTaskConflict)
			}
			assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
		})
	}
}

func assertManagedRunRetryLeaseFences(t *testing.T, tasks *TaskStore, source *model.Task) {
	t.Helper()
	fixture := newManagedRunRetryFixture(t, tasks, source)
	before := snapshotManagedRetry(t, tasks, fixture)
	leases := tasks.RunLeaseStore()
	oldToken, err := leases.Acquire(fixture.run.UUID, "old-owner", time.Minute, true)
	require.NoError(t, err)
	old, err := tasks.LeaseScopedStores(*oldToken)
	require.NoError(t, err)
	require.ErrorIs(t, old.Runtime.RetryRun(source.RunUUID, 0, nil, "new-manifest", false), ErrRunLeaseScope)
	require.NoError(t, leases.Release(*oldToken))
	token, err := leases.Acquire(fixture.run.UUID, "current-owner", time.Minute, true)
	require.NoError(t, err)
	current, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	require.ErrorIs(t, old.Runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "new-manifest", true), ErrRunLeaseConflict)
	assert.Equal(t, before, snapshotManagedRetry(t, tasks, fixture))
	require.NoError(t, current.Runtime.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "new-manifest", false))
	require.NoError(t, leases.Release(*token))
	require.ErrorIs(t, old.Runtime.RetryRun(fixture.run.UUID, 8, fixture.taskIDs(), "another-manifest", false), ErrRunLeaseConflict)
}

type delayedRetryClock struct {
	util.TimeInterface
	once   sync.Once
	called bool
}

func (clock *delayedRetryClock) Now() time.Time {
	clock.once.Do(func() {
		clock.called = true
		time.Sleep(400 * time.Millisecond)
	})
	return clock.TimeInterface.Now()
}

func assertConcurrentManagedRunRetry(t *testing.T, dbs []*sql.DB, tasks *TaskStore, source *model.Task) {
	t.Helper()
	fixture := newManagedRunRetryFixture(t, tasks, source)
	workers := len(dbs)
	if workers < 2 {
		workers = 2
	}
	results := recurringConcurrent(workers, func(i int) error {
		store := NewTaskStore(dbs[i%len(dbs)], tasks.time, tasks.uuid, tasks.dbDialect).RuntimeStore()
		return store.RetryRun(fixture.run.UUID, 7, fixture.taskIDs(), "concurrent-manifest", false)
	})
	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, ErrRuntimeTaskConflict)
		}
	}
	assert.Equal(t, 1, successes, "only one generation-fenced user retry may reopen the run")
	run, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).GetRun(fixture.run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStatePending, run.State)
	assert.EqualValues(t, 8, run.RetryGeneration)
	rows, err := tasks.RuntimeStore().ListByRun(fixture.run.UUID)
	require.NoError(t, err)
	assert.Len(t, rows, len(fixture.selected))
}
