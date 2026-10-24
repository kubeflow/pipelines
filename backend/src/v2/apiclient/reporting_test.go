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

package apiclient

import (
	"context"
	"net"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestLauncherIdentityOnEveryTaskAndArtifactRPC(t *testing.T) {
	t.Setenv(KFPTaskIDEnvVar, "task-a")
	t.Setenv(KFPTaskAttemptIDEnvVar, "attempt-a")
	headers := make(chan metadata.MD, 8)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
		incoming, _ := metadata.FromIncomingContext(ctx)
		headers <- incoming.Copy()
		return &emptypb.Empty{}, nil
	}))
	apiv2.RegisterRunServiceServer(server, &apiv2.UnimplementedRunServiceServer{})
	apiv2.RegisterArtifactServiceServer(server, &apiv2.UnimplementedArtifactServiceServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	client, err := New(&Config{Endpoint: listener.Addr().String(), TokenSource: tokenSourceFunc(func(context.Context) (string, error) {
		return "runtime-test", nil
	})}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	// Identity belongs to this connection, not later process-environment changes.
	t.Setenv(KFPTaskIDEnvVar, "task-b")
	t.Setenv(KFPTaskAttemptIDEnvVar, "attempt-b")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"preserved", "yes", KFPTaskIDHeader, "wrong", KFPTaskAttemptIDHeader, "wrong",
	))
	calls := []func() error{
		func() error { _, err := client.Run.UpdateTask(ctx, &apiv2.UpdateTaskRequest{}); return err },
		func() error { _, err := client.Run.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{}); return err },
		func() error {
			_, err := client.Artifact.CreateArtifact(ctx, &apiv2.CreateArtifactRequest{})
			return err
		},
		func() error {
			_, err := client.Artifact.CreateArtifactsBulk(ctx, &apiv2.CreateArtifactsBulkRequest{})
			return err
		},
		func() error {
			_, err := client.Artifact.CreateArtifactTask(ctx, &apiv2.CreateArtifactTaskRequest{})
			return err
		},
		func() error {
			_, err := client.Artifact.CreateArtifactTasksBulk(ctx, &apiv2.CreateArtifactTasksBulkRequest{})
			return err
		},
	}
	for _, call := range calls {
		require.NoError(t, call())
		incoming := <-headers
		assert.Equal(t, []string{"task-a"}, incoming.Get(KFPTaskIDHeader))
		assert.Equal(t, []string{"attempt-a"}, incoming.Get(KFPTaskAttemptIDHeader))
		assert.Equal(t, []string{"yes"}, incoming.Get("preserved"))
		assert.NotEmpty(t, incoming.Get("authorization"), "attempt metadata must preserve existing credentials")
	}
}

func TestLauncherReportIdentityConfiguration(t *testing.T) {
	t.Setenv(KFPTaskIDEnvVar, "legacy-task")
	t.Setenv(KFPTaskAttemptIDEnvVar, "")
	interceptor, err := taskReportInterceptor(&Config{})
	require.NoError(t, err)
	assert.Nil(t, interceptor, "Argo callers without attempts must keep the existing protocol")
	t.Setenv(KFPTaskIDEnvVar, "")
	t.Setenv(KFPTaskAttemptIDEnvVar, "attempt")
	_, err = New(&Config{Endpoint: "127.0.0.1:1"}, nil)
	require.ErrorContains(t, err, KFPTaskIDEnvVar)

	interceptor, err = taskReportInterceptor(&Config{TaskID: "explicit-task", TaskAttemptID: "explicit-attempt"})
	require.NoError(t, err)
	err = interceptor(context.Background(), "/test", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		headers, _ := metadata.FromOutgoingContext(ctx)
		assert.Equal(t, []string{"explicit-task"}, headers.Get(KFPTaskIDHeader))
		assert.Equal(t, []string{"explicit-attempt"}, headers.Get(KFPTaskAttemptIDHeader))
		return nil
	})
	require.NoError(t, err)
}
