// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartupRepairsReadySchedulesAndPreservesPendingTick(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint(pending), func(t *testing.T) {
			manager, clients := onlineAdoptionManager(t)
			job := onlineLegacyJob(t, clients, pending)
			ctx := context.Background()
			require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
			state, err := clients.JobStore().GetRecurringRunState(job.UUID)
			require.NoError(t, err)
			live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			beforeIndex := state.LastRunIndex
			live.Status.Trigger.LastIndex = util.Int64Pointer(beforeIndex)
			if pending {
				state, err = clients.JobStore().ClaimRecurringRun(job.UUID, "pending", beforeIndex, 200, 210, "")
				require.NoError(t, err)
			}
			live.Spec.ServiceAccount = "tampered"
			live.Spec.Enabled = !job.Enabled
			_, err = clients.SwfClient().ScheduledWorkflow("ns1").Update(ctx, live)
			require.NoError(t, err)
			ready := false
			worker := NewAutomaticRecurringRunSynchronization(manager, func(context.Context) error {
				if !ready {
					return fmt.Errorf("rolling")
				}
				return nil
			})
			require.Error(t, worker.scan(ctx))
			require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
			ready = true
			require.NoError(t, worker.scan(ctx))
			require.True(t, worker.startupDone)
			live, err = clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, job.ServiceAccount, live.Spec.ServiceAccount)
			require.Equal(t, job.Enabled, live.Spec.Enabled)
			require.Equal(t, beforeIndex, *live.Status.Trigger.LastIndex)
			actual, err := clients.JobStore().GetRecurringRunState(job.UUID)
			require.NoError(t, err)
			require.Equal(t, state, actual)
		})
	}
}
func TestStartupFailedReadyRepairIsRetriedFromDurablePending(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	live, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	live.Spec.ServiceAccount = "tampered"
	_, err = clients.SwfClient().ScheduledWorkflow("ns1").Update(ctx, live)
	require.NoError(t, err)
	manager.swfClient = &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	worker := NewAutomaticRecurringRunSynchronization(manager, func(context.Context) error { return nil })
	now := time.Unix(1000, 0)
	worker.now = func() time.Time { return now }
	require.NoError(t, worker.scan(ctx))
	require.True(t, worker.startupDone)
	require.Error(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	manager.swfClient = clients.SwfClient()
	now = now.Add(recurringRunRetryInitial)
	require.NoError(t, worker.scan(ctx))
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
	require.NotContains(t, worker.retries, job.UUID)
	// A deleted SQL state is never synthesized by the startup repair path.
	db, err := manager.recurringRunAdoptionDB(ctx)
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	_, err = manager.PrepareRecurringRunStartupRepairs(ctx, "", 100)
	require.NoError(t, err)
	require.Error(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	_, err = clients.JobStore().GetRecurringRunState(job.UUID)
	require.Error(t, err)
}

func TestStartupRepairSkipsUnchangedScheduledWorkflow(t *testing.T) {
	manager, clients := onlineAdoptionManager(t)
	job := onlineLegacyJob(t, clients, true)
	ctx := context.Background()
	_, err := manager.PrepareRecurringRunStartupRepairs(ctx, "", 100)
	require.NoError(t, err)
	require.NoError(t, manager.SynchronizeRecurringRun(ctx, job.UUID))
	// Reject every update: an unchanged CR must still finish its durable repair.
	manager.swfClient = &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	worker := NewAutomaticRecurringRunSynchronization(manager, func(context.Context) error { return nil })
	require.NoError(t, worker.scan(ctx))
	require.True(t, worker.startupDone)
	require.NoError(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID))
}
