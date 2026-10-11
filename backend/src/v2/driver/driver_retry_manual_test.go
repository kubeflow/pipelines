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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDriverRetryManualRetryRestoresPreservedCachedCheckpoint(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	opts.CacheDisabled = false
	opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	faults.cacheTask = &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{ParameterKey: "value", Value: structpb.NewStringValue("original"), Type: api.IOType_OUTPUT}}}}
	faults.loseCheckpoint = true
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost checkpoint response")
	require.NotNil(t, first)
	require.NotNil(t, first.Cached)
	require.True(t, *first.Cached)
	preserved, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: first.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_CACHED, preserved.GetState())
	preserved = proto.Clone(preserved).(*api.PipelineTask)
	checkpoint := preserved.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue()
	require.NotEmpty(t, checkpoint)
	calls := faults.cacheCalls
	faults.cacheErr = fmt.Errorf("cache unavailable after the original driver completed")

	// RetryRun preserves successful native tasks even when Argo never received
	// their handoff. A new generation must acknowledge the saved result.
	opts.DriverRetryGeneration++
	opts.DriverRetryAttempt = 0
	opts.PodUID = "manual-retry"
	second, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, first.TaskID, second.TaskID)
	assert.Equal(t, first.Cached, second.Cached)
	assert.True(t, proto.Equal(first.ExecutorInput, second.ExecutorInput))
	assert.Equal(t, first.PodSpecPatch, second.PodSpecPatch)
	assert.Equal(t, calls, faults.cacheCalls)
	stored, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: second.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	assert.Equal(t, api.PipelineTask_CACHED, stored.GetState())
	assert.True(t, proto.Equal(preserved.GetOutputs(), stored.GetOutputs()))
	assert.True(t, proto.Equal(preserved.GetEndTime(), stored.GetEndTime()))
	assert.Equal(t, "1", stored.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
	assert.Equal(t, checkpoint, stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
}

func TestDriverRetryManualRetryFinishesPreservedCachedTaskWithoutCheckpoint(t *testing.T) {
	tc, opts, faults := retryContainerContext(t)
	opts.CacheDisabled = false
	opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	faults.cacheTask = &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{ParameterKey: "value", Value: structpb.NewStringValue("original"), Type: api.IOType_OUTPUT}}}}
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	preserved, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: first.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	preserved = proto.Clone(preserved).(*api.PipelineTask)
	require.Equal(t, api.PipelineTask_CACHED, preserved.GetState())
	require.NotEmpty(t, preserved.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())

	// This is the persisted snapshot after the cached state commits and before
	// the handoff checkpoint commits. A killed process cannot reset it to RUNNING.
	delete(preserved.StatusMetadata.CustomProperties, driverCheckpointKey)
	_, err = tc.MockAPI.UpdateTask(context.Background(), &api.UpdateTaskRequest{TaskId: preserved.TaskId, RunId: opts.Run.RunId, Task: proto.Clone(preserved).(*api.PipelineTask)})
	require.NoError(t, err)
	calls := faults.cacheCalls
	faults.cacheErr = fmt.Errorf("cache unavailable after the original cache decision")
	opts.DriverRetryGeneration++
	opts.DriverRetryAttempt = 0
	second, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	assert.Equal(t, first.TaskID, second.TaskID)
	require.NotNil(t, second.Cached)
	assert.True(t, *second.Cached)
	assert.Equal(t, calls, faults.cacheCalls, "manual retry must use the frozen cache decision")
	stored, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: second.TaskID, RunId: opts.Run.RunId})
	require.NoError(t, err)
	assert.Equal(t, api.PipelineTask_CACHED, stored.GetState())
	assert.True(t, proto.Equal(preserved.GetOutputs(), stored.GetOutputs()))
	assert.Equal(t, "1", stored.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
	assert.NotEmpty(t, stored.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
}

func TestDriverRetryManualRetryRejectsIncompatibleStoredGeneration(t *testing.T) {
	for _, test := range []struct {
		name               string
		storedGeneration   int64
		incomingGeneration int64
		cached             bool
	}{
		{name: "nonterminal previous generation", storedGeneration: 0, incomingGeneration: 1},
		{name: "older attempt cannot adopt newer terminal task", storedGeneration: 1, incomingGeneration: 0, cached: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tc, opts, faults := retryContainerContext(t)
			opts.DriverRetryGeneration = test.storedGeneration
			if test.cached {
				opts.CacheDisabled = false
				opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
				opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
				faults.cacheTask = &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{Parameters: []*api.PipelineTask_InputOutputs_IOParameter{{ParameterKey: "value", Value: structpb.NewStringValue("original"), Type: api.IOType_OUTPUT}}}}
			}
			first, err := Container(context.Background(), opts, tc.ClientManager)
			require.NoError(t, err)
			preserved, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: first.TaskID, RunId: opts.Run.RunId})
			require.NoError(t, err)
			preserved = proto.Clone(preserved).(*api.PipelineTask)
			if test.cached {
				require.Equal(t, api.PipelineTask_CACHED, preserved.GetState())
			} else {
				require.Equal(t, api.PipelineTask_RUNNING, preserved.GetState())
			}
			opts.DriverRetryGeneration = test.incomingGeneration
			_, err = Container(context.Background(), opts, tc.ClientManager)
			require.ErrorContains(t, err, "driver task belongs to retry generation")
			stored, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{TaskId: first.TaskID, RunId: opts.Run.RunId})
			require.NoError(t, err)
			assert.True(t, proto.Equal(preserved, stored), "rejected recovery must leave the task unchanged")
		})
	}
}
