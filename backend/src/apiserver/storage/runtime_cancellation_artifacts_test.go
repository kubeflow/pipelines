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
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cancellationLogFixture(task *model.Task, attemptID string) (*model.Task, *model.Artifact, *model.ArtifactTask) {
	uri := "file:///logs/attempt"
	artifact := &model.Artifact{
		UUID: util.NewDeterministicUUID("logs:" + task.UUID + ":" + attemptID), Name: "executor-logs", URI: &uri,
	}
	link := &model.ArtifactTask{
		UUID: util.NewDeterministicUUID("log-link:" + artifact.UUID), ArtifactID: artifact.UUID,
		TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT),
		ArtifactKey: "executor-logs", Iteration: model.ArtifactTaskNoIteration,
	}
	update := &model.Task{
		UUID: task.UUID, RunUUID: task.RunUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		StatusMetadata: model.JSONData{"logs": uri},
	}
	return update, artifact, link
}

func TestRuntimeCancellationArtifactsAtomicAndIdempotent(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	attempt := runtimeAttemptFixture(task, "attempt")
	require.NoError(t, runtime.Create(attempt))
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	token, err := tasks.RunLeaseStore().Acquire(task.RunUUID, "worker", time.Minute, false)
	require.NoError(t, err)
	bound, err := tasks.LeaseScopedStores(*token)
	require.NoError(t, err)
	runtime = bound.Runtime
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", model.RuntimeStateCancelling, task.RunUUID)
	require.NoError(t, err)
	attempt.State = model.RuntimeTaskCanceling
	require.NoError(t, runtime.Update(attempt, "attempt"))
	before, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	attempt.ExecutionReference["completion"] = map[string]any{"count": float64(1)}
	update, artifact, link := cancellationLogFixture(task, "attempt")
	checkpoint := func() error {
		return runtime.CheckpointCancellationArtifacts(attempt, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link})
	}
	_, err = db.Exec(`CREATE TRIGGER reject_cancel_log BEFORE INSERT ON artifact_tasks
		BEGIN SELECT RAISE(ABORT, 'log relationship unavailable'); END`)
	require.NoError(t, err)
	require.ErrorContains(t, checkpoint(), "log relationship unavailable")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Zero(t, count)
	current, err := runtime.Get(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, before, current, "log metadata, links, public diagnostics, and reference progress must roll back together")
	public, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Empty(t, public.StatusMetadata)
	_, err = db.Exec("DROP TRIGGER reject_cancel_log")
	require.NoError(t, err)
	require.NoError(t, checkpoint())
	require.NoError(t, checkpoint(), "replaying stopped-workload log publication must reuse deterministic identities")
	artifact.Name = "changed"
	require.ErrorContains(t, checkpoint(), "different metadata")
	artifact.Name = "executor-logs"
	link.Producer = model.JSONData{"taskName": "changed"}
	require.ErrorContains(t, checkpoint(), "different output")
	link.Producer = nil
	normal := *attempt
	normal.State = model.RuntimeTaskRunning
	require.ErrorIs(t, runtime.CheckpointArtifacts(&normal, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link}), ErrRuntimeRunCanceled)
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", model.RuntimeStateRunning, task.RunUUID)
	require.NoError(t, err)
	require.ErrorIs(t, runtime.CheckpointArtifacts(&normal, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link}), ErrRuntimeRunCanceled,
		"the task cancellation phase remains authoritative before run cancellation is recorded")
	_, err = db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", model.RuntimeStateCanceled, task.RunUUID)
	require.NoError(t, err)
	stale := "old-attempt"
	attempt.AttemptUUID = &stale
	require.ErrorIs(t, checkpoint(), ErrRuntimeTaskConflict)
	active := "attempt"
	attempt.AttemptUUID = &active
	attempt.State = model.RuntimeTaskCancelFinalizing
	require.NoError(t, checkpoint())
	attempt.State = model.RuntimeTaskCanceling
	require.ErrorContains(t, checkpoint(), "cannot return")
	attempt.State = model.RuntimeTaskCancelFinalizing
	public, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_RUNNING), public.State)
	assert.Empty(t, public.OutputParameters)
	require.Len(t, public.OutputArtifactsHydrated, 1)
	assert.Equal(t, "executor-logs", public.OutputArtifactsHydrated[0].Key)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
	assert.Equal(t, 1, count)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&count))
	assert.Equal(t, 1, count)
	require.NoError(t, runtime.Finalize(task.UUID, "attempt", &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED),
	}))
	require.ErrorIs(t, checkpoint(), ErrRuntimeTaskConflict)
	public, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), public.State)
	require.Len(t, public.OutputArtifactsHydrated, 1, "finalization must retain published cancellation logs")
	require.NoError(t, tasks.RunLeaseStore().Release(*token))
	require.ErrorIs(t, checkpoint(), ErrRunLeaseConflict, "stale workers cannot recreate already-finalized log metadata")
}

func TestRuntimeCancellationArtifactsRejectNormalOutputsAndInvalidFences(t *testing.T) {
	for _, invalid := range []string{
		"runtime-phase", "current-phase", "public-terminal", "public-unspecified", "current-public-terminal",
		"null-attempt", "task-id", "run-id", "retry-count", "request", "parameters", "link-key", "link-type",
		"hydrated-key", "hydrated-type", "artifact-namespace", "missing-artifact",
	} {
		t.Run(invalid, func(t *testing.T) {
			db, tasks, runtime, task := runtimeFixture(t)
			attempt := runtimeAttemptFixture(task, "attempt")
			attempt.State = model.RuntimeTaskCanceling
			require.NoError(t, runtime.Create(attempt))
			_, err := db.Exec("UPDATE run_details SET State = ? WHERE UUID = ?", model.RuntimeStateCanceled, task.RunUUID)
			require.NoError(t, err)
			update, artifact, link := cancellationLogFixture(task, "attempt")
			switch invalid {
			case "runtime-phase":
				attempt.State = model.RuntimeTaskRunning
			case "current-phase":
				_, err = db.Exec("UPDATE runtime_tasks SET State = ? WHERE TaskUUID = ?", model.RuntimeTaskRunning, task.UUID)
				require.NoError(t, err)
			case "public-terminal":
				update.State = model.TaskStatus(apiv2.PipelineTask_SKIPPED)
			case "public-unspecified":
				update.State = 0
			case "current-public-terminal":
				_, err = db.Exec("UPDATE tasks SET State = ? WHERE UUID = ?", apiv2.PipelineTask_SKIPPED, task.UUID)
				require.NoError(t, err)
			case "null-attempt":
				attempt.AttemptUUID = nil
			case "task-id":
				update.UUID = "different"
			case "run-id":
				update.RunUUID = "2"
			case "retry-count":
				attempt.RetryCount++
			case "request":
				attempt.ExecutionRequest = model.JSONData{"changed": true}
			case "parameters":
				update.OutputParameters = model.JSONSlice{map[string]any{"parameterKey": "result", "type": "OUTPUT"}}
			case "link-key":
				link.ArtifactKey = "normal-output"
			case "link-type":
				link.Type = model.IOType(apiv2.IOType_COMPONENT_INPUT)
			case "hydrated-key":
				update.OutputArtifactsHydrated = runtimeOutputUpdate(t, task).OutputArtifactsHydrated
			case "hydrated-type":
				update.OutputArtifactsHydrated = []model.TaskArtifactHydrated{{Key: "executor-logs", Type: apiv2.IOType_COMPONENT_INPUT, Value: artifact}}
			case "artifact-namespace":
				artifact.Namespace = "different"
			case "missing-artifact":
				link.ArtifactID = "missing"
			}
			err = runtime.CheckpointCancellationArtifacts(attempt, update, []*model.Artifact{artifact}, []*model.ArtifactTask{link})
			require.Error(t, err)
			switch invalid {
			case "current-phase":
				require.ErrorIs(t, err, ErrRuntimeRunCanceled)
			case "current-public-terminal":
				require.ErrorIs(t, err, ErrRuntimeTaskConflict)
			case "null-attempt":
				require.ErrorContains(t, err, "current nonempty execution attempt")
			case "retry-count":
				require.ErrorContains(t, err, "only BeginRetry")
			case "request":
				require.ErrorContains(t, err, "immutable")
			case "parameters":
				require.ErrorContains(t, err, "without normal output parameters")
			case "link-key", "link-type":
				require.ErrorContains(t, err, "executor-logs OUTPUT links")
			case "hydrated-key", "hydrated-type":
				require.ErrorContains(t, err, "executor-logs OUTPUT artifacts")
			}
			var count int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&count))
			assert.Zero(t, count)
			public, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Empty(t, public.OutputParameters)
			assert.Empty(t, public.OutputArtifactsHydrated)
			_, err = runtime.Get(task.UUID)
			require.NoError(t, err)
		})
	}
}
