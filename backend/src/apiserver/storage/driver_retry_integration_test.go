// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

// TestDriverRetryProductionDatabases reuses the disposable-schema helpers and
// KFP_RECURRING_{MYSQL,POSTGRES}_TEST_DSN opt-in from recurring-run tests.
func TestDriverRetryProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			if driver == "mysql" {
				if dsn := os.Getenv("KFP_RECURRING_MYSQL_TEST_DSN"); dsn != "" {
					config, err := mysqldriver.ParseDSN(dsn)
					require.NoError(t, err)
					// Match the production client manager: no-op UPDATEs count
					// matched rows instead of falsely appearing to be missing.
					config.ClientFoundRows = true
					t.Setenv("KFP_RECURRING_MYSQL_TEST_DSN", config.FormatDSN())
				}
			}
			dbs, d := recurringIntegrationDatabases(t, driver)
			tasks := make([]*TaskStore, len(dbs))
			runs := make([]*RunStore, len(dbs))
			for i, db := range dbs {
				tasks[i] = NewTaskStore(db, util.NewRealTime(), util.NewUUIDGenerator(), d)
				runs[i] = NewRunStore(db, util.NewRealTime(), d)
			}
			createRun := func(id string) {
				_, err := runs[0].CreateRun(&model.Run{UUID: id, DisplayName: id, Namespace: "ns1", StorageState: model.StorageStateAvailable,
					RunDetails: model.RunDetails{State: model.RuntimeStateRunning}})
				require.NoError(t, err)
			}
			request := func(runID, name string, attempt int) *model.Task {
				task := attemptedDriverTask("0", strconv.Itoa(attempt))
				task.RunUUID, task.Name, task.ScopePath = runID, name, "root."+name
				return task
			}
			sessionID := func(index int) int64 {
				query := "SELECT CONNECTION_ID()"
				if driver == "pgx" {
					query = "SELECT pg_backend_pid()"
				}
				var id int64
				require.NoError(t, dbs[index].QueryRow(query).Scan(&id))
				return id
			}

			t.Run("concurrent-claims", func(t *testing.T) {
				createRun("claims")
				created := make([]*model.Task, len(tasks))
				errs := recurringConcurrent(len(tasks), func(i int) error {
					var err error
					created[i], err = tasks[i].CreateTask(request("claims", "task", 0))
					return err
				})
				for i, err := range errs {
					require.NoError(t, err)
					require.Equal(t, created[0].UUID, created[i].UUID)
				}
				errs = recurringConcurrent(len(tasks), func(i int) error {
					_, err := tasks[i].CreateTask(request("claims", "task", i+1))
					return err
				})
				for _, err := range errs {
					if err != nil {
						require.ErrorContains(t, err, "different driver retry attempt")
					}
				}
				current, err := tasks[0].GetTask(created[0].UUID)
				require.NoError(t, err)
				require.Equal(t, int64(len(tasks)), *current.DriverRetryAttempt)
				count, err := tasks[0].GetTaskCountForRun("claims")
				require.NoError(t, err)
				require.Equal(t, 1, count)
			})

			t.Run("insert-conflict-keeps-claim-transaction-usable", func(t *testing.T) {
				createRun("insert-conflict")
				require.NoError(t, tasks[0].ensureDriverRetryPresence("insert-conflict"))
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				require.NoError(t, tasks[0].lockRunForDriverTaskWrite(tx, "insert-conflict", 0))
				gate := &driverInsertGate{Tx: tx, reached: make(chan struct{}), proceed: make(chan struct{})}
				result := make(chan error, 1)
				var recovered *model.Task
				go func() {
					var err error
					recovered, err = tasks[0].createTaskWithExecutor(gate, request("insert-conflict", "task", 0), true)
					if err == nil {
						_, err = tasks[0].claimDriverTaskAttempt(tx, recovered, request("insert-conflict", "task", 0))
					}
					if err == nil {
						err = tx.Commit()
					}
					result <- err
				}()
				select {
				case <-gate.reached:
				case err := <-result:
					t.Fatalf("create stopped before the insert race: %v", err)
				case <-time.After(10 * time.Second):
					t.Fatal("create did not reach the insert race")
				}
				canonical, err := tasks[1].CreateTask(request("insert-conflict", "task", 0))
				close(gate.proceed)
				require.NoError(t, err)
				require.NoError(t, waitDriverDatabaseResult(t, result))
				require.Equal(t, canonical.UUID, recovered.UUID)
			})

			t.Run("stopped-identity-serializes-with-delayed-create", func(t *testing.T) {
				createRun("stop-race")
				require.NoError(t, tasks[0].ensureDriverRetryPresence("stop-race"))
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				require.NoError(t, tasks[0].lockRunForDriverTaskWrite(tx, "stop-race", 0))
				pending := request("stop-race", "delayed", 0)
				require.NoError(t, tasks[0].checkDriverTaskStop(tx, pending))
				stopSession := sessionID(1)
				result := make(chan error, 1)
				go func() { result <- tasks[1].FinalizeStoppedDriver("stop-race", 0, "delayed", "", nil) }()
				recurringWaitForLock(t, dbs[2], driver, stopSession)
				created, err := tasks[0].createTaskWithExecutor(tx, pending, true)
				require.NoError(t, err)
				require.NoError(t, tx.Commit())
				require.NoError(t, waitDriverDatabaseResult(t, result))
				stopped, err := tasks[0].GetTask(created.UUID)
				require.NoError(t, err)
				require.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), stopped.State)
				require.NoError(t, tasks[0].FinalizeStoppedDriver("stop-race", 0, "missing", "", nil))
				_, err = tasks[0].CreateTask(request("stop-race", "missing", 0))
				require.ErrorContains(t, err, "different driver retry attempt")
				_, err = tasks[0].CreateTask(request("stop-race", "sibling", 0))
				require.NoError(t, err)
			})

			t.Run("blocked-task-does-not-block-independent-task", func(t *testing.T) {
				createRun("parallel-writes")
				first, err := tasks[0].CreateTask(request("parallel-writes", "first", 0))
				require.NoError(t, err)
				second, err := tasks[0].CreateTask(request("parallel-writes", "second", 0))
				require.NoError(t, err)
				blockedSession := sessionID(1)
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				require.NoError(t, tasks[0].lockRunForDriverTaskWrite(tx, first.RunUUID, 0))
				_, err = tasks[0].getTaskForUpdate(tx, first.UUID)
				require.NoError(t, err)
				blocked := make(chan error, 1)
				go func() { _, err := tasks[1].UpdateTask(driverWrite(first, 0, retryInt(0))); blocked <- err }()
				recurringWaitForLock(t, dbs[2], driver, blockedSession)
				independent := make(chan error, 1)
				go func() { _, err := tasks[3].UpdateTask(driverWrite(second, 0, retryInt(0))); independent <- err }()
				require.NoError(t, waitDriverDatabaseResult(t, independent))
				require.NoError(t, tx.Commit())
				require.NoError(t, waitDriverDatabaseResult(t, blocked))
			})

			t.Run("queued-source-write-rejects-newer-claim", func(t *testing.T) {
				createRun("source-race")
				parent, err := tasks[0].CreateTask(request("source-race", "parent", 4))
				require.NoError(t, err)
				child, err := tasks[0].CreateTask(request("source-race", "child", 0))
				require.NoError(t, err)
				queued := *parent
				queued.DriverWriteAuthority = &model.DriverTaskAuthority{Generation: 0, SourceTaskID: child.UUID, SourceAttempt: retryInt(0)}
				queued.State = model.TaskStatus(api.PipelineTask_CACHED)
				blockedSession := sessionID(1)
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				require.NoError(t, tasks[0].lockRunForDriverTaskWrite(tx, child.RunUUID, 0))
				locked, err := tasks[0].getTaskForUpdate(tx, child.UUID)
				require.NoError(t, err)
				result := make(chan error, 1)
				go func() { _, err := tasks[1].UpdateTask(&queued); result <- err }()
				recurringWaitForLock(t, dbs[2], driver, blockedSession)
				_, err = tasks[0].claimDriverTaskAttempt(tx, locked, request("source-race", "child", 1))
				require.NoError(t, err)
				require.NoError(t, tx.Commit())
				require.ErrorContains(t, waitDriverDatabaseResult(t, result), "different driver retry attempt")
				after, err := tasks[0].GetTask(parent.UUID)
				require.NoError(t, err)
				require.Equal(t, parent.State, after.State)
			})

			t.Run("terminal-report-waits-for-task-write", func(t *testing.T) {
				createRun("terminal")
				task, err := tasks[0].CreateTask(request("terminal", "task", 0))
				require.NoError(t, err)
				run, err := runs[0].GetRun("terminal", false)
				require.NoError(t, err)
				run.State, run.FinishedAtInSec = model.RuntimeStateFailed, 123
				blockedSession := sessionID(1)
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				require.NoError(t, tasks[0].lockRunForDriverTaskWrite(tx, task.RunUUID, 0))
				finished := make(chan error, 1)
				go func() {
					_, err := runs[1].UpdateRunIfRuntimeManifestsUnchanged(run, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest)
					finished <- err
				}()
				recurringWaitForLock(t, dbs[2], driver, blockedSession)
				require.NoError(t, tx.Commit())
				require.NoError(t, waitDriverDatabaseResult(t, finished))
				_, err = tasks[0].UpdateTask(driverWrite(task, 0, retryInt(0)))
				require.ErrorContains(t, err, "has finished")
				final, err := tasks[0].GetTask(task.UUID)
				require.NoError(t, err)
				require.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), final.State)

				// Manual retry's exclusive run lock also waits for shared readers.
				tx, err = dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				q := d.QuoteIdentifier
				query, args, err := d.QueryBuilder().Select(q("UUID")).From(q("run_details")).Where(map[string]interface{}{q("UUID"): "terminal"}).ToSql()
				require.NoError(t, err)
				var id string
				require.NoError(t, tx.QueryRow(d.SelectForShare(query), args...).Scan(&id))
				claimed := make(chan error, 1)
				go func() { _, _, _, _, err := runs[1].ClaimRunForRetry("terminal", false); claimed <- err }()
				recurringWaitForLock(t, dbs[2], driver, blockedSession)
				require.NoError(t, tx.Commit())
				require.NoError(t, waitDriverDatabaseResult(t, claimed))
				_, err = tasks[0].UpdateTask(driverWrite(task, 0, retryInt(0)))
				require.ErrorContains(t, err, "different retry generation")
			})

			t.Run("finalization-locks-only-retry-closure", func(t *testing.T) {
				for _, tagged := range []bool{false, true} {
					id := fmt.Sprintf("closure-%t", tagged)
					createRun(id)
					parentRequest := request(id, "parent", 0)
					parentRequest.DriverClaim = false
					parentRequest.DriverRetryGeneration = nil
					parentRequest.DriverRetryAttempt = nil
					parentRequest.DriverWriteAuthority.SourceAttempt = nil
					parent, err := tasks[0].CreateTask(parentRequest)
					require.NoError(t, err)
					unrelatedRequest := request(id, "unrelated", 0)
					unrelatedRequest.DriverClaim = false
					unrelatedRequest.DriverRetryGeneration = nil
					unrelatedRequest.DriverRetryAttempt = nil
					unrelatedRequest.DriverWriteAuthority.SourceAttempt = nil
					_, err = tasks[0].CreateTask(unrelatedRequest)
					require.NoError(t, err)
					if tagged {
						childRequest := request(id, "child", 0)
						childRequest.ParentTaskUUID = &parent.UUID
						_, err = tasks[0].CreateTask(childRequest)
						require.NoError(t, err)
					}
					recorder := &driverLockRecorder{DBDialect: d}
					store := NewRunStore(dbs[0], util.NewRealTime(), recorder)
					run, err := store.GetRun(id, false)
					require.NoError(t, err)
					run.State = model.RuntimeStateFailed
					updated, err := store.UpdateRunIfRuntimeManifestsUnchanged(run, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest)
					require.NoError(t, err)
					require.True(t, updated)
					lockedTasks := 0
					for _, query := range recorder.exclusive {
						if strings.Contains(query, "FROM "+d.QuoteIdentifier("tasks")) {
							lockedTasks++
							require.Contains(t, query, d.QuoteIdentifier("UUID")+" IN (")
						}
					}
					if tagged {
						require.Equal(t, 2, lockedTasks)
					} else {
						require.Zero(t, lockedTasks)
					}
				}
			})
		})
	}
}

type driverInsertGate struct {
	*sql.Tx
	reached, proceed chan struct{}
}

func (g *driverInsertGate) Exec(query string, args ...any) (sql.Result, error) {
	close(g.reached)
	<-g.proceed
	return g.Tx.Exec(query, args...)
}

func waitDriverDatabaseResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("database operation did not complete while unrelated task remained locked")
		return nil
	}
}
