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
	swfapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	"math"
	"strconv"
	"testing"
	"time"
)

func TestScheduledWorkflowWrap_maxConcurrency(t *testing.T) {
	// nil
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// lower than min
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxConcurrency: Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// higher than max
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxConcurrency: Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(10), schedule.maxConcurrency())
}

func TestScheduledWorkflowWrap_maxHistory(t *testing.T) {
	// nil
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(10), schedule.maxHistory())

	// lower than min
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(0), schedule.maxHistory())

	// higher than max
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(100), schedule.maxHistory())
}

func TestScheduledWorkflowWrap_hasRunAtLeastOnce(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: nil,
			},
		},
	})
	assert.Equal(t, false, schedule.hasRunAtLeastOnce())

	// Ran one workflow
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: Metav1TimePointer(metav1.NewTime(time.Unix(50, 0).UTC())),
			},
		},
	})
	assert.Equal(t, true, schedule.hasRunAtLeastOnce())
}

func TestScheduledWorkflowWrap_lastIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(0), schedule.lastIndex())

	// Ran one workflow
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(50), schedule.lastIndex())
}

func TestScheduledWorkflowWrap_nextIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.nextIndex())

	// Ran one workflow
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(51), schedule.nextIndex())
}

func TestScheduledWorkflowWrap_MinIndex(t *testing.T) {
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(100),
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(0), schedule.MinIndex())

	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: Int64Pointer(20),
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(30), schedule.MinIndex())
}

func TestScheduledWorkflowWrap_isOneOffRun(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, true, schedule.isOneOffRun())

	// Cron schedule
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())

	// Periodic schedule
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())
}

func TestScheduledWorkflowWrap_nextResourceID(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51", schedule.nextResourceID())
}

func TestScheduledWorkflowWrap_NextResourceName(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51-2626342551", schedule.NextResourceName())
}

func TestScheduledWorkflowWrap_GetNextScheduledEpoch_OneTimeRun(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(1 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())
	lastTimeRun := metav1.NewTime(time.Unix(11*hour, 0).UTC())
	never := int64(math.MaxInt64)

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow := schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, true, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Has already run
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: &lastTimeRun,
			},
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, never, nextScheduledEpoch)

	// Should not run yet because it is not time
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, pastEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Should not run because the schedule is disabled
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled: false,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(0) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)

	// Should not run because there are active workflows
	schedule = NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled: true,
		},
	})
	nextScheduledEpoch, mustRunNow = schedule.GetNextScheduledEpoch(
		int64(1) /* active workflow count */, nowEpoch)
	assert.Equal(t, false, mustRunNow)
	assert.Equal(t, creationTimestamp.Unix(), nextScheduledEpoch)
}

func TestScheduledWorkflowWrap_GetNextScheduledEpoch_CronSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{
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

func TestScheduledWorkflowWrap_GetNextScheduledEpoch_PeriodicSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
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

func TestScheduledWorkflowWrap_GetNextScheduledEpoch_UpdateStatus_NoWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
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
		[]swfapi.WorkflowStatus{*status1, *status2, *status3},
		[]swfapi.WorkflowStatus{*status1, *status2, *status3, *status4})

	expected := &swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Labels: map[string]string{
				LabelKeyScheduledWorkflowEnabled: "true",
				LabelKeyScheduledWorkflowStatus:  string(swfapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:               swfapi.ScheduledWorkflowEnabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			},
			},
			WorkflowHistory: &swfapi.WorkflowHistory{
				Active:    []swfapi.WorkflowStatus{*status3, *status1, *status2},
				Completed: []swfapi.WorkflowStatus{*status3, *status1, *status4, *status2},
			},
			Trigger: swfapi.TriggerStatus{
				NextTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
			},
		},
	}

	assert.Equal(t, expected, schedule.ScheduledWorkflow())
}

func createStatus(workflowName string, scheduledEpoch int64) *swfapi.WorkflowStatus {
	return &swfapi.WorkflowStatus{
		Name:        workflowName,
		ScheduledAt: metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC()),
	}
}

func TestScheduledWorkflowWrap_GetNextScheduledEpoch_UpdateStatus_WithWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
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
		[]swfapi.WorkflowStatus{*status1, *status2, *status3},
		[]swfapi.WorkflowStatus{*status1, *status2, *status3, *status4})

	expected := &swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Labels: map[string]string{
				LabelKeyScheduledWorkflowEnabled: "true",
				LabelKeyScheduledWorkflowStatus:  string(swfapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:               swfapi.ScheduledWorkflowEnabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(updatedEpoch, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			}},
			WorkflowHistory: &swfapi.WorkflowHistory{
				Active:    []swfapi.WorkflowStatus{*status3, *status1, *status2},
				Completed: []swfapi.WorkflowStatus{*status3, *status1, *status4, *status2},
			},
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
				NextTriggeredTime: Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch+minute, 0).UTC())),
				LastIndex: Int64Pointer(int64(1)),
			},
		},
	}

	assert.Equal(t, expected, schedule.ScheduledWorkflow())
}

func TestScheduledWorkflowWrap_NewWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	nowEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflowWrap(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "SCHEDULE1",
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
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
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULE1",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowEpoch":     strconv.Itoa(int(scheduledEpoch)),
				"scheduledworkflows.kubeflow.org/workflowIndex":              "1"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Kind:               "ScheduledWorkflow",
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
