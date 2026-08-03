// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// withBrokenParameterIterator clones taskSpec and replaces its iterator with a
// ParameterIterator whose Raw items are invalid JSON, forcing ResolveInputs to
// fail while still exposing a ParameterIterator for type inference.
func withBrokenParameterIterator(taskSpec *pipelinespec.PipelineTaskSpec) *pipelinespec.PipelineTaskSpec {
	cloned := proto.Clone(taskSpec).(*pipelinespec.PipelineTaskSpec)
	cloned.Iterator = &pipelinespec.PipelineTaskSpec_ParameterIterator{
		ParameterIterator: &pipelinespec.ParameterIteratorSpec{
			ItemInput: "item",
			Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
				Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "not-json"},
			},
		},
	}
	return cloned
}

func TestDAG_RetryPreResolutionFailureFinalizesExistingTask(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_raw_Iterator.yaml",
	)

	_, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", tc.RootTask)
	createDataSetExecution, _ := tc.RunContainerDriver("create-dataset", secondaryPipelineTask, nil, false)
	tc.RunLauncher(createDataSetExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)

	_, loopTask := tc.RunDagDriver("for-loop-2", secondaryPipelineTask)
	require.Equal(t, apiv2beta1.PipelineTask_LOOP, loopTask.Type)
	loopTaskID := loopTask.GetTaskId()

	// RunDagDriver leaves the loop scope pushed; pop before retrying the same task.
	tc.ExitDag()

	loopTask.State = apiv2beta1.PipelineTask_RUNNING
	loopTask.EndTime = nil
	loopTask.StatusMetadata = nil
	_, err := tc.ClientManager.KFPAPIClient().UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: loopTaskID,
		Task:   loopTask,
		RunId:  tc.Run.GetRunId(),
	})
	require.NoError(t, err)

	tc.RefreshRun()
	err = tc.Push("for-loop-2")
	require.NoError(t, err)
	defer tc.ExitDag()

	taskSpec := tc.GetLast().GetTaskSpec()
	opts := tc.setupDagOptions(secondaryPipelineTask, taskSpec, nil)
	opts.Task = withBrokenParameterIterator(taskSpec)

	_, err = DAG(context.Background(), opts, tc.ClientManager)
	require.Error(t, err)

	tc.RefreshRun()
	fullView := apiv2beta1.GetRunRequest_FULL
	run, err := tc.ClientManager.KFPAPIClient().GetRun(context.Background(), &apiv2beta1.GetRunRequest{
		RunId: tc.Run.GetRunId(),
		View:  &fullView,
	})
	require.NoError(t, err)

	var matchingTasks []*apiv2beta1.PipelineTask
	for _, task := range run.GetTasks() {
		if task.GetName() == "for-loop-2" && task.GetParentTaskId() == secondaryPipelineTask.GetTaskId() {
			matchingTasks = append(matchingTasks, task)
		}
	}
	require.Len(t, matchingTasks, 1, "expected exactly one loop task")

	finalizedTask := matchingTasks[0]
	assert.Equal(t, loopTaskID, finalizedTask.GetTaskId())
	assert.Equal(t, apiv2beta1.PipelineTask_LOOP, finalizedTask.GetType())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, finalizedTask.GetState())
	assert.NotNil(t, finalizedTask.GetEndTime())
	assert.NotEmpty(t, finalizedTask.GetStatusMetadata().GetMessage())

	var secondaryTask *apiv2beta1.PipelineTask
	for _, task := range run.GetTasks() {
		if task.GetName() == "secondary-pipeline" {
			secondaryTask = task
			break
		}
	}
	require.NotNil(t, secondaryTask)
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, secondaryTask.GetState(),
		"parent task should be FAILED after child failure propagation")
}

func TestContainer_RetryPreResolutionFailureFinalizesExistingTask(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/cache_test.yaml",
	)

	execution, containerTask := tc.RunContainerDriver("create-dataset", tc.RootTask, nil, true)
	require.NotNil(t, execution)
	require.Equal(t, apiv2beta1.PipelineTask_RUNTIME, containerTask.Type)
	containerTaskID := containerTask.GetTaskId()

	containerTask.State = apiv2beta1.PipelineTask_RUNNING
	containerTask.EndTime = nil
	containerTask.StatusMetadata = nil
	_, err := tc.ClientManager.KFPAPIClient().UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: containerTaskID,
		Task:   containerTask,
		RunId:  tc.Run.GetRunId(),
	})
	require.NoError(t, err)

	tc.RefreshRun()
	err = tc.Push("create-dataset")
	require.NoError(t, err)
	defer func() {
		_, ok := tc.Pop()
		require.True(t, ok)
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	require.NoError(t, err)
	opts := tc.setupContainerOptions(tc.RootTask, taskSpec, kubernetesExecutorConfig)
	opts.Task = withBrokenParameterIterator(taskSpec)

	_, err = Container(context.Background(), opts, tc.ClientManager)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshall raw string")

	finalizedTask, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: containerTaskID,
		RunId:  tc.Run.GetRunId(),
	})
	require.NoError(t, err)
	assert.Equal(t, containerTaskID, finalizedTask.GetTaskId())
	assert.Equal(t, apiv2beta1.PipelineTask_RUNTIME, finalizedTask.GetType())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, finalizedTask.GetState())
	assert.NotNil(t, finalizedTask.GetEndTime())
	assert.NotEmpty(t, finalizedTask.GetStatusMetadata().GetMessage())
}

func TestContainer_RetryPreResolutionFailureWithIterationIndex(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_raw_Iterator.yaml",
	)

	_, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", tc.RootTask)
	createDataSetExecution, _ := tc.RunContainerDriver("create-dataset", secondaryPipelineTask, nil, false)
	tc.RunLauncher(createDataSetExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)

	_, loopTask := tc.RunDagDriver("for-loop-2", secondaryPipelineTask)
	require.Equal(t, apiv2beta1.PipelineTask_LOOP, loopTask.Type)

	iterIdx := int64(0)
	processExecution, processTask := tc.RunContainerDriver("process-dataset", loopTask, &iterIdx, true)
	require.NotNil(t, processExecution)
	processTaskID := processTask.GetTaskId()

	processTask.State = apiv2beta1.PipelineTask_RUNNING
	processTask.EndTime = nil
	processTask.StatusMetadata = nil
	_, err := tc.ClientManager.KFPAPIClient().UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: processTaskID,
		Task:   processTask,
		RunId:  tc.Run.GetRunId(),
	})
	require.NoError(t, err)

	tc.RefreshRun()
	err = tc.Push("process-dataset")
	require.NoError(t, err)
	defer func() {
		_, ok := tc.Pop()
		require.True(t, ok)
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	require.NoError(t, err)
	opts := tc.setupContainerOptions(loopTask, taskSpec, kubernetesExecutorConfig)
	opts.IterationIndex = 0
	opts.Task = withBrokenParameterIterator(taskSpec)

	_, err = Container(context.Background(), opts, tc.ClientManager)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshall raw string")

	finalizedTask, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: processTaskID,
		RunId:  tc.Run.GetRunId(),
	})
	require.NoError(t, err)
	assert.Equal(t, processTaskID, finalizedTask.GetTaskId())
	assert.Equal(t, apiv2beta1.PipelineTask_RUNTIME, finalizedTask.GetType())
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, finalizedTask.GetState())
	assert.NotNil(t, finalizedTask.GetEndTime())
	assert.NotEmpty(t, finalizedTask.GetStatusMetadata().GetMessage())
	require.NotNil(t, finalizedTask.GetTypeAttributes())
	assert.Equal(t, int64(0), finalizedTask.GetTypeAttributes().GetIterationIndex())
}

func TestApplyInferredDAGTaskType(t *testing.T) {
	tests := []struct {
		name         string
		opts         common.Options
		expectedType apiv2beta1.PipelineTask_TaskType
	}{
		{
			name: "LOOP - has ParameterIterator and IterationIndex < 0",
			opts: common.Options{
				Task: &pipelinespec.PipelineTaskSpec{
					Iterator: &pipelinespec.PipelineTaskSpec_ParameterIterator{
						ParameterIterator: &pipelinespec.ParameterIteratorSpec{
							ItemInput: "item",
							Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
								Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "[1,2,3]"},
							},
						},
					},
				},
				IterationIndex: -1,
			},
			expectedType: apiv2beta1.PipelineTask_LOOP,
		},
		{
			name: "DAG default - has ParameterIterator but IterationIndex >= 0 (iteration body)",
			opts: common.Options{
				Task: &pipelinespec.PipelineTaskSpec{
					Iterator: &pipelinespec.PipelineTaskSpec_ParameterIterator{
						ParameterIterator: &pipelinespec.ParameterIteratorSpec{
							ItemInput: "item",
							Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
								Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "[1,2,3]"},
							},
						},
					},
				},
				IterationIndex: 0,
			},
			expectedType: apiv2beta1.PipelineTask_DAG,
		},
		{
			name: "CONDITION_BRANCH - has trigger condition",
			opts: common.Options{
				Task: &pipelinespec.PipelineTaskSpec{
					TriggerPolicy: &pipelinespec.PipelineTaskSpec_TriggerPolicy{
						Condition: "inputs.parameters['flag'] == 'true'",
					},
				},
				IterationIndex: -1,
			},
			expectedType: apiv2beta1.PipelineTask_CONDITION_BRANCH,
		},
		{
			name: "CONDITION - name prefix condition",
			opts: common.Options{
				Task:           &pipelinespec.PipelineTaskSpec{},
				TaskName:       "condition-1",
				IterationIndex: -1,
			},
			expectedType: apiv2beta1.PipelineTask_CONDITION,
		},
		{
			name: "DAG default - name prefix condition-branch does not match CONDITION",
			opts: common.Options{
				Task:           &pipelinespec.PipelineTaskSpec{},
				TaskName:       "condition-branch-1",
				IterationIndex: -1,
			},
			expectedType: apiv2beta1.PipelineTask_DAG,
		},
		{
			name: "DAG default - no special conditions",
			opts: common.Options{
				Task:           &pipelinespec.PipelineTaskSpec{},
				TaskName:       "my-subdag",
				IterationIndex: -1,
			},
			expectedType: apiv2beta1.PipelineTask_DAG,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := &apiv2beta1.PipelineTask{Type: apiv2beta1.PipelineTask_DAG}
			applyInferredDAGTaskType(test.opts, task)
			assert.Equal(t, test.expectedType, task.GetType())
		})
	}
}

func TestApplyInferredDAGTaskType_NilTask(t *testing.T) {
	applyInferredDAGTaskType(common.Options{Task: &pipelinespec.PipelineTaskSpec{}, IterationIndex: -1}, nil)
}
