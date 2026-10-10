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

package kfpapi

import (
	"context"
	"strings"
	"testing"

	runtimeapi "github.com/kubeflow/pipelines/backend/api/runtime/go_client"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type recoveryRecordingClient struct {
	runtimeapi.DriverTaskServiceClient
	requests []*runtimeapi.WriteTaskRequest
	gets     []*runtimeapi.GetTaskRequest
	auth     []string
}

func (r *recoveryRecordingClient) record(ctx context.Context, request *runtimeapi.WriteTaskRequest) (*runtimeapi.TaskResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	r.auth = append(r.auth, md.Get("authorization")...)
	r.requests = append(r.requests, request)
	recovery := &runtimeapi.DriverRecovery{Generation: proto.Int64(3), Attempt: proto.Int64(2)}
	if request.IncludeRecovery {
		recovery.CheckpointJson = proto.String(`{"saved":true}`)
	}
	return &runtimeapi.TaskResponse{Task: proto.Clone(request.Task).(*api.PipelineTask), Recovery: recovery}, nil
}
func (r *recoveryRecordingClient) CreateTask(ctx context.Context, request *runtimeapi.WriteTaskRequest, _ ...grpc.CallOption) (*runtimeapi.TaskResponse, error) {
	return r.record(ctx, request)
}
func (r *recoveryRecordingClient) UpdateTask(ctx context.Context, request *runtimeapi.WriteTaskRequest, _ ...grpc.CallOption) (*runtimeapi.TaskResponse, error) {
	return r.record(ctx, request)
}
func (r *recoveryRecordingClient) GetTask(_ context.Context, request *runtimeapi.GetTaskRequest, _ ...grpc.CallOption) (*runtimeapi.TaskResponse, error) {
	r.gets = append(r.gets, request)
	return &runtimeapi.TaskResponse{Task: &api.PipelineTask{TaskId: request.TaskId}}, nil
}
func (r *recoveryRecordingClient) UpdateTasksBulk(ctx context.Context, request *runtimeapi.WriteTasksRequest, _ ...grpc.CallOption) (*runtimeapi.TasksResponse, error) {
	result := &runtimeapi.TasksResponse{}
	for _, task := range request.Tasks {
		response, err := r.record(ctx, task)
		if err != nil {
			return nil, err
		}
		result.Tasks = append(result.Tasks, response)
	}
	return result, nil
}

func TestRuntimeRecoveryUsesTypedPrivateProtocol(t *testing.T) {
	for _, full := range []bool{false, true} {
		transport := &recoveryRecordingClient{}
		client := NewWithRetryGeneration(&apiclient.Client{DriverTask: transport}, 3)
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "bound-token"))
		if full {
			ctx = WithDriverRecovery(ctx)
		}
		task := &api.PipelineTask{TaskId: "task", RunId: "run", StatusMetadata: &api.PipelineTask_StatusMetadata{Message: "message", CustomProperties: map[string]*structpb.Value{
			util.DriverRetryGenerationKey: structpb.NewStringValue("3"), util.DriverRetryAttemptKey: structpb.NewStringValue("2"), util.DriverCheckpointKey: structpb.NewStringValue(`{"saved":true}`), "visible": structpb.NewStringValue("value"),
		}}}
		created, err := client.CreateTask(ctx, &api.CreateTaskRequest{RunId: "run", Task: task})
		require.NoError(t, err)
		require.Equal(t, "2", created.GetStatusMetadata().GetCustomProperties()[util.DriverRetryAttemptKey].GetStringValue())
		_, err = client.UpdateTask(ctx, &api.UpdateTaskRequest{RunId: "run", TaskId: "task", Task: task})
		require.NoError(t, err)
		_, err = client.GetTask(ctx, &api.GetTaskRequest{RunId: "run", TaskId: "task"})
		require.NoError(t, err)
		bulk, err := client.UpdateTasksBulk(ctx, &api.UpdateTasksBulkRequest{RunId: "run", Tasks: map[string]*api.PipelineTask{"task": task}})
		require.NoError(t, err)
		require.Len(t, bulk.Tasks["task"].GetStatusMetadata().GetCustomProperties(), 1)
		require.True(t, transport.requests[0].Claim)
		require.False(t, transport.requests[1].Claim)
		require.Equal(t, full, transport.gets[0].IncludeRecovery)
		require.False(t, transport.requests[2].IncludeRecovery)
		for _, request := range transport.requests {
			require.EqualValues(t, 3, request.Authority.GetGeneration())
			require.Len(t, request.Task.GetStatusMetadata().GetCustomProperties(), 1)
			for key := range request.Task.GetStatusMetadata().GetCustomProperties() {
				require.False(t, strings.HasPrefix(key, "_kfp_driver_"))
			}
			require.Equal(t, full, request.Recovery.CheckpointJson != nil)
		}
		require.Len(t, task.GetStatusMetadata().GetCustomProperties(), 4, "transport must not mutate caller state")
		require.Equal(t, []string{"bound-token", "bound-token", "bound-token"}, transport.auth)
	}
}

func TestUntaggedChildCarriesImmutableGenerationWithoutParentOwner(t *testing.T) {
	transport := &recoveryRecordingClient{}
	client := NewWithRetryGeneration(&apiclient.Client{DriverTask: transport}, 3)
	parent := &api.PipelineTask{TaskId: "parent", StatusMetadata: &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
		util.DriverRetryGenerationKey: structpb.NewStringValue("4"), util.DriverRetryAttemptKey: structpb.NewStringValue("9"), util.DriverCheckpointKey: structpb.NewStringValue(`{"parent":true}`),
	}}}
	source := &api.PipelineTask{TaskId: "untagged-child"}
	util.CopyDriverRetryGeneration(parent, source)
	_, err := client.UpdateTask(context.Background(), &api.UpdateTaskRequest{RunId: "run", TaskId: "parent", Task: parent})
	require.NoError(t, err)
	request := transport.requests[0]
	require.EqualValues(t, 3, request.Authority.GetGeneration(), "must never adopt generation from a refreshed parent")
	require.Equal(t, "untagged-child", request.Authority.SourceTaskId)
	require.Nil(t, request.Authority.SourceAttempt)
	require.Nil(t, request.Recovery, "a child never supplies its parent's ownership or checkpoint")
	require.Empty(t, request.Task.GetStatusMetadata().GetCustomProperties())
	require.Nil(t, request.Task.StatusMetadata, "authority-only metadata must preserve patch omission")
}

func TestRecoveryContextDoesNotTurnUnclaimedWritesIntoRecoveryWrites(t *testing.T) {
	client := &clientAdapter{generation: 0}
	request, err := client.runtimeTaskRequest(WithDriverRecovery(context.Background()), "run", &api.PipelineTask{TaskId: "iteration"}, false)
	require.NoError(t, err)
	require.False(t, request.UpdateRecovery)
	require.False(t, request.Claim)
}
