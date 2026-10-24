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

package clientmanager

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteRuntimeConnectionOptions(t *testing.T) {
	dsn, err := sqliteRuntimeDSN("file:runtime.db?cache=shared&_journal_mode=WAL&_busy_timeout=30000")
	require.NoError(t, err)
	_, query, _ := strings.Cut(dsn, "?")
	options, err := url.ParseQuery(query)
	require.NoError(t, err)
	assert.Equal(t, "immediate", options.Get("_txlock"))
	assert.Equal(t, "private", options.Get("cache"))
	assert.Equal(t, "on", options.Get("_foreign_keys"))
	assert.Equal(t, "30000", options.Get("_busy_timeout"))
	assert.Equal(t, "WAL", options.Get("_journal_mode"))
	memory, err := sqliteRuntimeDSN("file::memory:?cache=shared")
	require.NoError(t, err)
	assert.Contains(t, memory, "cache=shared", "named in-memory databases must retain their shared identity")
	for _, query := range []string{"_txlock=deferred", "_busy_timeout=0", "_busy_timeout=30001", "_foreign_keys=off"} {
		_, err := sqliteRuntimeDSN("file:runtime.db?" + query)
		require.Error(t, err, query)
	}
}

func sqliteRuntimeStartup(t *testing.T) (*sql.DB, dialect.DBDialect) {
	t.Helper()
	path := fmt.Sprintf(".runtime-sqlite-%d.db", time.Now().UnixNano())
	t.Cleanup(func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			err := os.Remove(path + suffix)
			if !os.IsNotExist(err) {
				require.NoError(t, err)
			}
		}
	})
	oldDriver, oldDSN := viper.GetString("DBDriverName"), viper.GetString(sqliteDSN)
	t.Cleanup(func() {
		viper.Set("DBDriverName", oldDriver)
		viper.Set(sqliteDSN, oldDSN)
	})
	viper.Set("DBDriverName", "sqlite")
	viper.Set(sqliteDSN, "file:"+path+"?cache=shared&_journal_mode=WAL&_busy_timeout=30000")
	db, d, _ := InitDBClient(0)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	var mode string
	require.NoError(t, db.QueryRow("PRAGMA journal_mode").Scan(&mode))
	assert.Equal(t, "wal", mode)
	var foreignKeys int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	assert.Equal(t, 1, foreignKeys)
	return db, d
}

func TestSQLiteRuntimeLocksBeforeReadingSnapshot(t *testing.T) {
	db, _ := sqliteRuntimeStartup(t)
	first, err := db.Begin()
	require.NoError(t, err)
	defer first.Rollback()
	var count int
	require.NoError(t, first.QueryRow("SELECT COUNT(*) FROM run_details").Scan(&count))
	begun, done := make(chan struct{}), make(chan error, 1)
	go func() {
		second, err := db.Begin()
		if err != nil {
			done <- err
			return
		}
		defer second.Rollback()
		close(begun)
		_, err = second.Exec("UPDATE run_details SET FinishedAtInSec = 1")
		if err == nil {
			err = second.Commit()
		}
		done <- err
	}()
	select {
	case <-begun:
		require.NoError(t, first.Rollback())
		require.NoError(t, <-done)
		t.Fatal("second writer began before the first committed; deferred transactions permit WAL snapshot-upgrade deadlocks")
	case err := <-done:
		require.NoError(t, err)
		t.Fatal("second writer did not wait for the current writer")
	case <-time.After(50 * time.Millisecond):
	}
	_, err = first.Exec("UPDATE run_details SET FinishedAtInSec = 1")
	require.NoError(t, err)
	require.NoError(t, first.Commit())
	require.NoError(t, <-done, "the busy handler must wait at BEGIN, before any read snapshot is established")
}

func TestSQLiteDeferredTransactionsReproduceSnapshotUpgradeFailure(t *testing.T) {
	path := fmt.Sprintf(".runtime-sqlite-deferred-%d.db", time.Now().UnixNano())
	t.Cleanup(func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			err := os.Remove(path + suffix)
			if !os.IsNotExist(err) {
				require.NoError(t, err)
			}
		}
	})
	db, err := sql.Open("sqlite3", "file:"+path+"?_journal_mode=WAL&_busy_timeout=30000&_txlock=deferred")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(2)
	_, err = db.Exec("CREATE TABLE counter (id INTEGER PRIMARY KEY, value INTEGER NOT NULL)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO counter VALUES (1, 0)")
	require.NoError(t, err)
	first, err := db.Begin()
	require.NoError(t, err)
	defer first.Rollback()
	var value int
	require.NoError(t, first.QueryRow("SELECT value FROM counter WHERE id = 1").Scan(&value))
	second, err := db.Begin()
	require.NoError(t, err)
	defer second.Rollback()
	_, err = second.Exec("UPDATE counter SET value = 1 WHERE id = 1")
	require.NoError(t, err)
	require.NoError(t, second.Commit())
	start := time.Now()
	_, err = first.Exec("UPDATE counter SET value = 2 WHERE id = 1")
	var sqliteError sqlite3.Error
	require.ErrorAs(t, err, &sqliteError)
	assert.Equal(t, sqlite3.ErrBusySnapshot, sqliteError.ExtendedCode)
	assert.Less(t, time.Since(start), time.Second, "busy_timeout cannot repair an obsolete WAL read snapshot")
}

func TestSQLiteRuntimeConcurrentStoreTransactionsAndAPIDeletion(t *testing.T) {
	db, d := sqliteRuntimeStartup(t)
	clock, ids := util.NewRealTime(), util.NewUUIDGenerator()
	runs := storage.NewRunStore(db, clock, d)
	tasks := storage.NewTaskStore(db, clock, ids, d)
	runtime := tasks.RuntimeStore()
	const writers = 12
	start, results := make(chan struct{}), make(chan error, writers)
	for i := range writers {
		go func() {
			<-start
			results <- func() error {
				runID := fmt.Sprintf("concurrent-run-%d", i)
				_, err := runs.CreateRun(&model.Run{UUID: runID, RunDetails: model.RunDetails{State: model.RuntimeStateRunning}})
				if err != nil {
					return err
				}
				request := &model.Task{
					RunUUID: runID, Name: "leaf", ScopePath: "root", Type: model.TaskType(apiv2.PipelineTask_RUNTIME),
					State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
				}
				task, err := tasks.CreateTask(request)
				if err != nil {
					return err
				}
				attemptID := fmt.Sprintf("attempt-%d", i)
				attempt := &model.RuntimeTask{
					TaskUUID: task.UUID, RunUUID: runID, AttemptUUID: &attemptID, State: model.RuntimeTaskRunning,
					ExecutionRequest: model.JSONData{"version": 1},
				}
				if err := runtime.Create(attempt); err != nil {
					return err
				}
				attempt.ExecutionReference = model.JSONData{"name": attemptID}
				if err := runtime.Update(attempt, attemptID); err != nil {
					return err
				}
				outputs := model.JSONSlice{map[string]any{"parameterKey": "answer", "type": "OUTPUT", "value": float64(i)}}
				update := &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), OutputParameters: outputs}
				if _, err := tasks.UpdateTask(update); err != nil {
					return err
				}
				update.State = model.TaskStatus(apiv2.PipelineTask_RUNNING)
				attempt.State = model.RuntimeTaskPublishing
				if err := runtime.Checkpoint(attempt, update); err != nil {
					return err
				}
				if i%2 == 0 {
					if err := runs.DeleteRun(runID); err != nil {
						return err
					}
					update.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
					if err := runtime.Finalize(task.UUID, attemptID, update); !errors.Is(err, storage.ErrRuntimeTaskConflict) {
						return fmt.Errorf("deleted attempt must stay deleted, got %v", err)
					}
					return nil
				}
				update.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
				if err := runtime.Finalize(task.UUID, attemptID, update); err != nil {
					return err
				}
				existing, err := tasks.CreateTask(request)
				if err != nil {
					return err
				}
				if existing.UUID != task.UUID || existing.State != model.TaskStatus(apiv2.PipelineTask_SUCCEEDED) || len(existing.OutputParameters) == 0 {
					return errors.New("repeated materialization reset a terminal task")
				}
				if err := runtime.FinalizeRun(runID, model.RuntimeStateSucceeded, clock.Now().Unix()); err != nil {
					return err
				}
				return runs.DeleteRun(runID)
			}()
		}()
	}
	close(start)
	for range writers {
		require.NoError(t, <-results, "concurrent API and worker transactions must wait rather than fail on lock upgrades")
	}
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM runtime_tasks").Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM run_details").Scan(&count))
	assert.Zero(t, count)
}
