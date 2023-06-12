// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package component

import (
	"context"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/blob"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

var addNumbersComponent = &pipelinespec.ComponentSpec{
	Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "add"},
	InputDefinitions: &pipelinespec.ComponentInputsSpec{
		Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
			"a": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER, DefaultValue: structpb.NewNumberValue(5)},
			"b": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
	OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
		Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
			"Output": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
}

// Tests that launcher correctly executes the user component and successfully writes output parameters to file.
func Test_executeV2_Parameters(t *testing.T) {
	defer func() {
		os.RemoveAll("/tmp")
	}()

	tests := []struct {
		name                string
		executorInput       *pipelinespec.ExecutorInput
		executorArgs        []string
		wantErr             bool
		expectedOutputParam float64
	}{
		{
			"happy pass",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
				Outputs: &pipelinespec.ExecutorInput_Outputs{
					Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
						"Output": {OutputFile: "/tmp/kfp/outputs/Output"},
					},
					OutputFile: "/tmp/kfp_outputs/output_metadata.json",
				},
			},
			[]string{"-c", "apt-get update\napt-get install python3.7-venv\npython3 -m venv .venv && source .venv/bin/activate\nexport CLUSTER_SPEC=\"{\\\"task\\\":{\\\"type\\\":\\\"workerpool0\\\",\\\"index\\\":0,\\\"trial\\\":\\\"TRIAL_ID\\\"}}\"\nif ! [ -x \"$(command -v pip)\" ]; then\n    apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location 'kfp==2.0.0-rc.1' && \"$0\" \"$@\"\nprogram_path=$(mktemp -d)\nprintf \"%s\" \"$0\" > \"$program_path/ephemeral_component.py\"\npython3 -m kfp.components.executor_main                         --component_module_path                         \"$program_path/ephemeral_component.py\"                         \"$@\"\n", "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef add_numbers(a: int, b: int) -> int:\n    return a + b\n\n", "--executor_input", "{{$}}", "--function_to_execute", "add_numbers"},
			false,
			3,
		},
		{
			"use default value",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"b": structpb.NewNumberValue(2)},
				},
				Outputs: &pipelinespec.ExecutorInput_Outputs{
					Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
						"Output": {OutputFile: "/tmp/kfp/outputs/Output"},
					},
					OutputFile: "/tmp/kfp_outputs/output_metadata.json",
				},
			},
			[]string{"-c", "apt-get update\napt-get install python3.7-venv\npython3 -m venv .venv && source .venv/bin/activate\nexport CLUSTER_SPEC=\"{\\\"task\\\":{\\\"type\\\":\\\"workerpool0\\\",\\\"index\\\":0,\\\"trial\\\":\\\"TRIAL_ID\\\"}}\"\nif ! [ -x \"$(command -v pip)\" ]; then\n    apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location 'kfp==2.0.0-rc.1' && \"$0\" \"$@\"\nprogram_path=$(mktemp -d)\nprintf \"%s\" \"$0\" > \"$program_path/ephemeral_component.py\"\npython3 -m kfp.components.executor_main                         --component_module_path                         \"$program_path/ephemeral_component.py\"                         \"$@\"\n", "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef add_numbers(a: int, b: int) -> int:\n    return a + b\n\n", "--executor_input", "{{$}}", "--function_to_execute", "add_numbers"},
			false,
			7,
		},
		{
			"missing parameter",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{},
				},
				Outputs: &pipelinespec.ExecutorInput_Outputs{
					Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
						"Output": {OutputFile: "/tmp/kfp/outputs/Output"},
					},
					OutputFile: "/tmp/kfp_outputs/output_metadata.json",
				},
			},
			[]string{"-c", "apt-get update\napt-get install python3.7-venv\npython3 -m venv .venv && source .venv/bin/activate\nexport CLUSTER_SPEC=\"{\\\"task\\\":{\\\"type\\\":\\\"workerpool0\\\",\\\"index\\\":0,\\\"trial\\\":\\\"TRIAL_ID\\\"}}\"\nif ! [ -x \"$(command -v pip)\" ]; then\n    apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location 'kfp==2.0.0-rc.1' && \"$0\" \"$@\"\nprogram_path=$(mktemp -d)\nprintf \"%s\" \"$0\" > \"$program_path/ephemeral_component.py\"\npython3 -m kfp.components.executor_main                         --component_module_path                         \"$program_path/ephemeral_component.py\"                         \"$@\"\n", "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef add_numbers(a: int, b: int) -> int:\n    return a + b\n\n", "--executor_input", "{{$}}", "--function_to_execute", "add_numbers"},
			true,
			0,
		},
		{
			"param placeholders",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "[[ {{$.inputs.parameters['a']}} + {{$.inputs.parameters['b']}} == 7 ]] || exit 1"},
			false,
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeKubernetesClientset := &fake.Clientset{}
			fakeMetadataClient := metadata.NewFakeClient()
			bucket, err := blob.OpenBucket(nil, "gs://ml-pipeline-test")
			assert.Nil(t, err)
			bucketConfig, err := objectstore.ParseBucketConfig("gs://ml-pipeline-test/pipeline-root/")
			assert.Nil(t, err)
			executorOutput, _, err := executeV2(context.Background(), test.executorInput, addNumbersComponent, "sh", test.executorArgs, bucket, bucketConfig, fakeMetadataClient, "namespace", fakeKubernetesClientset)

			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, executorOutput)
				assert.Equal(t, test.expectedOutputParam, executorOutput.GetParameterValues()["Output"].GetNumberValue())
			}
		})
	}
}
