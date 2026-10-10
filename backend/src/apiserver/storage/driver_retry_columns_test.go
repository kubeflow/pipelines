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
	"strings"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverRetryResetRemainsDiscoverableBeforeRestart(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
	child := createDriverRetryFinalizationTask(t, tasks, "child", parent, api.PipelineTask_RUNNING, util.StringPointer("0"))
	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	_, _, _, generation, err := runs.ClaimRunForRetry("run-1", false)
	require.NoError(t, err)
	require.Equal(t, int64(1), generation)
	require.NoError(t, tasks.ResetTasksForRetry("run-1", generation, []string{parent.UUID, child.UUID}))
	reset, err := tasks.GetTask(child.UUID)
	require.NoError(t, err)
	require.Equal(t, generation, *reset.DriverRetryGeneration)
	require.Nil(t, reset.DriverRetryAttempt)
	require.Nil(t, reset.DriverCheckpoint)
	require.Nil(t, reset.StatusMetadata)
	// Workflow fails before either reset task starts a pod.
	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	for _, id := range []string{parent.UUID, child.UUID} {
		task, err := tasks.GetTask(id)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), task.State)
	}
	run, err := runs.GetRun("run-1", false)
	require.NoError(t, err)
	require.Equal(t, generation, *run.DriverRetryFinalizedGeneration)
	require.ErrorContains(t, tasks.ResetTasksForRetry("run-1", generation, []string{child.UUID}), "has finished")
}

func TestDriverRetryResyncDoesNotReloadTasks(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	task := createDriverRetryFinalizationTask(t, tasks, "child", nil, api.PipelineTask_RUNNING, util.StringPointer("0"))
	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	// If resync even hydrates the retry candidate, it now fails to decode.
	_, err := db.Exec("UPDATE tasks SET InputParameters='invalid JSON' WHERE UUID=?", task.UUID)
	require.NoError(t, err)
	recorder := &driverLockRecorder{DBDialect: runs.dbDialect}
	runs.dbDialect = recorder
	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	for _, query := range recorder.exclusive {
		assert.NotContains(t, query, `FROM "tasks"`)
	}
	var indexName string
	require.NoError(t, db.QueryRow(`SELECT name FROM pragma_index_list('tasks') WHERE name='idx_task_driver_retry'`).Scan(&indexName))
	rows, err := db.Query(`SELECT name FROM pragma_index_info('idx_task_driver_retry') ORDER BY seqno`)
	require.NoError(t, err)
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		require.NoError(t, rows.Scan(&column))
		columns = append(columns, column)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"RunUUID", "DriverRetryGeneration", "UUID"}, columns)
}

func TestDriverRecoveryPayloadExcludedFromBulkReads(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	request := attemptedDriverTask("0", "0")
	request.DriverRecoveryUpdate = true
	request.DriverCheckpoint = util.StringPointer(`{"payload":"` + strings.Repeat("x", 10000) + `"}`)
	request.DriverCachedOutputs = util.StringPointer(`{"cache":true}`)
	task, err := tasks.CreateTask(request)
	require.NoError(t, err)
	require.NotNil(t, task.DriverCheckpoint)
	single, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	require.Equal(t, request.DriverCheckpoint, single.DriverCheckpoint)
	bulk, err := tasks.GetTasksByIDs([]string{task.UUID})
	require.NoError(t, err)
	assert.Nil(t, bulk[task.UUID].DriverCheckpoint)
	opts, err := list.NewOptions(&model.Task{}, 10, "", nil)
	require.NoError(t, err)
	page, _, _, err := tasks.ListTasks(&model.FilterContext{}, opts)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Nil(t, page[0].DriverCachedOutputs)
	run, err := runs.GetRun("run-1", true)
	require.NoError(t, err)
	require.Len(t, run.Tasks, 1)
	assert.Nil(t, run.Tasks[0].DriverCheckpoint)
}

func TestFinalizeStoppedDriverFencesLateWritesAndPreservesCompletion(t *testing.T) {
	for _, state := range []api.PipelineTask_TaskState{api.PipelineTask_RUNNING, api.PipelineTask_SUCCEEDED, api.PipelineTask_CACHED, api.PipelineTask_SKIPPED} {
		t.Run(state.String(), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
			child := createDriverRetryFinalizationTask(t, tasks, "child", parent, state, util.StringPointer("0"))
			late := driverWrite(child, 0, retryInt(0))
			late.State = model.TaskStatus(api.PipelineTask_RUNNING)
			require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "child", parent.UUID, nil))
			stopped, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			parentAfter, err := tasks.GetTask(parent.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), parentAfter.State, "a committed cache/skip result does not leave unfinished ancestors running after handoff failure")
			require.NotNil(t, stopped.DriverStoppedGeneration)
			assert.Equal(t, int64(0), *stopped.DriverStoppedGeneration)
			if state == api.PipelineTask_RUNNING {
				assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), stopped.State)
			} else {
				assert.Equal(t, model.TaskStatus(state), stopped.State)
			}
			_, err = tasks.UpdateTask(late)
			require.ErrorContains(t, err, "different driver retry attempt")
			claim := attemptedDriverTask("0", "1")
			claim.Name = "child"
			claim.ScopePath = "root.child"
			claim.ParentTaskUUID = &parent.UUID
			_, err = tasks.CreateTask(claim)
			require.ErrorContains(t, err, "different driver retry attempt")
			require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "child", parent.UUID, nil))
			again, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, stopped, again)
			if state == api.PipelineTask_RUNNING {
				propagated := driverWrite(parent, 0, retryInt(0))
				propagated.DriverWriteAuthority.SourceTaskID = child.UUID
				propagated.State = model.TaskStatus(api.PipelineTask_SUCCEEDED)
				_, err = tasks.UpdateTask(propagated)
				require.ErrorContains(t, err, "different driver retry attempt")
			}
			_, err = db.Exec("UPDATE run_details SET RetryGeneration=1 WHERE UUID=?", "run-1")
			require.NoError(t, err)
			if state == api.PipelineTask_RUNNING {
				require.NoError(t, tasks.ResetTasksForRetry("run-1", 1, []string{parent.UUID, child.UUID}))
			}
			claim.DriverRetryGeneration = retryInt(1)
			claim.DriverRetryAttempt = retryInt(0)
			claim.DriverWriteAuthority = &model.DriverTaskAuthority{Generation: 1, SourceAttempt: retryInt(0)}
			reopened, err := tasks.CreateTask(claim)
			require.NoError(t, err)
			assert.Nil(t, reopened.DriverStoppedGeneration)
			require.ErrorContains(t, tasks.FinalizeStoppedDriver("run-1", 0, "child", parent.UUID, nil), "different retry generation")
		})
	}
}

func TestFinalizeStoppedDriverScopesIterationAndMissingTask(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	tasks.uuid = util.NewUUIDGenerator()
	for i := int64(0); i < 2; i++ {
		request := attemptedDriverTask("0", "0")
		request.TypeAttrs = model.JSONData{"iterationIndex": i}
		_, err := tasks.CreateTask(request)
		require.NoError(t, err)
	}
	require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "missing", "", nil))
	require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "driver-task", "", retryInt(1)))
	for i := int64(0); i < 2; i++ {
		request := attemptedDriverTask("0", "0")
		request.TypeAttrs = model.JSONData{"iterationIndex": i}
		task, err := tasks.FindTaskByLogicalIdentity(request)
		require.NoError(t, err)
		if i == 1 {
			assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), task.State)
		} else {
			assert.Equal(t, model.TaskStatus(api.PipelineTask_RUNNING), task.State)
		}
	}
}

func TestFinalizeStoppedDriverAllowsSuccessfulSiblingPropagation(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
	_ = createDriverRetryFinalizationTask(t, tasks, "stopped", parent, api.PipelineTask_RUNNING, util.StringPointer("0"))
	sibling := createDriverRetryFinalizationTask(t, tasks, "sibling", parent, api.PipelineTask_RUNNING, nil)
	require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "stopped", parent.UUID, nil))
	own := driverWrite(sibling, 0, nil)
	own.State = model.TaskStatus(api.PipelineTask_SUCCEEDED)
	succeeded, err := tasks.UpdateTask(own)
	require.NoError(t, err)
	require.Equal(t, own.State, succeeded.State)
	before, err := tasks.GetTask(parent.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), before.State)
	propagation := driverWrite(parent, 0, nil)
	propagation.DriverWriteAuthority.SourceTaskID = sibling.UUID
	propagation.State = model.TaskStatus(api.PipelineTask_SUCCEEDED)
	propagation.StatusMetadata = model.JSONData{"message": ""}
	propagation.OutputParameters = model.JSONSlice{map[string]interface{}{"parameterKey": "result", "value": "complete", "type": "OUTPUT"}}
	after, err := tasks.UpdateTask(propagation)
	require.NoError(t, err)
	assert.Equal(t, before.State, after.State)
	assert.Equal(t, before.StatusMetadata, after.StatusMetadata)
	assert.Equal(t, before.FinishedInSec, after.FinishedInSec)
	require.Len(t, after.OutputParameters, 1)
}

func TestFinalizeStoppedDriverDAGAndMissingChild(t *testing.T) {
	for _, taskType := range []api.PipelineTask_TaskType{api.PipelineTask_DAG, api.PipelineTask_LOOP, api.PipelineTask_CONDITION, api.PipelineTask_CONDITION_BRANCH} {
		t.Run(taskType.String(), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			request := attemptedDriverTask("0", "0")
			request.Type = model.TaskType(taskType)
			task, err := tasks.CreateTask(request)
			require.NoError(t, err)
			require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, task.Name, "", nil))
			stopped, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), stopped.State)
		})
	}
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
	require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "child-not-created", parent.UUID, nil))
	stopped, err := tasks.GetTask(parent.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), stopped.State)
	count, err := tasks.GetTaskCountForRun("run-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStoppedDriverIdentityRejectsDelayedCreateOnlyInItsGeneration(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, api.PipelineTask_RUNNING, nil)
	require.NoError(t, tasks.FinalizeStoppedDriver("run-1", 0, "late-child", parent.UUID, nil))
	request := attemptedDriverTask("0", "0")
	request.Name = "late-child"
	request.ScopePath = "root.late-child"
	request.ParentTaskUUID = &parent.UUID
	_, err := tasks.CreateTask(request)
	require.ErrorContains(t, err, "different driver retry attempt")
	request.DriverClaim = false
	request.DriverRetryGeneration = nil
	request.DriverRetryAttempt = nil
	request.DriverWriteAuthority.SourceAttempt = nil
	_, err = tasks.CreateTask(request)
	require.ErrorContains(t, err, "different driver retry attempt", "ordinary delayed runtime creates share the stop fence")
	request.Name = "sibling"
	request.ScopePath = "root.sibling"
	_, err = tasks.CreateTask(request)
	require.NoError(t, err, "other sibling identities are not stopped")
	_, err = db.Exec("UPDATE run_details SET RetryGeneration=1 WHERE UUID=?", "run-1")
	require.NoError(t, err)
	require.NoError(t, tasks.ResetTasksForRetry("run-1", 1, []string{parent.UUID}))
	request = attemptedDriverTask("1", "0")
	request.Name = "late-child"
	request.ScopePath = "root.late-child"
	request.ParentTaskUUID = &parent.UUID
	_, err = tasks.CreateTask(request)
	require.NoError(t, err)
	var stops int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM driver_task_stops WHERE RunUUID=?", "run-1").Scan(&stops))
	require.Equal(t, 1, stops)
	_, err = db.Exec("DELETE FROM tasks WHERE RunUUID=?", "run-1")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM run_details WHERE UUID=?", "run-1")
	require.NoError(t, err)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM driver_task_stops").Scan(&stops))
	assert.Zero(t, stops)
}
