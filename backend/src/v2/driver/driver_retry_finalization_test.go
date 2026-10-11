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

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func finalAttemptContext(t *testing.T, kind string) (*TestContext, common.Options, *driverAttemptAPI, driverOperation) {
	t.Helper()
	if kind == "container" {
		tc, opts, client := retryAttemptContext(t)
		return tc, opts, client, Container
	}
	tc, opts := newDAGRetryTestContext(t)
	client := &driverAttemptAPI{driverRetryFaultAPI: &driverRetryFaultAPI{MockAPI: tc.MockAPI}}
	tc.ClientManager = client_manager.NewFakeClientManager(tc.ClientManager.K8sClient(), client)
	return tc, opts, client, DAG
}

func TestDriverRetryFinalAttemptFailurePropagates(t *testing.T) {
	for _, kind := range []string{"container", "dag"} {
		for _, final := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/final=%t", kind, final), func(t *testing.T) {
				tc, opts, _, drive := finalAttemptContext(t, kind)
				opts.Task = withBrokenParameterIterator(opts.Task)
				if final {
					opts.DriverRetryAttempt = opts.DriverRetryMaxCount
				}
				_, err := drive(context.Background(), opts, tc.ClientManager)
				require.Error(t, err)
				task := getOnlyDAGRetryTask(t, tc, opts)
				require.Equal(t, err.Error(), task.GetStatusMetadata().GetMessage())
				generation, attempt, err := driverAttemptOwner(task)
				require.NoError(t, err)
				require.Equal(t, opts.DriverRetryGeneration, generation)
				require.EqualValues(t, opts.DriverRetryAttempt, attempt)
				if final {
					require.Equal(t, api.PipelineTask_FAILED, task.GetState())
					require.NotNil(t, task.GetEndTime())
					parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: tc.RootTask.TaskId})
					require.NoError(t, err)
					require.Equal(t, api.PipelineTask_FAILED, parent.GetState())
					require.NotNil(t, parent.GetEndTime())
				} else {
					require.Equal(t, api.PipelineTask_RUNNING, task.GetState())
					require.Nil(t, task.GetEndTime())
					assertDAGRetryParentRunning(t, tc)
				}
			})
		}
	}
}

type finalAttemptParentReadAPI struct {
	*driverAttemptAPI
	parentID string
}

func (f *finalAttemptParentReadAPI) GetTask(ctx context.Context, req *api.GetTaskRequest) (*api.PipelineTask, error) {
	if req.GetTaskId() == f.parentID {
		f.parentID = ""
		return nil, fmt.Errorf("injected parent read failure")
	}
	return f.driverAttemptAPI.GetTask(ctx, req)
}

func TestDriverRetryFinalAttemptFailureBeforeCleanup(t *testing.T) {
	for _, kind := range []string{"container", "dag"} {
		t.Run(kind, func(t *testing.T) {
			tc, opts, client, drive := finalAttemptContext(t, kind)
			opts.DriverRetryAttempt = opts.DriverRetryMaxCount
			message := "pipeline name is required"
			if kind == "container" {
				message = "injected parent read failure"
				faults := &finalAttemptParentReadAPI{driverAttemptAPI: client, parentID: tc.RootTask.TaskId}
				tc.ClientManager = client_manager.NewFakeClientManager(tc.ClientManager.K8sClient(), faults)
			} else {
				opts.PipelineName = ""
			}
			_, err := drive(context.Background(), opts, tc.ClientManager)
			require.ErrorContains(t, err, message)
			task := getOnlyDAGRetryTask(t, tc, opts)
			require.Equal(t, api.PipelineTask_FAILED, task.GetState())
			require.NotNil(t, task.GetEndTime())
			require.Contains(t, task.GetStatusMetadata().GetMessage(), message)
			parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: tc.RootTask.TaskId})
			require.NoError(t, err)
			require.Equal(t, api.PipelineTask_FAILED, parent.GetState())
		})
	}
}

func TestDriverRetryFinalAttemptFailureCannotOverwriteNewOwner(t *testing.T) {
	for _, manualRetry := range []bool{false, true} {
		t.Run(fmt.Sprintf("manualRetry=%t", manualRetry), func(t *testing.T) {
			tc, opts, client := retryAttemptContext(t)
			opts.DriverRetryAttempt = opts.DriverRetryMaxCount
			var newer *api.PipelineTask
			_, err := recoverDriver(context.Background(), opts, tc.ClientManager, func(ctx context.Context, claimed common.Options, _ client_manager.ClientManagerInterface) (*Execution, error) {
				if manualRetry {
					client.generation++
					newer = proto.Clone(claimed.DriverRetryTask).(*api.PipelineTask)
					owner := claimed
					owner.DriverRetryGeneration = client.generation
					owner.DriverRetryAttempt = 0
					setDriverRetryOwner(newer, owner)
					_, updateErr := client.MockAPI.UpdateTask(ctx, &api.UpdateTaskRequest{RunId: newer.RunId, TaskId: newer.TaskId, Task: newer})
					require.NoError(t, updateErr)
				} else {
					var updateErr error
					newer, updateErr = client.advanceOwner(ctx, claimed.DriverRetryTask)
					require.NoError(t, updateErr)
				}
				return nil, fmt.Errorf("old driver failed")
			})
			require.ErrorContains(t, err, "old driver failed")
			require.ErrorContains(t, err, "stale driver update")
			after := getOnlyDAGRetryTask(t, tc, opts)
			require.True(t, proto.Equal(newer, after), "stale finalization must not change the new owner's task")
			assertDAGRetryParentRunning(t, tc)
		})
	}
}

func TestDriverRetryFinalAttemptPreservesCommittedCacheResult(t *testing.T) {
	tc, opts, client := retryAttemptContext(t)
	opts.DriverRetryAttempt = opts.DriverRetryMaxCount
	enableAttemptTestCache(&opts, client)
	client.loseCached = true
	_, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost cached task response")
	task := getOnlyDAGRetryTask(t, tc, opts)
	require.Equal(t, api.PipelineTask_CACHED, task.GetState())
	require.NotNil(t, task.GetEndTime())
	require.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())
	require.Equal(t, "frozen cache result", task.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	require.Equal(t, 1, client.cacheCalls)
}

func TestDriverRetryFinalAttemptLostCheckpointResponse(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(fmt.Sprintf("cached=%t", cached), func(t *testing.T) {
			tc, opts, client := retryAttemptContext(t)
			opts.DriverRetryAttempt = opts.DriverRetryMaxCount
			if cached {
				enableAttemptTestCache(&opts, client)
			}
			client.loseCheckpoint = true
			execution, err := Container(context.Background(), opts, tc.ClientManager)
			require.ErrorContains(t, err, "lost checkpoint response")
			require.NotNil(t, execution)
			task := getOnlyDAGRetryTask(t, tc, opts)
			require.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
			require.NotNil(t, task.GetEndTime())
			if cached {
				require.Equal(t, api.PipelineTask_CACHED, task.GetState())
				require.Equal(t, "frozen cache result", task.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
				client.generation++
				opts.DriverRetryGeneration = client.generation
				opts.DriverRetryAttempt = 0
				replayed, err := Container(context.Background(), opts, tc.ClientManager)
				require.NoError(t, err)
				require.Equal(t, execution.TaskID, replayed.TaskID)
				require.Empty(t, getOnlyDAGRetryTask(t, tc, opts).GetStatusMetadata().GetMessage())
				require.Equal(t, 1, client.cacheCalls)
			} else {
				require.Equal(t, api.PipelineTask_FAILED, task.GetState())
				parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: tc.RootTask.TaskId})
				require.NoError(t, err)
				require.Equal(t, api.PipelineTask_FAILED, parent.GetState())
			}
		})
	}
}

func TestDriverRetryFinalAttemptLostReplayAcknowledgment(t *testing.T) {
	tc, opts, client := retryAttemptContext(t)
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	opts.DriverRetryAttempt = opts.DriverRetryMaxCount
	opts.Task = withBrokenParameterIterator(opts.Task)
	client.loseCheckpoint = true
	replayed, err := Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost checkpoint response")
	require.Equal(t, first.TaskID, replayed.TaskID)
	task := getOnlyDAGRetryTask(t, tc, opts)
	require.Equal(t, api.PipelineTask_FAILED, task.GetState())
	require.NotNil(t, task.GetEndTime())
	parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: tc.RootTask.TaskId})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_FAILED, parent.GetState())
}

func TestDriverRetryFinalAttemptCorruptCheckpoint(t *testing.T) {
	tc, opts, client := retryAttemptContext(t)
	first, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	task := getOnlyDAGRetryTask(t, tc, opts)
	driverRecoveryMetadata(task).CustomProperties[driverCheckpointKey] = structpb.NewStringValue("invalid checkpoint")
	_, err = client.MockAPI.UpdateTask(context.Background(), &api.UpdateTaskRequest{RunId: task.RunId, TaskId: first.TaskID, Task: task})
	require.NoError(t, err)
	opts.DriverRetryAttempt = opts.DriverRetryMaxCount
	_, err = Container(context.Background(), opts, tc.ClientManager)
	require.ErrorContains(t, err, "invalid saved driver handoff")
	task = getOnlyDAGRetryTask(t, tc, opts)
	require.Equal(t, api.PipelineTask_FAILED, task.GetState())
	require.NotNil(t, task.GetEndTime())
	parent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: tc.RootTask.TaskId})
	require.NoError(t, err)
	require.Equal(t, api.PipelineTask_FAILED, parent.GetState())
}
