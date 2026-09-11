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
	"fmt"
	"strings"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func advanceScheduledRunClock(clients *resource.FakeClientManager, epoch int64) {
	for clients.Time().Now().Unix() < epoch {
	}
}

func requireScheduledRunParameters(t *testing.T, manager *resource.ResourceManager, run *api.Run, index, scheduledAt int64) {
	t.Helper()
	stored, err := manager.GetRun(run.RunId)
	require.NoError(t, err)
	execution, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(stored.WorkflowRuntimeManifest))
	require.NoError(t, err)
	workflow, ok := execution.(*util.Workflow)
	require.True(t, ok)
	require.Equal(t, fmt.Sprintf("authorized-%d-%s", index, time.Unix(scheduledAt, 0).UTC().Format("20060102150405")),
		workflow.GetWorkflowParametersAsMap()["param1"])
	require.Equal(t, "custom-sa", execution.ServiceAccount())
	require.Equal(t, scheduledAt, run.ScheduledAt.Seconds)
}

func TestRecurringRunIgnoresCRSchedulingStateAndClientScheduledAt(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	forgedTime := metav1.NewTime(time.Unix(1, 0))
	swf.Status.Trigger.LastIndex = util.Int64Pointer(9999)
	swf.Status.Trigger.LastTriggeredTime = &forgedTime
	swf.Spec.Trigger = swfapi.Trigger{PeriodicSchedule: &swfapi.PeriodicSchedule{
		StartTime: &forgedTime, IntervalSecond: 1,
	}}
	swf.Spec.NoCatchup = util.BoolPointer(true)
	swf.Spec.MaxConcurrency = util.Int64Pointer(10)
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(swf)))
	storedJob, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, job.Trigger, storedJob.Trigger)
	require.Equal(t, job.NoCatchup, storedJob.NoCatchup)
	require.Equal(t, job.MaxConcurrency, storedJob.MaxConcurrency)

	server := createRunServer(manager)
	request := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "first-tick", RecurringRunId: job.UUID,
		ScheduledAt: timestamppb.New(time.Unix(999999, 0)),
		PipelineSource: &api.Run_PipelineVersionReference{PipelineVersionReference: &api.PipelineVersionReference{
			PipelineId: "attacker-pipeline", PipelineVersionId: "attacker-version",
		}},
	}}
	first, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	requireScheduledRunParameters(t, manager, first, 1, 100)
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, int64(1), state.LastRunIndex)
	require.Equal(t, int64(100), state.LastScheduledAtInSec)
	require.False(t, state.Pending)

	// A new request key and hostile timestamp cannot accelerate the next tick.
	request.Run.DisplayName = "second-tick"
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "no authorized tick is due")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())

	// Even when another tick is due, the trusted concurrency limit still applies.
	advanceScheduledRunClock(clients, 200)
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "maximum concurrency")
	afterDenied, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, state, afterDenied)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())

	completed, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	completed.State = model.RuntimeStateSucceeded
	require.NoError(t, clients.RunStore().UpdateRun(completed))
	second, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	requireScheduledRunParameters(t, manager, second, 2, 110)
	require.NotEqual(t, first.RunId, second.RunId)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())

	// Replaying an older key returns its original execution without a new claim.
	request.Run.DisplayName = "first-tick"
	replayed, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, first.RunId, replayed.RunId)
	requireScheduledRunParameters(t, manager, replayed, 1, 100)
	afterReplay, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, int64(2), afterReplay.LastRunIndex)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())

	// The latest completed key remains consumed after its run is deleted.
	require.NoError(t, manager.DeleteRun(ctx, second.RunId))
	request.Run.DisplayName = "second-tick"
	acknowledged, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, second.RunId, acknowledged.RunId)
	require.Equal(t, second.CreatedAt, acknowledged.CreatedAt)
	require.Equal(t, second.ScheduledAt, acknowledged.ScheduledAt)
	require.Equal(t, "custom-sa", acknowledged.ServiceAccount)
	require.Zero(t, acknowledged.State)
	_, err = manager.GetRun(second.RunId)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
	stateAfterAcknowledgement, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, afterReplay, stateAfterAcknowledgement)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunRejectsOverlongRequestKeyBeforeClaim(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	server := createRunServer(manager)
	before, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	request := &api.CreateRunRequest{Run: &api.Run{DisplayName: strings.Repeat("x", 256), RecurringRunId: job.UUID}}
	_, err = server.CreateRun(ctx, request)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	after, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
	request.Run.DisplayName = "valid-tick"
	run, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	requireScheduledRunParameters(t, manager, run, 1, 100)
}

func TestRecurringRunUsesStoredNoCatchupPolicy(t *testing.T) {
	for _, noCatchup := range []bool{false, true} {
		t.Run(fmt.Sprintf("noCatchup=%t", noCatchup), func(t *testing.T) {
			clients, manager, job, review := newAuthorizedScheduleWithCatchupPolicy(t, noCatchup)
			review.controllerAllowed = true
			ctx := scheduleContext(scheduleControllerIdentity)
			swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
			swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			swf.Spec.NoCatchup = util.BoolPointer(!noCatchup)
			_, err = swfs.Update(ctx, swf)
			require.NoError(t, err)
			require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(swf)))
			advanceScheduledRunClock(clients, 200)
			server := createRunServer(manager)
			run, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
				DisplayName: "delayed-tick", RecurringRunId: job.UUID,
				ScheduledAt: timestamppb.New(time.Unix(999999, 0)),
			}})
			require.NoError(t, err)
			expectedTime := int64(100)
			if noCatchup {
				expectedTime = run.CreatedAt.Seconds
				require.GreaterOrEqual(t, expectedTime, int64(200))
			}
			requireScheduledRunParameters(t, manager, run, 1, expectedTime)
		})
	}
}

func TestRecurringRunEvaluatesStoredCronInConfiguredTimezone(t *testing.T) {
	viper.Set(scheduleutil.TimeZone, "America/New_York")
	t.Cleanup(func() { viper.Set(scheduleutil.TimeZone, "") })
	start := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)
	clients, manager, job, review := newAuthorizedScheduleWithTrigger(t, false, model.Trigger{CronSchedule: model.CronSchedule{
		Cron: util.StringPointer("0 0 0 * * *"), CronScheduleStartTimeInSec: util.Int64Pointer(start.Unix()),
	}}, start.Add(5*time.Hour))
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	swf.Spec.CronSchedule.Cron = "* * * * * *"
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	server := createRunServer(manager)
	run, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "midnight-new-york", RecurringRunId: job.UUID,
		ScheduledAt: timestamppb.New(start),
	}})
	require.NoError(t, err)
	// Midnight in New York is 04:00 UTC during daylight saving time.
	requireScheduledRunParameters(t, manager, run, 1, start.Add(4*time.Hour).Unix())
}

func TestRecurringRunResumesOnlyTheAuthorizedPendingTick(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	// Simulate a server stopping after the durable claim, before workflow creation.
	pending, err := clients.JobStore().ClaimRecurringRun(job.UUID, "pending-tick", 0, 100, 103, "")
	require.NoError(t, err)
	advanceScheduledRunClock(clients, 500)
	ctx := scheduleContext(scheduleControllerIdentity)
	server := createRunServer(manager)
	request := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "different-tick", RecurringRunId: job.UUID,
		ScheduledAt: timestamppb.New(time.Unix(999999, 0)),
	}}
	review.controllerAllowed = true
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "previous tick is still pending")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())

	request.Run.DisplayName = "pending-tick"
	review.controllerAllowed = false
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "Unauthorized")
	afterDenied, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, pending, afterDenied)

	review.controllerAllowed = true
	run, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	requireScheduledRunParameters(t, manager, run, 1, 100)
	require.Equal(t, int64(103), run.CreatedAt.Seconds)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	completed, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, int64(1), completed.LastRunIndex)
	require.False(t, completed.Pending)
}

func TestRecurringRunReplayStillChecksEnabledStateAndAuthorization(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprintf("deleted=%t", deleted), func(t *testing.T) {
			for _, denial := range []string{"disabled-db", "disabled-cr", "revoked-service-account", "removed-allowlist", "denied-namespace"} {
				t.Run(denial, func(t *testing.T) {
					clients, manager, job, review := newAuthorizedSchedule(t)
					review.controllerAllowed = true
					ctx := scheduleContext(scheduleControllerIdentity)
					server := createRunServer(manager)
					request := &api.CreateRunRequest{Run: &api.Run{DisplayName: "same-tick", RecurringRunId: job.UUID}}
					run, err := server.CreateRun(ctx, request)
					require.NoError(t, err)
					workflowCount := 1
					if deleted {
						require.NoError(t, manager.DeleteRun(ctx, run.RunId))
						workflowCount = 0
					}
					before, err := clients.JobStore().GetRecurringRunState(job.UUID)
					require.NoError(t, err)
					switch denial {
					case "disabled-db":
						require.NoError(t, clients.JobStore().ChangeJobMode(job.UUID, false))
					case "disabled-cr":
						swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
						swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
						require.NoError(t, err)
						swf.Spec.Enabled = false
						_, err = swfs.Update(ctx, swf)
						require.NoError(t, err)
					case "revoked-service-account":
						review.controllerAllowed = false
					case "removed-allowlist":
						viper.Set(common.AllowedServiceAccountsFlag, "")
					case "denied-namespace":
						review.denyRunCreation = true
					}
					_, err = server.CreateRun(ctx, request)
					require.Error(t, err)
					require.Equal(t, workflowCount, clients.ExecClientFake.GetWorkflowCount())
					after, err := clients.JobStore().GetRecurringRunState(job.UUID)
					require.NoError(t, err)
					require.Equal(t, before, after)
				})
			}
		})
	}
}

func TestRecurringRunRejectsLegacyScheduleWithoutTrustedState(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	_, err := clients.DB().Exec(`DELETE FROM recurring_run_states WHERE "JobUUID" = ?`, job.UUID)
	require.NoError(t, err)
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(99)
	lastTriggered := metav1.NewTime(time.Unix(1, 0))
	swf.Status.Trigger.LastTriggeredTime = &lastTriggered
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	server := createRunServer(manager)
	_, err = server.CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "untrusted-progress", RecurringRunId: job.UUID,
		ScheduledAt: timestamppb.New(time.Unix(100, 0)),
	}})
	require.ErrorContains(t, err, "no trusted scheduling state")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
	_, err = clients.JobStore().GetRecurringRunState(job.UUID)
	require.ErrorContains(t, err, "no trusted scheduling state")
}
