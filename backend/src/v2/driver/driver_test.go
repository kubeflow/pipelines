// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_initPodSpecPatch_acceleratorConfig(t *testing.T) {
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
			podSpec, err := initPodSpecPatch(tt.args.container, tt.args.componentSpec, tt.args.executorInput, tt.args.executionID, tt.args.pipelineName, tt.args.runID)
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

func Test_createPVC(t *testing.T) {
	type args struct {
		inputs    *pipelinespec.ExecutorInput_Inputs
		namespace string
	}
	k8sClient := fake.NewSimpleClientset()
	accessModesList, _ := structpb.NewList([]interface{}{"ReadWriteOnce"})
	accessModes := structpb.NewListValue(accessModesList)
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errMsg  string
	}{
		{
			"valid",
			args{
				&pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"access_modes":       accessModes,
						"pvc_name":           structpb.NewStringValue("my-pvc"),
						"size":               structpb.NewStringValue("5Gi"),
						"storage_class_name": structpb.NewStringValue("standard"),
						"volume_name":        structpb.NewStringValue("volume-name"),
					},
				},
				"default",
			},
			false,
			"",
		},
		{
			"missing access_modes",
			args{
				&pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"access_modes":       nil,
						"pvc_name":           structpb.NewStringValue("my-pvc-2"),
						"size":               structpb.NewStringValue("5Gi"),
						"storage_class_name": structpb.NewStringValue("standard"),
						"volume_name":        structpb.NewStringValue("volume-name"),
					},
				},
				"default",
			},
			true,
			"parameter access_modes not provided",
		},
		{
			"both pvc_name_suffix and pvc_name provided",
			args{
				&pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"access_modes":       accessModes,
						"pvc_name":           structpb.NewStringValue("my-pvc-3"),
						"pvc_name_suffix":    structpb.NewStringValue("-my-pvc"),
						"size":               structpb.NewStringValue("5Gi"),
						"storage_class_name": structpb.NewStringValue("standard"),
						"volume_name":        structpb.NewStringValue("volume-name"),
					},
				},
				"default",
			},
			true,
			"at most one of pvc_name and pvc_name_suffix can be non-empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvcName, err := createPVC(k8sClient, tt.args.inputs, tt.args.namespace)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, pvcName, "")
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				pvc, err := k8sClient.CoreV1().PersistentVolumeClaims("default").Get(context.TODO(), pvcName, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, pvc)
			}
		})
	}
}

func Test_deletePVC(t *testing.T) {
	type args struct {
		inputs    *pipelinespec.ExecutorInput_Inputs
		namespace string
	}
	k8sClient := fake.NewSimpleClientset()
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errMsg  string
	}{
		{
			"valid",
			args{
				&pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"pvc_name": structpb.NewStringValue("my-pvc"),
					},
				},
				"default",
			},
			true,
			"not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deletePVC(k8sClient, tt.args.inputs, tt.args.namespace)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_makeVolumeMountPatch(t *testing.T) {
	type args struct {
		pvcMount []*kubernetesplatform.PvcMount
		dag      *metadata.DAG
		dagTasks map[string]*metadata.Execution
	}
	// TODO(lingqinggan): add more test cases for task output parameter and component input.
	// Omitted now due to type Execution defined in metadata has unexported fields.
	tests := []struct {
		name     string
		args     args
		wantPath string
		wantName string
		wantErr  bool
		errMsg   string
	}{
		{
			"pvc name: constant",
			args{
				[]*kubernetesplatform.PvcMount{
					&kubernetesplatform.PvcMount{
						MountPath:    "/mnt/path",
						PvcReference: &kubernetesplatform.PvcMount_Constant{Constant: "pvc-name"},
					},
				},
				nil,
				nil,
			},
			"/mnt/path",
			"pvc-name",
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumeMounts, volumes, err := makeVolumeMountPatch(tt.args.pvcMount, tt.args.dag, tt.args.dagTasks)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, volumeMounts)
				assert.Nil(t, volumes)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, 1, len(volumeMounts))
				assert.Equal(t, 1, len(volumes))
				assert.Equal(t, volumeMounts[0].MountPath, tt.wantPath)
				assert.Equal(t, volumeMounts[0].Name, tt.wantName)
				assert.Equal(t, volumes[0].Name, tt.wantName)
			}
		})
	}
}

func Test_initPodSpecPatch_resourceRequests(t *testing.T) {
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
		notWant string
	}{
		{
			"Valid - with requests",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
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
			},
			`"resources":{"limits":{"cpu":"2","memory":"1500M"},"requests":{"cpu":"1","memory":"650M"}}`,
			"",
		},
		{
			"Valid - zero requests",
			args{
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.7",
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
			},
			`"resources":{"limits":{"cpu":"2","memory":"1500M"}}`,
			`"requests"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podSpec, err := initPodSpecPatch(tt.args.container, tt.args.componentSpec, tt.args.executorInput, tt.args.executionID, tt.args.pipelineName, tt.args.runID)
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
