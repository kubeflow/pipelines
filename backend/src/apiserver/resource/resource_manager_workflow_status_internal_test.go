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

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	apiservermlflow "github.com/kubeflow/pipelines/backend/src/apiserver/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func statusWithCachedServiceAccount(storedSpec bool) v1alpha1.WorkflowStatus {
	tmpl := *testWorkflow.Spec.Templates[0].DeepCopy()
	tmpl.Name = "cached-step"
	status := v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowSucceeded}
	if storedSpec {
		status.StoredWorkflowSpec = &v1alpha1.WorkflowSpec{
			ServiceAccountName: "cached-sa",
			Templates:          []v1alpha1.Template{tmpl},
		}
	} else {
		tmpl.ServiceAccountName = "cached-sa"
		status.StoredTemplates = map[string]v1alpha1.Template{"cached-step": tmpl}
	}
	return status
}

func TestCreateRunAndJob_DiscardCallerWorkflowStatus(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	viper.Set(common.AllowedServiceAccountsFlag, "cached-sa")
	t.Cleanup(func() { viper.Set(common.AllowedServiceAccountsFlag, "") })

	for _, storedSpec := range []bool{false, true} {
		name := "stored templates"
		if storedSpec {
			name = "stored workflow spec"
		}
		t.Run(name, func(t *testing.T) {
			for _, recurring := range []bool{false, true} {
				kind := "run"
				if recurring {
					kind = "recurring run"
				}
				t.Run(kind, func(t *testing.T) {
					store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
					defer store.Close()
					workflow := testWorkflow.DeepCopy()
					workflow.Status = statusWithCachedServiceAccount(storedSpec)
					pipelineSpec := model.PipelineSpec{
						WorkflowSpecManifest: model.LargeText(util.NewWorkflow(workflow).ToStringForStore()),
						Parameters:           `[{"name":"param1","value":"world"}]`,
					}
					var executionSpec util.ExecutionSpec
					if recurring {
						job, err := manager.CreateJob(multiUserContext(), &model.Job{
							DisplayName: "j1", Enabled: true, ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
						})
						require.NoError(t, err)
						schedule, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(context.Background(), job.K8SName, metav1.GetOptions{})
						require.NoError(t, err)
						executionSpec, err = util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, schedule.Spec.Workflow)
						require.NoError(t, err)
					} else {
						run, err := manager.CreateRun(multiUserContext(), &model.Run{
							DisplayName: "run1", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
						})
						require.NoError(t, err)
						executionSpec, err = store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
						require.NoError(t, err)
					}
					assert.Equal(t, v1alpha1.WorkflowStatus{}, executionSpec.(*util.Workflow).Status)
					assert.Equal(t, "testy", executionSpec.(*util.Workflow).Spec.Entrypoint)
				})
			}
		})
	}
}

func TestCreateRun_RejectsStatusOnlyEntrypoint(t *testing.T) {
	for _, storedSpec := range []bool{false, true} {
		name := "stored templates"
		if storedSpec {
			name = "stored workflow spec"
		}
		t.Run(name, func(t *testing.T) {
			store, manager, experiment := initWithExperiment(t)
			defer store.Close()
			workflow := testWorkflow.DeepCopy()
			workflow.Spec.Entrypoint = "cached-step"
			workflow.Status = statusWithCachedServiceAccount(storedSpec)
			run := &model.Run{
				DisplayName: "run1", ExperimentId: experiment.UUID,
				PipelineSpec: model.PipelineSpec{
					WorkflowSpecManifest: model.LargeText(util.NewWorkflow(workflow).ToStringForStore()),
					Parameters:           `[{"name":"param1","value":"world"}]`,
				},
			}
			_, err := manager.CreateRun(context.Background(), run)
			require.ErrorContains(t, err, "cached-step")
			assert.Zero(t, store.ExecClientFake.GetWorkflowCount())
			_, err = manager.GetRun(run.UUID)
			assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
		})
	}
}

func TestRetryRun_AuthorizesAndPreservesCachedWorkflowTemplates(t *testing.T) {
	viper.Set(common.AllowedServiceAccountsFlag, "cached-sa")
	t.Cleanup(func() { viper.Set(common.AllowedServiceAccountsFlag, "") })
	for _, storedSpec := range []bool{false, true} {
		name := "stored templates"
		if storedSpec {
			name = "stored workflow spec"
		}
		t.Run(name, func(t *testing.T) {
			for _, authorized := range []bool{false, true} {
				verdict := "denied"
				if authorized {
					verdict = "authorized"
				}
				t.Run(verdict, func(t *testing.T) {
					store, manager, run := initWithOneTimeFailedRun(t)
					defer store.Close()
					run, err := manager.GetRun(run.UUID)
					require.NoError(t, err)
					workflow, err := util.NewWorkflowFromBytesJSON([]byte(run.WorkflowRuntimeManifest))
					require.NoError(t, err)
					require.NoError(t, workflow.Decompress())
					cached := statusWithCachedServiceAccount(storedSpec)
					workflow.Status.StoredTemplates = cached.StoredTemplates
					workflow.Status.StoredWorkflowSpec = cached.StoredWorkflowSpec
					syncWorkflowReportWithFakeCluster(t, store, workflow)
					run.WorkflowRuntimeManifest = model.LargeText(workflow.ToStringForStore())
					require.NoError(t, manager.runStore.UpdateRun(run))
					before, err := manager.GetRun(run.UUID)
					require.NoError(t, err)
					liveBefore, err := store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
					require.NoError(t, err)
					beforeManifest := liveBefore.ToStringForStore()

					viper.Set(common.MultiUserMode, "true")
					t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
					if authorized {
						manager.subjectAccessReviewClient = client.NewFakeSubjectAccessReviewClient()
					} else {
						manager.subjectAccessReviewClient = client.NewFakeSubjectAccessReviewClientUnauthorized()
						manager.k8sCoreClient = client.NewFakeKubernetesCoreClientWithBadPodClient()
					}
					err = manager.RetryRun(multiUserContext(), run.UUID)
					if authorized {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "Unauthorized")
					}
					after, err := manager.GetRun(run.UUID)
					require.NoError(t, err)
					liveAfter, err := store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
					require.NoError(t, err)
					if !authorized {
						assert.Equal(t, before, after, "authorization must precede the retry claim")
						assert.Equal(t, beforeManifest, liveAfter.ToStringForStore())
						return
					}
					assert.Equal(t, model.RuntimeStateRunning, after.State)
					assert.Equal(t, v1alpha1.WorkflowRunning, liveAfter.(*util.Workflow).Status.Phase)
					assert.Equal(t, cached.StoredTemplates, liveAfter.(*util.Workflow).Status.StoredTemplates)
					assert.Equal(t, cached.StoredWorkflowSpec, liveAfter.(*util.Workflow).Status.StoredWorkflowSpec)
				})
			}
		})
	}
}

type cachedTemplateMutatingDispatcher struct {
	serviceAccountMutatingDispatcher
	storedSpec bool
}

func (d *cachedTemplateMutatingDispatcher) OnBeforeRunCreation(_ context.Context, run *apiserverPlugins.PendingRun, executionSpec util.ExecutionSpec) error {
	executionSpec.(*util.Workflow).Status = statusWithCachedServiceAccount(d.storedSpec)
	return apiserverPlugins.SetPendingRunPluginOutput(run, apiservermlflow.PluginName, d.output)
}

func TestCreateRun_PluginCachedTemplateUnauthorizedCleansUp(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	viper.Set(common.AllowedServiceAccountsFlag, "cached-sa")
	t.Cleanup(func() { viper.Set(common.AllowedServiceAccountsFlag, "") })
	for _, storedSpec := range []bool{false, true} {
		name := "stored templates"
		if storedSpec {
			name = "stored workflow spec"
		}
		t.Run(name, func(t *testing.T) {
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			dispatcher := &cachedTemplateMutatingDispatcher{
				serviceAccountMutatingDispatcher: serviceAccountMutatingDispatcher{
					output: apiservermlflow.SuccessfulPluginOutput("exp-1", "experiment", "parent-run-1", "https://mlflow.example/runs/parent-run-1"),
				},
				storedSpec: storedSpec,
			}
			manager.pluginDispatcher = dispatcher
			run := &model.Run{
				DisplayName: "run1", ExperimentId: experiment.UUID,
				PipelineSpec: model.PipelineSpec{
					WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()),
					Parameters:           `[{"name":"param1","value":"world"}]`,
				},
			}
			_, err := manager.CreateRun(multiUserContext(), run)
			require.ErrorContains(t, err, "Unauthorized")
			assert.Zero(t, store.ExecClientFake.GetWorkflowCount())
			_, err = manager.GetRun(run.UUID)
			assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
			require.Len(t, dispatcher.endedRuns, 1)
			assert.Equal(t, run.UUID, dispatcher.endedRuns[0].RunID)
			assert.True(t, proto.Equal(dispatcher.output, dispatcher.endedRuns[0].PluginsOutput[apiservermlflow.PluginName]))
		})
	}
}
