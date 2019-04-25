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
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func TestScheduledWorkflow_Getters(t *testing.T) {
	// Base case
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{
				CronSchedule: &swfapi.CronSchedule{
					StartTime: MetaV1TimePointer(metav1.NewTime(time.Unix(10, 0).UTC())),
					EndTime:   MetaV1TimePointer(metav1.NewTime(time.Unix(20, 0).UTC())),
					Cron:      "MY_CRON",
				},
				PeriodicSchedule: &swfapi.PeriodicSchedule{
					StartTime:      MetaV1TimePointer(metav1.NewTime(time.Unix(30, 0).UTC())),
					EndTime:        MetaV1TimePointer(metav1.NewTime(time.Unix(40, 0).UTC())),
					IntervalSecond: 50,
				},
			},
		},
	})
	assert.Equal(t, Int64Pointer(10), workflow.CronScheduleStartTimeInSecOrNull())
	assert.Equal(t, Int64Pointer(20), workflow.CronScheduleEndTimeInSecOrNull())
	assert.Equal(t, "MY_CRON", workflow.CronOrEmpty())
	assert.Equal(t, Int64Pointer(30), workflow.PeriodicScheduleStartTimeInSecOrNull())
	assert.Equal(t, Int64Pointer(40), workflow.PeriodicScheduleEndTimeInSecOrNull())
	assert.Equal(t, int64(50), workflow.IntervalSecondOr0())

	// Values unspecified
	workflow = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Trigger: swfapi.Trigger{},
		},
	})
	assert.Equal(t, (*int64)(nil), workflow.CronScheduleStartTimeInSecOrNull())
	assert.Equal(t, (*int64)(nil), workflow.CronScheduleEndTimeInSecOrNull())
	assert.Equal(t, "", workflow.CronOrEmpty())
	assert.Equal(t, (*int64)(nil), workflow.PeriodicScheduleStartTimeInSecOrNull())
	assert.Equal(t, (*int64)(nil), workflow.PeriodicScheduleEndTimeInSecOrNull())
	assert.Equal(t, int64(0), workflow.IntervalSecondOr0())

}

func TestScheduledWorkflow_ConditionSummary(t *testing.T) {
	// Base case
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:               swfapi.ScheduledWorkflowEnabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			},
			},
		},
	})
	assert.Equal(t, "Enabled", workflow.ConditionSummary())

	// Multiple conditions
	workflow = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{
				Type:               swfapi.ScheduledWorkflowEnabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			}, {
				Type:               swfapi.ScheduledWorkflowDisabled,
				Status:             core.ConditionTrue,
				LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
				LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
				Reason:             string(swfapi.ScheduledWorkflowEnabled),
				Message:            "The schedule is enabled.",
			},
			},
		},
	})
	assert.Equal(t, "Disabled", workflow.ConditionSummary())

	// No conditions
	workflow = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{},
		},
	})
	assert.Equal(t, "NO_STATUS", workflow.ConditionSummary())
}

func TestScheduledWorkflow_ParametersAsString(t *testing.T) {
	// Base case
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "NEW_VALUE1"},
					{Name: "PARAM2", Value: "NEW_VALUE2"},
				},
				Spec: workflowapi.WorkflowSpec{
					ServiceAccountName: "SERVICE_ACCOUNT",
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM3", Value: StringPointer("VALUE3")},
							{Name: "PARAM4", Value: StringPointer("VALUE4")},
						},
					},
				},
			},
		},
	})

	result, err := workflow.ParametersAsString()
	assert.Nil(t, err)

	assert.Equal(t,
		"[{\"name\":\"PARAM1\",\"value\":\"NEW_VALUE1\"},{\"name\":\"PARAM2\",\"value\":\"NEW_VALUE2\"}]",
		result)

	// No params
	workflow = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{},
	})

	result, err = workflow.ParametersAsString()
	assert.Nil(t, err)

	assert.Equal(t, "[]", result)

}
