// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// adoptLegacyRecurringRunProgress accepts one offline snapshot. It never imports
// execution inputs from the CR or manufactures a run for an unpersisted workflow.
func adoptLegacyRecurringRunProgress(job *model.Job, swf *scheduledworkflow.ScheduledWorkflow, runs []*model.Run, workflows util.ExecutionSpecList, now int64) (*model.RecurringRunState, error) {
	if job == nil || swf == nil || job.UUID == "" || job.Namespace == "" || job.K8SName == "" ||
		string(swf.UID) != job.UUID || swf.Name != job.K8SName || swf.Namespace != job.Namespace {
		return nil, fmt.Errorf("recurring run and ScheduledWorkflow identities differ; restore the original scheduled workflow before adoption")
	}
	fail := func(format string, args ...any) (*model.RecurringRunState, error) {
		return nil, fmt.Errorf("recurring run %s: %s; reconcile legacy execution records before adoption", job.UUID, fmt.Sprintf(format, args...))
	}
	index := int64(0)
	if swf.Status.Trigger.LastIndex != nil {
		index = *swf.Status.Trigger.LastIndex
	}
	last := swf.Status.Trigger.LastTriggeredTime
	if index < 0 || index == math.MaxInt64 || (index == 0) != (last == nil) {
		return fail("invalid or incomplete scheduling progress")
	}
	state := &model.RecurringRunState{JobUUID: job.UUID}
	if index > 0 {
		if last.Unix() <= 0 || last.Unix() > now {
			return fail("last triggered time is invalid or in the future")
		}
		state.LastRunIndex = index
		state.LastScheduledAtInSec = last.Unix()
		// A historical run may have been deleted. This timestamp is used only
		// to acknowledge the retained baseline, never to create an execution.
		state.LastCreatedAtInSec = last.Unix()
		state.RequestKey = legacyRecurringRunRequestKey(swf, index)
		state.PipelineVersionID = job.PipelineVersionId
	}
	byID := make(map[string]*model.Run)
	byIndex := make(map[int64]*model.Run)
	for _, run := range runs {
		if run == nil || run.RecurringRunId != job.UUID || run.ImportedFrom != nil {
			continue
		}
		runIndex, err := legacyRecurringRunPersistedIndex(swf, run)
		if err != nil && legacyRecurringRunIsTerminal(run) && run.ScheduledAtInSec > 0 && run.ScheduledAtInSec < state.LastScheduledAtInSec {
			// Retained historical records are not required to reconstruct the
			// accepted baseline. They cannot introduce an unacknowledged tick.
			continue
		}
		if err != nil {
			return fail("run %s has no valid controller index: %v", run.UUID, err)
		}
		if run.UUID == "" || run.Namespace != job.Namespace || run.K8SName == "" {
			return fail("run %s has inconsistent execution identity", run.UUID)
		}
		if _, exists := byID[run.UUID]; exists {
			return fail("duplicate run %s", run.UUID)
		}
		byID[run.UUID] = run
		if _, exists := byIndex[runIndex]; exists {
			return fail("multiple runs claim index %d", runIndex)
		}
		byIndex[runIndex] = run
		if runIndex > index+1 {
			return fail("run %s skips scheduling indices", run.UUID)
		}
		if run.ScheduledAtInSec <= 0 || run.ScheduledAtInSec > now || run.CreatedAtInSec <= 0 || run.CreatedAtInSec > now {
			return fail("run %s has invalid execution timestamps", run.UUID)
		}
		if runIndex == index && run.ScheduledAtInSec != state.LastScheduledAtInSec && run.ScheduledAtInSec != run.CreatedAtInSec {
			return fail("run %s conflicts with the last triggered time", run.UUID)
		}
		if runIndex < index && run.ScheduledAtInSec > state.LastScheduledAtInSec && run.ScheduledAtInSec != run.CreatedAtInSec {
			return fail("run %s conflicts with earlier scheduling progress", run.UUID)
		}
	}
	seenIndices := make(map[int64]bool)
	for _, workflow := range workflows {
		if workflow == nil {
			continue
		}
		meta := workflow.ExecutionObjectMeta()
		owner := metav1.GetControllerOf(meta)
		owned := owner != nil && string(owner.UID) == job.UUID
		labeled := workflow.ExecutionNamespace() == job.Namespace && meta.Labels[util.LabelKeyWorkflowScheduledWorkflowName] == job.K8SName
		if !owned && !labeled {
			continue
		}
		if !owned || owner.Kind != "ScheduledWorkflow" || owner.Name != job.K8SName || workflow.ExecutionNamespace() != job.Namespace || !labeled {
			return fail("workflow %s has conflicting schedule identity", workflow.ExecutionName())
		}
		workflowIndex, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowIndex])
		if err != nil || workflowIndex <= 0 || workflowIndex > index+1 {
			return fail("workflow %s has an invalid or skipped index", workflow.ExecutionName())
		}
		if seenIndices[workflowIndex] {
			return fail("multiple workflows claim index %d", workflowIndex)
		}
		seenIndices[workflowIndex] = true
		epoch, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowEpoch])
		if err != nil || epoch <= 0 || epoch > now {
			return fail("workflow %s has an invalid scheduled time", workflow.ExecutionName())
		}
		run := byID[meta.Labels[util.LabelKeyWorkflowRunId]]
		if run == nil {
			if !workflow.ExecutionStatus().IsInFinalState() || workflowIndex > index {
				return fail("workflow %s has no persisted run", workflow.ExecutionName())
			}
			if workflowIndex == index && epoch != state.LastScheduledAtInSec && epoch != meta.CreationTimestamp.Unix() {
				return fail("workflow %s conflicts with the last triggered time", workflow.ExecutionName())
			}
			// Retention may remove completed historical run rows.
			continue
		}
		if run.K8SName != workflow.ExecutionName() || byIndex[workflowIndex] != run || run.ScheduledAtInSec != epoch {
			return fail("workflow %s conflicts with its persisted run", workflow.ExecutionName())
		}
		if !workflow.ExecutionStatus().IsInFinalState() && legacyRecurringRunIsTerminal(run) {
			return fail("active workflow %s has a terminal persisted run and would escape concurrency accounting", workflow.ExecutionName())
		}
	}
	if run := byIndex[index]; run != nil {
		state.LastRunUUID = run.UUID
		state.LastCreatedAtInSec = run.CreatedAtInSec
		state.PipelineVersionID = run.PipelineVersionId
	}
	if run := byIndex[index+1]; run != nil {
		scheduledAt := run.ScheduledAtInSec
		if scheduledAt == run.CreatedAtInSec {
			// Older API requests recorded creation time as scheduled time. Recover
			// the actual due tick so catch-up does not skip the remaining backlog.
			var err error
			scheduledAt, err = legacyRecurringRunDueTime(job, swf, state, run.CreatedAtInSec)
			if err != nil {
				return fail("cannot recover the due time of run %s: %v", run.UUID, err)
			}
		}
		if scheduledAt <= state.LastScheduledAtInSec {
			return fail("unacknowledged run %s does not advance scheduled time", run.UUID)
		}
		state.LastRunUUID = run.UUID
		state.LastRunIndex = index + 1
		state.LastScheduledAtInSec = scheduledAt
		state.LastCreatedAtInSec = run.CreatedAtInSec
		state.RequestKey = legacyRecurringRunRequestKey(swf, index+1)
		state.PipelineVersionID = run.PipelineVersionId
	}
	return state, nil
}

func legacyRecurringRunDueTime(job *model.Job, swf *scheduledworkflow.ScheduledWorkflow, state *model.RecurringRunState, createdAt int64) (int64, error) {
	if !job.CronSchedule.IsEmpty() && !job.PeriodicSchedule.IsEmpty() || job.IntervalSecond != nil && *job.IntervalSecond < 1 {
		return 0, fmt.Errorf("stored schedule is invalid")
	}
	schedule, err := template.NewGenericScheduledWorkflow(job)
	if err != nil {
		return 0, err
	}
	schedule.CreationTimestamp = swf.CreationTimestamp
	if schedule.CreationTimestamp.IsZero() {
		schedule.CreationTimestamp = metav1.NewTime(time.Unix(job.CreatedAtInSec, 0))
	}
	if schedule.CreationTimestamp.Unix() <= 0 {
		return 0, fmt.Errorf("schedule creation time is missing")
	}
	if state.LastRunIndex > 0 {
		last := metav1.NewTime(time.Unix(state.LastScheduledAtInSec, 0))
		schedule.Status.Trigger.LastTriggeredTime = &last
	}
	// A schedule may have been disabled after this execution was submitted.
	// Reconstruct its historical due time without changing the stored enablement.
	schedule.Spec.Enabled = true
	location, err := scheduleutil.GetLocation()
	if err != nil {
		return 0, err
	}
	dueAt, due := scheduleutil.NewScheduledWorkflow(schedule).GetNextScheduledEpoch(0, createdAt, *location)
	if !due || dueAt <= 0 {
		return 0, fmt.Errorf("stored schedule had no tick due when the run was created")
	}
	return dueAt, nil
}

func legacyRecurringRunRequestKey(swf *scheduledworkflow.ScheduledWorkflow, index int64) string {
	copy := swf.DeepCopy()
	copy.Status.Trigger.LastIndex = util.Int64Pointer(index - 1)
	return scheduleutil.NewScheduledWorkflow(copy).NextResourceName()
}

func legacyRecurringRunIndex(swf *scheduledworkflow.ScheduledWorkflow, requestKey string) (int64, error) {
	prefix := swf.Name + "-"
	if !strings.HasPrefix(requestKey, prefix) {
		return 0, fmt.Errorf("unexpected request key prefix")
	}
	parts := strings.Split(strings.TrimPrefix(requestKey, prefix), "-")
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected request key format")
	}
	index, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || index <= 0 || index == math.MaxInt64 || legacyRecurringRunRequestKey(swf, index) != requestKey {
		return 0, fmt.Errorf("invalid request key index or checksum")
	}
	return index, nil
}

func legacyRecurringRunPersistedIndex(swf *scheduledworkflow.ScheduledWorkflow, run *model.Run) (int64, error) {
	if index, err := legacyRecurringRunIndex(swf, run.DisplayName); err == nil {
		return index, nil
	}
	// Persistence-agent recovery records the compiled workflow name as the
	// display name. Its retained execution still carries the controller index.
	manifest := run.WorkflowRuntimeManifest
	if manifest == "" {
		manifest = run.WorkflowSpecManifest
	}
	execution, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(manifest))
	if err != nil {
		return 0, err
	}
	meta := execution.ExecutionObjectMeta()
	owner := metav1.GetControllerOf(meta)
	if execution.ExecutionName() != run.K8SName || execution.ExecutionNamespace() != run.Namespace ||
		owner == nil || owner.UID != swf.UID || owner.Name != swf.Name || owner.Kind != "ScheduledWorkflow" ||
		meta.Labels[util.LabelKeyWorkflowRunId] != run.UUID || meta.Labels[util.LabelKeyWorkflowScheduledWorkflowName] != swf.Name {
		return 0, fmt.Errorf("persisted execution identity differs")
	}
	epoch, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowEpoch])
	if err != nil || epoch != run.ScheduledAtInSec {
		return 0, fmt.Errorf("persisted execution time differs")
	}
	index, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowIndex])
	if err != nil || index <= 0 || index == math.MaxInt64 {
		return 0, fmt.Errorf("persisted execution index is invalid")
	}
	return index, nil
}

func legacyRecurringRunIsTerminal(run *model.Run) bool {
	state := run.State
	if state == "" {
		state = model.RuntimeState(run.Conditions)
	}
	switch state.ToV2() {
	case model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateSkipped, model.RuntimeStateCanceled:
		return true
	default:
		return false
	}
}
