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

package kfpapi

import (
	"context"
	"net"
	"testing"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type hangingRunServer struct {
	gc.UnimplementedRunServiceServer
}

func (hangingRunServer) GetRun(ctx context.Context, _ *gc.GetRunRequest) (*gc.Run, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (hangingRunServer) CreateTask(ctx context.Context, _ *gc.CreateTaskRequest) (*gc.PipelineTask, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// newHangingRunAPI returns an API adapter backed by a real gRPC server that
// accepts every Run call and never replies.
func newHangingRunAPI(t *testing.T, policy RetryPolicy) API {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	server := grpc.NewServer()
	gc.RegisterRunServiceServer(server, hangingRunServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := &apiclient.Client{
		Run:  gc.NewRunServiceClient(conn),
		Conn: conn,
	}
	t.Cleanup(func() { _ = client.Close() })
	return NewWithRetryPolicy(client, policy)
}

func TestClientAdapterGetRunBoundsHangingGRPC(t *testing.T) {
	api := newHangingRunAPI(t, RetryPolicy{
		AttemptTimeout: 50 * time.Millisecond,
		TotalTimeout:   80 * time.Millisecond,
		MaxAttempts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})

	start := time.Now()
	_, err := api.GetRun(context.Background(), &gc.GetRunRequest{RunId: "test-run"})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, time.Second)
	require.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
}

func TestClientAdapterCreateTaskBoundsHangingGRPCWithoutRetry(t *testing.T) {
	api := newHangingRunAPI(t, RetryPolicy{
		AttemptTimeout: 20 * time.Millisecond,
		TotalTimeout:   80 * time.Millisecond,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})

	start := time.Now()
	_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{})
	elapsed := time.Since(start)

	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Contains(t, err.Error(), "may or may not have been committed")
	// A single attempt bounded by TotalTimeout, not MaxAttempts*AttemptTimeout.
	require.Less(t, elapsed, time.Second)
	require.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
}
