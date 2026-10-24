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
	"fmt"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runLeaseLiveFixture(t *testing.T, driver string) ([]*sql.DB, dialect.DBDialect, *TaskStore, *model.Task) {
	t.Helper()
	dbs, d := recurringIntegrationDatabases(t, driver)
	clock := util.NewFakeTime(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	tasks := NewTaskStore(dbs[0], clock, util.NewUUIDGenerator(), d)
	executor := "kubernetes"
	_, err := NewRunStore(dbs[0], clock, d).CreateRun(&model.Run{
		UUID: "lease-run", RunDetails: model.RunDetails{State: model.RuntimeStateRunning, ExecutorType: &executor},
	})
	require.NoError(t, err)
	task, err := tasks.CreateTask(&model.Task{RunUUID: "lease-run", Name: "task", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
	require.NoError(t, err)
	return dbs, d, tasks, task
}

func TestRunLeaseLiveMutationSurfaceMatrix(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, _, tasks, task := runLeaseLiveFixture(t, driver)
			assertRunLeaseMutationSurfaces(t, dbs[0], tasks, tasks.RuntimeStore(), task)
		})
	}
}

func TestRunLeaseLiveSimultaneousClaimsAndCleanup(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d, tasks, task := runLeaseLiveFixture(t, driver)
			discovered, err := tasks.RunLeaseStore().ListEligible("kubernetes", 2, true)
			require.NoError(t, err)
			assert.Equal(t, []string{task.RunUUID}, discovered)
			tokens := make([]*RunLeaseToken, len(dbs))
			errs := recurringConcurrent(len(dbs), func(i int) error {
				var err error
				tokens[i], err = NewTaskStore(dbs[i], tasks.time, tasks.uuid, d).RunLeaseStore().
					Acquire(task.RunUUID, fmt.Sprintf("replica-%d", i), time.Minute, false)
				return err
			})
			var winner *RunLeaseToken
			for i, err := range errs {
				if err == nil {
					require.Nil(t, winner)
					winner = tokens[i]
				} else {
					require.ErrorIs(t, err, ErrRunLeaseConflict)
				}
			}
			require.NotNil(t, winner)
			store := tasks.RunLeaseStore()
			discovered, err = store.ListEligible("kubernetes", 2, false)
			require.NoError(t, err)
			assert.Empty(t, discovered, "discovery must not return a live owner")
			require.NoError(t, store.Renew(*winner, time.Minute))
			q, qb := d.QuoteIdentifier, d.QueryBuilder()
			query, args, err := qb.Update(q("run_leases")).Set(q("ExpiresAt"), sq.Expr(leaseDatabaseNow(d)+" - 1")).
				Where(sq.Eq{q("RunUUID"): task.RunUUID}).ToSql()
			require.NoError(t, err)
			_, err = dbs[0].Exec(query, args...)
			require.NoError(t, err)
			require.ErrorIs(t, store.Check(*winner), ErrRunLeaseConflict)
			require.ErrorIs(t, store.Renew(*winner, time.Minute), ErrRunLeaseConflict)
			require.ErrorIs(t, store.Release(*winner), ErrRunLeaseConflict)
			next, err := store.Acquire(task.RunUUID, "takeover", time.Minute, false)
			require.NoError(t, err)
			assert.Greater(t, next.Generation, winner.Generation)
			query, args, err = qb.Update(q("run_details")).Set(q("State"), string(model.RuntimeStateCanceled)).
				Where(sq.Eq{q("UUID"): task.RunUUID}).ToSql()
			require.NoError(t, err)
			_, err = dbs[0].Exec(query, args...)
			require.NoError(t, err)
			require.NoError(t, store.Check(*next), "cleanup remains valid after the public terminal state")
			require.NoError(t, store.Renew(*next, time.Minute))
			require.NoError(t, store.Release(*next))
			_, err = store.Acquire(task.RunUUID, "cleanup", time.Minute, false)
			require.ErrorIs(t, err, ErrRunLeaseIneligible)
			cleanup, err := store.Acquire(task.RunUUID, "cleanup", time.Minute, true)
			require.NoError(t, err)
			bound, err := tasks.LeaseScopedStores(*cleanup)
			require.NoError(t, err)
			require.NoError(t, bound.Runs.DeleteRun(task.RunUUID))
			require.ErrorIs(t, store.Check(*cleanup), ErrRunLeaseConflict)
		})
	}
}

func TestRunLeaseLiveExpiryAfterLockWait(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d, tasks, task := runLeaseLiveFixture(t, driver)
			store := tasks.RunLeaseStore()
			lockLease := func() *sql.Tx {
				tx, err := dbs[1].Begin()
				require.NoError(t, err)
				q := d.QuoteIdentifier
				query, args, err := d.QueryBuilder().Select(q("Generation")).From(q("run_leases")).
					Where(sq.Eq{q("RunUUID"): task.RunUUID}).ToSql()
				require.NoError(t, err)
				var generation int64
				require.NoError(t, tx.QueryRow(d.SelectForUpdate(query), args...).Scan(&generation))
				return tx
			}
			sessionID := func(db *sql.DB) int64 {
				query := "SELECT CONNECTION_ID()"
				if driver == "pgx" {
					query = "SELECT pg_backend_pid()"
				}
				var id int64
				require.NoError(t, db.QueryRow(query).Scan(&id))
				return id
			}
			first, err := store.Acquire(task.RunUUID, "renew-waiter", 400*time.Millisecond, false)
			require.NoError(t, err)
			tx := lockLease()
			defer tx.Rollback()
			renewStore := NewTaskStore(dbs[2], tasks.time, tasks.uuid, d).RunLeaseStore()
			renewID := sessionID(dbs[2])
			renewed := make(chan error, 1)
			go func() { renewed <- renewStore.Renew(*first, time.Minute) }()
			recurringWaitForLock(t, dbs[3], driver, renewID)
			time.Sleep(500 * time.Millisecond)
			require.NoError(t, tx.Commit())
			require.ErrorIs(t, <-renewed, ErrRunLeaseConflict, "blocked renewal must sample time after its lock wait")

			second, err := store.Acquire(task.RunUUID, "write-waiter", 400*time.Millisecond, false)
			require.NoError(t, err)
			tx = lockLease()
			defer tx.Rollback()
			otherTasks := NewTaskStore(dbs[4], tasks.time, tasks.uuid, d)
			bound, err := otherTasks.LeaseScopedStores(*second)
			require.NoError(t, err)
			writerID := sessionID(dbs[4])
			written := make(chan error, 1)
			go func() {
				_, err := bound.Tasks.UpdateTask(&model.Task{UUID: task.UUID, StatusMetadata: model.JSONData{"expired": true}})
				written <- err
			}()
			recurringWaitForLock(t, dbs[3], driver, writerID)
			time.Sleep(500 * time.Millisecond)
			require.NoError(t, tx.Commit())
			require.ErrorIs(t, <-written, ErrRunLeaseConflict, "guarded write must sample time after its lock wait")
			stored, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Empty(t, stored.StatusMetadata)

			third, err := store.Acquire(task.RunUUID, "transaction-expiry", 400*time.Millisecond, false)
			require.NoError(t, err)
			bound, err = tasks.LeaseScopedStores(*third)
			require.NoError(t, err)
			mutation, err := bound.Tasks.lease.begin()
			require.NoError(t, err)
			q := d.QuoteIdentifier
			query, args, err := d.QueryBuilder().Update(q("tasks")).Set(q("StatusMetadata"), `{"partial":true}`).
				Where(sq.Eq{q("UUID"): task.UUID}).ToSql()
			require.NoError(t, err)
			_, err = mutation.Exec(query, args...)
			require.NoError(t, err)
			time.Sleep(500 * time.Millisecond)
			require.ErrorIs(t, mutation.Commit(), ErrRunLeaseConflict)
			stored, err = tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Empty(t, stored.StatusMetadata, "expiry must roll back a whole mutation transaction")
		})
	}
}
