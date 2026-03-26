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
	"encoding/json"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			Conditions: []swfapi.ScheduledWorkflowCondition{
				{
					Type:               swfapi.ScheduledWorkflowEnabled,
					Status:             corev1.ConditionTrue,
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
			Conditions: []swfapi.ScheduledWorkflowCondition{
				{
					Type:               swfapi.ScheduledWorkflowEnabled,
					Status:             corev1.ConditionTrue,
					LastProbeTime:      metav1.NewTime(time.Unix(10, 0).UTC()),
					LastTransitionTime: metav1.NewTime(time.Unix(20, 0).UTC()),
					Reason:             string(swfapi.ScheduledWorkflowEnabled),
					Message:            "The schedule is enabled.",
				}, {
					Type:               swfapi.ScheduledWorkflowDisabled,
					Status:             corev1.ConditionTrue,
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
	spec, err := json.Marshal(workflowapi.WorkflowSpec{
		ServiceAccountName: "SERVICE_ACCOUNT",
		Arguments: workflowapi.Arguments{
			Parameters: []workflowapi.Parameter{
				{Name: "PARAM3", Value: workflowapi.AnyStringPtr("VALUE3")},
				{Name: "PARAM4", Value: workflowapi.AnyStringPtr("VALUE4")},
			},
		},
	})
	assert.Nil(t, err)

	// v2 runtime config's string parameter
	workflowV2 := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "STRING_PARAM1", Value: "\"ONE\""},
				},
				Spec: string(spec),
			},
		},
	})
	resultV2, err := workflowV2.ParametersAsString()
	assert.Nil(t, err)
	assert.Equal(t,
		"{\"STRING_PARAM1\":\"ONE\"}",
		resultV2)

	// v2 runtime config's numeric parameter
	workflowV2 = NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v2beta1",
			Kind:       "ScheduledWorkflow",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "NUMERIC_PARAM2", Value: "2"},
				},
				Spec: string(spec),
			},
		},
	})
	resultV2, err = workflowV2.ParametersAsString()
	assert.Nil(t, err)
	assert.Equal(t,
		"{\"NUMERIC_PARAM2\":2}",
		resultV2)

	// v1 parameters
	workflowV1 := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1beta1",
			Kind:       "ScheduledWorkflow",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "one"},
					{Name: "PARAM2", Value: "2"},
				},
				Spec: string(spec),
			},
		},
	})
	resultV1, err := workflowV1.ParametersAsString()
	assert.Nil(t, err)
	assert.Equal(t,
		`[{"name":"PARAM1","value":"one"},{"name":"PARAM2","value":"2"}]`,
		resultV1)
	// No params
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		Spec: swfapi.ScheduledWorkflowSpec{},
	})

	result, err := workflow.ParametersAsString()
	assert.Nil(t, err)

	assert.Equal(t, "", result)
}

func TestScheduledWorkflow_MaxConcurrencyOr0(t *testing.T) {
	testCases := []struct {
		name           string
		maxConcurrency *int64
		expected       int64
	}{
		{
			name:           "MaxConcurrency set returns value",
			maxConcurrency: Int64Pointer(5),
			expected:       5,
		},
		{
			name:           "MaxConcurrency nil returns 0",
			maxConcurrency: nil,
			expected:       0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				Spec: swfapi.ScheduledWorkflowSpec{
					MaxConcurrency: testCase.maxConcurrency,
				},
			})
			assert.Equal(t, testCase.expected, workflow.MaxConcurrencyOr0())
		})
	}
}

func TestScheduledWorkflow_NoCatchupOrFalse(t *testing.T) {
	testCases := []struct {
		name      string
		noCatchup *bool
		expected  bool
	}{
		{
			name:      "NoCatchup true returns true",
			noCatchup: BoolPointer(true),
			expected:  true,
		},
		{
			name:      "NoCatchup false returns false",
			noCatchup: BoolPointer(false),
			expected:  false,
		},
		{
			name:      "NoCatchup nil returns false",
			noCatchup: nil,
			expected:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				Spec: swfapi.ScheduledWorkflowSpec{
					NoCatchup: testCase.noCatchup,
				},
			})
			assert.Equal(t, testCase.expected, workflow.NoCatchupOrFalse())
		})
	}
}

func TestScheduledWorkflow_ToStringForStore(t *testing.T) {
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-schedule",
			Namespace: "default",
		},
	})
	result := workflow.ToStringForStore()
	assert.Contains(t, result, "my-schedule")
	assert.Contains(t, result, "default")
}

func TestScheduledWorkflow_ParametersAsString_Legacy(t *testing.T) {
	// Legacy (empty type metadata) → error
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "VALUE1"},
				},
			},
		},
	})
	result, err := workflow.ParametersAsString()
	assert.NotNil(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "empty type metadata")
}

func TestScheduledWorkflow_ParametersAsString_Unknown(t *testing.T) {
	// Unknown type → error
	workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "something.else/v1",
			Kind:       "SomethingElse",
		},
		Spec: swfapi.ScheduledWorkflowSpec{
			Workflow: &swfapi.WorkflowResource{
				Parameters: []swfapi.Parameter{
					{Name: "PARAM1", Value: "VALUE1"},
				},
			},
		},
	})
	result, err := workflow.ParametersAsString()
	assert.NotNil(t, err)
	assert.Empty(t, result)
}

func TestScheduledWorkflow_Get(t *testing.T) {
	swf := &swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-schedule"},
	}
	workflow := NewScheduledWorkflow(swf)
	assert.Equal(t, swf, workflow.Get())
}

func TestScheduledWorkflow_GetVersion(t *testing.T) {
	testCases := []struct {
		name       string
		apiVersion string
		kind       string
		expected   ScheduledWorkflowType
	}{
		{
			name:       "v1beta1 returns SWFv1",
			apiVersion: "kubeflow.org/v1beta1",
			kind:       "ScheduledWorkflow",
			expected:   SWFv1,
		},
		{
			name:       "v2beta1 returns SWFv2",
			apiVersion: "kubeflow.org/v2beta1",
			kind:       "ScheduledWorkflow",
			expected:   SWFv2,
		},
		{
			name:       "empty returns SWFlegacy",
			apiVersion: "",
			kind:       "",
			expected:   SWFlegacy,
		},
		{
			name:       "unknown returns SWFunknown",
			apiVersion: "something.else/v1",
			kind:       "SomethingElse",
			expected:   SWFunknown,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: testCase.apiVersion,
					Kind:       testCase.kind,
				},
			})
			assert.Equal(t, testCase.expected, workflow.GetVersion())
		})
	}
}
