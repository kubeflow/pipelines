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

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestWorkflow_ScheduledWorkflowUUIDAsStringOrEmpty(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "MY_UID", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, true, workflow.HasScheduledWorkflowAsParent())

	// No kind
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// Wrong kind
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "WRONG_KIND",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// No API version
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ScheduledWorkflow",
				Name: "SCHEDULE_NAME",
				UID:  types.UID("MY_UID"),
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

	// No UID
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
			}},
		},
	})
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())

}

func TestWorkflow_ScheduledAtInSecOr0(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULED_WORKFLOW_NAME",
				"scheduledworkflows.kubeflow.org/workflowEpoch":              "100",
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50"},
		},
	})
	assert.Equal(t, int64(100), workflow.ScheduledAtInSecOr0())

	// No scheduled epoch
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow": "true",
				"scheduledworkflows.kubeflow.org/scheduledWorkflowName":      "SCHEDULED_WORKFLOW_NAME",
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50"},
		},
	})
	assert.Equal(t, int64(0), workflow.ScheduledAtInSecOr0())

	// No map
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})
	assert.Equal(t, int64(0), workflow.ScheduledAtInSecOr0())
}

func TestCondition(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.NodeRunning,
		},
	})
	assert.Equal(t, "Running", workflow.Condition())

	// No status
	workflow = NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{},
	})
	assert.Equal(t, "", workflow.Condition())
}

func TestToStringForStore(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})
	assert.Equal(t,
		"{\"metadata\":{\"name\":\"WORKFLOW_NAME\",\"creationTimestamp\":null},\"spec\":{\"templates\":null,\"entrypoint\":\"\",\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		workflow.ToStringForStore())
}

func TestWorkflow_OverrideName(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
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

	assert.Equal(t, expected, workflow.Get())
}

func TestWorkflow_OverrideParameters(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
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
					{Name: "PARAM5", Value: StringPointer("")},
				},
			},
		},
	}
	assert.Equal(t, expected, workflow.Get())
}

func TestWorkflow_SetOwnerReferences(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetOwnerReferences(&swfapi.ScheduledWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "SCHEDULE_NAME",
		},
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Kind:               "ScheduledWorkflow",
				Name:               "SCHEDULE_NAME",
				Controller:         BoolPointer(true),
				BlockOwnerDeletion: BoolPointer(true),
			}},
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

func TestSetLabels(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetLabels("key", "value")

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

func TestGetSpec(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: StringPointer("VALUE")},
				},
			},
		},
		Status: workflowapi.WorkflowStatus{
			Message: "I AM A MESSAGE",
		},
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: StringPointer("VALUE")},
				},
			},
		},
	}

	assert.Equal(t, expected, workflow.GetSpec().Get())
}

func TestVerifyParameters(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: StringPointer("NEW_VALUE1")},
					{Name: "PARAM2", Value: StringPointer("VALUE2")},
					{Name: "PARAM3", Value: StringPointer("NEW_VALUE3")},
					{Name: "PARAM5", Value: StringPointer("")},
				},
			},
		},
	})
	assert.Nil(t, workflow.VerifyParameters(map[string]string{"PARAM1": "V1", "PARAM2": "V2"}))
}

func TestVerifyParameters_Failed(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: StringPointer("NEW_VALUE1")},
					{Name: "PARAM2", Value: StringPointer("VALUE2")},
					{Name: "PARAM3", Value: StringPointer("NEW_VALUE3")},
					{Name: "PARAM5", Value: StringPointer("")},
				},
			},
		},
	})
	assert.NotNil(t, workflow.VerifyParameters(map[string]string{"PARAM1": "V1", "NON_EXIST": "V2"}))
}

func TestFindS3ArtifactKey_Succeed(t *testing.T) {
	expectedPath := "expected/path"
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{
							workflowapi.Artifact{
								Name: "artifact-1",
								ArtifactLocation: workflowapi.ArtifactLocation{
									S3: &workflowapi.S3Artifact{
										Key: expectedPath,
									},
								},
							},
						},
					},
				},
			},
		},
	})

	actualPath := workflow.FindObjectStoreArtifactKeyOrEmpty("node-1", "artifact-1")

	assert.Equal(t, expectedPath, actualPath)
}

func TestFindS3ArtifactKey_ArtifactNotFound(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": workflowapi.NodeStatus{
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{
							workflowapi.Artifact{
								Name: "artifact-2",
								ArtifactLocation: workflowapi.ArtifactLocation{
									S3: &workflowapi.S3Artifact{
										Key: "foo/bar",
									},
								},
							},
						},
					},
				},
			},
		},
	})

	actualPath := workflow.FindObjectStoreArtifactKeyOrEmpty("node-1", "artifact-1")

	assert.Empty(t, actualPath)
}

func TestFindS3ArtifactKey_NodeNotFound(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{},
		},
	})

	actualPath := workflow.FindObjectStoreArtifactKeyOrEmpty("node-1", "artifact-1")

	assert.Empty(t, actualPath)
}
