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

package executorplugin

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/require"
)

func TestServerDrainsDetachedExecution(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup completes", true: "cleanup times out"}[timeout], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started, cleanup, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
			h := asyncHandler(t, ctx, func(ctx context.Context, _ driverapi.DriverPluginArgs) (*driver.Execution, error) {
				defer close(finished)
				close(started)
				<-ctx.Done()
				close(cleanup)
				<-release
				return nil, ctx.Err()
			}, func(context.Context, driverapi.DriverPluginArgs) (string, error) { return "attempt", nil })
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			grace := time.Second
			if timeout {
				grace = 30 * time.Millisecond
			}
			stopped := make(chan error, 1)
			go func() { stopped <- serveHTTP(ctx, cancel, listener, h, grace) }()
			req := authenticatedRequest(http.MethodPost, requestBody(t, "CONTAINER"))
			req.URL.Scheme = "http"
			req.URL.Host = listener.Addr().String()
			req.URL.Path = "/api/v1/template.execute"
			req.RequestURI = ""
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			response.Body.Close()
			<-started
			cancel()
			<-cleanup
			if timeout {
				require.ErrorContains(t, <-stopped, "cleanup did not finish")
				close(release)
			} else {
				select {
				case err := <-stopped:
					t.Fatalf("returned before detached cleanup: %v", err)
				case <-time.After(30 * time.Millisecond):
				}
				close(release)
				require.NoError(t, <-stopped)
			}
			<-finished
		})
	}
}

func TestServerWaitsForHTTPShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})
	stopped := make(chan error, 1)
	go func() { stopped <- serveHTTP(ctx, cancel, listener, h, time.Second) }()
	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String() + "/api/v1/template.execute")
		if response != nil {
			response.Body.Close()
		}
		requestDone <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-stopped:
		t.Fatalf("returned before HTTP drain: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-requestDone)
	require.NoError(t, <-stopped)
}

func TestServeReportsInvalidListenAddress(t *testing.T) {
	token := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(token, []byte("token"), 0600))
	err := Serve(context.Background(), "not-a-listen-address", token)
	require.ErrorContains(t, err, "cannot listen")
}
