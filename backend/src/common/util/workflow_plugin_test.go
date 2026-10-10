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
)

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
