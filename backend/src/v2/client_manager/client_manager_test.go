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

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
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
