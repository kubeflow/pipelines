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
	"context"
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	argoinformer "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	argolister "github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client/artifactclient"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"
)

func TestWorkflow_NewWorkflowFromBytes(t *testing.T) {
	// Error case
	workflow, err := NewWorkflowFromBytes([]byte("this is invalid format"))
	assert.Empty(t, workflow)
	assert.Error(t, err)
	assert.EqualError(t, err,
		"InvalidInputError: Failed to unmarshal the inputs: "+
			"error unmarshaling JSON: while decoding JSON: json: cannot unmarshal "+
			"string into Go value of type v1alpha1.Workflow")

	// Normal case
	bytes, err := yaml.Marshal(workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
		Status: workflowapi.WorkflowStatus{
			Message: "I AM A MESSAGE",
		},
	})
	assert.Empty(t, err)
	assert.NotEmpty(t, bytes)

	workflow, err = NewWorkflowFromBytes(bytes)
	assert.Empty(t, err)
	assert.NotEmpty(t, workflow)
}

func TestWorkflow_NewWorkflowFromInterface(t *testing.T) {
	// Error case
	workflow, err := NewWorkflowFromInterface("this is invalid format")
	assert.Empty(t, workflow)
	assert.Error(t, err)
	assert.EqualError(t, err,
		NewInvalidInputError("not Workflow struct").Error())

	// Normal case
	workflow, err = NewWorkflowFromInterface(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
		Status: workflowapi.WorkflowStatus{
			Message: "I AM A MESSAGE",
		},
	})
	assert.Empty(t, err)
	assert.NotEmpty(t, workflow)
}

func TestWorkflow_ScheduledWorkflowUUIDAsStringOrEmpty(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
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
				APIVersion: "kubeflow.org/v1beta1",
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
				APIVersion: "kubeflow.org/v1beta1",
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
				APIVersion: "kubeflow.org/v1beta1",
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
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50",
			},
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
				"scheduledworkflows.kubeflow.org/workflowIndex":              "50",
			},
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

	// Invalid epoch value (non-numeric)
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
			Labels: map[string]string{
				"scheduledworkflows.kubeflow.org/workflowEpoch": "not-a-number",
			},
		},
	})
	assert.Equal(t, int64(0), workflow.ScheduledAtInSecOr0())
}

func TestCondition(t *testing.T) {
	// Base case
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})
	assert.Equal(t, "Running", string(workflow.Condition()))

	// No status
	workflow = NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{},
	})
	assert.Equal(t, "", string(workflow.Condition()))
}

func TestToStringForStore(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})
	assert.Equal(t,
		"{\"metadata\":{\"name\":\"WORKFLOW_NAME\"},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		workflow.ToStringForStore())
}

func TestToStringForSchedule(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})
	assert.Equal(t,
		"{\"metadata\":{\"name\":\"WORKFLOW_NAME\"},\"spec\":{\"arguments\":{}},\"status\":{\"startedAt\":null,\"finishedAt\":null}}",
		workflow.ToStringForSchedule())
}

func TestWorkflow_OverrideName(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
	})

	workflow.SetExecutionName("NEW_WORKFLOW_NAME")

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "NEW_WORKFLOW_NAME",
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

func stringToPointer(str string) *string {
	return &str
}

func TestWorkflow_SpecParameters(t *testing.T) {
	execSpec, err := NewExecutionSpecFromInterface(ArgoWorkflow, &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
					{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
					{Name: "PARAM3", Value: workflowapi.AnyStringPtr("VALUE3")},
					{Name: "PARAM4", Value: workflowapi.AnyStringPtr("")},
					{Name: "PARAM5", Value: workflowapi.AnyStringPtr("VALUE5")},
				},
			},
		},
	})

	assert.Nil(t, err)
	expectedParam := SpecParameters{
		SpecParameter{Name: "PARAM1", Value: stringToPointer("VALUE1")},
		SpecParameter{Name: "PARAM2", Value: stringToPointer("VALUE2")},
		SpecParameter{Name: "PARAM3", Value: stringToPointer("VALUE3")},
		SpecParameter{Name: "PARAM4", Value: stringToPointer("")},
		SpecParameter{Name: "PARAM5", Value: stringToPointer("VALUE5")},
	}
	assert.Equal(t, execSpec.SpecParameters(), expectedParam)
}

func TestWorkflow_SetSpecParameters(t *testing.T) {
	execSpec, err := NewExecutionSpecFromInterface(ArgoWorkflow, &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
					{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
					{Name: "PARAM3", Value: workflowapi.AnyStringPtr("VALUE3")},
					{Name: "PARAM4", Value: workflowapi.AnyStringPtr("")},
					{Name: "PARAM5", Value: workflowapi.AnyStringPtr("VALUE5")},
				},
			},
		},
	})

	assert.Nil(t, err)

	newParams := SpecParameters{
		SpecParameter{Name: "PARAM1", Value: stringToPointer("NEW_VALUE1")},
		SpecParameter{Name: "PARAM2", Value: stringToPointer("NEW_VALUE2")},
		SpecParameter{Name: "PARAM3", Value: stringToPointer("VALUE3")},
		SpecParameter{Name: "PARAM4", Value: stringToPointer("")},
	}

	execSpec.SetSpecParameters(newParams)
	assert.Equal(t, execSpec.SpecParameters(), newParams)
}

func TestWorkflow_OverrideParameters(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *workflowapi.Workflow
		overrides map[string]string
		expected  *workflowapi.Workflow
	}{
		{
			name: "override parameters",
			workflow: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "WORKFLOW_NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
							{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
							{Name: "PARAM3", Value: workflowapi.AnyStringPtr("VALUE3")},
							{Name: "PARAM4", Value: workflowapi.AnyStringPtr("")},
							{Name: "PARAM5", Value: workflowapi.AnyStringPtr("VALUE5")},
						},
					},
				},
			},
			overrides: map[string]string{
				"PARAM1": "NEW_VALUE1",
				"PARAM3": "NEW_VALUE3",
				"PARAM4": "NEW_VALUE4",
				"PARAM5": "",
				"PARAM9": "NEW_VALUE9",
			},
			expected: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "WORKFLOW_NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1", Value: workflowapi.AnyStringPtr("NEW_VALUE1")},
							{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
							{Name: "PARAM3", Value: workflowapi.AnyStringPtr("NEW_VALUE3")},
							{Name: "PARAM4", Value: workflowapi.AnyStringPtr("NEW_VALUE4")},
							{Name: "PARAM5", Value: workflowapi.AnyStringPtr("")},
						},
					},
				},
			},
		},
		{
			name: "handles missing parameter values",
			workflow: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1"}, // note, there's no value here
						},
					},
				},
			},
			overrides: nil,
			expected: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1"},
						},
					},
				},
			},
		},
		{
			name: "overrides a missing parameter value",
			workflow: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1"}, // note, there's no value here
						},
					},
				},
			},
			overrides: map[string]string{
				"PARAM1": "VALUE1",
			},
			expected: &workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "NAME",
				},
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{
						Parameters: []workflowapi.Parameter{
							{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := NewWorkflow(tt.workflow)
			workflow.OverrideParameters(tt.overrides)
			assert.Equal(t, tt.expected, workflow.Get())
		})
	}
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
				APIVersion:         "kubeflow.org/v1beta1",
				Kind:               "ScheduledWorkflow",
				Name:               "SCHEDULE_NAME",
				Controller:         BoolPointer(true),
				BlockOwnerDeletion: BoolPointer(true),
			}},
		},
	}

	assert.Equal(t, expected, workflow.Get())
}

func TestWorkflow_SetLabelsToAllTemplates(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{Metadata: workflowapi.Metadata{}},
			},
		},
	})
	workflow.SetLabelsToAllTemplates("key", "value")
	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{{
				Metadata: workflowapi.Metadata{
					Labels: map[string]string{
						"key": "value",
					},
				},
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

func TestGetWorkflowSpec(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "WORKFLOW_NAME",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
		Status: workflowapi.WorkflowStatus{
			Message: "I AM A MESSAGE",
		},
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
	}

	assert.Equal(t, expected, workflow.GetExecutionSpec().(*Workflow).Get())
}

func TestGetWorkflowSpecTruncatesNameIfLongerThan200Runes(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "THIS_NAME_IS_GREATER_THAN_200_CHARACTERS_AND_WILL_BE_TRUNCATED_AFTER_THE_X_OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOXZZZZZZZZ",
			Labels: map[string]string{"key": "value"},
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
		Status: workflowapi.WorkflowStatus{
			Message: "I AM A MESSAGE",
		},
	})

	expected := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "THIS_NAME_IS_GREATER_THAN_200_CHARACTERS_AND_WILL_BE_TRUNCATED_AFTER_THE_X_OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOX",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM", Value: workflowapi.AnyStringPtr("VALUE")},
				},
			},
		},
	}

	assert.Equal(t, expected, workflow.GetExecutionSpec().(*Workflow).Get())
}

func TestVerifyParameters(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "WORKFLOW_NAME",
		},
		Spec: workflowapi.WorkflowSpec{
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: "PARAM1", Value: workflowapi.AnyStringPtr("NEW_VALUE1")},
					{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
					{Name: "PARAM3", Value: workflowapi.AnyStringPtr("NEW_VALUE3")},
					{Name: "PARAM5", Value: workflowapi.AnyStringPtr("")},
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
					{Name: "PARAM1", Value: workflowapi.AnyStringPtr("NEW_VALUE1")},
					{Name: "PARAM2", Value: workflowapi.AnyStringPtr("VALUE2")},
					{Name: "PARAM3", Value: workflowapi.AnyStringPtr("NEW_VALUE3")},
					{Name: "PARAM5", Value: workflowapi.AnyStringPtr("")},
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
				"node-1": {
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{
							Name: "artifact-1",
							ArtifactLocation: workflowapi.ArtifactLocation{
								S3: &workflowapi.S3Artifact{
									Key: expectedPath,
								},
							},
						}},
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
				"node-1": {
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{
							Name: "artifact-2",
							ArtifactLocation: workflowapi.ArtifactLocation{
								S3: &workflowapi.S3Artifact{
									Key: "foo/bar",
								},
							},
						}},
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

func TestReplaceUID(t *testing.T) {
	workflowString := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-owner-reference-
spec:
  entrypoint: k8s-owner-reference
  templates:
  - name: k8s-owner-reference
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          generateName: owned-eg-
          ownerReferences:
          - apiVersion: argoproj.io/v1alpha1
            blockOwnerDeletion: true
            kind: Workflow
            name: "{{workflow.name}}"
            uid: "{{workflow.uid}}"
        data:
          some: value`
	var workflow Workflow
	err := yaml.Unmarshal([]byte(workflowString), &workflow)
	assert.Nil(t, err)
	workflow.ReplaceUID("12345")
	expectedWorkflowString := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-owner-reference-
spec:
  entrypoint: k8s-owner-reference
  templates:
  - name: k8s-owner-reference
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          generateName: owned-eg-
          ownerReferences:
          - apiVersion: argoproj.io/v1alpha1
            blockOwnerDeletion: true
            kind: Workflow
            name: "{{workflow.name}}"
            uid: "12345"
        data:
          some: value`

	var expectedWorkflow Workflow
	err = yaml.Unmarshal([]byte(expectedWorkflowString), &expectedWorkflow)
	assert.Nil(t, err)
	assert.Equal(t, expectedWorkflow, workflow)
}

// Group A: Simple Getters/Setters
func TestWorkflow_SimpleAccessors(t *testing.T) {
	testCases := []struct {
		name   string
		verify func(t *testing.T)
	}{
		{
			name: "ExecutionType returns ArgoWorkflow",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{})
				assert.Equal(t, ArgoWorkflow, workflow.ExecutionType())
			},
		},
		{
			name: "ExecutionStatus returns self",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{})
				assert.NotNil(t, workflow.ExecutionStatus())
			},
		},
		{
			name: "ServiceAccount returns spec value",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					Spec: workflowapi.WorkflowSpec{
						ServiceAccountName: "test-sa",
					},
				})
				assert.Equal(t, "test-sa", workflow.ServiceAccount())
			},
		},
		{
			name: "SetServiceAccount updates spec value",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{})
				workflow.SetServiceAccount("new-sa")
				assert.Equal(t, "new-sa", workflow.Spec.ServiceAccountName)
			},
		},
		{
			name: "Version returns ResourceVersion",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{ResourceVersion: "42"},
				})
				assert.Equal(t, "42", workflow.Version())
			},
		},
		{
			name: "SetVersion updates ResourceVersion",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{})
				workflow.SetVersion("99")
				assert.Equal(t, "99", workflow.ResourceVersion)
			},
		},
		{
			name: "ExecutionName returns Name",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: "my-workflow"},
				})
				assert.Equal(t, "my-workflow", workflow.ExecutionName())
			},
		},
		{
			name: "ExecutionNamespace returns Namespace",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{Namespace: "my-namespace"},
				})
				assert.Equal(t, "my-namespace", workflow.ExecutionNamespace())
			},
		},
		{
			name: "SetExecutionNamespace updates Namespace",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{})
				workflow.SetExecutionNamespace("new-ns")
				assert.Equal(t, "new-ns", workflow.Namespace)
			},
		},
		{
			name: "ExecutionUID returns string of UID",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{UID: types.UID("abc-123")},
				})
				assert.Equal(t, "abc-123", workflow.ExecutionUID())
			},
		},
		{
			name: "ExecutionObjectMeta returns correct ObjectMeta",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
				})
				objectMeta := workflow.ExecutionObjectMeta()
				assert.Equal(t, "test-wf", objectMeta.Name)
			},
		},
		{
			name: "ExecutionTypeMeta returns correct TypeMeta",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					TypeMeta: metav1.TypeMeta{Kind: "Workflow", APIVersion: "argoproj.io/v1alpha1"},
				})
				typeMeta := workflow.ExecutionTypeMeta()
				assert.Equal(t, "Workflow", typeMeta.Kind)
				assert.Equal(t, "argoproj.io/v1alpha1", typeMeta.APIVersion)
			},
		},
		{
			name: "Message returns status message",
			verify: func(t *testing.T) {
				workflow := NewWorkflow(&workflowapi.Workflow{
					Status: workflowapi.WorkflowStatus{Message: "some error occurred"},
				})
				assert.Equal(t, "some error occurred", workflow.Message())
			},
		},
		{
			name: "FinishedAtTime returns status finished time",
			verify: func(t *testing.T) {
				finishedTime := metav1.NewTime(time.Unix(1000, 0).UTC())
				workflow := NewWorkflow(&workflowapi.Workflow{
					Status: workflowapi.WorkflowStatus{FinishedAt: finishedTime},
				})
				assert.Equal(t, finishedTime, workflow.FinishedAtTime())
			},
		},
		{
			name: "StartedAtTime returns status started time",
			verify: func(t *testing.T) {
				startedTime := metav1.NewTime(time.Unix(500, 0).UTC())
				workflow := NewWorkflow(&workflowapi.Workflow{
					Status: workflowapi.WorkflowStatus{StartedAt: startedTime},
				})
				assert.Equal(t, startedTime, workflow.StartedAtTime())
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.verify(t)
		})
	}
}

// Group B: State Check Methods
func TestWorkflow_IsInFinalState(t *testing.T) {
	testCases := []struct {
		name     string
		phase    workflowapi.WorkflowPhase
		expected bool
	}{
		{"Succeeded is final", workflowapi.WorkflowSucceeded, true},
		{"Failed is final", workflowapi.WorkflowFailed, true},
		{"Error is final", workflowapi.WorkflowError, true},
		{"Running is not final", workflowapi.WorkflowRunning, false},
		{"Pending is not final", workflowapi.WorkflowPending, false},
		{"Empty phase is not final", workflowapi.WorkflowPhase(""), false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Status: workflowapi.WorkflowStatus{Phase: testCase.phase},
			})
			assert.Equal(t, testCase.expected, workflow.IsInFinalState())
		})
	}
}

func TestWorkflow_PersistedFinalState(t *testing.T) {
	testCases := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "label present returns true",
			labels:   map[string]string{LabelKeyWorkflowPersistedFinalState: "true"},
			expected: true,
		},
		{
			name:     "no labels returns false",
			labels:   nil,
			expected: false,
		},
		{
			name:     "other labels only returns false",
			labels:   map[string]string{"some-other-key": "value"},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Labels: testCase.labels},
			})
			assert.Equal(t, testCase.expected, workflow.PersistedFinalState())
		})
	}
}

func TestWorkflow_IsV2Compatible(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name:        "annotation true",
			annotations: map[string]string{"pipelines.kubeflow.org/v2_pipeline": "true"},
			expected:    true,
		},
		{
			name:        "annotation missing",
			annotations: nil,
			expected:    false,
		},
		{
			name:        "annotation false",
			annotations: map[string]string{"pipelines.kubeflow.org/v2_pipeline": "false"},
			expected:    false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Annotations: testCase.annotations},
			})
			assert.Equal(t, testCase.expected, workflow.IsV2Compatible())
		})
	}
}

func TestWorkflow_HasMetrics(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    map[string]workflowapi.NodeStatus
		expected bool
	}{
		{
			name:     "non-nil nodes returns true",
			nodes:    map[string]workflowapi.NodeStatus{"node-1": {}},
			expected: true,
		},
		{
			name:     "nil nodes returns false",
			nodes:    nil,
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Status: workflowapi.WorkflowStatus{Nodes: testCase.nodes},
			})
			assert.Equal(t, testCase.expected, workflow.HasMetrics())
		})
	}
}

func TestWorkflow_HasNodes(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    map[string]workflowapi.NodeStatus
		expected bool
	}{
		{
			name:     "populated nodes returns true",
			nodes:    map[string]workflowapi.NodeStatus{"node-1": {}},
			expected: true,
		},
		{
			name:     "empty map returns false",
			nodes:    map[string]workflowapi.NodeStatus{},
			expected: false,
		},
		{
			name:     "nil returns false",
			nodes:    nil,
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Status: workflowapi.WorkflowStatus{Nodes: testCase.nodes},
			})
			assert.Equal(t, testCase.expected, workflow.HasNodes())
		})
	}
}

func TestWorkflow_IsTerminating(t *testing.T) {
	zero := int64(0)
	nonZero := int64(100)

	testCases := []struct {
		name                  string
		activeDeadlineSeconds *int64
		phase                 workflowapi.WorkflowPhase
		expected              bool
	}{
		{
			name:                  "deadline 0 and running is terminating",
			activeDeadlineSeconds: &zero,
			phase:                 workflowapi.WorkflowRunning,
			expected:              true,
		},
		{
			name:                  "nil deadline is not terminating",
			activeDeadlineSeconds: nil,
			phase:                 workflowapi.WorkflowRunning,
			expected:              false,
		},
		{
			name:                  "non-zero deadline is not terminating",
			activeDeadlineSeconds: &nonZero,
			phase:                 workflowapi.WorkflowRunning,
			expected:              false,
		},
		{
			name:                  "in final state is not terminating",
			activeDeadlineSeconds: &zero,
			phase:                 workflowapi.WorkflowSucceeded,
			expected:              false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Spec: workflowapi.WorkflowSpec{
					ActiveDeadlineSeconds: testCase.activeDeadlineSeconds,
				},
				Status: workflowapi.WorkflowStatus{Phase: testCase.phase},
			})
			assert.Equal(t, testCase.expected, workflow.IsTerminating())
		})
	}
}

// Group C: Logic Methods
func TestWorkflow_FinishedAt(t *testing.T) {
	testCases := []struct {
		name       string
		finishedAt metav1.Time
		expected   int64
	}{
		{
			name:       "zero FinishedAt returns 0",
			finishedAt: metav1.Time{},
			expected:   0,
		},
		{
			name:       "non-zero FinishedAt returns Unix timestamp",
			finishedAt: metav1.NewTime(time.Unix(1234567890, 0).UTC()),
			expected:   1234567890,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Status: workflowapi.WorkflowStatus{FinishedAt: testCase.finishedAt},
			})
			assert.Equal(t, testCase.expected, workflow.FinishedAt())
		})
	}
}

func TestWorkflow_GetWorkflowParametersAsMap(t *testing.T) {
	testCases := []struct {
		name     string
		params   []workflowapi.Parameter
		expected map[string]string
	}{
		{
			name: "params with values",
			params: []workflowapi.Parameter{
				{Name: "param1", Value: workflowapi.AnyStringPtr("value1")},
				{Name: "param2", Value: workflowapi.AnyStringPtr("value2")},
			},
			expected: map[string]string{"param1": "value1", "param2": "value2"},
		},
		{
			name: "param with nil value returns empty string",
			params: []workflowapi.Parameter{
				{Name: "param1"},
			},
			expected: map[string]string{"param1": ""},
		},
		{
			name:     "no params returns empty map",
			params:   nil,
			expected: map[string]string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workflow := NewWorkflow(&workflowapi.Workflow{
				Spec: workflowapi.WorkflowSpec{
					Arguments: workflowapi.Arguments{Parameters: testCase.params},
				},
			})
			assert.Equal(t, testCase.expected, workflow.GetWorkflowParametersAsMap())
		})
	}
}

func TestWorkflow_SetAnnotations(t *testing.T) {
	// nil Annotations → initialized and key set
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{},
	})
	workflow.SetAnnotations("key1", "value1")
	assert.Equal(t, "value1", workflow.Annotations["key1"])

	// existing Annotations → key added
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"existing": "val"},
		},
	})
	workflow.SetAnnotations("key2", "value2")
	assert.Equal(t, "val", workflow.Annotations["existing"])
	assert.Equal(t, "value2", workflow.Annotations["key2"])
}

func TestWorkflow_SetPodMetadataLabels(t *testing.T) {
	// nil PodMetadata → initialized, label set
	workflow := NewWorkflow(&workflowapi.Workflow{})
	workflow.SetPodMetadataLabels("key1", "value1")
	assert.Equal(t, "value1", workflow.Spec.PodMetadata.Labels["key1"])

	// existing PodMetadata with nil Labels → initialized, label set
	workflow = NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			PodMetadata: &workflowapi.Metadata{},
		},
	})
	workflow.SetPodMetadataLabels("key2", "value2")
	assert.Equal(t, "value2", workflow.Spec.PodMetadata.Labels["key2"])
}

func TestWorkflow_SetAnnotationsToAllTemplatesIfKeyNotExist(t *testing.T) {
	// Template without annotations → annotation added
	workflow := NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{Name: "template1", Metadata: workflowapi.Metadata{}},
			},
		},
	})
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist("new-key", "new-value")
	assert.Equal(t, "new-value", workflow.Spec.Templates[0].Metadata.Annotations["new-key"])

	// Template with existing key → not overwritten
	workflow = NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{
					Name: "template1",
					Metadata: workflowapi.Metadata{
						Annotations: map[string]string{"existing-key": "original-value"},
					},
				},
			},
		},
	})
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist("existing-key", "new-value")
	assert.Equal(t, "original-value", workflow.Spec.Templates[0].Metadata.Annotations["existing-key"])

	// Empty templates → no panic
	workflow = NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{Templates: nil},
	})
	workflow.SetAnnotationsToAllTemplatesIfKeyNotExist("key", "value")
}

func TestWorkflow_SetCannonicalLabels(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{},
	})
	workflow.SetCannonicalLabels("my-schedule", 1000, 5)

	assert.Equal(t, "my-schedule", workflow.Labels[LabelKeyWorkflowScheduledWorkflowName])
	assert.Equal(t, FormatInt64ForLabel(1000), workflow.Labels[LabelKeyWorkflowEpoch])
	assert.Equal(t, FormatInt64ForLabel(5), workflow.Labels[LabelKeyWorkflowIndex])
	assert.Equal(t, "true", workflow.Labels[LabelKeyWorkflowIsOwnedByScheduledWorkflow])
}

func TestWorkflow_PatchTemplateOutputArtifacts(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{
					Name: "template1",
					Outputs: workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{
							{Name: "mlpipeline-ui-metadata"},
							{Name: "mlpipeline-metrics"},
							{Name: "other-artifact"},
						},
					},
				},
			},
		},
	})

	workflow.PatchTemplateOutputArtifacts()

	assert.True(t, workflow.Spec.Templates[0].Outputs.Artifacts[0].Optional)
	assert.True(t, workflow.Spec.Templates[0].Outputs.Artifacts[1].Optional)
	assert.False(t, workflow.Spec.Templates[0].Outputs.Artifacts[2].Optional)
}

func TestWorkflow_NodeStatuses(t *testing.T) {
	startTime := metav1.NewTime(time.Unix(100, 0).UTC())
	finishTime := metav1.NewTime(time.Unix(200, 0).UTC())

	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:          "node-1",
					DisplayName: "Step 1",
					Phase:       workflowapi.NodeSucceeded,
					StartedAt:   startTime,
					FinishedAt:  finishTime,
					Children:    []string{"node-2"},
				},
			},
		},
	})

	statuses := workflow.NodeStatuses()
	assert.Len(t, statuses, 1)

	nodeStatus := statuses["node-1"]
	assert.Equal(t, "Step 1", nodeStatus.DisplayName)
	assert.Equal(t, string(workflowapi.NodeSucceeded), nodeStatus.State)
	assert.Equal(t, startTime.Unix(), nodeStatus.StartTime)
	assert.Equal(t, finishTime.Unix(), nodeStatus.FinishTime)
	assert.Equal(t, []string{"node-2"}, nodeStatus.Children)

	// Empty nodes → empty map
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status:     workflowapi.WorkflowStatus{Nodes: nil},
	})
	statuses = workflow.NodeStatuses()
	assert.Empty(t, statuses)
}

func TestWorkflow_NodeStatuses_PodMessageInfersFailed(t *testing.T) {
	startTime := metav1.NewTime(time.Unix(100, 0).UTC())
	tests := []struct {
		name      string
		node      workflowapi.NodeStatus
		wantState string
	}{
		{
			// Typical v2 Kubernetes secretAsEnv + missing Secret: task stays Pending while Message
			// carries CreateContainerConfigError (frontend must not treat this as success).
			name: "pending pod CreateContainerConfigError",
			node: workflowapi.NodeStatus{
				ID:          "n1",
				DisplayName: "print-envvar",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodePending,
				Message:     `containers with unready status: [main] (Message: secret "my-secret-that-doesnt-exist" not found: CreateContainerConfigError)`,
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "running pod CreateContainerError",
			node: workflowapi.NodeStatus{
				ID:          "n2",
				DisplayName: "step",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodeRunning,
				Message:     "failed to create container: CreateContainerError",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "pending pod ImagePullBackOff",
			node: workflowapi.NodeStatus{
				ID:          "n2b",
				DisplayName: "pull",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodePending,
				Message:     "Back-off pulling image \"x\": ImagePullBackOff",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "running pod ErrImagePull",
			node: workflowapi.NodeStatus{
				ID:          "n2c",
				DisplayName: "pull2",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodeRunning,
				Message:     "Failed to pull image: ErrImagePull",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "pending pod InvalidImageName",
			node: workflowapi.NodeStatus{
				ID:          "n2d",
				DisplayName: "badimg",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodePending,
				Message:     "InvalidImageName: invalid reference format",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "running pod CrashLoopBackOff",
			node: workflowapi.NodeStatus{
				ID:          "n2e",
				DisplayName: "crash",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodeRunning,
				Message:     "Back-off restarting failed container: CrashLoopBackOff",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "pending pod RunContainerError",
			node: workflowapi.NodeStatus{
				ID:          "n2f",
				DisplayName: "runerr",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodePending,
				Message:     "failed to start container: RunContainerError",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
		{
			name: "pending pod without fatal message",
			node: workflowapi.NodeStatus{
				ID:          "n3",
				DisplayName: "ok",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodePending,
				Message:     "ContainerCreating",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodePending),
		},
		{
			name: "dag node with CreateContainerConfigError text ignored",
			node: workflowapi.NodeStatus{
				ID:          "n4",
				DisplayName: "root",
				Type:        workflowapi.NodeTypeDAG,
				Phase:       workflowapi.NodeRunning,
				Message:     "child has CreateContainerConfigError",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeRunning),
		},
		{
			name: "pod already failed unchanged",
			node: workflowapi.NodeStatus{
				ID:          "n5",
				DisplayName: "x",
				Type:        workflowapi.NodeTypePod,
				Phase:       workflowapi.NodeFailed,
				Message:     "CreateContainerConfigError: ...",
				StartedAt:   startTime,
			},
			wantState: string(workflowapi.NodeFailed),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := NewWorkflow(&workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "wf"},
				Status: workflowapi.WorkflowStatus{
					Nodes: map[string]workflowapi.NodeStatus{"x": tt.node},
				},
			})
			ns := wf.NodeStatuses()["x"]
			assert.Equal(t, tt.wantState, ns.State)
		})
	}
}

func TestWorkflow_NodeStatuses_InferredFailureFinishTimeUsesStartedAt(t *testing.T) {
	startTime := metav1.NewTime(time.Unix(100, 0).UTC())
	node := workflowapi.NodeStatus{
		ID:          "p1",
		DisplayName: "task",
		Type:        workflowapi.NodeTypePod,
		Phase:       workflowapi.NodePending,
		Message:     "CreateContainerConfigError: mount failed",
		StartedAt:   startTime,
		// FinishedAt zero
	}
	wf := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf"},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{"x": node},
		},
	})
	ns := wf.NodeStatuses()["x"]
	assert.Equal(t, string(workflowapi.NodeFailed), ns.State)
	assert.Equal(t, startTime.Unix(), ns.FinishTime)
}

func TestWorkflow_Compare(t *testing.T) {
	client := &WorkflowClient{}

	oldWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1"},
	}
	newWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{ResourceVersion: "2"},
	}

	// Different ResourceVersions → true
	assert.True(t, client.Compare(oldWorkflow, newWorkflow))

	// Same ResourceVersion → false
	sameWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1"},
	}
	assert.False(t, client.Compare(oldWorkflow, sameWorkflow))
}

func TestRetrievePodName(t *testing.T) {
	testCases := []struct {
		name     string
		workflow workflowapi.Workflow
		node     workflowapi.NodeStatus
		expected string
	}{
		{
			name: "v1 APIVersion returns node ID",
			workflow: workflowapi.Workflow{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			},
			node:     workflowapi.NodeStatus{ID: "my-wf-node-abc123"},
			expected: "my-wf-node-abc123",
		},
		{
			name: "pod-name-format=v1 annotation returns node ID",
			workflow: workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-wf",
					Annotations: map[string]string{"workflows.argoproj.io/pod-name-format": "v1"},
				},
			},
			node:     workflowapi.NodeStatus{ID: "my-wf-node-abc123"},
			expected: "my-wf-node-abc123",
		},
		{
			name: "same name as workflow returns workflow name",
			workflow: workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			},
			node:     workflowapi.NodeStatus{ID: "my-wf-abc123", Name: "my-wf"},
			expected: "my-wf",
		},
		{
			name: "non-inline node returns name with template",
			workflow: workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			},
			node: workflowapi.NodeStatus{
				ID:           "my-wf-step1-abc123",
				Name:         "my-wf.step1",
				TemplateName: "step1",
			},
			expected: "my-wf-step1-abc123",
		},
		{
			name: "inline node with .inline in name returns wf-hash",
			workflow: workflowapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			},
			node: workflowapi.NodeStatus{
				ID:           "my-wf-12345-def456",
				Name:         "my-wf.step.inline",
				TemplateName: "step",
			},
			expected: "my-wf-def456",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := RetrievePodName(testCase.workflow, testCase.node)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestWorkflow_NewWorkflowFromScheduleWorkflowSpecBytesJSON(t *testing.T) {
	// Valid full workflow JSON
	validJSON := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"test-wf"},"spec":{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}}`
	workflow, err := NewWorkflowFromScheduleWorkflowSpecBytesJSON([]byte(validJSON))
	assert.Nil(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "test-wf", workflow.Name)
	assert.Equal(t, "main", workflow.Spec.Entrypoint)

	// Valid workflow spec JSON (no entrypoint at top level - falls back to spec unmarshal)
	specJSON := `{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}`
	workflow, err = NewWorkflowFromScheduleWorkflowSpecBytesJSON([]byte(specJSON))
	assert.Nil(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "argoproj.io/v1alpha1", workflow.APIVersion)
	assert.Equal(t, "Workflow", workflow.Kind)

	// Invalid JSON → error
	workflow, err = NewWorkflowFromScheduleWorkflowSpecBytesJSON([]byte("not valid json"))
	assert.NotNil(t, err)
	assert.Nil(t, workflow)
}

func TestWorkflow_NewWorkflowFromBytesJSON(t *testing.T) {
	// Valid JSON
	validJSON := `{"metadata":{"name":"test-wf"},"spec":{"entrypoint":"main"}}`
	workflow, err := NewWorkflowFromBytesJSON([]byte(validJSON))
	assert.Nil(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "test-wf", workflow.Name)

	// Invalid JSON → error
	workflow, err = NewWorkflowFromBytesJSON([]byte("not valid json"))
	assert.NotNil(t, err)
	assert.Nil(t, workflow)
}

func TestWorkflow_Get(t *testing.T) {
	argoWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
	}
	workflow := NewWorkflow(argoWorkflow)
	assert.Equal(t, argoWorkflow, workflow.Get())
}

func TestWorkflow_HasScheduledWorkflowAsParent_NoOwnerRefs(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "WORKFLOW_NAME",
			OwnerReferences: nil,
		},
	})
	assert.Equal(t, false, workflow.HasScheduledWorkflowAsParent())
	assert.Equal(t, "", workflow.ScheduledWorkflowUUIDAsStringOrEmpty())
}

func TestWorkflow_FindObjectStoreArtifactKeyOrEmpty_NilNodes(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{Nodes: nil},
	})
	assert.Empty(t, workflow.FindObjectStoreArtifactKeyOrEmpty("node-1", "artifact-1"))
}

func TestWorkflow_FindObjectStoreArtifactKeyOrEmpty_NilOutputs(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {Outputs: nil},
			},
		},
	})
	assert.Empty(t, workflow.FindObjectStoreArtifactKeyOrEmpty("node-1", "artifact-1"))
}

func TestUnmarshParametersWorkflow(t *testing.T) {
	// Empty string returns nil
	params, err := UnmarshParametersWorkflow("")
	assert.Nil(t, err)
	assert.Nil(t, params)

	// Valid params
	paramsJSON := `[{"name":"param1","value":"value1"},{"name":"param2","value":"value2"}]`
	params, err = UnmarshParametersWorkflow(paramsJSON)
	assert.Nil(t, err)
	assert.Len(t, params, 2)
	assert.Equal(t, "param1", params[0].Name)
	assert.Equal(t, "value1", *params[0].Value)

	// Invalid JSON
	params, err = UnmarshParametersWorkflow("invalid json")
	assert.NotNil(t, err)
	assert.Nil(t, params)
}

func TestMarshalParametersWorkflow(t *testing.T) {
	// nil params returns "[]"
	result, err := MarshalParametersWorkflow(nil)
	assert.Nil(t, err)
	assert.Equal(t, "[]", result)

	// Valid params
	value := "value1"
	params := SpecParameters{
		{Name: "param1", Value: &value},
	}
	result, err = MarshalParametersWorkflow(params)
	assert.Nil(t, err)
	assert.Contains(t, result, "param1")
	assert.Contains(t, result, "value1")
}

func TestWorkflow_SetLabelsToAllTemplates_EmptyTemplates(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{Templates: nil},
	})
	// Should not panic
	workflow.SetLabelsToAllTemplates("key", "value")
}

func TestWorkflow_SetLabelsToAllTemplates_OverwriteExisting(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{
					Name: "template1",
					Metadata: workflowapi.Metadata{
						Labels: map[string]string{"key": "old-value"},
					},
				},
			},
		},
	})
	workflow.SetLabelsToAllTemplates("key", "new-value")
	assert.Equal(t, "new-value", workflow.Spec.Templates[0].Metadata.Labels["key"])
}

func TestWorkflow_CanRetry(t *testing.T) {
	// No offloaded node status → no error
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{},
	})
	assert.Nil(t, workflow.CanRetry())

	// Offloaded node status → error
	workflow = NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			OffloadNodeStatusVersion: "some-version",
		},
	})
	err := workflow.CanRetry()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Cannot retry workflow with offloaded node status")
}

func TestWorkflow_GenerateRetryExecution_FailedWorkflow(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-wf",
			Labels: map[string]string{
				"workflows.argoproj.io/completed":   "true",
				LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Spec: workflowapi.WorkflowSpec{
			Entrypoint: "main",
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Name:  "my-wf",
					Phase: workflowapi.NodeSucceeded,
					Type:  workflowapi.NodeTypePod,
				},
				"node-2": {
					ID:           "node-2",
					Name:         "my-wf.step2",
					Phase:        workflowapi.NodeFailed,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "step2",
					Children:     []string{},
				},
			},
		},
	})

	retryExec, podsToDelete, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)
	assert.Contains(t, podsToDelete, "my-wf-step2-2")

	retryWorkflow := retryExec.(*Workflow)
	assert.Equal(t, workflowapi.WorkflowRunning, retryWorkflow.Status.Phase)
	assert.Equal(t, "", retryWorkflow.Status.Message)

	// The succeeded node should be carried forward
	_, succeededExists := retryWorkflow.Status.Nodes["node-1"]
	assert.True(t, succeededExists)

	// The failed pod node should not be in the new nodes
	_, failedExists := retryWorkflow.Status.Nodes["node-2"]
	assert.False(t, failedExists)

	// Persisted final state label should be removed
	_, hasFinalStateLabel := retryWorkflow.Labels[LabelKeyWorkflowPersistedFinalState]
	assert.False(t, hasFinalStateLabel)
}

func TestWorkflow_GenerateRetryExecution_ErrorWorkflow(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-wf",
			Labels: map[string]string{},
		},
		Spec: workflowapi.WorkflowSpec{
			Entrypoint: "main",
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowError,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Name:  "my-wf",
					Phase: workflowapi.NodeSucceeded,
					Type:  workflowapi.NodeTypePod,
				},
			},
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)
}

func TestWorkflow_GenerateRetryExecution_RunningWorkflow(t *testing.T) {
	// Running workflows should not be retried
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.NotNil(t, err)
	assert.Nil(t, retryExec)
	assert.Contains(t, err.Error(), "Workflow must be Failed/Error to retry")
}

func TestWorkflow_GenerateRetryExecution_SucceededWorkflow(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowSucceeded,
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.NotNil(t, err)
	assert.Nil(t, retryExec)
}

func TestWorkflow_GenerateRetryExecution_WithDAGNode(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-wf",
			Labels: map[string]string{},
		},
		Spec: workflowapi.WorkflowSpec{
			Entrypoint: "main",
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"dag-node": {
					ID:       "dag-node",
					Name:     "my-wf.dag",
					Phase:    workflowapi.NodeFailed,
					Type:     workflowapi.NodeTypeDAG,
					Children: []string{"child-node"},
				},
				"child-node": {
					ID:           "child-node",
					Name:         "my-wf.dag.step1",
					Phase:        workflowapi.NodeFailed,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "step1",
				},
			},
		},
	})

	retryExec, podsToDelete, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)

	retryWorkflow := retryExec.(*Workflow)
	// DAG node should be reset to Running
	dagNode, exists := retryWorkflow.Status.Nodes["dag-node"]
	assert.True(t, exists)
	assert.Equal(t, workflowapi.NodeRunning, dagNode.Phase)
	assert.Equal(t, "", dagNode.Message)

	// Pod node should be deleted
	assert.Contains(t, podsToDelete, "my-wf-step1-node")
}

func TestWorkflow_GenerateRetryExecution_RunningNode(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Labels: map[string]string{}},
		Spec:       workflowapi.WorkflowSpec{Entrypoint: "main"},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Name:  "my-wf.step1",
					Phase: workflowapi.NodeRunning,
					Type:  workflowapi.NodeTypePod,
				},
			},
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.NotNil(t, err)
	assert.Nil(t, retryExec)
	assert.Contains(t, err.Error(), "Workflow cannot be retried with node")
}

func TestWorkflow_GenerateRetryExecution_SkippedNode(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Labels: map[string]string{}},
		Spec:       workflowapi.WorkflowSpec{Entrypoint: "main"},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Name:  "my-wf.step1",
					Phase: workflowapi.NodeSkipped,
					Type:  workflowapi.NodeTypePod,
				},
				"node-2": {
					ID:           "node-2",
					Name:         "my-wf.step2",
					Phase:        workflowapi.NodeFailed,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "step2",
				},
			},
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)

	retryWorkflow := retryExec.(*Workflow)
	// Skipped node should be carried forward
	_, exists := retryWorkflow.Status.Nodes["node-1"]
	assert.True(t, exists)
}

func TestWorkflow_GenerateRetryExecution_WithTerminatedDeadline(t *testing.T) {
	zero := int64(0)
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Labels: map[string]string{}},
		Spec: workflowapi.WorkflowSpec{
			Entrypoint:            "main",
			ActiveDeadlineSeconds: &zero,
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:           "node-1",
					Name:         "my-wf.step1",
					Phase:        workflowapi.NodeFailed,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "step1",
				},
			},
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)

	retryWorkflow := retryExec.(*Workflow)
	// ActiveDeadlineSeconds should be unset (was 0 indicating termination)
	assert.Nil(t, retryWorkflow.Spec.ActiveDeadlineSeconds)
}

func TestWorkflow_GenerateRetryExecution_OnExitNode(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Labels: map[string]string{}},
		Spec:       workflowapi.WorkflowSpec{Entrypoint: "main"},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
			Nodes: map[string]workflowapi.NodeStatus{
				"node-main": {
					ID:           "node-main",
					Name:         "my-wf.step1",
					Phase:        workflowapi.NodeFailed,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "step1",
				},
				"node-onexit": {
					ID:           "node-onexit",
					Name:         "my-wf.onExit.cleanup",
					Phase:        workflowapi.NodeSucceeded,
					Type:         workflowapi.NodeTypePod,
					TemplateName: "cleanup",
				},
			},
		},
	})

	retryExec, _, err := workflow.GenerateRetryExecution()
	assert.Nil(t, err)
	assert.NotNil(t, retryExec)

	retryWorkflow := retryExec.(*Workflow)
	// OnExit succeeded node should NOT be carried forward
	_, onExitExists := retryWorkflow.Status.Nodes["node-onexit"]
	assert.False(t, onExitExists)
}

func TestWorkflow_Validate(t *testing.T) {
	// Valid workflow
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Spec: workflowapi.WorkflowSpec{
			Entrypoint: "main",
			Templates: []workflowapi.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image:   "alpine",
						Command: []string{"echo", "hello"},
					},
				},
			},
		},
	})
	err := workflow.Validate(true, false)
	assert.Nil(t, err)

	// Invalid workflow (no entrypoint)
	workflow = NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Spec: workflowapi.WorkflowSpec{
			Templates: []workflowapi.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image: "alpine",
					},
				},
			},
		},
	})
	err = workflow.Validate(true, false)
	assert.NotNil(t, err)

	// ignoreEntrypoint = true → should succeed even without entrypoint
	err = workflow.Validate(true, true)
	assert.Nil(t, err)
}

func TestWorkflow_Decompress(t *testing.T) {
	// Normal workflow (not compressed) → no error
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
	})
	err := workflow.Decompress()
	assert.Nil(t, err)
}

func TestTransformJSONForBackwardCompatibility(t *testing.T) {
	// numberValue → number_value
	input := `{"metrics":[{"name":"accuracy","numberValue":0.95}]}`
	expected := `{"metrics":[{"name":"accuracy","number_value":0.95}]}`
	result, err := transformJSONForBackwardCompatibility(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, result)

	// Already snake_case → unchanged
	input = `{"metrics":[{"name":"accuracy","number_value":0.95}]}`
	result, err = transformJSONForBackwardCompatibility(input)
	assert.Nil(t, err)
	assert.Equal(t, input, result)

	// No metrics → unchanged
	input = `{"some":"value"}`
	result, err = transformJSONForBackwardCompatibility(input)
	assert.Nil(t, err)
	assert.Equal(t, input, result)
}

func TestReadNodeMetricsJSONOrEmpty(t *testing.T) {
	// No outputs → empty
	nodeStatus := &workflowapi.NodeStatus{
		ID:      "node-1",
		Outputs: nil,
	}
	wf := &workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}}
	result, err := readNodeMetricsJSONOrEmpty("run-1", nodeStatus, nil, wf)
	assert.Nil(t, err)
	assert.Empty(t, result)

	// No artifacts → empty
	nodeStatus = &workflowapi.NodeStatus{
		ID: "node-1",
		Outputs: &workflowapi.Outputs{
			Artifacts: nil,
		},
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, nil, wf)
	assert.Nil(t, err)
	assert.Empty(t, result)

	// No metrics artifact → empty
	nodeStatus = &workflowapi.NodeStatus{
		ID: "node-1",
		Outputs: &workflowapi.Outputs{
			Artifacts: []workflowapi.Artifact{
				{Name: "some-other-artifact"},
			},
		},
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, nil, wf)
	assert.Nil(t, err)
	assert.Empty(t, result)

	// readArtifact returns error → error
	nodeStatus = &workflowapi.NodeStatus{
		ID: "node-1",
		Outputs: &workflowapi.Outputs{
			Artifacts: []workflowapi.Artifact{
				{Name: "mlpipeline-metrics"},
			},
		},
	}
	mockReadArtifactError := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, fmt.Errorf("connection refused")
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifactError, wf)
	assert.NotNil(t, err)
	assert.Empty(t, result)

	// readArtifact returns nil response → empty
	mockReadArtifactNil := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, nil
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifactNil, wf)
	assert.Nil(t, err)
	assert.Empty(t, result)

	// readArtifact returns empty data → empty
	mockReadArtifactEmpty := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte{}}, nil
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifactEmpty, wf)
	assert.Nil(t, err)
	assert.Empty(t, result)

	// readArtifact returns invalid tgz → error
	mockReadArtifactBadTgz := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte("not a tgz file")}, nil
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifactBadTgz, wf)
	assert.NotNil(t, err)
	assert.Empty(t, result)

	// readArtifact returns valid tgz with metrics → success
	metricsJSON := `{"metrics":[{"name":"accuracy","number_value":0.95}]}`
	tgzContent, archiveErr := ArchiveTgz(map[string]string{"metrics.json": metricsJSON})
	assert.Nil(t, archiveErr)
	mockReadArtifactSuccess := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}
	result, err = readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifactSuccess, wf)
	assert.Nil(t, err)
	assert.Equal(t, metricsJSON, result)
}

func TestCollectNodeMetricsOrNil(t *testing.T) {
	wf := workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}}

	// Incomplete node → nil
	nodeStatus := &workflowapi.NodeStatus{
		ID:    "node-1",
		Phase: workflowapi.NodeRunning,
	}
	metrics, err := collectNodeMetricsOrNil("run-1", nodeStatus, nil, wf)
	assert.Nil(t, err)
	assert.Nil(t, metrics)

	// Completed node with no outputs → nil
	nodeStatus = &workflowapi.NodeStatus{
		ID:    "node-1",
		Phase: workflowapi.NodeSucceeded,
	}
	metrics, err = collectNodeMetricsOrNil("run-1", nodeStatus, nil, wf)
	assert.Nil(t, err)
	assert.Nil(t, metrics)
}

func TestCollectionMetrics_NoNodes(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{},
		},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, nil
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	assert.Empty(t, runMetrics)
	assert.Empty(t, partialFailures)
}

func TestCollectionMetrics_WithRunningNode(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Phase: workflowapi.NodeRunning,
				},
			},
		},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, nil
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	assert.Empty(t, runMetrics)
	assert.Empty(t, partialFailures)
}

func TestCollectionMetrics_WithCompletedNodeNoMetrics(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Phase: workflowapi.NodeSucceeded,
				},
			},
		},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, nil
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	assert.Empty(t, runMetrics)
	assert.Empty(t, partialFailures)
}

func TestCollectionMetrics_WithError(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Phase: workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{
							{Name: "mlpipeline-metrics"},
						},
					},
				},
			},
		},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return nil, fmt.Errorf("artifact read failed")
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	assert.Empty(t, runMetrics)
	assert.Len(t, partialFailures, 1)
}

func TestCollectionMetrics_WithValidMetrics(t *testing.T) {
	metricsJSON := `{"metrics":[{"name":"accuracy","number_value":0.95}]}`
	tgzContent, err := ArchiveTgz(map[string]string{"metrics.json": metricsJSON})
	assert.Nil(t, err)

	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:    "node-1",
					Phase: workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{
							{Name: "mlpipeline-metrics"},
						},
					},
				},
			},
		},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	assert.Len(t, runMetrics, 1)
	assert.Empty(t, partialFailures)
	assert.Equal(t, "accuracy", runMetrics[0].GetName())
	assert.Equal(t, "node-1", runMetrics[0].GetNodeId())
}

func TestCollectNodeMetricsOrNil_InvalidJSON(t *testing.T) {
	invalidMetricsJSON := `not valid json`
	tgzContent, err := ArchiveTgz(map[string]string{"metrics.json": invalidMetricsJSON})
	assert.Nil(t, err)

	wf := workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}}
	nodeStatus := &workflowapi.NodeStatus{
		ID:    "node-1",
		Phase: workflowapi.NodeSucceeded,
		Outputs: &workflowapi.Outputs{
			Artifacts: []workflowapi.Artifact{
				{Name: "mlpipeline-metrics"},
			},
		},
	}

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}

	metrics, metricsErr := collectNodeMetricsOrNil("run-1", nodeStatus, mockReadArtifact, wf)
	assert.NotNil(t, metricsErr)
	assert.Nil(t, metrics)
}

func TestCollectNodeMetricsOrNil_EmptyMetrics(t *testing.T) {
	emptyMetricsJSON := `{}`
	tgzContent, err := ArchiveTgz(map[string]string{"metrics.json": emptyMetricsJSON})
	assert.Nil(t, err)

	wf := workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}}
	nodeStatus := &workflowapi.NodeStatus{
		ID:    "node-1",
		Phase: workflowapi.NodeSucceeded,
		Outputs: &workflowapi.Outputs{
			Artifacts: []workflowapi.Artifact{
				{Name: "mlpipeline-metrics"},
			},
		},
	}

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}

	metrics, metricsErr := collectNodeMetricsOrNil("run-1", nodeStatus, mockReadArtifact, wf)
	assert.Nil(t, metricsErr)
	assert.Nil(t, metrics)
}

func TestCollectionMetrics_MetricsCountLimit(t *testing.T) {
	// Create a workflow with many nodes that produce metrics to hit the limit
	metricsJSON := `{"metrics":[{"name":"m1","number_value":0.1},{"name":"m2","number_value":0.2},{"name":"m3","number_value":0.3},{"name":"m4","number_value":0.4},{"name":"m5","number_value":0.5},{"name":"m6","number_value":0.6},{"name":"m7","number_value":0.7},{"name":"m8","number_value":0.8},{"name":"m9","number_value":0.9},{"name":"m10","number_value":1.0}]}`
	tgzContent, err := ArchiveTgz(map[string]string{"metrics.json": metricsJSON})
	assert.Nil(t, err)

	nodes := make(map[string]workflowapi.NodeStatus)
	// Create 6 nodes, each producing 10 metrics = 60 total, which exceeds maxMetricsCountLimit (50)
	for i := 0; i < 6; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodes[nodeID] = workflowapi.NodeStatus{
			ID:    nodeID,
			Phase: workflowapi.NodeSucceeded,
			Outputs: &workflowapi.Outputs{
				Artifacts: []workflowapi.Artifact{
					{Name: "mlpipeline-metrics"},
				},
			},
		}
	}

	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-wf",
			Labels: map[string]string{LabelKeyWorkflowRunId: "run-1"},
		},
		Status: workflowapi.WorkflowStatus{Nodes: nodes},
	})

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}
	runMetrics, partialFailures := workflow.CollectionMetrics(mockReadArtifact)
	// Should be capped at maxMetricsCountLimit (50)
	assert.LessOrEqual(t, len(runMetrics), 50)
	assert.Empty(t, partialFailures)
}

func TestReadNodeMetricsJSONOrEmpty_MultipleTgzFiles(t *testing.T) {
	// Multiple files in tgz → error
	tgzContent, err := ArchiveTgz(map[string]string{
		"file1.json": "content1",
		"file2.json": "content2",
	})
	assert.Nil(t, err)

	nodeStatus := &workflowapi.NodeStatus{
		ID: "node-1",
		Outputs: &workflowapi.Outputs{
			Artifacts: []workflowapi.Artifact{
				{Name: "mlpipeline-metrics"},
			},
		},
	}
	wf := &workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}}

	mockReadArtifact := func(request *artifactclient.ReadArtifactRequest) (*artifactclient.ReadArtifactResponse, error) {
		return &artifactclient.ReadArtifactResponse{Data: []byte(tgzContent)}, nil
	}
	result, err := readNodeMetricsJSONOrEmpty("run-1", nodeStatus, mockReadArtifact, wf)
	assert.NotNil(t, err)
	assert.Empty(t, result)
}

func TestWorkflow_SetExecutionName(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "gen-prefix-",
			Name:         "old-name",
		},
	})
	workflow.SetExecutionName("new-name")
	assert.Equal(t, "new-name", workflow.Name)
	assert.Equal(t, "", workflow.GenerateName)
}

func TestWorkflow_Condition(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowSucceeded,
		},
	})
	assert.Equal(t, "Succeeded", string(workflow.Condition()))
}

func TestWorkflow_HasScheduledWorkflowAsParent(t *testing.T) {
	t.Run("no owner references returns false", func(t *testing.T) {
		workflow := NewWorkflow(&workflowapi.Workflow{})
		assert.False(t, workflow.HasScheduledWorkflowAsParent())
	})

	t.Run("with ScheduledWorkflow owner returns true", func(t *testing.T) {
		workflow := NewWorkflow(&workflowapi.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "kubeflow.org/v1beta1",
						Kind:       "ScheduledWorkflow",
						Name:       "my-schedule",
						UID:        types.UID("sched-uid"),
					},
				},
			},
		})
		assert.True(t, workflow.HasScheduledWorkflowAsParent())
	})
}

func TestWorkflow_FindObjectStoreArtifactKeyOrEmpty_NoS3(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node1": {
					ID: "node1",
					Outputs: &workflowapi.Outputs{
						Artifacts: workflowapi.Artifacts{
							{Name: "artifact1"},
						},
					},
				},
			},
		},
	})
	assert.Equal(t, "", workflow.FindObjectStoreArtifactKeyOrEmpty("node1", "artifact1"))
}

func TestWorkflow_FindObjectStoreArtifactKeyOrEmpty_WithS3(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node1": {
					ID: "node1",
					Outputs: &workflowapi.Outputs{
						Artifacts: workflowapi.Artifacts{
							{
								Name: "artifact1",
								ArtifactLocation: workflowapi.ArtifactLocation{
									S3: &workflowapi.S3Artifact{
										Key: "my-bucket/key/path",
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Equal(t, "my-bucket/key/path", workflow.FindObjectStoreArtifactKeyOrEmpty("node1", "artifact1"))
}

func TestWorkflow_FindObjectStoreArtifactKeyOrEmpty_NodeNotFound(t *testing.T) {
	workflow := NewWorkflow(&workflowapi.Workflow{
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"other-node": {ID: "other-node"},
			},
		},
	})
	assert.Equal(t, "", workflow.FindObjectStoreArtifactKeyOrEmpty("node1", "artifact1"))
}

// nonWorkflowExecution implements ExecutionSpec via embedding but is not *Workflow.
// Used to test type assertion failures in WorkflowInterface methods.
type nonWorkflowExecution struct {
	ExecutionSpec
}

func TestWorkflowInterface_Create(t *testing.T) {
	fakeClient := argofake.NewSimpleClientset()
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	t.Run("success", func(t *testing.T) {
		workflow := NewWorkflow(&workflowapi.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
			Spec:       workflowapi.WorkflowSpec{Entrypoint: "main"},
		})
		result, err := wfi.Create(context.Background(), workflow, metav1.CreateOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-wf", result.ExecutionName())
	})

	t.Run("wrong type returns error", func(t *testing.T) {
		result, err := wfi.Create(context.Background(), &nonWorkflowExecution{}, metav1.CreateOptions{})
		assert.NotNil(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not a valid ExecutionSpec")
	})
}

func TestWorkflowInterface_Update(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
		Spec:       workflowapi.WorkflowSpec{Entrypoint: "main"},
	}
	fakeClient := argofake.NewSimpleClientset(existingWorkflow)
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	t.Run("success", func(t *testing.T) {
		updated := NewWorkflow(&workflowapi.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
			Spec:       workflowapi.WorkflowSpec{Entrypoint: "updated-main"},
		})
		result, err := wfi.Update(context.Background(), updated, metav1.UpdateOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("wrong type returns error", func(t *testing.T) {
		result, err := wfi.Update(context.Background(), &nonWorkflowExecution{}, metav1.UpdateOptions{})
		assert.NotNil(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not a valid ExecutionSpec")
	})
}

func TestWorkflowInterface_Delete(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
	}
	fakeClient := argofake.NewSimpleClientset(existingWorkflow)
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	err := wfi.Delete(context.Background(), "test-wf", metav1.DeleteOptions{})
	assert.Nil(t, err)
}

func TestWorkflowInterface_DeleteCollection(t *testing.T) {
	fakeClient := argofake.NewSimpleClientset()
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	err := wfi.DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{})
	assert.Nil(t, err)
}

func TestWorkflowInterface_Get(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
	}
	fakeClient := argofake.NewSimpleClientset(existingWorkflow)
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	t.Run("success", func(t *testing.T) {
		result, err := wfi.Get(context.Background(), "test-wf", metav1.GetOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-wf", result.ExecutionName())
	})

	t.Run("not found", func(t *testing.T) {
		result, err := wfi.Get(context.Background(), "nonexistent", metav1.GetOptions{})
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkflowInterface_List(t *testing.T) {
	workflow1 := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", Namespace: "default"},
	}
	workflow2 := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-2", Namespace: "default"},
	}
	fakeClient := argofake.NewSimpleClientset(workflow1, workflow2)
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	result, err := wfi.List(context.Background(), metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Len(t, *result, 2)
}

func TestWorkflowInterface_Patch(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
	}
	fakeClient := argofake.NewSimpleClientset(existingWorkflow)
	wfi := &WorkflowInterface{
		workflowInterface: fakeClient.ArgoprojV1alpha1().Workflows("default"),
	}

	patchData := []byte(`{"metadata":{"labels":{"new-label":"value"}}}`)
	result, err := wfi.Patch(context.Background(), "test-wf", types.MergePatchType, patchData, metav1.PatchOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func TestWorkflowInformer_InformerFactoryStart(t *testing.T) {
	fakeClient := argofake.NewSimpleClientset()
	factory := argoinformer.NewSharedInformerFactory(fakeClient, 0)
	workflowInformer := factory.Argoproj().V1alpha1().Workflows()

	wfi := &WorkflowInformer{
		informer: workflowInformer,
		factory:  factory,
	}

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	// Should not panic
	wfi.InformerFactoryStart(stopChannel)
}

func TestWorkflowInformer_HasSynced(t *testing.T) {
	fakeClient := argofake.NewSimpleClientset()
	factory := argoinformer.NewSharedInformerFactory(fakeClient, 0)
	workflowInformer := factory.Argoproj().V1alpha1().Workflows()

	wfi := &WorkflowInformer{
		informer: workflowInformer,
		factory:  factory,
	}

	hasSyncedFunc := wfi.HasSynced()
	assert.NotNil(t, hasSyncedFunc)
}

func TestWorkflowInformer_AddEventHandler(t *testing.T) {
	fakeClient := argofake.NewSimpleClientset()
	factory := argoinformer.NewSharedInformerFactory(fakeClient, 0)
	workflowInformer := factory.Argoproj().V1alpha1().Workflows()

	wfi := &WorkflowInformer{
		informer: workflowInformer,
		factory:  factory,
	}

	registration, err := wfi.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {},
		UpdateFunc: func(oldObj, newObj interface{}) {},
		DeleteFunc: func(obj interface{}) {},
	})
	assert.Nil(t, err)
	assert.NotNil(t, registration)
}

// mockWorkflowInformerWithLister implements the WorkflowInformer interface
// using a pre-populated indexer, avoiding the need for informer cache sync.
type mockWorkflowInformerWithLister struct {
	workflowLister argolister.WorkflowLister
}

func (m *mockWorkflowInformerWithLister) Informer() cache.SharedIndexInformer {
	return nil
}

func (m *mockWorkflowInformerWithLister) Lister() argolister.WorkflowLister {
	return m.workflowLister
}

func TestWorkflowInformer_Get(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
	}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	indexer.Add(existingWorkflow)
	lister := argolister.NewWorkflowLister(indexer)

	wfi := &WorkflowInformer{
		informer: &mockWorkflowInformerWithLister{workflowLister: lister},
	}

	t.Run("success", func(t *testing.T) {
		result, isNotFound, err := wfi.Get("default", "test-wf")
		assert.Nil(t, err)
		assert.False(t, isNotFound)
		assert.NotNil(t, result)
		assert.Equal(t, "test-wf", result.ExecutionName())
	})

	t.Run("not found", func(t *testing.T) {
		result, isNotFound, err := wfi.Get("default", "nonexistent")
		assert.NotNil(t, err)
		assert.True(t, isNotFound)
		assert.Nil(t, result)
	})
}

func TestWorkflowInformer_List(t *testing.T) {
	existingWorkflow := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf", Namespace: "default"},
	}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	indexer.Add(existingWorkflow)
	lister := argolister.NewWorkflowLister(indexer)

	wfi := &WorkflowInformer{
		informer: &mockWorkflowInformerWithLister{workflowLister: lister},
	}

	selector := labels.Everything()
	result, err := wfi.List(&selector)
	assert.Nil(t, err)
	assert.Len(t, result, 1)
}
