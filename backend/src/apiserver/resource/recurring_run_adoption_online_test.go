// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func onlineAdoptionManager(t *testing.T) (*ResourceManager, *FakeClientManager) {
	t.Helper()
	initEnvVars()
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	clients := NewFakeClientManagerOrFatal(util.NewFakeTime(time.Unix(100, 0)))
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	return NewResourceManager(clients, &ResourceManagerOptions{CollectMetrics: false}), clients
}
func onlineLegacyJob(t *testing.T, clients *FakeClientManager, enabled bool) *model.Job {
	t.Helper()
	db, err := clients.TransferDB()
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&model.Job{}).Count(&count).Error)
	name := fmt.Sprintf("legacy-%d", count)
	swf, err := clients.SwfClient().ScheduledWorkflow("ns1").Create(context.Background(), &swfapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{GenerateName: name}, Spec: swfapi.ScheduledWorkflowSpec{Enabled: enabled, ServiceAccount: "untrusted"}})
	require.NoError(t, err)
	swf.UID = types.UID(name)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(3)
	last := metav1.NewTime(time.Unix(90, 0))
	swf.Status.Trigger.LastTriggeredTime = &last
	job, err := clients.JobStore().CreateJob(&model.Job{UUID: string(swf.UID), K8SName: swf.Name, Namespace: swf.Namespace, Enabled: enabled, ServiceAccount: "stored", MaxConcurrency: 1, Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{IntervalSecond: util.Int64Pointer(10)}}})
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: "legacy-2.18:" + job.UUID}).Error)
	return job
}
func TestOnlineResourceAdoptionResumesAndPreservesCurrentDisabledMode(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	interrupted := &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	manager.swfClient = interrupted
	require.ErrorContains(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID), "interrupted")
	require.ErrorContains(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID), "incomplete")
	db, err := clients.TransferDB()
	require.NoError(t, err)
	receipt, err := storage.GetLegacyRecurringRunAdoptionForJob(db, job.UUID)
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	// The normal mode-change path shares the job lock and remains usable while
	// adoption is pending. Resume must use this current disabled authorization.
	manager.swfClient = clients.SwfClient()
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	current, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, current.Spec.Enabled)
	require.Equal(t, "stored", current.Spec.ServiceAccount)
	require.EqualValues(t, 3, *current.Status.Trigger.LastIndex)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	require.ErrorContains(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID), "refusing to reseed")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
}
func TestOnlineResourceAdoptionIsolatesInvalidRecordsAndLateInventory(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	bad := onlineLegacyJob(t, clients, true)
	good := onlineLegacyJob(t, clients, false)
	ctx := context.Background()
	broken, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, bad.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	broken.Status.Trigger.LastIndex = util.Int64Pointer(-1)
	require.Error(t, manager.AdoptLegacyRecurringRun(ctx, bad.UUID))
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, good.UUID))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, good.UUID))
	late := onlineLegacyJob(t, clients, true)
	pending, err := manager.ListLegacyRecurringRunAdoptionCandidates(ctx, "", 100)
	require.NoError(t, err)
	var ids []string
	for _, candidate := range pending {
		ids = append(ids, candidate.ID)
	}
	require.ElementsMatch(t, []string{bad.UUID, late.UUID}, ids)
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, late.UUID))
}

func TestOnlineAdoptionRetainsExecutionBeforeAcknowledgedTime(t *testing.T) {
	_, clients := onlineAdoptionManager(t)
	job, swf, run, wf := legacyAdoptionFixture()
	run.K8SName = run.DisplayName
	wf.Name = run.K8SName
	run.ScheduledAtInSec, run.CreatedAtInSec = 90, 99
	wf.SetCannonicalLabels(swf.Name, 90, 3)
	wf.CreationTimestamp = metav1.NewTime(time.Unix(99, 0))
	run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
	require.NoError(t, err)
	require.EqualValues(t, 100, state.LastScheduledAtInSec)
	require.EqualValues(t, 99, state.LastCreatedAtInSec)
	require.Equal(t, run.UUID, state.LastRunUUID)
	db, err := clients.TransferDB()
	require.NoError(t, err)
	require.NoError(t, db.Create(job).Error)
	require.NoError(t, storage.ApplyLegacyRecurringRunAdoptionForJob(db, storage.RecurringRunAdoptionCandidate{Job: *job, State: *state}, 210))
}

func TestOnlineModeFailureRemainsDurablyPending(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	manager.swfClient = &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	require.ErrorContains(t, manager.ChangeJobMode(ctx, job.UUID, false), "interrupted")
	current, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, current.Enabled)
	require.ErrorContains(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID), "incomplete")
	pending, err := manager.ListLegacyRecurringRunAdoptionCandidates(ctx, "", 100)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	manager.swfClient = clients.SwfClient()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, live.Spec.Enabled)
}

func TestOnlineModeReconciliationUsesManagedFence(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	calls := 0
	manager.options.EnsureRecurringRunModeChanged = func(_ context.Context, id string) error {
		calls++
		require.Equal(t, job.UUID, id)
		return fmt.Errorf("old writers remain")
	}
	require.ErrorContains(t, manager.ChangeJobMode(ctx, job.UUID, false), "old writers remain")
	require.Equal(t, 1, calls)
	current, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, current.Enabled)
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, live.Spec.Enabled)
	require.ErrorContains(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID), "incomplete")
}

func TestOnlineModeDoesNotAcknowledgePendingTick(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	state, err := clients.JobStore().ClaimRecurringRun(job.UUID, "pending", 3, 100, 100, "")
	require.NoError(t, err)
	require.True(t, state.Pending)
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.EqualValues(t, 3, *live.Status.Trigger.LastIndex)
	require.False(t, live.Spec.Enabled)
	current, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, state, current)
}

func TestOnlineDisableMissingCRStillRevokesAuthorization(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	require.NoError(t, clients.SwfClient().ScheduledWorkflow("ns1").Delete(ctx, job.K8SName, &metav1.DeleteOptions{}))
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
	stored, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	require.Error(t, manager.ChangeJobMode(ctx, job.UUID, true))
	stored, err = clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
	_, err = clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.Error(t, err)
}

func TestOnlineScheduleMutationsWaitForWriterHandoff(t *testing.T) {
	for _, enable := range []bool{false, true} {
		t.Run(fmt.Sprintf("enable=%t", enable), func(t *testing.T) {
			manager, clients := onlineAdoptionManager(t)
			job := onlineLegacyJob(t, clients, !enable)
			ctx := context.Background()
			ready := false
			manager.options.ScheduleWritersReady = func(context.Context) error {
				if !ready {
					return fmt.Errorf("private writer infrastructure detail")
				}
				return nil
			}
			liveBefore, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			liveBefore = liveBefore.DeepCopy()
			storedBefore, err := clients.JobStore().GetJob(job.UUID)
			require.NoError(t, err)
			db, err := clients.TransferDB()
			require.NoError(t, err)
			assertNotApplied := func(err error) {
				t.Helper()
				require.Error(t, err)
				status := util.ToGRPCStatus(err)
				require.Equal(t, codes.Unavailable, status.Code())
				require.Equal(t, err.(*util.UserError).ExternalMessage(), status.Message())
				require.Contains(t, status.Message(), "was not applied")
				require.Contains(t, status.Message(), "handoff")
				require.Contains(t, status.Message(), "after handoff completes")
				require.NotContains(t, status.Message(), "private writer infrastructure detail")
			}
			assertNotApplied(manager.ChangeJobMode(ctx, job.UUID, enable))
			current, err := clients.JobStore().GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, storedBefore, current)
			live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, liveBefore, live)
			var states, receipts int64
			require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&states).Error)
			require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Count(&receipts).Error)
			require.Zero(t, states)
			require.Zero(t, receipts)
			var before int64
			require.NoError(t, db.Model(&model.Job{}).Count(&before).Error)
			newJob := &model.Job{DisplayName: "new", Namespace: "ns1", Enabled: true, PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld), RuntimeConfig: model.RuntimeConfig{Parameters: `{ "text": "world" }`}}}
			_, err = manager.CreateJob(ctx, newJob)
			assertNotApplied(err)
			require.Empty(t, newJob.UUID)
			var after int64
			require.NoError(t, db.Model(&model.Job{}).Count(&after).Error)
			require.Equal(t, before, after)
			require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&states).Error)
			require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Count(&receipts).Error)
			require.Zero(t, states)
			require.Zero(t, receipts)
			_, err = clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, "job-", metav1.GetOptions{})
			require.Error(t, err)
			ready = true
			require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, enable))
			current, err = clients.JobStore().GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, enable, current.Enabled)
			live, err = clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, enable, live.Spec.Enabled)
			created, err := manager.CreateJob(ctx, newJob)
			require.NoError(t, err)
			require.NotEmpty(t, created.UUID)
			require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, created.UUID))
		})
	}
}

func TestOnlineAndManualAdoptionRefuseMissingTransferredState(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, false)
	db, err := clients.TransferDB()
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TransferReceipt{Key: "receipt", Source: "source", Namespace: job.Namespace, Kind: "schedule", SourceID: "old", TargetID: job.UUID, Digest: "digest"}).Error)
	require.ErrorContains(t, manager.AdoptLegacyRecurringRun(context.Background(), job.UUID), "transferred scheduling state is missing")
	_, err = manager.AdoptLegacyRecurringRuns(context.Background())
	require.ErrorContains(t, err, "transferred scheduling state is missing")
	var count int64
	require.NoError(t, db.Model(&model.RecurringRunState{}).Where(&model.RecurringRunState{JobUUID: job.UUID}).Count(&count).Error)
	require.Zero(t, count)
}

func TestOnlineModeChangeSupersedesCachedReconciliationFailure(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	a := NewAutomaticRecurringRunAdoption(manager, func(context.Context) error { return nil })
	manager.options.EnsureRecurringRunModeChanged = a.EnsureAfterModeChange
	manager.swfClient = &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	require.ErrorContains(t, manager.ChangeJobMode(ctx, job.UUID, false), "interrupted")
	require.Contains(t, a.retries, job.UUID)
	manager.swfClient = clients.SwfClient()
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, true))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, live.Spec.Enabled)
	require.NotContains(t, a.retries, job.UUID)
}

type testLegacyWorkflowPager struct {
	call int
	mode string
	t    *testing.T
}

func (p *testLegacyWorkflowPager) ListPage(_ context.Context, opts metav1.ListOptions) (util.ExecutionSpecList, string, error) {
	require.EqualValues(p.t, 100, opts.Limit)
	if p.call > 0 {
		require.Equal(p.t, fmt.Sprint(p.call), opts.Continue)
	}
	p.call++
	if p.mode == "error" {
		return nil, "", fmt.Errorf("interrupted inventory")
	}
	if p.mode == "empty" {
		return nil, "1", nil
	}
	if p.mode == "repeat" {
		return make(util.ExecutionSpecList, 1), "1", nil
	}
	next := fmt.Sprint(p.call)
	if (p.mode == "complete" && p.call == 2) || (p.mode == "large" && p.call == 12) {
		next = ""
	}
	return make(util.ExecutionSpecList, 100), next, nil
}
func TestLegacyWorkflowInventoryIsCompleteAndBounded(t *testing.T) {
	for _, mode := range []string{"complete", "large", "empty", "repeat", "error"} {
		t.Run(mode, func(t *testing.T) {
			pager := &testLegacyWorkflowPager{mode: mode, t: t}
			count := 0
			err := visitLegacyWorkflowInventory(context.Background(), pager, func(util.ExecutionSpec) error { count++; return nil })
			if mode == "complete" || mode == "large" {
				require.NoError(t, err)
				if mode == "large" {
					require.Equal(t, 1200, count)
				} else {
					require.Equal(t, 200, count)
				}
			} else {
				require.Error(t, err)
			}
			require.LessOrEqual(t, pager.call, 12)
		})
	}
}

// Page failures must not turn a partial inventory into trusted SQL progress.
type interruptedInventoryClient struct {
	util.ExecutionClient
	pager *interruptedInventoryPager
}

func (c *interruptedInventoryClient) Execution(string) util.ExecutionInterface { return c.pager }

type interruptedInventoryPager struct {
	util.ExecutionInterface
	calls int
	fail  bool
}

func (p *interruptedInventoryPager) ListPage(ctx context.Context, opts metav1.ListOptions) (util.ExecutionSpecList, string, error) {
	p.calls++
	if opts.Continue == "last" {
		if p.fail {
			return nil, "", fmt.Errorf("late inventory interruption")
		}
		return make(util.ExecutionSpecList, 100), "", nil
	}
	// Irrelevant namespace history still has to be traversed completely.
	next := fmt.Sprint(p.calls)
	if p.calls%12 == 11 {
		next = "last"
	}
	return make(util.ExecutionSpecList, 100), next, ctx.Err()
}
func TestOnlineAdoptionLargeInventoryRetryCommitsOnlyCompleteEvidence(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, false)
	ctx := context.Background()
	swf, err := clients.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(1200)
	db, err := clients.TransferDB()
	require.NoError(t, err)
	for index := int64(1); index <= 1200; index++ {
		run := &model.Run{UUID: fmt.Sprintf("retained-%04d", index), RecurringRunId: job.UUID, Namespace: job.Namespace,
			DisplayName: legacyRecurringRunRequestKey(swf, index), K8SName: fmt.Sprintf("workflow-%04d", index),
			RunDetails: model.RunDetails{ScheduledAtInSec: 90, CreatedAtInSec: 90, State: model.RuntimeStateSucceeded}}
		require.NoError(t, db.Create(run).Error)
	}
	pager := &interruptedInventoryPager{ExecutionInterface: clients.ExecClient().Execution(job.Namespace), fail: true}
	manager.execClient = &interruptedInventoryClient{ExecutionClient: clients.ExecClient(), pager: pager}
	require.ErrorContains(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID), "late inventory interruption")
	require.Equal(t, 12, pager.calls)
	var count int64
	require.NoError(t, db.Model(&model.RecurringRunState{}).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Count(&count).Error)
	require.Zero(t, count)
	// A new process discards its partial scan; a complete retry uses all history.
	manager = NewResourceManager(clients, &ResourceManagerOptions{CollectMetrics: false})
	pager.calls, pager.fail = 0, false
	manager.execClient = &interruptedInventoryClient{ExecutionClient: clients.ExecClient(), pager: pager}
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.EqualValues(t, 1200, state.LastRunIndex)
	require.Equal(t, "retained-1200", state.LastRunUUID)
	require.Equal(t, 12, pager.calls)
	// After the atomic commit, retries use durable state without another scan.
	pager.fail = true
	require.NoError(t, manager.AdoptLegacyRecurringRun(ctx, job.UUID))
	require.Equal(t, 12, pager.calls)
}
