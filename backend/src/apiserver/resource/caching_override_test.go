// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

// Coverage for the run-level enable_caching override (RuntimeConfig.EnableCaching)
// on direct runs and on every recurring-run execution path.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nestedCachingPipeline has tasks under ExitHandler, dsl.If, ParallelFor and
// a sub-pipeline, so only a traversal of nested DAGs reaches all of them.
const nestedCachingPipelineTasks = 9

func nestedCachingPipeline(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../template/testdata/caching_override/input.yaml")
	require.NoError(t, err)
	return string(raw)
}

// taskCaching returns the enable_cache setting of every task in every DAG.
func taskCaching(t *testing.T, manifest model.LargeText) map[string]bool {
	t.Helper()
	tmpl, err := template.New([]byte(manifest), template.TemplateOptions{})
	require.NoError(t, err)
	spec := tmpl.(*template.V2Spec).PipelineSpec()
	got := map[string]bool{}
	for name, task := range spec.GetRoot().GetDag().GetTasks() {
		got["root/"+name] = task.GetCachingOptions().GetEnableCache()
	}
	for component, spec := range spec.GetComponents() {
		for name, task := range spec.GetDag().GetTasks() {
			got[component+"/"+name] = task.GetCachingOptions().GetEnableCache()
		}
	}
	return got
}

func requireAllTasksCaching(t *testing.T, manifest model.LargeText, want bool) {
	t.Helper()
	got := taskCaching(t, manifest)
	require.Len(t, got, nestedCachingPipelineTasks)
	for task, enabled := range got {
		require.Equal(t, want, enabled, "enable_cache of task %s", task)
	}
}

func setMultiUserMode(t *testing.T, enabled bool) {
	t.Helper()
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, enabled)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
}

func initWithNestedCachingVersion(t *testing.T) (*FakeClientManager, *ResourceManager, *model.Experiment, *model.PipelineVersion) {
	t.Helper()
	store, manager, experiment := initWithExperiment(t)
	pipeline, err := manager.CreatePipeline(createPipeline("nested-caching", "", "ns1"))
	require.NoError(t, err)
	version, err := manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "v1", "", "", nestedCachingPipeline(t), "", "ns1"))
	require.NoError(t, err)
	return store, manager, experiment, version
}

var enableCachingCases = []struct {
	name          string
	enableCaching *bool
}{
	{"unset", nil},
	{"false", util.BoolPointer(false)},
	{"true", util.BoolPointer(true)},
}

// The driver reads task specs from the manifest stored with the run, so the
// override must be in that manifest. The uploaded version must not change.
func TestCreateRun_EnableCaching(t *testing.T) {
	for _, source := range []string{"version reference", "inline spec"} {
		for _, tc := range enableCachingCases {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				store, manager, experiment, version := initWithNestedCachingVersion(t)
				defer store.Close()
				pipelineSpec := model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{EnableCaching: tc.enableCaching}}
				if source == "inline spec" {
					pipelineSpec.PipelineSpecManifest = model.LargeText(nestedCachingPipeline(t))
				} else {
					pipelineSpec.PipelineId = version.PipelineId
					pipelineSpec.PipelineVersionId = version.UUID
				}
				created, err := manager.CreateRun(context.Background(), &model.Run{
					DisplayName: "run", Namespace: "ns1", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
				})
				require.NoError(t, err)

				stored, err := manager.GetRun(created.UUID)
				require.NoError(t, err)
				require.Equal(t, tc.enableCaching, stored.EnableCaching)
				if tc.enableCaching == nil {
					require.Equal(t, taskCaching(t, model.LargeText(nestedCachingPipeline(t))), taskCaching(t, stored.PipelineSpecManifest))
				} else {
					requireAllTasksCaching(t, stored.PipelineSpecManifest, *tc.enableCaching)
				}
				unchanged, err := manager.GetPipelineVersion(version.UUID)
				require.NoError(t, err)
				require.Equal(t, version.PipelineSpec, unchanged.PipelineSpec, "the uploaded pipeline version must not change")
			})
		}
	}
}

// The override only changes task settings; the deployment-wide setting is
// passed to the driver separately and still disables caching.
func TestCreateRun_EnableCachingKeepsDeploymentDisable(t *testing.T) {
	for _, cacheDisabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("CacheDisabled=%t", cacheDisabled), func(t *testing.T) {
			store, manager, experiment, version := initWithNestedCachingVersion(t)
			defer store.Close()
			manager.options.CacheDisabled = cacheDisabled
			created, err := manager.CreateRun(context.Background(), &model.Run{
				DisplayName: "run", Namespace: "ns1", ExperimentId: experiment.UUID,
				PipelineSpec: model.PipelineSpec{
					PipelineVersionId: version.UUID,
					RuntimeConfig:     model.RuntimeConfig{EnableCaching: util.BoolPointer(true)},
				},
			})
			require.NoError(t, err)
			requireAllTasksCaching(t, created.PipelineSpecManifest, true)
			workflow := string(created.PipelineRuntimeManifest)
			require.Contains(t, workflow, fmt.Sprintf(`"cache_disabled":%t`, cacheDisabled))
			require.NotContains(t, workflow, fmt.Sprintf(`"cache_disabled":%t`, !cacheDisabled))
		})
	}
}

// Single-user schedules pinned to a version or an inline spec embed a
// compiled workflow that the controller submits without calling CreateRun.
// The persistence agent's report then creates the run from the job, so the
// job must store the effective manifest, and reconciliation after a restart
// must keep it.
func TestCreateJob_EnableCachingPinnedSingleUser(t *testing.T) {
	setMultiUserMode(t, false)
	for _, source := range []string{"version reference", "inline spec"} {
		for _, tc := range enableCachingCases {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				store, manager, experiment, version := initWithNestedCachingVersion(t)
				defer store.Close()
				pipelineSpec := model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{EnableCaching: tc.enableCaching}}
				if source == "inline spec" {
					pipelineSpec.PipelineSpecManifest = model.LargeText(nestedCachingPipeline(t))
				} else {
					pipelineSpec.PipelineVersionId = version.UUID
				}
				job, err := manager.CreateJob(context.Background(), &model.Job{
					DisplayName: "pinned", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true, MaxConcurrency: 1,
					PipelineSpec: pipelineSpec,
				})
				require.NoError(t, err)
				swfClient := store.SwfClient().ScheduledWorkflow(job.Namespace)
				swf, err := swfClient.Get(context.Background(), job.K8SName, v1.GetOptions{})
				require.NoError(t, err)
				require.NotNil(t, swf.Spec.Workflow.Spec, "the controller submits the embedded workflow")

				// A new manager reads the job back from the database, as after a restart.
				restarted := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				stored, err := restarted.GetJob(job.UUID)
				require.NoError(t, err)
				require.Equal(t, tc.enableCaching, stored.EnableCaching)
				require.NoError(t, restarted.ReconcileSwfCrs(context.Background()))
				reconciled, err := swfClient.Get(context.Background(), job.K8SName, v1.GetOptions{})
				require.NoError(t, err)
				require.Equal(t, swf.Spec, reconciled.Spec, "reconciliation must keep the override")

				reported, err := restarted.recurringRunReportPipelineSpec(stored, testWorkflowWithoutStatus(), "run-id")
				require.NoError(t, err)
				require.Equal(t, tc.enableCaching, reported.EnableCaching)
				if tc.enableCaching == nil {
					require.Equal(t, taskCaching(t, model.LargeText(nestedCachingPipeline(t))), taskCaching(t, reported.PipelineSpecManifest))
				} else {
					requireAllTasksCaching(t, reported.PipelineSpecManifest, *tc.enableCaching)
				}
			})
		}
	}
}

// Single-user schedules that follow the latest version run through CreateRun.
// The ScheduledWorkflow carries the override for the controller to send, and
// it is applied after the run's version is resolved. A request without the
// field (a ScheduledWorkflow without a workflow resource) uses the job's value.
func TestCreateJob_EnableCachingFollowLatestSingleUser(t *testing.T) {
	setMultiUserMode(t, false)
	for _, sendOverride := range []bool{true, false} {
		for _, tc := range enableCachingCases[1:] {
			t.Run(fmt.Sprintf("%s/sent=%t", tc.name, sendOverride), func(t *testing.T) {
				store, manager, experiment, version := initWithNestedCachingVersion(t)
				defer store.Close()
				job, err := manager.CreateJob(context.Background(), &model.Job{
					DisplayName: "latest", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true, MaxConcurrency: 1,
					PipelineSpec: model.PipelineSpec{
						PipelineId:    version.PipelineId,
						RuntimeConfig: model.RuntimeConfig{EnableCaching: tc.enableCaching},
					},
				})
				require.NoError(t, err)
				require.Empty(t, job.PipelineVersionId)
				swf, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(context.Background(), job.K8SName, v1.GetOptions{})
				require.NoError(t, err)
				require.Nil(t, swf.Spec.Workflow.Spec)
				require.Equal(t, tc.enableCaching, swf.Spec.Workflow.EnableCaching)

				// The request the controller builds from the ScheduledWorkflow.
				runtimeConfig := model.RuntimeConfig{}
				if sendOverride {
					runtimeConfig.EnableCaching = swf.Spec.Workflow.EnableCaching
				}
				created, err := manager.CreateRun(context.Background(), &model.Run{
					DisplayName: "tick", Namespace: job.Namespace, ExperimentId: job.ExperimentId, RecurringRunId: job.UUID,
					PipelineSpec: model.PipelineSpec{PipelineId: swf.Spec.PipelineId, RuntimeConfig: runtimeConfig},
				})
				require.NoError(t, err)
				require.Equal(t, version.UUID, created.PipelineVersionId)
				require.Equal(t, tc.enableCaching, created.EnableCaching)
				requireAllTasksCaching(t, created.PipelineSpecManifest, *tc.enableCaching)

				// A report that wins the race restores the same effective manifest.
				workflow := testWorkflowWithoutStatus()
				workflow.SetAnnotations(recurringRunPipelineVersionAnnotation, version.UUID)
				reported, err := manager.recurringRunReportPipelineSpec(job, workflow, "run-id")
				require.NoError(t, err)
				require.Equal(t, created.PipelineSpecManifest, reported.PipelineSpecManifest)
			})
		}
	}
}

// In multi-user mode the controller sends only the recurring run ID, and the
// API server restores execution inputs from the stored job.
func TestCreateJob_EnableCachingMultiUser(t *testing.T) {
	setMultiUserMode(t, true)
	for _, pinned := range []bool{true, false} {
		for _, tc := range enableCachingCases {
			t.Run(fmt.Sprintf("pinned=%t/%s", pinned, tc.name), func(t *testing.T) {
				store, manager, experiment, version := initWithNestedCachingVersion(t)
				defer store.Close()
				manager.time = fixedRecurringTime{epoch: 200}
				ctx := multiUserContext()
				pipelineSpec := model.PipelineSpec{PipelineId: version.PipelineId, RuntimeConfig: model.RuntimeConfig{EnableCaching: tc.enableCaching}}
				if pinned {
					pipelineSpec.PipelineVersionId = version.UUID
				}
				job, err := manager.CreateJob(ctx, &model.Job{
					DisplayName: "schedule", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true, MaxConcurrency: 1,
					Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
						PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
					}},
					PipelineSpec: pipelineSpec,
				})
				require.NoError(t, err)

				// The controller's request, restored by a manager that reads the job back from the database.
				restarted := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				restarted.time = fixedRecurringTime{epoch: 200}
				run := &model.Run{DisplayName: "tick", RecurringRunId: job.UUID}
				require.NoError(t, restarted.PrepareRecurringRun(ctx, run))
				require.Equal(t, tc.enableCaching, run.EnableCaching)
				created, err := restarted.CreateRun(ctx, run)
				require.NoError(t, err)
				require.Equal(t, version.UUID, created.PipelineVersionId)
				stored, err := restarted.GetRun(created.UUID)
				require.NoError(t, err)
				require.Equal(t, tc.enableCaching, stored.EnableCaching)
				if tc.enableCaching == nil {
					require.Equal(t, taskCaching(t, version.PipelineSpec), taskCaching(t, stored.PipelineSpecManifest))
				} else {
					requireAllTasksCaching(t, stored.PipelineSpecManifest, *tc.enableCaching)
				}
			})
		}
	}
}

func TestCreateRun_EnableCachingRejectedWithoutPipelineSpec(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	_, err := manager.CreateRun(context.Background(), &model.Run{
		DisplayName: "run", Namespace: "ns1", ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: "kind: ConfigMap\n",
			RuntimeConfig:        model.RuntimeConfig{EnableCaching: util.BoolPointer(false)},
		},
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "pipeline spec"), err.Error())
}
