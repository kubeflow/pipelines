// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func driverRetryWriteTask(generation interface{}) *model.Task {
	return &model.Task{
		Namespace: "ns1",
		RunUUID:   "run-1",
		Name:      "driver-task",
		ScopePath: "root.driver-task",
		Type:      model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:     model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
		StatusMetadata: model.JSONData{"customProperties": map[string]interface{}{
			util.DriverRetryGenerationKey: generation,
		}},
	}
}

func TestDriverRetryTaskWriteCurrentGeneration(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	_, err := db.Exec("UPDATE run_details SET RetryGeneration = 7 WHERE UUID = ?", "run-1")
	require.NoError(t, err)
	request := driverRetryWriteTask("7")
	created, err := tasks.CreateTask(request)
	require.NoError(t, err)
	replayed, err := tasks.CreateTask(request)
	require.NoError(t, err)
	assert.Equal(t, created.UUID, replayed.UUID)

	request.UUID = created.UUID
	request.State = model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED)
	request.FinishedInSec = 42
	updated, err := tasks.UpdateTask(request)
	require.NoError(t, err)
	assert.Equal(t, request.State, updated.State)
	assert.Equal(t, int64(42), updated.FinishedInSec)
	assert.Equal(t, request.StatusMetadata, updated.StatusMetadata)
}

func TestDriverRetryTaskWriteRejectsStaleGeneration(t *testing.T) {
	for _, operation := range []string{"create", "repeated create", "update"} {
		t.Run(operation, func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			request := driverRetryWriteTask("0")
			var before *model.Task
			if operation != "create" {
				created, err := tasks.CreateTask(request)
				require.NoError(t, err)
				request.UUID = created.UUID
				before, err = tasks.GetTask(created.UUID)
				require.NoError(t, err)
			}
			_, err := db.Exec("UPDATE run_details SET RetryGeneration = 1 WHERE UUID = ?", "run-1")
			require.NoError(t, err)
			request.State = model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
			request.FinishedInSec = 42
			if operation == "update" {
				_, err = tasks.UpdateTask(request)
			} else {
				_, err = tasks.CreateTask(request)
			}
			require.ErrorContains(t, err, "different retry generation")
			if before != nil {
				after, err := tasks.GetTask(before.UUID)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			} else {
				var count int
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
				assert.Zero(t, count)
			}
		})
	}
}

func TestDriverRetryTaskWriteRejectsFinalizedRun(t *testing.T) {
	for _, state := range []string{"SUCCEEDED", "FAILED", "CANCELED", "SKIPPED", "Error", ""} {
		t.Run(state, func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			request := driverRetryWriteTask("0")
			created, err := tasks.CreateTask(request)
			require.NoError(t, err)
			before, err := tasks.GetTask(created.UUID)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE run_details SET State = ?, Conditions = 'Failed' WHERE UUID = ?", state, "run-1")
			require.NoError(t, err)
			_, err = tasks.CreateTask(request)
			require.ErrorContains(t, err, "has finished")
			request.UUID = created.UUID
			request.State = model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED)
			_, err = tasks.UpdateTask(request)
			require.ErrorContains(t, err, "has finished")
			after, err := tasks.GetTask(created.UUID)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestDriverRetryTaskWriteMalformedGeneration(t *testing.T) {
	for _, generation := range []interface{}{"", "-1", "01", "+1", "invalid", "9223372036854775808", float64(0), true, nil} {
		t.Run(fmt.Sprintf("%v", generation), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			valid := driverRetryWriteTask("0")
			created, err := tasks.CreateTask(valid)
			require.NoError(t, err)
			request := driverRetryWriteTask(generation)
			_, err = tasks.CreateTask(request)
			require.ErrorContains(t, err, "nonnegative decimal string")
			request.UUID = created.UUID
			_, err = tasks.UpdateTask(request)
			require.ErrorContains(t, err, "nonnegative decimal string")
		})
	}
}

func TestDriverRetryTaskWriteRequiresMatchingRun(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	request := driverRetryWriteTask("0")
	created, err := tasks.CreateTask(request)
	require.NoError(t, err)
	request.UUID = created.UUID
	request.RunUUID = ""
	_, err = tasks.CreateTask(request)
	require.ErrorContains(t, err, "require a run ID")
	_, err = tasks.UpdateTask(request)
	require.ErrorContains(t, err, "require a run ID")
	request.RunUUID = "missing"
	_, err = tasks.CreateTask(request)
	require.Error(t, err)
	_, err = tasks.UpdateTask(request)
	require.Error(t, err)
	_, err = db.Exec("UPDATE run_details SET State = 'RUNNING' WHERE UUID = ?", "run-2")
	require.NoError(t, err)
	request.RunUUID = "run-2"
	_, err = tasks.UpdateTask(request)
	require.ErrorContains(t, err, "does not match the stored task")
	after, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	assert.Equal(t, "run-1", after.RunUUID)
}

func TestDriverRetryTaskWriteClearsPreviousEndTime(t *testing.T) {
	for _, tagged := range []bool{true, false} {
		t.Run(fmt.Sprint(tagged), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			request := driverRetryWriteTask("0")
			request.State = model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
			request.FinishedInSec = 42
			if !tagged {
				// Existing untagged behavior permits writes after a run finishes.
				request.RunUUID = "run-2"
				request.StatusMetadata = model.JSONData{"customProperties": map[string]interface{}{"unrelated": "value"}}
			}
			created, err := tasks.CreateTask(request)
			require.NoError(t, err)
			request.UUID = created.UUID
			request.State = model.TaskStatus(apiv2beta1.PipelineTask_RUNNING)
			request.FinishedInSec = 0
			updated, err := tasks.UpdateTask(request)
			require.NoError(t, err)
			assert.Equal(t, request.State, updated.State)
			if tagged {
				assert.Zero(t, updated.FinishedInSec)
			} else {
				assert.Equal(t, int64(42), updated.FinishedInSec)
			}
		})
	}
}

func TestDriverRetryTaskWriteRollsBack(t *testing.T) {
	for _, operation := range []string{"INSERT", "UPDATE"} {
		t.Run(operation, func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			request := driverRetryWriteTask("0")
			var before *model.Task
			if operation == "UPDATE" {
				created, err := tasks.CreateTask(request)
				require.NoError(t, err)
				request.UUID = created.UUID
				before, err = tasks.GetTask(created.UUID)
				require.NoError(t, err)
			}
			runBefore, err := runs.GetRun("run-1", false)
			require.NoError(t, err)
			// SQLite FAIL preserves earlier trigger writes unless the surrounding
			// transaction rolls back, proving task writes use the fenced transaction.
			_, err = db.Exec(fmt.Sprintf(`CREATE TRIGGER reject_driver_write BEFORE %s ON tasks
				BEGIN UPDATE run_details SET Conditions = 'partial write' WHERE UUID = 'run-1';
				SELECT RAISE(FAIL, 'injected driver write failure'); END`, operation))
			require.NoError(t, err)
			if operation == "UPDATE" {
				request.State = model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED)
				_, err = tasks.UpdateTask(request)
			} else {
				_, err = tasks.CreateTask(request)
			}
			require.ErrorContains(t, err, "injected driver write failure")
			runAfter, err := runs.GetRun("run-1", false)
			require.NoError(t, err)
			assert.Equal(t, runBefore, runAfter)
			if before != nil {
				after, err := tasks.GetTask(before.UUID)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			} else {
				var count int
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
				assert.Zero(t, count)
			}
			_, err = db.Exec("DROP TRIGGER reject_driver_write")
			require.NoError(t, err)
			if operation == "UPDATE" {
				_, err = tasks.UpdateTask(request)
			} else {
				_, err = tasks.CreateTask(request)
			}
			require.NoError(t, err, "rollback must release the run lock")
		})
	}
}
