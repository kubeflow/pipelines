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
	"encoding/json"
	"fmt"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func createDriverRetryFinalizationTask(t *testing.T, store *TaskStore, name string, parent *model.Task, state apiv2beta1.PipelineTask_TaskState, generation *string) *model.Task {
	t.Helper()
	task := &model.Task{
		Namespace: "ns1",
		RunUUID:   "run-1",
		Name:      name,
		ScopePath: "root." + name,
		Type:      model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:     model.TaskStatus(state),
		TypeAttrs: model.JSONData{},
		Pods:      model.JSONSlice{},
	}
	if parent != nil {
		task.ParentTaskUUID = util.StringPointer(parent.UUID)
	}
	if generation != nil {
		metadata, err := model.ProtoMessageToJSONData(&apiv2beta1.PipelineTask_StatusMetadata{
			CustomProperties: map[string]*structpb.Value{
				util.DriverRetryGenerationKey: structpb.NewStringValue(*generation),
				"plugin_resource_id":          structpb.NewStringValue("existing-resource"),
			},
		})
		require.NoError(t, err)
		task.StatusMetadata = metadata
	}
	store.uuid = util.NewUUIDGenerator()
	created, err := store.CreateTask(task)
	require.NoError(t, err)
	return created
}

func reportDriverRetryTerminalRun(t *testing.T, store *RunStore, state model.RuntimeState) {
	t.Helper()
	run, err := store.GetRun("run-1", false)
	require.NoError(t, err)
	expectedWorkflow := run.WorkflowRuntimeManifest
	expectedPipeline := run.PipelineRuntimeManifest
	run.State = state
	run.Conditions = string(state.ToExecutionPhase())
	run.FinishedAtInSec = 123
	run.WorkflowRuntimeManifest = "terminal-workflow"
	updated, err := store.UpdateRunIfRuntimeManifestsUnchanged(run, expectedWorkflow, expectedPipeline)
	require.NoError(t, err)
	require.True(t, updated)
}

func TestDriverRetryFinalizationPropagatesToAncestors(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStateFailed, model.RuntimeStateErrorV1, model.RuntimeStateCanceled} {
		t.Run(string(state), func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			_, err := db.Exec("UPDATE run_details SET RetryGeneration = 7 WHERE UUID = ?", "run-1")
			require.NoError(t, err)
			root := createDriverRetryFinalizationTask(t, tasks, "root", nil, apiv2beta1.PipelineTask_RUNNING, nil)
			parent := createDriverRetryFinalizationTask(t, tasks, "parent", root, apiv2beta1.PipelineTask_RUNNING, nil)
			child := createDriverRetryFinalizationTask(t, tasks, "child", parent, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("7"))
			child.StatusMetadata["message"] = "cache service unavailable"
			parameters, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result", Value: structpb.NewStringValue("preserved"), Type: apiv2beta1.IOType_OUTPUT,
			}})
			require.NoError(t, err)
			child.OutputParameters = parameters
			child, err = tasks.UpdateTask(child)
			require.NoError(t, err)
			unrelated := createDriverRetryFinalizationTask(t, tasks, "unrelated", root, apiv2beta1.PipelineTask_RUNNING, nil)

			reportDriverRetryTerminalRun(t, runs, state)
			for _, before := range []*model.Task{root, parent, child} {
				after, err := tasks.GetTask(before.UUID)
				require.NoError(t, err)
				assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), after.State)
				assert.Equal(t, int64(123), after.FinishedInSec)
				require.Len(t, after.StateHistory, len(before.StateHistory)+1)
				assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), getLastTaskState(after.StateHistory))
				assert.NotEmpty(t, after.StatusMetadata["message"])
			}
			after, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, child.StatusMetadata, after.StatusMetadata)
			assert.Equal(t, child.OutputParameters, after.OutputParameters)
			assert.Equal(t, child.Pods, after.Pods)
			unrelatedAfter, err := tasks.GetTask(unrelated.UUID)
			require.NoError(t, err)
			assert.Equal(t, unrelated, unrelatedAfter)

			reportDriverRetryTerminalRun(t, runs, state)
			replayed, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, after, replayed, "duplicate terminal reports must not append task history")
		})
	}
}

func TestDriverRetryFinalizationPreservesTerminalTasks(t *testing.T) {
	for _, state := range []apiv2beta1.PipelineTask_TaskState{
		apiv2beta1.PipelineTask_SUCCEEDED,
		apiv2beta1.PipelineTask_CACHED,
		apiv2beta1.PipelineTask_SKIPPED,
		apiv2beta1.PipelineTask_FAILED,
	} {
		t.Run(state.String(), func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, apiv2beta1.PipelineTask_RUNNING, nil)
			child := createDriverRetryFinalizationTask(t, tasks, "terminal-child", parent, state, util.StringPointer("0"))
			child.FinishedInSec = 99
			child, err := tasks.UpdateTask(child)
			require.NoError(t, err)

			reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
			after, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, child, after)
			parentAfter, err := tasks.GetTask(parent.UUID)
			require.NoError(t, err)
			assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), parentAfter.State)
		})
	}
}

func TestDriverRetryFinalizationCorrectsPrematureSuccessfulAncestors(t *testing.T) {
	for _, parentState := range []apiv2beta1.PipelineTask_TaskState{apiv2beta1.PipelineTask_CACHED, apiv2beta1.PipelineTask_SUCCEEDED, apiv2beta1.PipelineTask_SKIPPED} {
		for _, parentFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parentFirst=%t", parentState, parentFirst), func(t *testing.T) {
				db, tasks, runs := initializeTaskStore()
				defer db.Close()
				createOrderedTask := func(order int, name string, parent *model.Task, state apiv2beta1.PipelineTask_TaskState) *model.Task {
					task := createDriverRetryFinalizationTask(t, tasks, name, parent, state, util.StringPointer("0"))
					id := fmt.Sprintf("%08d-0000-0000-0000-000000000000", order)
					_, err := db.Exec("UPDATE tasks SET UUID = ? WHERE UUID = ?", id, task.UUID)
					require.NoError(t, err)
					task.UUID = id
					return task
				}
				root := createOrderedTask(2, "root", nil, apiv2beta1.PipelineTask_SUCCEEDED)
				grandparent := createOrderedTask(3, "grandparent", root, apiv2beta1.PipelineTask_CACHED)
				parentOrder, childOrder := 1, 9
				if !parentFirst {
					parentOrder, childOrder = childOrder, parentOrder
				}
				parent := createOrderedTask(parentOrder, "parent", grandparent, parentState)
				parent.FinishedInSec = 99
				parent, err := tasks.UpdateTask(parent)
				require.NoError(t, err)
				// A lost response after propagating a cache hit reopens only the
				// child. Its previously committed successful ancestors must follow
				// the child's final failure, regardless of task UUID ordering.
				child := createOrderedTask(childOrder, "reopened-child", parent, apiv2beta1.PipelineTask_RUNNING)
				child.StatusMetadata["message"] = "status update response lost"
				child, err = tasks.UpdateTask(child)
				require.NoError(t, err)
				sibling := createOrderedTask(4, "successful-sibling", root, apiv2beta1.PipelineTask_SUCCEEDED)
				completedChild := createOrderedTask(5, "cached-sibling-child", sibling, apiv2beta1.PipelineTask_CACHED)

				reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
				var finalized []*model.Task
				for _, before := range []*model.Task{root, grandparent, parent, child} {
					after, err := tasks.GetTask(before.UUID)
					require.NoError(t, err)
					assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), after.State)
					assert.Equal(t, int64(123), after.FinishedInSec)
					require.Len(t, after.StateHistory, len(before.StateHistory)+1)
					assert.Equal(t, before.StateHistory, after.StateHistory[:len(before.StateHistory)])
					assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), getLastTaskState(after.StateHistory))
					assert.Equal(t, before.StatusMetadata["customProperties"], after.StatusMetadata["customProperties"])
					finalized = append(finalized, after)
				}
				assert.Equal(t, "status update response lost", finalized[3].StatusMetadata["message"])
				for _, before := range []*model.Task{sibling, completedChild} {
					after, err := tasks.GetTask(before.UUID)
					require.NoError(t, err)
					assert.Equal(t, before, after)
				}

				reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
				for _, before := range finalized {
					after, err := tasks.GetTask(before.UUID)
					require.NoError(t, err)
					assert.Equal(t, before, after, "repeated reports must preserve task history and metadata")
				}
			})
		}
	}
}

func TestDriverRetryFinalizationPropagatesExistingFailureToSuccessfulAncestor(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, apiv2beta1.PipelineTask_CACHED, util.StringPointer("0"))
	child := createDriverRetryFinalizationTask(t, tasks, "failed-child", parent, apiv2beta1.PipelineTask_FAILED, util.StringPointer("0"))
	child.FinishedInSec = 99
	child, err := tasks.UpdateTask(child)
	require.NoError(t, err)

	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	parentAfter, err := tasks.GetTask(parent.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), parentAfter.State)
	require.Len(t, parentAfter.StateHistory, len(parent.StateHistory)+1)
	childAfter, err := tasks.GetTask(child.UUID)
	require.NoError(t, err)
	assert.Equal(t, child, childAfter, "an already-failed task keeps its original finish time and history")
}

func TestDriverRetryFinalizationRequiresMatchingGeneration(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, apiv2beta1.PipelineTask_RUNNING, nil)
	changeStoredGeneration := func(task *model.Task, generation string) *model.Task {
		task.StatusMetadata["customProperties"].(map[string]interface{})[util.DriverRetryGenerationKey] = generation
		metadata, err := json.Marshal(task.StatusMetadata)
		require.NoError(t, err)
		_, err = db.Exec("UPDATE tasks SET StatusMetadata = ? WHERE UUID = ?", string(metadata), task.UUID)
		require.NoError(t, err)
		stored, err := tasks.GetTask(task.UUID)
		require.NoError(t, err)
		return stored
	}
	stale := createDriverRetryFinalizationTask(t, tasks, "stale", parent, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
	stale = changeStoredGeneration(stale, "1")
	invalid := createDriverRetryFinalizationTask(t, tasks, "invalid", parent, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
	invalid = changeStoredGeneration(invalid, "invalid")
	otherRun := createDriverRetryFinalizationTask(t, tasks, "other-run", nil, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
	_, err := db.Exec("UPDATE tasks SET RunUUID = ? WHERE UUID = ?", "run-2", otherRun.UUID)
	require.NoError(t, err)
	otherRun, err = tasks.GetTask(otherRun.UUID)
	require.NoError(t, err)

	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	for _, before := range []*model.Task{parent, stale, invalid, otherRun} {
		after, err := tasks.GetTask(before.UUID)
		require.NoError(t, err)
		assert.Equal(t, before, after)
	}
}

func TestDriverRetryFinalizationRequiresUnsuccessfulTerminalReport(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStateRunning, model.RuntimeStatePending, model.RuntimeStateCancelling, model.RuntimeStateSucceeded, model.RuntimeStateSkipped} {
		t.Run(string(state), func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			child := createDriverRetryFinalizationTask(t, tasks, "child", nil, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
			reportDriverRetryTerminalRun(t, runs, state)
			after, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, child, after)
		})
	}
}

func TestDriverRetryFinalizationDoesNotApplyToOrdinaryRunUpdates(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	child := createDriverRetryFinalizationTask(t, tasks, "child", nil, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
	run, err := runs.GetRun("run-1", false)
	require.NoError(t, err)
	run.State = model.RuntimeStateFailed
	run.FinishedAtInSec = 123
	require.NoError(t, runs.UpdateRun(run))
	after, err := tasks.GetTask(child.UUID)
	require.NoError(t, err)
	assert.Equal(t, child, after)
}

func TestDriverRetryFinalizationRejectsStaleReport(t *testing.T) {
	for _, change := range []string{"generation", "manifest"} {
		t.Run(change, func(t *testing.T) {
			db, tasks, runs := initializeTaskStore()
			defer db.Close()
			child := createDriverRetryFinalizationTask(t, tasks, "child", nil, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
			run, err := runs.GetRun("run-1", false)
			require.NoError(t, err)
			expectedWorkflow, expectedPipeline := run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest
			if change == "generation" {
				_, err = db.Exec("UPDATE run_details SET RetryGeneration = 1 WHERE UUID = ?", run.UUID)
			} else {
				_, err = db.Exec("UPDATE run_details SET WorkflowRuntimeManifest = ? WHERE UUID = ?", "new-workflow", run.UUID)
			}
			require.NoError(t, err)
			run.State = model.RuntimeStateFailed
			run.FinishedAtInSec = 123
			updated, err := runs.UpdateRunIfRuntimeManifestsUnchanged(run, expectedWorkflow, expectedPipeline)
			require.NoError(t, err)
			assert.False(t, updated)
			after, err := tasks.GetTask(child.UUID)
			require.NoError(t, err)
			assert.Equal(t, child, after)
		})
	}
}

func TestDriverRetryFinalizationRollsBackRunAndTaskWrites(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, apiv2beta1.PipelineTask_RUNNING, nil)
	child := createDriverRetryFinalizationTask(t, tasks, "child", parent, apiv2beta1.PipelineTask_RUNNING, util.StringPointer("0"))
	// The leaf is written before its parent, so this failure must roll back
	// both a completed task write and the already-executed run update.
	_, err := db.Exec(fmt.Sprintf(`CREATE TRIGGER reject_driver_parent_update BEFORE UPDATE ON tasks
		WHEN OLD.UUID = '%s' BEGIN SELECT RAISE(ABORT, 'injected parent update failure'); END`, parent.UUID))
	require.NoError(t, err)
	before, err := runs.GetRun("run-1", false)
	require.NoError(t, err)
	run, err := runs.GetRun("run-1", false)
	require.NoError(t, err)
	run.State = model.RuntimeStateFailed
	run.FinishedAtInSec = 123
	run.WorkflowRuntimeManifest = "terminal-workflow"
	updated, err := runs.UpdateRunIfRuntimeManifestsUnchanged(run, before.WorkflowRuntimeManifest, before.PipelineRuntimeManifest)
	require.ErrorContains(t, err, "injected parent update failure")
	assert.False(t, updated)
	after, err := runs.GetRun("run-1", false)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	for _, task := range []*model.Task{parent, child} {
		after, err := tasks.GetTask(task.UUID)
		require.NoError(t, err)
		assert.Equal(t, task, after)
	}
}

func TestDriverRetryFinalizationRejectsQueuedAncestorUpdate(t *testing.T) {
	db, tasks, runs := initializeTaskStore()
	defer db.Close()
	parent := createDriverRetryFinalizationTask(t, tasks, "parent", nil, apiv2beta1.PipelineTask_RUNNING, nil)
	_ = createDriverRetryFinalizationTask(t, tasks, "cached-child", parent, apiv2beta1.PipelineTask_CACHED, util.StringPointer("0"))
	queuedParent := &apiv2beta1.PipelineTask{StatusMetadata: &apiv2beta1.PipelineTask_StatusMetadata{Message: "original parent metadata"}}
	origin := &apiv2beta1.PipelineTask{StatusMetadata: &apiv2beta1.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
		util.DriverRetryGenerationKey: structpb.NewStringValue("0"),
	}}}
	util.CopyDriverRetryGeneration(queuedParent, origin)
	metadata, err := model.ProtoMessageToJSONData(queuedParent.GetStatusMetadata())
	require.NoError(t, err)
	queuedUpdate := &model.Task{
		UUID: parent.UUID, RunUUID: parent.RunUUID,
		State: model.TaskStatus(apiv2beta1.PipelineTask_CACHED), StatusMetadata: metadata,
	}
	reportDriverRetryTerminalRun(t, runs, model.RuntimeStateFailed)
	_, err = tasks.UpdateTask(queuedUpdate)
	require.ErrorContains(t, err, "has finished")
	after, err := tasks.GetTask(parent.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_FAILED), after.State)
	assert.Equal(t, int64(123), after.FinishedInSec)
}
