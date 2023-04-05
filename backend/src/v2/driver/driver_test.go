// Copyright 2023 The Kubeflow Authors
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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_makePodSpecPatch_acceleratorConfig(t *testing.T) {
	viper.Set("KFP_POD_NAME", "MyWorkflowPod")
	viper.Set("KFP_POD_UID", "a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6")
	type args struct {
		container     *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
		componentSpec *pipelinespec.ComponentSpec
		executorInput *pipelinespec.ExecutorInput
		executionID   int64
		pipelineName  string
		runID         string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			"Valid - nvidia.com/gpu",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:    1.0,
						MemoryLimit: 0.65,
						Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
							Type:  "nvidia.com/gpu",
							Count: 1,
						},
					},
				},
				&pipelinespec.ComponentSpec{
					Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "addition"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{
						Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
							"a": {Type: pipelinespec.PrimitiveType_DOUBLE},
							"b": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"Output": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
				},
				nil,
				1,
				"MyPipeline",
				"a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6",
			},
			`"nvidia.com/gpu":"1"`,
			false,
			"",
		},
		{
			"Valid - amd.com/gpu",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:    1.0,
						MemoryLimit: 0.65,
						Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
							Type:  "amd.com/gpu",
							Count: 1,
						},
					},
				},
				&pipelinespec.ComponentSpec{
					Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "addition"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{
						Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
							"a": {Type: pipelinespec.PrimitiveType_DOUBLE},
							"b": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"Output": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
				},
				nil,
				1,
				"MyPipeline",
				"a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6",
			},
			`"amd.com/gpu":"1"`,
			false,
			"",
		},
		{
			"Valid - cloud-tpus.google.com/v3",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:    1.0,
						MemoryLimit: 0.65,
						Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
							Type:  "cloud-tpus.google.com/v3",
							Count: 1,
						},
					},
				},
				&pipelinespec.ComponentSpec{
					Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "addition"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{
						Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
							"a": {Type: pipelinespec.PrimitiveType_DOUBLE},
							"b": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"Output": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
				},
				nil,
				1,
				"MyPipeline",
				"a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6",
			},
			`"cloud-tpus.google.com/v3":"1"`,
			false,
			"",
		},
		{
			"Valid - cloud-tpus.google.com/v2",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:    1.0,
						MemoryLimit: 0.65,
						Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
							Type:  "cloud-tpus.google.com/v2",
							Count: 1,
						},
					},
				},
				&pipelinespec.ComponentSpec{
					Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "addition"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{
						Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
							"a": {Type: pipelinespec.PrimitiveType_DOUBLE},
							"b": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"Output": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
				},
				nil,
				1,
				"MyPipeline",
				"a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6",
			},
			`"cloud-tpus.google.com/v2":"1"`,
			false,
			"",
		},
		{
			"Valid - custom string",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:    1.0,
						MemoryLimit: 0.65,
						Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
							Type:  "custom.example.com/accelerator-v1",
							Count: 1,
						},
					},
				},
				&pipelinespec.ComponentSpec{
					Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "addition"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{
						Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
							"a": {Type: pipelinespec.PrimitiveType_DOUBLE},
							"b": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"Output": {Type: pipelinespec.PrimitiveType_DOUBLE},
						},
					},
				},
				nil,
				1,
				"MyPipeline",
				"a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6",
			},
			`"custom.example.com/accelerator-v1":"1"`,
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makePodSpecPatch(tt.args.container, tt.args.componentSpec, tt.args.executorInput, tt.args.executionID, tt.args.pipelineName, tt.args.runID, nil, nil, nil, nil)
			if tt.wantErr {
				assert.Empty(t, got)
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.Contains(t, got, tt.want)
			}
		})
	}
}
