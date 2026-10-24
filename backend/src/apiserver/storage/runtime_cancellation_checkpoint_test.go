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
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeFrameworkCancellationCheckpointFencedAndProvisional(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	_, err := db.Exec("UPDATE run_details SET State = 'CANCELING', ExecutorType = 'docker' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	update := &model.Task{
		UUID: task.UUID, RunUUID: task.RunUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		StatusMetadata: model.JSONData{"customProperties": map[string]any{"attempt_logs": "file:///logs/attempt"}},
	}
	identity := &model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt"}
	_, err = tasks.ReportTasks(identity, []*model.Task{update})
	require.ErrorIs(t, err, ErrRuntimeRunCanceled)
	require.ErrorIs(t, runtime.Checkpoint(attempt, update), ErrRuntimeRunCanceled)
	attempt.State = model.RuntimeTaskCanceling
	attempt.PublicationRetryCount = 1
	require.NoError(t, runtime.CheckpointCancellation(attempt, update))
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskCanceling, current.State)
	assert.Equal(t, 1, current.PublicationRetryCount)
	assert.Zero(t, current.RetryCount)

	attempt.State = model.RuntimeTaskCancelFinalizing
	attempt.PublicationRetryCount = 2
	require.NoError(t, runtime.CheckpointCancellation(attempt, update))
	require.ErrorIs(t, runtime.Update(attempt, "attempt"), ErrRuntimeRunCanceled)
	require.ErrorIs(t, runtime.Checkpoint(attempt, update), ErrRuntimeRunCanceled)
	_, err = tasks.ReportTasks(identity, []*model.Task{update})
	require.ErrorIs(t, err, ErrRuntimeRunCanceled)

	stale := *attempt
	staleID := "stale"
	stale.AttemptUUID = &staleID
	require.ErrorIs(t, runtime.CheckpointCancellation(&stale, update), ErrRuntimeTaskConflict)
	changedRetry := *attempt
	changedRetry.RetryCount++
	require.ErrorContains(t, runtime.CheckpointCancellation(&changedRetry, update), "only BeginRetry")
	terminal := *update
	terminal.State = model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
	require.ErrorContains(t, runtime.CheckpointCancellation(attempt, &terminal), "provisional")
	rewind := *attempt
	rewind.State = model.RuntimeTaskCanceling
	require.ErrorContains(t, runtime.CheckpointCancellation(&rewind, update), "cannot return")
	invalid := *update
	invalid.StatusMetadata = model.JSONData{"invalid": make(chan bool)}
	require.Error(t, runtime.CheckpointCancellation(attempt, &invalid))
	stored, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), stored.State)
	assert.Equal(t, update.StatusMetadata, stored.StatusMetadata)
	assert.Empty(t, stored.OutputParameters)
	current, err = runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeTaskCancelFinalizing, current.State)
	assert.Equal(t, 2, current.PublicationRetryCount)
	require.NoError(t, runtime.Finalize(task.UUID, "attempt", &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)}))
	require.ErrorIs(t, runtime.CheckpointCancellation(attempt, update), ErrRuntimeTaskConflict)
	stored, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), stored.State)
	assert.Equal(t, update.StatusMetadata, stored.StatusMetadata, "cancellation log references must survive finalization")
}
