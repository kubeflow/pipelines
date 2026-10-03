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

package util_test

import (
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkflowServiceAccountsIncludesStoredExecutionState(t *testing.T) {
	w := util.NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{ServiceAccountName: "submitted-sa"},
		Status: workflowapi.WorkflowStatus{
			StoredWorkflowSpec: &workflowapi.WorkflowSpec{
				ServiceAccountName: "stored-workflow-sa",
				Executor:           &workflowapi.ExecutorConfig{ServiceAccountName: "stored-executor-sa"},
				PodSpecPatch:       `serviceAccountName: stored-patch-sa`,
				ExecutorPlugins: []workflowapi.ExecutorPlugin{{
					ObjectMeta: metav1.ObjectMeta{Name: "stored"},
					Spec:       workflowapi.ExecutorPluginSpec{Sidecar: workflowapi.ExecutorPluginSidecar{AutomountServiceAccountToken: true}},
				}},
				TemplateDefaults: &workflowapi.Template{
					ServiceAccountName: "{{workflow.serviceAccountName}}",
					Executor:           &workflowapi.ExecutorConfig{ServiceAccountName: "stored-default-executor-sa"},
				},
				Templates: []workflowapi.Template{{Name: "stored", ServiceAccountName: "stored-template-sa"}},
				ArtifactGC: &workflowapi.WorkflowLevelArtifactGC{
					ArtifactGC: workflowapi.ArtifactGC{Strategy: workflowapi.ArtifactGCOnWorkflowCompletion, ServiceAccountName: "stored-gc-sa"},
				},
			},
			StoredTemplates: map[string]workflowapi.Template{
				"local/mapped": {
					Name: "mapped", ServiceAccountName: "mapped-sa",
					Steps: []workflowapi.ParallelSteps{{Steps: []workflowapi.WorkflowStep{{
						Name: "inline", Inline: &workflowapi.Template{ServiceAccountName: "inline-sa"},
					}}}},
					Outputs: workflowapi.Outputs{Artifacts: workflowapi.Artifacts{{Name: "cached-output"}}},
				},
			},
			Nodes: workflowapi.Nodes{
				"succeeded": {Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeSucceeded, Outputs: &workflowapi.Outputs{
					Artifacts: workflowapi.Artifacts{
						{Name: "inherited-gc"},
						{Name: "override-gc", ArtifactGC: &workflowapi.ArtifactGC{Strategy: workflowapi.ArtifactGCOnWorkflowDeletion, ServiceAccountName: "node-gc-sa"}},
						{Name: "inactive", ArtifactGC: &workflowapi.ArtifactGC{Strategy: workflowapi.ArtifactGCNever, ServiceAccountName: "inactive-sa"}},
					},
				}},
			},
		},
	})
	original := w.DeepCopy()
	accounts, err := w.ServiceAccounts(false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"submitted-sa", "stored-workflow-sa", "stored-executor-sa", "stored-patch-sa", "stored-executor-plugin",
		"stored-default-executor-sa", "stored-template-sa", "mapped-sa", "inline-sa", "stored-gc-sa", "node-gc-sa",
	}, accounts)
	assert.Equal(t, original, w.Workflow, "authorization must not discard retry state")
}

func TestWorkflowServiceAccountsRejectsUnsafeStoredTemplates(t *testing.T) {
	for _, location := range []string{"stored templates", "stored workflow templates", "stored defaults"} {
		t.Run(location, func(t *testing.T) {
			for _, kind := range []string{"external reference", "dynamic patch"} {
				t.Run(kind, func(t *testing.T) {
					tmpl := workflowapi.Template{Name: "stored"}
					wantError := "external workflow template references"
					if kind == "external reference" {
						tmpl.DAG = &workflowapi.DAGTemplate{Tasks: []workflowapi.DAGTask{{TemplateRef: &workflowapi.TemplateRef{Name: "external", Template: "task"}}}}
					} else {
						tmpl.PodSpecPatch = `serviceAccountName: "{{workflow.parameters.account}}"`
						wantError = "podSpecPatch contains a template expression"
					}
					w := util.NewWorkflow(&workflowapi.Workflow{})
					switch location {
					case "stored templates":
						w.Status.StoredTemplates = map[string]workflowapi.Template{"stored": tmpl}
					case "stored workflow templates":
						w.Status.StoredWorkflowSpec = &workflowapi.WorkflowSpec{Templates: []workflowapi.Template{tmpl}}
					case "stored defaults":
						w.Status.StoredWorkflowSpec = &workflowapi.WorkflowSpec{TemplateDefaults: &tmpl}
					}
					_, err := w.ServiceAccounts(false)
					require.ErrorContains(t, err, wantError)
				})
			}
		})
	}
}

func TestWorkflowServiceAccountsChecksStoredNodeArtifactGCDefaults(t *testing.T) {
	w := util.NewWorkflow(&workflowapi.Workflow{Status: workflowapi.WorkflowStatus{
		StoredWorkflowSpec: &workflowapi.WorkflowSpec{ArtifactGC: &workflowapi.WorkflowLevelArtifactGC{
			ArtifactGC:   workflowapi.ArtifactGC{Strategy: workflowapi.ArtifactGCOnWorkflowCompletion},
			PodSpecPatch: `serviceAccountName: cached-gc-sa`,
		}},
		Nodes: workflowapi.Nodes{"retained": {Type: workflowapi.NodeTypePod, Outputs: &workflowapi.Outputs{
			Artifacts: workflowapi.Artifacts{{Name: "output"}},
		}}},
	}})
	accounts, err := w.ServiceAccounts(false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"default", "cached-gc-sa"}, accounts)
	w.Status.StoredWorkflowSpec.ArtifactGC.PodSpecPatch = `{{workflow.parameters.gc-patch}}`
	_, err = w.ServiceAccounts(false)
	require.ErrorContains(t, err, "podSpecPatch contains a template expression")
}
