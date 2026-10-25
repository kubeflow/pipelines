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

package template

import (
	"encoding/json"
	"os"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRunWorkflowTaskCredentials(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
	t.Setenv(argocompiler.DriverImageEnvVar, "driver:patched")
	t.Setenv(argocompiler.LauncherImageEnvVar, "launcher:patched")
	t.Setenv(argocompiler.DriverCommandEnvVar, "custom-driver --wrapped")
	t.Setenv(argocompiler.LauncherCommandEnvVar, "custom-launcher --wrapped")
	bytes, err := os.ReadFile("testdata/hello_world.yaml")
	require.NoError(t, err)
	src, err := New(bytes, TemplateOptions{})
	require.NoError(t, err)
	before := string(src.Bytes())
	run := &model.Run{Namespace: "team-a", ServiceAccount: "custom-runner", RecurringRunId: "old-schedule", PipelineSpec: model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{Parameters: `{"y":"world"}`}}}
	index := int64(7)
	for _, id := range []string{"fresh-one-off", "fresh-recurring-tick"} {
		exec, err := src.RunWorkflow(run, RunWorkflowOptions{RunID: id, RunAt: 1700000000, RecurringRunIndex: &index})
		require.NoError(t, err)
		var wf wfapi.Workflow
		require.NoError(t, json.Unmarshal([]byte(exec.ToStringForStore()), &wf))
		require.Equal(t, "custom-runner", wf.Spec.ServiceAccountName)
		require.Equal(t, "team-a", wf.Namespace)
		require.NoError(t, util.NewWorkflow(&wf).ValidateTaskRuntimeCredentials("pipelines.kubeflow.org/task-write"))
	}
	require.Equal(t, before, string(src.Bytes()))
	viper.Set(common.MultiUserMode, false)
	exec, err := src.RunWorkflow(run, RunWorkflowOptions{RunID: "single", RunAt: 1700000000})
	require.NoError(t, err)
	require.NotContains(t, exec.ToStringForStore(), util.TaskTokenPathEnv)
}
