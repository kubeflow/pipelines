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

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/stretchr/testify/require"
)

func faultRequest(t *testing.T, task, generation string, finalizer bool) *http.Request {
	t.Helper()
	args := driverapi.DriverPluginArgs{DriverType: driverapi.ContainerDriver, Namespace: "kubeflow", RunID: "test-run", TaskName: task, DriverRetryGeneration: generation, DriverRetryFinalize: finalizer, KubernetesConfig: "{}", LogLevel: "1", PublishLogs: "false"}
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	var values map[string]any
	require.NoError(t, json.Unmarshal(encoded, &values))
	plugin, err := driverapi.EncodePluginValue(values)
	require.NoError(t, err)
	_, owned, err := driverapi.ParseDriverPluginArgs(plugin)
	require.True(t, owned)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{"template": map[string]any{"plugin": json.RawMessage(plugin)}})
	require.NoError(t, err)
	return httptest.NewRequest(http.MethodPost, "/api/v1/template.execute", bytes.NewReader(body))
}

func responseHandler(phase string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(driverapi.DriverResponse{Node: driverapi.Node{Phase: phase}})
	})
}

func TestCrashRequiresSuccessfulResponseAndSurvivesProcessRestart(t *testing.T) {
	directory := t.TempDir()
	first := &faultHandler{next: responseHandler("Running"), directory: directory, crash: func() { panic("process exited") }}
	running := httptest.NewRecorder()
	first.ServeHTTP(running, faultRequest(t, "restart-driver", "0", false))
	require.Contains(t, running.Body.String(), "Running")
	require.NoFileExists(t, filepath.Join(directory, "lost-response-test-run"))
	first.next = responseHandler("Succeeded")
	lost := httptest.NewRecorder()
	require.PanicsWithValue(t, "process exited", func() { first.ServeHTTP(lost, faultRequest(t, "restart-driver", "0", false)) })
	require.Empty(t, lost.Body.String(), "successful response must not reach Argo before the process exits")
	committed, err := os.ReadFile(filepath.Join(directory, "lost-response-test-run"))
	require.NoError(t, err)
	require.Contains(t, string(committed), "Succeeded")
	restarted := &faultHandler{next: responseHandler("Succeeded"), directory: directory, crash: func() { t.Fatal("must not crash again after restart") }}
	replay := httptest.NewRecorder()
	restarted.ServeHTTP(replay, faultRequest(t, "restart-driver", "0", false))
	require.Equal(t, string(committed), replay.Body.String())
	restarted.next = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"node":{"phase":"Succeeded","outputs":{"parameters":[{"name":"task-id","value":"changed"}]},"message":""}}`))
	})
	changed := httptest.NewRecorder()
	restarted.ServeHTTP(changed, faultRequest(t, "restart-driver", "0", false))
	require.Contains(t, changed.Body.String(), "replay changed the committed task ID or handoff outputs")
}

func TestExhaustionFaultDoesNotPoisonFinalizerOrManualRetry(t *testing.T) {
	handler := &faultHandler{next: responseHandler("Succeeded"), directory: t.TempDir()}
	for _, tc := range []struct {
		name, task, generation, phase string
		finalizer                     bool
	}{
		{name: "automatic retry", task: "exhaust-driver", generation: "0", phase: "Failed"},
		{name: "finalizer", task: "exhaust-driver", generation: "0", phase: "Succeeded", finalizer: true},
		{name: "manual retry", task: "exhaust-driver", generation: "1", phase: "Succeeded"},
		{name: "other task", task: "cleanup", generation: "0", phase: "Succeeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, faultRequest(t, tc.task, tc.generation, tc.finalizer))
			var result driverapi.DriverResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			require.Equal(t, tc.phase, result.Node.Phase)
		})
	}
}
