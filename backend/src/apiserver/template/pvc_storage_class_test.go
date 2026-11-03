// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// driverPluginArgsFromWorkflow extracts the args map of the container-driver
// executor plugin from a marshaled compiled workflow or ScheduledWorkflow.
func driverPluginArgsFromWorkflow(t *testing.T, marshaled []byte) map[string]any {
	t.Helper()
	var root map[string]any
	require.NoError(t, json.Unmarshal(marshaled, &root))
	spec, ok := root["spec"].(map[string]any)
	require.True(t, ok, "workflow should have a spec")
	// A ScheduledWorkflow embeds the Argo workflow as a JSON string under
	// spec.workflow.spec.
	if workflow, ok := spec["workflow"].(map[string]any); ok {
		workflowJSON, ok := workflow["spec"].(string)
		require.True(t, ok, "embedded workflow should have a spec")
		require.NoError(t, json.Unmarshal([]byte(workflowJSON), &root))
		spec, ok = root["spec"].(map[string]any)
		require.True(t, ok, "embedded workflow should have a spec")
	}
	templates, ok := spec["templates"].([]any)
	require.True(t, ok, "workflow should have spec.templates")
	for _, tmpl := range templates {
		plugin, ok := tmpl.(map[string]any)["plugin"].(map[string]any)
		if !ok {
			continue
		}
		driverPlugin, ok := plugin["driver-plugin"].(map[string]any)
		if !ok {
			continue
		}
		args, ok := driverPlugin["args"].(map[string]any)
		if !ok {
			continue
		}
		if args["type"] == "CONTAINER" {
			return args
		}
	}
	t.Fatal("workflow should contain a CONTAINER driver template with plugin arguments")
	return nil
}

// TestTemplateOptionsDefaultPVCStorageClassPropagates verifies that the
// server-configured default StorageClass set on TemplateOptions reaches the
// compiled workflow's container-driver plugin arguments, for both one-off
// runs and recurring runs (ScheduledWorkflow).
func TestTemplateOptionsDefaultPVCStorageClassPropagates(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	specBytes, err := os.ReadFile("testdata/pipeline_with_volume.yaml")
	require.NoError(t, err)

	for _, test := range []struct {
		name         string
		defaultClass *string
		wantKey      bool
	}{
		{name: "configured propagates to driver plugin args", defaultClass: new("server-default"), wantKey: true},
		{name: "unset omits the key", wantKey: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := New(specBytes, TemplateOptions{DefaultPVCStorageClass: test.defaultClass})
			require.NoError(t, err)

			t.Run("RunWorkflow", func(t *testing.T) {
				executionSpec, err := tmpl.RunWorkflow(&model.Run{DisplayName: "run"}, RunWorkflowOptions{})
				require.NoError(t, err)
				marshaled, err := json.Marshal(executionSpec)
				require.NoError(t, err)
				args := driverPluginArgsFromWorkflow(t, marshaled)
				if test.wantKey {
					assert.Equal(t, *test.defaultClass, args["default_pvc_storage_class"])
				} else {
					assert.NotContains(t, args, "default_pvc_storage_class")
				}
			})

			t.Run("ScheduledWorkflow", func(t *testing.T) {
				swf, err := tmpl.ScheduledWorkflow(&model.Job{DisplayName: "job", K8SName: "job"})
				require.NoError(t, err)
				marshaled, err := json.Marshal(swf)
				require.NoError(t, err)
				args := driverPluginArgsFromWorkflow(t, marshaled)
				if test.wantKey {
					assert.Equal(t, *test.defaultClass, args["default_pvc_storage_class"])
				} else {
					assert.NotContains(t, args, "default_pvc_storage_class")
				}
			})
		})
	}
}
