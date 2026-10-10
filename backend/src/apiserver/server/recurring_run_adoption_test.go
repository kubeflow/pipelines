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

package server

import (
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func removeLegacySchedulingState(t *testing.T, clients *resource.FakeClientManager, jobID string) {
	t.Helper()
	_, err := clients.DB().Exec(`DELETE FROM recurring_run_states WHERE "JobUUID" = ?`, jobID)
	require.NoError(t, err)
}

// markScheduleAsLegacy models a job created before native progress and seals existed.
func markScheduleAsLegacy(t *testing.T, clients *resource.FakeClientManager, jobID string) {
	t.Helper()
	removeLegacySchedulingState(t, clients, jobID)
	_, err := clients.DB().Exec(`DELETE FROM recurring_run_adoptions WHERE "ID" = ?`, "legacy-2.18:"+jobID)
	require.NoError(t, err)
}

func TestRecurringRunAdoptionRestoresExecutionFromStoredDefinition(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	server := createRunServer(manager)
	request := &api.CreateRunRequest{Run: &api.Run{DisplayName: "adopted-tick", RecurringRunId: job.UUID}}
	markScheduleAsLegacy(t, clients, job.UUID)
	_, err := server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "no trusted scheduling state")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())

	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	require.Equal(t, int64(1), receipt.AdoptedCount)
	require.JSONEq(t, `["`+job.UUID+`"]`, string(receipt.JobIDs))
	storedJob, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, job.PipelineSpec, storedJob.PipelineSpec)
	require.Equal(t, job.ServiceAccount, storedJob.ServiceAccount)
	require.Equal(t, job.Trigger, storedJob.Trigger)

	// Once adopted, controller-editable inputs cannot replace the stored definition.
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, swf.Spec.Workflow)
	swf.Spec.ServiceAccount = "privileged-sa"
	swf.Spec.PipelineId = "foreign-pipeline"
	swf.Spec.Workflow = &swfapi.WorkflowResource{
		PipelineRoot: "s3://foreign-root",
		Parameters:   []swfapi.Parameter{{Name: "param1", Value: "foreign-input"}},
	}
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(swf)))
	request.Run.ServiceAccount = "privileged-sa"
	run, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, job.ExperimentId, run.ExperimentId)
	require.Equal(t, "custom-sa", run.ServiceAccount)
	requireScheduledRunParameters(t, manager, run, 1, 100)
	storedRun, err := manager.GetRun(run.RunId)
	require.NoError(t, err)
	require.NotContains(t, string(storedRun.PipelineRuntimeManifest), "foreign-input")
	require.NotContains(t, string(storedRun.PipelineRuntimeManifest), "foreign-root")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())

	// Adoption does not bypass the controller's continuing execution authorization.
	review.controllerAllowed = false
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "Unauthorized")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAdoptionPreservesDisabledUntilAPIEnable(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	jobs := createJobServer(manager)
	_, err := jobs.DisableRecurringRun(ctx, &api.DisableRecurringRunRequest{RecurringRunId: job.UUID})
	require.NoError(t, err)
	markScheduleAsLegacy(t, clients, job.UUID)

	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
	swf, err := clients.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, swf.Spec.Enabled)
	server := createRunServer(manager)
	request := &api.CreateRunRequest{Run: &api.Run{DisplayName: "enabled-after-adoption", RecurringRunId: job.UUID}}
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "disabled")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())

	_, err = jobs.EnableRecurringRun(ctx, &api.EnableRecurringRunRequest{RecurringRunId: job.UUID})
	require.NoError(t, err)
	run, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, "custom-sa", run.ServiceAccount)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAdoptionReceiptPreventsReseeding(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	markScheduleAsLegacy(t, clients, job.UUID)
	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	before, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(999)
	forgedTime := metav1.NewTime(time.Unix(1, 0))
	swf.Status.Trigger.LastTriggeredTime = &forgedTime
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)

	again, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.Equal(t, receipt, again)
	after, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, before, after)

	// Corruption after adoption must retain the receipt that prevents reseeding.
	removeLegacySchedulingState(t, clients, job.UUID)
	again, err = manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.Equal(t, receipt, again)
	_, err = clients.JobStore().GetRecurringRunState(job.UUID)
	require.ErrorContains(t, err, "no trusted scheduling state")
	_, err = createRunServer(manager).CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "cannot-reseed", RecurringRunId: job.UUID,
	}})
	require.ErrorContains(t, err, "no trusted scheduling state")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAdoptionContinuesAfterRetainedTick(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	request := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: scheduleutil.NewScheduledWorkflow(swf).NextResourceName(), RecurringRunId: job.UUID,
	}}
	server := createRunServer(manager)
	first, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	completed, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	completed.State = model.RuntimeStateSucceeded
	require.NoError(t, clients.RunStore().UpdateRun(completed))
	executions := clients.ExecClientFake.Execution(job.Namespace)
	execution, err := executions.Get(ctx, completed.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	execution.(*util.Workflow).Status.Phase = workflowapi.WorkflowSucceeded
	_, err = executions.Update(ctx, execution, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Preserve the controller's last accepted tick before removing API-owned state.
	swf.Status.Trigger.LastIndex = util.Int64Pointer(1)
	lastScheduled := metav1.NewTime(first.ScheduledAt.AsTime())
	swf.Status.Trigger.LastTriggeredTime = &lastScheduled
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	markScheduleAsLegacy(t, clients, job.UUID)
	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, int64(1), state.LastRunIndex)
	require.Equal(t, first.ScheduledAt.Seconds, state.LastScheduledAtInSec)

	replayed, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, first.RunId, replayed.RunId)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	advanceScheduledRunClock(clients, 200)
	swf, err = swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	request.Run.DisplayName = scheduleutil.NewScheduledWorkflow(swf).NextResourceName()
	second, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.NotEqual(t, first.RunId, second.RunId)
	requireScheduledRunParameters(t, manager, second, 2, 110)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAdoptionResolvesLegacyResourceReferences(t *testing.T) {
	f := newRecurringAcknowledgementFixture(t)
	ctx := scheduleContext(scheduleControllerIdentity)
	legacy := *f.job
	references := legacy.ToV1().ResourceReferences
	require.Len(t, references, 4)
	for _, ref := range references {
		stored, err := f.clients.ResourceReferenceStore().GetResourceReference(f.job.UUID, model.JobResourceType, ref.ReferenceType)
		require.NoError(t, err)
		require.Equal(t, ref.ReferenceUUID, stored.ReferenceUUID)
	}
	_, err := f.clients.DB().Exec(`UPDATE jobs SET "Namespace" = '', "ExperimentUUID" = '', "PipelineId" = '', "PipelineVersionId" = '' WHERE "UUID" = ?`, f.job.UUID)
	require.NoError(t, err)
	markScheduleAsLegacy(t, f.clients, f.job.UUID)
	resolved, err := f.manager.GetJob(f.job.UUID)
	require.NoError(t, err)
	require.Equal(t, f.job.Namespace, resolved.Namespace)
	require.Equal(t, f.job.ExperimentId, resolved.ExperimentId)
	require.Equal(t, f.job.PipelineId, resolved.PipelineId)
	require.Equal(t, f.job.PipelineVersionId, resolved.PipelineVersionId)

	receipt, err := f.manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	require.Equal(t, int64(1), receipt.AdoptedCount)
	// Adoption accepts the resolved definition without rewriting its legacy representation.
	var namespace, experimentID, pipelineID, versionID string
	require.NoError(t, f.clients.DB().QueryRow(`SELECT "Namespace", "ExperimentUUID", "PipelineId", "PipelineVersionId" FROM jobs WHERE "UUID" = ?`, f.job.UUID).
		Scan(&namespace, &experimentID, &pipelineID, &versionID))
	require.Empty(t, namespace)
	require.Empty(t, experimentID)
	require.Empty(t, pipelineID)
	require.Empty(t, versionID)
	run, err := f.submit("reference-backed-adopted-tick")
	require.NoError(t, err)
	require.Equal(t, f.job.ExperimentId, run.ExperimentId)
	stored, err := f.manager.GetRun(run.RunId)
	require.NoError(t, err)
	require.Equal(t, f.job.Namespace, stored.Namespace)
	require.Equal(t, f.job.PipelineId, stored.PipelineId)
	require.Equal(t, f.job.PipelineVersionId, stored.PipelineVersionId)
	require.Equal(t, 1, f.clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAdoptionReplaysRecoveredHistoricalRunIdentity(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	request := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: scheduleutil.NewScheduledWorkflow(swf).NextResourceName(), RecurringRunId: job.UUID,
	}}
	server := createRunServer(manager)
	first, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	legacy, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	executions := clients.ExecClientFake.Execution(job.Namespace)
	execution, err := executions.Get(ctx, legacy.K8SName, metav1.GetOptions{})
	require.NoError(t, err)

	// Old persistence-agent recovery used the compiled workflow's display name
	// and its historical UUID, rather than the controller key or tick-derived ID.
	const historicalID = "b64f701a-c789-4e7e-8bab-cb34a22f2c62"
	require.NotEqual(t, first.RunId, historicalID)
	execution.SetLabels(util.LabelKeyWorkflowRunId, historicalID)
	execution.(*util.Workflow).Status.Phase = workflowapi.WorkflowSucceeded
	_, err = executions.Update(ctx, execution, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, clients.RunStore().DeleteRun(first.RunId))
	legacy.UUID = historicalID
	legacy.DisplayName = legacy.K8SName
	legacy.State = model.RuntimeStateSucceeded
	legacy.WorkflowRuntimeManifest = model.LargeText(execution.ToStringForStore())
	legacy.PipelineRuntimeManifest = legacy.WorkflowRuntimeManifest
	db, err := clients.TransferDB()
	require.NoError(t, err)
	require.NoError(t, db.Create(legacy).Error)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(1)
	lastScheduled := metav1.NewTime(first.ScheduledAt.AsTime())
	swf.Status.Trigger.LastTriggeredTime = &lastScheduled
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	markScheduleAsLegacy(t, clients, job.UUID)

	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	replayed, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, historicalID, replayed.RunId)
	require.Equal(t, legacy.DisplayName, replayed.DisplayName)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())

	// Retention of the baseline survives deletion of its row without creating a
	// replacement run or reporting a different ID for the same consumed tick.
	require.NoError(t, clients.RunStore().DeleteRun(historicalID))
	acknowledged, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, historicalID, acknowledged.RunId)
	require.Equal(t, first.ScheduledAt, acknowledged.ScheduledAt)
	_, err = manager.GetRun(historicalID)
	require.Error(t, err)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())

	advanceScheduledRunClock(clients, 200)
	swf, err = swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	request.Run.DisplayName = scheduleutil.NewScheduledWorkflow(swf).NextResourceName()
	second, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.NotEqual(t, historicalID, second.RunId)
	requireScheduledRunParameters(t, manager, second, 2, 110)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Empty(t, state.LastRunUUID, "new deterministic ticks must not inherit the adopted historical run ID")
}
