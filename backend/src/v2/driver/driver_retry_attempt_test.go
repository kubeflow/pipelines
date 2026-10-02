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
	"strconv"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// The general MockAPI deliberately lacks storage locking and ownership checks.
// This adapter models the claim/update contract and clones transport values so
// these tests exercise driver behavior without accidentally adopting an owner
// through shared pointers.
type driverAttemptAPI struct {
	*driverRetryFaultAPI
	generation              int64
	loseClaim               bool
	queueUpdate             bool
	queuedUpdate            *api.UpdateTaskRequest
	supersedeOnGet          bool
	supersedeUpdateResponse bool
}

func driverAttemptOwner(task *api.PipelineTask) (generation, attempt int64, err error) {
	properties := task.GetStatusMetadata().GetCustomProperties()
	generation, err = strconv.ParseInt(properties[util.DriverRetryGenerationKey].GetStringValue(), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("missing driver generation: %w", err)
	}
	attempt, err = strconv.ParseInt(properties[util.DriverRetryAttemptKey].GetStringValue(), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("missing driver attempt: %w", err)
	}
	return generation, attempt, nil
}

func (f *driverAttemptAPI) CreateTask(ctx context.Context, request *api.CreateTaskRequest) (*api.PipelineTask, error) {
	generation, attempt, err := driverAttemptOwner(request.GetTask())
	if err != nil {
		return nil, err
	}
	if generation != f.generation {
		return nil, fmt.Errorf("stale driver generation")
	}
	stored, err := f.MockAPI.CreateTask(ctx, proto.Clone(request).(*api.CreateTaskRequest))
	if err != nil {
		return nil, err
	}
	stored = proto.Clone(stored).(*api.PipelineTask)
	oldGeneration, oldAttempt, err := driverAttemptOwner(stored)
	if err != nil {
		return nil, err
	}
	preserved := stored.GetState() == api.PipelineTask_CACHED || stored.GetState() == api.PipelineTask_SUCCEEDED || stored.GetState() == api.PipelineTask_SKIPPED
	if oldGeneration > generation || (oldGeneration == generation && oldAttempt > attempt) || (oldGeneration < generation && !preserved) {
		return nil, fmt.Errorf("stale driver claim")
	}
	// Claiming updates only ownership; it must not replace cached results or a
	// completed handoff with the recovery request's skeleton.
	metadata := driverRecoveryMetadata(stored)
	metadata.CustomProperties[util.DriverRetryGenerationKey] = structpb.NewStringValue(strconv.FormatInt(generation, 10))
	metadata.CustomProperties[util.DriverRetryAttemptKey] = structpb.NewStringValue(strconv.FormatInt(attempt, 10))
	stored, err = f.MockAPI.UpdateTask(ctx, &api.UpdateTaskRequest{RunId: stored.RunId, TaskId: stored.TaskId, Task: stored})
	if err != nil {
		return nil, err
	}
	if f.loseClaim {
		f.loseClaim = false
		return nil, fmt.Errorf("lost claim response")
	}
	return proto.Clone(stored).(*api.PipelineTask), nil
}

func (f *driverAttemptAPI) GetTask(ctx context.Context, request *api.GetTaskRequest) (*api.PipelineTask, error) {
	stored, err := f.MockAPI.GetTask(ctx, request)
	if err != nil {
		return nil, err
	}
	if f.supersedeOnGet {
		f.supersedeOnGet = false
		return f.advanceOwner(ctx, stored)
	}
	return proto.Clone(stored).(*api.PipelineTask), nil
}

func (f *driverAttemptAPI) UpdateTask(ctx context.Context, request *api.UpdateTaskRequest) (*api.PipelineTask, error) {
	if f.queueUpdate {
		f.queueUpdate = false
		f.queuedUpdate = proto.Clone(request).(*api.UpdateTaskRequest)
		return nil, fmt.Errorf("transport failed before queued update committed")
	}
	stored, err := f.MockAPI.GetTask(ctx, &api.GetTaskRequest{RunId: request.RunId, TaskId: request.TaskId})
	if err != nil {
		return nil, err
	}
	generation, attempt, err := driverAttemptOwner(request.GetTask())
	if err != nil {
		return nil, err
	}
	storedGeneration, storedAttempt, err := driverAttemptOwner(stored)
	if err != nil {
		return nil, err
	}
	if generation != f.generation || generation != storedGeneration || attempt != storedAttempt {
		return nil, fmt.Errorf("stale driver update")
	}
	updated, err := f.driverRetryFaultAPI.UpdateTask(ctx, request)
	if err != nil {
		return nil, err
	}
	if f.supersedeUpdateResponse {
		f.supersedeUpdateResponse = false
		return f.advanceOwner(ctx, updated)
	}
	return proto.Clone(updated).(*api.PipelineTask), nil
}

func (f *driverAttemptAPI) advanceOwner(ctx context.Context, task *api.PipelineTask) (*api.PipelineTask, error) {
	task = proto.Clone(task).(*api.PipelineTask)
	_, attempt, err := driverAttemptOwner(task)
	if err != nil {
		return nil, err
	}
	checkpoint, err := marshalDriverCheckpoint(&Execution{TaskID: task.TaskId, Cached: util.BoolPointer(true)})
	if err != nil {
		return nil, err
	}
	metadata := driverRecoveryMetadata(task)
	metadata.CustomProperties[util.DriverRetryAttemptKey] = structpb.NewStringValue(strconv.FormatInt(attempt+1, 10))
	metadata.CustomProperties[driverCheckpointKey] = structpb.NewStringValue(checkpoint)
	metadata.Message = "owned by the newer attempt"
	task.State = api.PipelineTask_CACHED
	updated, err := f.MockAPI.UpdateTask(ctx, &api.UpdateTaskRequest{RunId: task.RunId, TaskId: task.TaskId, Task: task})
	if err != nil {
		return nil, err
	}
	return proto.Clone(updated).(*api.PipelineTask), nil
}

func retryAttemptContext(t *testing.T) (*TestContext, common.Options, *driverAttemptAPI) {
	t.Helper()
	tc, opts, faults := retryContainerContext(t)
	client := &driverAttemptAPI{driverRetryFaultAPI: faults}
	tc.ClientManager = client_manager.NewFakeClientManager(tc.ClientManager.K8sClient(), client)
	return tc, opts, client
}

func enableAttemptTestCache(opts *common.Options, client *driverAttemptAPI) {
	opts.CacheDisabled = false
	opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	client.cacheTask = &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{
		ParameterKey: "result", Value: structpb.NewStringValue("frozen cache result"), Type: api.IOType_OUTPUT,
	}}}}
}

func TestDriverRetryAttemptDelayedUpdatePreservesNewerHandoff(t *testing.T) {
	ctx := context.Background()
	tc, opts, client := retryAttemptContext(t)
	enableAttemptTestCache(&opts, client)
	client.queueUpdate = true
	_, err := Container(ctx, opts, tc.ClientManager)
	require.ErrorContains(t, err, "transport failed")
	require.NotNil(t, client.queuedUpdate)
	opts.DriverRetryAttempt = 1
	execution, err := Container(ctx, opts, tc.ClientManager)
	require.NoError(t, err)
	before, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_CACHED, before.State)
	require.NotEmpty(t, before.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
	require.NotEmpty(t, before.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())
	_, err = client.UpdateTask(ctx, client.queuedUpdate)
	require.ErrorContains(t, err, "stale driver update")
	after, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	require.True(t, proto.Equal(before, after), "rejected update must not change state, end time, cache, or checkpoint")
	client.cacheErr = fmt.Errorf("cache unavailable after handoff")
	opts.DriverRetryAttempt = 2
	replayed, err := Container(ctx, opts, tc.ClientManager)
	require.NoError(t, err)
	require.Equal(t, execution.TaskID, replayed.TaskID)
	require.Equal(t, execution.Cached, replayed.Cached)
	require.Equal(t, execution.PodSpecPatch, replayed.PodSpecPatch)
	require.True(t, proto.Equal(execution.ExecutorInput, replayed.ExecutorInput))
	require.Equal(t, 1, client.cacheCalls)
}

func TestDriverRetryAttemptLostClaimsAndHandoffDoNotRepeatWork(t *testing.T) {
	ctx := context.Background()
	tc, opts, client := retryAttemptContext(t)
	enableAttemptTestCache(&opts, client)
	client.loseClaim = true
	_, err := Container(ctx, opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost claim response")
	require.Zero(t, client.cacheCalls)
	// Repeating the same claim after a lost response is idempotent. This
	// attempt completes driver work, but loses its committed handoff response.
	client.loseCheckpoint = true
	execution, err := Container(ctx, opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost checkpoint response")
	require.NotNil(t, execution)
	stored, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	checkpoint := stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue()
	cache := stored.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue()
	require.NotEmpty(t, checkpoint)
	require.NotEmpty(t, cache)
	client.cacheErr = fmt.Errorf("cache unavailable after original decision")
	opts.DriverRetryAttempt = 1
	for i := 0; i < 2; i++ {
		replayed, err := Container(ctx, opts, tc.ClientManager)
		require.NoError(t, err)
		require.Equal(t, execution.TaskID, replayed.TaskID)
		require.Equal(t, execution.Cached, replayed.Cached)
		require.Equal(t, execution.PodSpecPatch, replayed.PodSpecPatch)
		require.True(t, proto.Equal(execution.ExecutorInput, replayed.ExecutorInput))
	}
	stored, err = client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	require.Equal(t, "1", stored.GetStatusMetadata().GetCustomProperties()[util.DriverRetryAttemptKey].GetStringValue())
	require.Equal(t, checkpoint, stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
	require.Equal(t, cache, stored.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())
	require.Equal(t, 1, client.cacheCalls)
}

func TestDriverRetryAttemptCannotAdoptRefreshedOwner(t *testing.T) {
	for _, operationFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("operation_fails=%t", operationFails), func(t *testing.T) {
			ctx := context.Background()
			tc, opts, client := retryAttemptContext(t)
			var taskID string
			_, err := recoverDriver(ctx, opts, tc.ClientManager, func(_ context.Context, current common.Options, _ client_manager.ClientManagerInterface) (*Execution, error) {
				taskID = current.DriverRetryTask.GetTaskId()
				client.supersedeOnGet = true
				if operationFails {
					return nil, fmt.Errorf("old operation failed")
				}
				return &Execution{TaskID: taskID}, nil
			})
			require.Error(t, err, "refreshing a newer claim must not authorize an old driver's final write")
			stored, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: taskID})
			require.NoError(t, err)
			require.Equal(t, api.PipelineTask_CACHED, stored.State)
			require.Equal(t, "1", stored.GetStatusMetadata().GetCustomProperties()[util.DriverRetryAttemptKey].GetStringValue())
			require.Equal(t, "owned by the newer attempt", stored.GetStatusMetadata().GetMessage())
			checkpoint, err := restoreDriverCheckpoint(stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue(), taskID)
			require.NoError(t, err)
			require.Equal(t, util.BoolPointer(true), checkpoint.Cached)
		})
	}
}

func TestDriverRetryAttemptCannotAdoptUpdateResponseOwner(t *testing.T) {
	tc, opts, client := retryAttemptContext(t)
	client.supersedeUpdateResponse = true
	operationCalls := 0
	_, err := recoverDriver(context.Background(), opts, tc.ClientManager, func(_ context.Context, current common.Options, _ client_manager.ClientManagerInterface) (*Execution, error) {
		operationCalls++
		return &Execution{TaskID: current.DriverRetryTask.GetTaskId()}, nil
	})
	require.Error(t, err, "a later claim returned by response hydration must not authorize this driver")
	require.Zero(t, operationCalls)
}

func TestDriverRetryAttemptManualGenerationRestartsAtZero(t *testing.T) {
	ctx := context.Background()
	tc, opts, client := retryAttemptContext(t)
	enableAttemptTestCache(&opts, client)
	opts.DriverRetryAttempt = 2
	execution, err := Container(ctx, opts, tc.ClientManager)
	require.NoError(t, err)
	before, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	client.generation = 1
	opts.DriverRetryGeneration = 1
	opts.DriverRetryAttempt = 0
	client.cacheErr = fmt.Errorf("cache unavailable after preserved completion")
	replayed, err := Container(ctx, opts, tc.ClientManager)
	require.NoError(t, err)
	require.Equal(t, execution.TaskID, replayed.TaskID)
	require.Equal(t, execution.Cached, replayed.Cached)
	require.Equal(t, execution.PodSpecPatch, replayed.PodSpecPatch)
	require.True(t, proto.Equal(execution.ExecutorInput, replayed.ExecutorInput))
	after, err := client.GetTask(ctx, &api.GetTaskRequest{RunId: opts.Run.RunId, TaskId: execution.TaskID})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_CACHED, after.State)
	require.True(t, proto.Equal(before.GetEndTime(), after.GetEndTime()))
	require.True(t, proto.Equal(before.GetOutputs(), after.GetOutputs()))
	require.Equal(t, "1", after.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
	require.Equal(t, "0", after.GetStatusMetadata().GetCustomProperties()[util.DriverRetryAttemptKey].GetStringValue())
	require.Equal(t, before.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey], after.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey])
	require.Equal(t, 1, client.cacheCalls)
	opts.DriverRetryGeneration = 0
	opts.DriverRetryAttempt = 2
	_, err = Container(ctx, opts, tc.ClientManager)
	require.ErrorContains(t, err, "stale driver generation")
}
