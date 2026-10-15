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
	"testing"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func credentialSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: mlflow.CredentialSecretName, Namespace: "ns1"},
		Data: map[string][]byte{
			"custom-token":    []byte("  secret-token\n"),
			"custom-username": []byte("  secret-user\n"),
			"custom-password": []byte("  secret-password\n"),
		},
	}
}

func TestResolveSecretMLflowCredentials_RequiredReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		authType string
		ref      *commonplugins.CredentialSecretRef
		wantErr  string
	}{
		{name: "bearer nil ref", authType: mlflow.AuthTypeBearer, wantErr: "credentialSecretRef is required"},
		{name: "basic nil ref", authType: mlflow.AuthTypeBasicAuth, wantErr: "credentialSecretRef is required"},
		{name: "missing token key", authType: mlflow.AuthTypeBearer, ref: &commonplugins.CredentialSecretRef{}, wantErr: "tokenKey is required"},
		{name: "missing username key", authType: mlflow.AuthTypeBasicAuth, ref: &commonplugins.CredentialSecretRef{PasswordKey: "custom-password"}, wantErr: "usernameKey is required"},
		{name: "missing password key", authType: mlflow.AuthTypeBasicAuth, ref: &commonplugins.CredentialSecretRef{UsernameKey: "custom-username"}, wantErr: "passwordKey is required"},
		{name: "unsupported auth nil ref", authType: mlflow.AuthTypeNone, wantErr: "unsupported secret-based MLflow auth type"},
		{name: "unsupported auth with ref", authType: mlflow.AuthTypeNone, ref: &commonplugins.CredentialSecretRef{}, wantErr: "unsupported secret-based MLflow auth type"},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := mlflow.ResolveSecretMLflowCredentials(context.Background(), k8sfake.NewClientset(credentialSecret()), "ns1", test.ref, test.authType)
			require.ErrorContains(t, err, test.wantErr)
			assert.Empty(t, credentials)
		})
	}
}

func TestResolveSecretMLflowCredentials_InvalidValues(t *testing.T) {
	for _, credential := range []struct {
		role     string
		key      string
		authType string
	}{
		{role: "bearer token", key: "custom-token", authType: mlflow.AuthTypeBearer},
		{role: "username", key: "custom-username", authType: mlflow.AuthTypeBasicAuth},
		{role: "password", key: "custom-password", authType: mlflow.AuthTypeBasicAuth},
	} {
		for _, value := range []struct {
			name string
			data []byte
		}{
			{name: "missing"},
			{name: "empty", data: []byte("")},
			{name: "whitespace", data: []byte(" \n\t ")},
		} {
			t.Run(credential.role+"/"+value.name, func(t *testing.T) {
				secret := credentialSecret()
				if value.data == nil {
					delete(secret.Data, credential.key)
				} else {
					secret.Data[credential.key] = value.data
				}
				credentials, err := mlflow.ResolveSecretMLflowCredentials(context.Background(), k8sfake.NewClientset(secret), "ns1", &commonplugins.CredentialSecretRef{
					TokenKey: "custom-token", UsernameKey: "custom-username", PasswordKey: "custom-password",
				}, credential.authType)
				require.ErrorContains(t, err, "required MLflow "+credential.role+" credential")
				if value.data == nil {
					assert.Contains(t, err.Error(), "does not contain a value")
				} else {
					assert.Contains(t, err.Error(), "empty value")
				}
				assert.Empty(t, credentials)
				for _, privateValue := range []string{"custom-token", "custom-username", "custom-password", "secret-token", "secret-user", "secret-password"} {
					assert.NotContains(t, err.Error(), privateValue)
				}
			})
		}
	}
}

func TestResolveSecretMLflowCredentials_SecretLookupFailures(t *testing.T) {
	for _, test := range []struct {
		name      string
		client    kubernetes.Interface
		namespace string
		wantErr   string
	}{
		{name: "nil client", namespace: "ns1", wantErr: "clientSet is nil"},
		{name: "empty namespace", client: k8sfake.NewClientset(), wantErr: "namespace is empty"},
		{name: "missing Secret", client: k8sfake.NewClientset(), namespace: "ns1", wantErr: "failed to read MLflow credentials secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := mlflow.ResolveSecretMLflowCredentials(context.Background(), test.client, test.namespace, &commonplugins.CredentialSecretRef{TokenKey: "custom-token"}, mlflow.AuthTypeBearer)
			require.ErrorContains(t, err, test.wantErr)
			assert.Empty(t, credentials)
			if test.name == "missing Secret" {
				var statusErr *apierrors.StatusError
				require.ErrorAs(t, err, &statusErr)
				assert.True(t, apierrors.IsNotFound(statusErr))
			}
		})
	}
}

func TestResolveSecretMLflowCredentials_Success(t *testing.T) {
	for _, test := range []struct {
		name string
		want mlflow.MLflowCredentials
	}{
		{name: "bearer", want: mlflow.MLflowCredentials{AuthType: mlflow.AuthTypeBearer, BearerToken: "secret-token"}},
		{name: "basic auth", want: mlflow.MLflowCredentials{AuthType: mlflow.AuthTypeBasicAuth, Username: "secret-user", Password: "secret-password"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := mlflow.ResolveSecretMLflowCredentials(context.Background(), k8sfake.NewClientset(credentialSecret()), "ns1", &commonplugins.CredentialSecretRef{
				TokenKey: "custom-token", UsernameKey: "custom-username", PasswordKey: "custom-password",
			}, test.want.AuthType)
			require.NoError(t, err)
			assert.Equal(t, test.want, credentials)
		})
	}
}
