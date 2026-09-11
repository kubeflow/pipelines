// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateRunPendingFollowLatestTickKeepsClaimedPipelineVersion(t *testing.T) {
	initEnvVars()
	for key, value := range map[string]string{common.MultiUserMode: "true", v1AllowedNamespaces: "ns1"} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.time = fixedRecurringTime{epoch: 200}
	ctx := multiUserContext()
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "version-freeze", Namespace: "ns1"})
	require.NoError(t, err)
	pipeline, err := manager.CreatePipeline(createPipeline("version-freeze", "", "ns1"))
	require.NoError(t, err)
	publishVersion := func(name string) *model.PipelineVersion {
		workflow := util.NewWorkflow(testWorkflow.DeepCopy())
		workflow.Spec.Templates[0].Container.Args = []string{name}
		version, err := manager.CreatePipelineVersion(createPipelineVersion(
			pipeline.UUID, name, name, "", workflow.ToStringForStore(), "", "ns1"))
		require.NoError(t, err)
		return version
	}
	versionA := publishVersion("version-a")
	job, err := manager.CreateJob(ctx, &model.Job{
		DisplayName: "follow-latest", Namespace: "ns1", ExperimentId: experiment.UUID,
		Enabled: true, MaxConcurrency: 1,
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
		}},
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID,
			Parameters: `[{"name":"param1","value":"tick-[[Index]]-[[ScheduledTime]]-[[CurrentTime]]"}]`,
		},
	})
	require.NoError(t, err)
	require.Empty(t, job.PipelineVersionId)

	// Simulate interruption after the first version and tick inputs were claimed,
	// but before the workflow and run were persisted.
	claim, err := store.JobStore().ClaimRecurringRun(job.UUID, "interrupted-tick", 0, 110, 150, versionA.UUID)
	require.NoError(t, err)
	require.True(t, claim.Pending)
	versionB := publishVersion("version-b")
	latest, err := manager.GetLatestPipelineVersion(pipeline.UUID)
	require.NoError(t, err)
	require.Equal(t, versionB.UUID, latest.UUID)

	createTick := func(requestKey string) *model.Run {
		run := &model.Run{DisplayName: requestKey, RecurringRunId: job.UUID}
		require.NoError(t, manager.PrepareRecurringRun(ctx, run))
		created, err := manager.CreateRun(ctx, run)
		require.NoError(t, err)
		return created
	}
	checkWorkflow := func(run *model.Run, marker, parameters string) {
		execution, err := store.ExecClientFake.Execution(job.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
		require.NoError(t, err)
		workflow := execution.(*util.Workflow)
		require.Equal(t, []string{marker}, workflow.Spec.Templates[0].Container.Args)
		require.Equal(t, parameters, workflow.GetWorkflowParametersAsMap()["param1"])
	}
	retried := createTick("interrupted-tick")
	require.Equal(t, versionA.UUID, retried.PipelineVersionId)
	require.Equal(t, int64(110), retried.ScheduledAtInSec)
	require.Equal(t, int64(150), retried.CreatedAtInSec)
	checkWorkflow(retried, "version-a", "tick-1-19700101000150-19700101000230")
	state, err := store.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.False(t, state.Pending)
	require.Equal(t, versionA.UUID, state.PipelineVersionID)

	// Finishing the first run releases the concurrency limit. A distinct due tick
	// resolves latest again instead of pinning the whole recurring run to A.
	retried.State = model.RuntimeStateSucceeded
	retried.FinishedAtInSec = 201
	require.NoError(t, store.RunStore().UpdateRun(retried))
	manager.time = fixedRecurringTime{epoch: 210}
	next := createTick("next-tick")
	require.Equal(t, versionB.UUID, next.PipelineVersionId)
	require.Equal(t, int64(120), next.ScheduledAtInSec)
	require.Equal(t, int64(210), next.CreatedAtInSec)
	require.NotEqual(t, retried.UUID, next.UUID)
	checkWorkflow(next, "version-b", "tick-2-19700101000200-19700101000330")
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())
	storedJob, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Empty(t, storedJob.PipelineVersionId)
}

func TestCreateRunAcknowledgesTickAfterPipelineVersionDeletion(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprintf("deleted=%t", deleted), func(t *testing.T) {
			initEnvVars()
			for key, value := range map[string]string{common.MultiUserMode: "true", v1AllowedNamespaces: "ns1"} {
				previous := viper.Get(key)
				viper.Set(key, value)
				t.Cleanup(func() { viper.Set(key, previous) })
			}
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			manager.time = fixedRecurringTime{epoch: 200}
			ctx := multiUserContext()
			experiment, err := manager.CreateExperiment(&model.Experiment{Name: "deleted-tick", Namespace: "ns1"})
			require.NoError(t, err)
			pipeline, err := manager.CreatePipeline(createPipeline("deleted-tick", "", "ns1"))
			require.NoError(t, err)
			publishVersion := func(name string) *model.PipelineVersion {
				workflow := util.NewWorkflow(testWorkflow.DeepCopy())
				workflow.Spec.Templates[0].Container.Args = []string{name}
				version, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, name, name, "", workflow.ToStringForStore(), "", "ns1"))
				require.NoError(t, err)
				return version
			}
			versionA := publishVersion("version-a")
			job, err := manager.CreateJob(ctx, &model.Job{
				DisplayName: "follow-latest", Namespace: "ns1", ExperimentId: experiment.UUID,
				Enabled: true, MaxConcurrency: 1,
				Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
				}},
				PipelineSpec: model.PipelineSpec{
					PipelineId: pipeline.UUID,
					Parameters: `[{"name":"param1","value":"tick-[[Index]]"}]`,
				},
			})
			require.NoError(t, err)
			createTick := func(requestKey string) *model.Run {
				run := &model.Run{DisplayName: requestKey, RecurringRunId: job.UUID}
				require.NoError(t, manager.PrepareRecurringRun(ctx, run))
				created, err := manager.CreateRun(ctx, run)
				require.NoError(t, err)
				return created
			}
			first := createTick("unacknowledged-tick")
			require.Equal(t, versionA.UUID, first.PipelineVersionId)
			first.State = model.RuntimeStateSucceeded
			first.FinishedAtInSec = 201
			require.NoError(t, store.RunStore().UpdateRun(first))
			completedState, err := store.JobStore().GetRecurringRunState(job.UUID)
			require.NoError(t, err)
			require.False(t, completedState.Pending)

			// The historical template can disappear before acknowledgement, whether
			// or not retention has also removed the completed run.
			versionB := publishVersion("version-b")
			if deleted {
				require.NoError(t, manager.DeleteRun(ctx, first.UUID))
			}
			require.NoError(t, manager.DeletePipelineVersion(versionA.UUID))
			latest, err := manager.GetLatestPipelineVersion(pipeline.UUID)
			require.NoError(t, err)
			require.Equal(t, versionB.UUID, latest.UUID)
			manager.time = fixedRecurringTime{epoch: 210}
			acknowledged := createTick("unacknowledged-tick")
			require.Equal(t, first.UUID, acknowledged.UUID)
			require.Equal(t, first.DisplayName, acknowledged.DisplayName)
			require.Equal(t, job.UUID, acknowledged.RecurringRunId)
			require.Equal(t, first.Namespace, acknowledged.Namespace)
			require.Equal(t, first.ExperimentId, acknowledged.ExperimentId)
			require.Equal(t, first.ServiceAccount, acknowledged.ServiceAccount)
			require.Equal(t, versionA.UUID, acknowledged.PipelineVersionId)
			require.Equal(t, first.CreatedAtInSec, acknowledged.CreatedAtInSec)
			require.Equal(t, first.ScheduledAtInSec, acknowledged.ScheduledAtInSec)
			workflowCount := 1
			if deleted {
				require.Equal(t, model.RuntimeStateUnspecified, acknowledged.State)
				_, err = manager.GetRun(first.UUID)
				require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
				workflowCount = 0
			} else {
				persisted, err := manager.GetRun(first.UUID)
				require.NoError(t, err)
				require.Equal(t, persisted, acknowledged)
			}
			require.Equal(t, workflowCount, store.ExecClientFake.GetWorkflowCount())
			stateAfterAcknowledgement, err := store.JobStore().GetRecurringRunState(job.UUID)
			require.NoError(t, err)
			require.Equal(t, completedState, stateAfterAcknowledgement)

			next := createTick("next-tick")
			require.Equal(t, versionB.UUID, next.PipelineVersionId)
			require.NotEqual(t, first.UUID, next.UUID)
			require.Equal(t, int64(120), next.ScheduledAtInSec)
			execution, err := store.ExecClientFake.Execution(job.Namespace).Get(ctx, next.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			workflow := execution.(*util.Workflow)
			require.Equal(t, []string{"version-b"}, workflow.Spec.Templates[0].Container.Args)
			require.Equal(t, "tick-2", workflow.GetWorkflowParametersAsMap()["param1"])
			require.Equal(t, workflowCount+1, store.ExecClientFake.GetWorkflowCount())
		})
	}
}
