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
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
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

			templateName := c.addContainerExecutorTemplate(&pipelinespec.PipelineTaskSpec{ComponentRef: &pipelinespec.ComponentRef{Name: "comp-test-ref"}}, &kubernetesplatform.KubernetesExecutorConfig{})
			assert.NotEmpty(t, templateName, "Template name should not be empty")

			// The new design has a single executor template (no outer DAG wrapper).
			// Index by the returned name so the test catches compiler naming changes.
			executorTemplate, exists := c.templates[templateName]
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

func makeResolver(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := values[name]
		return v, ok
	}
}

func TestResolvePvcName(t *testing.T) {
	resolver := makeResolver(map[string]string{"my-param": "resolved-pvc"})

	tests := []struct {
		name     string
		pm       *kubernetesplatform.PvcMount
		resolver func(string) (string, bool)
		want     string
	}{
		{
			name: "deprecated Constant field",
			pm:   &kubernetesplatform.PvcMount{PvcReference: &kubernetesplatform.PvcMount_Constant{Constant: "static-pvc"}},
			want: "static-pvc",
		},
		{
			name: "deprecated ComponentInputParameter resolved",
			pm:   &kubernetesplatform.PvcMount{PvcReference: &kubernetesplatform.PvcMount_ComponentInputParameter{ComponentInputParameter: "my-param"}},
			want: "resolved-pvc",
		},
		{
			name: "deprecated ComponentInputParameter not resolvable",
			pm:   &kubernetesplatform.PvcMount{PvcReference: &kubernetesplatform.PvcMount_ComponentInputParameter{ComponentInputParameter: "unknown-param"}},
			want: "",
		},
		{
			name: "deprecated TaskOutputParameter skipped",
			pm: &kubernetesplatform.PvcMount{
				PvcReference: &kubernetesplatform.PvcMount_TaskOutputParameter{
					TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
						ProducerTask: "createpvc", OutputParameterKey: "name",
					},
				},
			},
			want: "",
		},
		{
			name: "new PvcNameParameter with ComponentInputParameter resolved",
			pm: &kubernetesplatform.PvcMount{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
						ComponentInputParameter: "my-param",
					},
				},
			},
			want: "resolved-pvc",
		},
		{
			name: "new PvcNameParameter with ComponentInputParameter not resolvable",
			pm: &kubernetesplatform.PvcMount{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
						ComponentInputParameter: "missing-param",
					},
				},
			},
			want: "",
		},
		{
			name: "new PvcNameParameter with RuntimeValue constant",
			pm: &kubernetesplatform.PvcMount{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
						RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
							Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
								Constant: structpb.NewStringValue("constant-pvc"),
							},
						},
					},
				},
			},
			want: "constant-pvc",
		},
		{
			name: "new PvcNameParameter with TaskOutputParameter skipped",
			pm: &kubernetesplatform.PvcMount{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter{
						TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
							ProducerTask: "createpvc", OutputParameterKey: "name",
						},
					},
				},
			},
			want: "",
		},
		{
			name: "nil resolver with ComponentInputParameter skipped",
			pm: &kubernetesplatform.PvcMount{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
						ComponentInputParameter: "my-param",
					},
				},
			},
			resolver: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resolver
			if tt.resolver != nil || tt.name == "nil resolver with ComponentInputParameter skipped" {
				r = tt.resolver
			}
			got := resolvePvcName(tt.pm, r)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestApplyStaticK8sConfig_PvcMount verifies that PVC volumes are correctly
// embedded in the executor template when PVC names can be resolved at compile time.
func TestApplyStaticK8sConfig_PvcMount(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	resolver := makeResolver(map[string]string{"pvc_name": "my-runtime-pvc"})

	tests := []struct {
		name          string
		cfg           *kubernetesplatform.KubernetesExecutorConfig
		wantVolumes   []string // expected volume names (PVC)
		wantMountPath string
	}{
		{
			name: "deprecated Constant PVC name",
			cfg: &kubernetesplatform.KubernetesExecutorConfig{
				PvcMount: []*kubernetesplatform.PvcMount{
					{
						PvcReference: &kubernetesplatform.PvcMount_Constant{Constant: "static-pvc"},
						MountPath:    "/data",
					},
				},
			},
			wantVolumes:   []string{"static-pvc"},
			wantMountPath: "/data",
		},
		{
			name: "ComponentInputParameter resolved from runtime config",
			cfg: &kubernetesplatform.KubernetesExecutorConfig{
				PvcMount: []*kubernetesplatform.PvcMount{
					{
						PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
							Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
								ComponentInputParameter: "pvc_name",
							},
						},
						MountPath: "/mnt/pvc",
					},
				},
			},
			wantVolumes:   []string{"my-runtime-pvc"},
			wantMountPath: "/mnt/pvc",
		},
		{
			name: "unresolvable ComponentInputParameter skipped",
			cfg: &kubernetesplatform.KubernetesExecutorConfig{
				PvcMount: []*kubernetesplatform.PvcMount{
					{
						PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
							Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
								ComponentInputParameter: "nonexistent",
							},
						},
						MountPath: "/mnt/pvc",
					},
				},
			},
			wantVolumes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &wfapi.Template{
				Container: &k8score.Container{},
			}
			applyStaticK8sConfig(tmpl, tt.cfg, resolver)

			// Collect only PVC volume names (not emptyDir, etc.).
			var pvcVolumeNames []string
			for _, v := range tmpl.Volumes {
				if v.PersistentVolumeClaim != nil {
					pvcVolumeNames = append(pvcVolumeNames, v.Name)
				}
			}

			assert.Equal(t, tt.wantVolumes, pvcVolumeNames)
			if len(tt.wantVolumes) > 0 {
				var foundMount bool
				for _, vm := range tmpl.Container.VolumeMounts {
					if vm.MountPath == tt.wantMountPath {
						foundMount = true
						break
					}
				}
				assert.True(t, foundMount, "expected volume mount at %s", tt.wantMountPath)
			}
		})
	}
}

// TestAddContainerExecutorTemplate_PvcMount_RuntimeConfig verifies that when
// a run's RuntimeConfig contains a PVC name parameter, the executor template
// produced by addContainerExecutorTemplate has PVC volumes embedded.
func TestAddContainerExecutorTemplate_PvcMount_RuntimeConfig(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{Templates: []wfapi.Template{}},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{
			RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"pvc_name": structpb.NewStringValue("test-pvc-123"),
				},
			},
		},
	}

	k8sCfg := &kubernetesplatform.KubernetesExecutorConfig{
		PvcMount: []*kubernetesplatform.PvcMount{
			{
				PvcNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
						ComponentInputParameter: "pvc_name",
					},
				},
				MountPath: "/data",
			},
		},
	}

	task := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-producer"},
	}

	tmplName := c.addContainerExecutorTemplate(task, k8sCfg)
	tmpl, ok := c.templates[tmplName]
	require.True(t, ok, "template %q should exist", tmplName)

	var pvcVol *k8score.Volume
	for i, v := range tmpl.Volumes {
		if v.PersistentVolumeClaim != nil {
			pvcVol = &tmpl.Volumes[i]
			break
		}
	}
	require.NotNil(t, pvcVol, "executor template must have a PVC volume when pvc_name is in RuntimeConfig")
	assert.Equal(t, "test-pvc-123", pvcVol.PersistentVolumeClaim.ClaimName)

	var foundMount bool
	for _, vm := range tmpl.Container.VolumeMounts {
		if vm.MountPath == "/data" && vm.Name == "test-pvc-123" {
			foundMount = true
			break
		}
	}
	assert.True(t, foundMount, "main container must have a volumeMount for the PVC at /data")
}
