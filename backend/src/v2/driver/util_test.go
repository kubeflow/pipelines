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
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/stretchr/testify/assert"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

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
