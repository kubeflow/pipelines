// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"hash/fnv"
	core "k8s.io/api/core/v1"
	"math"
	"sort"
	"strconv"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultMaxConcurrency = int64(1)
	minMaxConcurrency     = int64(1)
	maxMaxConcurrency     = int64(10)
	defaultMaxHistory     = int64(10)
	minMaxHistory         = int64(0)
	maxMaxHistory         = int64(100)
)

// ScheduledWorkflow is a type to help manipulate ScheduledWorkflow objects.
type ScheduledWorkflow struct {
	*swfapi.ScheduledWorkflow
	uuid commonutil.UUIDGeneratorInterface
}

// NewScheduledWorkflow creates an instance of ScheduledWorkflow.
func NewScheduledWorkflow(swf *swfapi.ScheduledWorkflow) *ScheduledWorkflow {
	return &ScheduledWorkflow{
		swf, commonutil.NewUUIDGenerator(),
	}
}

// Get converts this object to a swfapi.ScheduledWorkflow.
func (s *ScheduledWorkflow) Get() *swfapi.ScheduledWorkflow {
	return s.ScheduledWorkflow
}

func (s *ScheduledWorkflow) creationEpoch() int64 {
	return s.CreationTimestamp.Unix()
}

func (s *ScheduledWorkflow) enabled() bool {
	return s.Spec.Enabled
}

func (s *ScheduledWorkflow) maxConcurrency() int64 {
	if s.Spec.MaxConcurrency == nil {
		return defaultMaxConcurrency
	}

	if *s.Spec.MaxConcurrency < minMaxConcurrency {
		return minMaxConcurrency
	}

	if *s.Spec.MaxConcurrency > maxMaxConcurrency {
		return maxMaxConcurrency
	}

	return *s.Spec.MaxConcurrency
}

func (s *ScheduledWorkflow) maxHistory() int64 {
	if s.Spec.MaxHistory == nil {
		return defaultMaxHistory
	}

	if *s.Spec.MaxHistory < minMaxHistory {
		return minMaxHistory
	}

	if *s.Spec.MaxHistory > maxMaxHistory {
		return maxMaxHistory
	}

	return *s.Spec.MaxHistory
}

func (s *ScheduledWorkflow) hasRunAtLeastOnce() bool {
	return s.Status.Trigger.LastTriggeredTime != nil
}

func (s *ScheduledWorkflow) lastIndex() int64 {
	if s.Status.Trigger.LastIndex == nil {
		return 0
	} else {
		return *s.Status.Trigger.LastIndex
	}
}

func (s *ScheduledWorkflow) nextIndex() int64 {
	return s.lastIndex() + 1
}

// MinIndex returns the minimum index of the workflow to retrieve as part of the workflow
// history.
func (s *ScheduledWorkflow) MinIndex() int64 {
	result := s.lastIndex() - s.maxHistory()
	if result < 0 {
		return 0
	}
	return result
}

func (s *ScheduledWorkflow) isOneOffRun() bool {
	return s.Spec.Trigger.CronSchedule == nil &&
		s.Spec.Trigger.PeriodicSchedule == nil
}

func (s *ScheduledWorkflow) nextResourceID() string {
	return s.Name + "-" + strconv.FormatInt(s.nextIndex(), 10)
}

// NextResourceName creates a deterministic resource name for the next resource.
func (s *ScheduledWorkflow) NextResourceName() string {
	nextResourceID := s.nextResourceID()
	h := fnv.New32a()
	_, _ = h.Write([]byte(nextResourceID))
	return fmt.Sprintf("%s-%v", nextResourceID, h.Sum32())
}

func (s *ScheduledWorkflow) getWorkflowParametersAsMap() map[string]string {
	resultAsArray := s.Spec.Workflow.Parameters
	resultAsMap := make(map[string]string)
	for _, param := range resultAsArray {
		resultAsMap[param.Name] = param.Value
	}
	return resultAsMap
}

// NewWorkflow creates a workflow for this schedule. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Schedule resource that 'owns' it.
func (s *ScheduledWorkflow) NewWorkflow(
	nextScheduledEpoch int64, nowEpoch int64) (commonutil.ExecutionSpec, error) {

	// Creating the workflow.
	execSpec, err := commonutil.ScheduleSpecToExecutionSpec(commonutil.CurrentExecutionType(), s.Spec.Workflow)
	if err != nil {
		return nil, err
	}

	uuid, err := s.uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// Set the name of the workflow.
	execSpec.SetExecutionName(s.NextResourceName())

	// Get the workflow parameters and format them.
	formatter := commonutil.NewSWFParameterFormatter(uuid.String(), nextScheduledEpoch, nowEpoch, s.nextIndex())
	formattedParams := formatter.FormatWorkflowParameters(s.getWorkflowParametersAsMap())

	// Set the parameters.
	execSpec.OverrideParameters(formattedParams)

	execSpec.SetCannonicalLabels(s.Name, nextScheduledEpoch, s.nextIndex())
	execSpec.SetLabels(commonutil.LabelKeyWorkflowRunId, uuid.String())
	// Pod pipeline/runid label is used by v2 compatible mode.
	execSpec.SetPodMetadataLabels(commonutil.LabelKeyWorkflowRunId, uuid.String())
	// Replace {{workflow.uid}} with runId
	err = execSpec.ReplaceUID(uuid.String())
	if err != nil {
		return nil, err
	}
	// The the owner references.
	execSpec.SetOwnerReferences(s.ScheduledWorkflow)

	return execSpec, nil
}

// GetNextScheduledEpoch returns the next epoch at which a workflow should be scheduled,
// and whether it should be run now.
func (s *ScheduledWorkflow) GetNextScheduledEpoch(activeWorkflowCount int64, nowEpoch int64, location time.Location) (
	nextScheduleEpoch int64, shouldRunNow bool) {

	// Get the next scheduled time.
	nextScheduledEpoch := s.getNextScheduledEpoch(nowEpoch, location)

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

func (s *ScheduledWorkflow) getNextScheduledEpoch(nowEpoch int64, location time.Location) int64 {
	catchup := true
	if s.Spec.NoCatchup != nil {
		catchup = !*s.Spec.NoCatchup
	}
	// Periodic schedule
	if s.Spec.Trigger.PeriodicSchedule != nil {
		schedule := NewPeriodicSchedule(s.Spec.Trigger.PeriodicSchedule)
		if catchup {
			return schedule.GetNextScheduledEpoch(
				commonutil.ToInt64Pointer(s.Status.Trigger.LastTriggeredTime),
				s.creationEpoch())
		}
		return schedule.GetNextScheduledEpochNoCatchup(
			commonutil.ToInt64Pointer(s.Status.Trigger.LastTriggeredTime),
			s.creationEpoch(), nowEpoch)

	}

	// Cron schedule
	if s.Spec.Trigger.CronSchedule != nil {
		nowTime := time.Unix(nowEpoch, 0)
		schedule := NewCronSchedule(s.Spec.Trigger.CronSchedule)
		if catchup {
			return schedule.GetNextScheduledTime(
				s.Status.Trigger.LastTriggeredTime,
				time.Unix(s.creationEpoch(), 0).In(&location), &location).Unix()
		}
		return schedule.GetNextScheduledTimeNoCatchup(
			s.Status.Trigger.LastTriggeredTime,
			time.Unix(s.creationEpoch(), 0).In(&location), nowTime, &location).Unix()
	}

	return s.getNextScheduledEpochForOneTimeRun()
}

func (s *ScheduledWorkflow) getNextScheduledEpochForOneTimeRun() int64 {
	if s.Status.Trigger.LastTriggeredTime != nil {
		return math.MaxInt64
	}

	return s.creationEpoch()
}

func (s *ScheduledWorkflow) setLabel(key string, value string) {
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	s.Labels[key] = value
}

// UpdateStatus updates the status of a workflow in the Kubernetes API server.
func (s *ScheduledWorkflow) UpdateStatus(updatedEpoch int64, submitted bool,
	scheduledEpoch int64, active []swfapi.WorkflowStatus,
	completed []swfapi.WorkflowStatus, location *time.Location) {

	updatedTime := metav1.NewTime(time.Unix(updatedEpoch, 0).UTC())

	conditionType, status, message := s.getStatusAndMessage(len(active))

	condition := swfapi.ScheduledWorkflowCondition{
		Type:               conditionType,
		Status:             status,
		LastProbeTime:      updatedTime,
		LastTransitionTime: updatedTime,
		Reason:             string(conditionType),
		Message:            message,
	}

	conditions := make([]swfapi.ScheduledWorkflowCondition, 0)
	conditions = append(conditions, condition)

	s.Status.Conditions = conditions

	// Sort and set inactive workflows.
	sort.Slice(active, func(i, j int) bool {
		return active[i].ScheduledAt.Unix() > active[j].ScheduledAt.Unix()
	})

	sort.Slice(completed, func(i, j int) bool {
		return completed[i].ScheduledAt.Unix() > completed[j].ScheduledAt.Unix()
	})

	s.Status.WorkflowHistory = &swfapi.WorkflowHistory{
		Active:    active,
		Completed: completed,
	}

	s.setLabel(commonutil.LabelKeyScheduledWorkflowEnabled, strconv.FormatBool(
		s.enabled()))
	s.setLabel(commonutil.LabelKeyScheduledWorkflowStatus, string(conditionType))

	if submitted {
		s.updateLastTriggeredTime(scheduledEpoch)
		s.Status.Trigger.LastIndex = commonutil.Int64Pointer(s.nextIndex())
		nextTriggerTime := s.getNextScheduledEpoch(0, *location)
		s.updateNextTriggeredTime(nextTriggerTime)
	} else {
		// LastTriggeredTime is unchanged.
		s.updateNextTriggeredTime(scheduledEpoch)
		// LastIndex is unchanged
	}
}

func (s *ScheduledWorkflow) updateLastTriggeredTime(epoch int64) {
	s.Status.Trigger.LastTriggeredTime = commonutil.Metav1TimePointer(
		metav1.NewTime(time.Unix(epoch, 0).UTC()))
}

func (s *ScheduledWorkflow) updateNextTriggeredTime(epoch int64) {
	if epoch < maxTime.Unix() {
		s.Status.Trigger.NextTriggeredTime = commonutil.Metav1TimePointer(
			metav1.NewTime(time.Unix(epoch, 0).UTC()))
	} else {
		s.Status.Trigger.NextTriggeredTime = nil
	}
}

func (s *ScheduledWorkflow) getStatusAndMessage(activeCount int) (
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
