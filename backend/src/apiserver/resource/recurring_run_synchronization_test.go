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

	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
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
	name := "adopted"
	swf, err := clients.SwfClient().ScheduledWorkflow("ns1").Create(context.Background(), &swfapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{GenerateName: name}, Spec: swfapi.ScheduledWorkflowSpec{Enabled: enabled, ServiceAccount: "untrusted"}})
	require.NoError(t, err)
	swf.UID = types.UID(name)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(0)
	last := metav1.NewTime(time.Unix(90, 0))
	swf.Status.Trigger.LastTriggeredTime = &last
	job, err := clients.JobStore().CreateJob(&model.Job{UUID: string(swf.UID), K8SName: swf.Name, Namespace: swf.Namespace, Enabled: enabled, ServiceAccount: "stored", MaxConcurrency: 1, Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{IntervalSecond: util.Int64Pointer(10)}}})
	require.NoError(t, err)
	return job
}

type adoptionInterruptedCR struct {
	swfclient.ScheduledWorkflowInterface
	failUpdate bool
}

func (c *adoptionInterruptedCR) ScheduledWorkflow(string) swfclient.ScheduledWorkflowInterface {
	return c
}

func (c *adoptionInterruptedCR) Update(ctx context.Context, swf *swfapi.ScheduledWorkflow) (*swfapi.ScheduledWorkflow, error) {
	if c.failUpdate {
		return nil, fmt.Errorf("interrupted Kubernetes update")
	}
	return c.ScheduledWorkflowInterface.Update(ctx, swf)
}

func TestOnlineModeFailureRemainsDurablyPending(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	manager.swfClient = &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	require.ErrorContains(t, manager.ChangeJobMode(ctx, job.UUID, false), "interrupted")
	current, err := clients.JobStore().GetJob(job.UUID)
	require.NoError(t, err)
	require.False(t, current.Enabled)
	require.ErrorContains(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID), "incomplete")
	pending, err := manager.ListPendingRecurringRunSynchronizations(ctx, "", 100)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	manager.swfClient = clients.SwfClient()
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, live.Spec.Enabled)
}

func TestOnlineModeDoesNotAcknowledgePendingTick(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	state, err := clients.JobStore().ClaimRecurringRun(job.UUID, "pending", 0, 100, 100, "")
	require.NoError(t, err)
	require.True(t, state.Pending)
	require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.EqualValues(t, 0, *live.Status.Trigger.LastIndex)
	require.False(t, live.Spec.Enabled)
	current, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, state, current)
}

func TestAdoptedHistoricalReplayPreservesIdentityAndDeletion(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	db, err := manager.transferDB()
	require.NoError(t, err)
	historical := &model.Run{UUID: "historical-uuid", DisplayName: "compiled-workflow-name", Namespace: job.Namespace, RecurringRunId: job.UUID, RunDetails: model.RunDetails{CreatedAtInSec: 90, ScheduledAtInSec: 80}}
	require.NoError(t, db.Create(historical).Error)
	state := &model.RecurringRunState{JobUUID: job.UUID, RequestKey: "controller-request", LastRunUUID: historical.UUID, LastRunIndex: 3, LastCreatedAtInSec: 90, LastScheduledAtInSec: 80}
	requested := &model.Run{Namespace: job.Namespace, DisplayName: state.RequestKey}
	retained, err := manager.getRetainedRecurringRunTick(requested, job, state)
	require.NoError(t, err)
	require.Equal(t, historical.UUID, retained.replay.UUID)
	require.NoError(t, db.Delete(historical).Error)
	retained, err = manager.getRetainedRecurringRunTick(requested, job, state)
	require.NoError(t, err)
	require.Equal(t, historical.UUID, retained.replay.UUID)
	var count int64
	require.NoError(t, db.Model(&model.Run{}).Where(&model.Run{UUID: historical.UUID}).Count(&count).Error)
	require.Zero(t, count)
}

func TestOnlineModeChangeSupersedesCachedReconciliationFailure(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	a := NewAutomaticRecurringRunSynchronization(manager, func(context.Context) error { return nil })
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
			db, err := manager.transferDB()
			require.NoError(t, err)
			var statesBefore []model.RecurringRunState
			var receiptsBefore []model.RecurringRunAdoption
			require.NoError(t, db.Find(&statesBefore).Error)
			require.NoError(t, db.Find(&receiptsBefore).Error)
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
			var states []model.RecurringRunState
			var receipts []model.RecurringRunAdoption
			require.NoError(t, db.Find(&states).Error)
			require.NoError(t, db.Find(&receipts).Error)
			require.Equal(t, statesBefore, states)
			require.Equal(t, receiptsBefore, receipts)
			var before int64
			require.NoError(t, db.Model(&model.Job{}).Count(&before).Error)
			newJob := &model.Job{DisplayName: "new", Namespace: "ns1", Enabled: true, PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld), RuntimeConfig: model.RuntimeConfig{Parameters: `{ "text": "world" }`}}}
			_, err = manager.CreateJob(ctx, newJob)
			assertNotApplied(err)
			require.Empty(t, newJob.UUID)
			var after int64
			require.NoError(t, db.Model(&model.Job{}).Count(&after).Error)
			require.Equal(t, before, after)
			require.NoError(t, db.Find(&states).Error)
			require.NoError(t, db.Find(&receipts).Error)
			require.Equal(t, statesBefore, states)
			require.Equal(t, receiptsBefore, receipts)
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
