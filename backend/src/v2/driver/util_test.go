// Copyright 2021-2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func Test_isInputParameterChannel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "wellformed pipeline channel should produce no errors",
			input:   "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			isValid: true,
		},
		{
			name:    "pipeline channel index should have quotes",
			input:   "{{$.inputs.parameters[pipelinechannel--someParameterName]}}",
			isValid: false,
		},
		{
			name:    "plain text as pipelinechannel of parameter type is invalid",
			input:   "randomtext",
			isValid: false,
		},
		{
			name:    "inputs should be prefixed with $.",
			input:   "{{inputs.parameters['pipelinechannel--someParameterName']}}",
			isValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, isInputParameterChannel(test.input), test.isValid)
		})
	}
}

func Test_isAlreadyExistsErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "grpc already exists",
			err:  status.Error(codes.AlreadyExists, "duplicate execution"),
			want: true,
		},
		{
			name: "wrapped grpc already exists",
			err:  fmt.Errorf("wrapped: %w", status.Error(codes.AlreadyExists, "duplicate execution")),
			want: true,
		},
		{
			name: "already exists in message",
			err:  fmt.Errorf("rpc error: code = Internal desc = AlreadyExists: execution exists"),
			want: true,
		},
		{
			name: "duplicate entry in message",
			err:  fmt.Errorf("sql failure: Duplicate entry 'run/abc'"),
			want: true,
		},
		{
			name: "other error",
			err:  status.Error(codes.Internal, "some other failure"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isAlreadyExistsErr(test.err))
		})
	}
}

func Test_extractInputParameterFromChannel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard parameter pipeline channel input",
			input:    "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			expected: "pipelinechannel--someParameterName",
			wantErr:  false,
		},
		{
			name:     "a more complex parameter pipeline channel input",
			input:    "{{$.inputs.parameters['pipelinechannel--somePara-me_terName']}}",
			expected: "pipelinechannel--somePara-me_terName",
			wantErr:  false,
		},
		{
			name:    "invalid input should return err",
			input:   "invalidvalue",
			wantErr: true,
		},
		{
			name:    "invalid input should return err 2",
			input:   "pipelinechannel--somePara-me_terName",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := extractInputParameterFromChannel(test.input)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, actual, test.expected)
			}
		})
	}
}

func Test_inputParamConstant(t *testing.T) {
	tests := []struct {
		name       string
		inputValue string
	}{
		{
			name:       "simple string value",
			inputValue: "hello",
		},
		{
			name:       "empty string value",
			inputValue: "",
		},
		{
			name:       "numeric string value",
			inputValue: "42",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paramSpec := inputParamConstant(test.inputValue)
			assert.NotNil(t, paramSpec)
			runtimeValue := paramSpec.GetRuntimeValue()
			assert.NotNil(t, runtimeValue)
			assert.Equal(t, test.inputValue, runtimeValue.GetConstant().GetStringValue())
		})
	}
}

func Test_inputParamComponent(t *testing.T) {
	tests := []struct {
		name       string
		inputValue string
	}{
		{
			name:       "standard component input parameter",
			inputValue: "my_param",
		},
		{
			name:       "empty string",
			inputValue: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paramSpec := inputParamComponent(test.inputValue)
			assert.NotNil(t, paramSpec)
			assert.Equal(t, test.inputValue, paramSpec.GetComponentInputParameter())
		})
	}
}

func Test_inputParamTaskOutput(t *testing.T) {
	tests := []struct {
		name         string
		producerTask string
		outputParam  string
	}{
		{
			name:         "standard task output parameter",
			producerTask: "task-1",
			outputParam:  "output_key",
		},
		{
			name:         "empty producer task",
			producerTask: "",
			outputParam:  "output_key",
		},
		{
			name:         "empty output param",
			producerTask: "task-1",
			outputParam:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paramSpec := inputParamTaskOutput(test.producerTask, test.outputParam)
			assert.NotNil(t, paramSpec)
			taskOutputSpec := paramSpec.GetTaskOutputParameter()
			assert.NotNil(t, taskOutputSpec)
			assert.Equal(t, test.producerTask, taskOutputSpec.GetProducerTask())
			assert.Equal(t, test.outputParam, taskOutputSpec.GetOutputParameterKey())
		})
	}
}

func Test_getItems(t *testing.T) {
	tests := []struct {
		name      string
		value     *structpb.Value
		wantCount int
		wantErr   bool
	}{
		{
			name: "list value returns items directly",
			value: structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("a"),
					structpb.NewStringValue("b"),
					structpb.NewStringValue("c"),
				},
			}),
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "string value with valid JSON array",
			value:     structpb.NewStringValue(`["x","y"]`),
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "string value with empty JSON array",
			value:     structpb.NewStringValue(`[]`),
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:    "string value with invalid JSON",
			value:   structpb.NewStringValue(`not-json`),
			wantErr: true,
		},
		{
			name:    "number value returns error",
			value:   structpb.NewNumberValue(42),
			wantErr: true,
		},
		{
			name:    "bool value returns error",
			value:   structpb.NewBoolValue(true),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, err := getItems(test.value)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.wantCount, len(items))
			}
		})
	}
}
func Test_resolvePodSpecRuntimeParameter(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		executorInput *pipelinespec.ExecutorInput
		wantErr       bool
	}{
		{
			name:     "should retrieve correct parameter value",
			input:    "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			expected: "test2",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"pipelinechannel--":                  structpb.NewStringValue("test1"),
						"pipelinechannel--someParameterName": structpb.NewStringValue("test2"),
						"someParameterName":                  structpb.NewStringValue("test3"),
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "return err when no match is found",
			input:    "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			expected: "test1",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"doesNotMatch": structpb.NewStringValue("test2"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:          "return const val when input is not a pipeline channel",
			input:         "not-pipeline-channel",
			expected:      "not-pipeline-channel",
			executorInput: &pipelinespec.ExecutorInput{},
			wantErr:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := resolver.ResolveParameterOrPipelineChannel(test.input, test.executorInput)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, actual, test.expected)
			}
		})
	}
}
