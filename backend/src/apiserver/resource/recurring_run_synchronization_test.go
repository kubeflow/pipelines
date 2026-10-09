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
