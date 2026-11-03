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

package mlflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func unsetRuntimeCredentialEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{commonmlflow.EnvMLflowTrackingToken, commonmlflow.EnvMLflowTrackingUsername, commonmlflow.EnvMLflowTrackingPassword} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func writeNamespaceFile(t *testing.T, namespace string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "namespace")
	require.NoError(t, os.WriteFile(path, []byte(namespace), 0600))
	return path
}

func TestResolveRuntimeCredentials_SecretAndNamespaceTrim(t *testing.T) {
	for _, test := range []struct {
		name         string
		want         commonmlflow.MLflowCredentials
		unrelatedEnv string
	}{
		{name: "bearer", want: commonmlflow.MLflowCredentials{AuthType: commonmlflow.AuthTypeBearer, BearerToken: "secret-token"}, unrelatedEnv: commonmlflow.EnvMLflowTrackingUsername},
		{name: "basic", want: commonmlflow.MLflowCredentials{AuthType: commonmlflow.AuthTypeBasicAuth, Username: "secret-user", Password: "secret-password"}, unrelatedEnv: commonmlflow.EnvMLflowTrackingToken},
	} {
		t.Run(test.name, func(t *testing.T) {
			unsetRuntimeCredentialEnv(t)
			t.Setenv(test.unrelatedEnv, "unrelated-credential")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/namespaces/ns1/secrets/"+commonmlflow.CredentialSecretName, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: commonmlflow.CredentialSecretName, Namespace: "ns1"},
					Data: map[string][]byte{
						"custom-token": []byte("secret-token"), "custom-username": []byte("secret-user"), "custom-password": []byte("secret-password"),
					},
				}))
			}))
			defer server.Close()
			loadConfig := func() (*rest.Config, error) { return &rest.Config{Host: server.URL}, nil }
			credentials, err := resolveRuntimeCredentials(context.Background(), commonmlflow.MLflowRuntimeConfig{
				AuthType:            test.want.AuthType,
				CredentialSecretRef: &commonplugins.CredentialSecretRef{TokenKey: "custom-token", UsernameKey: "custom-username", PasswordKey: "custom-password"},
			}, writeNamespaceFile(t, " \tns1\n"), loadConfig)
			require.NoError(t, err)
			assert.Equal(t, test.want, credentials)
		})
	}
}

func TestResolveRuntimeCredentials_NamespaceFailures(t *testing.T) {
	unsetRuntimeCredentialEnv(t)
	for _, test := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "missing namespace file", path: filepath.Join(t.TempDir(), "missing"), wantErr: "mount the service account namespace file"},
		{name: "empty namespace", path: writeNamespaceFile(t, ""), wantErr: "pod namespace for MLflow credentials is empty"},
		{name: "whitespace namespace", path: writeNamespaceFile(t, " \n\t"), wantErr: "pod namespace for MLflow credentials is empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := resolveRuntimeCredentials(context.Background(), commonmlflow.MLflowRuntimeConfig{AuthType: commonmlflow.AuthTypeBearer}, test.path, func() (*rest.Config, error) {
				t.Error("Kubernetes config must not be loaded without a namespace")
				return nil, errors.New("unexpected config load")
			})
			require.ErrorContains(t, err, test.wantErr)
			assert.Empty(t, credentials)
			if test.name == "missing namespace file" {
				assert.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}

func TestResolveRuntimeCredentials_KubernetesInitializationFailures(t *testing.T) {
	unsetRuntimeCredentialEnv(t)
	configErr := errors.New("kubeconfig unavailable")
	for _, test := range []struct {
		name    string
		config  *rest.Config
		err     error
		wantErr string
	}{
		{name: "config failure", err: configErr, wantErr: "provide service account credentials or a valid kubeconfig"},
		{name: "client failure", config: &rest.Config{Host: "https://localhost", TLSClientConfig: rest.TLSClientConfig{CAData: []byte("invalid CA")}}, wantErr: "check the Kubernetes API endpoint and TLS configuration"},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := resolveRuntimeCredentials(context.Background(), commonmlflow.MLflowRuntimeConfig{AuthType: commonmlflow.AuthTypeBearer}, writeNamespaceFile(t, "ns1"), func() (*rest.Config, error) {
				return test.config, test.err
			})
			require.ErrorContains(t, err, test.wantErr)
			assert.Empty(t, credentials)
			if test.err != nil {
				assert.ErrorIs(t, err, test.err)
			}
		})
	}
}

func TestResolveRuntimeCredentials_SecretNotFound(t *testing.T) {
	unsetRuntimeCredentialEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		assert.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{
			Status: metav1.StatusFailure, Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound,
		}))
	}))
	defer server.Close()
	credentials, err := resolveRuntimeCredentials(context.Background(), commonmlflow.MLflowRuntimeConfig{
		AuthType: commonmlflow.AuthTypeBearer, CredentialSecretRef: &commonplugins.CredentialSecretRef{TokenKey: "token"},
	}, writeNamespaceFile(t, "ns1"), func() (*rest.Config, error) { return &rest.Config{Host: server.URL}, nil })
	require.ErrorContains(t, err, "failed to read MLflow credentials secret")
	var statusErr *apierrors.StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.True(t, apierrors.IsNotFound(statusErr))
	assert.Empty(t, credentials)
}

func TestResolveRuntimeCredentials_SecretRequestCancellation(t *testing.T) {
	unsetRuntimeCredentialEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("canceled context must not send a Secret request")
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	credentials, err := resolveRuntimeCredentials(ctx, commonmlflow.MLflowRuntimeConfig{
		AuthType: commonmlflow.AuthTypeBearer, CredentialSecretRef: &commonplugins.CredentialSecretRef{TokenKey: "token"},
	}, writeNamespaceFile(t, "ns1"), func() (*rest.Config, error) { return &rest.Config{Host: server.URL}, nil })
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, credentials)
}
