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
	"fmt"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attemptedDriverTask(generation, attempt string) *model.Task {
	task := driverRetryWriteTask(generation)
	driverTaskProperties(task)[util.DriverRetryAttemptKey] = attempt
	return task
}

func TestDriverRetryAttemptClaimPreservesCompletedHandoff(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	created, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.NoError(t, err)
	late, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	completed, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	completed.State = model.TaskStatus(api.PipelineTask_CACHED)
	completed.FinishedInSec = 42
	driverTaskProperties(completed)["_kfp_driver_checkpoint"] = "saved handoff"
	driverTaskProperties(completed)["_kfp_driver_cached_outputs"] = "saved cache decision"
	completed, err = tasks.UpdateTask(completed)
	require.NoError(t, err)

	claimed, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, completed.UUID, claimed.UUID)
	assert.Equal(t, completed.State, claimed.State)
	assert.Equal(t, completed.FinishedInSec, claimed.FinishedInSec)
	assert.Equal(t, completed.StateHistory, claimed.StateHistory)
	assert.Equal(t, completed.Pods, claimed.Pods)
	assert.Equal(t, "saved handoff", driverTaskProperties(claimed)["_kfp_driver_checkpoint"])
	assert.Equal(t, "saved cache decision", driverTaskProperties(claimed)["_kfp_driver_cached_outputs"])
	assert.Equal(t, "1", driverTaskProperties(claimed)[util.DriverRetryAttemptKey])
	// Retrying a claim after losing its response must leave the same owner and
	// recovery data, while delayed older claims and writes cannot erase them.
	repeated, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, claimed, repeated)
	_, err = tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.UpdateTask(late)
	require.ErrorContains(t, err, "different driver retry attempt")
	unclaimed := attemptedDriverTask("0", "2")
	unclaimed.UUID = created.UUID
	_, err = tasks.UpdateTask(unclaimed)
	require.ErrorContains(t, err, "different driver retry attempt")
	for _, tagged := range []bool{true, false} {
		missing := driverRetryWriteTask("0")
		if !tagged {
			missing.StatusMetadata = nil
		}
		_, err = tasks.CreateTask(missing)
		require.ErrorContains(t, err, "different driver retry attempt")
		missing.UUID = created.UUID
		_, err = tasks.UpdateTask(missing)
		require.ErrorContains(t, err, "different driver retry attempt")
	}
	after, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	assert.Equal(t, claimed, after)
	// The rightful owner can continue using the saved handoff.
	after.StatusMetadata["message"] = "acknowledged"
	_, err = tasks.UpdateTask(after)
	require.NoError(t, err)
}

func TestDriverRetryAttemptManualGeneration(t *testing.T) {
	for _, state := range []api.PipelineTask_TaskState{api.PipelineTask_CACHED, api.PipelineTask_SUCCEEDED, api.PipelineTask_SKIPPED, api.PipelineTask_FAILED} {
		t.Run(state.String(), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			task, err := tasks.CreateTask(attemptedDriverTask("0", "2"))
			require.NoError(t, err)
			task.State = model.TaskStatus(state)
			driverTaskProperties(task)["_kfp_driver_checkpoint"] = "saved handoff"
			_, err = tasks.UpdateTask(task)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE run_details SET RetryGeneration = 1 WHERE UUID = ?", "run-1")
			require.NoError(t, err)
			if state == api.PipelineTask_FAILED {
				_, err = tasks.CreateTask(attemptedDriverTask("1", "0"))
				require.ErrorContains(t, err, "must be reset")
				require.NoError(t, tasks.ResetTasksForRetry([]string{task.UUID}))
			}
			claimed, err := tasks.CreateTask(attemptedDriverTask("1", "0"))
			require.NoError(t, err)
			assert.Equal(t, task.UUID, claimed.UUID)
			assert.Equal(t, "1", driverTaskProperties(claimed)[util.DriverRetryGenerationKey])
			assert.Equal(t, "0", driverTaskProperties(claimed)[util.DriverRetryAttemptKey])
			if state != api.PipelineTask_FAILED {
				assert.Equal(t, model.TaskStatus(state), claimed.State)
				assert.Equal(t, "saved handoff", driverTaskProperties(claimed)["_kfp_driver_checkpoint"])
			} else {
				assert.NotContains(t, driverTaskProperties(claimed), "_kfp_driver_checkpoint")
			}
			_, err = tasks.UpdateTask(task)
			require.ErrorContains(t, err, "different retry generation")
		})
	}
}

func TestDriverRetryAttemptSourceFencesParentWrites(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	tasks.uuid = util.NewUUIDGenerator()
	parentRequest := attemptedDriverTask("0", "4")
	parentRequest.Name, parentRequest.ScopePath = "parent", "root.parent"
	parent, err := tasks.CreateTask(parentRequest)
	require.NoError(t, err)
	driverTaskProperties(parent)["_kfp_driver_checkpoint"] = "parent handoff"
	parent, err = tasks.UpdateTask(parent)
	require.NoError(t, err)
	child, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	parentWrite := func(attempt string) *model.Task {
		request := attemptedDriverTask("0", "0") // A stale parent snapshot is not its source's claim.
		request.UUID = parent.UUID
		request.Name, request.ScopePath = "parent", "root.parent"
		request.State = model.TaskStatus(api.PipelineTask_CACHED)
		properties := driverTaskProperties(request)
		properties[util.DriverRetrySourceTaskKey] = child.UUID
		properties[util.DriverRetrySourceAttemptKey] = attempt
		properties["_kfp_driver_checkpoint"] = "stale parent handoff"
		return request
	}
	updated, err := tasks.UpdateTask(parentWrite("1"))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(api.PipelineTask_CACHED), updated.State)
	assert.Equal(t, "4", driverTaskProperties(updated)[util.DriverRetryAttemptKey])
	assert.Equal(t, "parent handoff", driverTaskProperties(updated)["_kfp_driver_checkpoint"])
	assert.NotContains(t, driverTaskProperties(updated), util.DriverRetrySourceTaskKey)
	assert.NotContains(t, driverTaskProperties(updated), util.DriverRetrySourceAttemptKey)
	_, err = tasks.UpdateTask(parentWrite("0"))
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.CreateTask(attemptedDriverTask("0", "2"))
	require.NoError(t, err)
	_, err = tasks.UpdateTask(parentWrite("1"))
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.UpdateTask(parentWrite("2"))
	require.NoError(t, err)

	// A source from another run cannot authorize the write.
	_, err = db.Exec("UPDATE run_details SET State = 'RUNNING' WHERE UUID = ?", "run-2")
	require.NoError(t, err)
	foreign := attemptedDriverTask("0", "2")
	foreign.RunUUID = "run-2"
	foreign, err = tasks.CreateTask(foreign)
	require.NoError(t, err)
	crossRun := parentWrite("2")
	driverTaskProperties(crossRun)[util.DriverRetrySourceTaskKey] = foreign.UUID
	_, err = tasks.UpdateTask(crossRun)
	require.ErrorContains(t, err, "different driver retry attempt")
}

func TestDriverRetryAttemptMalformedFences(t *testing.T) {
	invalidAttempts := []interface{}{"", "-1", "01", "+1", "invalid", "9223372036854775808", float64(1), true, nil}
	for _, key := range []string{util.DriverRetryAttemptKey, util.DriverRetrySourceAttemptKey} {
		for _, value := range invalidAttempts {
			t.Run(fmt.Sprintf("%s/%v", key, value), func(t *testing.T) {
				db, tasks, _ := initializeTaskStore()
				defer db.Close()
				created, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
				require.NoError(t, err)
				request := attemptedDriverTask("0", "0")
				request.UUID = created.UUID
				driverTaskProperties(request)[key] = value
				_, err = tasks.CreateTask(request)
				require.ErrorContains(t, err, "nonnegative decimal string")
				_, err = tasks.UpdateTask(request)
				require.ErrorContains(t, err, "nonnegative decimal string")
			})
		}
	}
	for _, properties := range []map[string]interface{}{
		{util.DriverRetryAttemptKey: "0"},
		{util.DriverRetryGenerationKey: "0", util.DriverRetrySourceTaskKey: "source"},
		{util.DriverRetryGenerationKey: "0", util.DriverRetrySourceAttemptKey: "0"},
		{util.DriverRetryGenerationKey: "0", util.DriverRetrySourceTaskKey: "", util.DriverRetrySourceAttemptKey: "0"},
		{util.DriverRetryGenerationKey: "0", util.DriverRetrySourceTaskKey: float64(1), util.DriverRetrySourceAttemptKey: "0"},
		{util.DriverRetrySourceTaskKey: "source", util.DriverRetrySourceAttemptKey: "0"},
	} {
		t.Run(fmt.Sprint(properties), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			created, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
			require.NoError(t, err)
			request := attemptedDriverTask("0", "0")
			request.UUID = created.UUID
			request.StatusMetadata["customProperties"] = properties
			_, err = tasks.CreateTask(request)
			require.Error(t, err)
			_, err = tasks.UpdateTask(request)
			require.Error(t, err)
		})
	}
}

func TestDriverRetryAttemptClaimUpgradesGenerationOnlyTask(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	legacy, err := tasks.CreateTask(driverRetryWriteTask("0"))
	require.NoError(t, err)
	_, err = tasks.UpdateTask(legacy)
	require.NoError(t, err)
	claimed, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, legacy.UUID, claimed.UUID)
	assert.Equal(t, "1", driverTaskProperties(claimed)[util.DriverRetryAttemptKey])
	_, err = tasks.UpdateTask(legacy)
	require.ErrorContains(t, err, "different driver retry attempt")
}

func TestDriverRetryAttemptSourcePreservesParentGeneration(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	tasks.uuid = util.NewUUIDGenerator()
	parentRequest := attemptedDriverTask("0", "4")
	parentRequest.Name, parentRequest.ScopePath = "parent", "root.parent"
	parentRequest.State = model.TaskStatus(api.PipelineTask_SUCCEEDED)
	parent, err := tasks.CreateTask(parentRequest)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE run_details SET RetryGeneration = 1 WHERE UUID = ?", "run-1")
	require.NoError(t, err)
	child, err := tasks.CreateTask(attemptedDriverTask("1", "0"))
	require.NoError(t, err)
	metadata, properties := copyDriverTaskMetadata(parent)
	properties[util.DriverRetryGenerationKey] = "1"
	properties[util.DriverRetrySourceTaskKey] = child.UUID
	properties[util.DriverRetrySourceAttemptKey] = "0"
	parent.StatusMetadata = metadata
	parent.State = model.TaskStatus(api.PipelineTask_CACHED)
	updated, err := tasks.UpdateTask(parent)
	require.NoError(t, err)
	assert.Equal(t, "0", driverTaskProperties(updated)[util.DriverRetryGenerationKey])
	assert.Equal(t, "4", driverTaskProperties(updated)[util.DriverRetryAttemptKey])
	assert.NotContains(t, driverTaskProperties(updated), util.DriverRetrySourceTaskKey)
	assert.NotContains(t, driverTaskProperties(updated), util.DriverRetrySourceAttemptKey)
	// A dependent write must not accidentally adopt the preserved parent into
	// the new generation; its own new driver must explicitly claim it.
	parentRequest = attemptedDriverTask("1", "0")
	parentRequest.Name, parentRequest.ScopePath = "parent", "root.parent"
	claimed, err := tasks.CreateTask(parentRequest)
	require.NoError(t, err)
	assert.Equal(t, parent.UUID, claimed.UUID)
	assert.Equal(t, "1", driverTaskProperties(claimed)[util.DriverRetryGenerationKey])
	assert.Equal(t, "0", driverTaskProperties(claimed)[util.DriverRetryAttemptKey])
}

func TestDriverRetryAttemptClaimFailureRollsBack(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	before, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_driver_claim BEFORE UPDATE ON tasks
		BEGIN SELECT RAISE(ABORT, 'injected claim failure'); END`)
	require.NoError(t, err)
	_, err = tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.ErrorContains(t, err, "injected claim failure")
	after, err := tasks.GetTask(before.UUID)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	_, err = db.Exec("DROP TRIGGER reject_driver_claim")
	require.NoError(t, err)
	claimed, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, "1", driverTaskProperties(claimed)[util.DriverRetryAttemptKey])
}
