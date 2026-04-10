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
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddContainerExecutorTemplate(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
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

			c := &workflowCompiler{
				templates: make(map[string]*wfapi.Template),
				wf: &wfapi.Workflow{
					Spec: wfapi.WorkflowSpec{
						Templates: []wfapi.Template{},
					},
				},
				spec: &pipelinespec.PipelineSpec{
					PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
				},
				job: &pipelinespec.PipelineJob{},
			}

			c.addContainerExecutorTemplate(&pipelinespec.PipelineTaskSpec{ComponentRef: &pipelinespec.ComponentRef{Name: "comp-test-ref"}}, &kubernetesplatform.KubernetesExecutorConfig{})
			assert.NotEmpty(t, "system-container-impl", "Template name should not be empty")

			// The new design has a single executor template (no outer DAG wrapper).
			executorTemplate, exists := c.templates["system-container-impl"]
			assert.True(t, exists, "Template should exist with the returned name")
			assert.NotNil(t, executorTemplate, "Executor template should not be nil")

			// Driver init container must be first so it runs before the launcher copy.
			require.GreaterOrEqual(t, len(executorTemplate.InitContainers), 2, "must have kfp-driver and kfp-launcher init containers")
			assert.Equal(t, "kfp-driver", executorTemplate.InitContainers[0].Name)
			assert.Equal(t, "kfp-launcher", executorTemplate.InitContainers[1].Name)

			// Main container image must be parameterized.
			require.NotNil(t, executorTemplate.Container)
			assert.Equal(t, inputValue(paramImage), executorTemplate.Container.Image)

			// Driver outputs volume must be present.
			var foundDriverOutputsVol bool
			for _, vol := range executorTemplate.Volumes {
				if vol.Name == volumeNameDriverOutputs {
					foundDriverOutputsVol = true
					break
				}
			}
			assert.True(t, foundDriverOutputsVol, "kfp-driver-outputs volume must be present")
		})
	}

}

func TestContainerDriverTemplate_IncludesKFPPodNameEnv(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{},
	}

	name := c.addContainerDriverTemplate()
	require.Equal(t, "system-container-driver", name)

	tmpl, exists := c.templates[name]
	require.True(t, exists, "system-container-driver template should exist")
	require.NotNil(t, tmpl.Container, "template should have a container")

	var foundKFPPodName bool
	for _, env := range tmpl.Container.Env {
		if env.Name == "KFP_POD_NAME" {
			foundKFPPodName = true
			require.NotNil(t, env.ValueFrom, "KFP_POD_NAME should use ValueFrom")
			require.NotNil(t, env.ValueFrom.FieldRef, "KFP_POD_NAME should use fieldRef")
			assert.Equal(t, "metadata.name", env.ValueFrom.FieldRef.FieldPath,
				"KFP_POD_NAME must reference metadata.name via the downward API")
			break
		}
	}
	assert.True(t, foundKFPPodName,
		"system-container-driver template must include KFP_POD_NAME env var to avoid hostname truncation for long pod names")
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
					"{{inputs.parameters.pod-metadata-annotation-key}}": "{{inputs.parameters.pod-metadata-annotation-val}}",
				},
				Labels: map[string]string{
					"{{inputs.parameters.pod-metadata-label-key}}": "{{inputs.parameters.pod-metadata-label-val}}",
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
					"{{inputs.parameters.pod-metadata-annotation-key}}": "{{inputs.parameters.pod-metadata-annotation-val}}",
					"run_id": "654321",
				},
				Labels: map[string]string{
					"{{inputs.parameters.pod-metadata-label-key}}": "{{inputs.parameters.pod-metadata-label-val}}",
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
