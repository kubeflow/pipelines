// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newHandler(t *testing.T, execute DriverExecutor) http.Handler {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("agent-token\n"), 0600))
	handler, err := newHandlerWithResolver(context.Background(), tokenPath, execute, func(context.Context, driverapi.DriverPluginArgs) (string, error) { return "workflow-uid/node", nil })
	require.NoError(t, err)
	return handler
}

func requestBody(t *testing.T, kind string) string {
	t.Helper()
	// Include empty-but-required fields, which cannot be represented by DTOs
	// with omitempty when checking field presence.
	args := map[string]any{
		"type": kind, "pipeline_name": "pipeline", "run_id": "run", "run_name": "workflow",
		"run_display_name": "run", "parent_task_id": "", "task_name": "task", "namespace": "ns",
		"iteration_index": "-1", "ml_pipeline_server_address": "ml-pipeline", "ml_pipeline_server_port": "8887",
		"log_level": "1", "publish_logs": "true", "cache_disabled": false, "ml_pipeline_tls_enabled": false,
		"http_proxy": "", "https_proxy": "", "no_proxy": "",
		"runtime_args": `{"setting":"value"}`,
	}
	if kind == "CONTAINER" {
		args["kubernetes_config"] = ""
	}
	if kind == "ROOT_DAG" {
		args["runtime_config"] = ""
	}
	body, err := json.Marshal(map[string]any{"template": map[string]any{"plugin": map[string]any{
		"driver-plugin": map[string]any{"args": args},
	}}})
	require.NoError(t, err)
	return string(body)
}

func authenticatedRequest(method, body string) *http.Request {
	request := httptest.NewRequest(method, "/api/v1/template.execute", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer agent-token")
	return request
}

func TestExecutePluginSuccessAndFailurePreserveOutputs(t *testing.T) {
	count, cached, condition := 3, false, false
	execution := &driver.Execution{
		TaskID: "task-id", IterationCount: &count, Cached: &cached, Condition: &condition,
		PodSpecPatch:  `{"containers":[{"name":"main"}]}`,
		ExecutorInput: &pipelinespec.ExecutorInput{},
	}
	for _, tc := range []struct {
		name      string
		execution *driver.Execution
		err       error
		phase     string
	}{
		{"success", execution, nil, "Succeeded"},
		{"failure preserves outputs", execution, assert.AnError, "Failed"},
		{"failure without execution", nil, assert.AnError, "Failed"},
		{"nil execution success", nil, nil, "Succeeded"},
		{"executor input serialization failure", &driver.Execution{
			TaskID: "task-id", ExecutorInput: &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{"invalid": {Kind: &structpb.Value_StringValue{StringValue: string([]byte{0xff})}}},
			}},
		}, nil, "Failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			handler := newHandler(t, func(ctx context.Context, args driverapi.DriverPluginArgs) (*driver.Execution, error) {
				calls++
				assert.NotNil(t, ctx)
				assert.Equal(t, "CONTAINER", args.Type)
				assert.Equal(t, "value", args.RuntimeArgs["setting"])
				return tc.execution, tc.err
			})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, requestBody(t, "CONTAINER")))
			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.Equal(t, 1, calls)
			var response driverapi.DriverResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, tc.phase, response.Node.Phase)
			if tc.phase == "Failed" {
				assert.Contains(t, response.Node.Message, "unable to drive execution")
			} else {
				assert.Empty(t, response.Node.Message)
			}
			outputs := map[string]string{}
			for _, output := range response.Node.Outputs.Parameters {
				outputs[output.Name] = output.Value
			}
			if tc.execution == execution {
				assert.Equal(t, map[string]string{"task-id": "task-id", "iteration-count": "3", "cached-decision": "false", "condition": "false", "pod-spec-patch": execution.PodSpecPatch}, outputs)
			} else if tc.execution == nil {
				assert.Empty(t, outputs)
				assert.Contains(t, recorder.Body.String(), `"parameters":[]`)
			} else {
				assert.Equal(t, "task-id", outputs["task-id"])
				assert.Contains(t, response.Node.Message, "failed to marshal ExecutorInput")
			}
		})
	}
}

func TestExecutePluginDefaultOutputs(t *testing.T) {
	for _, kind := range []string{"ROOT_DAG", "DAG", "CONTAINER"} {
		t.Run(kind, func(t *testing.T) {
			handler := newHandler(t, func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) {
				return &driver.Execution{}, nil
			})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authenticatedRequest(http.MethodPost, requestBody(t, kind)))
			require.Equal(t, http.StatusOK, recorder.Code)
			var response driverapi.DriverResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			want := []driverapi.Parameter{{Name: "condition", Value: "nil"}, {Name: "pod-spec-patch", Value: ""}}
			if kind != "CONTAINER" {
				want = append([]driverapi.Parameter{{Name: "iteration-count", Value: "0"}}, want...)
			}
			assert.Equal(t, want, response.Node.Outputs.Parameters)
		})
	}
}

type trackedBody struct {
	io.Reader
	closed   bool
	readErr  error
	closeErr error
}

func (b *trackedBody) Read(p []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	return b.Reader.Read(p)
}
func (b *trackedBody) Close() error { b.closed = true; return b.closeErr }

func TestExecutePluginRejectsMalformedAndMethodRequests(t *testing.T) {
	valid := requestBody(t, "CONTAINER")
	for _, tc := range []struct {
		name, method, body, message string
		status                      int
		readErr                     error
	}{
		{"method", http.MethodGet, valid, "Method not allowed", http.StatusMethodNotAllowed, nil},
		{"empty", http.MethodPost, "", "failed to parse driver request body", http.StatusBadRequest, nil},
		{"malformed", http.MethodPost, "{", "failed to parse driver request body", http.StatusBadRequest, nil},
		{"missing template", http.MethodPost, `{}`, "Template is empty", http.StatusBadRequest, nil},
		{"missing plugin", http.MethodPost, `{"template":{}}`, "Plugin is empty", http.StatusBadRequest, nil},
		{"null driver", http.MethodPost, `{"template":{"plugin":{"driver-plugin":null}}}`, "DriverPlugin must be an object", http.StatusBadRequest, nil},
		{"missing args", http.MethodPost, `{"template":{"plugin":{"driver-plugin":{}}}}`, "Args is empty", http.StatusBadRequest, nil},
		{"null args", http.MethodPost, `{"template":{"plugin":{"driver-plugin":{"args":null}}}}`, "Args is empty", http.StatusBadRequest, nil},
		{"array args", http.MethodPost, `{"template":{"plugin":{"driver-plugin":{"args":[]}}}}`, "failed to parse driver request args", http.StatusBadRequest, nil},
		{"missing required field", http.MethodPost, strings.Replace(valid, `"http_proxy":"",`, "", 1), "http_proxy", http.StatusBadRequest, nil},
		{"unset proxy", http.MethodPost, strings.Replace(valid, `"http_proxy":""`, `"http_proxy":"unset"`, 1), "http_proxy", http.StatusBadRequest, nil},
		{"unknown driver", http.MethodPost, strings.Replace(valid, "CONTAINER", "UNKNOWN", 1), "unknown driver type", http.StatusBadRequest, nil},
		{"read failure", http.MethodPost, valid, "failed to read driver request body", http.StatusBadRequest, assert.AnError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newHandler(t, func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) {
				t.Fatal("invalid request must not execute the driver")
				return nil, nil
			})
			request := authenticatedRequest(tc.method, "")
			body := &trackedBody{Reader: strings.NewReader(tc.body), readErr: tc.readErr, closeErr: assert.AnError}
			request.Body = body
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assert.Equal(t, tc.status, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.message)
			assert.True(t, body.closed)
		})
	}
}

func TestNewHandlerAuthentication(t *testing.T) {
	for _, header := range []string{"", "Bearer wrong-token", "agent-token", "bearer agent-token", "Bearer agent-token"} {
		t.Run(header, func(t *testing.T) {
			calls := 0
			handler := newHandler(t, func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error) { calls++; return nil, nil })
			request := authenticatedRequest(http.MethodPost, requestBody(t, "ROOT_DAG"))
			request.Header.Set("Authorization", header)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if header == "Bearer agent-token" {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, 1, calls)
			} else {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
				assert.Zero(t, calls)
			}
		})
	}
	tokenPath := filepath.Join(t.TempDir(), "token")
	_, err := NewHandler(tokenPath, nil)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, os.WriteFile(tokenPath, []byte(" \n"), 0600))
	_, err = NewHandler(tokenPath, nil)
	require.ErrorContains(t, err, "token is empty")
}
