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

package client

import (
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowcommon "github.com/argoproj/argo/workflow/common"
	swfapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/kubeflow/pipelines/resources/scheduledworkflow/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"testing"
	"time"
)

func TestToWorkflowStatuses(t *testing.T) {
	workflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "WORKFLOW_NAME",
			Namespace:         "NAMESPACE",
			SelfLink:          "SELF_LINK",
			UID:               "UID",
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				util.LabelKeyWorkflowEpoch: "54",
				util.LabelKeyWorkflowIndex: "55",
			},
		},
		Status: workflowapi.WorkflowStatus{
			Phase:      workflowapi.NodeRunning,
			Message:    "WORKFLOW_MESSAGE",
			StartedAt:  metav1.NewTime(time.Unix(51, 0).UTC()),
			FinishedAt: metav1.NewTime(time.Unix(52, 0).UTC()),
		},
	}

	result := toWorkflowStatuses([]*workflowapi.Workflow{workflow})

	expected := &swfapi.WorkflowStatus{
		Name:        "WORKFLOW_NAME",
		Namespace:   "NAMESPACE",
		SelfLink:    "SELF_LINK",
		UID:         "UID",
		Phase:       workflowapi.NodeRunning,
		Message:     "WORKFLOW_MESSAGE",
		CreatedAt:   metav1.NewTime(time.Unix(50, 0).UTC()),
		StartedAt:   metav1.NewTime(time.Unix(51, 0).UTC()),
		FinishedAt:  metav1.NewTime(time.Unix(52, 0).UTC()),
		ScheduledAt: metav1.NewTime(time.Unix(54, 0).UTC()),
		Index:       55,
	}

	assert.Equal(t, []swfapi.WorkflowStatus{*expected}, result)
}

func TestToWorkflowStatuses_NullOrEmpty(t *testing.T) {
	workflow := &workflowapi.Workflow{}

	result := toWorkflowStatuses([]*workflowapi.Workflow{workflow})

	expected := &swfapi.WorkflowStatus{
		Name:        "",
		Namespace:   "",
		SelfLink:    "",
		UID:         "",
		Phase:       "",
		Message:     "",
		CreatedAt:   metav1.NewTime(time.Time{}.UTC()),
		StartedAt:   metav1.NewTime(time.Time{}.UTC()),
		FinishedAt:  metav1.NewTime(time.Time{}.UTC()),
		ScheduledAt: metav1.NewTime(time.Time{}.UTC()),
		Index:       0,
	}

	assert.Equal(t, []swfapi.WorkflowStatus{*expected}, result)
}

func TestRetrieveScheduledTime(t *testing.T) {

	// Base case.
	workflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				util.LabelKeyWorkflowEpoch: "54",
			},
		},
	}
	result := retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(54, 0).UTC()), result)

	// No label
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				"WRONG_LABEL": "54",
			},
		},
	}
	result = retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(50, 0).UTC()), result)

	// Parsing problem
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Unix(50, 0).UTC()),
			Labels: map[string]string{
				util.LabelKeyWorkflowEpoch: "UNPARSABLE_@%^%@^#%",
			},
		},
	}
	result = retrieveScheduledTime(workflow)
	assert.Equal(t, metav1.NewTime(time.Unix(50, 0).UTC()), result)
}

func TestRetrieveIndex(t *testing.T) {

	// Base case.
	workflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				util.LabelKeyWorkflowIndex: "100",
			},
		},
	}
	result := retrieveIndex(workflow)
	assert.Equal(t, int64(100), result)

	// No label
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"WRONG_LABEL": "100",
			},
		},
	}
	result = retrieveIndex(workflow)
	assert.Equal(t, int64(0), result)

	// Parsing problem
	workflow = &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				util.LabelKeyWorkflowIndex: "UNPARSABLE_LABEL_!@^^!%@#%",
			},
		},
	}
	result = retrieveIndex(workflow)
	assert.Equal(t, int64(0), result)
}

func TestLabelSelectorToGetWorkflows(t *testing.T) {

	// Completed
	result := getLabelSelectorToGetWorkflows(
		"PIPELINE_NAME",
		true, /* completed */
		50 /* min index */)

	expected := labels.NewSelector()

	req, err := labels.NewRequirement(workflowcommon.LabelKeyCompleted, selection.Equals,
		[]string{"true"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	req, err = labels.NewRequirement(util.LabelKeyWorkflowScheduledWorkflowName, selection.Equals,
		[]string{"PIPELINE_NAME"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	req, err = labels.NewRequirement(util.LabelKeyWorkflowIndex, selection.GreaterThan,
		[]string{"50"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	assert.Equal(t, expected, *result)

	// Not completed
	result = getLabelSelectorToGetWorkflows(
		"PIPELINE_NAME",
		false, /* completed */
		50 /* min index */)

	expected = labels.NewSelector()

	req, err = labels.NewRequirement(workflowcommon.LabelKeyCompleted, selection.NotEquals,
		[]string{"true"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	req, err = labels.NewRequirement(util.LabelKeyWorkflowScheduledWorkflowName, selection.Equals,
		[]string{"PIPELINE_NAME"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	req, err = labels.NewRequirement(util.LabelKeyWorkflowIndex, selection.GreaterThan,
		[]string{"50"})
	assert.Nil(t, err)
	expected = expected.Add(*req)

	assert.Equal(t, expected, *result)
}
