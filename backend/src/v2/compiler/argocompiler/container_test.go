// Copyright 2021-2024 The Kubeflow Authors
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

package argocompiler

import (
	"os"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
)

func TestAddContainerExecutorTemplate(t *testing.T) {
	tests := []struct {
		name                  string
		configMapName         string
		configMapKey          string
		mountPath             string
		expectedVolumeName    string
		expectedConfigMapName string
		expectedMountPath     string
	}{
		{
			name:                  "Test with valid settings",
			configMapName:         "kube-root-ca.crt",
			configMapKey:          "ca.crt",
			mountPath:             "/etc/ssl/custom",
			expectedVolumeName:    "ca-bundle",
			expectedConfigMapName: "kube-root-ca.crt",
			expectedMountPath:     "/etc/ssl/custom/ca.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EXECUTOR_CABUNDLE_CONFIGMAP_NAME", tt.configMapName)
			os.Setenv("EXECUTOR_CABUNDLE_CONFIGMAP_KEY", tt.configMapKey)
			os.Setenv("EXECUTOR_CABUNDLE_MOUNTPATH", tt.mountPath)

			c := &workflowCompiler{
				templates: make(map[string]*wfapi.Template),
				wf: &wfapi.Workflow{
					Spec: wfapi.WorkflowSpec{
						Templates: []wfapi.Template{},
					},
				},
			}

			c.addContainerExecutorTemplate("test-ref")
			assert.NotEmpty(t, "system-container-impl", "Template name should not be empty")

			executorTemplate, exists := c.templates["system-container-impl"]
			assert.True(t, exists, "Template should exist with the returned name")
			assert.NotNil(t, executorTemplate, "Executor template should not be nil")

			foundVolume := false
			for _, volume := range executorTemplate.Volumes {
				if volume.Name == tt.expectedVolumeName {
					foundVolume = true
					assert.Equal(t, tt.expectedConfigMapName, volume.VolumeSource.ConfigMap.Name, "ConfigMap name should match")
					break
				}
			}
			assert.True(t, foundVolume, "CA bundle volume should be included in the template")

			foundVolumeMount := false
			if executorTemplate.Container != nil {
				for _, mount := range executorTemplate.Container.VolumeMounts {
					if mount.Name == tt.expectedVolumeName && mount.MountPath == tt.expectedMountPath {
						foundVolumeMount = true
						break
					}
				}
			}
			assert.True(t, foundVolumeMount, "CA bundle volume mount should be included in the container")
		})
	}
	defer func() {
		os.Unsetenv("EXECUTOR_CABUNDLE_CONFIGMAP_NAME")
		os.Unsetenv("EXECUTOR_CABUNDLE_CONFIGMAP_KEY")
		os.Unsetenv("EXECUTOR_CABUNDLE_MOUNTPATH")
	}()

}

func Test_extendPodMetadata(t *testing.T) {
	tests := []struct {
		name                     string
		podMetadata              *wfapi.Metadata
		kubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig
		expected                 *wfapi.Metadata
	}{
		{
			"Valid - add pod labels and annotations",
			&wfapi.Metadata{},
			&kubernetesplatform.KubernetesExecutorConfig{
				PodMetadata: &kubernetesplatform.PodMetadata{
					Annotations: map[string]string{
						"run_id": "123456",
					},
					Labels: map[string]string{
						"kubeflow.com/kfp": "pipeline-node",
					},
				},
			},
			&wfapi.Metadata{
				Annotations: map[string]string{
					"run_id": "123456",
				},
				Labels: map[string]string{
					"kubeflow.com/kfp": "pipeline-node",
				},
			},
		},
		{
			"Valid - try overwrite default pod labels and annotations",
			&wfapi.Metadata{
				Annotations: map[string]string{
					"run_id": "654321",
				},
				Labels: map[string]string{
					"kubeflow.com/kfp": "default-node",
				},
			},
			&kubernetesplatform.KubernetesExecutorConfig{
				PodMetadata: &kubernetesplatform.PodMetadata{
					Annotations: map[string]string{
						"run_id": "123456",
					},
					Labels: map[string]string{
						"kubeflow.com/kfp": "pipeline-node",
					},
				},
			},
			&wfapi.Metadata{
				Annotations: map[string]string{
					"run_id": "654321",
				},
				Labels: map[string]string{
					"kubeflow.com/kfp": "default-node",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extendPodMetadata(tt.podMetadata, tt.kubernetesExecutorConfig)
			assert.Equal(t, tt.expected, tt.podMetadata)
		})
	}
}
