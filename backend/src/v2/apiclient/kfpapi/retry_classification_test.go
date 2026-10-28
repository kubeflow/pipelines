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
	"path"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type rpcKind int

const (
	readRPC rpcKind = iota
	writeRPC
)

// classifiedMethod is one API method, the RPC it sends and whether it is retried.
type classifiedMethod struct {
	rpc  string
	kind rpcKind
	call func(context.Context, API) error
}

// classifiedMethods lists every API method that sends an RPC. A new API method
// must be added here as a read or a write before the completeness check passes.
var classifiedMethods = map[string]classifiedMethod{
	"GetRun": {"GetRun", readRPC, func(ctx context.Context, api API) error {
		_, err := api.GetRun(ctx, &gc.GetRunRequest{})
		return err
	}},
	"ListRuns": {"ListRuns", readRPC, func(ctx context.Context, api API) error {
		_, err := api.ListRuns(ctx, &gc.ListRunsRequest{})
		return err
	}},
	"GetTask": {"GetTask", readRPC, func(ctx context.Context, api API) error {
		_, err := api.GetTask(ctx, &gc.GetTaskRequest{})
		return err
	}},
	"ListTasks": {"ListTasks", readRPC, func(ctx context.Context, api API) error {
		_, err := api.ListTasks(ctx, &gc.ListTasksRequest{})
		return err
	}},
	"FindCachedTask": {"FindCachedTask", readRPC, func(ctx context.Context, api API) error {
		_, err := api.FindCachedTask(ctx, &gc.FindCachedTaskRequest{})
		return err
	}},
	"ListArtifactsByURI": {"ListArtifacts", readRPC, func(ctx context.Context, api API) error {
		_, err := api.ListArtifactsByURI(ctx, "s3://bucket/key", "ns")
		return err
	}},
	"ListArtifactTasks": {"ListArtifactTasks", readRPC, func(ctx context.Context, api API) error {
		_, err := api.ListArtifactTasks(ctx, &gc.ListArtifactTasksRequest{})
		return err
	}},
	"GetPipelineVersion": {"GetPipelineVersion", readRPC, func(ctx context.Context, api API) error {
		_, err := api.GetPipelineVersion(ctx, &gc.GetPipelineVersionRequest{})
		return err
	}},
	"CreateTask": {"CreateTask", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.CreateTask(ctx, &gc.CreateTaskRequest{})
		return err
	}},
	"UpdateTask": {"UpdateTask", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.UpdateTask(ctx, &gc.UpdateTaskRequest{})
		return err
	}},
	"UpdateTasksBulk": {"UpdateTasksBulk", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.UpdateTasksBulk(ctx, &gc.UpdateTasksBulkRequest{})
		return err
	}},
	"CreateArtifact": {"CreateArtifact", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.CreateArtifact(ctx, &gc.CreateArtifactRequest{})
		return err
	}},
	"CreateArtifactsBulk": {"CreateArtifactsBulk", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.CreateArtifactsBulk(ctx, &gc.CreateArtifactsBulkRequest{})
		return err
	}},
	"CreateArtifactTask": {"CreateArtifactTask", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.CreateArtifactTask(ctx, &gc.CreateArtifactTaskRequest{})
		return err
	}},
	"CreateArtifactTasks": {"CreateArtifactTasksBulk", writeRPC, func(ctx context.Context, api API) error {
		_, err := api.CreateArtifactTasks(ctx, &gc.CreateArtifactTasksBulkRequest{})
		return err
	}},
}

// composedMethods send RPCs only through the classified methods above.
var composedMethods = map[string]bool{
	"FetchPipelineSpecFromRun": true,
	"UpdateStatuses":           true,
}

func TestEveryAPIMethodIsClassifiedAsReadOrWrite(t *testing.T) {
	apiType := reflect.TypeOf((*API)(nil)).Elem()
	var unclassified []string
	for i := 0; i < apiType.NumMethod(); i++ {
		name := apiType.Method(i).Name
		if _, ok := classifiedMethods[name]; !ok && !composedMethods[name] {
			unclassified = append(unclassified, name)
		}
	}
	sort.Strings(unclassified)
	require.Empty(t, unclassified, "classify each new runtime API method as a retried read or a single-attempt write")
	require.Equal(t, apiType.NumMethod(), len(classifiedMethods)+len(composedMethods))
}

// unavailableServer answers every RPC with UNAVAILABLE and counts the calls per method.
type unavailableServer struct {
	mu    sync.Mutex
	calls map[string]int
}

func (s *unavailableServer) intercept(_ context.Context, _ any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[path.Base(info.FullMethod)]++
	return nil, status.Error(codes.Unavailable, "temporarily unavailable")
}

func (s *unavailableServer) count(rpc string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[rpc]
}

func TestClientAdapterRetriesOnlyReadsOnUnavailable(t *testing.T) {
	server := &unavailableServer{calls: map[string]int{}}
	client := newTestGRPCClient(t, func(s *grpc.Server) {
		gc.RegisterRunServiceServer(s, gc.UnimplementedRunServiceServer{})
		gc.RegisterArtifactServiceServer(s, gc.UnimplementedArtifactServiceServer{})
		gc.RegisterPipelineServiceServer(s, gc.UnimplementedPipelineServiceServer{})
	}, grpc.UnaryInterceptor(server.intercept))

	policy := slowPolicy()
	noBackoff := retryEnv{
		sleep:  func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
		jitter: func(time.Duration) time.Duration { return 0 },
	}
	api := newWithRetryEnv(client, policy, noBackoff)

	for name, method := range classifiedMethods {
		t.Run(name, func(t *testing.T) {
			err := method.call(context.Background(), api)
			require.Equal(t, codes.Unavailable, status.Code(err))

			want := 1
			if method.kind == readRPC {
				want = policy.MaxAttempts
			}
			require.Equal(t, want, server.count(method.rpc), "%s sends %s", name, method.rpc)
		})
	}
}
