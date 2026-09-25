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

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	util "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedSchedule_AcceptsCompiledIRAndPersistedForms(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	data, err := os.ReadFile("../../../apiserver/template/testdata/hello_world.yaml")
	require.NoError(t, err)
	tmpl, err := template.New([]byte(strings.ReplaceAll(string(data), "namespace/n1/pipeline/hello-world", "hello-world")), template.TemplateOptions{})
	require.NoError(t, err)
	schedule, err := tmpl.ScheduledWorkflow(&model.Job{
		UUID: "recurring-run", Namespace: "ns1", ServiceAccount: "pipeline-runner", Enabled: true,
		PipelineSpec: model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{Parameters: `{"y":"[[ScheduledTime]]-[[CurrentTime]]-[[Index]]"}`}},
	})
	require.NoError(t, err)
	execution, err := commonutil.ScheduleSpecToExecutionSpec(commonutil.ArgoWorkflow, schedule.Spec.Workflow)
	require.NoError(t, err)
	workflow := execution.(*commonutil.Workflow)
	wrapped, err := json.Marshal(workflow.Workflow)
	require.NoError(t, err)
	bare, err := json.Marshal(workflow.Spec)
	require.NoError(t, err)
	var bareObject map[string]interface{}
	require.NoError(t, json.Unmarshal(bare, &bareObject))
	// These are the persisted forms understood by ScheduleSpecToExecutionSpec.
	for _, spec := range []interface{}{string(wrapped), string(bare), bareObject} {
		schedule.Spec.Workflow.Spec = spec
		swf := util.NewScheduledWorkflow(schedule.DeepCopy())
		require.False(t, hasUnsupportedWorkflowTemplate(swf))
		original, err := commonutil.ScheduleSpecToExecutionSpec(commonutil.ArgoWorkflow, schedule.Spec.Workflow)
		require.NoError(t, err)
		next, err := swf.NewWorkflow(10, 20)
		require.NoError(t, err)
		require.Equal(t, "pipeline-runner", next.ServiceAccount())
		// Embedded IR runtime parameters are not workflow-level arguments.
		// Their macro binding is a pre-existing limitation; preserve them here.
		scheduled := next.(*commonutil.Workflow)
		require.Equal(t, workflow.GetTemplateByName("entrypoint").DAG.Tasks[0].Arguments.Parameters,
			scheduled.GetTemplateByName("entrypoint").DAG.Tasks[0].Arguments.Parameters)
		require.Equal(t, original.ExecutionNamespace(), next.ExecutionNamespace())
	}

	// Historical v2-compatible templates are not IR-compiled workflows.
	for _, spec := range []string{
		`{"entrypoint":"main"}`,
		`{"metadata":{"annotations":{"pipelines.kubeflow.org/v2_pipeline":"true"}},"spec":{"entrypoint":"main"}}`,
		`not JSON`,
	} {
		schedule.Spec.Workflow.Spec = spec
		require.True(t, hasUnsupportedWorkflowTemplate(util.NewScheduledWorkflow(schedule)))
	}
	// Pipeline-reference schedules still use CreateRun instead of an embedded workflow.
	schedule.Spec.Workflow.Spec = nil
	require.False(t, hasUnsupportedWorkflowTemplate(util.NewScheduledWorkflow(schedule)))
}
