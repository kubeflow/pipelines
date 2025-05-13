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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
)

func Test_initPodSpecPatch_acceleratorConfig(t *testing.T) {
	viper.Set("KFP_POD_NAME", "MyWorkflowPod")
	viper.Set("KFP_POD_UID", "a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6")

	proxy.InitializeConfigWithEmptyForTests()

	type args struct {
		container        *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
		componentSpec    *pipelinespec.ComponentSpec
		executorInput    *pipelinespec.ExecutorInput
		executionID      int64
		pipelineName     string
		runID            string
		pipelineLogLevel string
		publishLogs      string
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
					Image:   "python:3.9",
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
				"1",
				"false",
			},
			`"nvidia.com/gpu":"1"`,
			false,
			"",
		},
		{
			"Valid - amd.com/gpu",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
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
				"1",
				"false",
			},
			`"amd.com/gpu":"1"`,
			false,
			"",
		},
		{
			"Valid - cloud-tpus.google.com/v3",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
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
				"1",
				"false",
			},
			`"cloud-tpus.google.com/v3":"1"`,
			false,
			"",
		},
		{
			"Valid - cloud-tpus.google.com/v2",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
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
				"1",
				"false",
			},
			`"cloud-tpus.google.com/v2":"1"`,
			false,
			"",
		},
		{
			"Valid - custom string",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
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
				"1",
				"false",
			},
			`"custom.example.com/accelerator-v1":"1"`,
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podSpec, err := initPodSpecPatch(
				tt.args.container,
				tt.args.componentSpec,
				tt.args.executorInput,
				tt.args.executionID,
				tt.args.pipelineName,
				tt.args.runID,
				tt.args.pipelineLogLevel,
				tt.args.publishLogs,
				"false",
			)
			if tt.wantErr {
				assert.Nil(t, podSpec)
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				podSpecString, err := json.Marshal(podSpec)
				assert.Nil(t, err)
				assert.Contains(t, string(podSpecString), tt.want)
			}
		})
	}
}

func Test_initPodSpecPatch_resource_placeholders(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.9",
		Args:    []string{"--function_to_execute", "add"},
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
		Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
			ResourceCpuRequest:    "{{$.inputs.parameters['pipelinechannel--cpu_request']}}",
			ResourceCpuLimit:      "{{$.inputs.parameters['pipelinechannel--cpu_limit']}}",
			ResourceMemoryRequest: "{{$.inputs.parameters['pipelinechannel--memory_request']}}",
			ResourceMemoryLimit:   "{{$.inputs.parameters['pipelinechannel--memory_limit']}}",
			Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
				ResourceType:  "{{$.inputs.parameters['pipelinechannel--accelerator_type']}}",
				ResourceCount: "{{$.inputs.parameters['pipelinechannel--accelerator_count']}}",
			},
		},
	}
	componentSpec := &pipelinespec.ComponentSpec{}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"cpu_request": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--cpu_request']}}",
					},
				},
				"pipelinechannel--cpu_request": {
					Kind: &structpb.Value_StringValue{
						StringValue: "200m",
					},
				},
				"cpu_limit": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--cpu_limit']}}",
					},
				},
				"pipelinechannel--cpu_limit": {
					Kind: &structpb.Value_StringValue{
						StringValue: "400m",
					},
				},
				"memory_request": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--memory_request']}}",
					},
				},
				"pipelinechannel--memory_request": {
					Kind: &structpb.Value_StringValue{
						StringValue: "100Mi",
					},
				},
				"memory_limit": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--memory_limit']}}",
					},
				},
				"pipelinechannel--memory_limit": {
					Kind: &structpb.Value_StringValue{
						StringValue: "500Mi",
					},
				},
				"accelerator_type": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--accelerator_type']}}",
					},
				},
				"pipelinechannel--accelerator_type": {
					Kind: &structpb.Value_StringValue{
						StringValue: "nvidia.com/gpu",
					},
				},
				"accelerator_count": {
					Kind: &structpb.Value_StringValue{
						StringValue: "{{$.inputs.parameters['pipelinechannel--accelerator_count']}}",
					},
				},
				"pipelinechannel--accelerator_count": {
					Kind: &structpb.Value_StringValue{
						StringValue: "1",
					},
				},
			},
		},
	}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"1",
		"false",
		"false",
	)
	assert.Nil(t, err)
	assert.Len(t, podSpec.Containers, 1)

	res := podSpec.Containers[0].Resources
	assert.Equal(t, k8sres.MustParse("200m"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("400m"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("100Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("500Mi"), res.Limits[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1"), res.Limits[k8score.ResourceName("nvidia.com/gpu")])
}

func Test_initPodSpecPatch_legacy_resources(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.9",
		Args:    []string{"--function_to_execute", "add"},
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
		Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
			CpuRequest:            200,
			CpuLimit:              400,
			ResourceMemoryRequest: "100Mi",
			ResourceMemoryLimit:   "500Mi",
			Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{
				Type:  "nvidia.com/gpu",
				Count: 1,
			},
		},
	}
	componentSpec := &pipelinespec.ComponentSpec{}
	executorInput := &pipelinespec.ExecutorInput{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"1",
		"false",
		"false",
	)
	assert.Nil(t, err)
	assert.Len(t, podSpec.Containers, 1)

	res := podSpec.Containers[0].Resources
	assert.Equal(t, k8sres.MustParse("200"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("400"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("100Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("500Mi"), res.Limits[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1"), res.Limits[k8score.ResourceName("nvidia.com/gpu")])
}

func Test_initPodSpecPatch_modelcar_input_artifact(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.9",
		Args:    []string{"--function_to_execute", "add"},
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
	}
	componentSpec := &pipelinespec.ComponentSpec{}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"my-model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "oci://registry.domain.local/my-model:latest",
						},
					},
				},
			},
		},
	}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"1",
		"false",
		"false",
	)
	assert.Nil(t, err)

	assert.Len(t, podSpec.InitContainers, 1)
	assert.Equal(t, podSpec.InitContainers[0].Name, "oci-prepull-0")
	assert.Equal(t, podSpec.InitContainers[0].Image, "registry.domain.local/my-model:latest")

	assert.Len(t, podSpec.Volumes, 1)
	assert.Equal(t, podSpec.Volumes[0].Name, "oci-0")
	assert.NotNil(t, podSpec.Volumes[0].EmptyDir)

	assert.Len(t, podSpec.Containers, 2)
	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "oci-0")
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].MountPath, "/oci/registry.domain.local_my-model:latest")
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].SubPath, "registry.domain.local_my-model:latest")

	assert.Equal(t, podSpec.Containers[1].Name, "oci-0")
	assert.Equal(t, podSpec.Containers[1].Image, "registry.domain.local/my-model:latest")
	assert.Len(t, podSpec.Containers[1].VolumeMounts, 1)
	assert.Equal(t, podSpec.Containers[1].VolumeMounts[0].Name, "oci-0")
	assert.Equal(t, podSpec.Containers[1].VolumeMounts[0].MountPath, "/oci/registry.domain.local_my-model:latest")
	assert.Equal(t, podSpec.Containers[1].VolumeMounts[0].SubPath, "registry.domain.local_my-model:latest")
}

// Validate that setting publishLogs to true propagates to the driver container
// commands in the podSpec.
func Test_initPodSpecPatch_publishLogs(t *testing.T) {
	podSpec, err := initPodSpecPatch(
		&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{},
		&pipelinespec.ComponentSpec{},
		&pipelinespec.ExecutorInput{},
		// executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"1",
		"true",
		"false",
	)
	assert.Nil(t, err)
	cmd := podSpec.Containers[0].Command
	assert.Contains(t, cmd, "--publish_logs")
	// TODO: There may be a simpler way to check this.
	for idx, val := range cmd {
		if val == "--publish_logs" {
			assert.Equal(t, cmd[idx+1], "true")
		}
	}

}

func Test_initPodSpecPatch_resourceRequests(t *testing.T) {
	viper.Set("KFP_POD_NAME", "MyWorkflowPod")
	viper.Set("KFP_POD_UID", "a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6")
	type args struct {
		container        *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
		componentSpec    *pipelinespec.ComponentSpec
		executorInput    *pipelinespec.ExecutorInput
		executionID      int64
		pipelineName     string
		runID            string
		pipelineLogLevel string
		publishLogs      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		notWant string
	}{
		{
			"Valid - with requests",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:      2.0,
						MemoryLimit:   1.5,
						CpuRequest:    1.0,
						MemoryRequest: 0.65,
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
				"1",
				"false",
			},
			`"resources":{"limits":{"cpu":"2","memory":"1500M"},"requests":{"cpu":"1","memory":"650M"}}`,
			"",
		},
		{
			"Valid - zero requests",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.9",
					Args:    []string{"--function_to_execute", "add"},
					Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
					Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
						CpuLimit:      2.0,
						MemoryLimit:   1.5,
						CpuRequest:    0,
						MemoryRequest: 0,
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
				"1",
				"false",
			},
			`"resources":{"limits":{"cpu":"2","memory":"1500M"}}`,
			`"requests"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podSpec, err := initPodSpecPatch(
				tt.args.container,
				tt.args.componentSpec,
				tt.args.executorInput,
				tt.args.executionID,
				tt.args.pipelineName,
				tt.args.runID,
				tt.args.pipelineLogLevel,
				tt.args.publishLogs,
				"false",
			)
			assert.Nil(t, err)
			assert.NotEmpty(t, podSpec)
			podSpecString, err := json.Marshal(podSpec)
			assert.Nil(t, err)
			if tt.want != "" {
				assert.Contains(t, string(podSpecString), tt.want)
			}
			if tt.notWant != "" {
				assert.NotContains(t, string(podSpecString), tt.notWant)
			}
		})
	}
}

func Test_initPodSpecPatch_inputTaskFinalStatus(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.9",
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
		Args:    []string{"--executor-input", "{{$}}", "--function_to_execute", "exit-op"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-exit-op"},
		InputDefinitions: &pipelinespec.ComponentInputsSpec{
			Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
				"status": {ParameterType: pipelinespec.ParameterType_TASK_FINAL_STATUS},
			},
		},
	}
	finalStatusStruct, err := structpb.NewStruct(map[string]interface{}{
		"state":                   "test-state",
		"pipelineTaskName":        "test-pipeline-task-name",
		"pipelineJobResourceName": "test-job-resource-name",
		"error":                   map[string]interface{}{},
	})
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"status": {
					Kind: &structpb.Value_StructValue{
						StructValue: finalStatusStruct,
					},
				},
			},
		},
	}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"1",
		"false",
		"false",
	)
	require.Nil(t, err)

	expectedExecutorInput := map[string]interface{}{
		"inputs": map[string]interface{}{
			"parameterValues": map[string]interface{}{
				"status": map[string]interface{}{
					"error":                   map[string]interface{}{},
					"pipelineJobResourceName": "test-job-resource-name",
					"pipelineTaskName":        "test-pipeline-task-name",
					"state":                   "test-state",
				},
			},
		},
	}
	expectedComponentSpec := map[string]interface{}{
		"executorLabel": "exec-exit-op",
		"inputDefinitions": map[string]interface{}{
			"parameters": map[string]interface{}{
				"status": map[string]interface{}{
					"parameterType": "TASK_FINAL_STATUS",
				},
			},
		},
	}
	actualExecutorInput := map[string]interface{}{}
	actualComponentSpec := map[string]interface{}{}

	for i, arg := range podSpec.Containers[0].Command {
		if arg == "--executor_input" {
			err := json.Unmarshal([]byte(podSpec.Containers[0].Command[i+1]), &actualExecutorInput)
			fmt.Println(podSpec.Containers[0].Command[i+1])
			require.Nil(t, err)
		}
		if arg == "--component_spec" {
			err := json.Unmarshal([]byte(podSpec.Containers[0].Command[i+1]), &actualComponentSpec)
			require.Nil(t, err)
		}
	}

	assert.Equal(t, expectedExecutorInput, actualExecutorInput)
	assert.Equal(t, expectedComponentSpec, actualComponentSpec)
}
