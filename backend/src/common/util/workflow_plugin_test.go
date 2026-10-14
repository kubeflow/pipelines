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

package util_test

import (
	"encoding/json"
	"testing"

	wf "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestRuntimeConfigAndEnvVarsRespectRoles(t *testing.T) {
	for _, test := range []struct {
		name     string
		roles    []util.ExecutionRuntimeRole
		driver   bool
		launcher bool
	}{
		{name: "no roles"},
		{name: "driver", roles: []util.ExecutionRuntimeRole{util.ExecutionRuntimeRoleDriver}, driver: true},
		{name: "launcher", roles: []util.ExecutionRuntimeRole{util.ExecutionRuntimeRoleLauncher}, launcher: true},
		{name: "both", roles: []util.ExecutionRuntimeRole{util.ExecutionRuntimeRoleDriver, util.ExecutionRuntimeRoleLauncher}, driver: true, launcher: true},
		{name: "unknown", roles: []util.ExecutionRuntimeRole{"unknown"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			template := func(name string, role util.ExecutionRuntimeRole) wf.Template {
				return wf.Template{
					Name:     name,
					Metadata: wf.Metadata{Annotations: map[string]string{util.AnnotationKeyRuntimeRole: string(role)}},
					Container: &corev1.Container{Env: []corev1.EnvVar{
						{Name: "KEEP", Value: "kept"},
						{Name: "KEY", Value: "old"},
					}},
				}
			}
			pluginJSON := `{"driver-plugin":{"args":{"runtime_args":"{\"KEEP\":\"kept\"}"}}}`
			workflow := util.NewWorkflow(&wf.Workflow{Spec: wf.WorkflowSpec{Templates: []wf.Template{
				template("driver-container", util.ExecutionRuntimeRoleDriver),
				template("launcher", util.ExecutionRuntimeRoleLauncher),
				template("unannotated", ""),
				{Name: "dag", DAG: &wf.DAGTemplate{}},
				{Name: "driver-plugin", Plugin: &wf.Plugin{Object: wf.Object{Value: []byte(pluginJSON)}}},
			}}})
			envWorkflow := util.NewWorkflow(workflow.DeepCopy())
			updates := []corev1.EnvVar{{Name: "KEY", Value: "new"}, {Name: "ADDED", Value: "value"}}
			require.NoError(t, envWorkflow.UpsertRuntimeEnvVars(updates, test.roles...))
			require.NoError(t, workflow.UpsertRuntimeConfig(map[string]string{"KEY": "new", "ADDED": "value"}, test.roles...))

			for index, selected := range []bool{test.driver, test.launcher, false} {
				expected := []corev1.EnvVar{{Name: "KEEP", Value: "kept"}, {Name: "KEY", Value: "old"}}
				if selected {
					expected = append(expected[:1], updates...)
				}
				require.ElementsMatch(t, expected, workflow.Spec.Templates[index].Container.Env)
				require.ElementsMatch(t, expected, envWorkflow.Spec.Templates[index].Container.Env)
			}
			require.Nil(t, workflow.Spec.Templates[3].Container)
			require.Equal(t, pluginJSON, string(envWorkflow.Spec.Templates[4].Plugin.Value))
			if test.driver {
				var plugin map[string]map[string]map[string]string
				require.NoError(t, json.Unmarshal(workflow.Spec.Templates[4].Plugin.Value, &plugin))
				var settings map[string]string
				require.NoError(t, json.Unmarshal([]byte(plugin["driver-plugin"]["args"]["runtime_args"]), &settings))
				require.Equal(t, map[string]string{"KEEP": "kept", "KEY": "new", "ADDED": "value"}, settings)
			} else {
				require.Equal(t, pluginJSON, string(workflow.Spec.Templates[4].Plugin.Value))
			}
		})
	}
}

func TestRuntimeConfigMergesEmptyAndNullSettings(t *testing.T) {
	for _, previous := range []string{"", "null", "{}", `{"keep":"old","override":"old"}`} {
		t.Run(previous, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"driver-plugin": map[string]any{"args": map[string]any{"runtime_args": previous}}})
			require.NoError(t, err)
			workflow := &util.Workflow{Workflow: &wf.Workflow{Spec: wf.WorkflowSpec{Templates: []wf.Template{{Name: "driver", Plugin: &wf.Plugin{Object: wf.Object{Value: raw}}}}}}}
			require.NoError(t, workflow.UpsertRuntimeConfig(map[string]string{"override": "new"}, util.ExecutionRuntimeRoleDriver))
			var plugin map[string]map[string]map[string]string
			require.NoError(t, json.Unmarshal(workflow.Spec.Templates[0].Plugin.Value, &plugin))
			var settings map[string]string
			require.NoError(t, json.Unmarshal([]byte(plugin["driver-plugin"]["args"]["runtime_args"]), &settings))
			require.Equal(t, "new", settings["override"])
			if previous != "" && previous != "null" && previous != "{}" {
				require.Equal(t, "old", settings["keep"])
			}
		})
	}
}

func TestRuntimeConfigSkipsOtherPlugins(t *testing.T) {
	for _, raw := range []string{`{"hello": {"config": "unchanged"}}`, `{"driver-plugin":null}`, `{"driver-plugin":[]}`} {
		workflow := &util.Workflow{Workflow: &wf.Workflow{Spec: wf.WorkflowSpec{Templates: []wf.Template{{Name: "plugin", Plugin: &wf.Plugin{Object: wf.Object{Value: []byte(raw)}}}}}}}
		err := workflow.UpsertRuntimeConfig(map[string]string{"KFP_MLFLOW_CONFIG": "value"}, util.ExecutionRuntimeRoleDriver)
		if raw == `{"hello": {"config": "unchanged"}}` {
			require.NoError(t, err)
			require.Equal(t, raw, string(workflow.Spec.Templates[0].Plugin.Value))
		} else {
			require.Error(t, err)
		}
	}
}
