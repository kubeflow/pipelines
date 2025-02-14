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
	"testing"

	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
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
			podSpec, err := initPodSpecPatch(tt.args.container, tt.args.componentSpec, tt.args.executorInput, tt.args.executionID, tt.args.pipelineName, tt.args.runID, false, "unused-mlmd-server-address", "unused-mlmd-server-port", false, "unused-ca-cert-path")
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
					{
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
				assert.Equal(t, volumes[0].PersistentVolumeClaim.ClaimName, tt.wantName)
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
			podSpec, err := initPodSpecPatch(tt.args.container, tt.args.componentSpec, tt.args.executorInput, tt.args.executionID, tt.args.pipelineName, tt.args.runID, false, "unused-mlmd-server-address", "unused-mlmd-server-port", false, "unused-ca-cert-path")
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

func Test_makePodSpecPatch_nodeSelector(t *testing.T) {
	viper.Set("KFP_POD_NAME", "MyWorkflowPod")
	viper.Set("KFP_POD_UID", "a1b2c3d4-a1b2-a1b2-a1b2-a1b2c3d4e5f6")
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - NVIDIA GPU on GKE",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					Labels: map[string]string{
						"cloud.google.com/gke-accelerator": "nvidia-tesla-k80",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				NodeSelector: map[string]string{"cloud.google.com/gke-accelerator": "nvidia-tesla-k80"},
			},
		},
		{
			"Valid - operating system and arch",
			&kubernetesplatform.KubernetesExecutorConfig{
				NodeSelector: &kubernetesplatform.NodeSelector{
					Labels: map[string]string{
						"beta.kubernetes.io/os":   "linux",
						"beta.kubernetes.io/arch": "amd64",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				NodeSelector: map[string]string{"beta.kubernetes.io/arch": "amd64", "beta.kubernetes.io/os": "linux"},
			},
		},
		{
			"Valid - empty",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(got, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_Secret(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - secret as volume",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName: "secret1",
						MountPath:  "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{false}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - secret as volume with optional false",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName: "secret1",
						MountPath:  "/data/path",
						Optional:   &[]bool{false}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{false}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - secret as volume with optional true",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsVolume: []*kubernetesplatform.SecretAsVolume{
					{
						SecretName: "secret1",
						MountPath:  "/data/path",
						Optional:   &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "secret1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "secret1",
						VolumeSource: k8score.VolumeSource{
							Secret: &k8score.SecretVolumeSource{SecretName: "secret1", Optional: &[]bool{true}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - secret not specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - secret as env",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretName: "my-secret",
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										k8score.LocalObjectReference{Name: "my-secret"},
										"password",
										nil,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(tt.podSpec, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_ConfigMap(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - config map as volume",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName: "cm1",
						MountPath:     "/data/path",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{false}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - config map as volume with optional false",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName: "cm1",
						MountPath:     "/data/path",
						Optional:      &[]bool{false}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{false}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - config map as volume with optional true",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsVolume: []*kubernetesplatform.ConfigMapAsVolume{
					{
						ConfigMapName: "cm1",
						MountPath:     "/data/path",
						Optional:      &[]bool{true}[0],
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "cm1",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "cm1",
						VolumeSource: k8score.VolumeSource{
							ConfigMap: &k8score.ConfigMapVolumeSource{
								LocalObjectReference: k8score.LocalObjectReference{Name: "cm1"},
								Optional:             &[]bool{true}[0]},
						},
					},
				},
			},
		},
		{
			"Valid - config map not specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - config map as env",
			&kubernetesplatform.KubernetesExecutorConfig{
				ConfigMapAsEnv: []*kubernetesplatform.ConfigMapAsEnv{
					{
						ConfigMapName: "my-cm",
						KeyToEnv: []*kubernetesplatform.ConfigMapAsEnv_ConfigMapKeyToEnvMap{
							{
								ConfigMapKey: "foo",
								EnvVar:       "CONFIG_MAP_VAR",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "CONFIG_MAP_VAR",
								ValueFrom: &k8score.EnvVarSource{
									ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
										k8score.LocalObjectReference{Name: "my-cm"},
										"foo",
										nil,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(tt.podSpec, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_ImagePullSecrets(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - SecretA and SecretB",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "SecretA"},
					{SecretName: "SecretB"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ImagePullSecrets: []k8score.LocalObjectReference{
					{Name: "SecretA"},
					{Name: "SecretB"},
				},
			},
		},
		{
			"Valid - No ImagePullSecrets",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - empty",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(got, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_Tolerations(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - toleration",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						Key:      "key1",
						Operator: "Equal",
						Value:    "value1",
						Effect:   "NoSchedule",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Key:               "key1",
						Operator:          "Equal",
						Value:             "value1",
						Effect:            "NoSchedule",
						TolerationSeconds: nil,
					},
				},
			},
		},
		{
			"Valid - no tolerations",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - only pass operator",
			&kubernetesplatform.KubernetesExecutorConfig{
				Tolerations: []*kubernetesplatform.Toleration{
					{
						Operator: "Contains",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				Tolerations: []k8score.Toleration{
					{
						Operator: "Contains",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(got, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_FieldPathAsEnv(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - FieldPathAsEnv",
			&kubernetesplatform.KubernetesExecutorConfig{
				FieldPathAsEnv: []*kubernetesplatform.FieldPathAsEnv{
					{Name: "KFP_RUN_NAME", FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "KFP_RUN_NAME",
								ValueFrom: &k8score.EnvVarSource{
									FieldRef: &k8score.ObjectFieldSelector{
										FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"Valid - Mix env values",
			&kubernetesplatform.KubernetesExecutorConfig{
				SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
					{
						SecretName: "my-secret",
						KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{
							{
								SecretKey: "password",
								EnvVar:    "SECRET_VAR",
							},
						},
					},
				},
				FieldPathAsEnv: []*kubernetesplatform.FieldPathAsEnv{
					{Name: "KFP_RUN_NAME", FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']"},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						Env: []k8score.EnvVar{
							{
								Name: "SECRET_VAR",
								ValueFrom: &k8score.EnvVarSource{
									SecretKeyRef: &k8score.SecretKeySelector{
										k8score.LocalObjectReference{Name: "my-secret"},
										"password",
										nil,
									},
								},
							},
							{
								Name: "KFP_RUN_NAME",
								ValueFrom: &k8score.EnvVarSource{
									FieldRef: &k8score.ObjectFieldSelector{
										FieldPath: "metadata.annotations['pipelines.kubeflow.org/run_name']",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(got, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_ActiveDeadlineSeconds(t *testing.T) {
	var timeoutSeconds int64 = 20
	var NegativeTimeoutSeconds int64 = -20
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		expected   *k8score.PodSpec
	}{
		{
			"Valid - With ActiveDeadlineSeconds",
			&kubernetesplatform.KubernetesExecutorConfig{
				ActiveDeadlineSeconds: timeoutSeconds,
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
				ActiveDeadlineSeconds: &timeoutSeconds,
			},
		},
		{
			"Valid - Negative input ignored",
			&kubernetesplatform.KubernetesExecutorConfig{
				ActiveDeadlineSeconds: NegativeTimeoutSeconds,
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - No ActiveDeadlineSeconds",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &k8score.PodSpec{Containers: []k8score.Container{
				{
					Name: "main",
				},
			}}
			err := extendPodSpecPatch(got, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_extendPodSpecPatch_ImagePullPolicy(t *testing.T) {
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - Always",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "Always",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "Always",
					},
				},
			},
		},
		{
			"Valid - IfNotPresent",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "IfNotPresent",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "IfNotPresent",
					},
				},
			},
		},
		{
			"Valid - Never",
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullPolicy: "Never",
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name:            "main",
						ImagePullPolicy: "Never",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(tt.podSpec, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}

func Test_extendPodSpecPatch_GenericEphemeralVolume(t *testing.T) {
	storageClass := "storageClass"
	tests := []struct {
		name       string
		k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
		podSpec    *k8score.PodSpec
		expected   *k8score.PodSpec
	}{
		{
			"Valid - single volume added (default storage class)",
			&kubernetesplatform.KubernetesExecutorConfig{
				GenericEphemeralVolume: []*kubernetesplatform.GenericEphemeralVolume{
					{
						VolumeName:          "volume",
						MountPath:           "/data/path",
						AccessModes:         []string{"ReadWriteOnce"},
						Size:                "5Gi",
						DefaultStorageClass: true,
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "volume",
								MountPath: "/data/path",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "volume",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.ResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("5Gi"),
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
		{
			"Valid - no generic volumes specified",
			&kubernetesplatform.KubernetesExecutorConfig{},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
		},
		{
			"Valid - multiple volumes specified (one with labels, one with storage class)",
			&kubernetesplatform.KubernetesExecutorConfig{
				GenericEphemeralVolume: []*kubernetesplatform.GenericEphemeralVolume{
					{
						VolumeName:          "volume",
						MountPath:           "/data/path",
						AccessModes:         []string{"ReadWriteOnce"},
						Size:                "5Gi",
						DefaultStorageClass: true,
					},
					{
						VolumeName:       "volume2",
						MountPath:        "/data/path2",
						AccessModes:      []string{"ReadWriteOnce"},
						Size:             "10Gi",
						StorageClassName: storageClass,
						Metadata: &kubernetesplatform.PodMetadata{
							Annotations: map[string]string{
								"annotation1": "a1",
							},
							Labels: map[string]string{
								"label1": "l1",
							},
						},
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
					},
				},
			},
			&k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "volume",
								MountPath: "/data/path",
							},
							{
								Name:      "volume2",
								MountPath: "/data/path2",
							},
						},
					},
				},
				Volumes: []k8score.Volume{
					{
						Name: "volume",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.ResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("5Gi"),
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "volume2",
						VolumeSource: k8score.VolumeSource{
							Ephemeral: &k8score.EphemeralVolumeSource{
								VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
									ObjectMeta: metav1.ObjectMeta{
										Annotations: map[string]string{
											"annotation1": "a1",
										},
										Labels: map[string]string{
											"label1": "l1",
										},
									},
									Spec: k8score.PersistentVolumeClaimSpec{
										AccessModes: []k8score.PersistentVolumeAccessMode{k8score.ReadWriteOnce},
										Resources: k8score.ResourceRequirements{
											Requests: k8score.ResourceList{
												k8score.ResourceStorage: k8sres.MustParse("10Gi"),
											},
										},
										StorageClassName: &storageClass,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extendPodSpecPatch(tt.podSpec, tt.k8sExecCfg, nil, nil)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, tt.podSpec)
		})
	}
}
