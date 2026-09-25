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

package template_test

import (
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestArgoFreshWorkflowsDiscardSubmittedStatus(t *testing.T) {
	wf := &workflowapi.Workflow{
		TypeMeta:   metav1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: metav1.ObjectMeta{Name: "fresh"},
		Spec: workflowapi.WorkflowSpec{Entrypoint: "main", Templates: []workflowapi.Template{{
			Name: "main", Container: &corev1.Container{Image: "alpine"},
		}}},
		Status: workflowapi.WorkflowStatus{
			Phase:              workflowapi.WorkflowRunning,
			StoredTemplates:    map[string]workflowapi.Template{"stored": {Name: "stored", ServiceAccountName: "unauthorized"}},
			StoredWorkflowSpec: &workflowapi.WorkflowSpec{ServiceAccountName: "unauthorized"},
		},
	}
	original := wf.DeepCopy()
	data, err := yaml.Marshal(wf)
	require.NoError(t, err)
	validated, err := template.ValidateWorkflow(data)
	require.NoError(t, err)
	assert.Empty(t, validated.Status)

	// The direct constructor bypasses manifest validation; both execution paths
	// must still discard status without changing the reusable source template.
	tmpl, err := template.NewArgoTemplateFromWorkflow(wf)
	require.NoError(t, err)
	run, err := tmpl.RunWorkflow(&model.Run{}, template.RunWorkflowOptions{RunID: "fresh-run"})
	require.NoError(t, err)
	assert.Empty(t, run.(*util.Workflow).Status)
	job, err := tmpl.ScheduledWorkflow(&model.Job{UUID: "fresh-job"})
	require.NoError(t, err)
	require.IsType(t, "", job.Spec.Workflow.Spec)
	scheduled, err := util.NewWorkflowFromScheduleWorkflowSpecBytesJSON([]byte(job.Spec.Workflow.Spec.(string)))
	require.NoError(t, err)
	assert.Empty(t, scheduled.Status)
	assert.Equal(t, original, wf)
}
