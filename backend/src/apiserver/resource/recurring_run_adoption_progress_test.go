// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"math"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func legacyAdoptionFixture() (*model.Job, *scheduledworkflow.ScheduledWorkflow, *model.Run, *util.Workflow) {
	job := &model.Job{UUID: "schedule-uid", Namespace: "ns", K8SName: "legacy-schedule", Enabled: true,
		PipelineSpec: model.PipelineSpec{PipelineVersionId: "stored-version"}}
	swf := &scheduledworkflow.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{
		UID: types.UID(job.UUID), Name: job.K8SName, Namespace: job.Namespace,
	}}
	swf.Status.Trigger.LastIndex = util.Int64Pointer(3)
	last := metav1.NewTime(time.Unix(100, 0))
	swf.Status.Trigger.LastTriggeredTime = &last
	run := &model.Run{
		UUID: "legacy-random-run-uuid", Namespace: job.Namespace, RecurringRunId: job.UUID,
		DisplayName: legacyRecurringRunRequestKey(swf, 3), K8SName: "compiler-generated-workflow-name",
		PipelineSpec: model.PipelineSpec{PipelineVersionId: "executed-version"},
		RunDetails:   model.RunDetails{ScheduledAtInSec: 100, CreatedAtInSec: 110, State: model.RuntimeStateRunning},
	}
	wf := util.NewWorkflow(&workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{
		Name: run.K8SName, Namespace: job.Namespace, UID: "workflow-uid",
		CreationTimestamp: metav1.NewTime(time.Unix(110, 0)),
	}, Status: workflowapi.WorkflowStatus{Phase: workflowapi.WorkflowRunning}})
	wf.SetOwnerReferences(swf)
	wf.SetCannonicalLabels(swf.Name, 100, 3)
	wf.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	return job, swf, run, wf
}

func TestAdoptLegacyRecurringRunProgressPreservesBaseline(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	before := swf.DeepCopy()
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, &model.RecurringRunState{JobUUID: job.UUID, LastRunIndex: 3,
		LastScheduledAtInSec: 100, LastCreatedAtInSec: 110, RequestKey: run.DisplayName,
		PipelineVersionID: "executed-version", LastRunUUID: run.UUID}, state)
	require.NotEqual(t, run.DisplayName, run.K8SName)
	require.Equal(t, before, swf)
	require.Equal(t, "legacy-random-run-uuid", run.UUID)
}

func TestAdoptLegacyRecurringRunProgressRetainedBaseline(t *testing.T) {
	job, swf, _, _ := legacyAdoptionFixture()
	job.Enabled = false
	swf.Spec.Enabled = true
	swf.Spec.PipelineVersionId = "untrusted-cr-version"
	state, err := adoptLegacyRecurringRunProgress(job, swf, nil, nil, 200)
	require.NoError(t, err)
	require.Equal(t, int64(3), state.LastRunIndex)
	require.Equal(t, int64(100), state.LastScheduledAtInSec)
	require.Equal(t, "stored-version", state.PipelineVersionID)
	require.False(t, state.Pending)
	require.False(t, job.Enabled)
}

func TestAdoptLegacyRecurringRunProgressNeverStarted(t *testing.T) {
	for _, explicitZero := range []bool{false, true} {
		job, swf, _, _ := legacyAdoptionFixture()
		swf.Status.Trigger.LastIndex = nil
		if explicitZero {
			swf.Status.Trigger.LastIndex = util.Int64Pointer(0)
		}
		swf.Status.Trigger.LastTriggeredTime = nil
		state, err := adoptLegacyRecurringRunProgress(job, swf, nil, nil, 200)
		require.NoError(t, err)
		require.Equal(t, &model.RecurringRunState{JobUUID: job.UUID}, state)
	}
}

func TestAdoptLegacyRecurringRunProgressUnacknowledgedRun(t *testing.T) {
	for _, retainedWorkflow := range []bool{false, true} {
		job, swf, run, wf := legacyAdoptionFixture()
		run.DisplayName = legacyRecurringRunRequestKey(swf, 4)
		run.ScheduledAtInSec = 120
		run.CreatedAtInSec = 125
		wf.SetCannonicalLabels(swf.Name, 120, 4)
		var workflows util.ExecutionSpecList
		if retainedWorkflow {
			workflows = util.ExecutionSpecList{wf}
		}
		state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, workflows, 200)
		require.NoError(t, err)
		require.Equal(t, int64(4), state.LastRunIndex)
		require.Equal(t, int64(120), state.LastScheduledAtInSec)
		require.Equal(t, int64(125), state.LastCreatedAtInSec)
		require.Equal(t, legacyRecurringRunRequestKey(swf, 4), state.RequestKey)
		require.False(t, state.Pending)
		require.Equal(t, run.UUID, state.LastRunUUID)
	}
}

func TestAdoptLegacyRecurringRunProgressUnacknowledgedFirstRun(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	swf.Status.Trigger.LastIndex = nil
	swf.Status.Trigger.LastTriggeredTime = nil
	run.DisplayName = legacyRecurringRunRequestKey(swf, 1)
	wf.SetCannonicalLabels(swf.Name, 100, 1)
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, int64(1), state.LastRunIndex)
	require.Equal(t, int64(100), state.LastScheduledAtInSec)
	require.Equal(t, run.DisplayName, state.RequestKey)
	_, err = adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.ErrorContains(t, err, "no persisted run")
}

func TestAdoptLegacyRecurringRunProgressReporterRecoveredRun(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	job.IntervalSecond = util.Int64Pointer(20)
	swf.CreationTimestamp = metav1.NewTime(time.Unix(50, 0))
	run.DisplayName = run.K8SName
	run.ScheduledAtInSec = 120
	run.CreatedAtInSec = 125
	wf.CreationTimestamp = metav1.NewTime(time.Unix(125, 0))
	wf.SetCannonicalLabels(swf.Name, 120, 4)
	run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, int64(4), state.LastRunIndex)
	require.EqualValues(t, 120, state.LastScheduledAtInSec)
	require.Equal(t, legacyRecurringRunRequestKey(swf, 4), state.RequestKey)
	require.NotEqual(t, run.DisplayName, state.RequestKey)
	require.Equal(t, run.UUID, state.LastRunUUID)
}

func TestAdoptLegacyRecurringRunProgressIgnoresImportedAndOldHistory(t *testing.T) {
	job, swf, run, _ := legacyAdoptionFixture()
	origin := "another-cluster"
	run.ImportedFrom = &origin
	run.DisplayName = "imported-display-name"
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, nil, 200)
	require.NoError(t, err)
	require.Equal(t, int64(3), state.LastRunIndex)
	run.ImportedFrom = nil
	run.State = model.RuntimeStateSucceeded
	run.ScheduledAtInSec = 90
	state, err = adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, nil, 200)
	require.NoError(t, err)
	require.Equal(t, int64(3), state.LastRunIndex)
}

func TestAdoptLegacyRecurringRunProgressLegacyCreationTimeFallback(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	// Old API requests omitted ScheduledAt, so both the row and workflow label
	// contain creation time. The accepted CR still retains the actual due time.
	run.ScheduledAtInSec = run.CreatedAtInSec
	wf.SetCannonicalLabels(swf.Name, run.CreatedAtInSec, 3)
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, int64(100), state.LastScheduledAtInSec)
}

func TestAdoptLegacyRecurringRunProgressRecoversUnacknowledgedDueTime(t *testing.T) {
	for _, first := range []bool{false, true} {
		for _, noCatchup := range []bool{false, true} {
			job, swf, run, wf := legacyAdoptionFixture()
			job.NoCatchup = noCatchup
			job.IntervalSecond = util.Int64Pointer(10)
			job.CreatedAtInSec = 50
			swf.CreationTimestamp = metav1.NewTime(time.Unix(100, 0))
			// Only SQL inputs determine the due time, even for a disabled job.
			job.Enabled = false
			swf.Spec.PeriodicSchedule = &scheduledworkflow.PeriodicSchedule{IntervalSecond: 500}
			swf.Spec.NoCatchup = util.BoolPointer(!noCatchup)
			nextIndex := int64(4)
			if first {
				swf.Status.Trigger.LastIndex = nil
				swf.Status.Trigger.LastTriggeredTime = nil
				nextIndex = 1
			}
			run.DisplayName = legacyRecurringRunRequestKey(swf, nextIndex)
			run.ScheduledAtInSec, run.CreatedAtInSec = 200, 200
			wf.SetCannonicalLabels(swf.Name, 200, nextIndex)
			state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
			require.NoError(t, err)
			require.Equal(t, nextIndex, state.LastRunIndex)
			require.Equal(t, run.UUID, state.LastRunUUID)
			wantDue := int64(110)
			if noCatchup {
				wantDue = 200
			}
			require.Equal(t, wantDue, state.LastScheduledAtInSec)
			if !noCatchup {
				last := state.LastScheduledAtInSec
				next := scheduleutil.NewPeriodicSchedule(&scheduledworkflow.PeriodicSchedule{IntervalSecond: 10}).GetNextScheduledEpoch(&last, 100)
				require.Equal(t, int64(120), next)
			}
			require.False(t, job.Enabled)
		}
	}
}

func TestAdoptLegacyRecurringRunProgressRejectsInvalidFallbackSchedule(t *testing.T) {
	for _, interval := range []int64{-1, 500} {
		job, swf, run, wf := legacyAdoptionFixture()
		job.IntervalSecond = util.Int64Pointer(interval)
		swf.CreationTimestamp = metav1.NewTime(time.Unix(50, 0))
		run.DisplayName = legacyRecurringRunRequestKey(swf, 4)
		run.ScheduledAtInSec, run.CreatedAtInSec = 200, 200
		wf.SetCannonicalLabels(swf.Name, 200, 4)
		_, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
		require.ErrorContains(t, err, "cannot recover the due time")
	}
}

func TestAdoptLegacyRecurringRunProgressLegacyCronZeroPeriodicFields(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	cron := "0 * * * * *"
	job.Cron = &cron
	// The 2.17 reporter writes zero (not NULL) for unused periodic fields.
	job.PeriodicSchedule = model.PeriodicSchedule{
		PeriodicScheduleStartTimeInSec: util.Int64Pointer(0),
		PeriodicScheduleEndTimeInSec:   util.Int64Pointer(0),
		IntervalSecond:                 util.Int64Pointer(0),
	}
	swf.CreationTimestamp = metav1.NewTime(time.Unix(50, 0))
	run.DisplayName = legacyRecurringRunRequestKey(swf, 4)
	run.ScheduledAtInSec, run.CreatedAtInSec = 200, 200
	wf.SetCannonicalLabels(swf.Name, 200, 4)
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
	require.NoError(t, err)
	require.EqualValues(t, 4, state.LastRunIndex)
	require.EqualValues(t, 120, state.LastScheduledAtInSec)
	require.Equal(t, run.UUID, state.LastRunUUID)

	// A genuinely active but incomplete periodic schedule must still reject.
	job.Cron = nil
	job.PeriodicScheduleStartTimeInSec = util.Int64Pointer(50)
	_, err = adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
	require.ErrorContains(t, err, "stored schedule is invalid")
}

func TestAdoptLegacyRecurringRunProgressRetainedWorkflowPrecreationEpoch(t *testing.T) {
	job, swf, _, wf := legacyAdoptionFixture()
	wf.Status.Phase = workflowapi.WorkflowSucceeded
	// The API stamps epoch before submitting the workflow to Kubernetes. Neither
	// that timestamp nor creationTimestamp is the acknowledged trigger time.
	wf.SetCannonicalLabels(swf.Name, 110, 3)
	wf.CreationTimestamp = metav1.NewTime(time.Unix(115, 0))
	state, err := adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.EqualValues(t, 3, state.LastRunIndex)
	require.EqualValues(t, 100, state.LastScheduledAtInSec)
	require.Empty(t, state.LastRunUUID)

	wf.Status.Phase = workflowapi.WorkflowRunning
	_, err = adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.ErrorContains(t, err, "no persisted run")
	wf.Status.Phase = workflowapi.WorkflowSucceeded
	wf.SetCannonicalLabels(swf.Name, 110, 4)
	_, err = adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.ErrorContains(t, err, "no persisted run")
	wf.SetCannonicalLabels(swf.Name, 201, 3)
	_, err = adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.ErrorContains(t, err, "invalid scheduled time")
}

func TestAdoptLegacyRecurringRunProgressRecoveredBaselinePrecreationEpoch(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	run.DisplayName = run.K8SName
	run.ScheduledAtInSec, run.CreatedAtInSec = 110, 115
	wf.SetCannonicalLabels(swf.Name, 110, 3)
	wf.CreationTimestamp = metav1.NewTime(time.Unix(115, 0))
	run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
	require.NoError(t, err)
	require.EqualValues(t, 3, state.LastRunIndex)
	require.EqualValues(t, 100, state.LastScheduledAtInSec)
	require.EqualValues(t, 115, state.LastCreatedAtInSec)
	require.Equal(t, run.UUID, state.LastRunUUID)

	// Timestamp compatibility does not relax the persisted execution identity.
	wf.OwnerReferences[0].UID = "another-schedule"
	run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
	_, err = adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, nil, 210)
	require.ErrorContains(t, err, "persisted execution identity differs")
}

func TestAdoptLegacyRecurringRunProgressRecoveredRequestName(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	run.K8SName = run.DisplayName
	wf.Name = run.K8SName
	run.ScheduledAtInSec, run.CreatedAtInSec = 90, 99
	wf.SetCannonicalLabels(swf.Name, 90, 3)
	wf.CreationTimestamp = metav1.NewTime(time.Unix(99, 0))
	run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
	state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
	require.NoError(t, err)
	require.EqualValues(t, 3, state.LastRunIndex)
	require.EqualValues(t, 100, state.LastScheduledAtInSec)
	require.EqualValues(t, 99, state.LastCreatedAtInSec)
	require.Equal(t, run.UUID, state.LastRunUUID)
	for _, mutation := range []func(*util.Workflow){
		func(w *util.Workflow) { w.Name = "other" },
		func(w *util.Workflow) { w.Namespace = "other" },
		func(w *util.Workflow) { w.SetCannonicalLabels(swf.Name, 91, 3) },
		func(w *util.Workflow) { w.OwnerReferences[0].UID = "other" },
		func(w *util.Workflow) { w.SetCannonicalLabels(swf.Name, 90, 4) },
		func(w *util.Workflow) { w.CreationTimestamp = metav1.NewTime(time.Unix(120, 0)) },
		func(w *util.Workflow) { w.SetLabels(util.LabelKeyWorkflowRunId, "other") },
	} {
		changed := util.NewWorkflow(wf.DeepCopy())
		mutation(changed)
		run.WorkflowRuntimeManifest = model.LargeText(changed.ToStringForStore())
		_, err = adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, nil, 210)
		require.Error(t, err)
	}
}

func TestAdoptLegacyRecurringRunProgressEmbeddedPendingPreservesDue(t *testing.T) {
	for _, noCatchup := range []bool{false, true} {
		for _, created := range []int64{135, 136} {
			job, swf, run, wf := legacyAdoptionFixture()
			job.NoCatchup = noCatchup
			job.IntervalSecond = util.Int64Pointer(30)
			run.DisplayName = legacyRecurringRunRequestKey(swf, 4)
			run.K8SName = run.DisplayName
			wf.Name = run.K8SName
			run.ScheduledAtInSec, run.CreatedAtInSec = 135, created
			wf.SetCannonicalLabels(swf.Name, 135, 4)
			wf.CreationTimestamp = metav1.NewTime(time.Unix(created, 0))
			run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
			state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
			require.NoError(t, err)
			require.EqualValues(t, 4, state.LastRunIndex)
			require.EqualValues(t, 135, state.LastScheduledAtInSec)
			require.EqualValues(t, created, state.LastCreatedAtInSec)
		}
	}
}

func TestAdoptLegacyRecurringRunProgressRecoveredUnacknowledgedPrecreationEpoch(t *testing.T) {
	for _, noCatchup := range []bool{false, true} {
		t.Run(map[bool]string{false: "catchup", true: "no catchup"}[noCatchup], func(t *testing.T) {
			job, swf, run, wf := legacyAdoptionFixture()
			job.NoCatchup = noCatchup
			job.IntervalSecond = util.Int64Pointer(10)
			swf.CreationTimestamp = metav1.NewTime(time.Unix(50, 0))
			run.DisplayName = run.K8SName
			run.ScheduledAtInSec, run.CreatedAtInSec = 200, 205
			wf.SetCannonicalLabels(swf.Name, 200, 4)
			wf.CreationTimestamp = metav1.NewTime(time.Unix(205, 0))
			run.WorkflowRuntimeManifest = model.LargeText(wf.ToStringForStore())
			state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 210)
			require.NoError(t, err)
			wantDue := int64(110)
			if noCatchup {
				wantDue = 200
			}
			require.EqualValues(t, 4, state.LastRunIndex)
			require.Equal(t, wantDue, state.LastScheduledAtInSec)
			require.EqualValues(t, 205, state.LastCreatedAtInSec)
			require.Equal(t, run.UUID, state.LastRunUUID)
		})
	}
}

func TestAdoptLegacyRecurringRunProgressIgnoresUnrelatedWorkflows(t *testing.T) {
	job, swf, _, wf := legacyAdoptionFixture()
	wf.Namespace = "another-namespace"
	wf.OwnerReferences[0].UID = "another-schedule-uid"
	state, err := adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{nil, wf}, 200)
	require.NoError(t, err)
	require.Equal(t, int64(3), state.LastRunIndex)
}

func TestAdoptLegacyRecurringRunProgressAllowsDeletedHistoricalRun(t *testing.T) {
	job, swf, _, wf := legacyAdoptionFixture()
	wf.Status.Phase = workflowapi.WorkflowSucceeded
	state, err := adoptLegacyRecurringRunProgress(job, swf, nil, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, int64(3), state.LastRunIndex)
}

func TestAdoptLegacyRecurringRunProgressRejectsInconsistentEvidence(t *testing.T) {
	tests := []struct {
		name string
		edit func(*model.Job, *scheduledworkflow.ScheduledWorkflow, *model.Run, *util.Workflow)
		want string
	}{
		{"different CR UID", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.UID = "replacement"
		}, "identities differ"},
		{"different CR namespace", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Namespace = "another"
		}, "identities differ"},
		{"different CR name", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Name = "replacement"
		}, "identities differ"},
		{"missing time", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Status.Trigger.LastTriggeredTime = nil
		}, "incomplete scheduling progress"},
		{"missing index", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Status.Trigger.LastIndex = nil
		}, "incomplete scheduling progress"},
		{"negative index", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Status.Trigger.LastIndex = util.Int64Pointer(-1)
		}, "incomplete scheduling progress"},
		{"exhausted index", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Status.Trigger.LastIndex = util.Int64Pointer(math.MaxInt64)
		}, "incomplete scheduling progress"},
		{"future baseline", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			tm := metav1.NewTime(time.Unix(201, 0))
			s.Status.Trigger.LastTriggeredTime = &tm
		}, "future"},
		{"missing baseline with runs", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, _ *model.Run, _ *util.Workflow) {
			s.Status.Trigger.LastIndex = nil
			s.Status.Trigger.LastTriggeredTime = nil
		}, "skips scheduling indices"},
		{"run namespace mismatch", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.Namespace = "another"
		}, "inconsistent execution identity"},
		{"run request key mismatch", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.DisplayName = "invalid"
		}, "no valid controller index"},
		{"run index gap", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.DisplayName = legacyRecurringRunRequestKey(s, 5)
		}, "skips scheduling indices"},
		{"run future time", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.ScheduledAtInSec = 201
		}, "invalid execution timestamps"},
		{"run conflicting time", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.ScheduledAtInSec = 90
		}, "conflicts with the last triggered time"},
		{"workflow owner mismatch", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.OwnerReferences[0].UID = "replacement"
		}, "conflicting schedule identity"},
		{"workflow missing labels", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			delete(w.Labels, util.LabelKeyWorkflowScheduledWorkflowName)
		}, "conflicting schedule identity"},
		{"workflow invalid index", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Labels[util.LabelKeyWorkflowIndex] = "invalid"
		}, "invalid or skipped index"},
		{"workflow index gap", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Labels[util.LabelKeyWorkflowIndex] = "5"
		}, "invalid or skipped index"},
		{"workflow missing run", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Labels[util.LabelKeyWorkflowRunId] = "missing"
		}, "no persisted run"},
		{"workflow name mismatch", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Name = "other"
		}, "conflicts with its persisted run"},
		{"workflow epoch mismatch", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Labels[util.LabelKeyWorkflowEpoch] = "99"
		}, "conflicts with its persisted run"},
		{"workflow epoch invalid", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, _ *model.Run, w *util.Workflow) {
			w.Labels[util.LabelKeyWorkflowEpoch] = "invalid"
		}, "invalid scheduled time"},
		{"active workflow terminal DB", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.State = model.RuntimeStateSucceeded
		}, "escape concurrency accounting"},
		{"active workflow legacy terminal DB", func(_ *model.Job, _ *scheduledworkflow.ScheduledWorkflow, r *model.Run, _ *util.Workflow) {
			r.State = ""
			r.Conditions = model.LegacyStateDone
		}, "escape concurrency accounting"},
		{"unacknowledged time does not advance", func(_ *model.Job, s *scheduledworkflow.ScheduledWorkflow, r *model.Run, w *util.Workflow) {
			r.DisplayName = legacyRecurringRunRequestKey(s, 4)
			w.Labels[util.LabelKeyWorkflowIndex] = "4"
		}, "does not advance scheduled time"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, swf, run, wf := legacyAdoptionFixture()
			tt.edit(job, swf, run, wf)
			state, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf}, 200)
			require.ErrorContains(t, err, tt.want)
			require.Nil(t, state)
		})
	}
}

func TestAdoptLegacyRecurringRunProgressRejectsDuplicates(t *testing.T) {
	job, swf, run, wf := legacyAdoptionFixture()
	other := *run
	other.UUID = "another-run-uuid"
	_, err := adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run, &other}, nil, 200)
	require.ErrorContains(t, err, "multiple runs claim index")
	_, err = adoptLegacyRecurringRunProgress(job, swf, []*model.Run{run}, util.ExecutionSpecList{wf, wf}, 200)
	require.ErrorContains(t, err, "multiple workflows claim index")
}
