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
	"sync/atomic"
	"testing"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type hangingRunServer struct {
	gc.UnimplementedRunServiceServer
	getRunCalls     atomic.Int32
	createTaskCalls atomic.Int32
}

func (s *hangingRunServer) GetRun(ctx context.Context, _ *gc.GetRunRequest) (*gc.Run, error) {
	s.getRunCalls.Add(1)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (s *hangingRunServer) CreateTask(ctx context.Context, _ *gc.CreateTaskRequest) (*gc.PipelineTask, error) {
	s.createTaskCalls.Add(1)
	<-ctx.Done()
	return nil, ctx.Err()
}

// newHangingRunAPI returns an API adapter backed by a real gRPC server that
// accepts every Run call and never replies.
func newHangingRunAPI(t *testing.T, policy RetryPolicy) API {
	api, _ := newHangingRunAPIWithServer(t, policy)
	return api
}

func newHangingRunAPIWithServer(t *testing.T, policy RetryPolicy) (API, *hangingRunServer) {
	t.Helper()
	runServer := &hangingRunServer{}
	client := newTestGRPCClient(t, func(server *grpc.Server) {
		gc.RegisterRunServiceServer(server, runServer)
	})
	return NewWithRetryPolicy(client, policy), runServer
}

// newTestGRPCClient serves the services registered by register on a local
// port and returns a runtime API client connected to it.
func newTestGRPCClient(t *testing.T, register func(*grpc.Server), opts ...grpc.ServerOption) *apiclient.Client {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	server := grpc.NewServer(opts...)
	register(server)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	// Connect before the test starts timing calls, so connection setup never
	// consumes a test's attempt or caller deadline.
	readyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn.Connect()
	for state := conn.GetState(); state != connectivity.Ready; state = conn.GetState() {
		require.True(t, conn.WaitForStateChange(readyCtx, state), "test gRPC server did not become ready")
	}

	client := &apiclient.Client{
		Run:      gc.NewRunServiceClient(conn),
		Artifact: gc.NewArtifactServiceClient(conn),
		Pipeline: gc.NewPipelineServiceClient(conn),
		Conn:     conn,
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClientAdapterGetRunBoundsHangingGRPC(t *testing.T) {
	api := newHangingRunAPI(t, RetryPolicy{
		AttemptTimeout: 50 * time.Millisecond,
		TotalTimeout:   80 * time.Millisecond,
		MaxAttempts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
		WriteTimeout:   time.Second,
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
		TotalTimeout:   20 * time.Millisecond,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
		WriteTimeout:   80 * time.Millisecond,
	})

	start := time.Now()
	_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{})
	elapsed := time.Since(start)

	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Contains(t, err.Error(), "may or may not have been committed")
	// A single attempt bounded by WriteTimeout, not by the read budgets.
	require.Less(t, elapsed, time.Second)
	require.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
}

func TestClientAdapterGetRunRetriesHangingGRPCWithinBudget(t *testing.T) {
	api, server := newHangingRunAPIWithServer(t, RetryPolicy{
		AttemptTimeout: 100 * time.Millisecond,
		TotalTimeout:   2 * time.Second,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
		WriteTimeout:   time.Second,
	})

	start := time.Now()
	_, err := api.GetRun(context.Background(), &gc.GetRunRequest{RunId: "test-run"})

	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Less(t, time.Since(start), 2*time.Second)
	// Under load an attempt can time out before its handler starts, so assert the
	// retry and the attempt limit rather than an exact server-side count.
	require.Eventually(t, func() bool { return server.getRunCalls.Load() >= 2 },
		time.Second, 10*time.Millisecond, "the hanging read must be retried")
	require.LessOrEqual(t, server.getRunCalls.Load(), int32(3))
}

func TestClientAdapterEarlierCallerDeadlineWinsOverGRPCBudgets(t *testing.T) {
	api, server := newHangingRunAPIWithServer(t, slowPolicy())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := api.GetRun(ctx, &gc.GetRunRequest{RunId: "test-run"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 2*time.Second)

	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start = time.Now()
	_, err = api.CreateTask(ctx, &gc.CreateTaskRequest{})
	require.ErrorIs(t, err, context.DeadlineExceeded, "a write stopped by its caller returns the caller's error")
	require.NotContains(t, err.Error(), "may or may not have been committed")
	require.Less(t, time.Since(start), 2*time.Second)

	// Both calls have returned, so any retry would already have been sent.
	require.LessOrEqual(t, server.getRunCalls.Load(), int32(1), "a read must not be retried after the caller's deadline")
	require.LessOrEqual(t, server.createTaskCalls.Load(), int32(1), "a write is never retried")
}
