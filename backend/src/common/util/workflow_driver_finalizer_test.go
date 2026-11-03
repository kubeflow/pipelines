// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateRetryExecutionReplaysDriverFinalizers(t *testing.T) {
	for _, podNameFormat := range []string{"v1", "v2"} {
		for _, reference := range []string{"local", "namespaced-ref", "cluster-ref", "stored-scope"} {
			t.Run(podNameFormat+"/"+reference, func(t *testing.T) {
				finalizer := wfapi.Template{Name: "stop-driver", Metadata: wfapi.Metadata{Annotations: map[string]string{AnnotationKeyDriverRetryFinalizer: "true"}}}
				workflow := NewWorkflow(&wfapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: "wf", Labels: map[string]string{}, Annotations: map[string]string{
						"workflows.argoproj.io/pod-name-format": podNameFormat, AnnotationKeyRetryGeneration: "0",
					}},
					Spec: wfapi.WorkflowSpec{Templates: []wfapi.Template{
						finalizer,
						{Name: "driver", Metadata: wfapi.Metadata{Annotations: map[string]string{AnnotationKeyTaskDriverRetry: "true"}}},
						{Name: "user-hook"},
					}},
					Status: wfapi.WorkflowStatus{Phase: wfapi.WorkflowFailed, Nodes: wfapi.Nodes{
						"root": {ID: "root", Name: "wf", Type: wfapi.NodeTypeDAG, Phase: wfapi.NodeFailed,
							Children: []string{"retry", "hook-retry", "succeeded", "user-hook"}, OutboundNodes: []string{"hook-pod", "succeeded"}},
						"retry":      {ID: "retry", Name: "wf.task-driver", Type: wfapi.NodeTypeRetry, Phase: wfapi.NodeFailed, TemplateName: "driver", Children: []string{"attempt", "hook-retry"}},
						"attempt":    {ID: "attempt", Name: "wf.task-driver(0)", Type: wfapi.NodeTypePod, Phase: wfapi.NodeFailed, TemplateName: "driver"},
						"hook-retry": {ID: "hook-retry", Name: "wf.task-driver.onExit", Type: wfapi.NodeTypeRetry, Phase: wfapi.NodeSucceeded, TemplateName: finalizer.Name, Children: []string{"hook-pod"}},
						"hook-pod":   {ID: "hook-pod", Name: "wf.task-driver.onExit(0)", Type: wfapi.NodeTypePod, Phase: wfapi.NodeSucceeded, TemplateName: finalizer.Name},
						"succeeded":  {ID: "succeeded", Name: "wf.other-driver", Type: wfapi.NodeTypePod, Phase: wfapi.NodeSucceeded, TemplateName: "driver"},
						"user-hook":  {ID: "user-hook", Name: "wf.other.onExit", Type: wfapi.NodeTypePod, Phase: wfapi.NodeSucceeded, TemplateName: "user-hook"},
					}},
				})
				if reference != "local" {
					// A local template with the same name must not shadow the node's reference.
					workflow.Spec.Templates[0].Metadata.Annotations = nil
					for _, id := range []string{"hook-retry", "hook-pod"} {
						node := workflow.Status.Nodes[id]
						if reference == "stored-scope" {
							node.TemplateScope = "namespaced/library"
						} else {
							node.TemplateRef = &wfapi.TemplateRef{Name: "library", Template: finalizer.Name, ClusterScope: reference == "cluster-ref"}
							node.TemplateName = ""
						}
						workflow.Status.Nodes[id] = node
						scope, resource := node.GetTemplateScope()
						_, err := workflow.SetStoredTemplate(scope, resource, &node, &finalizer)
						require.NoError(t, err)
					}
				}
				before := workflow.DeepCopy()
				retry, pods, err := workflow.GenerateRetryExecution()
				require.NoError(t, err)
				retried := retry.(*Workflow)
				expectedPods := []string{"attempt", "hook-pod"}
				if podNameFormat == "v2" {
					expectedPods = []string{"wf-driver-attempt", "wf-stop-driver-pod"}
				}
				assert.ElementsMatch(t, expectedPods, pods)
				for _, id := range []string{"retry", "attempt", "hook-retry", "hook-pod"} {
					assert.NotContains(t, retried.Status.Nodes, id)
				}
				for _, id := range []string{"succeeded", "user-hook"} {
					assert.Equal(t, workflow.Status.Nodes[id], retried.Status.Nodes[id])
				}
				assert.Equal(t, wfapi.NodeRunning, retried.Status.Nodes["root"].Phase)
				assert.ElementsMatch(t, []string{"succeeded", "user-hook"}, retried.Status.Nodes["root"].Children)
				assert.Equal(t, []string{"succeeded"}, retried.Status.Nodes["root"].OutboundNodes)
				assert.Equal(t, workflow.Spec, retried.Spec, "manual retry must keep the finalizer hook templates")
				assert.Equal(t, before, workflow.Workflow, "retry preparation must not mutate the existing workflow")
			})
		}
	}
}
