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

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func TestAdoptLegacyRecurringRunResumesInterruptedCutoverFromSQL(t *testing.T) {
	initEnvVars()
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	clients := NewFakeClientManagerOrFatal(util.NewFakeTime(time.Unix(100, 0)))
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := NewResourceManager(clients, &ResourceManagerOptions{CollectMetrics: false})
	ctx := context.Background()
	swf, err := clients.SwfClient().ScheduledWorkflow("ns1").Create(ctx, &swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "legacy"},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: true, ServiceAccount: "untrusted-cr-account"},
	})
	require.NoError(t, err)
	swf.Status.Trigger.LastIndex = util.Int64Pointer(3)
	last := metav1.NewTime(time.Unix(90, 0))
	swf.Status.Trigger.LastTriggeredTime = &last
	job, err := clients.JobStore().CreateJob(&model.Job{
		UUID: string(swf.UID), K8SName: swf.Name, Namespace: swf.Namespace,
		Enabled: true, MaxConcurrency: 1, ServiceAccount: "stored-account",
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{IntervalSecond: util.Int64Pointer(10)}},
	})
	require.NoError(t, err)
	db, err := clients.TransferDB()
	require.NoError(t, err)
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	require.NoError(t, db.Delete(&model.RecurringRunAdoption{}, &model.RecurringRunAdoption{ID: "legacy-2.18:" + job.UUID}).Error)
	interrupted := &adoptionInterruptedCR{ScheduledWorkflowInterface: clients.SwfClient().ScheduledWorkflow("ns1"), failUpdate: true}
	manager.swfClient = interrupted
	_, err = manager.AdoptLegacyRecurringRuns(ctx)
	require.ErrorContains(t, err, "interrupted Kubernetes update")
	receipt, err := storage.GetLegacyRecurringRunAdoption(db)
	require.NoError(t, err)
	require.False(t, receipt.Ready)
	require.ErrorContains(t, manager.RequireRecurringRunAdoptionReady(ctx, job.UUID), "adoption is incomplete")

	// A new process resumes the recorded inventory, even if CR progress changed
	// after the SQL commit. This is not a second adoption of the mutable CR.
	swf.Status.Trigger.LastIndex = util.Int64Pointer(99)
	later := metav1.NewTime(time.Unix(99, 0))
	swf.Status.Trigger.LastTriggeredTime = &later
	swf.Spec.ServiceAccount = "edited-again"
	restarted := NewResourceManager(clients, &ResourceManagerOptions{CollectMetrics: false})
	completed, err := restarted.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, completed.Ready)
	require.Equal(t, receipt.CompletedAt, completed.CompletedAt)
	require.NoError(t, restarted.requireRecurringRunAdoptionReady(ctx))
	current, err := clients.SwfClient().ScheduledWorkflow("ns1").Get(ctx, swf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.EqualValues(t, 3, *current.Status.Trigger.LastIndex)
	require.EqualValues(t, 90, current.Status.Trigger.LastTriggeredTime.Unix())
	require.Equal(t, "stored-account", current.Spec.ServiceAccount)
	require.True(t, current.Spec.Enabled)
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())

	// Completion is a permanent seal, not permission to reset deleted state.
	require.NoError(t, db.Delete(&model.RecurringRunState{}, &model.RecurringRunState{JobUUID: job.UUID}).Error)
	current.Status.Trigger.LastIndex = util.Int64Pointer(200)
	again, err := restarted.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.Equal(t, completed, again)
	_, err = clients.JobStore().GetRecurringRunState(job.UUID)
	require.ErrorContains(t, err, "no trusted scheduling state")
	require.EqualValues(t, 200, *current.Status.Trigger.LastIndex)
}
