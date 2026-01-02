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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
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
			actual, err := resolvePodSpecInputRuntimeParameter(test.input, test.executorInput)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, actual, test.expected)
			}
		})
	}
}

func Test_resolveContainerArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		executorInput *pipelinespec.ExecutorInput
		expected      []string
		wantErr       bool
	}{
		{
			name: "IfPresent with parameter present",
			args: []string{
				"{{$.inputs.parameters['file']}}",
				`{"IfPresent": {"InputName": "line_number", "Then": ["-n"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file":        structpb.NewStringValue("/etc/hosts"),
						"line_number": structpb.NewBoolValue(true),
					},
				},
			},
			expected: []string{"/etc/hosts", "-n"},
			wantErr:  false,
		},
		{
			name: "IfPresent with parameter absent",
			args: []string{
				"{{$.inputs.parameters['file']}}",
				`{"IfPresent": {"InputName": "line_number", "Then": ["-n"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file": structpb.NewStringValue("/etc/hosts"),
					},
				},
			},
			expected: []string{"/etc/hosts"},
			wantErr:  false,
		},
		{
			name: "IfPresent with else clause",
			args: []string{
				"{{$.inputs.parameters['file']}}",
				`{"IfPresent": {"InputName": "line_number", "Then": ["-n"], "Else": ["--no-line-numbers"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file": structpb.NewStringValue("/etc/hosts"),
					},
				},
			},
			expected: []string{"/etc/hosts", "--no-line-numbers"},
			wantErr:  false,
		},
		{
			name: "IfPresent with multiple values",
			args: []string{
				`{"IfPresent": {"InputName": "verbose", "Then": ["--verbose", "--debug"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"verbose": structpb.NewBoolValue(true),
					},
				},
			},
			expected: []string{"--verbose", "--debug"},
			wantErr:  false,
		},
		{
			name: "input parameter channel",
			args: []string{
				"{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"pipelinechannel--someParameterName": structpb.NewStringValue("resolved-value"),
					},
				},
			},
			expected: []string{"resolved-value"},
			wantErr:  false,
		},
		{
			name: "regular arg",
			args: []string{
				"regular-arg",
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{},
				},
			},
			expected: []string{"regular-arg"},
			wantErr:  false,
		},
		{
			name: "IfPresent with non-string in array",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag", 123]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "IfPresent with single string value",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": "--flag"}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: []string{"--flag"},
			wantErr:  false,
		},
		// Error scenario tests
		{
			name: "malformed IfPresent JSON - invalid syntax",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag"]`, // missing closing brace
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "malformed IfPresent JSON - invalid structure",
			args: []string{
				`{"IfPresent": "invalid"}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid InputName reference - missing parameter in Then clause",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["{{$.inputs.parameters['missing-param']}}"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid InputName reference - missing parameter in Else clause",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Else": ["{{$.inputs.parameters['missing-param']}}"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "arrays containing non-string values - object in array",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag", {"key": "value"}]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "arrays containing non-string values - array in array",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag", ["nested", "array"]]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "arrays containing non-string values - boolean in array",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag", true]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "arrays containing non-string values - null in array",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["--flag", null]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "nested parameter resolution failure - invalid parameter channel format",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["{{$.inputs.parameters['invalid-format']}}"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "nested parameter resolution failure - parameter channel not in executorInput",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": ["{{$.inputs.parameters['pipelinechannel--nonexistent']}}"]}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "unexpected type in IfPresent Then/Else - number",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": 123}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "unexpected type in IfPresent Then/Else - object",
			args: []string{
				`{"IfPresent": {"InputName": "flag", "Then": {"key": "value"}}}`,
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"flag": structpb.NewBoolValue(true),
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := resolveContainerArgs(test.args, test.executorInput)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}
