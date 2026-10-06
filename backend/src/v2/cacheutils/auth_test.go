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

package cacheutils

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type transportTaskServer struct {
	api.UnimplementedTaskServiceServer
	calls chan string
}

func (s *transportTaskServer) ListTasksV1(ctx context.Context, req *api.ListTasksRequest) (*api.ListTasksResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.calls <- strings.Join(md.Get("authorization"), "") + ":" + req.GetResourceReferenceKey().GetId()
	return &api.ListTasksResponse{}, nil
}
func (s *transportTaskServer) CreateTaskV1(ctx context.Context, req *api.CreateTaskRequest) (*api.Task, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.calls <- strings.Join(md.Get("authorization"), "")
	return req.Task, nil
}

func TestNewClientSendsCacheCredentialsOverGRPC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	t.Setenv("KFP_CACHE_API_TOKEN_PATH", path)
	t.Setenv("KF_PIPELINES_SA_TOKEN_PATH", "/unrelated-sdk-token")
	require.NoError(t, os.WriteFile(path, []byte("first"), 0600))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	svc := &transportTaskServer{calls: make(chan string, 2)}
	api.RegisterTaskServiceServer(server, svc)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	c, err := NewClient(host, port, false, nil)
	require.NoError(t, err)
	_, err = c.GetExecutionCache("fp", "pipeline/p", "team-a")
	require.NoError(t, err)
	require.Equal(t, "Bearer first:team-a", <-svc.calls)
	require.NoError(t, os.WriteFile(path, []byte("rotated"), 0600))
	require.NoError(t, c.CreateExecutionCache(context.Background(), &api.Task{Namespace: "team-a"}))
	require.Equal(t, "Bearer rotated", <-svc.calls)
}

func TestCacheAuthRotatesToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	t.Setenv("KFP_CACHE_API_TOKEN_PATH", path)
	for _, token := range []string{"first", "rotated"} {
		require.NoError(t, os.WriteFile(path, []byte(token+"\n"), 0600))
		called := false
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer stale"))
		err := cacheAuthInterceptor(ctx, "method", nil, nil, nil, func(ctx context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			called = true
			md, _ := metadata.FromOutgoingContext(ctx)
			require.Equal(t, []string{"Bearer " + token}, md.Get("authorization"))
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	}
	for _, contents := range []string{"", "   "} {
		require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
		require.Error(t, cacheAuthInterceptor(context.Background(), "method", nil, nil, nil, nil))
	}
	require.NoError(t, os.Remove(path))
	require.Error(t, cacheAuthInterceptor(context.Background(), "method", nil, nil, nil, nil))
	_, err := NewClient("unused", "1", true, nil)
	require.NoError(t, err, "disabled cache must not require credentials")
}

type recordingTaskClient struct {
	api.TaskServiceClient
	request *api.ListTasksRequest
}

func (c *recordingTaskClient) ListTasksV1(_ context.Context, request *api.ListTasksRequest, _ ...grpc.CallOption) (*api.ListTasksResponse, error) {
	c.request = request
	return &api.ListTasksResponse{}, nil
}

func TestCacheLookupProvidesNamespaceReference(t *testing.T) {
	svc := &recordingTaskClient{}
	c := &client{svc: svc}
	_, err := c.GetExecutionCache("fp", "pipeline/p", "team-a")
	require.NoError(t, err)
	require.Equal(t, api.ResourceType_NAMESPACE, svc.request.ResourceReferenceKey.Type)
	require.Equal(t, "team-a", svc.request.ResourceReferenceKey.Id)
}
