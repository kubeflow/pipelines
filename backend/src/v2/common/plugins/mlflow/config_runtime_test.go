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

package mlflow_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins/mlflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{commonmlflow.EnvMLflowTrackingToken, commonmlflow.EnvMLflowTrackingUsername, commonmlflow.EnvMLflowTrackingPassword} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestBuildMLflowTaskRequestContext_RuntimeCredentials(t *testing.T) {
	for _, test := range []struct {
		name          string
		authType      string
		env           map[string]string
		authorization string
	}{
		{name: "no auth", authType: commonmlflow.AuthTypeNone},
		{name: "launcher bearer", authType: commonmlflow.AuthTypeBearer, env: map[string]string{commonmlflow.EnvMLflowTrackingToken: "  launcher-token\n"}, authorization: "Bearer launcher-token"},
		{name: "launcher basic", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingUsername: " launcher-user ", commonmlflow.EnvMLflowTrackingPassword: " launcher-pass\n"}, authorization: "Basic " + base64.StdEncoding.EncodeToString([]byte("launcher-user:launcher-pass"))},
		{name: "kubernetes", authType: commonmlflow.AuthTypeKubernetes, authorization: "Bearer service-account-token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			clearCredentialEnv(t)
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			if test.authType == commonmlflow.AuthTypeKubernetes {
				t.Setenv("KUBERNETES_SERVICE_HOST", "")
				t.Setenv("KUBERNETES_SERVICE_PORT", "")
				kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
				require.NoError(t, os.WriteFile(kubeconfig, []byte(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://localhost
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: service-account-token
`), 0600))
				t.Setenv("KUBECONFIG", kubeconfig)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/2.0/mlflow/runs/create", r.URL.Path)
				assert.Equal(t, test.authorization, r.Header.Get("Authorization"))
				assert.Equal(t, "workspace", r.Header.Get("X-MLflow-Workspace"))
				_, _ = fmt.Fprint(w, `{"run":{"info":{"run_id":"task-run"}}}`)
			}))
			defer server.Close()
			requestContext, err := mlflow.BuildMLflowTaskRequestContext(context.Background(), commonmlflow.MLflowRuntimeConfig{
				Endpoint: server.URL, Timeout: "1s", AuthType: test.authType,
				Workspace: "workspace", WorkspacesEnabled: true,
				CredentialSecretRef: &commonplugins.CredentialSecretRef{TokenKey: "token", UsernameKey: "username", PasswordKey: "password"},
			})
			require.NoError(t, err)
			require.NotNil(t, requestContext)
			runID, err := requestContext.Client.CreateRun(context.Background(), "experiment", "task", nil)
			require.NoError(t, err)
			assert.Equal(t, "task-run", runID)
		})
	}
}

func TestBuildMLflowTaskRequestContext_InvalidEnvDoesNotFallBackToSecret(t *testing.T) {
	for _, test := range []struct {
		name     string
		authType string
		env      map[string]string
		wantErr  string
	}{
		{name: "empty bearer", authType: commonmlflow.AuthTypeBearer, env: map[string]string{commonmlflow.EnvMLflowTrackingToken: ""}, wantErr: commonmlflow.EnvMLflowTrackingToken},
		{name: "whitespace bearer", authType: commonmlflow.AuthTypeBearer, env: map[string]string{commonmlflow.EnvMLflowTrackingToken: " \n\t"}, wantErr: commonmlflow.EnvMLflowTrackingToken},
		{name: "username only", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingUsername: "user"}, wantErr: commonmlflow.EnvMLflowTrackingPassword},
		{name: "password only", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingPassword: "password"}, wantErr: commonmlflow.EnvMLflowTrackingUsername},
		{name: "empty username only", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingUsername: ""}, wantErr: commonmlflow.EnvMLflowTrackingUsername},
		{name: "empty password only", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingPassword: ""}, wantErr: commonmlflow.EnvMLflowTrackingUsername},
		{name: "whitespace username", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingUsername: " \n", commonmlflow.EnvMLflowTrackingPassword: "password"}, wantErr: commonmlflow.EnvMLflowTrackingUsername},
		{name: "whitespace password", authType: commonmlflow.AuthTypeBasicAuth, env: map[string]string{commonmlflow.EnvMLflowTrackingUsername: "user", commonmlflow.EnvMLflowTrackingPassword: " \t"}, wantErr: commonmlflow.EnvMLflowTrackingPassword},
	} {
		t.Run(test.name, func(t *testing.T) {
			clearCredentialEnv(t)
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			requestContext, err := mlflow.BuildMLflowTaskRequestContext(context.Background(), commonmlflow.MLflowRuntimeConfig{
				Endpoint: "https://mlflow.example.com", Timeout: "1s", AuthType: test.authType,
				CredentialSecretRef: &commonplugins.CredentialSecretRef{TokenKey: "token", UsernameKey: "username", PasswordKey: "password"},
			})
			require.ErrorContains(t, err, test.wantErr+" is empty")
			assert.Nil(t, requestContext)
		})
	}
}

func TestBuildMLflowTaskRequestContext_UnsupportedAuthType(t *testing.T) {
	requestContext, err := mlflow.BuildMLflowTaskRequestContext(context.Background(), commonmlflow.MLflowRuntimeConfig{AuthType: "unsupported"})
	require.ErrorContains(t, err, `unsupported MLflow auth type "unsupported"`)
	assert.Nil(t, requestContext)
}
