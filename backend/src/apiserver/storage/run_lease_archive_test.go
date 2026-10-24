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
	"fmt"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertArchivedActiveLeaseTakeover(t *testing.T, tasks *TaskStore) {
	t.Helper()
	runs := NewRunStore(tasks.db, tasks.time, tasks.dbDialect)
	executor := "kubernetes"
	const runID = "archived-active-recovery"
	_, err := runs.CreateRun(&model.Run{UUID: runID, RunDetails: model.RunDetails{State: model.RuntimeStateRunning, ExecutorType: &executor}})
	require.NoError(t, err)
	task, err := tasks.CreateTask(&model.Task{RunUUID: runID, Name: "active", State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
	require.NoError(t, err)
	runtime := tasks.RuntimeStore()
	require.NoError(t, runtime.Create(runtimeAttemptFixture(task, "archived-attempt")))
	leases := tasks.RunLeaseStore()
	first, err := leases.Acquire(runID, "old-owner", time.Minute, false)
	require.NoError(t, err)
	old, err := tasks.LeaseScopedStores(*first)
	require.NoError(t, err)
	require.NoError(t, runs.ArchiveRun(runID))
	stored, err := runs.GetRun(runID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, stored.State, "archiving is not cancellation")
	assert.Equal(t, model.StorageStateArchived, stored.StorageState)
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Update(q("run_leases")).
		SetMap(sq.Eq{q("ExpiresAt"): sq.Expr(leaseDatabaseNow(tasks.dbDialect) + " - 1")}).
		Where(sq.Eq{q("RunUUID"): runID}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
	eligible, err := leases.ListEligible(executor, 8, true)
	require.NoError(t, err)
	require.Contains(t, eligible, runID)
	next, err := leases.Acquire(runID, "new-owner", time.Minute, false)
	require.NoError(t, err, "archived active work must remain claimable after owner loss")
	assert.Greater(t, next.Generation, first.Generation)
	_, err = old.Tasks.UpdateTask(&model.Task{UUID: task.UUID, StatusMetadata: model.JSONData{"stale": true}})
	require.ErrorIs(t, err, ErrRunLeaseConflict)
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, "archived-attempt", *current.AttemptUUID, "takeover must preserve the workload attempt")
}

func assertArchivedBatchDoesNotStarveUnrelatedRun(t *testing.T, tasks *TaskStore) {
	t.Helper()
	runs := NewRunStore(tasks.db, tasks.time, tasks.dbDialect)
	executor := "kubernetes"
	const batchSize = 8
	for i := range batchSize {
		id := fmt.Sprintf("archived-candidate-%d", i)
		_, err := runs.CreateRun(&model.Run{UUID: id, StorageState: model.StorageStateArchived, RunDetails: model.RunDetails{
			State: model.RuntimeStateRunning, ExecutorType: &executor, CreatedAtInSec: int64(i + 1),
		}})
		require.NoError(t, err)
		task, err := tasks.CreateTask(&model.Task{RunUUID: id, Name: "active", State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
		require.NoError(t, err)
		require.NoError(t, tasks.RuntimeStore().Create(runtimeAttemptFixture(task, fmt.Sprintf("batch-attempt-%d", i))))
	}
	const unrelated = "unrelated-fresh-work"
	_, err := runs.CreateRun(&model.Run{UUID: unrelated, RunDetails: model.RunDetails{
		State: model.RuntimeStatePending, ExecutorType: &executor, CreatedAtInSec: 100,
	}})
	require.NoError(t, err)
	leases := tasks.RunLeaseStore()
	first, err := leases.ListEligible(executor, batchSize, true)
	require.NoError(t, err)
	require.Len(t, first, batchSize)
	require.NotContains(t, first, unrelated)
	for _, id := range first {
		_, err := leases.Acquire(id, "recovery-worker", time.Minute, false)
		assert.NoError(t, err, "discovered archived recovery must be acquirable: %s", id)
	}
	next, err := leases.ListEligible(executor, batchSize, true)
	require.NoError(t, err)
	assert.Contains(t, next, unrelated, "a full archived batch must not keep reappearing ahead of unrelated work")
	for _, id := range first {
		assert.NotContains(t, next, id, "claimed archived work must leave discovery")
	}
	_, err = leases.Acquire(unrelated, "fresh-worker", time.Minute, false)
	require.NoError(t, err)
}

func TestRunLeaseArchivedActiveRunTakeover(t *testing.T) {
	_, tasks, _, _ := runtimeFixture(t)
	assertArchivedActiveLeaseTakeover(t, tasks)
}

func TestRunLeaseArchivedBatchDoesNotStarveUnrelatedRun(t *testing.T) {
	_, tasks, _, _ := runtimeFixture(t)
	assertArchivedBatchDoesNotStarveUnrelatedRun(t, tasks)
}

func TestRunLeaseLiveArchivedEligibility(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			for _, scenario := range []struct {
				name string
				run  func(*testing.T, *TaskStore)
			}{
				{"takeover", assertArchivedActiveLeaseTakeover},
				{"full-batch", assertArchivedBatchDoesNotStarveUnrelatedRun},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					_, _, tasks, _ := runLeaseLiveFixture(t, driver)
					scenario.run(t, tasks)
				})
			}
		})
	}
}
