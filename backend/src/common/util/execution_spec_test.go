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
	// TODO: fix this when PipelineRun get implemented
	execSpec, err = NewExecutionSpecFromInterface(TektonPipelineRun, test)
	assert.Empty(t, execSpec)
	assert.Error(t, err)
	assert.EqualError(t, err, "Invalid input error: not PipelineRun struct")
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
