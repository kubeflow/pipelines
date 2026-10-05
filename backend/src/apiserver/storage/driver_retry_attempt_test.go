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
	"strconv"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func retryInt(value int64) *int64 { return &value }
func attemptedDriverTask(generation, attempt string) *model.Task {
	task := driverRetryWriteTask(generation)
	value, err := strconv.ParseInt(attempt, 10, 64)
	if err != nil {
		panic(err)
	}
	task.DriverRetryGeneration = retryInt(task.DriverWriteAuthority.Generation)
	task.DriverRetryAttempt = retryInt(value)
	task.DriverWriteAuthority.SourceAttempt = retryInt(value)
	task.DriverClaim = true
	return task
}
func driverWrite(task *model.Task, generation int64, attempt *int64) *model.Task {
	request := *task
	request.DriverWriteAuthority = &model.DriverTaskAuthority{Generation: generation, SourceAttempt: attempt}
	request.DriverClaim = false
	return &request
}

// Fixtures are newly created owners; race tests always capture explicit authority.
func driverFixtureWrite(task *model.Task) *model.Task {
	generation := int64(0)
	if task.DriverRetryGeneration != nil {
		generation = *task.DriverRetryGeneration
	}
	return driverWrite(task, generation, task.DriverRetryAttempt)
}

func TestDriverRetryAttemptClaimPreservesCompletedHandoff(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	created, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.NoError(t, err)
	late := driverWrite(created, 0, retryInt(0))
	completed := driverWrite(created, 0, retryInt(0))
	completed.State = model.TaskStatus(api.PipelineTask_CACHED)
	completed.FinishedInSec = 42
	completed.DriverRecoveryUpdate = true
	completed.DriverCheckpoint = util.StringPointer(`{"handoff":"saved"}`)
	completed.DriverCachedOutputs = util.StringPointer(`{"cache":"saved"}`)
	completed, err = tasks.UpdateTask(completed)
	require.NoError(t, err)
	claimed, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, completed.State, claimed.State)
	assert.Equal(t, completed.FinishedInSec, claimed.FinishedInSec)
	assert.Equal(t, completed.StateHistory, claimed.StateHistory)
	assert.Equal(t, completed.DriverCheckpoint, claimed.DriverCheckpoint)
	assert.Equal(t, completed.DriverCachedOutputs, claimed.DriverCachedOutputs)
	assert.Equal(t, int64(1), *claimed.DriverRetryAttempt)
	repeated, err := tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
	assert.Equal(t, claimed, repeated)
	_, err = tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.UpdateTask(late)
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.UpdateTask(driverWrite(created, 0, retryInt(2)))
	require.ErrorContains(t, err, "different driver retry attempt")
	_, err = tasks.UpdateTask(claimed)
	require.ErrorContains(t, err, "different driver retry attempt", "reads never grant authority")
	_, err = tasks.CreateTask(driverRetryWriteTask("0"))
	require.ErrorContains(t, err, "different driver retry attempt")
	after, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	assert.Equal(t, claimed, after)
	_, err = tasks.UpdateTask(driverWrite(after, 0, retryInt(1)))
	require.NoError(t, err)
}

func TestDriverRetryAttemptManualGeneration(t *testing.T) {
	for _, state := range []api.PipelineTask_TaskState{api.PipelineTask_CACHED, api.PipelineTask_SUCCEEDED, api.PipelineTask_SKIPPED, api.PipelineTask_FAILED} {
		t.Run(state.String(), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			task, err := tasks.CreateTask(attemptedDriverTask("0", "2"))
			require.NoError(t, err)
			old := driverWrite(task, 0, retryInt(2))
			old.State = model.TaskStatus(state)
			old.DriverRecoveryUpdate = true
			old.DriverCheckpoint = util.StringPointer(`{"saved":true}`)
			task, err = tasks.UpdateTask(old)
			require.NoError(t, err)
			_, err = db.Exec("UPDATE run_details SET RetryGeneration=1 WHERE UUID=?", "run-1")
			require.NoError(t, err)
			if state == api.PipelineTask_FAILED {
				_, err = tasks.CreateTask(attemptedDriverTask("1", "0"))
				require.ErrorContains(t, err, "must be reset")
				require.NoError(t, tasks.ResetTasksForRetry("run-1", 1, []string{task.UUID}))
			}
			claimed, err := tasks.CreateTask(attemptedDriverTask("1", "0"))
			require.NoError(t, err)
			assert.Equal(t, int64(1), *claimed.DriverRetryGeneration)
			assert.Equal(t, int64(0), *claimed.DriverRetryAttempt)
			if state == api.PipelineTask_FAILED {
				assert.Nil(t, claimed.DriverCheckpoint)
			} else {
				assert.Equal(t, task.DriverCheckpoint, claimed.DriverCheckpoint)
				assert.Equal(t, task.State, claimed.State)
			}
			_, err = tasks.UpdateTask(old)
			require.ErrorContains(t, err, "different retry generation")
		})
	}
}

func TestDriverRetryAttemptSourceFencesParentWrites(t *testing.T) {
	for _, claimedChild := range []bool{false, true} {
		t.Run(strconv.FormatBool(claimedChild), func(t *testing.T) {
			db, tasks, _ := initializeTaskStore()
			defer db.Close()
			tasks.uuid = util.NewUUIDGenerator()
			request := attemptedDriverTask("0", "4")
			request.Name = "parent"
			request.ScopePath = "root.parent"
			parent, err := tasks.CreateTask(request)
			require.NoError(t, err)
			request = driverWrite(parent, 0, retryInt(4))
			request.DriverRecoveryUpdate = true
			request.DriverCheckpoint = util.StringPointer(`{"parent":true}`)
			parent, err = tasks.UpdateTask(request)
			require.NoError(t, err)
			childRequest := driverRetryWriteTask("0")
			var attempt *int64
			if claimedChild {
				childRequest = attemptedDriverTask("0", "1")
				attempt = retryInt(1)
			}
			childRequest.ParentTaskUUID = &parent.UUID
			child, err := tasks.CreateTask(childRequest)
			require.NoError(t, err)
			propagation := driverWrite(parent, 0, attempt)
			propagation.DriverWriteAuthority.SourceTaskID = child.UUID
			propagation.StatusMetadata = nil // Omitting metadata must preserve the parent's user metadata.
			propagation.DriverRetryAttempt = retryInt(99)
			propagation.DriverCheckpoint = util.StringPointer(`{"stale":true}`)
			propagation.State = model.TaskStatus(api.PipelineTask_CACHED)
			updated, err := tasks.UpdateTask(propagation)
			require.NoError(t, err)
			assert.Equal(t, int64(4), *updated.DriverRetryAttempt)
			assert.Equal(t, parent.DriverCheckpoint, updated.DriverCheckpoint)
			assert.Equal(t, parent.StatusMetadata, updated.StatusMetadata)
			if claimedChild {
				next := attemptedDriverTask("0", "2")
				next.ParentTaskUUID = &parent.UUID
				_, err = tasks.CreateTask(next)
				require.NoError(t, err)
				_, err = tasks.UpdateTask(propagation)
				require.ErrorContains(t, err, "different driver retry attempt")
			} else {
				propagation.DriverWriteAuthority.SourceAttempt = retryInt(4) // A refreshed parent cannot confer its claim.
				_, err = tasks.UpdateTask(propagation)
				require.ErrorContains(t, err, "different driver retry attempt")
			}
			propagation.DriverWriteAuthority.Generation = 1
			_, err = tasks.UpdateTask(propagation)
			require.ErrorContains(t, err, "different retry generation")
		})
	}
}

func TestDriverRetryRejectsInvalidAuthorityAndPublicRecovery(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	for _, mutate := range []func(*model.Task){
		func(task *model.Task) { task.DriverWriteAuthority = nil },
		func(task *model.Task) { task.DriverWriteAuthority.Generation = -1 },
		func(task *model.Task) { task.DriverRetryAttempt = retryInt(-1) },
		func(task *model.Task) { task.DriverRetryGeneration = nil },
		func(task *model.Task) { task.DriverWriteAuthority.SourceTaskID = "another" },
		func(task *model.Task) {
			task.DriverRecoveryUpdate = true
			task.DriverCheckpoint = util.StringPointer("not JSON")
		},
	} {
		request := attemptedDriverTask("0", "0")
		mutate(request)
		_, err := tasks.CreateTask(request)
		require.Error(t, err)
	}
	public := driverRetryWriteTask("0")
	public.DriverWriteAuthority = nil
	public.StatusMetadata = model.JSONData{"customProperties": map[string]interface{}{util.DriverRetryGenerationKey: "0", util.DriverRetryAttemptKey: "999"}}
	public.DriverRetryGeneration = retryInt(0)
	public.DriverRetryAttempt = retryInt(999)
	public.DriverCheckpoint = util.StringPointer(`{"forged":true}`)
	created, err := tasks.CreateTask(public)
	require.NoError(t, err)
	assert.Nil(t, created.DriverRetryGeneration)
	assert.Nil(t, created.DriverRetryAttempt)
	assert.Nil(t, created.DriverCheckpoint)
	public.UUID = created.UUID
	_, err = tasks.UpdateTask(public)
	require.NoError(t, err)
	after, err := tasks.GetTask(created.UUID)
	require.NoError(t, err)
	assert.Nil(t, after.DriverCheckpoint)
	assert.Nil(t, after.DriverRetryAttempt)
}

func TestDriverRetryAttemptClaimRollsBack(t *testing.T) {
	db, tasks, _ := initializeTaskStore()
	defer db.Close()
	before, err := tasks.CreateTask(attemptedDriverTask("0", "0"))
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_claim BEFORE UPDATE ON tasks BEGIN SELECT RAISE(FAIL,'injected claim failure'); END`)
	require.NoError(t, err)
	_, err = tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.ErrorContains(t, err, "injected claim failure")
	after, err := tasks.GetTask(before.UUID)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	_, err = db.Exec("DROP TRIGGER reject_claim")
	require.NoError(t, err)
	_, err = tasks.CreateTask(attemptedDriverTask("0", "1"))
	require.NoError(t, err)
}
