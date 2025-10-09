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
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
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
					Image:   "python:3.11",
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
					Image:   "python:3.11",
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
					Image:   "python:3.11",
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
					Image:   "python:3.11",
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
					Image:   "python:3.11",
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
			taskConfig := &TaskConfig{}

			podSpec, err := initPodSpecPatch(
				tt.args.container,
				tt.args.componentSpec,
				tt.args.executorInput,
				tt.args.executionID,
				tt.args.pipelineName,
				tt.args.runID,
				"my-run-name",
				tt.args.pipelineLogLevel,
				tt.args.publishLogs,
				"false",
				taskConfig,
				false,
				false,
				"",
				"metadata-grpc-service.kubeflow.svc.local",
				"8080",
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

			assert.Empty(t, taskConfig.Resources.Limits)
			assert.Empty(t, taskConfig.Resources.Requests)
		})
	}
}

func Test_initPodSpecPatch_resource_placeholders(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
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

	taskConfig := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"my-run-name",
		"1",
		"false",
		"false",
		taskConfig,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)
	assert.Len(t, podSpec.Containers, 1)

	res := podSpec.Containers[0].Resources
	assert.Equal(t, k8sres.MustParse("200m"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("400m"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("100Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("500Mi"), res.Limits[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1"), res.Limits[k8score.ResourceName("nvidia.com/gpu")])

	assert.Empty(t, taskConfig.Resources.Limits)
	assert.Empty(t, taskConfig.Resources.Requests)
}

func Test_initPodSpecPatch_legacy_resources(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
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
	taskConfig := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"my-run-name",
		"1",
		"false",
		"false",
		taskConfig,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)
	assert.Len(t, podSpec.Containers, 1)

	res := podSpec.Containers[0].Resources
	assert.Equal(t, k8sres.MustParse("200"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("400"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("100Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("500Mi"), res.Limits[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1"), res.Limits[k8score.ResourceName("nvidia.com/gpu")])

	assert.Empty(t, taskConfig.Resources.Limits)
	assert.Empty(t, taskConfig.Resources.Requests)
}

func Test_initPodSpecPatch_modelcar_input_artifact(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
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
	taskConfig := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"my-run-name",
		"1",
		"false",
		"false",
		taskConfig,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
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

	assert.Empty(t, taskConfig.Resources.Limits)
	assert.Empty(t, taskConfig.Resources.Requests)
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
		"my-run-name",
		"1",
		"true",
		"false",
		nil,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
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
					Image:   "python:3.11",
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
					Image:   "python:3.11",
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
			taskConfig := &TaskConfig{}

			podSpec, err := initPodSpecPatch(
				tt.args.container,
				tt.args.componentSpec,
				tt.args.executorInput,
				tt.args.executionID,
				tt.args.pipelineName,
				tt.args.runID,
				"my-run-name",
				tt.args.pipelineLogLevel,
				tt.args.publishLogs,
				"false",
				taskConfig,
				false,
				false,
				"",
				"metadata-grpc-service.kubeflow.svc.local",
				"8080",
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

			assert.Empty(t, taskConfig.Resources.Limits)
			assert.Empty(t, taskConfig.Resources.Requests)
		})
	}
}

func Test_initPodSpecPatch_TaskConfig_ForwardsResourcesOnly(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
		Args:    []string{"--function_to_execute", "add"},
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
		Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
			ResourceCpuLimit:      "2.0",
			ResourceMemoryLimit:   "1.5",
			ResourceCpuRequest:    "1.0",
			ResourceMemoryRequest: "0.65G",
		},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{
				Field:       pipelinespec.TaskConfigPassthroughType_RESOURCES,
				ApplyToTask: false,
			},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{}

	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"0254beba-0be4-4065-8d97-7dc5e3adf300",
		"my-run-name",
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)
	assert.NotNil(t, podSpec)
	assert.Len(t, podSpec.Containers, 1)

	assert.Empty(t, podSpec.Containers[0].Resources.Requests)
	assert.Empty(t, podSpec.Containers[0].Resources.Limits)

	// Forwarded resources captured in TaskConfig
	res := taskCfg.Resources
	assert.True(t, res.Requests[k8score.ResourceCPU].Equal(k8sres.MustParse("1")))
	assert.True(t, res.Limits[k8score.ResourceCPU].Equal(k8sres.MustParse("2")))
	assert.True(t, res.Requests[k8score.ResourceMemory].Equal(k8sres.MustParse("0.65G")))
	assert.True(t, res.Limits[k8score.ResourceMemory].Equal(k8sres.MustParse("1.5")))
}

func Test_initPodSpecPatch_inputTaskFinalStatus(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
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
		"my-run-name",
		"1",
		"false",
		"false",
		nil,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
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

func TestNeedsWorkspaceMount(t *testing.T) {
	tests := []struct {
		name          string
		executorInput *pipelinespec.ExecutorInput
		expected      bool
	}{
		{
			name: "workspace path placeholder in parameters",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"workspace_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "{{$.workspace_path}}",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "workspace path prefix in parameters",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "/kfp-workspace/data/file.txt",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "artifact with workspace metadata",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					Artifacts: map[string]*pipelinespec.ArtifactList{
						"model": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Metadata: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"_kfp_workspace": {
												Kind: &structpb.Value_BoolValue{
													BoolValue: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "workspace path with subdirectory",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "{{$.workspace_path}}/data",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no workspace usage",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"text": {
							Kind: &structpb.Value_StringValue{
								StringValue: "hello world",
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsWorkspaceMount(tt.executorInput)
			if result != tt.expected {
				t.Errorf("needsWorkspaceMount() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetWorkspaceMount(t *testing.T) {
	pvcName := "test-workflow-kfp-workspace"

	workspaceVolume, workspaceVolumeMount := getWorkspaceMount(pvcName)

	if workspaceVolume.Name != "kfp-workspace" {
		t.Errorf("Expected volume name kfp-workspace, got %s", workspaceVolume.Name)
	}

	if workspaceVolume.PersistentVolumeClaim == nil {
		t.Error("Expected PersistentVolumeClaim to be set")
	}

	if workspaceVolume.PersistentVolumeClaim.ClaimName != pvcName {
		t.Errorf("Expected claim name %s, got %s", pvcName, workspaceVolume.PersistentVolumeClaim.ClaimName)
	}

	if workspaceVolumeMount.Name != "kfp-workspace" {
		t.Errorf("Expected volume mount name kfp-workspace, got %s", workspaceVolumeMount.Name)
	}

	if workspaceVolumeMount.MountPath != "/kfp-workspace" {
		t.Errorf("Expected mount path /kfp-workspace, got %s", workspaceVolumeMount.MountPath)
	}
}

// Ensure that when workspace is used, missing RunName leads to an error during pod spec init.
func Test_initPodSpecPatch_WorkspaceRequiresRunName(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"}
	componentSpec := &pipelinespec.ComponentSpec{}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"workspace_param": {Kind: &structpb.Value_StringValue{StringValue: "{{$.workspace_path}}"}},
			},
		},
	}
	taskCfg := &TaskConfig{}
	_, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"run-id",
		"", // runName intentionally empty
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	require.NotNil(t, err)
}

func TestValidateVolumeMounts(t *testing.T) {
	tests := []struct {
		name        string
		podSpec     *k8score.PodSpec
		expectError bool
	}{
		{
			name: "no conflicting volume mounts",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "data",
								MountPath: "/data",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "conflicting volume mount path",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "workspace",
								MountPath: "/kfp-workspace",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "conflicting volume mount subpath",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "data",
								MountPath: "/kfp-workspace/data",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "conflicting volume name kfp-workspace",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "kfp-workspace",
								MountPath: "/data",
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumeMounts(tt.podSpec)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestWorkspaceMount_PassthroughVolumes_CaptureOnly(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{
				Field:       pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES,
				ApplyToTask: false,
			},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"workspace_param": {Kind: &structpb.Value_StringValue{StringValue: "{{$.workspace_path}}"}},
			},
		},
	}
	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec, componentSpec, executorInput,
		27, "test", "run", "my-run-name", "1", "false", "false", taskCfg, false, false, "", "metadata-grpc-service.kubeflow.svc.local", "8080",
	)
	assert.Nil(t, err)

	// Should not mount workspace to pod (no volumes on pod), only capture to TaskConfig
	assert.Empty(t, podSpec.Volumes)
	assert.Empty(t, podSpec.Containers[0].VolumeMounts)
	assert.NotEmpty(t, taskCfg.Volumes)
	assert.NotEmpty(t, taskCfg.VolumeMounts)

	if assert.Len(t, taskCfg.Volumes, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.Volumes[0].Name)
		if assert.NotNil(t, taskCfg.Volumes[0].PersistentVolumeClaim) {
			assert.Equal(t, "my-run-name-kfp-workspace", taskCfg.Volumes[0].PersistentVolumeClaim.ClaimName)
		}
	}

	if assert.Len(t, taskCfg.VolumeMounts, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.VolumeMounts[0].Name)
		assert.Equal(t, "/kfp-workspace", taskCfg.VolumeMounts[0].MountPath)
	}
}

func TestWorkspaceMount_PassthroughVolumes_ApplyAndCapture(t *testing.T) {
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{
				Field:       pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES,
				ApplyToTask: true,
			},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"workspace_param": {Kind: &structpb.Value_StringValue{StringValue: "{{$.workspace_path}}"}},
			},
		},
	}
	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec, componentSpec, executorInput,
		27, "test", "run", "my-run-name", "1", "false", "false", taskCfg, false, false, "", "metatadata-grpc-service.kubeflow.svc.local", "8080",
	)
	assert.Nil(t, err)
	// Should mount workspace to pod and also capture to TaskConfig
	assert.NotEmpty(t, podSpec.Volumes)
	assert.NotEmpty(t, podSpec.Containers[0].VolumeMounts)
	assert.NotEmpty(t, taskCfg.Volumes)
	assert.NotEmpty(t, taskCfg.VolumeMounts)

	if assert.Len(t, podSpec.Volumes, 1) {
		assert.Equal(t, "kfp-workspace", podSpec.Volumes[0].Name)
		if assert.NotNil(t, podSpec.Volumes[0].PersistentVolumeClaim) {
			assert.Equal(t, "my-run-name-kfp-workspace", podSpec.Volumes[0].PersistentVolumeClaim.ClaimName)
		}
	}

	if assert.Len(t, podSpec.Containers, 1) {
		if assert.Len(t, podSpec.Containers[0].VolumeMounts, 1) {
			assert.Equal(t, "kfp-workspace", podSpec.Containers[0].VolumeMounts[0].Name)
			assert.Equal(t, "/kfp-workspace", podSpec.Containers[0].VolumeMounts[0].MountPath)
		}
	}

	if assert.Len(t, taskCfg.Volumes, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.Volumes[0].Name)
		if assert.NotNil(t, taskCfg.Volumes[0].PersistentVolumeClaim) {
			assert.Equal(t, "my-run-name-kfp-workspace", taskCfg.Volumes[0].PersistentVolumeClaim.ClaimName)
		}
	}

	if assert.Len(t, taskCfg.VolumeMounts, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.VolumeMounts[0].Name)
		assert.Equal(t, "/kfp-workspace", taskCfg.VolumeMounts[0].MountPath)
	}
}

func TestWorkspaceMount_TriggeredByArtifactMetadata(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.9"}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{
				Field:       pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES,
				ApplyToTask: true,
			},
		},
	}

	// Build an ExecutorInput that does NOT reference workspace path in params,
	// but contains an input artifact marked as already in workspace.
	execInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"data": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "minio://mlpipeline/sample/sample.txt",
							Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
								"_kfp_workspace": structpb.NewBoolValue(true),
							}},
						},
					},
				},
			},
		},
	}

	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec, componentSpec, execInput,
		27, "test", "run", "my-run-name", "1", "false", "false", taskCfg, false, false, "",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)

	// Expect workspace volume mounted
	if assert.Len(t, podSpec.Volumes, 1) {
		assert.Equal(t, "kfp-workspace", podSpec.Volumes[0].Name)
		if assert.NotNil(t, podSpec.Volumes[0].PersistentVolumeClaim) {
			assert.Equal(t, "my-run-name-kfp-workspace", podSpec.Volumes[0].PersistentVolumeClaim.ClaimName)
		}
	}
	if assert.Len(t, podSpec.Containers, 1) {
		if assert.Len(t, podSpec.Containers[0].VolumeMounts, 1) {
			assert.Equal(t, "kfp-workspace", podSpec.Containers[0].VolumeMounts[0].Name)
			assert.Equal(t, "/kfp-workspace", podSpec.Containers[0].VolumeMounts[0].MountPath)
		}
	}

	// Expect volumes to be captured in TaskConfig
	if assert.Len(t, taskCfg.Volumes, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.Volumes[0].Name)
	}
	if assert.Len(t, taskCfg.VolumeMounts, 1) {
		assert.Equal(t, "kfp-workspace", taskCfg.VolumeMounts[0].Name)
		assert.Equal(t, "/kfp-workspace", taskCfg.VolumeMounts[0].MountPath)
	}
}

func Test_initPodSpecPatch_TaskConfig_Env_Passthrough_CaptureOnly(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image: "python:3.11",
		Env: []*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_EnvVar{{
			Name:  "FOO",
			Value: "bar",
		}},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{Field: pipelinespec.TaskConfigPassthroughType_ENV, ApplyToTask: false},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{}
	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"run",
		"my-run-name",
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)

	// Env should be captured to TaskConfig only, not applied to pod
	assert.Empty(t, podSpec.Containers[0].Env)

	if assert.Len(t, taskCfg.Env, 1) {
		assert.Equal(t, "FOO", taskCfg.Env[0].Name)
		assert.Equal(t, "bar", taskCfg.Env[0].Value)
	}
}

func Test_initPodSpecPatch_TaskConfig_Resources_Passthrough_ApplyAndCapture(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.11",
		Args:    []string{"--function_to_execute", "add"},
		Command: []string{"sh", "-ec", "python3 -m kfp.components.executor_main"},
		Resources: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{
			CpuLimit:      2.0,
			MemoryLimit:   1.5,
			CpuRequest:    1.0,
			MemoryRequest: 0.65,
		},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{Field: pipelinespec.TaskConfigPassthroughType_RESOURCES, ApplyToTask: true},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{}
	taskCfg := &TaskConfig{}
	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"run",
		"my-run-name",
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)
	// Resources should be both on pod and in TaskConfig
	assert.NotEmpty(t, podSpec.Containers[0].Resources.Requests)
	assert.NotEmpty(t, podSpec.Containers[0].Resources.Limits)
	assert.NotEmpty(t, taskCfg.Resources.Requests)
	assert.NotEmpty(t, taskCfg.Resources.Limits)

	resPod := podSpec.Containers[0].Resources
	assert.Equal(t, k8sres.MustParse("1"), resPod.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("2"), resPod.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("0.65G"), resPod.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1.5G"), resPod.Limits[k8score.ResourceMemory])

	resCfg := taskCfg.Resources
	assert.Equal(t, k8sres.MustParse("1"), resCfg.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("2"), resCfg.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("0.65G"), resCfg.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1.5G"), resCfg.Limits[k8score.ResourceMemory])
}

func Test_initPodSpecPatch_TaskConfig_Affinity_NodeSelector_Tolerations_Passthrough(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_AFFINITY, ApplyToTask: false},
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_NODE_SELECTOR, ApplyToTask: false},
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_TOLERATIONS, ApplyToTask: false},
		},
	}

	secs := int64(3600)
	k8sExecCfg := &kubernetesplatform.KubernetesExecutorConfig{
		NodeSelector: &kubernetesplatform.NodeSelector{Labels: map[string]string{"disktype": "ssd"}},
		Tolerations: []*kubernetesplatform.Toleration{{
			Key:               "example-key",
			Operator:          "Exists",
			Effect:            "NoExecute",
			TolerationSeconds: &secs,
		}},
		NodeAffinity: []*kubernetesplatform.NodeAffinityTerm{{
			MatchExpressions: []*kubernetesplatform.SelectorRequirement{{
				Key:      "zone",
				Operator: "In",
				Values:   []string{"us-west-1"},
			}},
		}},
	}

	opts := Options{
		PipelineName:             "p",
		RunID:                    "r",
		Component:                componentSpec,
		Container:                containerSpec,
		KubernetesExecutorConfig: k8sExecCfg,
	}

	executorInput := &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{ParameterValues: map[string]*structpb.Value{}}}

	taskCfg := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"run",
		"my-run-name",
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)

	err = extendPodSpecPatch(
		context.Background(),
		podSpec,
		opts,
		nil,
		nil,
		nil,
		map[string]*structpb.Value{},
		taskCfg,
	)
	assert.Nil(t, err)

	assert.Nil(t, podSpec.Affinity)
	assert.Empty(t, podSpec.NodeSelector)
	assert.Empty(t, podSpec.Tolerations)

	assert.Equal(t, map[string]string{"disktype": "ssd"}, taskCfg.NodeSelector)
	if assert.Len(t, taskCfg.Tolerations, 1) {
		assert.Equal(t, "example-key", taskCfg.Tolerations[0].Key)
		assert.Equal(t, k8score.TaintEffect("NoExecute"), taskCfg.Tolerations[0].Effect)
		if assert.NotNil(t, taskCfg.Tolerations[0].TolerationSeconds) {
			assert.Equal(t, int64(3600), *taskCfg.Tolerations[0].TolerationSeconds)
		}
	}

	if assert.NotNil(t, taskCfg.Affinity) && assert.NotNil(t, taskCfg.Affinity.NodeAffinity) {
		if assert.NotNil(t, taskCfg.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution) {
			terms := taskCfg.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			if assert.NotEmpty(t, terms) && assert.NotEmpty(t, terms[0].MatchExpressions) {
				expr := terms[0].MatchExpressions[0]
				assert.Equal(t, "zone", expr.Key)
				assert.Equal(t, k8score.NodeSelectorOpIn, expr.Operator)
				assert.Equal(t, []string{"us-west-1"}, expr.Values)
			}
		}
	}
}

func Test_initPodSpecPatch_TaskConfig_Affinity_NodeSelector_Tolerations_ApplyAndCapture(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"}
	componentSpec := &pipelinespec.ComponentSpec{
		TaskConfigPassthroughs: []*pipelinespec.TaskConfigPassthrough{
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_AFFINITY, ApplyToTask: true},
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_NODE_SELECTOR, ApplyToTask: true},
			{Field: pipelinespec.TaskConfigPassthroughType_KUBERNETES_TOLERATIONS, ApplyToTask: true},
		},
	}

	secs := int64(3600)
	weight := int32(100)
	k8sExecCfg := &kubernetesplatform.KubernetesExecutorConfig{
		NodeSelector: &kubernetesplatform.NodeSelector{Labels: map[string]string{"disktype": "ssd"}},
		Tolerations: []*kubernetesplatform.Toleration{{
			Key:               "example-key",
			Operator:          "Exists",
			Effect:            "NoExecute",
			TolerationSeconds: &secs,
		}},
		NodeAffinity: []*kubernetesplatform.NodeAffinityTerm{{
			MatchExpressions: []*kubernetesplatform.SelectorRequirement{{
				Key:      "zone",
				Operator: "In",
				Values:   []string{"us-west-1"},
			}},
			Weight: &weight,
		}},
	}

	opts := Options{
		PipelineName:             "p",
		RunID:                    "r",
		Component:                componentSpec,
		Container:                containerSpec,
		KubernetesExecutorConfig: k8sExecCfg,
	}

	executorInput := &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{ParameterValues: map[string]*structpb.Value{}}}
	taskCfg := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		containerSpec,
		componentSpec,
		executorInput,
		27,
		"test",
		"run",
		"my-run-name",
		"1",
		"false",
		"false",
		taskCfg,
		false,
		false,
		"",
		"metadata-grpc-service.kubeflow.svc.local",
		"8080",
	)
	assert.Nil(t, err)

	err = extendPodSpecPatch(
		context.Background(),
		podSpec,
		opts,
		nil,
		nil,
		nil,
		map[string]*structpb.Value{},
		taskCfg,
	)
	assert.Nil(t, err)

	assert.Equal(t, map[string]string{"disktype": "ssd"}, podSpec.NodeSelector)
	if assert.Len(t, podSpec.Tolerations, 1) {
		assert.Equal(t, "example-key", podSpec.Tolerations[0].Key)
		assert.Equal(t, k8score.TaintEffect("NoExecute"), podSpec.Tolerations[0].Effect)
		if assert.NotNil(t, podSpec.Tolerations[0].TolerationSeconds) {
			assert.Equal(t, int64(3600), *podSpec.Tolerations[0].TolerationSeconds)
		}
	}

	if assert.NotNil(t, podSpec.Affinity) && assert.NotNil(t, podSpec.Affinity.NodeAffinity) {
		prefs := podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		if assert.NotEmpty(t, prefs) {
			assert.Equal(t, int32(100), prefs[0].Weight)
			if assert.NotEmpty(t, prefs[0].Preference.MatchExpressions) {
				expr := prefs[0].Preference.MatchExpressions[0]
				assert.Equal(t, "zone", expr.Key)
				assert.Equal(t, k8score.NodeSelectorOpIn, expr.Operator)
				assert.Equal(t, []string{"us-west-1"}, expr.Values)
			}
		}
	}

	assert.Equal(t, map[string]string{"disktype": "ssd"}, taskCfg.NodeSelector)
	if assert.Len(t, taskCfg.Tolerations, 1) {
		assert.Equal(t, "example-key", taskCfg.Tolerations[0].Key)
		assert.Equal(t, k8score.TaintEffect("NoExecute"), taskCfg.Tolerations[0].Effect)
		if assert.NotNil(t, taskCfg.Tolerations[0].TolerationSeconds) {
			assert.Equal(t, int64(3600), *taskCfg.Tolerations[0].TolerationSeconds)
		}
	}

	if assert.NotNil(t, taskCfg.Affinity) && assert.NotNil(t, taskCfg.Affinity.NodeAffinity) {
		prefs := taskCfg.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		if assert.NotEmpty(t, prefs) {
			assert.Equal(t, int32(100), prefs[0].Weight)
			if assert.NotEmpty(t, prefs[0].Preference.MatchExpressions) {
				expr := prefs[0].Preference.MatchExpressions[0]
				assert.Equal(t, "zone", expr.Key)
				assert.Equal(t, k8score.NodeSelectorOpIn, expr.Operator)
				assert.Equal(t, []string{"us-west-1"}, expr.Values)
			}
		}
	}
}
