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
	"path/filepath"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// setupFakeKubernetesConfig writes a temp kubeconfig with the given bearer token
// and sets the KUBECONFIG env var so util.GetKubernetesConfig() picks it up.
func setupFakeKubernetesConfig(t *testing.T, token string) {
	t.Helper()
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: %s
`, token)
	p := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(p, []byte(kubeconfig), 0600))
	t.Setenv("KUBECONFIG", p)
}

func TestResolveMLflowPluginInput(t *testing.T) {
	tests := []struct {
		name      string
		input     *string
		want      *MLflowPluginInput
		wantError bool
	}{
		{
			name:  "nil input defaults",
			input: nil,
			want:  &MLflowPluginInput{ExperimentName: DefaultExperimentName},
		},
		{
			name:  "empty string defaults",
			input: strPtr(""),
			want:  &MLflowPluginInput{ExperimentName: DefaultExperimentName},
		},
		{
			name:  "missing mlflow block defaults",
			input: strPtr(`{"other":{"x":"y"}}`),
			want:  &MLflowPluginInput{ExperimentName: DefaultExperimentName},
		},
		{
			name:  "missing experiment_name defaults",
			input: strPtr(`{"mlflow":{"disabled":false}}`),
			want:  &MLflowPluginInput{ExperimentName: DefaultExperimentName},
		},
		{
			name:  "valid experiment_name is used",
			input: strPtr(`{"mlflow":{"experiment_name":"exp-1"}}`),
			want:  &MLflowPluginInput{ExperimentName: "exp-1"},
		},
		{
			name:  "experiment_id takes precedence and is preserved",
			input: strPtr(`{"mlflow":{"experiment_name":"exp-1","experiment_id":"42"}}`),
			want:  &MLflowPluginInput{ExperimentName: "exp-1", ExperimentID: "42"},
		},
		{
			name:  "disabled true is accepted",
			input: strPtr(`{"mlflow":{"disabled":true}}`),
			want:  &MLflowPluginInput{Disabled: true},
		},
		{
			name:      "invalid plugins_input json errors",
			input:     strPtr(`{`),
			wantError: true,
		},
		{
			name:      "non-string experiment_name errors",
			input:     strPtr(`{"mlflow":{"experiment_name":123}}`),
			wantError: true,
		},
		{
			name:      "non-string experiment_id errors",
			input:     strPtr(`{"mlflow":{"experiment_id":123}}`),
			wantError: true,
		},
		{
			name:      "unknown field is rejected",
			input:     strPtr(`{"mlflow":{"experiment_name":"exp-1","unknown":"x"}}`),
			wantError: true,
		},
		{
			name:      "mlflow block must be an object",
			input:     strPtr(`{"mlflow":"not-an-object"}`),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveMLflowPluginInput(tt.input)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSelectMLflowExperiment(t *testing.T) {
	tests := []struct {
		name               string
		input              *MLflowPluginInput
		settings           *commonmlflow.MLflowPluginSettings
		wantExperimentID   string
		wantExperimentName string
	}{
		{
			name:               "nil input and nil settings defaults to KFP-Default",
			input:              nil,
			settings:           nil,
			wantExperimentID:   "",
			wantExperimentName: DefaultExperimentName,
		},
		{
			name:               "experiment_id takes precedence",
			input:              &MLflowPluginInput{ExperimentID: "42", ExperimentName: "ignored"},
			settings:           nil,
			wantExperimentID:   "42",
			wantExperimentName: "",
		},
		{
			name:               "uses experiment_name when id absent",
			input:              &MLflowPluginInput{ExperimentName: "exp-a"},
			settings:           nil,
			wantExperimentID:   "",
			wantExperimentName: "exp-a",
		},
		{
			name:               "empty input falls back to admin-configured default",
			input:              &MLflowPluginInput{},
			settings:           &commonmlflow.MLflowPluginSettings{DefaultExperimentName: "Team-Pipelines"},
			wantExperimentID:   "",
			wantExperimentName: "Team-Pipelines",
		},
		{
			name:               "nil input falls back to admin-configured default",
			input:              nil,
			settings:           &commonmlflow.MLflowPluginSettings{DefaultExperimentName: "Org-Default"},
			wantExperimentID:   "",
			wantExperimentName: "Org-Default",
		},
		{
			name:               "user experiment_name overrides admin default",
			input:              &MLflowPluginInput{ExperimentName: "user-exp"},
			settings:           &commonmlflow.MLflowPluginSettings{DefaultExperimentName: "Admin-Default"},
			wantExperimentID:   "",
			wantExperimentName: "user-exp",
		},
		{
			name:               "empty input and empty admin default falls back to KFP-Default",
			input:              &MLflowPluginInput{},
			settings:           &commonmlflow.MLflowPluginSettings{},
			wantExperimentID:   "",
			wantExperimentName: DefaultExperimentName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotName := SelectMLflowExperiment(tt.input, tt.settings)
			assert.Equal(t, tt.wantExperimentID, gotID)
			assert.Equal(t, tt.wantExperimentName, gotName)
		})
	}
}

func TestGetGlobalMLflowConfig(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"workspacesEnabled": false,
			},
		},
	})

	cfg, ok, err := GetGlobalMLflowConfig()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "https://mlflow.example.com", cfg.Endpoint)
	assert.Equal(t, "10s", cfg.Timeout)
	require.NotNil(t, cfg.Settings)
}

func TestBuildMLflowRequestContextKubernetesAuth(t *testing.T) {
	setupFakeKubernetesConfig(t, "sa-token-value")

	workspacesEnabled := true
	mlflowPluginCfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "12s",
		TLS: &commonplugins.TLSConfig{
			InsecureSkipVerify: false,
		},
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &workspacesEnabled,
		},
	}

	mlflowCtx, err := BuildMLflowRunRequestContext("ns1", mlflowPluginCfg)
	require.NoError(t, err)
	require.NotNil(t, mlflowCtx)
	assert.Equal(t, "https://mlflow.example.com", mlflowCtx.BaseURL.String())
	assert.True(t, mlflowCtx.WorkspacesEnabled)
	assert.NotNil(t, mlflowCtx.Client)
}

func TestBuildMLflowRunRequestContext_InvalidEndpoint(t *testing.T) {
	enabled := true
	cfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: "not-a-url",
		Timeout:  "10s",
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &enabled,
		},
	}

	ctx, err := BuildMLflowRunRequestContext("ns1", cfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "invalid plugins.mlflow endpoint")
}

func TestBuildMLflowRunRequestContext_ZeroTimeout(t *testing.T) {
	setupFakeKubernetesConfig(t, "sa-token-value")

	enabled := true
	cfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "0s",
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &enabled,
		},
	}

	ctx, err := BuildMLflowRunRequestContext("ns1", cfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "timeout must be > 0")
}

func TestEnsureExperimentExists(t *testing.T) {
	t.Run("returns existing experiment from get-by-name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
			assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
			assert.Equal(t, "/api/2.0/mlflow/experiments/get-by-name", r.URL.Path)
			assert.Equal(t, "my-exp", r.URL.Query().Get("experiment_name"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"42","name":"my-exp"}}`))
		}))
		defer server.Close()

		mlflowCtx := newTestMLflowRequestContext(t, server.URL)
		defaultDesc := DefaultExperimentDescription
		exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "my-exp", &defaultDesc)
		require.NoError(t, err)
		require.NotNil(t, exp)
		assert.Equal(t, "42", exp.ID)
		assert.Equal(t, "my-exp", exp.Name)
	})

	t.Run("creates experiment when get-by-name returns not found", func(t *testing.T) {
		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
			assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
			switch r.URL.Path {
			case "/api/2.0/mlflow/experiments/get-by-name":
				callCount++
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
			case "/api/2.0/mlflow/experiments/create":
				bodyBytes, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(bodyBytes), `"name":"my-exp"`)
				assert.Contains(t, string(bodyBytes), `"description":"Created by Kubeflow Pipelines"`)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"experiment_id":"99"}`))
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer server.Close()

		defaultDesc := DefaultExperimentDescription
		mlflowCtx := newTestMLflowRequestContext(t, server.URL)
		exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "my-exp", &defaultDesc)
		require.NoError(t, err)
		require.NotNil(t, exp)
		assert.Equal(t, "99", exp.ID)
		assert.Equal(t, "my-exp", exp.Name)
		assert.Equal(t, 1, callCount)
	})
}

func TestBuildKFPTags(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:             "kfp-run-1",
		Namespace:         "ns-1",
		PipelineID:        "pipeline-1",
		PipelineVersionID: "pipeline-version-1",
	}
	tags := BuildKFPTags(run, "", "")
	require.Len(t, tags, 4)
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPRunID, Value: "kfp-run-1"})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPRunURL, Value: ""})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: "pipeline-1"})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: "pipeline-version-1"})
}

func TestBuildKFPTags_WithBaseURL(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:     "run-1",
		Namespace: "ns-1",
	}
	tags := BuildKFPTags(run, "https://kfp.example.com", "")
	require.Len(t, tags, 2)
	assert.Contains(t, tags, commonmlflow.Tag{
		Key:   TagKFPRunURL,
		Value: "https://kfp.example.com/#/runs/details/run-1",
	})
}

func TestBuildKFPTags_WithCustomPathTemplate(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:     "run-b",
		Namespace: "proj-1",
	}
	tmpl := "/demo/console/pipelines/{namespace}/runs/{run_id}"
	tags := BuildKFPTags(run, "https://console.example.com", tmpl)
	require.Len(t, tags, 2)
	assert.Contains(t, tags, commonmlflow.Tag{
		Key:   TagKFPRunURL,
		Value: "https://console.example.com/demo/console/pipelines/proj-1/runs/run-b",
	})
}

func TestBuildKFPTags_NilRun(t *testing.T) {
	assert.Nil(t, BuildKFPTags(nil, "", ""))
}

func TestCreateRunWithKFPTags(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
		assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/create":
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &receivedBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-parent-run-1"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)

	run := &apiserverPlugins.PendingRun{
		RunID:             "kfp-run-1",
		PipelineID:        "pipeline-1",
		PipelineVersionID: "pipeline-version-1",
	}
	tags := BuildKFPTags(run, "", "")
	mlflowRunID, err := mlflowCtx.Client.CreateRun(context.Background(), "exp-1", "sample-run", tags)
	require.NoError(t, err)
	assert.Equal(t, "mlflow-parent-run-1", mlflowRunID)

	// Verify tags were included in the CreateRun request body.
	rawTags, ok := receivedBody["tags"].([]interface{})
	require.True(t, ok, "tags should be present in CreateRun body")
	assert.Len(t, rawTags, 4)
}

func TestBuildPluginOutput(t *testing.T) {
	output := SuccessfulPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example/runs/run-1", "https://mlflow.example")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	require.Contains(t, output.Entries, "run_url")
	require.NotNil(t, output.Entries["run_url"].RenderType)
	assert.Equal(t, apiv2beta1.MetadataValue_URL, *output.Entries["run_url"].RenderType)
}

func TestSetPendingRunPluginOutput(t *testing.T) {
	// Start with a PendingRun that already has another plugin's output.
	existing := `{"other":{"state":"PLUGIN_SUCCEEDED"}}`
	run := &apiserverPlugins.PendingRun{
		RunID:         "run-1",
		PluginsOutput: &existing,
	}
	mlflowOutput := SuccessfulPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example/runs/run-1", "https://mlflow.example")
	err := SetPendingRunPluginOutput(run, "mlflow", mlflowOutput)
	require.NoError(t, err)
	require.NotNil(t, run.PluginsOutput)

	var envelope pluginsOutputEnvelope
	require.NoError(t, json.Unmarshal([]byte(*run.PluginsOutput), &envelope))
	assert.NotNil(t, envelope.others["other"], "pre-existing 'other' entry should be preserved")
	assert.NotEmpty(t, envelope.MLflow, "mlflow entry should be set")
	var parsed apiv2beta1.PluginOutput
	require.NoError(t, protojson.Unmarshal(envelope.MLflow, &parsed))
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, parsed.State)
	assert.Contains(t, parsed.Entries, "experiment_id")
}

func TestToMLflowTerminalStatus(t *testing.T) {
	assert.Equal(t, "FINISHED", ToMLflowTerminalStatus("SUCCEEDED"))
	assert.Equal(t, "KILLED", ToMLflowTerminalStatus("CANCELED"))
	assert.Equal(t, "KILLED", ToMLflowTerminalStatus("CANCELING"))
	assert.Equal(t, "FAILED", ToMLflowTerminalStatus("FAILED"))
	assert.Equal(t, "FAILED", ToMLflowTerminalStatus("UNKNOWN"))
}

func strPtr(s string) *string {
	return &s
}

// fakeKubeClientProvider implements KubeClientProvider for tests.
type fakeKubeClientProvider struct {
	clientSet kubernetes.Interface
}

func (f *fakeKubeClientProvider) GetClientSet() kubernetes.Interface {
	return f.clientSet
}

func TestResolveMLflowCredentials_EmptySAToken(t *testing.T) {
	setupFakeKubernetesConfig(t, "")

	_, err := commonmlflow.ResolveMLflowCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bearer token is empty")
}

func TestInjectMLflowRuntimeEnv(t *testing.T) {
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{
					Name: "system-dag-driver",
					Metadata: workflowapi.Metadata{
						Annotations: map[string]string{
							util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleDriver),
						},
					},
					Container: &corev1.Container{Args: []string{"--type", "DAG"}},
				},
				{
					Name: "system-container-impl",
					Metadata: workflowapi.Metadata{
						Annotations: map[string]string{
							util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleLauncher),
						},
					},
					Container: &corev1.Container{},
					InitContainers: []workflowapi.UserContainer{
						{Container: corev1.Container{Name: "kfp-launcher", Args: []string{"--copy", "/kfp-launcher/launch"}}},
					},
				},
			},
		},
	})

	env := map[string]string{
		commonmlflow.EnvMLflowConfig: `{"endpoint":"https://mlflow.example.com","parentRunId":"abc"}`,
	}
	err := apiserverPlugins.InjectPluginRuntimeEnv(workflow, env)
	require.NoError(t, err)

	expectedEnv := corev1.EnvVar{Name: commonmlflow.EnvMLflowConfig, Value: env[commonmlflow.EnvMLflowConfig]}

	// Driver container gets the env var.
	assert.Contains(t, workflow.Spec.Templates[0].Container.Env, expectedEnv)

	// Launcher main container (template with --copy init container) gets the env var.
	assert.Contains(t, workflow.Spec.Templates[1].Container.Env, expectedEnv)

	// Launcher init container does NOT get the env var (it only copies the binary).
	assert.NotContains(t, workflow.Spec.Templates[1].InitContainers[0].Env, expectedEnv)
}

func TestInjectMLflowRuntimeEnv_NilSpec(t *testing.T) {
	err := apiserverPlugins.InjectPluginRuntimeEnv(nil, map[string]string{"key": "val"})
	require.NoError(t, err, "nil spec should be a no-op")
}

func TestInjectMLflowRuntimeEnv_EmptyEnv(t *testing.T) {
	workflow := util.NewWorkflow(&workflowapi.Workflow{})
	err := apiserverPlugins.InjectPluginRuntimeEnv(workflow, map[string]string{})
	require.NoError(t, err, "empty env should be a no-op")
}

// newTestMLflowRequestContext creates a *commonmlflow.RequestContext backed by a service account
// token, pointing at serverURL with workspaces enabled.
func newTestMLflowRequestContext(t *testing.T, serverURL string) *commonmlflow.RequestContext {
	t.Helper()
	setupFakeKubernetesConfig(t, "bearer-secret")

	enabled := true
	mlflowPluginCfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: serverURL,
		Timeout:  "10s",
		TLS: &commonplugins.TLSConfig{
			InsecureSkipVerify: false,
		},
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &enabled,
		},
	}
	ctx, err := BuildMLflowRunRequestContext("ns1", mlflowPluginCfg)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	return ctx
}
