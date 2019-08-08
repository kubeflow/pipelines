// Copyright 2018 Google LLC
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
	"math"
	"strconv"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func TestScheduledWorkflow_maxConcurrency(t *testing.T) {
	// nil
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// lower than min
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxConcurrency: commonutil.Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(1), schedule.maxConcurrency())

	// higher than max
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxConcurrency: commonutil.Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(10), schedule.maxConcurrency())
}

func TestScheduledWorkflow_maxHistory(t *testing.T) {
	// nil
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(10), schedule.maxHistory())

	// lower than min
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: commonutil.Int64Pointer(0),
		},
	})
	assert.Equal(t, int64(0), schedule.maxHistory())

	// higher than max
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: commonutil.Int64Pointer(2000000),
		},
	})
	assert.Equal(t, int64(100), schedule.maxHistory())
}

func TestScheduledWorkflow_hasRunAtLeastOnce(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: nil,
			},
		},
	})
	assert.Equal(t, false, schedule.hasRunAtLeastOnce())

	// Ran one workflow
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastTriggeredTime: commonutil.Metav1TimePointer(metav1.NewTime(time.Unix(50, 0).UTC())),
			},
		},
	})
	assert.Equal(t, true, schedule.hasRunAtLeastOnce())
}

func TestScheduledWorkflow_lastIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(0), schedule.lastIndex())

	// Ran one workflow
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(50), schedule.lastIndex())
}

func TestScheduledWorkflow_nextIndex(t *testing.T) {
	// Never ran a workflow
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, int64(1), schedule.nextIndex())

	// Ran one workflow
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(51), schedule.nextIndex())
}

func TestScheduledWorkflow_MinIndex(t *testing.T) {
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: commonutil.Int64Pointer(100),
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(0), schedule.MinIndex())

	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			MaxHistory: commonutil.Int64Pointer(20),
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, int64(30), schedule.MinIndex())
}

func TestScheduledWorkflow_isOneOffRun(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{})
	assert.Equal(t, true, schedule.isOneOffRun())

	// Cron schedule
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())

	// Periodic schedule
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{},
			},
		},
	})
	assert.Equal(t, false, schedule.isOneOffRun())
}

func TestScheduledWorkflow_nextResourceID(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51", schedule.nextResourceID())
}

func TestScheduledWorkflow_NextResourceName(t *testing.T) {
	// No schedule
	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Trigger: swfapi.TriggerStatus{
				LastIndex: commonutil.Int64Pointer(50),
			},
		},
	})
	assert.Equal(t, "WORKFLOW_NAME-51-2626342551", schedule.NextResourceName())
}

func TestScheduledWorkflow_GetNextScheduledEpoch_OneTimeRun(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(1 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())
	lastTimeRun := metav1.NewTime(time.Unix(11*hour, 0).UTC())
	never := int64(math.MaxInt64)

	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
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
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
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
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
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
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
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
	schedule = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
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

func TestScheduledWorkflow_GetNextScheduledEpoch_CronSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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

func TestScheduledWorkflow_GetNextScheduledEpoch_PeriodicSchedule(t *testing.T) {

	// Must run now
	nowEpoch := int64(10 * hour)
	pastEpoch := int64(3 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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

func TestScheduledWorkflow_GetNextScheduledEpoch_UpdateStatus_NoWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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
				commonutil.LabelKeyScheduledWorkflowEnabled: "true",
				commonutil.LabelKeyScheduledWorkflowStatus:  string(swfapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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
				NextTriggeredTime: commonutil.Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
			},
		},
	}

	assert.Equal(t, expected, schedule.Get())
}

func createStatus(workflowName string, scheduledEpoch int64) *swfapi.WorkflowStatus {
	return &swfapi.WorkflowStatus{
		Name:        workflowName,
		ScheduledAt: metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC()),
	}
}

func TestScheduledWorkflow_GetNextScheduledEpoch_UpdateStatus_WithWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	updatedEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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

	workflow := commonutil.NewWorkflow(&workflowapi.Workflow{})

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
				commonutil.LabelKeyScheduledWorkflowEnabled: "true",
				commonutil.LabelKeyScheduledWorkflowStatus:  string(swfapi.ScheduledWorkflowEnabled),
			},
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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
				LastTriggeredTime: commonutil.Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch, 0).UTC())),
				NextTriggeredTime: commonutil.Metav1TimePointer(
					metav1.NewTime(time.Unix(scheduledEpoch+minute, 0).UTC())),
				LastIndex: commonutil.Int64Pointer(int64(1)),
			},
		},
	}

	assert.Equal(t, expected, schedule.Get())
}

func TestScheduledWorkflow_NewWorkflow(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	nowEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := ScheduledWorkflow{&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "SCHEDULE1",
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
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
							{Name: "PARAM1", Value: commonutil.StringPointer("VALUE1")},
							{Name: "PARAM2", Value: commonutil.StringPointer("VALUE2")},
						},
					},
				},
			},
		},
	}, commonutil.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655440001", nil)}

	result, err := schedule.NewWorkflow(scheduledEpoch, nowEpoch)
	assert.Nil(t, err)

	expected := &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE1-1-3321103997",
			Labels: map[string]string{
				"pipeline/runid": "123e4567-e89b-12d3-a456-426655440001",
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULE1",
				"scheduledworkflows.kubeflow.org/workflowEpoch":              strconv.Itoa(int(scheduledEpoch)),
				"scheduledworkflows.kubeflow.org/workflowIndex":              "1"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
				Kind:               "ScheduledWorkflow",
				Name:               "SCHEDULE1",
				UID:                "",
				Controller:         commonutil.BooleanPointer(true),
				BlockOwnerDeletion: commonutil.BooleanPointer(true)}},
		},
		Spec: workflowapi.WorkflowSpec{
			ServiceAccountName: "SERVICE_ACCOUNT",
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: commonutil.StringPointer("NEW_VALUE1")},
					{Name: "PARAM2", Value: commonutil.StringPointer("VALUE2")},
				},
			},
		},
	}

	assert.Equal(t, expected, result.Get())
}

func TestScheduledWorkflow_NewWorkflow_Parameterized(t *testing.T) {
	// Must run now
	scheduledEpoch := int64(10 * hour)
	nowEpoch := int64(11 * hour)
	creationTimestamp := metav1.NewTime(time.Unix(9*hour, 0).UTC())

	schedule := ScheduledWorkflow{&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "SCHEDULE1",
			CreationTimestamp: creationTimestamp,
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Enabled:        true,
			MaxConcurrency: commonutil.Int64Pointer(int64(10)),
			Trigger: swfapi.Trigger{
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					IntervalSecond: int64(60),
				},
			},
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "NEW_VALUE1_[[ScheduledTime]]"},
					{Name: "PARAM2", Value: "NEW_VALUE2_[[Index]]"},
				},
				Spec: workflowapi.WorkflowSpec{
					ServiceAccountName: "SERVICE_ACCOUNT",
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1", Value: commonutil.StringPointer("VALUE1")},
							{Name: "PARAM2", Value: commonutil.StringPointer("VALUE2")},
						},
					},
				},
			},
		},
	}, commonutil.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655440001", nil)}

	result, err := schedule.NewWorkflow(scheduledEpoch, nowEpoch)
	assert.Nil(t, err)
	expected := &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE1-1-3321103997",
			Labels: map[string]string{
				"pipeline/runid": "123e4567-e89b-12d3-a456-426655440001",
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULE1",
				"scheduledworkflows.kubeflow.org/workflowEpoch":              strconv.Itoa(int(scheduledEpoch)),
				"scheduledworkflows.kubeflow.org/workflowIndex":              "1"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
				Kind:               "ScheduledWorkflow",
				Name:               "SCHEDULE1",
				UID:                "",
				Controller:         commonutil.BooleanPointer(true),
				BlockOwnerDeletion: commonutil.BooleanPointer(true)}},
		},
		Spec: workflowapi.WorkflowSpec{
			ServiceAccountName: "SERVICE_ACCOUNT",
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: commonutil.StringPointer("NEW_VALUE1_19700101100000")},
					{Name: "PARAM2", Value: commonutil.StringPointer("NEW_VALUE2_1")},
				},
			},
		},
	}

	assert.Equal(t, expected, result.Get())
}
