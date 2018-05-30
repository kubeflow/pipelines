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
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	scheduleapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math"
	"strconv"
	"testing"
	"time"
	"k8s.io/kubernetes/pkg/apis/core"
)

func TestScheduleWrap_maxConcurrency(t *testing.T) {
	// nil
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// lower than min
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxConcurrency: Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// higher than max
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxConcurrency: Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(10), schedule.maxConcurrency())
}

func TestScheduleWrap_maxHistory(t *testing.T) {
	// nil
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{})
	assert.Equal(t, int64(10), schedule.maxHistory())

	// lower than min
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(1), schedule.maxHistory())

	// higher than max
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(100), schedule.maxHistory())
}

func TestScheduleWrap_hasRunAtLeastOnce(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastTriggeredTime: nil,
			},
		},
	})
	assert.Equal(t, false, schedule.hasRunAtLeastOnce())

	// Ran one workflow
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastTriggeredTime: Metav1TimePointer(metav1.NewTime(time.Unix(50, 0).UTC())),
			},
		},
	})
	assert.Equal(t, true, schedule.hasRunAtLeastOnce())
}

func TestScheduleWrap_lastIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{})
	assert.Equal(t, int64(0), schedule.lastIndex())

	// Ran one workflow
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(50), schedule.lastIndex())
}

func TestScheduleWrap_nextIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.nextIndex())

	// Ran one workflow
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(51), schedule.nextIndex())
}

func TestScheduleWrap_MinIndex(t *testing.T) {
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(100),
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(0), schedule.MinIndex())

	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(20),
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(30), schedule.MinIndex())
}

func TestScheduleWrap_isOneOffRun(t *testing.T) {
	// No schedule
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{})
	assert.Equal(t, true, schedule.isOneOffRun())

	// Cron schedule
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Trigger: scheduleapi.Trigger{
				CronSchedule: &scheduleapi.CronSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())

	// Periodic schedule
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())
}

func TestScheduleWrap_nextResourceID(t *testing.T) {
	// No schedule
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51", schedule.nextResourceID())
}

func TestScheduleWrap_NextResourceName(t *testing.T) {
	// No schedule
	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51-2626342551", schedule.NextResourceName())
}

func TestScheduleWrap_GetNextScheduledEpoch_OneTimeRun(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(1 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())
	lastTimeRun := metav1.NewTime(time.Unix(11*hour, 0).UTC())
	never := int64(math.MaxInt64)

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow := schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, true, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Has already run
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Trigger: scheduleapi.TriggerStatus{
				LastTriggeredTime: &lastTimeRun,
			},
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, never, nextScheduledEpoch)

	// Should not run yet because it is not time
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, pastEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Should not run because the schedule is disabled
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled: false,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Should not run because there are active workflows
	schedule = NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(1) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)
}

func TestScheduleWrap_GetNextScheduledEpoch_CronSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				CronSchedule: &scheduleapi.CronSchedule{
					Cron: "0 * * * * *",
				},
			},
		},
	})
	nextScheduledEpoch, mustRunNow := schedule.GetNextScheduledEpoch(
		int64(9) /* active workflow count */, nowEpoch)
	assert.Equal(t, true, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)

	// Must run later
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(9) /* active workflow count */, pastEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)

	// Cannot run because of concurrency
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(10) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)
}

func TestScheduleWrap_GetNextScheduledEpoch_PeriodicSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
	})
	nextScheduledEpoch, mustRunNow := schedule.GetNextScheduledEpoch(
		int64(9) /* active workflow count */, nowEpoch)
	assert.Equal(t, true, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)

	// Must run later
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(9) /* active workflow count */, pastEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)

	// Cannot run because of concurrency
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(10) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, int64(9*hour+minute), nextScheduledEpoch)

}

func TestScheduleWrap_GetNextScheduledEpoch_UpdateStatus_NoWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
	})

	status1 := createStatus("WORKFLOW1", 5)
	status2 := createStatus("WORKFLOW2", 3)
	status3 := createStatus("WORKFLOW3", 7)
	status4 := createStatus("WORKFLOW4", 4)

	schedule.UpdateStatus(
		updatedEpoch,
		nil, /* no workflow created during this run */
		scheduledEpoch,
		[]scheduleapi.WorkflowStatus{*status1, *status2, *status3},
		[]scheduleapi.WorkflowStatus{*status1, *status2, *status3, *status4})

	expected := &scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Labels: map[string]string{
				LabelKeyScheduledWorkflowEnabled: "true",
				LabelKeyScheduledWorkflowStatus:  string(scheduleapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Conditions: []scheduleapi.ScheduledWorkflowCondition{{
					Type: scheduleapi.ScheduledWorkflowEnabled,
					Status: core.ConditionTrue,
					LastProbeTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
					LastTransitionTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
					Reason: string(scheduleapi.ScheduledWorkflowEnabled),
					Message: "The schedule is enabled.",
				},
			},
			WorkflowHistory: &scheduleapi.WorkflowHistory{
				Active:    []scheduleapi.WorkflowStatus{*status3, *status1, *status2},
				Completed: []scheduleapi.WorkflowStatus{*status3, *status1, *status4, *status2},
			},
			Trigger: scheduleapi.TriggerStatus{
				NextTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
			},
		},
	}

	assert.Equal(t, expected, schedule.Schedule())
}

func createStatus(workflowName string, scheduledEpoch int64) *scheduleapi.WorkflowStatus {
	return &scheduleapi.WorkflowStatus{
		Name:        workflowName,
		ScheduledAt: metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC()),
	}
}

func TestScheduleWrap_GetNextScheduledEpoch_UpdateStatus_WithWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
	})

	status1 := createStatus("WORKFLOW1", 5)
	status2 := createStatus("WORKFLOW2", 3)
	status3 := createStatus("WORKFLOW3", 7)
	status4 := createStatus("WORKFLOW4", 4)

	workflow := NewWorkflowWrap(&workflowapi.Workflow{})

	schedule.UpdateStatus(
		updatedEpoch,
		workflow, /* no workflow created during this run */
		scheduledEpoch,
		[]scheduleapi.WorkflowStatus{*status1, *status2, *status3},
		[]scheduleapi.WorkflowStatus{*status1, *status2, *status3, *status4})

	expected := &scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Labels: map[string]string{
				LabelKeyScheduledWorkflowEnabled: "true",
				LabelKeyScheduledWorkflowStatus:  string(scheduleapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
		Status: scheduleapi.ScheduledWorkflowStatus{
			Conditions: []scheduleapi.ScheduledWorkflowCondition{{
				Type: scheduleapi.ScheduledWorkflowEnabled,
				Status: core.ConditionTrue,
				LastProbeTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				Reason: string(scheduleapi.ScheduledWorkflowEnabled),
				Message: "The schedule is enabled.",
			}},
			WorkflowHistory: &scheduleapi.WorkflowHistory{
				Active:    []scheduleapi.WorkflowStatus{*status3, *status1, *status2},
				Completed: []scheduleapi.WorkflowStatus{*status3, *status1, *status4, *status2},
			},
			Trigger: scheduleapi.TriggerStatus{
				LastTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
				NextTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch+minute, 0).UTC())),
				LastIndex: Int64Pointer(int64(1)),
			},
		},
	}

	assert.Equal(t, expected, schedule.Schedule())
}

func TestScheduleWrap_NewWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	nowEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduleWrap(&scheduleapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "SCHEDULE1",
			CreationTimestamp: creationTimestamp,
		},
		Spec: scheduleapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: scheduleapi.Trigger{
				PeriodicSchedule: &scheduleapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
			Workflow: &scheduleapi.WorkflowResource{
				Parameters: []scheduleapi.Parameter{
					{Name: "PARAM1", Value: "NEW_VALUE1"},
					{Name: "PARAM3", Value: "NEW_VALUE3"},
				},
				Spec: workflowapi.WorkflowSpec{
					ServiceAccountName: "SERVICE_ACCOUNT",
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1", Value: StringPointer("VALUE1")},
							{Name: "PARAM2", Value: StringPointer("VALUE2")},
						},
					},
				},
			},
		},
	})

	result := schedule.NewWorkflow(scheduledEpoch, nowEpoch)

	expected := &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE1-1-3321103997",
			Labels: map[string]string{
				"schedules.kubeflow.org/isOwnedBySchedule": "true",
				"schedules.kubeflow.org/scheduleName":      "SCHEDULE1",
				"schedules.kubeflow.org/scheduledEpoch":    strconv.Itoa(int(scheduledEpoch)),
				"schedules.kubeflow.org/workflowIndex":     "1"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Kind:               "Schedule",
				Name:               "SCHEDULE1",
				UID:                "",
				Controller:         BooleanPointer(true),
				BlockOwnerDeletion: BooleanPointer(true)}},
		},
		Spec: workflowapi.WorkflowSpec{
			ServiceAccountName: "SERVICE_ACCOUNT",
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: StringPointer("NEW_VALUE1")},
					{Name: "PARAM2", Value: StringPointer("VALUE2")},
				},
			},
		},
	}

	assert.Equal(t, expected, result.Workflow())
}
