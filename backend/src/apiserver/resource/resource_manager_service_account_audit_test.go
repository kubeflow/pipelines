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

package resource

import (
	"context"
	"flag"
	"os"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	plugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/apiserver/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func configureServiceAccountAuditTest(t *testing.T, audit bool) {
	t.Helper()
	viper.Set(common.WorkflowServiceAccountAudit, audit)
	viper.Set(common.MultiUserMode, "true")
	viper.Set(common.AllowedServiceAccountsFlag, "nested-sa,another-sa")
	t.Cleanup(func() {
		viper.Set(common.WorkflowServiceAccountAudit, nil)
		viper.Set(common.MultiUserMode, "false")
		viper.Set(common.AllowedServiceAccountsFlag, "")
	})
}

// These tests must remain sequential: glog, stderr and Viper are process-global.
func captureServiceAccountAuditLogs(t *testing.T, action func()) string {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "audit-log")
	require.NoError(t, err)
	defer output.Close()
	originalStderr := os.Stderr
	originalLogToStderr := flag.Lookup("logtostderr").Value.String()
	require.NoError(t, flag.Set("logtostderr", "true"))
	os.Stderr = output
	defer func() {
		os.Stderr = originalStderr
		_ = flag.Set("logtostderr", originalLogToStderr)
	}()
	action()
	data, err := os.ReadFile(output.Name())
	require.NoError(t, err)
	return string(data)
}

func TestWorkflowServiceAccountAuditChecks(t *testing.T) {
	for _, test := range []struct {
		name, finding string
		modify        func(*util.Workflow, *ResourceManager)
	}{
		{name: "SAR denial", finding: "account_denied"},
		{name: "allow list denial", finding: "account_not_allowed", modify: func(_ *util.Workflow, _ *ResourceManager) {
			viper.Set(common.AllowedServiceAccountsFlag, "")
		}},
		{name: "SAR unavailable", finding: "authorization_error", modify: func(_ *util.Workflow, manager *ResourceManager) {
			manager.subjectAccessReviewClient = client.NewFakeSubjectAccessReviewClientError()
		}},
		{name: "dynamic patch", finding: "inspection_incomplete", modify: func(workflow *util.Workflow, _ *ResourceManager) {
			workflow.Spec.PodSpecPatch = `{{workflow.parameters.private-patch-value}}`
		}},
		{name: "malformed patch", finding: "inspection_incomplete", modify: func(workflow *util.Workflow, _ *ResourceManager) {
			workflow.Spec.PodSpecPatch = `{"containers":"private-patch-value"}`
		}},
		{name: "retained external reference", finding: "inspection_incomplete", modify: func(workflow *util.Workflow, _ *ResourceManager) {
			workflow.Status.StoredWorkflowSpec = &workflowapi.WorkflowSpec{WorkflowTemplateRef: &workflowapi.WorkflowTemplateRef{Name: "external"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, audit := range []bool{false, true} {
				mode := "enforce"
				if audit {
					mode = "audit"
				}
				t.Run(mode, func(t *testing.T) {
					configureServiceAccountAuditTest(t, audit)
					store, manager, _ := initWithExperimentAndUnauthorizedSAR(t)
					defer store.Close()
					workflow := util.NewWorkflow(testWorkflow.DeepCopy())
					workflow.Spec.ServiceAccountName = common.DefaultPipelineRunnerServiceAccount
					workflow.Spec.Templates[0].ServiceAccountName = "nested-sa"
					workflow.Spec.TemplateDefaults = &workflowapi.Template{ServiceAccountName: "another-sa"}
					if test.modify != nil {
						test.modify(workflow, manager)
					}
					logs := captureServiceAccountAuditLogs(t, func() {
						err := manager.authorizeExecutionServiceAccounts(multiUserContext(), workflow, false, "ns1", "create_run")
						if audit {
							require.NoError(t, err)
						} else {
							require.Error(t, err)
						}
					})
					if audit {
						assert.Contains(t, logs, `operation="create_run" namespace="ns1" workflow="workflow-name"`)
						assert.Contains(t, logs, `finding="`+test.finding+`"`)
						assert.Contains(t, logs, common.WorkflowServiceAccountAudit+"=true")
						if test.finding != "inspection_incomplete" {
							assert.Contains(t, logs, `service_account="nested-sa"`)
							assert.Contains(t, logs, `service_account="another-sa"`)
						}
					} else {
						assert.NotContains(t, logs, "Workflow service account audit:")
					}
					assert.NotContains(t, logs, "private-patch-value")
				})
			}
		})
	}
}

func TestWorkflowServiceAccountAuditStillEnforcesMainAccount(t *testing.T) {
	for _, allowList := range []string{"", "main-sa"} {
		t.Run("allowed="+allowList, func(t *testing.T) {
			configureServiceAccountAuditTest(t, true)
			viper.Set(common.AllowedServiceAccountsFlag, allowList)
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			workflow := util.NewWorkflow(testWorkflow.DeepCopy())
			workflow.Spec.ServiceAccountName = "main-sa"
			workflow.Spec.PodSpecPatch = `{"serviceAccountName":"{{workflow.parameters.param1}}"}`
			_, err := manager.CreateRun(multiUserContext(), &model.Run{
				DisplayName: "audit", ExperimentId: experiment.UUID,
				PipelineSpec: model.PipelineSpec{
					WorkflowSpecManifest: model.LargeText(workflow.ToStringForStore()),
					Parameters:           `[{"name":"param1","value":"pipeline-runner"}]`,
				},
			})
			require.Error(t, err)
			assert.Zero(t, store.ExecClientFake.GetWorkflowCount())
		})
	}
}

func TestWorkflowServiceAccountAuditFreshWorkflows(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		for _, externalReference := range []bool{false, true} {
			name := version + "/valid"
			if externalReference {
				name = version + "/external reference"
			}
			t.Run(name, func(t *testing.T) {
				configureServiceAccountAuditTest(t, true)
				store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
				defer store.Close()
				workflow := util.NewWorkflow(testWorkflow.DeepCopy())
				workflow.Status.StoredTemplates = map[string]workflowapi.Template{"untrusted": {Name: "untrusted", ServiceAccountName: "nested-sa"}}
				if externalReference {
					workflow.Spec.WorkflowTemplateRef = &workflowapi.WorkflowTemplateRef{Name: "external"}
				}
				pipelineSpec := model.PipelineSpec{
					WorkflowSpecManifest: model.LargeText(workflow.ToStringForStore()),
					Parameters:           `[{"name":"param1","value":"world"}]`,
				}
				if version == "v2" {
					pipelineSpec = model.PipelineSpec{
						PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
						RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
					}
					if externalReference {
						viper.Set(common.CompiledPipelineSpecPatch, `{"workflowTemplateRef":{"name":"external"}}`)
						t.Cleanup(func() { viper.Set(common.CompiledPipelineSpecPatch, "") })
					}
				}
				logs := captureServiceAccountAuditLogs(t, func() {
					run, err := manager.CreateRun(multiUserContext(), &model.Run{DisplayName: "audit", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec})
					if externalReference {
						require.ErrorContains(t, err, "external workflow template references")
						assert.Zero(t, store.ExecClientFake.GetWorkflowCount())
						return
					}
					require.NoError(t, err)
					created, err := store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
					require.NoError(t, err)
					assert.Empty(t, created.(*util.Workflow).Status.StoredTemplates)
					if version == "v2" {
						assert.Contains(t, created.ToStringForStore(), "{{inputs.parameters.pod-spec-patch}}")
					}
				})
				assert.NotContains(t, logs, "Workflow service account audit:")
			})
		}
	}
}

func TestWorkflowServiceAccountAuditEnforcesImplicitMainAccount(t *testing.T) {
	for _, allowList := range []string{"", "default"} {
		t.Run("allowed="+allowList, func(t *testing.T) {
			configureServiceAccountAuditTest(t, true)
			viper.Set(common.AllowedServiceAccountsFlag, allowList)
			store, manager, _ := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			workflow := util.NewWorkflow(testWorkflow.DeepCopy())
			// Kubernetes uses default when a retained workflow or plugin omits
			// the main account. An inspection failure must not bypass its policy.
			workflow.Spec.ServiceAccountName = ""
			workflow.Spec.PodSpecPatch = `{{workflow.parameters.param1}}`
			logs := captureServiceAccountAuditLogs(t, func() {
				err := manager.authorizeExecutionServiceAccounts(multiUserContext(), workflow, false, "ns1", "retry_run")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "default")
			})
			assert.NotContains(t, logs, "Workflow service account audit:")
		})
	}
}

func TestWorkflowServiceAccountAuditCreateRunAndJob(t *testing.T) {
	for _, kind := range []string{"run", "dynamic run", "job", "job with plugins", "latest job"} {
		t.Run(kind, func(t *testing.T) {
			configureServiceAccountAuditTest(t, true)
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			pipelineSpec := model.PipelineSpec{WorkflowSpecManifest: workflowManifestWithTemplateServiceAccount("nested-sa")}
			if kind == "dynamic run" {
				workflow := util.NewWorkflow(testWorkflow.DeepCopy())
				workflow.Spec.PodSpecPatch = `{"serviceAccountName":"{{workflow.parameters.param1}}"}`
				pipelineSpec.WorkflowSpecManifest = model.LargeText(workflow.ToStringForStore())
			}
			if kind == "job with plugins" {
				manager.pluginDispatcher = &countingTerminalReportDispatcher{}
			}
			if kind == "latest job" {
				pipeline, err := manager.CreatePipeline(createPipeline("p1", "", "ns1"))
				require.NoError(t, err)
				_, err = manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "v1", "v1", "", string(pipelineSpec.WorkflowSpecManifest), "", "ns1"))
				require.NoError(t, err)
				pipelineSpec = model.PipelineSpec{PipelineId: pipeline.UUID}
			}
			operation := "create_recurring_run"
			logs := captureServiceAccountAuditLogs(t, func() {
				if kind == "run" || kind == "dynamic run" {
					operation = "create_run"
					pipelineSpec.Parameters = `[{"name":"param1","value":"pipeline-runner"}]`
					_, err := manager.CreateRun(multiUserContext(), &model.Run{DisplayName: "audit", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec})
					require.NoError(t, err)
					assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
				} else {
					job, err := manager.CreateJob(multiUserContext(), &model.Job{DisplayName: "audit", Enabled: true, ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec})
					require.NoError(t, err)
					_, err = store.SwfClient().ScheduledWorkflow(job.Namespace).Get(context.Background(), job.K8SName, metav1.GetOptions{})
					require.NoError(t, err)
				}
			})
			assert.Contains(t, logs, `operation="`+operation+`"`)
		})
	}
}

type auditTemplateMutatingDispatcher struct {
	serviceAccountMutatingDispatcher
}

func (d *auditTemplateMutatingDispatcher) OnBeforeRunCreation(_ context.Context, run *plugins.PendingRun, executionSpec util.ExecutionSpec) error {
	executionSpec.(*util.Workflow).Spec.Templates[0].ServiceAccountName = "nested-sa"
	return plugins.SetPendingRunPluginOutput(run, mlflow.PluginName, d.output)
}

func TestWorkflowServiceAccountAuditPluginMutation(t *testing.T) {
	for _, mainAccount := range []bool{false, true} {
		name := "additional account"
		if mainAccount {
			name = "main account"
		}
		t.Run(name, func(t *testing.T) {
			configureServiceAccountAuditTest(t, true)
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			dispatcher := &auditTemplateMutatingDispatcher{serviceAccountMutatingDispatcher{output: mlflow.SuccessfulPluginOutput("exp", "experiment", "parent", "https://mlflow.example/runs/parent")}}
			manager.pluginDispatcher = dispatcher
			if mainAccount {
				manager.pluginDispatcher = &dispatcher.serviceAccountMutatingDispatcher
			}
			logs := captureServiceAccountAuditLogs(t, func() {
				_, err := manager.CreateRun(multiUserContext(), &model.Run{
					DisplayName: "audit", ExperimentId: experiment.UUID,
					PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()), Parameters: `[{"name":"param1","value":"world"}]`},
				})
				if mainAccount {
					require.Error(t, err)
					assert.Zero(t, store.ExecClientFake.GetWorkflowCount())
					require.Len(t, dispatcher.endedRuns, 1)
				} else {
					require.NoError(t, err)
					assert.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
					assert.Empty(t, dispatcher.endedRuns)
				}
			})
			if !mainAccount {
				assert.Contains(t, logs, `operation="create_run_after_plugins"`)
			}
		})
	}
}

func TestWorkflowServiceAccountAuditRetryAndEnable(t *testing.T) {
	t.Run("retry", func(t *testing.T) {
		store, manager, run := initWithOneTimeFailedRun(t)
		defer store.Close()
		run, err := manager.GetRun(run.UUID)
		require.NoError(t, err)
		workflow, err := util.NewWorkflowFromBytesJSON([]byte(run.WorkflowRuntimeManifest))
		require.NoError(t, err)
		require.NoError(t, workflow.Decompress())
		workflow.Status.StoredTemplates = map[string]workflowapi.Template{"cached": {Name: "cached", ServiceAccountName: "nested-sa"}}
		run.WorkflowRuntimeManifest = model.LargeText(workflow.ToStringForStore())
		require.NoError(t, manager.runStore.UpdateRun(run))
		syncWorkflowReportWithFakeCluster(t, store, workflow)
		configureServiceAccountAuditTest(t, true)
		manager.subjectAccessReviewClient = client.NewFakeSubjectAccessReviewClientUnauthorized()
		logs := captureServiceAccountAuditLogs(t, func() {
			require.NoError(t, manager.RetryRun(multiUserContext(), run.UUID))
		})
		assert.Contains(t, logs, `operation="retry_run"`)
		after, err := manager.GetRun(run.UUID)
		require.NoError(t, err)
		assert.Equal(t, model.RuntimeStateRunning, after.State)
	})
	t.Run("enable", func(t *testing.T) {
		configureServiceAccountAuditTest(t, true)
		store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
		defer store.Close()
		job, err := manager.CreateJob(multiUserContext(), &model.Job{
			DisplayName: "audit", Enabled: false, ExperimentId: experiment.UUID,
			PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: workflowManifestWithTemplateServiceAccount("nested-sa")},
		})
		require.NoError(t, err)
		logs := captureServiceAccountAuditLogs(t, func() {
			require.NoError(t, manager.ChangeJobMode(multiUserContext(), job.UUID, true))
		})
		assert.Contains(t, logs, `operation="enable_recurring_run"`)
		after, err := manager.GetJob(job.UUID)
		require.NoError(t, err)
		assert.True(t, after.Enabled)
	})
}
