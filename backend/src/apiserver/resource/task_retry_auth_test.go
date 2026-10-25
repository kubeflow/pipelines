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

package resource

import (
	"context"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

type taskRetryPodTracker struct {
	client.KubernetesCoreInterface
	deletes int
}

func (c *taskRetryPodTracker) PodClient(ns string) coreclient.PodInterface {
	return &taskRetryPods{PodInterface: c.KubernetesCoreInterface.PodClient(ns), owner: c}
}

type taskRetryPods struct {
	coreclient.PodInterface
	owner *taskRetryPodTracker
}

func (p *taskRetryPods) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	p.owner.deletes++
	return p.PodInterface.Delete(ctx, name, opts)
}

func TestRetryTaskCredentials(t *testing.T) {
	for _, name := range []string{"current", "old", "inline runtime", "default patch", "saved launcher mount", "script runtime", "container set runtime"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(argocompiler.DriverImageEnvVar, "driver:patched")
			t.Setenv(argocompiler.LauncherImageEnvVar, "launcher:patched")
			store, manager, run := initWithOneTimeRunV2(t)
			defer store.Close()
			execution, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.PipelineRuntimeManifest))
			require.NoError(t, err)
			wf := execution.(*util.Workflow)
			if name == "old" {
				for i := range wf.Spec.Templates {
					for j := range wf.Spec.Templates[i].InitContainers {
						if wf.Spec.Templates[i].InitContainers[j].Name == "kfp-launcher" {
							wf.Spec.Templates[i].InitContainers[j].Image = "launcher:old"
						}
					}
				}
			}
			wf.Status.Phase = wfapi.WorkflowFailed
			wf.Status.Nodes = wfapi.Nodes{"complete": wfapi.NodeStatus{ID: "complete", Name: "complete", Type: wfapi.NodeTypePod, Phase: wfapi.NodeSucceeded, Outputs: &wfapi.Outputs{Parameters: []wfapi.Parameter{{Name: "execution-id", Value: wfapi.AnyStringPtr("42")}}}}}
			wf.Status.Nodes["failed"] = wfapi.NodeStatus{ID: "failed", Name: "failed", Type: wfapi.NodeTypePod, Phase: wfapi.NodeFailed}
			switch name {
			case "script runtime", "container set runtime":
				for i := range wf.Spec.Templates {
					tmpl := &wf.Spec.Templates[i]
					if tmpl.Container == nil || tmpl.Metadata.Annotations[util.AnnotationKeyRuntimeRole] != "driver" {
						continue
					}
					if name == "script runtime" {
						tmpl.Script = &wfapi.ScriptTemplate{Container: *tmpl.Container.DeepCopy(), Source: "echo unused"}
					} else {
						tmpl.ContainerSet = &wfapi.ContainerSetTemplate{}
					}
					tmpl.Container = nil
				}
			case "inline runtime":
				wf.Spec.Templates = append(wf.Spec.Templates, wfapi.Template{Name: "inline-runtime", DAG: &wfapi.DAGTemplate{Tasks: []wfapi.DAGTask{{Name: "runtime", Inline: wf.Spec.Templates[0].DeepCopy()}}}})
			case "default patch":
				wf.Spec.TemplateDefaults = &wfapi.Template{PodSpecPatch: `{"containers":[{"name":"main","env":[{"name":"KFP_TASK_API_TOKEN_PATH","value":"/missing/token"}]}]}`}
				wf.Spec.PodSpecPatch = `{"containers":[{"name":"main","env":[{"name":"KFP_TASK_API_TOKEN_PATH","value":"/kfp-task-api/token"}]}]}`
			case "saved launcher mount":
				wf.Status.Nodes["complete"].Outputs.Parameters = append(wf.Status.Nodes["complete"].Outputs.Parameters, wfapi.Parameter{Name: "pod-spec-patch", Value: wfapi.AnyStringPtr(`{"containers":[{"name":"main","volumeMounts":[{"mountPath":"/kfp-launcher","$patch":"delete"}]}]}`)})
			}
			run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
			run.State = model.RuntimeStateFailed
			run.Conditions = string(model.RuntimeStateFailed.ToV1())
			require.NoError(t, manager.runStore.UpdateRun(run))
			tracker := &taskRetryPodTracker{KubernetesCoreInterface: manager.k8sCoreClient}
			manager.k8sCoreClient = tracker
			before, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			viper.Set(common.MultiUserMode, true)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
			err = manager.RetryRun(multiUserContext(), run.UUID)
			after, getErr := manager.GetRun(run.UUID)
			require.NoError(t, getErr)
			if name != "current" {
				require.ErrorContains(t, err, "new run")
				require.Equal(t, codes.FailedPrecondition, status.Code(err))
				require.Equal(t, before, after)
				require.Zero(t, tracker.deletes)
			} else {
				require.NoError(t, err)
				updated, err := store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				got := updated.(*util.Workflow)
				require.NoError(t, got.ValidateTaskRuntimeCredentials(common.GetTaskTokenReviewAudience()))
				require.Equal(t, wf.Status.Nodes["complete"], got.Status.Nodes["complete"])
				require.Equal(t, wf.Spec.ServiceAccountName, got.Spec.ServiceAccountName)
				require.Equal(t, wf.Name, got.Name)
				for i, tmpl := range wf.Spec.Templates {
					if tmpl.Container != nil {
						require.Equal(t, tmpl.Container.Image, got.Spec.Templates[i].Container.Image)
						require.Equal(t, tmpl.Container.Command, got.Spec.Templates[i].Container.Command)
					}
				}
			}
		})
	}
}
