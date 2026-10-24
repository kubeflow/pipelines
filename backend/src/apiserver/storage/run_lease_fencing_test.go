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
	"errors"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLeaseAllOldOwnerMutationSurfacesFenced(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	assertRunLeaseMutationSurfaces(t, db, tasks, runtime, task)
}

func assertRunLeaseMutationSurfaces(t *testing.T, db *sql.DB, tasks *TaskStore, runtime RuntimeStoreInterface, task *model.Task) {
	t.Helper()
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Update(q("run_details")).
		Set(q("ExecutorType"), "kubernetes").Where(sq.Eq{q("UUID"): task.RunUUID}).ToSql()
	require.NoError(t, err)
	_, err = db.Exec(query, args...)
	require.NoError(t, err)
	current := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(current))
	leases := tasks.RunLeaseStore()
	oldToken, err := leases.Acquire(task.RunUUID, "old-worker", time.Minute, false)
	require.NoError(t, err)
	old, err := tasks.LeaseScopedStores(*oldToken)
	require.NoError(t, err)
	require.NoError(t, leases.Release(*oldToken))
	newToken, err := leases.Acquire(task.RunUUID, "new-worker", time.Minute, false)
	require.NoError(t, err)
	next, err := tasks.LeaseScopedStores(*newToken)
	require.NoError(t, err)
	run, err := next.Runs.GetRun(task.RunUUID, false)
	require.NoError(t, err)
	update := &model.Task{UUID: task.UUID, RunUUID: task.RunUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)}
	source := &model.Artifact{Namespace: task.Namespace, Name: "model", Type: model.ArtifactType(apiv2.Artifact_Model)}
	artifact, err := NewArtifactStore(db, tasks.time, tasks.uuid, tasks.dbDialect).CreateArtifact(source)
	require.NoError(t, err)
	link := &model.ArtifactTask{ArtifactID: artifact.UUID, TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "model"}
	publishing := *current
	publishing.State = model.RuntimeTaskPublishing
	canceling := *current
	canceling.State = model.RuntimeTaskCancelFinalizing
	terminal := &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)}
	retry := runtimeAttemptFixture(task, "retry-attempt")
	value := model.LargeText("stale")
	calls := map[string]func() error{
		"task-create": func() error {
			_, err := old.Tasks.CreateTask(&model.Task{RunUUID: task.RunUUID, Name: "stale"})
			return err
		},
		"task-update": func() error { _, err := old.Tasks.UpdateTask(update); return err },
		"task-retry":  func() error { return old.Tasks.ResetTasksForRetry([]string{task.UUID}) },
		"task-delete": func() error { return old.Tasks.DeleteTasksForRun(nil, task.RunUUID) },
		"run-update":  func() error { return old.Runs.UpdateRun(run) },
		"run-update-cas": func() error {
			_, err := old.Runs.UpdateRunIfRuntimeManifestsUnchanged(run, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest)
			return err
		},
		"run-workflow-cas": func() error {
			_, err := old.Runs.UpdateRunFromWorkflow(run, run.State, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest)
			return err
		},
		"run-archive":   func() error { return old.Runs.ArchiveRun(task.RunUUID) },
		"run-unarchive": func() error { return old.Runs.UnarchiveRun(task.RunUUID) },
		"run-terminate": func() error { return old.Runs.TerminateRun(task.RunUUID) },
		"run-plugins":   func() error { return old.Runs.UpdateRunPluginsOutput(task.RunUUID, &value) },
		"run-retry-claim": func() error {
			_, _, _, _, err := old.Runs.ClaimRunForRetry(task.RunUUID, false)
			return err
		},
		"run-retry-rollback":        func() error { return old.Runs.RollbackRetryClaim(task.RunUUID, "RUNNING", "Running", 0, 1) },
		"run-delete":                func() error { return old.Runs.DeleteRun(task.RunUUID) },
		"runtime-create":            func() error { return old.Runtime.Create(current) },
		"runtime-update":            func() error { return old.Runtime.Update(current, "attempt") },
		"runtime-checkpoint":        func() error { return old.Runtime.Checkpoint(&publishing, update) },
		"runtime-artifacts":         func() error { return old.Runtime.CheckpointArtifacts(&publishing, update, nil, nil) },
		"runtime-cancel-checkpoint": func() error { return old.Runtime.CheckpointCancellation(&canceling, update) },
		"runtime-finalize":          func() error { return old.Runtime.Finalize(task.UUID, "attempt", terminal) },
		"runtime-retry":             func() error { return old.Runtime.BeginRetry(task.UUID, "attempt", retry) },
		"runtime-finalize-run":      func() error { return old.Runtime.FinalizeRun(task.RunUUID, model.RuntimeStateSucceeded, 123) },
		"artifact-create":           func() error { _, err := old.Artifacts.CreateArtifact(source); return err },
		"artifact-with-task":        func() error { _, _, err := old.Artifacts.CreateArtifactWithTask(source, link); return err },
		"artifact-bulk": func() error {
			_, _, err := old.Artifacts.CreateArtifactsWithTasks([]*model.Artifact{source}, []*model.ArtifactTask{link})
			return err
		},
		"artifact-importer-reuse":      func() error { _, _, err := old.Artifacts.FindOrCreateArtifactWithTask(source, link); return err },
		"artifact-link":                func() error { _, err := old.ArtifactTasks.CreateArtifactTask(link); return err },
		"artifact-links-bulk":          func() error { _, err := old.ArtifactTasks.CreateArtifactTasks([]*model.ArtifactTask{link}); return err },
		"artifact-links-delete-output": func() error { return old.ArtifactTasks.DeleteOutputArtifactTasksByTaskIDs([]string{task.UUID}) },
		"artifact-links-delete-input":  func() error { return old.ArtifactTasks.DeleteInputArtifactTasksByTaskIDs([]string{task.UUID}) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) { require.ErrorIs(t, call(), ErrRunLeaseConflict) })
	}
	_, err = next.Tasks.UpdateTask(&model.Task{UUID: task.UUID, StatusMetadata: model.JSONData{"owner": "new"}})
	require.NoError(t, err)
	_, err = old.Tasks.ReportTasks(&model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt"},
		[]*model.Task{{UUID: task.UUID, StatusMetadata: model.JSONData{"launcher": "same-attempt"}}})
	require.NoError(t, err, "attempt-scoped launcher reports remain valid across worker generation changes")
	require.NoError(t, next.Runtime.Finalize(task.UUID, "attempt", terminal))
	require.ErrorIs(t, old.Runtime.Finalize(task.UUID, "attempt", terminal), ErrRunLeaseConflict,
		"finalization must check ownership even after the runtime row was removed")
	require.ErrorIs(t, old.Runtime.Create(current), ErrRunLeaseConflict, "stale callbacks cannot recreate deleted coordination state")
}

func TestRunLeaseExpiryRollbackAndScope(t *testing.T) {
	db, tasks, _, task := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	token, err := tasks.RunLeaseStore().Acquire(task.RunUUID, "worker", 200*time.Millisecond, false)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	_, err = bound.Tasks.CreateTask(&model.Task{RunUUID: "2", Name: "wrong-run"})
	require.ErrorIs(t, err, ErrRunLeaseScope)
	require.ErrorIs(t, bound.Runs.UpdateRun(&model.Run{UUID: "2"}), ErrRunLeaseScope)
	tx, err := bound.Tasks.lease.begin()
	require.NoError(t, err)
	_, err = tx.Exec("UPDATE tasks SET StatusMetadata = ? WHERE UUID = ?", `{"uncommitted":true}`, task.UUID)
	require.NoError(t, err)
	time.Sleep(250 * time.Millisecond)
	require.ErrorIs(t, tx.Commit(), ErrRunLeaseConflict)
	stored, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Empty(t, stored.StatusMetadata)
	require.ErrorIs(t, tasks.RunLeaseStore().Check(*token), ErrRunLeaseConflict)
	require.ErrorIs(t, tasks.RunLeaseStore().Renew(*token, time.Minute), ErrRunLeaseConflict)
	require.ErrorIs(t, tasks.RunLeaseStore().Release(*token), ErrRunLeaseConflict)
	second, err := tasks.RunLeaseStore().Acquire(task.RunUUID, "new", time.Minute, false)
	require.NoError(t, err)
	require.Greater(t, second.Generation, token.Generation)
	fresh, err := tasks.LeaseScopedStores(*second)
	require.NoError(t, err)
	artifact := &model.Artifact{Namespace: task.Namespace, Name: "rollback", Type: model.ArtifactType(apiv2.Artifact_Model)}
	fresh.Artifacts.createArtifactTaskInTransaction = func(sqlTransaction, *model.ArtifactTask) (*model.ArtifactTask, error) {
		return nil, errors.New("injected link failure")
	}
	_, _, err = fresh.Artifacts.CreateArtifactWithTask(artifact, &model.ArtifactTask{
		TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "rollback",
	})
	require.ErrorContains(t, err, "injected link failure")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Zero(t, count)
}
