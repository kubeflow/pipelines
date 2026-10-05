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
	"strings"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type logicalTaskInsertRace struct {
	*sql.Tx
	initialLookup bool
	insertSQL     string
	lockedRead    bool
}

func (r *logicalTaskInsertRace) QueryRow(query string, args ...any) *sql.Row {
	if !r.initialLookup {
		r.initialLookup = true
		// The first create looked up its identity before a concurrent writer
		// inserted the canonical row.
		query = strings.Replace(query, " WHERE ", " WHERE 1 = 0 AND ", 1)
	}
	if strings.HasSuffix(query, " FOR UPDATE") {
		r.lockedRead = true
		query = strings.TrimSuffix(query, " FOR UPDATE")
	}
	return r.Tx.QueryRow(query, args...)
}

func (r *logicalTaskInsertRace) Exec(query string, args ...any) (sql.Result, error) {
	r.insertSQL = query
	canonicalArgs := append([]any(nil), args...)
	for index, arg := range canonicalArgs {
		if arg == testUUID2 {
			canonicalArgs[index] = testUUID1
		}
	}
	if _, err := r.Tx.Exec(query, canonicalArgs...); err != nil {
		return nil, err
	}
	return r.Tx.Exec(query, args...)
}

func TestCreateTaskLogicalConflictDoesNotAbortTransaction(t *testing.T) {
	// PostgreSQL and SQLite share the ON CONFLICT syntax. Execute the exact
	// PostgreSQL INSERT against SQLite rather than mocking a successful insert.
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	tasks.dbDialect = dialect.NewDBDialect("pgx")
	tasks.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	race := &logicalTaskInsertRace{Tx: tx}
	created, err := tasks.createTaskWithExecutor(race, attemptedDriverTask("0", "0"), true)
	require.NoError(t, err)
	assert.Equal(t, testUUID1, created.UUID)
	assert.Contains(t, race.insertSQL, `ON CONFLICT ("LogicalKey") DO NOTHING`)
	assert.True(t, race.lockedRead, "canonical lookup must see committed rows under MySQL REPEATABLE READ")
	// The conflict must leave the surrounding claim transaction usable.
	_, err = tx.Exec("UPDATE run_details SET Conditions = 'still usable' WHERE UUID = ?", "run-1")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
	assert.Equal(t, 1, count)
}

type driverLockRecorder struct {
	dialect.DBDialect
	shared    []string
	exclusive []string
}

func (d *driverLockRecorder) SelectForShare(query string) string {
	d.shared = append(d.shared, query)
	return d.DBDialect.SelectForShare(query)
}

func (d *driverLockRecorder) SelectForUpdate(query string) string {
	d.exclusive = append(d.exclusive, query)
	return d.DBDialect.SelectForUpdate(query)
}

func TestDriverTaskWritesShareRunLockAndLockOnlyTheirTasks(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	tasks.uuid = util.NewUUIDGenerator()
	parentRequest := attemptedDriverTask("0", "4")
	parentRequest.Name, parentRequest.ScopePath = "parent", "root.parent"
	parent, err := tasks.CreateTask(parentRequest)
	require.NoError(t, err)
	child, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	recorder := &driverLockRecorder{DBDialect: tasks.dbDialect}
	tasks.dbDialect = recorder
	for _, source := range []bool{false, true} {
		t.Run(map[bool]string{false: "own", true: "dependent"}[source], func(t *testing.T) {
			recorder.shared, recorder.exclusive = nil, nil
			request, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			if source {
				request, err = tasks.GetTask(parent.UUID)
				require.NoError(t, err)
				driverTaskProperties(request)[util.DriverRetrySourceTaskKey] = child.UUID
				driverTaskProperties(request)[util.DriverRetrySourceAttemptKey] = "1"
			}
			_, err = tasks.UpdateTask(request)
			require.NoError(t, err)
			require.Len(t, recorder.shared, 1)
			assert.Contains(t, recorder.shared[0], "run_details")
			expectedTasks := 1
			if source {
				expectedTasks = 2
			}
			require.Len(t, recorder.exclusive, expectedTasks)
			for _, query := range recorder.exclusive {
				assert.NotContains(t, query, "run_details")
				assert.Contains(t, query, `"tasks"."UUID" = ?`)
			}
		})
	}
	recorder.shared, recorder.exclusive = nil, nil
	_, err = tasks.CreateTask(attemptedDriverTask("0", "2"))
	require.NoError(t, err)
	require.Len(t, recorder.shared, 1)
	require.Len(t, recorder.exclusive, 1, "existing task must be locked before advancing its attempt")
	assert.Contains(t, recorder.exclusive[0], `"tasks"."UUID" = ?`)
}

func TestDriverFinalizationSkipsUnrelatedTaskLocksAndPayloads(t *testing.T) {
	for _, retrying := range []bool{false, true} {
		t.Run(map[bool]string{false: "untagged", true: "mixed"}[retrying], func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
			unrelated := createDriverRetryFinalizationTask(t, tasks, "unrelated", nil, api.PipelineTask_RUNNING, nil)
			// Full hydration would fail; finalization must never read this
			// unrelated task's payload, nor exclusively lock its row.
			_, err := db.Exec("UPDATE tasks SET InputParameters = ? WHERE UUID = ?", "not JSON", unrelated.UUID)
			require.NoError(t, err)
			if retrying {
				createDriverRetryFinalizationTask(t, tasks, "child", parent, api.PipelineTask_RUNNING, util.StringPointer("0"))
			}
			recorder := &driverLockRecorder{DBDialect: runs.dbDialect}
			runs.dbDialect = recorder
			reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
			var taskQueries []string
			for _, query := range recorder.exclusive {
				if strings.Contains(query, `FROM "tasks"`) {
					taskQueries = append(taskQueries, query)
					assert.Contains(t, query, `"UUID" IN (`)
				}
			}
			if retrying {
				assert.Len(t, taskQueries, 2, "lock the candidate and its ancestor only")
				after, err := tasks.GetTask(parent.UUID)
				require.NoError(t, err)
				assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), after.State)
			} else {
				assert.Empty(t, taskQueries, "untagged runs must not acquire task row locks")
			}
		})
	}
}

func TestDriverTaskUpdatePreservesOmittedRecoveryPayload(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	task, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.NoError(t, err)
	driverTaskProperties(task)[util.DriverCheckpointKey] = "saved handoff"
	driverTaskProperties(task)[util.DriverCachedOutputsKey] = "frozen decision"
	task, err = tasks.UpdateTask(task)
	require.NoError(t, err)
	delete(driverTaskProperties(task), util.DriverCheckpointKey)
	delete(driverTaskProperties(task), util.DriverCachedOutputsKey)
	task.State = model.TaskStatus(api.PipelineTask_CACHED)
	updated, err := tasks.UpdateTask(task)
	require.NoError(t, err)
	assert.Equal(t, "saved handoff", driverTaskProperties(updated)[util.DriverCheckpointKey])
	assert.Equal(t, "frozen decision", driverTaskProperties(updated)[util.DriverCachedOutputsKey])
	driverTaskProperties(task)[util.DriverCheckpointKey] = "updated handoff"
	updated, err = tasks.UpdateTask(task)
	require.NoError(t, err)
	assert.Equal(t, "updated handoff", driverTaskProperties(updated)[util.DriverCheckpointKey])
	assert.Equal(t, "frozen decision", driverTaskProperties(updated)[util.DriverCachedOutputsKey])
}
