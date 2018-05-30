// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	swfapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	"hash/fnv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math"
	"sort"
	"strconv"
	"time"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	defaultMaxConcurrency = int64(1)
	minMaxConcurrency     = int64(1)
	maxMaxConcurrency     = int64(10)
	defaultMaxHistory     = int64(10)
	minMaxHistory         = int64(0)
	maxMaxHistory         = int64(100)
)

// ScheduledWorkflowWrap is a wrapper to help manipulate ScheduledWorkflow objects.
type ScheduledWorkflowWrap struct {
	schedule *swfapi.ScheduledWorkflow
}

func NewScheduledWorkflowWrap(schedule *swfapi.ScheduledWorkflow) *ScheduledWorkflowWrap {
	return &ScheduledWorkflowWrap{
		schedule: schedule,
	}
}

func (s *ScheduledWorkflowWrap) ScheduledWorkflow() *swfapi.ScheduledWorkflow {
	return s.schedule
}

func (s *ScheduledWorkflowWrap) creationEpoch() int64 {
	return s.schedule.CreationTimestamp.Unix()
}

func (s *ScheduledWorkflowWrap) Name() string {
	return s.schedule.Name
}

func (s *ScheduledWorkflowWrap) Namespace() string {
	return s.schedule.Namespace
}

func (s *ScheduledWorkflowWrap) enabled() bool {
	return s.schedule.Spec.Enabled
}

func (s *ScheduledWorkflowWrap) maxConcurrency() int64 {
	if s.schedule.Spec.MaxConcurrency == nil {
		return defaultMaxConcurrency
	}

	if *s.schedule.Spec.MaxConcurrency < minMaxConcurrency {
		return minMaxConcurrency
	}

	if *s.schedule.Spec.MaxConcurrency > maxMaxConcurrency {
		return maxMaxConcurrency
	}

	return *s.schedule.Spec.MaxConcurrency
}

func (s *ScheduledWorkflowWrap) maxHistory() int64 {
	if s.schedule.Spec.MaxHistory == nil {
		return defaultMaxHistory
	}

	if *s.schedule.Spec.MaxHistory < minMaxHistory {
		return minMaxHistory
	}

	if *s.schedule.Spec.MaxHistory > maxMaxHistory {
		return maxMaxHistory
	}

	return *s.schedule.Spec.MaxHistory
}

func (s *ScheduledWorkflowWrap) hasRunAtLeastOnce() bool {
	return s.schedule.Status.Trigger.LastTriggeredTime != nil
}

func (s *ScheduledWorkflowWrap) lastIndex() int64 {
	if s.schedule.Status.Trigger.LastIndex == nil {
		return 0
	} else {
		return *s.schedule.Status.Trigger.LastIndex
	}
}

func (s *ScheduledWorkflowWrap) nextIndex() int64 {
	return s.lastIndex() + 1
}

func (s *ScheduledWorkflowWrap) MinIndex() int64 {
	result := s.lastIndex() - s.maxHistory()
	if result < 0 {
		return 0
	}
	return result
}

func (s *ScheduledWorkflowWrap) isOneOffRun() bool {
	return s.schedule.Spec.Trigger.CronSchedule == nil &&
		s.schedule.Spec.Trigger.PeriodicSchedule == nil
}

func (s *ScheduledWorkflowWrap) nextResourceID() string {
	return s.schedule.Name + "-" + strconv.FormatInt(s.nextIndex(), 10)
}

// Creates a deterministic resource name for the next resource.
func (s *ScheduledWorkflowWrap) NextResourceName() string {
	nextResourceID := s.nextResourceID()
	h := fnv.New32a()
	_, _ = h.Write([]byte(nextResourceID))
	return fmt.Sprintf("%s-%v", nextResourceID, h.Sum32())
}

func (s *ScheduledWorkflowWrap) getWorkflowParametersAsMap() map[string]string {
	resultAsArray := s.schedule.Spec.Workflow.Parameters
	resultAsMap := make(map[string]string)
	for _, param := range resultAsArray {
		resultAsMap[param.Name] = param.Value
	}
	return resultAsMap
}

func (s *ScheduledWorkflowWrap) getFormattedWorkflowParametersAsMap(
	formatter *ParameterFormatter) map[string]string {

	result := make(map[string]string)
	for key, value := range s.getWorkflowParametersAsMap() {
		formatted := formatter.Format(value)
		result[key] = formatted
	}
	return result
}

// NewWorkflow creates a workflow for this schedule. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Schedule resource that 'owns' it.
func (s *ScheduledWorkflowWrap) NewWorkflow(
	nextScheduledEpoch int64, nowEpoch int64) *WorkflowWrap {

	const (
		workflowKind       = "Workflow"
		workflowApiVersion = "argoproj.io/v1alpha1"
	)

	// Creating the workflow.
	workflow := &workflowapi.Workflow{
		Spec: *s.schedule.Spec.Workflow.Spec.DeepCopy(),
	}
	workflow.Kind = workflowKind
	workflow.APIVersion = workflowApiVersion
	result := NewWorkflowWrap(workflow)

	// Set the name of the worfklow.
	result.OverrideName(s.NextResourceName())

	// Get the workflow parameters and format them.
	formatter := NewParameterFormatter(nextScheduledEpoch, nowEpoch, s.nextIndex())
	formattedParams := s.getFormattedWorkflowParametersAsMap(formatter)

	// Set the parameters.
	result.OverrideParameters(formattedParams)

	// Set the labels.
	result.SetCanonicalLabels(s.schedule.Name, nextScheduledEpoch, s.nextIndex())

	// The the owner references.
	result.SetOwnerReferences(s.schedule)

	return result
}

func (s *ScheduledWorkflowWrap) GetNextScheduledEpoch(activeWorkflowCount int64, nowEpoch int64) (
	nextScheduleEpoch int64, shouldRunNow bool) {

	// Get the next scheduled time.
	nextScheduledEpoch := s.getNextScheduledEpoch()

	// If the schedule is not enabled, we should not schedule the workflow now.
	if s.enabled() == false {
		return nextScheduledEpoch, false
	}

	// If the maxConcurrency is exceeded, return.
	if activeWorkflowCount >= s.maxConcurrency() {
		return nextScheduledEpoch, false
	}

	// If it is not yet time to schedule the next workflow...
	if nextScheduledEpoch > nowEpoch {
		return nextScheduledEpoch, false
	}

	return nextScheduledEpoch, true
}

func (s *ScheduledWorkflowWrap) getNextScheduledEpoch() int64 {
	// Periodic schedule
	if s.schedule.Spec.Trigger.PeriodicSchedule != nil {
		return NewPeriodicScheduleWrap(s.schedule.Spec.Trigger.PeriodicSchedule).
			GetNextScheduledEpoch(
				toInt64Pointer(s.schedule.Status.Trigger.LastTriggeredTime),
				s.creationEpoch())
	}

	// Cron schedule
	if s.schedule.Spec.Trigger.CronSchedule != nil {
		return NewCronScheduleWrap(s.schedule.Spec.Trigger.CronSchedule).
			GetNextScheduledEpoch(
				toInt64Pointer(s.schedule.Status.Trigger.LastTriggeredTime),
				s.creationEpoch())
	}

	return s.getNextScheduledEpochForOneTimeRun()
}

func (s *ScheduledWorkflowWrap) getNextScheduledEpochForOneTimeRun() int64 {
	if s.schedule.Status.Trigger.LastTriggeredTime != nil {
		return math.MaxInt64
	}

	return s.creationEpoch()
}

func (s *ScheduledWorkflowWrap) SetLabel(key string, value string) {
	if s.schedule.Labels == nil {
		s.schedule.Labels = make(map[string]string)
	}
	s.schedule.Labels[key] = value
}

func (s *ScheduledWorkflowWrap) UpdateStatus(updatedEpoch int64, workflow *WorkflowWrap,
	scheduledEpoch int64, active []swfapi.WorkflowStatus,
	completed []swfapi.WorkflowStatus) {

	updatedTime := 	metav1.NewTime(time.Unix(updatedEpoch, 0).UTC())

	conditionType, status, message := s.getStatusAndMessage(len(active))

	condition := swfapi.ScheduledWorkflowCondition{
		Type: conditionType,
		Status: status,
		LastProbeTime: updatedTime,
		LastTransitionTime: updatedTime,
		Reason: string(conditionType),
		Message: message,
	}

	conditions := make([]swfapi.ScheduledWorkflowCondition, 0)
	conditions = append(conditions, condition)

	s.schedule.Status.Conditions = conditions

	// Sort and set inactive workflows.
	sort.Slice(active, func(i, j int) bool {
		return active[i].ScheduledAt.Unix() > active[j].ScheduledAt.Unix()
	})

	sort.Slice(completed, func(i, j int) bool {
		return completed[i].ScheduledAt.Unix() > completed[j].ScheduledAt.Unix()
	})

	s.schedule.Status.WorkflowHistory = &swfapi.WorkflowHistory{
		Active:    active,
		Completed: completed,
	}

	s.SetLabel(LabelKeyScheduledWorkflowEnabled, strconv.FormatBool(
		s.enabled()))
	s.SetLabel(LabelKeyScheduledWorkflowStatus, string(conditionType))

	if workflow != nil {
		s.updateLastTriggeredTime(scheduledEpoch)
		s.schedule.Status.Trigger.LastIndex = Int64Pointer(s.nextIndex())
		s.updateNextTriggeredTime(s.getNextScheduledEpoch())
	} else {
		// LastTriggeredTime is unchanged.
		s.updateNextTriggeredTime(scheduledEpoch)
		// LastIndex is unchanged
	}
}

func (s *ScheduledWorkflowWrap) updateLastTriggeredTime(epoch int64) {
	s.schedule.Status.Trigger.LastTriggeredTime = Metav1TimePointer(
		metav1.NewTime(time.Unix(epoch, 0).UTC()))
}

func (s *ScheduledWorkflowWrap) updateNextTriggeredTime(epoch int64) {
	if epoch != math.MaxInt64 {
		s.schedule.Status.Trigger.NextTriggeredTime = Metav1TimePointer(
			metav1.NewTime(time.Unix(epoch, 0).UTC()))
	} else {
		s.schedule.Status.Trigger.NextTriggeredTime = nil
	}
}

func (s *ScheduledWorkflowWrap) getStatusAndMessage(activeCount int) (
	conditionType swfapi.ScheduledWorkflowConditionType,
	status core.ConditionStatus, message string) {
	// Schedule messages
	const (
		ScheduleEnabledMessage   = "The schedule is enabled."
		ScheduleDisabledMessage  = "The schedule is disabled."
		ScheduleRunningMessage   = "The one-off schedule is running."
		ScheduleSucceededMessage = "The one-off schedule has succeeded."
	)

	if s.isOneOffRun() {
		if s.hasRunAtLeastOnce() && activeCount == 0 {
			return swfapi.ScheduledWorkflowSucceeded, core.ConditionTrue, ScheduleSucceededMessage
		} else {
			return swfapi.ScheduledWorkflowRunning, core.ConditionTrue, ScheduleRunningMessage
		}
	} else {
		if s.enabled() {
			return swfapi.ScheduledWorkflowEnabled, core.ConditionTrue, ScheduleEnabledMessage
		} else {
			return swfapi.ScheduledWorkflowDisabled, core.ConditionTrue, ScheduleDisabledMessage
		}
	}
}
