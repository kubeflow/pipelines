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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// ---- Fakes ----

type fakeRunStoreUpdater struct {
	calls []fakeUpdateCall
}

type fakeUpdateCall struct {
	RunID         string
	PluginsOutput *model.LargeText
}

func (f *fakeRunStoreUpdater) UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error {
	f.calls = append(f.calls, fakeUpdateCall{RunID: runID, PluginsOutput: pluginsOutput})
	return nil
}

func fakeIsAuthorizedOK(_ context.Context, _ *authorizationv1.ResourceAttributes) error {
	return nil
}

// ---- Helpers ----

func setupSAToken(t *testing.T) func() {
	t.Helper()
	tokenFile, err := os.CreateTemp(t.TempDir(), "sa-token-*")
	require.NoError(t, err)
	_, err = tokenFile.WriteString("test-sa-token\n")
	require.NoError(t, err)
	require.NoError(t, tokenFile.Close())
	orig := ServiceAccountTokenPath
	ServiceAccountTokenPath = tokenFile.Name()
	return func() { ServiceAccountTokenPath = orig }
}

func setGlobalMLflowConfig(t *testing.T, endpoint string) {
	t.Helper()
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": endpoint,
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType":          "kubernetes",
				"workspacesEnabled": false,
			},
		},
	})
	t.Cleanup(func() { viper.Set("plugins", nil) })
}

func setGlobalMLflowConfigWithKFPBaseURL(t *testing.T, endpoint, kfpBaseURL string) {
	t.Helper()
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": endpoint,
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType":          "kubernetes",
				"workspacesEnabled": false,
				"kfpBaseURL":        kfpBaseURL,
			},
		},
	})
	t.Cleanup(func() { viper.Set("plugins", nil) })
}

// ---- OnRunStart tests ----

func TestOnRunStart_NoConfig_ReturnsNil(t *testing.T) {
	// No global config → OnRunStart is a no-op.
	viper.Set("plugins", nil)
	t.Cleanup(func() { viper.Set("plugins", nil) })

	handler := NewHandler(HandlerDeps{}, &PluginInput{ExperimentName: "Default"})
	output, err := handler.OnRunStart(context.Background(), &model.Run{UUID: "r1"}, "ns1")
	require.NoError(t, err)
	assert.Nil(t, output)
	assert.Empty(t, handler.RunStartEnv)
}

func TestOnRunStart_Disabled_ReturnsNil(t *testing.T) {
	handler := NewHandler(HandlerDeps{}, &PluginInput{Disabled: true})
	output, err := handler.OnRunStart(context.Background(), &model.Run{UUID: "r1"}, "ns1")
	require.NoError(t, err)
	assert.Nil(t, output)
}

func TestOnRunStart_NilInput_ReturnsNil(t *testing.T) {
	handler := NewHandler(HandlerDeps{}, nil)
	output, err := handler.OnRunStart(context.Background(), &model.Run{UUID: "r1"}, "ns1")
	require.NoError(t, err)
	assert.Nil(t, output)
}

func TestOnRunStart_Success(t *testing.T) {
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

	setGlobalMLflowConfig(t, server.URL)
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, nil) })

	handler := NewHandler(HandlerDeps{
		IsAuthorized: fakeIsAuthorizedOK,
	}, &PluginInput{ExperimentName: "Default"})

	run := &model.Run{
		UUID:        "kfp-run-1",
		DisplayName: "my-run",
	}
	output, err := handler.OnRunStart(context.Background(), run, "ns1")
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Contains(t, output.Entries, EntryExperimentID)
	assert.Equal(t, "exp-42", output.Entries[EntryExperimentID].Value.GetStringValue())
	assert.Contains(t, output.Entries, EntryRootRunID)
	assert.Equal(t, "mlflow-run-1", output.Entries[EntryRootRunID].Value.GetStringValue())

	// Verify RunStartEnv populated
	require.NotEmpty(t, handler.RunStartEnv)
	assert.Contains(t, handler.RunStartEnv, EnvTrackingURI)
	assert.Contains(t, handler.RunStartEnv, EnvWorkspace)
	assert.Equal(t, "ns1", handler.RunStartEnv[EnvWorkspace])
	assert.Contains(t, handler.RunStartEnv, EnvParentRunID)
	assert.Equal(t, "mlflow-run-1", handler.RunStartEnv[EnvParentRunID])
	assert.Contains(t, handler.RunStartEnv, EnvAuthType)
	assert.Equal(t, "kubernetes", handler.RunStartEnv[EnvAuthType])
}

func TestOnRunStart_MLflowFailure_ReturnsFailedOutput(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	setGlobalMLflowConfig(t, server.URL)
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, nil) })

	handler := NewHandler(HandlerDeps{
		IsAuthorized: fakeIsAuthorizedOK,
	}, &PluginInput{ExperimentName: "Default"})

	run := &model.Run{UUID: "kfp-run-2"}
	output, err := handler.OnRunStart(context.Background(), run, "ns1")
	require.Error(t, err)
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, output.State)
	assert.NotEmpty(t, output.StateMessage)
}

func TestOnRunStart_KFPBaseURL_AbsoluteRunURL(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	var tagValues = map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"exp-1","name":"Default"}}`))
		case "/api/2.0/mlflow/runs/create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"run-abs-1"}}}`))
		case "/api/2.0/mlflow/runs/set-tag":
			body, _ := io.ReadAll(r.Body)
			var tag struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			_ = json.Unmarshal(body, &tag)
			tagValues[tag.Key] = tag.Value
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	setGlobalMLflowConfigWithKFPBaseURL(t, server.URL, "https://pipelines.example.com")
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, nil) })

	handler := NewHandler(HandlerDeps{
		IsAuthorized: fakeIsAuthorizedOK,
	}, &PluginInput{ExperimentName: "Default"})

	run := &model.Run{UUID: "kfp-run-abs"}
	output, err := handler.OnRunStart(context.Background(), run, "ns1")
	require.NoError(t, err)
	require.NotNil(t, output)

	// Verify the tag has an absolute URL
	require.Contains(t, tagValues, TagKFPRunURL)
	assert.Equal(t, "https://pipelines.example.com/#/runs/details/kfp-run-abs", tagValues[TagKFPRunURL])
}

// ---- OnRunEnd / syncOnRunTerminal tests ----

func TestOnRunEnd_NilRun_ReturnsNil(t *testing.T) {
	handler := NewHandler(HandlerDeps{}, nil)
	err := handler.OnRunEnd(context.Background(), nil, "ns1")
	require.NoError(t, err)
}

func TestOnRunEnd_NoPluginOutput_ReturnsNil(t *testing.T) {
	handler := NewHandler(HandlerDeps{}, nil)
	run := &model.Run{UUID: "r1"}
	err := handler.OnRunEnd(context.Background(), run, "ns1")
	require.NoError(t, err)
}

func TestOnRunEnd_MissingRootRunID_PersistsFailed(t *testing.T) {
	store := &fakeRunStoreUpdater{}
	handler := NewHandler(HandlerDeps{RunStoreUpdater: store}, nil)

	// Build a run with plugin output that has no root_run_id
	outputJSON := `{"mlflow":{"entries":{"experiment_id":{"value":"42"}},"state":"PLUGIN_SUCCEEDED"}}`
	lt := model.LargeText(outputJSON)
	run := &model.Run{
		UUID:       "r-missing-root",
		RunDetails: model.RunDetails{PluginsOutputString: &lt},
	}

	err := handler.OnRunEnd(context.Background(), run, "ns1")
	require.NoError(t, err)

	// Verify store was called
	require.Len(t, store.calls, 1)
	assert.Equal(t, "r-missing-root", store.calls[0].RunID)
	assert.Contains(t, string(*store.calls[0].PluginsOutput), "missing parent root_run_id")
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

	setGlobalMLflowConfig(t, server.URL)

	store := &fakeRunStoreUpdater{}
	handler := NewHandler(HandlerDeps{RunStoreUpdater: store}, nil)

	outputJSON := fmt.Sprintf(`{"mlflow":{"entries":{"experiment_id":{"value":"exp-1"},"root_run_id":{"value":"mlflow-parent-1"}},"state":"PLUGIN_SUCCEEDED"}}`)
	lt := model.LargeText(outputJSON)
	run := &model.Run{
		UUID: "r-end-1",
		RunDetails: model.RunDetails{
			PluginsOutputString: &lt,
			State:               model.RuntimeState(apiv2beta1.RuntimeState_SUCCEEDED.String()),
			FinishedAtInSec:     1700000000,
		},
	}

	err := handler.OnRunEnd(context.Background(), run, "ns1")
	require.NoError(t, err)

	// Parent run should have been updated
	require.NotEmpty(t, updateCalls)
	assert.Contains(t, updateCalls[0], "mlflow-parent-1")
	assert.Contains(t, updateCalls[0], "FINISHED") // SUCCEEDED maps to FINISHED

	// Store should have been called
	require.Len(t, store.calls, 1)
	assert.Equal(t, "r-end-1", store.calls[0].RunID)
}

// ---- HandleRetry tests ----

func TestHandleRetry_NoPluginOutput_NoOp(t *testing.T) {
	store := &fakeRunStoreUpdater{}
	handler := NewHandler(HandlerDeps{RunStoreUpdater: store}, nil)
	run := &model.Run{UUID: "r-retry-noop"}

	handler.HandleRetry(context.Background(), run, "ns1")
	assert.Empty(t, store.calls)
}

func TestHandleRetry_MissingRootRunID_PersistsFailed(t *testing.T) {
	store := &fakeRunStoreUpdater{}
	handler := NewHandler(HandlerDeps{RunStoreUpdater: store}, nil)

	outputJSON := `{"mlflow":{"entries":{"experiment_id":{"value":"42"}},"state":"PLUGIN_SUCCEEDED"}}`
	lt := model.LargeText(outputJSON)
	run := &model.Run{
		UUID:       "r-retry-no-root",
		RunDetails: model.RunDetails{PluginsOutputString: &lt},
	}

	handler.HandleRetry(context.Background(), run, "ns1")

	require.Len(t, store.calls, 1)
	assert.Contains(t, string(*store.calls[0].PluginsOutput), "missing parent root_run_id")
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

	setGlobalMLflowConfig(t, server.URL)

	store := &fakeRunStoreUpdater{}
	handler := NewHandler(HandlerDeps{RunStoreUpdater: store}, nil)

	outputJSON := `{"mlflow":{"entries":{"experiment_id":{"value":"exp-1"},"root_run_id":{"value":"parent-1"}},"state":"PLUGIN_FAILED","stateMessage":"previous failure"}}`
	lt := model.LargeText(outputJSON)
	run := &model.Run{
		UUID:       "r-retry-ok",
		RunDetails: model.RunDetails{PluginsOutputString: &lt},
	}

	handler.HandleRetry(context.Background(), run, "ns1")

	// Parent reopened + nested-1 reopened = 2 update calls
	require.Len(t, updatePayloads, 2)
	assert.Contains(t, updatePayloads[0], "parent-1")
	assert.Contains(t, updatePayloads[0], "RUNNING")
	assert.Contains(t, updatePayloads[1], "nested-1")
	assert.Contains(t, updatePayloads[1], "RUNNING")

	// Store should have been called once
	require.Len(t, store.calls, 1)
	assert.Equal(t, "r-retry-ok", store.calls[0].RunID)
	assert.Contains(t, string(*store.calls[0].PluginsOutput), "SUCCEEDED")
}

// ---- BuildKFPRunURL tests ----

func TestBuildKFPRunURL(t *testing.T) {
	tests := []struct {
		name    string
		kfpBase string
		runID   string
		wantURL string
	}{
		{
			name:    "empty runID returns empty",
			kfpBase: "",
			runID:   "",
			wantURL: "",
		},
		{
			name:    "no base URL returns relative",
			kfpBase: "",
			runID:   "abc",
			wantURL: "/#/runs/details/abc",
		},
		{
			name:    "with base URL returns absolute",
			kfpBase: "https://pipelines.example.com",
			runID:   "abc",
			wantURL: "https://pipelines.example.com/#/runs/details/abc",
		},
		{
			name:    "trailing slash in base URL is trimmed",
			kfpBase: "https://pipelines.example.com/",
			runID:   "abc",
			wantURL: "https://pipelines.example.com/#/runs/details/abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildKFPRunURL(tt.kfpBase, tt.runID)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

// ---- SyncParentAndNestedRuns pagination test ----

func TestSyncParentAndNestedRuns_Pagination(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	clientSet := k8sfake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Name: "mlflow-creds", Namespace: "ns1"},
			Data:       map[string][]byte{"token": []byte("tok")},
		},
	)

	var searchCalls int
	var updateCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			updateCalls = append(updateCalls, string(body))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			searchCalls++
			if searchCalls == 1 {
				// First page returns one run + next_page_token
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-p1","status":"RUNNING"}}],"next_page_token":"page2"}`))
			} else {
				// Second page returns one run + no token
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-p2","status":"RUNNING"}}]}`))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	enabled := true
	requestCfg := &ResolvedConfig{
		Config:   &PluginConfig{Endpoint: server.URL, Timeout: "10s"},
		Settings: &PluginSettings{AuthType: AuthTypeBearer, CredentialSecretRef: &CredentialSecretRef{Name: "mlflow-creds", TokenKey: "token"}, WorkspacesEnabled: &enabled},
	}
	mlflowCtx, err := BuildRequestContext(context.Background(), clientSet, "ns1", requestCfg)
	require.NoError(t, err)

	endTime := int64(1700000000000)
	syncErrors := SyncParentAndNestedRuns(context.Background(), mlflowCtx, "parent-1", "exp-1", RunSyncModeTerminal, "FINISHED", &endTime)
	assert.Empty(t, syncErrors)

	// 2 search calls (pagination)
	assert.Equal(t, 2, searchCalls)
	// 1 parent update + 2 nested updates = 3 total
	assert.Len(t, updateCalls, 3)
	// Verify nested runs were updated
	found := strings.Join(updateCalls, " | ")
	assert.Contains(t, found, "nested-p1")
	assert.Contains(t, found, "nested-p2")
}
