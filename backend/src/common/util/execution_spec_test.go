// Copyright 2022 The Kubeflow Authors
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

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestExecutionSpec_NewExecutionSpec(t *testing.T) {
	execSpec, err := NewExecutionSpec([]byte{})
	assert.Nil(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, NewInvalidInputError("empty input").Error())

	execSpec, err = NewExecutionSpec([]byte("invalid format"))
	assert.Nil(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, "InvalidInputError: Failed to unmarshal the inputs: "+
		"error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string "+
		"into Go value of type v1.TypeMeta")

	// Normal case
	bytes, err := yaml.Marshal(workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
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
	assert.Nil(t, err)
	assert.NotEmpty(t, bytes)
	execSpec, err = NewExecutionSpec(bytes)
	assert.Nil(t, err)
	assert.NotEmpty(t, execSpec)
}

func TestExecutionSpec_NewExecutionSpecJSON(t *testing.T) {
	execSpec, err := NewExecutionSpecJSON(ArgoWorkflow, []byte{})
	assert.Nil(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, NewInvalidInputError("empty input").Error())

	execSpec, err = NewExecutionSpecJSON(ArgoWorkflow, []byte("invalid format"))
	assert.Nil(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, "InvalidInputError: Failed to unmarshal the inputs: "+
		"invalid character 'i' looking for beginning of value")

	// Normal case
	bytes, err := json.Marshal(workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
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
	assert.Nil(t, err)
	assert.NotEmpty(t, bytes)
	execSpec, err = NewExecutionSpecJSON(ArgoWorkflow, bytes)
	assert.Nil(t, err)
	assert.NotEmpty(t, execSpec)
}

func TestExecutionSpec_NewExecutionSpecFromInterface(t *testing.T) {
	test := &workflowapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
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
	}
	execSpec, err := NewExecutionSpecFromInterface(ArgoWorkflow, test)
	assert.Empty(t, err)
	assert.NotEmpty(t, execSpec)

	// unknown type
	execSpec, err = NewExecutionSpecFromInterface("Unimplemented", test)
	assert.Empty(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, "InternalServerError: type:Unimplemented: ExecutionType is not supported")
}

func TestExecutionSpec_UnmarshalParameters(t *testing.T) {
	orgParams := []workflowapi.Parameter{
		{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
		{Name: "PARAM2", Value: workflowapi.AnyStringPtr("")},
	}

	expectedParams := SpecParameters{
		SpecParameter{Name: "PARAM1", Value: stringToPointer("VALUE1")},
		SpecParameter{Name: "PARAM2", Value: stringToPointer("")},
	}

	paramStr, err := json.Marshal(orgParams)
	assert.Nil(t, err)

	specParams, err := UnmarshalParameters(ArgoWorkflow, string(paramStr))
	assert.Nil(t, err)
	assert.Equal(t, specParams, expectedParams)
}

func TestExecutionSpec_MarshalParameters(t *testing.T) {
	expectedParams := []workflowapi.Parameter{
		{Name: "PARAM1", Value: workflowapi.AnyStringPtr("VALUE1")},
		{Name: "PARAM2", Value: workflowapi.AnyStringPtr("")},
	}

	params := SpecParameters{
		SpecParameter{Name: "PARAM1", Value: stringToPointer("VALUE1")},
		SpecParameter{Name: "PARAM2", Value: stringToPointer("")},
	}

	expectedStr, err := json.Marshal(expectedParams)
	assert.Nil(t, err)

	paramStr, err := MarshalParameters(ArgoWorkflow, params)
	assert.Nil(t, err)
	assert.Equal(t, paramStr, string(expectedStr))
}

func TestCurrentExecutionType_And_SetExecutionType(t *testing.T) {
	// Save original and restore after test
	original := CurrentExecutionType()
	defer SetExecutionType(original)

	SetExecutionType(ArgoWorkflow)
	assert.Equal(t, ArgoWorkflow, CurrentExecutionType())

	SetExecutionType(Unknown)
	assert.Equal(t, Unknown, CurrentExecutionType())
}

func TestGetTerminatePatch(t *testing.T) {
	testCases := []struct {
		name     string
		execType ExecutionType
		expected interface{}
	}{
		{
			name:     "ArgoWorkflow returns activeDeadlineSeconds patch",
			execType: ArgoWorkflow,
			expected: map[string]interface{}{
				"spec": map[string]interface{}{
					"activeDeadlineSeconds": 0,
				},
			},
		},
		{
			name:     "Unknown type returns nil",
			execType: Unknown,
			expected: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := GetTerminatePatch(testCase.execType)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestNewExecutionSpec_UnknownKind(t *testing.T) {
	yamlBytes := []byte("apiVersion: v1\nkind: UnknownKind\n")
	result, err := NewExecutionSpec(yamlBytes)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown execution spec")
}

func TestNewExecutionSpecJSON_UnknownType(t *testing.T) {
	result, err := NewExecutionSpecJSON(Unknown, []byte(`{"metadata":{"name":"test"}}`))
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown execution spec")
}

func TestUnmarshalParameters_UnknownType(t *testing.T) {
	result, err := UnmarshalParameters(Unknown, "[]")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ExecutionType is not supported")
}

func TestMarshalParameters_UnknownType(t *testing.T) {
	result, err := MarshalParameters(Unknown, nil)
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ExecutionType is not supported")
}

func TestMarshalParameters_NilParams(t *testing.T) {
	result, err := MarshalParameters(ArgoWorkflow, nil)
	assert.Nil(t, err)
	assert.Equal(t, "[]", result)
}

func TestScheduleSpecToExecutionSpec_ArgoWorkflow_StringSpec(t *testing.T) {
	specJSON := `{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}`
	wfr := &swfapi.WorkflowResource{
		Spec: specJSON,
	}
	result, err := ScheduleSpecToExecutionSpec(ArgoWorkflow, wfr)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "argoproj.io/v1alpha1", result.ExecutionTypeMeta().APIVersion)
	assert.Equal(t, "Workflow", result.ExecutionTypeMeta().Kind)
}

func TestScheduleSpecToExecutionSpec_ArgoWorkflow_MapSpec(t *testing.T) {
	// When Spec is a map (not a string), it should be marshaled back to JSON
	wfr := &swfapi.WorkflowResource{
		Spec: map[string]interface{}{
			"entrypoint": "main",
			"templates": []interface{}{
				map[string]interface{}{
					"name": "main",
					"container": map[string]interface{}{
						"image": "alpine",
					},
				},
			},
		},
	}
	result, err := ScheduleSpecToExecutionSpec(ArgoWorkflow, wfr)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func TestScheduleSpecToExecutionSpec_UnknownType(t *testing.T) {
	wfr := &swfapi.WorkflowResource{Spec: "{}"}
	result, err := ScheduleSpecToExecutionSpec(Unknown, wfr)
	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ExecutionType is not supported")
}

func TestScheduleSpecToExecutionSpec_ArgoWorkflow_FullWorkflowJSON(t *testing.T) {
	// Full workflow JSON with entrypoint at top level
	fullJSON := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"test"},"spec":{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}}`
	wfr := &swfapi.WorkflowResource{
		Spec: fullJSON,
	}
	result, err := ScheduleSpecToExecutionSpec(ArgoWorkflow, wfr)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func TestScheduleSpecToExecutionSpec_ArgoWorkflow_InvalidMapSpec(t *testing.T) {
	// Spec is a map that marshals fine but contains invalid WorkflowSpec data
	// This tests the unmarshal error path when the map has invalid types
	wfr := &swfapi.WorkflowResource{
		Spec: map[string]interface{}{
			"entrypoint": 12345,          // Invalid: should be string, but json.Unmarshal tolerates this
			"templates":  "not-an-array", // Invalid: should be array
		},
	}
	result, err := ScheduleSpecToExecutionSpec(ArgoWorkflow, wfr)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestNewExecutionSpecFromInterface_UnknownType(t *testing.T) {
	result, err := NewExecutionSpecFromInterface(Unknown, nil)
	assert.Nil(t, result)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "ExecutionType is not supported")
}
