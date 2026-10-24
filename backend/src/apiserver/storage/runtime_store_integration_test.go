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

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuntimeProductionDatabases runs shared scheduling contracts on SQLite and
// the disposable MySQL/PostgreSQL fixtures using KFP_RECURRING_{MYSQL,POSTGRES}_TEST_DSN.
// It validates store behavior, not online schema migration safety.
func TestRuntimeProductionDatabases(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		db, tasks, _, source := runtimeFixture(t)
		assertRuntimeSchedulingStorageContracts(t, tasks, source)
		t.Run("concurrent-managed-run-retry", func(t *testing.T) {
			assertConcurrentManagedRunRetry(t, []*sql.DB{db}, tasks, source)
		})
		t.Run("concurrent-managed-retry-delete", func(t *testing.T) {
			assertConcurrentManagedRetryAndDelete(t, []*sql.DB{db}, tasks, source)
		})
	})
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			clock, ids := util.NewFakeTimeForEpoch(), util.NewUUIDGenerator()
			experiments, err := NewExperimentStore(dbs[0], clock, ids, d)
			require.NoError(t, err)
			experiment, err := experiments.CreateExperiment(&model.Experiment{Name: "runtime"})
			require.NoError(t, err)
			runs := NewRunStore(dbs[0], clock, d)
			for _, id := range []string{"runtime-run", "other-run"} {
				_, err = runs.CreateRun(&model.Run{
					UUID: id, ExperimentId: experiment.UUID, DisplayName: id,
					RunDetails: model.RunDetails{State: model.RuntimeStateRunning},
				})
				require.NoError(t, err)
			}
			tasks := NewTaskStore(dbs[0], clock, ids, d)
			task, err := tasks.CreateTask(&model.Task{
				RunUUID: "runtime-run", Name: "node", Fingerprint: "fingerprint",
				State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
			})
			require.NoError(t, err)
			q, qb := d.QuoteIdentifier, d.QueryBuilder()
			query, args, err := qb.Insert(q("runtime_tasks")).SetMap(sq.Eq{
				q("TaskUUID"): task.UUID, q("RunUUID"): "other-run", q("State"): model.RuntimeTaskWaiting,
				q("RetryCount"): 0,
			}).ToSql()
			require.NoError(t, err)
			_, err = dbs[0].Exec(query, args...)
			require.Error(t, err, "composite task/run FK must reject mismatched existing run")

			runtime := tasks.RuntimeStore()
			attempt := runtimeAttemptFixture(task, "attempt")
			require.NoError(t, runtime.Create(attempt))
			attempt.State = model.RuntimeTaskPublishing
			attempt.ExecutionReference["completion"] = map[string]any{"count": float64(2)}
			update := runtimeOutputUpdate(t, task)
			require.NoError(t, runtime.Checkpoint(attempt, update))
			require.NoError(t, runtime.Checkpoint(attempt, update))

			retries := recurringConcurrent(len(dbs), func(i int) error {
				store := NewTaskStore(dbs[i], clock, ids, d).RuntimeStore()
				next := runtimeAttemptFixture(task, util.NewDeterministicUUID("retry:"+task.UUID))
				next.State = model.RuntimeTaskBackoff
				return store.BeginRetry(task.UUID, "attempt", next)
			})
			successes := 0
			for _, err := range retries {
				if err == nil {
					successes++
				} else {
					require.ErrorIs(t, err, ErrRuntimeTaskConflict)
				}
			}
			assert.Equal(t, 1, successes, "only one retry may replace the old attempt")
			current, err := runtime.Get(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, 1, current.RetryCount)
			assert.NotContains(t, current.ExecutionReference, "completion")
			public, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Empty(t, public.OutputParameters)
			var links int
			query, args, err = qb.Select("COUNT(*)").From(q("artifact_tasks")).
				Where(sq.Eq{q("TaskID"): task.UUID, q("Type"): apiv2.IOType_OUTPUT}).ToSql()
			require.NoError(t, err)
			require.NoError(t, dbs[0].QueryRow(query, args...).Scan(&links))
			assert.Zero(t, links)

			current.State = model.RuntimeTaskPublishing
			require.NoError(t, runtime.Checkpoint(current, update))
			completions := recurringConcurrent(len(dbs), func(i int) error {
				return NewTaskStore(dbs[i], clock, ids, d).RuntimeStore().Finalize(task.UUID, *current.AttemptUUID, &model.Task{
					UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), FinishedInSec: 123,
				})
			})
			successes = 0
			for _, err := range completions {
				if err == nil {
					successes++
				} else {
					require.True(t, errors.Is(err, ErrRuntimeTaskConflict), "%v", err)
				}
			}
			assert.Equal(t, 1, successes, "duplicate completions must never recreate deleted runtime state")
			_, err = runtime.Get(task.UUID)
			require.ErrorIs(t, err, sql.ErrNoRows)
			public, err = tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), public.State)

			pending := &model.Task{
				RunUUID: task.RunUUID, Namespace: task.Namespace, Name: "cached-node", ScopePath: "root.cached-node",
				Type: model.TaskType(apiv2.PipelineTask_RUNTIME), State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
				Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
			}
			cached, err := runtime.CreateTask(pending)
			require.NoError(t, err)
			replayed, err := runtime.CreateTask(pending)
			require.NoError(t, err)
			assert.Equal(t, cached.UUID, replayed.UUID)
			current, err = runtime.Get(cached.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.RuntimeTaskWaiting, current.State)
			assert.Nil(t, current.AttemptUUID)
			require.Len(t, public.OutputArtifactsHydrated, 1)
			link := &model.ArtifactTask{
				TaskID: cached.UUID, RunUUID: cached.RunUUID, ArtifactID: public.OutputArtifactsHydrated[0].Value.UUID,
				Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "result",
			}
			require.NoError(t, runtime.FinalizePending(cached.UUID, &model.Task{
				UUID: cached.UUID, State: model.TaskStatus(apiv2.PipelineTask_CACHED), OutputParameters: public.OutputParameters,
			}, []*model.ArtifactTask{link}))
			replayed, err = runtime.CreateTask(pending)
			require.NoError(t, err)
			assert.Equal(t, cached.UUID, replayed.UUID)
			assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_CACHED), replayed.State)
			_, err = runtime.Get(cached.UUID)
			require.ErrorIs(t, err, sql.ErrNoRows)

			query, args, err = qb.Update(q("run_details")).SetMap(sq.Eq{q("ExecutorType"): "kubernetes"}).
				Where(sq.Eq{q("UUID"): task.RunUUID}).ToSql()
			require.NoError(t, err)
			_, err = dbs[0].Exec(query, args...)
			require.NoError(t, err)
			failedInput := *pending
			failedInput.Name, failedInput.ScopePath = "failed-node", "root.failed-node"
			failed, err := runtime.CreateTask(&failedInput)
			require.NoError(t, err)
			canceledAttempt := runtimeAttemptFixture(failed, "cancellation-attempt")
			require.NoError(t, runtime.Update(canceledAttempt, ""))
			canceledAttempt.State = model.RuntimeTaskCanceling
			require.NoError(t, runtime.Update(canceledAttempt, "cancellation-attempt"))
			logUpdate, logArtifact, logLink := cancellationLogFixture(failed, "cancellation-attempt")
			require.NoError(t, runtime.CheckpointCancellationArtifacts(canceledAttempt, logUpdate,
				[]*model.Artifact{logArtifact}, []*model.ArtifactTask{logLink}))
			require.NoError(t, runtime.CheckpointCancellationArtifacts(canceledAttempt, logUpdate,
				[]*model.Artifact{logArtifact}, []*model.ArtifactTask{logLink}))
			require.NoError(t, runtime.Finalize(failed.UUID, "cancellation-attempt", &model.Task{
				UUID: failed.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
			}))
			require.NoError(t, tasks.ResetTasksForRetry([]string{failed.UUID, cached.UUID}))
			current, err = runtime.Get(failed.UUID)
			require.NoError(t, err)
			assert.Equal(t, &model.RuntimeTask{TaskUUID: failed.UUID, RunUUID: failed.RunUUID, State: model.RuntimeTaskWaiting}, current)
			failed, err = tasks.GetTask(failed.UUID)
			require.NoError(t, err)
			assert.Empty(t, failed.OutputArtifactsHydrated)
			cached, err = tasks.GetTask(cached.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_CACHED), cached.State)
			require.Len(t, cached.OutputArtifactsHydrated, 1)
			assertRuntimeSchedulingStorageContracts(t, tasks, task)
			t.Run("concurrent-managed-run-retry", func(t *testing.T) {
				assertConcurrentManagedRunRetry(t, dbs, tasks, task)
			})
			t.Run("concurrent-managed-retry-delete", func(t *testing.T) {
				assertConcurrentManagedRetryAndDelete(t, dbs, tasks, task)
			})
		})
	}
}
