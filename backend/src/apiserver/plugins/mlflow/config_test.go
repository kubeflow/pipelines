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
	"os"
	"path/filepath"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
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
			want:  &MLflowPluginInput{},
		},
		{
			name:  "empty string defaults",
			input: strPtr(""),
			want:  &MLflowPluginInput{},
		},
		{
			name:  "missing mlflow block defaults",
			input: strPtr(`{"other":{"x":"y"}}`),
			want:  &MLflowPluginInput{},
		},
		{
			name:  "missing experiment_name defaults",
			input: strPtr(`{"mlflow":{"disabled":false}}`),
			want:  &MLflowPluginInput{},
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

func TestGetGlobalMLflowConfig_NotSet(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", nil)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	cfg, ok, err := GetGlobalMLflowConfig()
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, commonmlflow.MLflowPluginConfig{}, cfg)
}

func TestApplyMLflowSettingsDefaults_NilInput(t *testing.T) {
	settings := ApplyMLflowSettingsDefaults(nil)
	require.NotNil(t, settings)
	assert.Equal(t, commonmlflow.AuthTypeKubernetes, settings.AuthType)
	require.NotNil(t, settings.WorkspacesEnabled)
	assert.True(t, *settings.WorkspacesEnabled)
	assert.Equal(t, DefaultExperimentName, settings.DefaultExperimentName)
	require.NotNil(t, settings.ExperimentDescription)
	assert.Equal(t, DefaultExperimentDescription, *settings.ExperimentDescription)
}

func TestApplyMLflowSettingsDefaults_EmptySettings(t *testing.T) {
	settings := ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{})
	require.NotNil(t, settings)
	assert.Equal(t, commonmlflow.AuthTypeKubernetes, settings.AuthType)
	require.NotNil(t, settings.WorkspacesEnabled)
	assert.True(t, *settings.WorkspacesEnabled)
	assert.Equal(t, DefaultExperimentName, settings.DefaultExperimentName)
	require.NotNil(t, settings.ExperimentDescription)
	assert.Equal(t, DefaultExperimentDescription, *settings.ExperimentDescription)
}

func TestApplyMLflowSettingsDefaults_PreservesExistingValues(t *testing.T) {
	customDesc := "Custom Description"
	wsEnabled := false
	settings := ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
		AuthType:              commonmlflow.AuthTypeBearer,
		WorkspacesEnabled:     &wsEnabled,
		DefaultExperimentName: "CustomDefault",
		ExperimentDescription: &customDesc,
	})
	require.NotNil(t, settings)
	assert.Equal(t, commonmlflow.AuthTypeBearer, settings.AuthType)
	require.NotNil(t, settings.WorkspacesEnabled)
	assert.False(t, *settings.WorkspacesEnabled)
	assert.Equal(t, "CustomDefault", settings.DefaultExperimentName)
	require.NotNil(t, settings.ExperimentDescription)
	assert.Equal(t, "Custom Description", *settings.ExperimentDescription)
}

func TestResolveMLflowPluginInput_TrailingJSON(t *testing.T) {
	input := strPtr(`{"mlflow":{"experiment_name":"test"}} {"extra":"data"}`)
	_, err := ResolveMLflowPluginInput(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugins_input must be a valid JSON object")
}

func TestResolveMLflowPluginInput_CaseSensitiveKey(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		resolvedInput *MLflowPluginInput
	}{
		{"lowercase", `{"mlflow":{"experiment_name":"test"}}`, &MLflowPluginInput{ExperimentName: "test"}},
		{"uppercase", `{"MLFLOW":{"experiment_name":"test"}}`, &MLflowPluginInput{}},
		{"mixedcase", `{"MLflow":{"experiment_name":"test"}}`, &MLflowPluginInput{}},
		{"MiXeDcAsE", `{"mLfLoW":{"experiment_name":"test"}}`, &MLflowPluginInput{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveMLflowPluginInput(new(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.resolvedInput, got)
		})
	}
}

func TestGetLauncherNamespaceMLflowConfig_InvalidJSON(t *testing.T) {
	cfg, err := GetLauncherNamespaceMLflowConfig("ns1", `{invalid json`)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse MLflow config")
}

func TestGetLauncherNamespaceMLflowConfig_EmptyString(t *testing.T) {
	cfg, err := GetLauncherNamespaceMLflowConfig("ns1", `{}`)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Nil(t, cfg.Settings)
}

func TestMergePluginConfigAndSettingsDefaults(t *testing.T) {
	globalDesc := "Global desc"
	wsDisabled := false
	global := commonmlflow.MLflowPluginConfig{
		Endpoint: "https://global-mlflow.example.com",
		Timeout:  "30s",
		Settings: &commonmlflow.MLflowPluginSettings{
			ExperimentDescription: &globalDesc,
			KFPBaseURL:            "https://global-kfp.example.com",
			KFPRunURLPathTemplate: "/global/{run_id}",
			MLflowBaseURL:         "https://global-mlflow-ui.example.com",
			MLflowUIPathPrefix:    "/global-mlflow",
		},
	}
	namespace := &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://ns-mlflow.example.com",
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled:     &wsDisabled,
			KFPRunURLPathTemplate: "/ns/{namespace}/runs/{run_id}",
			MLflowBaseURL:         "https://ns-mlflow-ui.example.com",
			MLflowUIPathPrefix:    "/ns-mlflow",
		},
	}

	merged := commonmlflow.MergePluginConfig(global, namespace)
	assert.Equal(t, "https://ns-mlflow.example.com", merged.Endpoint)
	assert.Equal(t, "30s", merged.Timeout)

	settings := ApplyMLflowSettingsDefaults(merged.Settings)
	require.NotNil(t, settings)
	require.NotNil(t, settings.WorkspacesEnabled)
	assert.False(t, *settings.WorkspacesEnabled)
	require.NotNil(t, settings.ExperimentDescription)
	assert.Equal(t, "Global desc", *settings.ExperimentDescription)
	assert.Equal(t, "https://global-kfp.example.com", settings.KFPBaseURL)
	assert.Equal(t, "/ns/{namespace}/runs/{run_id}", settings.KFPRunURLPathTemplate)
	assert.Equal(t, "https://ns-mlflow-ui.example.com", settings.MLflowBaseURL)
	assert.Equal(t, "/ns-mlflow", settings.MLflowUIPathPrefix)
}

func TestApplySettingsDefaults_NonKubernetesAuthDefaults(t *testing.T) {
	settings := ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
		AuthType: commonmlflow.AuthTypeNone,
	})
	require.NotNil(t, settings.WorkspacesEnabled)
	assert.False(t, *settings.WorkspacesEnabled)
	assert.Equal(t, commonmlflow.AuthTypeNone, settings.AuthType)
}

func TestBuildMLflowRequestContextKubernetesAuth(t *testing.T) {
	setupFakeKubernetesConfig(t, "sa-token-value")

	workspacesEnabled := true
	mlflowPluginCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "12s",
		TLS: &commonplugins.TLSConfig{
			InsecureSkipVerify: false,
		},
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &workspacesEnabled,
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: "sa-token-value",
	})

	mlflowCtx, err := BuildMLflowRunRequestContext("ns1", mlflowPluginCfg)
	require.NoError(t, err)
	require.NotNil(t, mlflowCtx)
	assert.Equal(t, "https://mlflow.example.com", mlflowCtx.BaseURL.String())
	assert.True(t, mlflowCtx.WorkspacesEnabled)
	assert.NotNil(t, mlflowCtx.Client)
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

// TestResolveMLflowRequestConfig_NamespaceOnlyIsDisabled verifies that MLflow
// is disabled when no global plugins.mlflow config exists, even if the namespace
// has a valid kfp-launcher ConfigMap.  Global config is required to opt in.
func TestResolveMLflowRequestConfig_NamespaceOnlyIsDisabled(t *testing.T) {
	// Ensure no global config.
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", nil)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	// Verify global is absent.
	_, hasGlobal, err := GetGlobalMLflowConfig()
	require.NoError(t, err)
	require.False(t, hasGlobal, "global config should be absent for this test")

	// Create a fake namespace ConfigMap with MLflow config.
	clientSet := k8sfake.NewClientset(
		&corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      LauncherConfigMapName,
				Namespace: "ns-only",
			},
			Data: map[string]string{
				LauncherConfigKey: `{
					"settings": {
						"defaultExperimentName": "NamespaceDefault",
						"injectUserEnvVars": true
					}
				}`,
			},
		},
	)
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), "", "ns-only")
	require.NoError(t, err)
	require.Nil(t, resolved, "namespace-only config should NOT enable MLflow without global opt-in")
}

// TestResolveMLflowRequestConfig_NeitherGlobalNorNamespace verifies that MLflow
// is disabled (nil return) when both global and namespace configs are absent.
func TestResolveMLflowRequestConfig_NeitherGlobalNorNamespace(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", nil)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	clientSet := k8sfake.NewClientset() // no ConfigMap
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), "", "empty-ns")
	require.NoError(t, err)
	assert.Nil(t, resolved, "MLflow should be disabled when neither global nor namespace config exists")
}

func TestResolveMLflowRequestConfig_StandaloneIgnoresNamespaceLayers(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType":              "none",
				"defaultExperimentName": "GlobalDefault",
			},
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"endpoint": "https://server-side-override.example.com",
					"settings": map[string]interface{}{
						"defaultExperimentName": "ServerDefault",
					},
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, nil)
	})

	clientSet := k8sfake.NewClientset(
		&corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      LauncherConfigMapName,
				Namespace: "ns1",
			},
			Data: map[string]string{
				LauncherConfigKey: `{
					"settings": {
						"defaultExperimentName": "StandaloneIgnored"
					}
				}`,
			},
		},
	)
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), "", "ns1")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Config)
	require.NotNil(t, resolved.Config.Settings)
	assert.Equal(t, "https://global-mlflow.example.com", resolved.Config.Endpoint)
	assert.Equal(t, "GlobalDefault", resolved.Config.Settings.DefaultExperimentName)
	assert.Equal(t, commonmlflow.AuthTypeNone, resolved.Credentials.AuthType)
}

func TestResolveMLflowRequestConfig_MultiUserAppliesServerAndLauncherOverrides(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType":          "none",
				"workspacesEnabled": true,
				"kfpBaseURL":        "https://global-kfp.example.com",
			},
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"endpoint": "https://server-side-override.example.com",
					"timeout":  "25s",
					"settings": map[string]interface{}{
						"workspacesEnabled": false,
						"kfpBaseURL":        "https://server-kfp.example.com",
					},
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, nil)
	})

	launcherCfgOverride := `{
		"settings": {
			"experimentDescription": "LauncherDescription",
			"defaultExperimentName": "LauncherDefault",
			"injectUserEnvVars": true
		}
	}`

	kubeClients := &fakeKubeClientProvider{clientSet: k8sfake.NewClientset()}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), launcherCfgOverride, "ns1")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Config)
	require.NotNil(t, resolved.Config.Settings)
	assert.Equal(t, "https://server-side-override.example.com", resolved.Config.Endpoint)
	assert.Equal(t, "25s", resolved.Config.Timeout)
	require.NotNil(t, resolved.Config.Settings.WorkspacesEnabled)
	assert.False(t, *resolved.Config.Settings.WorkspacesEnabled)
	assert.Equal(t, "https://server-kfp.example.com", resolved.Config.Settings.KFPBaseURL)
	require.NotNil(t, resolved.Config.Settings.ExperimentDescription)
	assert.Equal(t, "LauncherDescription", *resolved.Config.Settings.ExperimentDescription)
	assert.Equal(t, "LauncherDefault", resolved.Config.Settings.DefaultExperimentName)
	require.NotNil(t, resolved.Config.Settings.InjectUserEnvVars)
	assert.True(t, *resolved.Config.Settings.InjectUserEnvVars)
	assert.Equal(t, commonmlflow.AuthTypeNone, resolved.Credentials.AuthType)
}

func TestResolveMLflowRequestConfig_GlobalCredentialSecretRefDefaultUsed(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType": "bearer",
				"credentialSecretRef": map[string]interface{}{
					"tokenKey": "global-token",
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	clientSet := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      commonmlflow.CredentialSecretName,
			Namespace: "ns-defaults",
		},
		Data: map[string][]byte{
			"global-token": []byte("global-secret"),
		},
	})
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), "", "ns-defaults")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Config.Settings)
	require.NotNil(t, resolved.Config.Settings.CredentialSecretRef)
	assert.Equal(t, "global-token", resolved.Config.Settings.CredentialSecretRef.TokenKey)
	assert.Equal(t, commonmlflow.AuthTypeBearer, resolved.Credentials.AuthType)
	assert.Equal(t, "global-secret", resolved.Credentials.BearerToken)
}

func TestResolveMLflowRequestConfig_MultiUserRejectsInheritedCredentialSecretRefWithoutLauncherOverride(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType": "basic-auth",
			},
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"settings": map[string]interface{}{
						"credentialSecretRef": map[string]interface{}{
							"usernameKey": "server-username",
							"passwordKey": "server-password",
						},
					},
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, nil)
	})

	clientSet := k8sfake.NewClientset()
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), "", "ns1")
	require.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "credentialSecretRef is required")
	assert.Contains(t, err.Error(), commonmlflow.AuthTypeBasicAuth)
}

func TestResolveMLflowRequestConfig_MultiUserLauncherCredentialSecretRefWinsOverServerSide(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"authType": "bearer",
				"credentialSecretRef": map[string]interface{}{
					"tokenKey": "global-token",
				},
			},
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"settings": map[string]interface{}{
						"credentialSecretRef": map[string]interface{}{
							"tokenKey": "server-token",
						},
					},
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, nil)
	})
	launcherCfgOverride := `{
		"settings": {
			"credentialSecretRef": {
				"tokenKey": "launcher-token"
			}
		}
	}`
	clientSet := k8sfake.NewClientset(
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      commonmlflow.CredentialSecretName,
				Namespace: "ns1",
			},
			Data: map[string][]byte{
				"global-token":   []byte("global-secret"),
				"server-token":   []byte("server-secret"),
				"launcher-token": []byte("launcher-secret"),
			},
		},
	)
	kubeClients := &fakeKubeClientProvider{clientSet: clientSet}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), launcherCfgOverride, "ns1")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Config.Settings)
	require.NotNil(t, resolved.Config.Settings.CredentialSecretRef)
	assert.Equal(t, "launcher-token", resolved.Config.Settings.CredentialSecretRef.TokenKey)
	assert.Equal(t, commonmlflow.AuthTypeBearer, resolved.Credentials.AuthType)
	assert.Equal(t, "launcher-secret", resolved.Credentials.BearerToken)
}

func TestResolveMLflowRequestConfig_MultiUserRejectsForbiddenLauncherFields(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, nil)
	})

	launcherCfgOverride := `{
					"endpoint": "https://forbidden-launcher-endpoint.example.com"
				}`

	kubeClients := &fakeKubeClientProvider{clientSet: k8sfake.NewClientset()}

	resolved, err := ResolveMLflowRequestConfig(context.Background(), kubeClients.GetClientSet(), launcherCfgOverride, "ns1")
	require.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), `unknown field "endpoint"`)
}

func TestGetLauncherNamespaceMLflowConfig_RejectsTrailingJSON(t *testing.T) {
	cfg, err := GetLauncherNamespaceMLflowConfig("ns1", `{"settings":{"defaultExperimentName":"LauncherDefault"}} true`)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse MLflow config")
}

func TestGetServerSideNamespaceMLflowConfig_RejectsUnknownFields(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"bogusField": true,
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	cfg, err := GetServerSideNamespaceMLflowConfig("ns1")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), `unknown field "bogusfield"`)
}

func TestResolveMLflowCredentials_EmptySAToken(t *testing.T) {
	setupFakeKubernetesConfig(t, "")

	_, err := commonmlflow.ResolveMLflowCredentials()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bearer token is empty")
}

func TestResolveBearerSecretCredentials(t *testing.T) {
	clientSet := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      commonmlflow.CredentialSecretName,
			Namespace: "ns1",
		},
		Data: map[string][]byte{
			"token": []byte("custom-token"),
		},
	})

	credentials, err := resolveBearerSecretCredentials(
		context.Background(),
		clientSet,
		"ns1",
		&commonplugins.CredentialSecretRef{
			TokenKey: "token",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, commonmlflow.AuthTypeBearer, credentials.AuthType)
	assert.Equal(t, "custom-token", credentials.BearerToken)
}

func TestResolveBasicAuthSecretCredentials_MissingPasswordKey(t *testing.T) {
	clientSet := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      commonmlflow.CredentialSecretName,
			Namespace: "ns1",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
		},
	})

	_, err := resolveBasicAuthSecretCredentials(
		context.Background(),
		clientSet,
		"ns1",
		&commonplugins.CredentialSecretRef{
			UsernameKey: "username",
			PasswordKey: "password",
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not contain key "password"`)
}

func TestBuildMLflowRequestContext_InvalidEndpoint(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "not-a-valid-url",
		Timeout:  "10s",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			AuthType: commonmlflow.AuthTypeNone,
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.endpoint")
	assert.Contains(t, err.Error(), "must be an absolute URL with a host")
}

func TestBuildMLflowRequestContext_ZeroTimeout(t *testing.T) {
	setupFakeKubernetesConfig(t, "valid-token")

	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "0s",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{}),
	}, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: "valid-token",
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "timeout must be > 0")
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

	env := []corev1.EnvVar{{
		Name:  commonmlflow.EnvMLflowConfig,
		Value: `{"endpoint":"https://mlflow.example.com","parentRunId":"abc"}`,
	}}
	err := InjectMLflowRuntimeEnv(workflow, env)
	require.NoError(t, err)

	expectedEnv := env[0]

	// Driver container gets the env var.
	assert.Contains(t, workflow.Spec.Templates[0].Container.Env, expectedEnv)

	// Launcher main container (template with --copy init container) gets the env var.
	assert.Contains(t, workflow.Spec.Templates[1].Container.Env, expectedEnv)

	// Launcher init container does NOT get the env var (it only copies the binary).
	assert.NotContains(t, workflow.Spec.Templates[1].InitContainers[0].Env, expectedEnv)
}

func TestInjectMLflowRuntimeEnv_NilSpec(t *testing.T) {
	err := InjectMLflowRuntimeEnv(nil, []corev1.EnvVar{{Name: "key", Value: "val"}})
	require.NoError(t, err, "nil spec should be a no-op")
}

func TestInjectMLflowRuntimeEnv_EmptyEnv(t *testing.T) {
	workflow := util.NewWorkflow(&workflowapi.Workflow{})
	err := InjectMLflowRuntimeEnv(workflow, []corev1.EnvVar{})
	require.NoError(t, err, "empty env should be a no-op")
}

// newTestMLflowRequestContext creates a *commonmlflow.RequestContext backed by a service account
// token, pointing at serverURL with workspaces enabled.
func newTestMLflowRequestContext(t *testing.T, serverURL string) *commonmlflow.RequestContext {
	t.Helper()
	setupFakeKubernetesConfig(t, "bearer-secret")

	enabled := true
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: serverURL,
		Timeout:  "10s",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled: &enabled,
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeKubernetes,
		BearerToken: "bearer-secret",
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	return ctx
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		want    bool
	}{
		{
			name: "enabled when plugins.mlflow is set",
			setup: func() {
				viper.Set("plugins.mlflow", map[string]interface{}{
					"endpoint": "https://mlflow.example.com",
				})
			},
			cleanup: func() {
				viper.Set("plugins", nil)
			},
			want: true,
		},
		{
			name: "disabled when plugins.mlflow is not set",
			setup: func() {
				viper.Set("plugins", nil)
			},
			cleanup: func() {},
			want:    false,
		},
		{
			name: "disabled when plugins is nil",
			setup: func() {
				viper.Set("plugins", nil)
			},
			cleanup: func() {},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()
			got := IsEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetServerSideNamespaceMLflowConfig_EmptyNamespace(t *testing.T) {
	cfg, err := GetServerSideNamespaceMLflowConfig("")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestGetServerSideNamespaceMLflowConfig_MissingNamespace(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
			"namespaces": map[string]interface{}{
				"ns1": map[string]interface{}{
					"endpoint": "https://ns1-mlflow.example.com",
				},
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	cfg, err := GetServerSideNamespaceMLflowConfig("ns2")
	require.NoError(t, err)
	assert.Nil(t, cfg, "non-existent namespace should return nil")
}

func TestGetServerSideNamespaceMLflowConfig_NoNamespacesConfigured(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "https://global-mlflow.example.com",
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	cfg, err := GetServerSideNamespaceMLflowConfig("ns1")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestGetServerSideNamespaceMLflowConfig_TrailingJSON(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"namespaces": map[string]interface{}{
				"ns1": json.RawMessage(`{"endpoint":"https://ns1.example.com"} true`),
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	cfg, err := GetServerSideNamespaceMLflowConfig("ns1")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to marshal")
}

func TestApplyLauncherNamespaceOverrides_NilLauncher(t *testing.T) {
	base := commonmlflow.MLflowPluginConfig{
		Endpoint: "https://base.example.com",
		Settings: &commonmlflow.MLflowPluginSettings{
			DefaultExperimentName: "Base",
		},
	}
	result := applyLauncherNamespaceOverrides(base, nil)
	assert.Equal(t, base, result)
}

func TestApplyLauncherNamespaceOverrides_WithOverrides(t *testing.T) {
	desc := "Base Description"
	base := commonmlflow.MLflowPluginConfig{
		Endpoint: "https://base.example.com",
		Settings: &commonmlflow.MLflowPluginSettings{
			DefaultExperimentName: "Base",
			ExperimentDescription: &desc,
		},
	}
	launcherDesc := "Launcher Description"
	injectVars := true
	launcher := &LauncherNamespaceMLflowConfig{
		Settings: &LauncherNamespaceMLflowSettings{
			ExperimentDescription: &launcherDesc,
			DefaultExperimentName: "LauncherExp",
			InjectUserEnvVars:     &injectVars,
		},
	}
	result := applyLauncherNamespaceOverrides(base, launcher)
	require.NotNil(t, result.Settings)
	assert.Equal(t, "LauncherExp", result.Settings.DefaultExperimentName)
	require.NotNil(t, result.Settings.ExperimentDescription)
	assert.Equal(t, "Launcher Description", *result.Settings.ExperimentDescription)
	require.NotNil(t, result.Settings.InjectUserEnvVars)
	assert.True(t, *result.Settings.InjectUserEnvVars)
}

func TestMergeLauncherNamespaceSettings_NilOverrides(t *testing.T) {
	base := &commonmlflow.MLflowPluginSettings{
		DefaultExperimentName: "Base",
	}
	result := mergeLauncherNamespaceSettings(base, nil)
	assert.Equal(t, base, result)
}

func TestMergeLauncherNamespaceSettings_NilBase(t *testing.T) {
	desc := "Override Description"
	overrides := &LauncherNamespaceMLflowSettings{
		ExperimentDescription: &desc,
		DefaultExperimentName: "Override",
	}
	result := mergeLauncherNamespaceSettings(nil, overrides)
	require.NotNil(t, result)
	assert.Equal(t, "Override", result.DefaultExperimentName)
	require.NotNil(t, result.ExperimentDescription)
	assert.Equal(t, "Override Description", *result.ExperimentDescription)
}

func TestMergeLauncherNamespaceSettings_PartialOverrides(t *testing.T) {
	baseDesc := "Base Description"
	base := &commonmlflow.MLflowPluginSettings{
		DefaultExperimentName: "Base",
		ExperimentDescription: &baseDesc,
		AuthType:              commonmlflow.AuthTypeNone,
	}
	injectVars := true
	overrides := &LauncherNamespaceMLflowSettings{
		InjectUserEnvVars: &injectVars,
	}
	result := mergeLauncherNamespaceSettings(base, overrides)
	require.NotNil(t, result)
	assert.Equal(t, "Base", result.DefaultExperimentName)
	assert.Equal(t, &baseDesc, result.ExperimentDescription)
	assert.Equal(t, commonmlflow.AuthTypeNone, result.AuthType)
	require.NotNil(t, result.InjectUserEnvVars)
	assert.True(t, *result.InjectUserEnvVars)
}

func TestMergeLauncherNamespaceSettings_CredentialSecretRef(t *testing.T) {
	base := &commonmlflow.MLflowPluginSettings{
		CredentialSecretRef: &commonplugins.CredentialSecretRef{
			TokenKey: "base-token",
		},
	}
	overrides := &LauncherNamespaceMLflowSettings{
		CredentialSecretRef: &commonplugins.CredentialSecretRef{
			TokenKey: "override-token",
		},
	}
	result := mergeLauncherNamespaceSettings(base, overrides)
	require.NotNil(t, result)
	require.NotNil(t, result.CredentialSecretRef)
	assert.Equal(t, "override-token", result.CredentialSecretRef.TokenKey)
}

func TestBuildMLflowRunRequestContext_NilConfig(t *testing.T) {
	ctx, err := BuildMLflowRunRequestContext("ns1", nil)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "MLflow config is nil")
}

func TestBuildMLflowRunRequestContext_NilSettings(t *testing.T) {
	requestCfg := &ResolvedMLflowConfig{
		Config: &commonmlflow.MLflowPluginConfig{
			Endpoint: "https://mlflow.example.com",
			Settings: nil,
		},
	}
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "MLflow plugin settings are nil")
}

func TestBuildMLflowRunRequestContext_EmptyEndpoint(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "endpoint must be set")
}

func TestBuildMLflowRunRequestContext_InvalidKFPBaseURL_WithQuery(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			KFPBaseURL: "https://kfp.example.com?tenant=a",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.kfpBaseURL")
	assert.Contains(t, err.Error(), "must not contain a query string")
}

func TestBuildMLflowRunRequestContext_InvalidKFPBaseURL_WithFragment(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			KFPBaseURL: "https://kfp.example.com/#/tenant",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.kfpBaseURL")
	assert.Contains(t, err.Error(), "must not contain a fragment")
}

func TestBuildMLflowRunRequestContext_InvalidKFPBaseURL_WithPath(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			KFPBaseURL: "https://kfp.example.com/prefix",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.kfpBaseURL")
	assert.Contains(t, err.Error(), "must not contain a path")
}

func TestBuildMLflowRunRequestContext_InvalidMLflowBaseURL_WithQuery(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			MLflowBaseURL: "https://mlflow-ui.example.com?tenant=a",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.mlflowBaseURL")
	assert.Contains(t, err.Error(), "must not contain a query string")
}

func TestBuildMLflowRunRequestContext_InvalidMLflowBaseURL_WithFragment(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			MLflowBaseURL: "https://mlflow-ui.example.com/#/tenant",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.mlflowBaseURL")
	assert.Contains(t, err.Error(), "must not contain a fragment")
}

func TestBuildMLflowRunRequestContext_InvalidMLflowBaseURL_WithPath(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			MLflowBaseURL: "https://mlflow-ui.example.com/prefix",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "plugins.mlflow.settings.mlflowBaseURL")
	assert.Contains(t, err.Error(), "must not contain a path")
}

func TestBuildMLflowRunRequestContext_ValidBaseURLs(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "30s",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			KFPBaseURL:    "https://kfp.example.com",
			MLflowBaseURL: "https://mlflow-ui.example.com",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestBuildMLflowRunRequestContext_EmptyBaseURLsAreAllowed(t *testing.T) {
	requestCfg := mustResolvedConfig(t, &commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Timeout:  "30s",
		Settings: ApplyMLflowSettingsDefaults(&commonmlflow.MLflowPluginSettings{
			KFPBaseURL:    "",
			MLflowBaseURL: "",
		}),
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	ctx, err := BuildMLflowRunRequestContext("ns1", requestCfg)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestNewResolvedMLflowConfig_NilConfig(t *testing.T) {
	cfg, err := newResolvedMLflowConfig(nil, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "MLflow config is nil")
}

func TestNewResolvedMLflowConfig_NilSettings(t *testing.T) {
	cfg, err := newResolvedMLflowConfig(&commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: nil,
	}, commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeNone,
	})
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "MLflow plugin settings are nil")
}

func TestNewResolvedMLflowConfig_EmptyAuthType(t *testing.T) {
	cfg, err := newResolvedMLflowConfig(&commonmlflow.MLflowPluginConfig{
		Endpoint: "https://mlflow.example.com",
		Settings: &commonmlflow.MLflowPluginSettings{
			AuthType: commonmlflow.AuthTypeBearer,
		},
	}, commonmlflow.MLflowCredentials{
		AuthType: "",
	})
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "missing resolved credentials")
}

func TestResolveConfiguredCredentials_NilSettings(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "settings are nil")
}

func TestResolveConfiguredCredentials_UnsupportedAuthType(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", &commonmlflow.MLflowPluginSettings{
		AuthType: "unsupported-auth-type",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported plugins.mlflow.settings.authType")
}

func TestResolveConfiguredCredentials_KubernetesAuth(t *testing.T) {
	setupFakeKubernetesConfig(t, "k8s-token")
	clientSet := k8sfake.NewClientset()
	creds, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", &commonmlflow.MLflowPluginSettings{
		AuthType: commonmlflow.AuthTypeKubernetes,
	})
	require.NoError(t, err)
	assert.Equal(t, commonmlflow.AuthTypeKubernetes, creds.AuthType)
	assert.Equal(t, "k8s-token", creds.BearerToken)
}

func TestResolveConfiguredCredentials_NoneAuth(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	creds, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", &commonmlflow.MLflowPluginSettings{
		AuthType: commonmlflow.AuthTypeNone,
	})
	require.NoError(t, err)
	assert.Equal(t, commonmlflow.AuthTypeNone, creds.AuthType)
}

func TestResolveConfiguredCredentials_BearerMissingRef(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", &commonmlflow.MLflowPluginSettings{
		AuthType:            commonmlflow.AuthTypeBearer,
		CredentialSecretRef: nil,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentialSecretRef is required")
}

func TestResolveConfiguredCredentials_BasicAuthMissingRef(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveConfiguredCredentials(context.Background(), clientSet, "ns1", &commonmlflow.MLflowPluginSettings{
		AuthType:            commonmlflow.AuthTypeBasicAuth,
		CredentialSecretRef: nil,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentialSecretRef is required")
}

func TestResolveBearerSecretCredentials_NilRef(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveBearerSecretCredentials(context.Background(), clientSet, "ns1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bearer auth requires credentialSecretRef")
}

func TestResolveBearerSecretCredentials_MissingTokenKey(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveBearerSecretCredentials(context.Background(), clientSet, "ns1", &commonplugins.CredentialSecretRef{
		TokenKey: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tokenKey is required")
}

func TestResolveBasicAuthSecretCredentials_NilRef(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveBasicAuthSecretCredentials(context.Background(), clientSet, "ns1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "basic auth requires credentialSecretRef")
}

func TestResolveBasicAuthSecretCredentials_MissingUsernameKey(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := resolveBasicAuthSecretCredentials(context.Background(), clientSet, "ns1", &commonplugins.CredentialSecretRef{
		UsernameKey: "",
		PasswordKey: "password",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usernameKey is required")
}

func TestResolveBasicAuthSecretCredentials_Success(t *testing.T) {
	clientSet := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      commonmlflow.CredentialSecretName,
			Namespace: "ns1",
		},
		Data: map[string][]byte{
			"username": []byte("test-user"),
			"password": []byte("test-pass"),
		},
	})

	creds, err := resolveBasicAuthSecretCredentials(context.Background(), clientSet, "ns1", &commonplugins.CredentialSecretRef{
		UsernameKey: "username",
		PasswordKey: "password",
	})
	require.NoError(t, err)
	assert.Equal(t, commonmlflow.AuthTypeBasicAuth, creds.AuthType)
	assert.Equal(t, "test-user", creds.Username)
	assert.Equal(t, "test-pass", creds.Password)
}

func TestGetMLflowCredentialSecret_NilClientSet(t *testing.T) {
	_, err := getMLflowCredentialSecret(context.Background(), nil, "ns1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientSet is nil")
}

func TestGetMLflowCredentialSecret_SecretNotFound(t *testing.T) {
	clientSet := k8sfake.NewClientset()
	_, err := getMLflowCredentialSecret(context.Background(), clientSet, "ns1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read MLflow credentials secret")
}

func TestReadRequiredSecretKey_MissingKey(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"other-key": []byte("value"),
		},
	}
	_, err := readRequiredSecretKey(secret, "ns1", "missing-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not contain key "missing-key"`)
}

func TestReadRequiredSecretKey_EmptyValue(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"empty-key": []byte(""),
		},
	}
	_, err := readRequiredSecretKey(secret, "ns1", "empty-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has an empty value")
}

func TestReadRequiredSecretKey_WhitespaceValue(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"whitespace-key": []byte("   \n\t   "),
		},
	}
	_, err := readRequiredSecretKey(secret, "ns1", "whitespace-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has an empty value")
}

func TestReadRequiredSecretKey_Success(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"valid-key": []byte("  valid-value  \n"),
		},
	}
	value, err := readRequiredSecretKey(secret, "ns1", "valid-key")
	require.NoError(t, err)
	assert.Equal(t, "valid-value", value)
}
