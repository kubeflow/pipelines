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
	scheduleapi "github.com/kubeflow/pipelines/pkg/apis/schedule/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestWorkflowWrap_OverrideName(t *testing.T) {
	workflow := NewWorkflowWrap(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.OverrideName("NEW_WORKFLOW_NAME")

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "NEW_WORKFLOW_NAME",
		},
	}

	assert.Equal(t, expected, workflow.Workflow())
}

func TestWorkflowWrap_OverrideParameters(t *testing.T) {
	workflow := NewWorkflowWrap(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: StringPointer("VALUE1")},
					{Name: "PARAM2", Value: StringPointer("VALUE2")},
					{Name: "PARAM3", Value: StringPointer("VALUE3")},
					{Name: "PARAM4", Value: StringPointer("")},
					{Name: "PARAM5", Value: StringPointer("VALUE5")},
				},
			},
		},
	})

	workflow.OverrideParameters(map[string]string{
		"PARAM1": "NEW_VALUE1",
		"PARAM3": "NEW_VALUE3",
		"PARAM4": "NEW_VALUE4",
		"PARAM5": "",
		"PARAM9": "NEW_VALUE9",
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: StringPointer("NEW_VALUE1")},
					{Name: "PARAM2", Value: StringPointer("VALUE2")},
					{Name: "PARAM3", Value: StringPointer("NEW_VALUE3")},
					{Name: "PARAM4", Value: StringPointer("NEW_VALUE4")},
					{Name: "PARAM5", Value: nil},
				},
			},
		},
	}
	assert.Equal(t, expected, workflow.Workflow())
}

func TestWorkflowWrap_SetCanonicalLabels(t *testing.T) {
	workflow := NewWorkflowWrap(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	const index = 50
	const nextScheduledEpoch = 100
	workflow.SetCanonicalLabels("PIPELINE_NAME", nextScheduledEpoch, index)

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"schedules.kubeflow.org/isOwnedBySchedule": "true",
				"schedules.kubeflow.org/scheduleName":      "PIPELINE_NAME",
				"schedules.kubeflow.org/scheduledEpoch":    "100",
				"schedules.kubeflow.org/workflowIndex":     "50"},
		},
	}

	assert.Equal(t, expected, workflow.Workflow())
}

func TestWorkflowWrap_SetOwnerReferences(t *testing.T) {
	workflow := NewWorkflowWrap(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetOwnerReferences(&scheduleapi.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE_NAME",
		},
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Kind:               "Schedule",
				Name:               "SCHEDULE_NAME",
				Controller:         BooleanPointer(true),
				BlockOwnerDeletion: BooleanPointer(true),
			}},
		},
	}

	assert.Equal(t, expected, workflow.Workflow())
}
