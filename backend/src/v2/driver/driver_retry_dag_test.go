// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func newDAGRetryTestContext(t *testing.T) (*TestContext, common.Options) {
	t.Helper()
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_naming_conflicts.yaml")
	require.NoError(t, tc.Push("pipeline-b"))
	task := proto.Clone(tc.GetLast().GetTaskSpec()).(*pipelinespec.PipelineTaskSpec)
	task.Inputs = &pipelinespec.TaskInputsSpec{}
	opts := tc.setupDagOptions(tc.RootTask, task, nil)
	opts.DriverRetryEnabled = true
	opts.DriverRetryMaxCount = 2
	tc.ClientManager = client_manager.NewFakeClientManager(tc.ClientManager.K8sClient(), &driverRetryFaultAPI{MockAPI: tc.MockAPI})
	return tc, opts
}

func dagRetryLoopTask(task *pipelinespec.PipelineTaskSpec) *pipelinespec.PipelineTaskSpec {
	task = proto.Clone(task).(*pipelinespec.PipelineTaskSpec)
	task.Iterator = &pipelinespec.PipelineTaskSpec_ParameterIterator{
		ParameterIterator: &pipelinespec.ParameterIteratorSpec{
			ItemInput: "item",
			Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
				Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "[8,10,12]"},
			},
		},
	}
	return task
}

func getOnlyDAGRetryTask(t *testing.T, tc *TestContext, opts common.Options) *api.PipelineTask {
	t.Helper()
	view := api.GetRunRequest_FULL
	run, err := tc.MockAPI.GetRun(context.Background(), &api.GetRunRequest{RunId: tc.Run.GetRunId(), View: &view})
	require.NoError(t, err)
	var tasks []*api.PipelineTask
	for _, task := range run.GetTasks() {
		if task.GetName() == opts.TaskName && task.GetParentTaskId() == opts.ParentTask.GetTaskId() {
			tasks = append(tasks, task)
		}
	}
	require.Len(t, tasks, 1, "driver attempts must share one logical DAG task")
	return tasks[0]
}

func assertDAGRetryParentRunning(t *testing.T, tc *TestContext) {
	t.Helper()
	parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.GetRunId(), TaskId: tc.RootTask.GetTaskId()})
	require.NoError(t, err)
	assert.Equal(t, api.PipelineTask_RUNNING, parent.GetState())
	assert.Nil(t, parent.GetEndTime())
}

func TestDAGDriverRetryPreservesLoopIdentityAndCheckpoint(t *testing.T) {
	for _, outerIndex := range []int{-1, 1} {
		t.Run(fmt.Sprint(outerIndex), func(t *testing.T) {
			tc, opts := newDAGRetryTestContext(t)
			opts.Task = dagRetryLoopTask(opts.Task)
			opts.IterationIndex = outerIndex
			first, err := DAG(context.Background(), opts, tc.ClientManager)
			require.NoError(t, err)
			require.NotNil(t, first.IterationCount)
			assert.Equal(t, 3, *first.IterationCount)
			task := getOnlyDAGRetryTask(t, tc, opts)
			assert.Equal(t, first.TaskID, task.GetTaskId())
			assert.Equal(t, api.PipelineTask_LOOP, task.GetType())
			assert.Equal(t, api.PipelineTask_RUNNING, task.GetState())
			require.NotNil(t, task.GetTypeAttributes())
			require.NotNil(t, task.GetTypeAttributes().IterationCount)
			assert.EqualValues(t, 3, task.GetTypeAttributes().GetIterationCount())
			if outerIndex >= 0 {
				require.NotNil(t, task.GetTypeAttributes().IterationIndex)
				assert.EqualValues(t, outerIndex, task.GetTypeAttributes().GetIterationIndex())
			} else {
				assert.Nil(t, task.GetTypeAttributes().IterationIndex)
			}
			var items []float64
			for _, parameter := range task.GetInputs().GetParameters() {
				if parameter.GetType() == api.IOType_ITERATOR_INPUT {
					items = append(items, parameter.GetValue().GetNumberValue())
				}
			}
			assert.Equal(t, []float64{8, 10, 12}, items)
			assert.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
			assertDAGRetryParentRunning(t, tc)

			// A completed handoff must replay without resolving inputs again.
			opts.Task = withBrokenParameterIterator(opts.Task)
			opts.DriverRetryAttempt = 1
			opts.PodUID = "checkpoint-replay-pod"
			replayed, err := DAG(context.Background(), opts, tc.ClientManager)
			require.NoError(t, err)
			assert.Equal(t, first.TaskID, replayed.TaskID)
			assert.Equal(t, first.IterationCount, replayed.IterationCount)
			assert.Equal(t, first.Condition, replayed.Condition)
			assert.True(t, proto.Equal(first.ExecutorInput, replayed.ExecutorInput))
			after := getOnlyDAGRetryTask(t, tc, opts)
			assert.Equal(t, task.GetType(), after.GetType())
			assert.Equal(t, task.GetState(), after.GetState())
			assert.True(t, proto.Equal(task.GetTypeAttributes(), after.GetTypeAttributes()))
			assert.True(t, proto.Equal(task.GetInputs(), after.GetInputs()))
			expectedMetadata := proto.Clone(task.GetStatusMetadata()).(*api.PipelineTask_StatusMetadata)
			expectedMetadata.CustomProperties[util.DriverRetryAttemptKey] = structpb.NewStringValue("1")
			assert.True(t, proto.Equal(expectedMetadata, after.GetStatusMetadata()))
			require.Len(t, after.GetPods(), len(task.GetPods())+1)
			assert.Equal(t, opts.PodUID, after.GetPods()[len(after.GetPods())-1].GetUid())
		})
	}
}

func TestDAGDriverRetryRecoversPreResolutionFailure(t *testing.T) {
	tc, opts := newDAGRetryTestContext(t)
	validTask := dagRetryLoopTask(opts.Task)
	opts.Task = withBrokenParameterIterator(validTask)
	_, err := DAG(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "error unmarshall raw string")
	failedAttempt := getOnlyDAGRetryTask(t, tc, opts)
	assert.Equal(t, api.PipelineTask_LOOP, failedAttempt.GetType())
	assert.Equal(t, api.PipelineTask_RUNNING, failedAttempt.GetState())
	assert.Nil(t, failedAttempt.GetEndTime())
	assert.Contains(t, failedAttempt.GetStatusMetadata().GetMessage(), "error unmarshall raw string")
	assert.NotContains(t, failedAttempt.GetStatusMetadata().GetCustomProperties(), driverCheckpointKey)
	assertDAGRetryParentRunning(t, tc)

	opts.Task = validTask
	opts.DriverRetryAttempt = 1
	opts.PodUID = "corrected-driver-pod"
	recovered, err := DAG(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, failedAttempt.GetTaskId(), recovered.TaskID)
	require.NotNil(t, recovered.IterationCount)
	assert.Equal(t, 3, *recovered.IterationCount)
	task := getOnlyDAGRetryTask(t, tc, opts)
	assert.Equal(t, api.PipelineTask_LOOP, task.GetType())
	assert.EqualValues(t, 3, task.GetTypeAttributes().GetIterationCount())
	assert.Empty(t, task.GetStatusMetadata().GetMessage())
	assert.Equal(t, "0", task.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
	assertDAGRetryParentRunning(t, tc)
}

type dagRetryCheckpointFailureAPI struct {
	kfpapi.API
	parentCheckpoint string
	failCheckpoint   bool
}

func (a *dagRetryCheckpointFailureAPI) UpdateTask(ctx context.Context, request *api.UpdateTaskRequest) (*api.PipelineTask, error) {
	checkpoint := request.GetTask().GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue()
	if a.failCheckpoint && checkpoint != "" && checkpoint != a.parentCheckpoint {
		a.failCheckpoint = false
		return nil, errors.New("checkpoint persistence unavailable")
	}
	return a.API.UpdateTask(ctx, request)
}

func TestDAGDriverRetryDoesNotInheritParentCheckpoint(t *testing.T) {
	tc, opts := newDAGRetryTestContext(t)
	opts.IterationIndex = 1
	parentCheckpoint, err := marshalDriverCheckpoint(&Execution{TaskID: tc.RootTask.GetTaskId()})
	require.NoError(t, err)
	parent := proto.Clone(tc.RootTask).(*api.PipelineTask)
	parent.StatusMetadata = &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
		util.DriverRetryGenerationKey: structpb.NewStringValue("37"),
		driverCheckpointKey:           structpb.NewStringValue(parentCheckpoint),
		driverCachedOutputsKey:        structpb.NewStringValue("parent cache state"),
		"plugins.mlflow.run_id":       structpb.NewStringValue("parent-plugin-run"),
	}}
	_, err = tc.MockAPI.UpdateTask(context.Background(), &api.UpdateTaskRequest{RunId: tc.Run.GetRunId(), TaskId: parent.GetTaskId(), Task: parent})
	require.NoError(t, err)
	opts.ParentTask = parent
	failingAPI := &dagRetryCheckpointFailureAPI{API: tc.ClientManager.KFPAPIClient(), parentCheckpoint: parentCheckpoint, failCheckpoint: true}
	manager := client_manager.NewFakeClientManager(tc.ClientManager.K8sClient(), failingAPI)

	_, err = DAG(context.Background(), opts, manager)
	require.ErrorContains(t, err, "checkpoint persistence unavailable")
	task := getOnlyDAGRetryTask(t, tc, opts)
	assert.Equal(t, api.PipelineTask_DAG, task.GetType())
	assert.EqualValues(t, 1, task.GetTypeAttributes().GetIterationIndex())
	properties := task.GetStatusMetadata().GetCustomProperties()
	assert.Equal(t, "0", properties[util.DriverRetryGenerationKey].GetStringValue())
	assert.NotContains(t, properties, driverCheckpointKey, "an interrupted child handoff must not leave its parent's checkpoint")
	assert.NotContains(t, properties, driverCachedOutputsKey)
	assert.Equal(t, "parent-plugin-run", properties["plugins.mlflow.run_id"].GetStringValue())

	opts.DriverRetryAttempt = 1
	opts.PodUID = "nested-retry-pod"
	recovered, err := DAG(context.Background(), opts, manager)
	require.NoError(t, err, "the retry must recover the child, not restore its parent's handoff")
	assert.Equal(t, task.GetTaskId(), recovered.TaskID)
	stored := getOnlyDAGRetryTask(t, tc, opts)
	saved, err := restoreDriverCheckpoint(stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue(), stored.GetTaskId())
	require.NoError(t, err)
	assert.Equal(t, recovered.TaskID, saved.TaskID)
	assert.NotContains(t, stored.GetStatusMetadata().GetCustomProperties(), driverCachedOutputsKey)
	storedParent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.GetRunId(), TaskId: parent.GetTaskId()})
	require.NoError(t, err)
	assert.True(t, proto.Equal(parent.GetStatusMetadata(), storedParent.GetStatusMetadata()))
}
