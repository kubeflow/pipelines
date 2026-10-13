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
	progress, err := newLegacyRecurringRunProgress(job, swf, now)
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		if err := progress.addRun(run); err != nil {
			return nil, err
		}
	}
	for _, workflow := range workflows {
		if err := progress.addWorkflow(workflow); err != nil {
			return nil, err
		}
	}
	return progress.finish()
}

// Only identity and progress evidence survives inventory; execution manifests
// are validated one at a time and are never retained by the accumulator.
type legacyRecurringRunEvidence struct {
	UUID, K8SName, PipelineVersionID string
	ScheduledAtInSec, CreatedAtInSec int64
	terminal, recoverDue             bool
}

type legacyRecurringRunProgress struct {
	job         *model.Job
	swf         *scheduledworkflow.ScheduledWorkflow
	now, index  int64
	state       *model.RecurringRunState
	byID        map[string]*legacyRecurringRunEvidence
	byIndex     map[int64]*legacyRecurringRunEvidence
	seenIndices map[int64]bool
	err         error
}

func (p *legacyRecurringRunProgress) fail(format string, args ...any) error {
	p.err = fmt.Errorf("recurring run %s: %s; reconcile legacy execution records before adoption", p.job.UUID, fmt.Sprintf(format, args...))
	return p.err
}

func newLegacyRecurringRunProgress(job *model.Job, swf *scheduledworkflow.ScheduledWorkflow, now int64) (*legacyRecurringRunProgress, error) {
	if job == nil || swf == nil || job.UUID == "" || job.Namespace == "" || job.K8SName == "" ||
		string(swf.UID) != job.UUID || swf.Name != job.K8SName || swf.Namespace != job.Namespace {
		return nil, fmt.Errorf("recurring run and ScheduledWorkflow identities differ; restore the original scheduled workflow before adoption")
	}
	p := &legacyRecurringRunProgress{job: job, swf: swf, now: now, byID: make(map[string]*legacyRecurringRunEvidence), byIndex: make(map[int64]*legacyRecurringRunEvidence), seenIndices: make(map[int64]bool)}
	index := int64(0)
	if swf.Status.Trigger.LastIndex != nil {
		index = *swf.Status.Trigger.LastIndex
	}
	last := swf.Status.Trigger.LastTriggeredTime
	if index < 0 || index == math.MaxInt64 || (index == 0) != (last == nil) {
		return nil, p.fail("invalid or incomplete scheduling progress")
	}
	state := &model.RecurringRunState{JobUUID: job.UUID}
	if index > 0 {
		if last.Unix() <= 0 || last.Unix() > now {
			return nil, p.fail("last triggered time is invalid or in the future")
		}
		state.LastRunIndex = index
		state.LastScheduledAtInSec = last.Unix()
		// A historical run may have been deleted. This timestamp is used only
		// to acknowledge the retained baseline, never to create an execution.
		state.LastCreatedAtInSec = last.Unix()
		state.RequestKey = legacyRecurringRunRequestKey(swf, index)
		state.PipelineVersionID = job.PipelineVersionId
	}
	p.index, p.state = index, state
	return p, nil
}

// Supply the complete run inventory before supplying workflows.
func (p *legacyRecurringRunProgress) addRun(run *model.Run) error {
	if p.err != nil {
		return p.err
	}
	if run == nil || run.RecurringRunId != p.job.UUID || run.ImportedFrom != nil {
		return nil
	}
	runIndex, recovered, err := legacyRecurringRunPersistedIndex(p.swf, run)
	if err != nil && legacyRecurringRunIsTerminal(run) && run.ScheduledAtInSec > 0 && run.ScheduledAtInSec < p.state.LastScheduledAtInSec {
		// Retained historical records are not required to reconstruct the
		// accepted baseline. They cannot introduce an unacknowledged tick.
		return nil
	}
	if err != nil {
		return p.fail("run %s has no valid controller index: %v", run.UUID, err)
	}
	if run.UUID == "" || run.Namespace != p.job.Namespace || run.K8SName == "" {
		return p.fail("run %s has inconsistent execution identity", run.UUID)
	}
	if _, exists := p.byID[run.UUID]; exists {
		return p.fail("duplicate run %s", run.UUID)
	}
	if _, exists := p.byIndex[runIndex]; exists {
		return p.fail("multiple runs claim index %d", runIndex)
	}
	// Embedded legacy workflows use the request key as their actual name
	// and stamp the true due time. API-created workflows use a generated
	// name and stamp API creation time, which needs due-time recovery.
	_, requestErr := legacyRecurringRunIndex(p.swf, run.DisplayName)
	embedded := recovered && requestErr == nil
	recoverDue := !embedded && (recovered || run.ScheduledAtInSec == run.CreatedAtInSec)
	if runIndex > p.index+1 {
		return p.fail("run %s skips scheduling indices", run.UUID)
	}
	if run.ScheduledAtInSec <= 0 || run.ScheduledAtInSec > p.now || run.CreatedAtInSec <= 0 || run.CreatedAtInSec > p.now {
		return p.fail("run %s has invalid execution timestamps", run.UUID)
	}
	// Reporter recovery preserves the epoch label but records Kubernetes
	// creation time separately. Neither must equal the acknowledged due time.
	if !recovered && runIndex == p.index && run.ScheduledAtInSec != p.state.LastScheduledAtInSec && run.ScheduledAtInSec != run.CreatedAtInSec {
		return p.fail("run %s conflicts with the last triggered time", run.UUID)
	}
	if !recovered && runIndex < p.index && run.ScheduledAtInSec > p.state.LastScheduledAtInSec && run.ScheduledAtInSec != run.CreatedAtInSec {
		return p.fail("run %s conflicts with earlier scheduling progress", run.UUID)
	}
	p.byID[run.UUID] = &legacyRecurringRunEvidence{
		UUID: run.UUID, K8SName: run.K8SName, ScheduledAtInSec: run.ScheduledAtInSec,
		CreatedAtInSec: run.CreatedAtInSec, PipelineVersionID: run.PipelineVersionId,
		terminal: legacyRecurringRunIsTerminal(run), recoverDue: recoverDue,
	}
	p.byIndex[runIndex] = p.byID[run.UUID]
	return nil
}

func (p *legacyRecurringRunProgress) addWorkflow(workflow util.ExecutionSpec) error {
	if p.err != nil {
		return p.err
	}
	if workflow == nil {
		return nil
	}
	meta := workflow.ExecutionObjectMeta()
	owner := metav1.GetControllerOf(meta)
	owned := owner != nil && string(owner.UID) == p.job.UUID
	labeled := workflow.ExecutionNamespace() == p.job.Namespace && meta.Labels[util.LabelKeyWorkflowScheduledWorkflowName] == p.job.K8SName
	if !owned && !labeled {
		return nil
	}
	if !owned || owner.Kind != "ScheduledWorkflow" || owner.Name != p.job.K8SName || workflow.ExecutionNamespace() != p.job.Namespace || !labeled {
		return p.fail("workflow %s has conflicting schedule identity", workflow.ExecutionName())
	}
	workflowIndex, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowIndex])
	if err != nil || workflowIndex <= 0 || workflowIndex > p.index+1 {
		return p.fail("workflow %s has an invalid or skipped index", workflow.ExecutionName())
	}
	if p.seenIndices[workflowIndex] {
		return p.fail("multiple workflows claim index %d", workflowIndex)
	}
	p.seenIndices[workflowIndex] = true
	epoch, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowEpoch])
	if err != nil || epoch <= 0 || epoch > p.now {
		return p.fail("workflow %s has an invalid scheduled time", workflow.ExecutionName())
	}
	run := p.byID[meta.Labels[util.LabelKeyWorkflowRunId]]
	if run == nil {
		if !workflow.ExecutionStatus().IsInFinalState() || workflowIndex > p.index {
			return p.fail("workflow %s has no persisted run", workflow.ExecutionName())
		}
		// Retention may remove completed, acknowledged history. The legacy API
		// stamped epoch before Kubernetes creation, not at the trigger's due time.
		// Neither timestamp is needed to advance the accepted baseline here.
		return nil
	}
	if run.K8SName != workflow.ExecutionName() || p.byIndex[workflowIndex] != run || run.ScheduledAtInSec != epoch {
		return p.fail("workflow %s conflicts with its persisted run", workflow.ExecutionName())
	}
	if !workflow.ExecutionStatus().IsInFinalState() && run.terminal {
		return p.fail("active workflow %s has a terminal persisted run and would escape concurrency accounting", workflow.ExecutionName())
	}
	return nil
}

// Call only after both inventory streams have completed successfully.
func (p *legacyRecurringRunProgress) finish() (*model.RecurringRunState, error) {
	if p.err != nil {
		return nil, p.err
	}
	if run := p.byIndex[p.index]; run != nil {
		p.state.LastRunUUID = run.UUID
		p.state.LastCreatedAtInSec = run.CreatedAtInSec
		p.state.PipelineVersionID = run.PipelineVersionID
	}
	if run := p.byIndex[p.index+1]; run != nil {
		scheduledAt := run.ScheduledAtInSec
		if run.recoverDue {
			// Older API requests stamped epoch before Kubernetes creation. Recovery
			// retains that epoch, but uses Kubernetes time as CreatedAt. Reconstruct
			// the due tick at the epoch so catch-up does not skip the backlog.
			var err error
			scheduledAt, err = legacyRecurringRunDueTime(p.job, p.swf, p.state, run.ScheduledAtInSec)
			if err != nil {
				return nil, p.fail("cannot recover the due time of run %s: %v", run.UUID, err)
			}
		}
		if scheduledAt <= p.state.LastScheduledAtInSec {
			return nil, p.fail("unacknowledged run %s does not advance scheduled time", run.UUID)
		}
		p.state.LastRunUUID = run.UUID
		p.state.LastRunIndex = p.index + 1
		p.state.LastScheduledAtInSec = scheduledAt
		p.state.LastCreatedAtInSec = run.CreatedAtInSec
		p.state.RequestKey = legacyRecurringRunRequestKey(p.swf, p.index+1)
		p.state.PipelineVersionID = run.PipelineVersionID
	}
	return p.state, nil
}

func legacyRecurringRunDueTime(job *model.Job, swf *scheduledworkflow.ScheduledWorkflow, state *model.RecurringRunState, createdAt int64) (int64, error) {
	periodic := !job.PeriodicSchedule.IsEmpty()
	if periodic && (!job.CronSchedule.IsEmpty() || job.IntervalSecond == nil || *job.IntervalSecond < 1) {
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

// The boolean reports whether the index came from a validated reporter-recovered
// execution, whose CreatedAt uses Kubernetes time rather than the API epoch.
func legacyRecurringRunPersistedIndex(swf *scheduledworkflow.ScheduledWorkflow, run *model.Run) (int64, bool, error) {
	requestIndex, requestErr := legacyRecurringRunIndex(swf, run.DisplayName)
	manifest := run.WorkflowRuntimeManifest
	if manifest == "" {
		manifest = run.WorkflowSpecManifest
	}
	if requestErr == nil && (run.DisplayName != run.K8SName || manifest == "") {
		return requestIndex, false, nil
	}
	// Reporter recovery can retain a Workflow name that is itself a valid
	// controller request key. Validate its persisted execution before deciding
	// that CreatedAt is API time: recovery uses Kubernetes creation time instead.
	execution, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(manifest))
	if err != nil {
		return 0, false, err
	}
	meta := execution.ExecutionObjectMeta()
	owner := metav1.GetControllerOf(meta)
	if execution.ExecutionName() != run.K8SName || execution.ExecutionNamespace() != run.Namespace ||
		owner == nil || owner.UID != swf.UID || owner.Name != swf.Name || owner.Kind != "ScheduledWorkflow" ||
		meta.Labels[util.LabelKeyWorkflowRunId] != run.UUID || meta.Labels[util.LabelKeyWorkflowScheduledWorkflowName] != swf.Name {
		return 0, false, fmt.Errorf("persisted execution identity differs")
	}
	epoch, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowEpoch])
	if err != nil || epoch != run.ScheduledAtInSec {
		return 0, false, fmt.Errorf("persisted execution time differs")
	}
	index, err := util.RetrieveInt64FromLabel(meta.Labels[util.LabelKeyWorkflowIndex])
	if err != nil || index <= 0 || index == math.MaxInt64 {
		return 0, false, fmt.Errorf("persisted execution index is invalid")
	}
	if requestErr == nil && (index != requestIndex || meta.CreationTimestamp.Unix() != run.CreatedAtInSec) {
		return 0, false, fmt.Errorf("persisted recovery index or creation time differs")
	}
	return index, true, nil
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
