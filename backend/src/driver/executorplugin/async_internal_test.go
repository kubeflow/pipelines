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

package executorplugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	wf "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	ep "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	rpc "github.com/argoproj/argo-workflows/v4/workflow/executor/plugins/rpc"
	client "github.com/argoproj/argo-workflows/v4/workflow/util/plugin"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

func asyncHandler(t *testing.T, ctx context.Context, run DriverExecutor, resolve attemptResolver) http.Handler {
	t.Helper()
	token := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(token, []byte("agent-token"), 0600))
	h, err := newHandlerWithResolver(ctx, token, run, resolve)
	require.NoError(t, err)
	return h
}

func rpcArgs(t *testing.T) ep.ExecuteTemplateArgs {
	t.Helper()
	var args ep.ExecuteTemplateArgs
	require.NoError(t, json.Unmarshal([]byte(requestBody(t, "CONTAINER")), &args))
	return args
}

func TestRPCSlowSingleFlightAndReplay(t *testing.T) {
	ctx, cancel := context.WithCancel(logging.TestContext(context.Background()))
	defer cancel()
	release := make(chan struct{})
	var calls atomic.Int32
	h := asyncHandler(t, ctx, func(ctx context.Context, _ driverapi.DriverPluginArgs) (*driver.Execution, error) {
		calls.Add(1)
		select {
		case <-release:
			return &driver.Execution{TaskID: "once"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, func(context.Context, driverapi.DriverPluginArgs) (string, error) { return "uid/node", nil })
	server := httptest.NewServer(h)
	defer server.Close()
	args := rpcArgs(t)
	short := client.New(server.URL, "agent-token", time.Second, wait.Backoff{Steps: 1})
	var running ep.ExecuteTemplateReply
	require.NoError(t, short.Call(ctx, "template.execute", args, &running))
	require.Equal(t, wf.NodeRunning, running.Node.Phase)
	require.Equal(t, time.Second, running.GetRequeue())
	// Independent pinned clients simulate simultaneous agent transport retries.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var reply ep.ExecuteTemplateReply
			err := rpc.New(server.URL, "agent-token").ExecuteTemplate(ctx, args, &reply)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, calls.Load())
	close(release)
	pinned := rpc.New(server.URL, "agent-token")
	require.Eventually(t, func() bool {
		var reply ep.ExecuteTemplateReply
		err := pinned.ExecuteTemplate(ctx, args, &reply)
		return err == nil && reply.Node.Phase == wf.NodeSucceeded
	}, time.Second, time.Millisecond)
	for range 3 {
		var reply ep.ExecuteTemplateReply
		require.NoError(t, pinned.ExecuteTemplate(ctx, args, &reply))
		require.Equal(t, wf.NodeSucceeded, reply.Node.Phase)
		require.Equal(t, "once", reply.Node.Outputs.Parameters[0].Value.String())
	}
	require.EqualValues(t, 1, calls.Load())
}

func TestRPCLostResponseAndDistinctArguments(t *testing.T) {
	var calls atomic.Int32
	h := asyncHandler(t, context.Background(), func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) {
		calls.Add(1)
		return &driver.Execution{TaskID: "created"}, nil
	}, func(context.Context, driverapi.DriverPluginArgs) (string, error) { return "uid/node", nil })
	var drop atomic.Bool
	drop.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if drop.CompareAndSwap(true, false) {
			h.ServeHTTP(httptest.NewRecorder(), r)
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer server.Close()
	pinned := rpc.New(server.URL, "agent-token")
	args := rpcArgs(t)
	var first ep.ExecuteTemplateReply
	_ = pinned.ExecuteTemplate(logging.TestContext(context.Background()), args, &first)
	var replay ep.ExecuteTemplateReply
	require.NoError(t, pinned.ExecuteTemplate(logging.TestContext(context.Background()), args, &replay))
	require.EqualValues(t, 1, calls.Load())
	for _, replacement := range []struct{ old, new string }{{`"task_name":"task"`, `"task_name":"other"`}, {`"iteration_index":"-1"`, `"iteration_index":"2"`}, {`"log_level":"1"`, `"log_level":"4"`}} {
		var distinct ep.ExecuteTemplateArgs
		require.NoError(t, json.Unmarshal([]byte(strings.Replace(requestBody(t, "CONTAINER"), replacement.old, replacement.new, 1)), &distinct))
		var reply ep.ExecuteTemplateReply
		require.NoError(t, pinned.ExecuteTemplate(logging.TestContext(context.Background()), distinct, &reply))
		require.Equal(t, wf.NodeSucceeded, reply.Node.Phase)
	}
	require.EqualValues(t, 4, calls.Load())
}

func TestRPCCancellationErrorsPanicsAndAttemptChange(t *testing.T) {
	for _, mode := range []string{"error", "panic", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var count atomic.Int32
			var attempt atomic.Int32
			started := make(chan struct{}, 2)
			finished := make(chan struct{}, 2)
			h := asyncHandler(t, ctx, func(ctx context.Context, _ driverapi.DriverPluginArgs) (*driver.Execution, error) {
				defer func() { finished <- struct{}{} }()
				count.Add(1)
				started <- struct{}{}
				switch mode {
				case "panic":
					panic("boom")
				case "cancel":
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return &driver.Execution{TaskID: "failed-task"}, errors.New("failed")
			}, func(context.Context, driverapi.DriverPluginArgs) (string, error) {
				return string(rune('0' + attempt.Load())), nil
			})
			body := requestBody(t, "CONTAINER")
			requestCtx, requestCancel := context.WithCancel(context.Background())
			req := authenticatedRequest(http.MethodPost, body).WithContext(requestCtx)
			done := make(chan struct{})
			go func() { h.ServeHTTP(httptest.NewRecorder(), req); close(done) }()
			<-started
			requestCancel()
			<-done
			if mode == "cancel" {
				require.EqualValues(t, 1, count.Load())
				select {
				case <-finished:
					t.Fatal("HTTP cancellation stopped driver work")
				default:
				}
				cancel()
				select {
				case <-finished:
				case <-time.After(time.Second):
					t.Fatal("service cancellation did not stop driver work")
				}
				return
			}
			for range 2 {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, authenticatedRequest(http.MethodPost, body))
				var reply driverapi.DriverResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &reply))
				require.Equal(t, "Failed", reply.Node.Phase)
			}
			require.EqualValues(t, 1, count.Load())
			attempt.Store(1)
			h.ServeHTTP(httptest.NewRecorder(), authenticatedRequest(http.MethodPost, body))
			require.EqualValues(t, 2, count.Load())
		})
	}
}

func TestRPCUnownedTemplateAllowsNextPlugin(t *testing.T) {
	h := newHandler(t, func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) {
		t.Error("unowned execution")
		return nil, nil
	})
	server := httptest.NewServer(h)
	defer server.Close()
	var args ep.ExecuteTemplateArgs
	require.NoError(t, json.Unmarshal([]byte(`{"template":{"plugin":{"hello":{}}}}`), &args))
	var reply ep.ExecuteTemplateReply
	require.NoError(t, rpc.New(server.URL, "agent-token").ExecuteTemplate(logging.TestContext(context.Background()), args, &reply))
	require.Nil(t, reply.Node)
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"node":{"phase":"Succeeded"}}`)) }))
	defer owner.Close()
	require.NoError(t, rpc.New(owner.URL, "agent-token").ExecuteTemplate(logging.TestContext(context.Background()), args, &reply))
	require.Equal(t, wf.NodeSucceeded, reply.Node.Phase)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authenticatedRequest(http.MethodPost, `{"template":{"plugin":{"hello":{}}}}`))
	require.Equal(t, "{}", rec.Body.String())
}
