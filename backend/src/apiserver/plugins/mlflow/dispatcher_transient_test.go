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
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type recordingPluginsOutputStore struct {
	writes []*model.LargeText
}

func (s *recordingPluginsOutputStore) UpdateRunPluginsOutput(_ string, output *model.LargeText) error {
	s.writes = append(s.writes, output)
	return nil
}

// A transient Kubernetes error while reading the MLflow credentials Secret is
// not a permanent configuration problem. OnRunEnd must ask the persistence
// agent to retry instead of reporting "nothing to retry", otherwise the MLflow
// parent run is never closed once the run is finalized.
func TestDispatcherOnRunEnd_TransientSecretReadFailureRequestsRetry(t *testing.T) {
	var parentUpdates atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			parentUpdates.Add(1)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			_, _ = w.Write([]byte(`{"runs":[]}`))
		default:
			t.Errorf("unexpected MLflow path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": server.URL,
			"timeout":  "1s",
			"settings": map[string]interface{}{
				"authType":            "bearer",
				"credentialSecretRef": map[string]interface{}{"tokenKey": "token"},
			},
		},
	})
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
		viper.Set(common.MultiUserMode, nil)
	})

	clientSet := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: commonmlflow.CredentialSecretName, Namespace: "ns1"},
		Data:       map[string][]byte{"token": []byte("secret-token")},
	})
	var secretReads atomic.Int32
	clientSet.PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		if secretReads.Add(1) == 1 {
			return true, nil, apierrors.NewServiceUnavailable("etcdserver: leader changed")
		}
		return false, nil, nil
	})

	store := &recordingPluginsOutputStore{}
	dispatcher, err := apiserverPlugins.NewRunPluginDispatcherImpl(
		[]apiserverPlugins.RunPluginHandler{NewMLflowRunHandler()},
		&fakeKubeClientProvider{clientSet: clientSet},
		store,
	)
	require.NoError(t, err)

	run := testPersistedRunWithPluginOutput("r-transient-secret", SuccessfulPluginOutput("exp-1", "Default", "parent-1", ""))
	run.State = "SUCCEEDED"

	nothingToRetry := dispatcher.OnRunEnd(context.Background(), run)

	require.Equal(t, int32(1), secretReads.Load(), "the Secret read should have been attempted once")
	if nothingToRetry {
		assert.NotZero(t, parentUpdates.Load(),
			"OnRunEnd reported nothing to retry, but the MLflow parent run was never closed; "+
				"the persistence agent will finalize the run and the parent stays RUNNING")
	}
	assert.False(t, nothingToRetry,
		"a transient Secret read failure (503) should request a retry, not be treated as permanently unavailable config")
}

// A Secret that does not exist is a configuration problem an operator has to
// fix, so it must keep the current "nothing to retry" behavior and not strand
// the run in terminal-report retry.
func TestDispatcherOnRunEnd_MissingSecretDoesNotRequestRetry(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "http://127.0.0.1:1",
			"timeout":  "1s",
			"settings": map[string]interface{}{
				"authType":            "bearer",
				"credentialSecretRef": map[string]interface{}{"tokenKey": "token"},
			},
		},
	})
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
		viper.Set(common.MultiUserMode, nil)
	})

	store := &recordingPluginsOutputStore{}
	dispatcher, err := apiserverPlugins.NewRunPluginDispatcherImpl(
		[]apiserverPlugins.RunPluginHandler{NewMLflowRunHandler()},
		&fakeKubeClientProvider{clientSet: k8sfake.NewClientset()},
		store,
	)
	require.NoError(t, err)

	run := testPersistedRunWithPluginOutput("r-missing-secret", SuccessfulPluginOutput("exp-1", "Default", "parent-1", ""))
	run.State = "SUCCEEDED"

	assert.True(t, dispatcher.OnRunEnd(context.Background(), run), "a missing Secret is permanent and must not request a retry")
	assert.Len(t, store.writes, 1, "the failed plugin output should still be persisted")
}

// A 500 from the kube-apiserver can persist, and a deferred terminal report is
// requeued without limit, so it must stay on the permanent path.
func TestDispatcherOnRunEnd_InternalErrorOnSecretReadDoesNotRequestRetry(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": "http://127.0.0.1:1",
			"timeout":  "1s",
			"settings": map[string]interface{}{
				"authType":            "bearer",
				"credentialSecretRef": map[string]interface{}{"tokenKey": "token"},
			},
		},
	})
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
		viper.Set(common.MultiUserMode, nil)
	})

	clientSet := k8sfake.NewClientset()
	clientSet.PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("storage error"))
	})

	store := &recordingPluginsOutputStore{}
	dispatcher, err := apiserverPlugins.NewRunPluginDispatcherImpl(
		[]apiserverPlugins.RunPluginHandler{NewMLflowRunHandler()},
		&fakeKubeClientProvider{clientSet: clientSet},
		store,
	)
	require.NoError(t, err)

	run := testPersistedRunWithPluginOutput("r-internal-secret", SuccessfulPluginOutput("exp-1", "Default", "parent-1", ""))
	run.State = "SUCCEEDED"

	assert.True(t, dispatcher.OnRunEnd(context.Background(), run), "a 500 on the Secret read must not request a retry")
	assert.Len(t, store.writes, 1, "the failed plugin output should still be persisted")
}
