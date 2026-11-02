// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client_manager_test

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	runtimeapi "github.com/kubeflow/pipelines/backend/api/runtime/go_client"
	gc "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/client-go/kubernetes/fake"
)

type tlsRunServer struct {
	gc.UnimplementedRunServiceServer
	calls atomic.Int32
}

func (s *tlsRunServer) GetRun(context.Context, *gc.GetRunRequest) (*gc.Run, error) {
	s.calls.Add(1)
	return &gc.Run{RunId: "run"}, nil
}

func TestClientManagerTLSTransport(t *testing.T) {
	certServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cert := certServer.TLS.Certificates[0]
	certServer.Close()
	path := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600))
	for _, mode := range []string{"tls custom ca", "tls system roots", "plaintext peer"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			var options []grpc.ServerOption
			if mode != "plaintext peer" {
				options = append(options, grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}})))
			}
			server := grpc.NewServer(options...)
			api := &tlsRunServer{}
			gc.RegisterRunServiceServer(server, api)
			go server.Serve(listener)
			defer server.Stop()
			host, port, err := net.SplitHostPort(listener.Addr().String())
			require.NoError(t, err)
			opts := &client_manager.Options{MLPipelineTLSEnabled: true, K8sClient: fake.NewSimpleClientset(), APIClientConfig: apiclient.FromEnvWithEndpointOverride(host, port)}
			if mode == "tls custom ca" {
				opts.CaCertPath = path
			}
			manager, err := client_manager.NewClientManager(opts)
			require.NoError(t, err)
			defer manager.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			run, err := manager.KFPAPIClient().GetRun(ctx, &gc.GetRunRequest{RunId: "run"})
			if mode == "tls custom ca" {
				require.NoError(t, err)
				require.Equal(t, "run", run.RunId)
				require.EqualValues(t, 1, api.calls.Load())
			} else {
				require.Error(t, err)
				require.Zero(t, api.calls.Load(), "TLS-enabled client must never send a plaintext RPC")
			}
		})
	}
}

func TestClientManagerRejectsInvalidSuppliedCA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.pem")
	require.NoError(t, os.WriteFile(path, []byte("invalid certificate"), 0600))
	_, err := client_manager.NewClientManager(&client_manager.Options{MLPipelineTLSEnabled: true, CaCertPath: path, K8sClient: fake.NewSimpleClientset()})
	require.Error(t, err)
}

// generationTaskServer captures the authority sent over the private driver API.
type generationTaskServer struct {
	runtimeapi.UnimplementedDriverTaskServiceServer
	requests chan *runtimeapi.WriteTaskRequest
}

func (s *generationTaskServer) CreateTask(ctx context.Context, req *runtimeapi.WriteTaskRequest) (*runtimeapi.TaskResponse, error) {
	select {
	case s.requests <- req:
		return &runtimeapi.TaskResponse{Task: req.Task}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func newGenerationTaskServer(t *testing.T) (string, <-chan *runtimeapi.WriteTaskRequest) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	api := &generationTaskServer{requests: make(chan *runtimeapi.WriteTaskRequest, 1)}
	runtimeapi.RegisterDriverTaskServiceServer(server, api)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	return listener.Addr().String(), api.requests
}

func newGenerationClientManager(t *testing.T, endpoint string, generation *int64) *client_manager.ClientManager {
	t.Helper()
	manager, err := client_manager.NewClientManager(&client_manager.Options{
		APIClientConfig:       &apiclient.Config{Endpoint: endpoint},
		DriverRetryGeneration: generation,
		K8sClient:             fake.NewSimpleClientset(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	return manager
}

func requireCreateTaskGeneration(t *testing.T, manager *client_manager.ClientManager, requests <-chan *runtimeapi.WriteTaskRequest, generation int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	task, err := manager.KFPAPIClient().CreateTask(ctx, &gc.CreateTaskRequest{
		RunId: "run",
		Task:  &gc.PipelineTask{TaskId: "task", Name: "task"},
	})
	require.NoError(t, err)
	require.Equal(t, "task", task.GetTaskId())
	select {
	case request := <-requests:
		require.Equal(t, "run", request.GetRunId())
		require.NotNil(t, request.Authority)
		require.NotNil(t, request.Authority.Generation, "generation zero must be explicit on the private API")
		require.Equal(t, generation, request.Authority.GetGeneration())
	case <-ctx.Done():
		t.Fatal("private driver API did not receive CreateTask")
	}
}

func TestClientManagerDriverRetryGenerationOverridesEnvironmentAndRemainsIsolated(t *testing.T) {
	endpoint, requests := newGenerationTaskServer(t)
	t.Setenv(util.DriverRetryGenerationEnv, "9")
	zero, one := int64(0), int64(1)
	original := newGenerationClientManager(t, endpoint, &zero)
	retried := newGenerationClientManager(t, endpoint, &one)

	// Neither later environment changes nor changes to the caller's option may
	// change the generation already bound to a request's client manager.
	t.Setenv(util.DriverRetryGenerationEnv, "17")
	zero, one = 23, 24
	for range 2 {
		requireCreateTaskGeneration(t, original, requests, 0)
		requireCreateTaskGeneration(t, retried, requests, 1)
	}
}

func TestClientManagerDriverRetryGenerationEnvironmentFallback(t *testing.T) {
	for _, tc := range []struct {
		name       string
		value      string
		generation int64
	}{
		{name: "empty defaults to zero", value: "", generation: 0},
		{name: "explicit zero", value: "0", generation: 0},
		{name: "retry generation", value: "7", generation: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, requests := newGenerationTaskServer(t)
			t.Setenv(util.DriverRetryGenerationEnv, tc.value)
			manager := newGenerationClientManager(t, endpoint, nil)
			t.Setenv(util.DriverRetryGenerationEnv, "18")
			requireCreateTaskGeneration(t, manager, requests, tc.generation)
		})
	}
}

func TestClientManagerDriverRetryGenerationOverrideIgnoresInvalidEnvironment(t *testing.T) {
	endpoint, requests := newGenerationTaskServer(t)
	t.Setenv(util.DriverRetryGenerationEnv, "invalid")
	zero := int64(0)
	manager := newGenerationClientManager(t, endpoint, &zero)
	requireCreateTaskGeneration(t, manager, requests, 0)
}

func TestClientManagerRejectsInvalidDriverRetryGeneration(t *testing.T) {
	negative := int64(-1)
	for _, tc := range []struct {
		name       string
		value      string
		generation *int64
	}{
		{name: "malformed environment", value: "invalid"},
		{name: "negative environment", value: "-1"},
		{name: "overflowing environment", value: "9223372036854775808"},
		{name: "negative override", value: "0", generation: &negative},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(util.DriverRetryGenerationEnv, tc.value)
			manager, err := client_manager.NewClientManager(&client_manager.Options{DriverRetryGeneration: tc.generation})
			require.ErrorContains(t, err, "invalid immutable driver retry generation")
			require.Nil(t, manager)
		})
	}
}
