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
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type recoveryRecordingClient struct {
	api.RunServiceClient
	views []string
	auth  []string
}

func (r *recoveryRecordingClient) record(ctx context.Context) {
	md, _ := metadata.FromOutgoingContext(ctx)
	r.views = append(r.views, firstMetadataValue(md.Get(util.DriverRecoveryViewHeader)))
	r.auth = append(r.auth, firstMetadataValue(md.Get("authorization")))
}
func firstMetadataValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
func (r *recoveryRecordingClient) GetTask(ctx context.Context, _ *api.GetTaskRequest, _ ...grpc.CallOption) (*api.PipelineTask, error) {
	r.record(ctx)
	return &api.PipelineTask{}, nil
}
func (r *recoveryRecordingClient) CreateTask(ctx context.Context, _ *api.CreateTaskRequest, _ ...grpc.CallOption) (*api.PipelineTask, error) {
	r.record(ctx)
	return &api.PipelineTask{}, nil
}
func (r *recoveryRecordingClient) UpdateTask(ctx context.Context, _ *api.UpdateTaskRequest, _ ...grpc.CallOption) (*api.PipelineTask, error) {
	r.record(ctx)
	return &api.PipelineTask{}, nil
}
func (r *recoveryRecordingClient) GetRun(ctx context.Context, _ *api.GetRunRequest, _ ...grpc.CallOption) (*api.Run, error) {
	r.record(ctx)
	return &api.Run{}, nil
}
func (r *recoveryRecordingClient) ListTasks(ctx context.Context, _ *api.ListTasksRequest, _ ...grpc.CallOption) (*api.ListTasksResponse, error) {
	r.record(ctx)
	return &api.ListTasksResponse{}, nil
}
func (r *recoveryRecordingClient) UpdateTasksBulk(ctx context.Context, _ *api.UpdateTasksBulkRequest, _ ...grpc.CallOption) (*api.UpdateTasksBulkResponse, error) {
	r.record(ctx)
	return &api.UpdateTasksBulkResponse{}, nil
}

func TestDriverRecoveryViewOnlyOnSingleTaskRPCs(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		transport := &recoveryRecordingClient{}
		client := New(&apiclient.Client{Run: transport})
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "bound-token"))
		view := util.DriverRecoveryViewOwnership
		if recovery {
			ctx = WithDriverRecovery(ctx)
			view = util.DriverRecoveryViewFull
		}
		_, err := client.GetTask(ctx, &api.GetTaskRequest{})
		require.NoError(t, err)
		_, err = client.CreateTask(ctx, &api.CreateTaskRequest{})
		require.NoError(t, err)
		_, err = client.UpdateTask(ctx, &api.UpdateTaskRequest{})
		require.NoError(t, err)
		_, err = client.GetRun(ctx, &api.GetRunRequest{})
		require.NoError(t, err)
		_, err = client.ListTasks(ctx, &api.ListTasksRequest{})
		require.NoError(t, err)
		_, err = client.UpdateTasksBulk(ctx, &api.UpdateTasksBulkRequest{})
		require.NoError(t, err)
		require.Equal(t, []string{view, view, view, "", "", ""}, transport.views)
		for _, auth := range transport.auth {
			require.Equal(t, "bound-token", auth)
		}
	}
}
