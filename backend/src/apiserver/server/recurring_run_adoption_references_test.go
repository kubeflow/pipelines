// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRecurringRunAdoptionPreservesReferenceBackedActiveRun(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	review.controllerAllowed = true
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	server := createRunServer(manager)
	first, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
		DisplayName: scheduleutil.NewScheduledWorkflow(swf).NextResourceName(), RecurringRunId: job.UUID,
	}})
	require.NoError(t, err)
	legacy, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	// Legacy associations and namespace are carried by references, not columns.
	_, err = clients.DB().Exec(`UPDATE run_details SET "JobUUID" = '', "Namespace" = '', "ExperimentUUID" = '' WHERE "UUID" = ?`, first.RunId)
	require.NoError(t, err)
	resolved, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	require.Equal(t, job.UUID, resolved.RecurringRunId)
	require.Equal(t, job.Namespace, resolved.Namespace)
	markScheduleAsLegacy(t, clients, job.UUID)
	receipt, err := manager.AdoptLegacyRecurringRuns(ctx)
	require.NoError(t, err)
	require.True(t, receipt.Ready)
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.LastRunIndex)
	require.Equal(t, first.RunId, state.LastRunUUID)
	var rawJob, rawNamespace, rawExperiment string
	require.NoError(t, clients.DB().QueryRow(`SELECT "JobUUID", "Namespace", "ExperimentUUID" FROM run_details WHERE "UUID" = ?`, first.RunId).
		Scan(&rawJob, &rawNamespace, &rawExperiment))
	require.Empty(t, rawJob)
	require.Empty(t, rawNamespace)
	require.Empty(t, rawExperiment)

	advanceScheduledRunClock(clients, 200)
	swf, err = swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	next := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: scheduleutil.NewScheduledWorkflow(swf).NextResourceName(), RecurringRunId: job.UUID,
	}}
	_, err = server.CreateRun(ctx, next)
	require.ErrorContains(t, err, "maximum concurrency")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	// Completion releases capacity without requiring historical-row normalization.
	_, err = clients.DB().Exec(`UPDATE run_details SET "State" = ? WHERE "UUID" = ?`, model.RuntimeStateSucceeded, first.RunId)
	require.NoError(t, err)
	executions := clients.ExecClientFake.Execution(job.Namespace)
	execution, err := executions.Get(ctx, legacy.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	execution.(*util.Workflow).Status.Phase = workflowapi.WorkflowSucceeded
	_, err = executions.Update(ctx, execution, metav1.UpdateOptions{})
	require.NoError(t, err)
	second, err := server.CreateRun(ctx, next)
	require.NoError(t, err)
	require.NotEqual(t, first.RunId, second.RunId)
	requireScheduledRunParameters(t, manager, second, 2, 110)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())
	// The adopted row remains replayable after durable progress moves beyond it.
	replayed, err := server.CreateRun(ctx, &api.CreateRunRequest{Run: &api.Run{
		DisplayName: first.DisplayName, RecurringRunId: job.UUID,
	}})
	require.NoError(t, err)
	require.Equal(t, first.RunId, replayed.RunId)
	require.Equal(t, 2, clients.ExecClientFake.GetWorkflowCount())
}
