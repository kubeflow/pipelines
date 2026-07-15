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
	"net/http"
	"net/http/httptest"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

type fakeRunPluginOutputStore struct {
	updatedRunIDs []string
}

func (f *fakeRunPluginOutputStore) UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error {
	f.updatedRunIDs = append(f.updatedRunIDs, runID)
	return nil
}

// TestDispatcherOnRunEnd_PermanentConfigFailureDoesNotRequestRetry covers the
// stranded-run scenario: when the MLflow config is permanently unavailable,
// OnRunEnd must record the failure in the plugin output and report "nothing to
// retry" so the run can still be finalized and its workflow garbage collected.
func TestDispatcherOnRunEnd_PermanentConfigFailureDoesNotRequestRetry(t *testing.T) {
	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", nil)
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	outputStore := &fakeRunPluginOutputStore{}
	dispatcher := NewRunPluginDispatcher(&fakeKubeClientProvider{clientSet: k8sfake.NewClientset()}, outputStore)

	pluginOutput := SuccessfulPluginOutput("exp-1", "Default", "parent-1", "")
	run := testPersistedRunWithPluginOutput("r-permanent-config", pluginOutput)
	run.State = "FAILED"

	assert.True(t, dispatcher.OnRunEnd(context.Background(), run),
		"a permanent config failure must not request a retry")

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Contains(t, result.StateMessage, "config unavailable")
	assert.Equal(t, []string{"r-permanent-config"}, outputStore.updatedRunIDs,
		"the failed plugin output should still be persisted")
}

// TestDispatcherOnRunEnd_TransientSyncFailureRequestsRetry verifies that a
// failing MLflow server (as opposed to missing config) still requests a retry
// when a parent run is left open.
func TestDispatcherOnRunEnd_TransientSyncFailureRequestsRetry(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR"}`))
	}))
	defer server.Close()

	originalPlugins := viper.Get("plugins")
	viper.Set("plugins", map[string]interface{}{
		"mlflow": map[string]interface{}{
			"endpoint": server.URL,
			"timeout":  "10s",
			"settings": map[string]interface{}{
				"workspacesEnabled": false,
			},
		},
	})
	t.Cleanup(func() {
		viper.Set("plugins", originalPlugins)
	})

	outputStore := &fakeRunPluginOutputStore{}
	dispatcher := NewRunPluginDispatcher(&fakeKubeClientProvider{clientSet: k8sfake.NewClientset()}, outputStore)

	pluginOutput := SuccessfulPluginOutput("exp-1", "Default", "parent-1", "")
	run := testPersistedRunWithPluginOutput("r-transient-sync", pluginOutput)
	run.State = "FAILED"

	assert.False(t, dispatcher.OnRunEnd(context.Background(), run),
		"a transient MLflow sync failure should request a retry")

	result := run.PluginsOutput[PluginName]
	require.NotNil(t, result)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, result.State)
	assert.Equal(t, []string{"r-transient-sync"}, outputStore.updatedRunIDs)
}
