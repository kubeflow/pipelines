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

package mlflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Helpers ----

func setupSAToken(t *testing.T) func() {
	t.Helper()
	setupFakeKubernetesConfig(t, "test-sa-token")
	return func() {} // cleanup handled by t.Cleanup in setupFakeKubernetesConfig
}

func testPluginConfig(endpoint string) *commonmlflow.PluginConfig {
	enabled := true
	return &commonmlflow.PluginConfig{
		Endpoint: endpoint,
		Timeout:  "10s",
		Settings: &commonmlflow.MLflowPluginSettings{WorkspacesEnabled: &enabled},
	}
}

func testPendingRun(id, displayName string) *apiserverPlugins.PendingRun {
	return &apiserverPlugins.PendingRun{
		RunID:       id,
		DisplayName: displayName,
		Namespace:   "ns1",
	}
}

func testPersistedRun(id string) *apiserverPlugins.PersistedRun {
	return &apiserverPlugins.PersistedRun{
		RunID:         id,
		Namespace:     "ns1",
		PluginsOutput: make(map[string]*apiv2beta1.PluginOutput),
	}
}

func testPersistedRunWithPluginOutput(id string, pluginOutput *apiv2beta1.PluginOutput) *apiserverPlugins.PersistedRun {
	r := testPersistedRun(id)
	if pluginOutput != nil {
		r.PluginsOutput[PluginName] = pluginOutput
	}
	return r
}

// ---- OnBeforeRunCreation tests ----

func TestOnBeforeRunCreation_NilConfig_ReturnsNil(t *testing.T) {
	handler := NewHandler(&MLflowPluginInput{ExperimentName: "Default"}, "ns1")
	output, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), nil)
	require.NoError(t, err)
	assert.Nil(t, output)
	assert.Empty(t, handler.RunStartEnv)
}

func TestOnBeforeRunCreation_Disabled_ReturnsNil(t *testing.T) {
	handler := NewHandler(&MLflowPluginInput{Disabled: true}, "ns1")
	output, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), testPluginConfig("http://localhost"))
	require.NoError(t, err)
	assert.Nil(t, output)
}

func TestOnBeforeRunCreation_NilInput_ReturnsNil(t *testing.T) {
	handler := NewHandler(nil, "ns1")
	output, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), testPluginConfig("http://localhost"))
	require.NoError(t, err)
	assert.Nil(t, output)
}

func TestOnBeforeRunCreation_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"exp-42","name":"Default"}}`))
		case "/api/2.0/mlflow/runs/create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-run-1"}}}`))
		case "/api/2.0/mlflow/runs/set-tag":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, nil) })

	handler := NewHandler(&MLflowPluginInput{ExperimentName: "Default"}, "ns1")

	run := testPendingRun("kfp-run-1", "my-run")
	output, err := handler.OnBeforeRunCreation(context.Background(), run, testPluginConfig(server.URL))
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Contains(t, output.Entries, EntryExperimentID)
	assert.Equal(t, "exp-42", output.Entries[EntryExperimentID].Value.GetStringValue())
	assert.Contains(t, output.Entries, EntryRootRunID)
	assert.Equal(t, "mlflow-run-1", output.Entries[EntryRootRunID].Value.GetStringValue())

	// Verify RunStartEnv contains single KFP_MLFLOW_CONFIG JSON env var
	require.NotEmpty(t, handler.RunStartEnv)
	assert.Contains(t, handler.RunStartEnv, commonmlflow.EnvMLflowConfig)

	var rtCfg commonmlflow.MLflowRuntimeConfig
	require.NoError(t, json.Unmarshal([]byte(handler.RunStartEnv[commonmlflow.EnvMLflowConfig]), &rtCfg))
	assert.Contains(t, rtCfg.Endpoint, server.URL)
	assert.Equal(t, "ns1", rtCfg.Workspace)
	assert.Equal(t, "mlflow-run-1", rtCfg.ParentRunID)
	assert.Equal(t, "exp-42", rtCfg.ExperimentID)
	assert.Equal(t, "kubernetes", rtCfg.AuthType)
	assert.False(t, rtCfg.InjectUserEnvVars, "InjectUserEnvVars should default to false")
}

func TestOnBeforeRunCreation_MLflowFailure_ReturnsFailedOutput(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, nil) })

	handler := NewHandler(&MLflowPluginInput{ExperimentName: "Default"}, "ns1")

	run := testPendingRun("kfp-run-2", "run-2")
	output, err := handler.OnBeforeRunCreation(context.Background(), run, testPluginConfig(server.URL))
	require.Error(t, err)
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, output.State)
	assert.NotEmpty(t, output.StateMessage)
}

// ---- OnRunEnd / syncOnRunTerminal tests ----

func TestOnRunEnd_NilRun_ReturnsNil(t *testing.T) {
	handler := NewHandler(nil, "ns1")
	err := handler.OnRunEnd(context.Background(), nil, testPluginConfig("http://localhost"))
	require.NoError(t, err)
}

func TestOnRunEnd_NoPluginOutput_ReturnsNil(t *testing.T) {
	handler := NewHandler(nil, "ns1")
	run := testPersistedRun("r1")
	err := handler.OnRunEnd(context.Background(), run, testPluginConfig("http://localhost"))
	require.NoError(t, err)
}

func TestOnRunEnd_MissingRootRunID_SetsFailedState(t *testing.T) {
	handler := NewHandler(nil, "ns1")

	// Build a run with plugin output that has no root_run_id
	pluginOutput := SuccessfulPluginOutput("42", "Default", "", "", "")
	run := testPersistedRunWithPluginOutput("r-missing-root", pluginOutput)

	err := handler.OnRunEnd(context.Background(), run, testPluginConfig("http://localhost"))
	require.NoError(t, err)

	// Verify the plugin output was updated in place
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "missing parent root_run_id")
}

func TestOnRunEnd_NilConfig_SetsFailedState(t *testing.T) {
	handler := NewHandler(nil, "ns1")

	pluginOutput := SuccessfulPluginOutput("42", "Default", "parent-1", "", "")
	run := testPersistedRunWithPluginOutput("r-nil-config", pluginOutput)

	err := handler.OnRunEnd(context.Background(), run, nil)
	require.NoError(t, err)

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "config unavailable")
}

func TestOnRunEnd_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	var updateCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			updateCalls = append(updateCalls, string(body))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	handler := NewHandler(nil, "ns1")

	pluginOutput := SuccessfulPluginOutput("exp-1", "Default", "mlflow-parent-1", "", server.URL)
	run := testPersistedRunWithPluginOutput("r-end-1", pluginOutput)
	run.State = "SUCCEEDED"

	err := handler.OnRunEnd(context.Background(), run, testPluginConfig(server.URL))
	require.NoError(t, err)

	// Parent run should have been updated
	require.NotEmpty(t, updateCalls)
	assert.Contains(t, updateCalls[0], "mlflow-parent-1")
	assert.Contains(t, updateCalls[0], "FINISHED") // SUCCEEDED maps to FINISHED

	// Plugin output should be updated in place
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result.State)
}

// ---- HandleRetry tests ----

func TestHandleRetry_NoPluginOutput_NoOp(t *testing.T) {
	handler := NewHandler(nil, "ns1")
	run := testPersistedRun("r-retry-noop")

	handler.HandleRetry(context.Background(), run, testPluginConfig("http://localhost"))
	// No plugin output → nothing to do
	assert.Empty(t, run.PluginsOutput)
}

func TestHandleRetry_MissingRootRunID_SetsFailedState(t *testing.T) {
	handler := NewHandler(nil, "ns1")

	pluginOutput := SuccessfulPluginOutput("42", "Default", "", "", "")
	run := testPersistedRunWithPluginOutput("r-retry-no-root", pluginOutput)

	handler.HandleRetry(context.Background(), run, testPluginConfig("http://localhost"))

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "missing parent root_run_id")
}

func TestHandleRetry_NilConfig_SetsFailedState(t *testing.T) {
	handler := NewHandler(nil, "ns1")

	pluginOutput := SuccessfulPluginOutput("42", "Default", "parent-1", "", "")
	run := testPersistedRunWithPluginOutput("r-retry-nil-config", pluginOutput)

	handler.HandleRetry(context.Background(), run, nil)

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "config unavailable")
}

func TestHandleRetry_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	var updatePayloads []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			updatePayloads = append(updatePayloads, string(body))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			// Return one failed nested run
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-1","status":"FAILED"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	handler := NewHandler(nil, "ns1")

	pluginOutput := FailedPluginOutput("exp-1", "Default", "parent-1", "", server.URL, "previous failure")
	run := testPersistedRunWithPluginOutput("r-retry-ok", pluginOutput)

	handler.HandleRetry(context.Background(), run, testPluginConfig(server.URL))

	// Parent reopened + nested-1 reopened = 2 update calls
	require.Len(t, updatePayloads, 2)
	assert.Contains(t, updatePayloads[0], "parent-1")
	assert.Contains(t, updatePayloads[0], "RUNNING")
	assert.Contains(t, updatePayloads[1], "nested-1")
	assert.Contains(t, updatePayloads[1], "RUNNING")

	// Plugin output updated in place
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result.State)
}

// ---- BuildKFPRunURL tests ----

func TestBuildKFPRunURL(t *testing.T) {
	tests := []struct {
		name       string
		runID      string
		kfpBaseURL string
		wantURL    string
	}{
		{
			name:    "empty runID returns empty",
			runID:   "",
			wantURL: "",
		},
		{
			name:    "no base URL returns relative path",
			runID:   "abc",
			wantURL: "/#/runs/details/abc",
		},
		{
			name:       "with base URL returns absolute URL",
			runID:      "abc",
			kfpBaseURL: "https://kfp.example.com",
			wantURL:    "https://kfp.example.com/#/runs/details/abc",
		},
		{
			name:       "trailing slash on base URL is trimmed",
			runID:      "abc",
			kfpBaseURL: "https://kfp.example.com/",
			wantURL:    "https://kfp.example.com/#/runs/details/abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildKFPRunURL(tt.runID, tt.kfpBaseURL)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

// ---- ModelToPersistedRun tests ----

func TestModelToPersistedRun_NilModel(t *testing.T) {
	_, err := ModelToPersistedRun(nil, "ns1")
	require.Error(t, err)
}

func TestModelToPersistedRun_BasicFields(t *testing.T) {
	pluginsJSON := `{"mlflow":{"entries":{"root_run_id":{"value":"parent-1"}},"state":"PLUGIN_SUCCEEDED"}}`
	lt := model.LargeText(pluginsJSON)
	m := &model.Run{
		UUID: "run-123",
	}
	m.RunDetails.State = "SUCCEEDED"
	m.RunDetails.FinishedAtInSec = 1700000000
	m.RunDetails.PluginsOutputString = &lt

	pr, err := ModelToPersistedRun(m, "ns1")
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.Equal(t, "run-123", pr.RunID)
	assert.Equal(t, "ns1", pr.Namespace)
	assert.Equal(t, "SUCCEEDED", pr.State)
	require.NotNil(t, pr.FinishedAt)
	assert.Equal(t, int64(1700000000), pr.FinishedAt.Unix())
	require.NotNil(t, pr.PluginsOutput[PluginName])
	assert.Equal(t, "parent-1", GetParentRunID(pr.PluginsOutput[PluginName]))
}

// ---- SerializePluginsOutput / DeserializePluginsOutput tests ----

func TestSerializeDeserializePluginsOutput_RoundTrip(t *testing.T) {
	original := map[string]*apiv2beta1.PluginOutput{
		"mlflow":       SuccessfulPluginOutput("exp-1", "Default", "parent-1", "", ""),
		"other_plugin": {State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	lt, err := SerializePluginsOutput(original)
	require.NoError(t, err)
	require.NotNil(t, lt)
	assert.Contains(t, string(*lt), "mlflow")
	assert.Contains(t, string(*lt), "other_plugin")

	result, err := DeserializePluginsOutput(lt)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotNil(t, result["mlflow"])
	assert.NotNil(t, result["other_plugin"])
	assert.Equal(t, "parent-1", GetParentRunID(result["mlflow"]))
}

// ---- SyncParentAndNestedRuns pagination test ----

func TestSyncParentAndNestedRuns_Pagination(t *testing.T) {
	var updateCalls []string
	// Track search calls per parent run ID to handle pagination and recursive child searches.
	searchCallsByParent := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			updateCalls = append(updateCalls, string(body))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			body, _ := io.ReadAll(r.Body)
			// Determine which parent run this search is for by inspecting the filter.
			parentID := "parent-1"
			if strings.Contains(string(body), "nested-p1") {
				parentID = "nested-p1"
			} else if strings.Contains(string(body), "nested-p2") {
				parentID = "nested-p2"
			}
			searchCallsByParent[parentID]++
			w.WriteHeader(http.StatusOK)
			switch parentID {
			case "parent-1":
				if searchCallsByParent[parentID] == 1 {
					// First page: one nested run + next_page_token
					_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-p1","status":"RUNNING"}}],"next_page_token":"page2"}`))
				} else {
					// Second page: one nested run, no more pages
					_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-p2","status":"RUNNING"}}]}`))
				}
			default:
				// Nested runs have no children
				_, _ = w.Write([]byte(`{"runs":[]}`))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	setupFakeKubernetesConfig(t, "sa-token")

	enabled := true
	requestCfg := &ResolvedConfig{
		Config:   &commonmlflow.PluginConfig{Endpoint: server.URL, Timeout: "10s"},
		Settings: &commonmlflow.MLflowPluginSettings{WorkspacesEnabled: &enabled},
	}
	mlflowCtx, err := BuildMLflowRunRequestContext(context.Background(), "ns1", requestCfg)
	require.NoError(t, err)

	endTime := int64(1700000000000)
	syncErrors := SyncParentAndNestedRuns(context.Background(), mlflowCtx, "parent-1", "exp-1", RunSyncModeTerminal, "FINISHED", &endTime)
	assert.Empty(t, syncErrors)

	// 2 search calls for parent-1 (pagination) + 1 each for nested-p1 and nested-p2 (no children) = 4 total
	assert.Equal(t, 2, searchCallsByParent["parent-1"])
	assert.Equal(t, 1, searchCallsByParent["nested-p1"])
	assert.Equal(t, 1, searchCallsByParent["nested-p2"])
	// 1 parent update + 2 nested updates = 3 total
	assert.Len(t, updateCalls, 3)
	// Verify nested runs were updated
	found := strings.Join(updateCalls, " | ")
	assert.Contains(t, found, "nested-p1")
	assert.Contains(t, found, "nested-p2")
}
