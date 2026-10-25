// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cacheutils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type recordingTaskClient struct {
	api.TaskServiceClient
	contexts []context.Context
}

func (s *recordingTaskClient) CreateTaskV1(ctx context.Context, request *api.CreateTaskRequest, opts ...grpc.CallOption) (*api.Task, error) {
	s.contexts = append(s.contexts, ctx)
	return request.Task, ctx.Err()
}

func TestCreateExecutionCacheAuthentication(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("dummy-token"), 0600))
	t.Setenv("KFP_TASK_API_TOKEN_PATH", tokenFile)
	t.Setenv("KF_PIPELINES_SA_TOKEN_PATH", filepath.Join(t.TempDir(), "user-sdk-token"))
	value, err := NewClient("localhost", "1", false, nil)
	require.NoError(t, err)
	c := value.(*client)
	recorder := &recordingTaskClient{}
	c.svc = recorder
	original := metadata.Pairs("authorization", "Bearer old", "authorization", "Bearer duplicate", "trace-id", "trace")
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), original), time.Minute)
	defer cancel()
	require.NoError(t, c.CreateExecutionCache(ctx, &api.Task{}))
	require.Len(t, recorder.contexts, 1)
	md, _ := metadata.FromOutgoingContext(recorder.contexts[0])
	require.Equal(t, []string{"Bearer dummy-token"}, md.Get("authorization"))
	require.Equal(t, []string{"trace"}, md.Get("trace-id"))
	require.Len(t, original.Get("authorization"), 2)
	deadline, _ := ctx.Deadline()
	gotDeadline, ok := recorder.contexts[0].Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, gotDeadline)
	require.NoError(t, os.WriteFile(tokenFile, []byte("rotated-token"), 0600))
	c.tokenSource.ResetTokenOlderThan(time.Now().Add(time.Second))
	require.NoError(t, c.CreateExecutionCache(ctx, &api.Task{}))
	rotated, _ := metadata.FromOutgoingContext(recorder.contexts[1])
	require.Equal(t, []string{"Bearer rotated-token"}, rotated.Get("authorization"))
	cancel()
	require.ErrorIs(t, c.CreateExecutionCache(ctx, &api.Task{}), context.Canceled)
}

func TestCreateExecutionCacheMissingCredential(t *testing.T) {
	t.Setenv("KFP_TASK_API_TOKEN_PATH", filepath.Join(t.TempDir(), "missing"))
	value, err := NewClient("localhost", "1", false, nil)
	require.NoError(t, err)
	c := value.(*client)
	recorder := &recordingTaskClient{}
	c.svc = recorder
	require.Error(t, c.CreateExecutionCache(context.Background(), &api.Task{}))
	require.Empty(t, recorder.contexts)
	_, err = NewClient("localhost", "1", true, nil)
	require.NoError(t, err)
}

func TestCreateExecutionCacheSingleUser(t *testing.T) {
	t.Setenv("KFP_TASK_API_TOKEN_PATH", "")
	value, err := NewClient("localhost", "1", false, nil)
	require.NoError(t, err)
	c := value.(*client)
	recorder := &recordingTaskClient{}
	c.svc = recorder
	require.NoError(t, c.CreateExecutionCache(context.Background(), &api.Task{}))
	md, _ := metadata.FromOutgoingContext(recorder.contexts[0])
	require.Empty(t, md.Get("authorization"))
}
