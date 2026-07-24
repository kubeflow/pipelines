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
	"net/url"
	"strings"
	"sync"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	structpb "google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
)

// ---- Helpers ----

func setupSAToken(t *testing.T) func() {
	t.Helper()
	setupFakeKubernetesConfig(t, "test-sa-token")
	return func() {} // cleanup handled by t.Cleanup in setupFakeKubernetesConfig
}

func testPluginConfig(endpoint string) *ResolvedMLflowConfig {
	enabled := true
	return &ResolvedMLflowConfig{
		Config: &commonmlflow.MLflowPluginConfig{
			Endpoint: endpoint,
			Timeout:  "10s",
			Settings: &commonmlflow.MLflowPluginSettings{WorkspacesEnabled: &enabled},
		}}
}

func testResolvedConfig(endpoint string) *ResolvedMLflowConfig {
	cfg := testPluginConfig(endpoint)
	cfg.Config.Settings = ApplyMLflowSettingsDefaults(cfg.Config.Settings)
	resolvedCfg, err := newResolvedMLflowConfig(cfg.Config, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: "test-sa-token",
	})
	if err != nil {
		panic(err)
	}
	return resolvedCfg
}

func mustResolvedConfig(t *testing.T, cfg *commonmlflow.MLflowPluginConfig, credentials commonmlflow.MLflowCredentials) *ResolvedMLflowConfig {
	t.Helper()
	resolvedCfg, err := newResolvedMLflowConfig(cfg, credentials)
	require.NoError(t, err)
	return resolvedCfg
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

func addLegacyEndpointEntry(pluginOutput *apiv2beta1.PluginOutput, endpoint string) *apiv2beta1.PluginOutput {
	if pluginOutput == nil {
		return nil
	}
	if pluginOutput.Entries == nil {
		pluginOutput.Entries = make(map[string]*apiv2beta1.MetadataValue)
	}
	pluginOutput.Entries["endpoint"] = &apiv2beta1.MetadataValue{
		Value: structpb.NewStringValue(endpoint),
	}
	return pluginOutput
}

// ---- OnBeforeRunCreation tests ----

func TestOnBeforeRunCreation_NilConfig_ReturnsNil(t *testing.T) {
	handler := NewMLflowRunHandler()
	pluginInput := &MLflowPluginInput{Disabled: false}
	output, env, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), nil, pluginInput)
	require.NoError(t, err)
	assert.Nil(t, output)
	assert.Empty(t, env)
}

func TestOnBeforeRunCreation_Disabled_ReturnsNil(t *testing.T) {
	handler := NewMLflowRunHandler()

	pluginInput := &MLflowPluginInput{Disabled: true}
	output, env, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), testPluginConfig("http://localhost"), pluginInput)
	require.NoError(t, err)
	assert.Nil(t, output)
	assert.Empty(t, env)
}

func TestOnBeforeRunCreation_NilInput_ReturnsNil(t *testing.T) {
	handler := NewMLflowRunHandler()
	pluginInput := &MLflowPluginInput{Disabled: true}

	output, env, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("r1", "run-1"), testResolvedConfig("http://localhost"), pluginInput)
	require.NoError(t, err)
	assert.Nil(t, output)
	assert.Empty(t, env)
}

func TestOnBeforeRunCreation_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			assert.Equal(t, "Configured-Default", r.URL.Query().Get("experiment_name"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"exp-42","name":"Configured-Default"}}`))
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

	handler := NewMLflowRunHandler()

	run := testPendingRun("kfp-run-1", "my-run")
	cfg := testResolvedConfig(server.URL)
	cfg.Config.Settings.DefaultExperimentName = "Configured-Default"
	output, env, err := handler.OnBeforeRunCreation(context.Background(), run, cfg, &MLflowPluginInput{Disabled: false})
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Contains(t, output.Entries, EntryExperimentID)
	assert.Equal(t, "exp-42", output.Entries[EntryExperimentID].Value.GetStringValue())
	assert.Contains(t, output.Entries, apiserverPlugins.EntryRootRunID)
	assert.Equal(t, "mlflow-run-1", output.Entries[apiserverPlugins.EntryRootRunID].Value.GetStringValue())
	require.NoError(t, err)
	require.NotEmpty(t, env)
	require.NotNil(t, output)

	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Contains(t, output.Entries, EntryExperimentID)
	assert.Equal(t, "exp-42", output.Entries[EntryExperimentID].Value.GetStringValue())
	assert.Contains(t, output.Entries, apiserverPlugins.EntryRootRunID)
	assert.Equal(t, "mlflow-run-1", output.Entries[apiserverPlugins.EntryRootRunID].Value.GetStringValue())

	// Verify RunStartEnv contains single KFP_MLFLOW_CONFIG JSON env var
	require.NotEmpty(t, env)

	var rtCfg commonmlflow.MLflowRuntimeConfig
	require.NoError(t, json.Unmarshal([]byte(getEnvVarValue(t, env, commonmlflow.EnvMLflowConfig)), &rtCfg))
	assert.Contains(t, rtCfg.Endpoint, server.URL)
	assert.Equal(t, "ns1", rtCfg.Workspace)
	assert.Equal(t, "mlflow-run-1", rtCfg.ParentRunID)
	assert.Equal(t, "exp-42", rtCfg.ExperimentID)
	assert.Equal(t, "kubernetes", rtCfg.AuthType)
	assert.False(t, rtCfg.InjectUserEnvVars, "InjectUserEnvVars should default to false")
}

func TestOnBeforeRunCreation_BasicAuthInjectsCredentialEnvVars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "basic-user", username)
		assert.Equal(t, "basic-pass", password)
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

	settings := ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
		AuthType: commonmlflow.AuthTypeBasicAuth,
		CredentialSecretRef: &commonplugins.CredentialSecretRef{
			UsernameKey: "username",
			PasswordKey: "password",
		},
	})
	handler := NewMLflowRunHandler()
	pluginInput := &MLflowPluginInput{Disabled: false}
	run := testPendingRun("kfp-run-1", "my-run")

	output, env, err := handler.OnBeforeRunCreation(context.Background(), run, mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: server.URL,
		Timeout:  "10s",
		Settings: settings,
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeBasicAuth,
		Username: "basic-user",
		Password: "basic-pass",
	}), pluginInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, commonmlflow.EnvMLflowTrackingUsername, env[1].Name)
	require.NotNil(t, env[1].ValueFrom)
	require.NotNil(t, env[1].ValueFrom.SecretKeyRef)
	assert.Equal(t, commonmlflow.CredentialSecretName, env[1].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "username", env[1].ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, commonmlflow.EnvMLflowTrackingPassword, env[2].Name)
	require.NotNil(t, env[2].ValueFrom)
	require.NotNil(t, env[2].ValueFrom.SecretKeyRef)
	assert.Equal(t, commonmlflow.CredentialSecretName, env[2].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "password", env[2].ValueFrom.SecretKeyRef.Key)

	var rtCfg commonmlflow.MLflowRuntimeConfig
	require.NoError(t, json.Unmarshal([]byte(getEnvVarValue(t, env, commonmlflow.EnvMLflowConfig)), &rtCfg))
	assert.Equal(t, commonmlflow.AuthTypeBasicAuth, rtCfg.AuthType)
	require.NotNil(t, rtCfg.CredentialSecretRef)
	assert.Equal(t, "username", rtCfg.CredentialSecretRef.UsernameKey)
	assert.Equal(t, "password", rtCfg.CredentialSecretRef.PasswordKey)
	assert.Empty(t, rtCfg.Workspace)
}

func TestOnBeforeRunCreation_BearerInjectsTokenCredentialEnvVar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer custom-token", r.Header.Get("Authorization"))
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

	settings := ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
		AuthType: commonmlflow.AuthTypeBearer,
		CredentialSecretRef: &commonplugins.CredentialSecretRef{
			TokenKey: "token",
		},
	})
	handler := NewMLflowRunHandler()
	pluginInput := &MLflowPluginInput{Disabled: false}

	output, env, err := handler.OnBeforeRunCreation(context.Background(), testPendingRun("kfp-run-1", "my-run"), mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: server.URL,
		Timeout:  "10s",
		Settings: settings,
	}, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeBearer,
		BearerToken: "custom-token",
	}), pluginInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, commonmlflow.EnvMLflowTrackingToken, env[1].Name)
	require.NotNil(t, env[1].ValueFrom)
	require.NotNil(t, env[1].ValueFrom.SecretKeyRef)
	assert.Equal(t, commonmlflow.CredentialSecretName, env[1].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", env[1].ValueFrom.SecretKeyRef.Key)

	var rtCfg commonmlflow.MLflowRuntimeConfig
	require.NoError(t, json.Unmarshal([]byte(getEnvVarValue(t, env, commonmlflow.EnvMLflowConfig)), &rtCfg))
	assert.Equal(t, commonmlflow.AuthTypeBearer, rtCfg.AuthType)
	require.NotNil(t, rtCfg.CredentialSecretRef)
	assert.Equal(t, "token", rtCfg.CredentialSecretRef.TokenKey)
}

func getEnvVarValue(t *testing.T, envVars []corev1.EnvVar, name string) string {
	t.Helper()
	for _, envVar := range envVars {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	t.Fatalf("env var %q not found", name)
	return ""
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

	handler := NewMLflowRunHandler()
	pluginInput := &MLflowPluginInput{Disabled: false}

	run := testPendingRun("kfp-run-2", "run-2")
	output, env, err := handler.OnBeforeRunCreation(context.Background(), run, testResolvedConfig(server.URL), pluginInput)
	require.Error(t, err)
	assert.Empty(t, env)
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, output.State)
	assert.NotEmpty(t, output.StateMessage)
}

// ---- OnRunEnd / syncOnRunTerminal tests ----

func TestOnRunEnd_NilRun_ReturnsNil(t *testing.T) {
	handler := NewMLflowRunHandler()
	retryable, err := handler.OnRunEnd(context.Background(), nil, testResolvedConfig("http://localhost"))
	require.NoError(t, err)
	assert.False(t, retryable)
}

func TestOnRunEnd_NoPluginOutput_ReturnsNil(t *testing.T) {
	handler := NewMLflowRunHandler()
	run := testPersistedRun("r1")
	retryable, err := handler.OnRunEnd(context.Background(), run, testResolvedConfig("http://localhost"))
	require.NoError(t, err)
	assert.False(t, retryable)
}

func TestOnRunEnd_MissingRootRunID_SetsFailedState(t *testing.T) {
	handler := NewMLflowRunHandler()

	// Build a run with plugin output that has no root_run_id
	pluginOutput := SuccessfulPluginOutput("42", "Default", "", "")
	run := testPersistedRunWithPluginOutput("r-missing-root", pluginOutput)

	retryable, err := handler.OnRunEnd(context.Background(), run, testResolvedConfig("http://localhost"))
	require.NoError(t, err)
	assert.False(t, retryable, "missing parent run id is permanent and must not request a retry")

	// Verify the plugin output was updated in place
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "missing parent root_run_id")
}

func TestOnRunEnd_NilConfig_SetsFailedState(t *testing.T) {
	handler := NewMLflowRunHandler()

	pluginOutput := SuccessfulPluginOutput("42", "Default", "parent-1", "")
	run := testPersistedRunWithPluginOutput("r-nil-config", pluginOutput)

	retryable, err := handler.OnRunEnd(context.Background(), run, nil)
	require.NoError(t, err)
	assert.False(t, retryable, "unavailable config is permanent and must not request a retry")

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

	handler := NewMLflowRunHandler()

	pluginOutput := SuccessfulPluginOutput("exp-1", "Default", "mlflow-parent-1", "")
	run := testPersistedRunWithPluginOutput("r-end-1", pluginOutput)
	run.State = "SUCCEEDED"

	retryable, err := handler.OnRunEnd(context.Background(), run, testResolvedConfig(server.URL))
	require.NoError(t, err)
	assert.False(t, retryable)

	// Parent run should have been updated
	require.NotEmpty(t, updateCalls)
	assert.Contains(t, updateCalls[0], "mlflow-parent-1")
	assert.Contains(t, updateCalls[0], "FINISHED") // SUCCEEDED maps to FINISHED

	// Plugin output should be updated in place
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result.State)
}

func TestHandlerOnRunEnd_PermanentConfigFailureDoesNotRequestRetry(t *testing.T) {
	handler := NewMLflowRunHandler()

	// Create a config with missing settings (permanent configuration error)
	pluginOutput := SuccessfulPluginOutput("exp-1", "Default", "parent-1", "")
	run := testPersistedRunWithPluginOutput("r-config-failure", pluginOutput)
	run.State = "SUCCEEDED"

	cfg := &ResolvedMLflowConfig{
		Config: &commonmlflow.MLflowPluginConfig{
			Endpoint: "http://localhost",
			Timeout:  "10s",
			Settings: nil, // Missing settings is a permanent config error
		},
	}

	retryable, err := handler.OnRunEnd(context.Background(), run, cfg)
	require.NoError(t, err)
	assert.False(t, retryable, "permanent config failure must not request a retry")

	// Verify the plugin output was updated with failed state
	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "resolved MLflow settings are missing")
}

// ---- HandleRetry tests ----

func TestHandleRetry_NoPluginOutput_NoOp(t *testing.T) {
	handler := NewMLflowRunHandler()
	run := testPersistedRun("r-retry-noop")

	handler.HandleRetry(context.Background(), run, testResolvedConfig("http://localhost"))
	// No plugin output → nothing to do
	assert.Empty(t, run.PluginsOutput)
}

func TestHandleRetry_MissingRootRunID_SetsFailedState(t *testing.T) {
	handler := NewMLflowRunHandler()

	pluginOutput := SuccessfulPluginOutput("42", "Default", "", "")
	run := testPersistedRunWithPluginOutput("r-retry-no-root", pluginOutput)

	handler.HandleRetry(context.Background(), run, testResolvedConfig("http://localhost"))

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "missing parent root_run_id")
}

func TestHandleRetry_NilConfig_SetsFailedState(t *testing.T) {
	handler := NewMLflowRunHandler()

	pluginOutput := SuccessfulPluginOutput("42", "Default", "parent-1", "")
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

	handler := NewMLflowRunHandler()

	pluginOutput := FailedPluginOutput("exp-1", "Default", "parent-1", "", "previous failure")
	run := testPersistedRunWithPluginOutput("r-retry-ok", pluginOutput)

	handler.HandleRetry(context.Background(), run, testResolvedConfig(server.URL))

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

func TestPostRunSyncUsesResolvedConfigInsteadOfLegacyPluginOutputEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		pluginOutput     *apiv2beta1.PluginOutput
		runState         string
		wantUpdateStatus string
		invoke           func(*Handler, *apiserverPlugins.PersistedRun, *ResolvedMLflowConfig) (bool, error)
	}{
		{
			name:             "terminal sync ignores legacy endpoint entry",
			pluginOutput:     SuccessfulPluginOutput("exp-1", "Default", "parent-1", ""),
			runState:         "SUCCEEDED",
			wantUpdateStatus: "FINISHED",
			invoke: func(handler *Handler, run *apiserverPlugins.PersistedRun, config *ResolvedMLflowConfig) (bool, error) {
				_, err := handler.OnRunEnd(context.Background(), run, config)
				return err == nil, err
			},
		},
		{
			name:             "retry sync ignores legacy endpoint entry",
			pluginOutput:     FailedPluginOutput("exp-1", "Default", "parent-1", "", "previous failure"),
			wantUpdateStatus: "RUNNING",
			invoke: func(handler *Handler, run *apiserverPlugins.PersistedRun, config *ResolvedMLflowConfig) (bool, error) {
				err := handler.HandleRetry(context.Background(), run, config)
				return err == nil, nil
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cleanup := setupSAToken(t)
			defer cleanup()

			var mu sync.Mutex
			var staleCalls int
			staleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				staleCalls++
				mu.Unlock()
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"stale endpoint should not be used"}`))
			}))
			defer staleServer.Close()

			searchCalls := 0
			updatePayloads := []string{}
			freshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/2.0/mlflow/runs/update":
					body, _ := io.ReadAll(r.Body)
					mu.Lock()
					updatePayloads = append(updatePayloads, string(body))
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				case "/api/2.0/mlflow/runs/search":
					mu.Lock()
					searchCalls++
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"runs":[]}`))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			defer freshServer.Close()

			handler := NewMLflowRunHandler()
			pluginOutput := addLegacyEndpointEntry(testCase.pluginOutput, staleServer.URL)
			run := testPersistedRunWithPluginOutput("r-sync-fresh-config", pluginOutput)
			run.State = testCase.runState

			success, err := testCase.invoke(handler, run, testPluginConfig(freshServer.URL))
			require.NoError(t, err)
			assert.True(t, success)

			mu.Lock()
			defer mu.Unlock()
			require.Zero(t, staleCalls, "legacy plugins_output endpoint should be ignored")
			require.Len(t, updatePayloads, 1)
			assert.Equal(t, 1, searchCalls)
			assert.Contains(t, updatePayloads[0], "parent-1")
			assert.Contains(t, updatePayloads[0], testCase.wantUpdateStatus)

			result := run.PluginsOutput[PluginName]
			require.NotNil(t, result)
			assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, result.State)
		})
	}
}

// ---- BuildKFPRunURL tests ----

func TestBuildKFPRunURL(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		namespace    string
		kfpBaseURL   string
		pathTemplate string
		wantURL      string
	}{
		{
			name:    "empty runID returns empty",
			runID:   "",
			wantURL: "",
		},
		{
			name:    "no base URL returns empty",
			runID:   "abc",
			wantURL: "",
		},
		{
			name:       "default KFP UI hash route",
			runID:      "run-xyz",
			namespace:  "team-a",
			kfpBaseURL: "https://kfp.example.com",
			wantURL:    "https://kfp.example.com/#/runs/details/run-xyz",
		},
		{
			name:       "default hash route without namespace segment",
			runID:      "run-xyz",
			namespace:  "",
			kfpBaseURL: "https://kfp.example.com",
			wantURL:    "https://kfp.example.com/#/runs/details/run-xyz",
		},
		{
			name:         "path template with placeholders",
			runID:        "run-b",
			namespace:    "ns-a",
			kfpBaseURL:   "https://console.example.com",
			pathTemplate: "/demo/console/pipelines/{namespace}/runs/{run_id}",
			wantURL:      "https://console.example.com/demo/console/pipelines/ns-a/runs/run-b",
		},
		{
			name:         "path template without leading slash normalized",
			runID:        "r",
			namespace:    "n",
			kfpBaseURL:   "https://x.example",
			pathTemplate: "clusters/{namespace}/runs/{run_id}",
			wantURL:      "https://x.example/clusters/n/runs/r",
		},
		{
			name:         "template with namespace placeholder rejects empty ns",
			runID:        "run-xyz",
			namespace:    "",
			kfpBaseURL:   "https://kfp.example.com",
			pathTemplate: "/demo/console/pipelines/{namespace}/runs/{run_id}",
			wantURL:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiserverPlugins.BuildKFPRunURL(tt.runID, tt.namespace, tt.kfpBaseURL, tt.pathTemplate)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestBuildRunURL(t *testing.T) {
	mustParseURL := func(raw string) *url.URL {
		t.Helper()
		u, err := url.Parse(raw)
		require.NoError(t, err)
		return u
	}
	tests := []struct {
		name         string
		requestCtx   *commonmlflow.RequestContext
		experimentID string
		runID        string
		settings     *commonmlflow.MLflowPluginSettings
		wantURL      string
	}{
		{
			name:         "endpoint base with default hash route",
			requestCtx:   &commonmlflow.RequestContext{BaseURL: mustParseURL("https://tracking.example:5000")},
			experimentID: "5",
			runID:        "abc123",
			wantURL:      "https://tracking.example:5000/#/experiments/5/runs/abc123",
		},
		{
			name: "mlflowBaseURL overrides browser entry point",
			requestCtx: &commonmlflow.RequestContext{
				BaseURL: mustParseURL("http://mlflow.internal.svc.cluster.local:5000"),
			},
			experimentID: "9",
			runID:        "run-z",
			settings: &commonmlflow.MLflowPluginSettings{
				MLflowBaseURL: "https://mlflow.example.com",
			},
			wantURL: "https://mlflow.example.com/#/experiments/9/runs/run-z",
		},
		{
			name: "optional path prefix before fragment",
			requestCtx: &commonmlflow.RequestContext{
				BaseURL: mustParseURL("https://dashboard.example.com"),
			},
			experimentID: "1",
			runID:        "r1",
			settings:     &commonmlflow.MLflowPluginSettings{MLflowUIPathPrefix: "/mlflow"},
			wantURL:      "https://dashboard.example.com/mlflow/#/experiments/1/runs/r1",
		},
		{
			name: "workspace query in hash fragment",
			requestCtx: &commonmlflow.RequestContext{
				BaseURL:           mustParseURL("https://tracking.example"),
				WorkspacesEnabled: true,
				Workspace:         "mlflow-ws-1",
			},
			experimentID: "5",
			runID:        "abc123",
			wantURL:      "https://tracking.example/#/experiments/5/runs/abc123?workspace=mlflow-ws-1",
		},
		{
			name:         "mlflowBaseURL without requestCtx.BaseURL",
			requestCtx:   &commonmlflow.RequestContext{},
			experimentID: "2",
			runID:        "run-a",
			settings: &commonmlflow.MLflowPluginSettings{
				MLflowBaseURL: "https://ml.example",
			},
			wantURL: "https://ml.example/#/experiments/2/runs/run-a",
		},
		{
			name:         "no mount base yields empty",
			requestCtx:   &commonmlflow.RequestContext{},
			experimentID: "5",
			runID:        "x",
			wantURL:      "",
		},
		{
			name:         "missing experiment id yields empty",
			requestCtx:   &commonmlflow.RequestContext{BaseURL: mustParseURL("https://x")},
			experimentID: "",
			runID:        "y",
			wantURL:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRunURL(tt.requestCtx, tt.experimentID, tt.runID, tt.settings)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestShouldSyncNestedRun(t *testing.T) {
	t.Run("terminal mode syncs non-terminal statuses", func(t *testing.T) {
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "RUNNING"))
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "SCHEDULED"))
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "PENDING"))
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, ""))
		assert.False(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "FINISHED"))
		assert.False(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "FAILED"))
		assert.False(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, "KILLED"))
	})

	t.Run("retry mode syncs only failed and killed", func(t *testing.T) {
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeRetry, "FAILED"))
		assert.True(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeRetry, "KILLED"))
		assert.False(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeRetry, "RUNNING"))
		assert.False(t, shouldSyncNestedRun(apiserverPlugins.RunSyncModeRetry, "FINISHED"))
	})
}

// ---- ModelToPersistedRun tests ----

func TestModelToPersistedRun_NilModel(t *testing.T) {
	_, err := apiserverPlugins.ModelToPersistedRun(nil, "ns1")
	require.Error(t, err)
}

func TestModelToPersistedRun_BasicFields(t *testing.T) {
	pluginsJSON := `{"mlflow":{"entries":{"root_run_id":{"value":"parent-1"}},"state":"PLUGIN_SUCCEEDED"}}`
	lt := model.LargeText(pluginsJSON)
	m := &model.Run{
		UUID: "run-123",
	}
	m.RunDetails.State = "SUCCEEDED"          //nolint:staticcheck // QF1008
	m.RunDetails.FinishedAtInSec = 1700000000 //nolint:staticcheck // QF1008
	m.RunDetails.PluginsOutputString = &lt    //nolint:staticcheck // QF1008

	pr, err := apiserverPlugins.ModelToPersistedRun(m, "ns1")
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.Equal(t, "run-123", pr.RunID)
	assert.Equal(t, "ns1", pr.Namespace)
	assert.Equal(t, "SUCCEEDED", pr.State)
	require.NotNil(t, pr.FinishedAt)
	assert.Equal(t, int64(1700000000), pr.FinishedAt.Unix())
	require.NotNil(t, pr.PluginsOutput[PluginName])
	assert.Equal(t, "parent-1", apiserverPlugins.GetParentRunID(pr.PluginsOutput[PluginName]))
}

// ---- SerializePluginsOutput / DeserializePluginsOutput tests ----

func TestSerializeDeserializePluginsOutput_RoundTrip(t *testing.T) {
	original := map[string]*apiv2beta1.PluginOutput{
		"mlflow":       SuccessfulPluginOutput("exp-1", "Default", "parent-1", ""),
		"other_plugin": {State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED},
	}
	lt, err := apiserverPlugins.SerializePluginsOutput(original)
	require.NoError(t, err)
	require.NotNil(t, lt)
	assert.Contains(t, string(*lt), "mlflow")
	assert.Contains(t, string(*lt), "other_plugin")

	result, err := apiserverPlugins.DeserializePluginsOutput(lt)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotNil(t, result["mlflow"])
	assert.NotNil(t, result["other_plugin"])
	assert.Equal(t, "parent-1", apiserverPlugins.GetParentRunID(result["mlflow"]))
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
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: server.URL,
		Timeout:  "10s",
		TLS: &commonplugins.TLSConfig{
			InsecureSkipVerify: true,
		},
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{WorkspacesEnabled: &enabled}),
	}, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: "bearer-secret",
	})
	mlflowCtx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.NoError(t, err)

	endTime := int64(1700000000000)
	syncErrors := SyncParentAndNestedRuns(context.Background(), mlflowCtx, "parent-1", "exp-1", apiserverPlugins.RunSyncModeTerminal, "FINISHED", &endTime)
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
