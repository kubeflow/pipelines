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
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Clone requests/responses so the mock cannot persist mutations that a real
// transport would not send. Failures can occur after a committed write.
type driverRetryFaultAPI struct {
	*kfpapi.MockAPI
	createCalls    int
	loseCreate     int
	failCreate     int
	loseCheckpoint bool
	loseCached     bool
	cacheCalls     int
	cacheTask      *api.PipelineTask
	cacheErr       error
}

func (f *driverRetryFaultAPI) CreateTask(ctx context.Context, req *api.CreateTaskRequest) (*api.PipelineTask, error) {
	f.createCalls++
	if f.createCalls == f.failCreate {
		return nil, fmt.Errorf("injected create failure")
	}
	task, err := f.MockAPI.CreateTask(ctx, proto.Clone(req).(*api.CreateTaskRequest))
	if err != nil {
		return nil, err
	}
	if f.createCalls == f.loseCreate {
		return nil, fmt.Errorf("lost create response")
	}
	return proto.Clone(task).(*api.PipelineTask), nil
}
func (f *driverRetryFaultAPI) UpdateTask(ctx context.Context, req *api.UpdateTaskRequest) (*api.PipelineTask, error) {
	task, err := f.MockAPI.UpdateTask(ctx, proto.Clone(req).(*api.UpdateTaskRequest))
	if err != nil {
		return nil, err
	}
	if f.loseCheckpoint && task.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue() != "" {
		f.loseCheckpoint = false
		return nil, fmt.Errorf("lost checkpoint response")
	}
	if f.loseCached && task.GetState() == api.PipelineTask_CACHED {
		f.loseCached = false
		return nil, fmt.Errorf("lost cached task response")
	}
	return proto.Clone(task).(*api.PipelineTask), nil
}
func (f *driverRetryFaultAPI) FindCachedTask(ctx context.Context, req *api.FindCachedTaskRequest) (*api.FindCachedTaskResponse, error) {
	f.cacheCalls++
	if f.cacheErr != nil {
		return nil, f.cacheErr
	}
	if f.cacheTask != nil {
		return &api.FindCachedTaskResponse{Task: proto.Clone(f.cacheTask).(*api.PipelineTask)}, nil
	}
	return f.MockAPI.FindCachedTask(ctx, req)
}

func retryContainerContext(t *testing.T) (*TestContext, common.Options, *driverRetryFaultAPI) {
	t.Helper()
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/cache_test.yaml")
	require.NoError(t, tc.Push("create-dataset"))
	opts := tc.setupContainerOptions(tc.RootTask, tc.GetLast().GetTaskSpec(), nil)
	opts.DriverRetryEnabled = true
	opts.DriverRetryMaxCount = 2
	opts.CacheDisabled = true
	faults := &driverRetryFaultAPI{MockAPI: tc.MockAPI}
	tc.ClientManager = clientmanager.NewFakeClientManager(tc.ClientManager.K8sClient(), faults)
	return tc, opts, faults
}

func TestDriverRetryContainerRecoversLostCreate(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	faults.loseCreate = 1
	_, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost create response")
	opts.DriverRetryAttempt++
	execution, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	require.NotEmpty(t, execution.TaskID)
	tasks, err := tc.MockAPI.ListTasks(context.Background(), &api.ListTasksRequest{RunId: tc.Run.RunId})
	require.NoError(t, err)
	require.Len(t, tasks.GetTasks(), 2, "one root and one logical container task")
}

func TestDriverRetryContainerKeepsOutputAllocationAndPodHistory(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	faults.failCreate = 2 // after output allocation, before task handoff
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "injected create failure")
	require.NotNil(t, first)
	require.NotNil(t, first.ExecutorInput)
	opts.DriverRetryAttempt++
	opts.PodUID = "second-attempt"
	second, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	require.True(t, proto.Equal(first.ExecutorInput.GetOutputs(), second.ExecutorInput.GetOutputs()))
	task, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: second.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	assert.Empty(t, task.GetStatusMetadata().GetMessage())
	assert.Nil(t, task.EndTime)
	require.Len(t, task.GetPods(), 2)
	opts.DriverRetryAttempt++
	opts.PodUID = "checkpoint-replay"
	replay, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, second.TaskID, replay.TaskID)
	assert.True(t, proto.Equal(second.ExecutorInput, replay.ExecutorInput))
	task, err = tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: second.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	assert.Len(t, task.GetPods(), 3)
}

func TestDriverRetryContainerRecoversLostCheckpointResponse(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	opts.CacheDisabled = false
	faults.loseCheckpoint = true
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost checkpoint response")
	calls := faults.cacheCalls
	faults.cacheErr = fmt.Errorf("cache unavailable after handoff")
	opts.DriverRetryAttempt++
	second, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, first.TaskID, second.TaskID)
	assert.True(t, proto.Equal(first.ExecutorInput, second.ExecutorInput))
	assert.Equal(t, first.PodSpecPatch, second.PodSpecPatch)
	assert.Equal(t, calls, faults.cacheCalls, "saved handoff must bypass cache and driver side effects")
}

func TestDriverRetryContainerFreezesPartialCacheHit(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	opts.CacheDisabled = false
	opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	faults.cacheTask = &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{ParameterKey: "value", Value: structpb.NewStringValue("original"), Type: api.IOType_OUTPUT}}}}
	faults.loseCached = true
	_, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost cached task response")
	calls := faults.cacheCalls
	faults.cacheErr = fmt.Errorf("cache no longer available")
	opts.DriverRetryAttempt++
	execution, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	require.True(t, *execution.Cached)
	assert.Equal(t, calls, faults.cacheCalls)
	task, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: execution.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	assert.Equal(t, "original", task.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	assert.Equal(t, api.PipelineTask_CACHED, task.State)
	assert.Empty(t, task.GetStatusMetadata().GetMessage())
}

func TestDriverRetryContainerPreResolutionFailureRecovers(t *testing.T) {
	tc, opts, _ := retryContainerContext(t)
	validTask := opts.Task
	opts.Task = withBrokenParameterIterator(validTask)
	_, err := Container(context.Background(), opts, tc.ClientManager)
	require.Error(t, err)
	tasks, err := tc.MockAPI.ListTasks(context.Background(), &api.ListTasksRequest{RunId: tc.Run.RunId})
	require.NoError(t, err)
	var failedAttempt *api.PipelineTask
	for _, task := range tasks.Tasks {
		assert.Equal(t, api.PipelineTask_RUNNING, task.State)
		if task.Name == opts.TaskName {
			failedAttempt = task
		}
	}
	require.NotNil(t, failedAttempt)
	require.NotEmpty(t, failedAttempt.GetStatusMetadata().GetMessage())
	opts.Task = validTask
	opts.DriverRetryAttempt++
	execution, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, failedAttempt.TaskId, execution.TaskID)
}

func TestDriverRetryOutputIdentitySeparatesManualRetriesAndParents(t *testing.T) {
	_, opts, _ := retryContainerContext(t)
	id := driverOutputAllocationID(opts)
	opts.DriverRetryAttempt++
	assert.Equal(t, id, driverOutputAllocationID(opts))
	opts.DriverRetryGeneration++
	assert.NotEqual(t, id, driverOutputAllocationID(opts))
	opts.DriverRetryGeneration--
	opts.ParentTask = proto.Clone(opts.ParentTask).(*api.PipelineTask)
	opts.ParentTask.TaskId = "another-outer-iteration"
	assert.NotEqual(t, id, driverOutputAllocationID(opts))
}

type unsupportedRetryDispatcher struct{ plugins.NoOpDispatcher }

func TestDriverRetryRejectsUnsafeRuntimeOptions(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*common.Options)
		want   string
	}{
		{"plugins", func(o *common.Options) { o.PluginDispatcher = unsupportedRetryDispatcher{} }, "plugin"},
		{"native PVC", func(o *common.Options) {
			o.Container = proto.Clone(o.Container).(*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec)
			o.Container.Image = "argostub/createpvc"
		}, "Kubernetes platform"},
		{"attempt", func(o *common.Options) { o.DriverRetryAttempt = 3 }, "invalid driver retry"},
		{"generation", func(o *common.Options) { o.DriverRetryGeneration = -1 }, "invalid driver retry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tc, opts, faults := retryContainerContext(t)
			test.change(&opts)
			_, err := Container(context.Background(), opts, tc.ClientManager)
			require.ErrorContains(t, err, test.want)
			assert.Zero(t, faults.createCalls)
		})
	}
}

func TestDriverRetryCheckpointValidation(t *testing.T) {
	for _, encoded := range []string{"bad json", `{"version":2,"taskId":"task"}`, `{"version":1,"taskId":"different"}`, `{"version":1,"taskId":"task","executorInput":{"notAField":true}}`} {
		_, err := restoreDriverCheckpoint(encoded, "task")
		require.Error(t, err)
	}
	execution := &Execution{TaskID: "task", IterationCount: util.IntPointer(0), Condition: util.BoolPointer(false), Cached: util.BoolPointer(true)}
	encoded, err := marshalDriverCheckpoint(execution)
	require.NoError(t, err)
	restored, err := restoreDriverCheckpoint(encoded, "task")
	require.NoError(t, err)
	assert.Equal(t, execution, restored)
}
