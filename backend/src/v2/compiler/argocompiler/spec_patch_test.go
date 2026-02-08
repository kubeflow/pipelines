// Copyright 2025 The Kubeflow Authors
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
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyWorkflowSpecPatch(t *testing.T) {
	tests := []struct {
		name         string
		patchJSON    string
		expectError  bool
		validateFunc func(t *testing.T, wf *wfapi.Workflow)
	}{
		{
			name:        "empty patch should skip patching",
			patchJSON:   "{}",
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				// Should remain unchanged
				assert.Equal(t, "original-sa", wf.Spec.ServiceAccountName)
			},
		},
		{
			name:        "empty string should skip patching",
			patchJSON:   "",
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				// Should remain unchanged
				assert.Equal(t, "original-sa", wf.Spec.ServiceAccountName)
			},
		},
		{
			name:        "patch service account name",
			patchJSON:   `{"serviceAccountName": "custom-sa"}`,
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				assert.Equal(t, "custom-sa", wf.Spec.ServiceAccountName)
			},
		},
		{
			name:        "patch node selector",
			patchJSON:   `{"nodeSelector": {"node-type": "gpu", "zone": "us-west1"}}`,
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				assert.Equal(t, "gpu", wf.Spec.NodeSelector["node-type"])
				assert.Equal(t, "us-west1", wf.Spec.NodeSelector["zone"])
			},
		},
		{
			name: "patch multiple fields",
			patchJSON: `{
				"serviceAccountName": "gpu-runner",
				"nodeSelector": {"accelerator": "nvidia-tesla-k80"},
				"tolerations": [
					{"key": "nvidia.com/gpu", "operator": "Exists", "effect": "NoSchedule"}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				assert.Equal(t, "gpu-runner", wf.Spec.ServiceAccountName)
				assert.Equal(t, "nvidia-tesla-k80", wf.Spec.NodeSelector["accelerator"])
				assert.Len(t, wf.Spec.Tolerations, 1)
				assert.Equal(t, "nvidia.com/gpu", wf.Spec.Tolerations[0].Key)
				assert.Equal(t, corev1.TolerationOperator("Exists"), wf.Spec.Tolerations[0].Operator)
				assert.Equal(t, corev1.TaintEffectNoSchedule, wf.Spec.Tolerations[0].Effect)
			},
		},
		{
			name: "patch pod metadata",
			patchJSON: `{
				"podMetadata": {
					"labels": {"env": "prod", "team": "ml"},
					"annotations": {"monitoring": "enabled"}
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				assert.NotNil(t, wf.Spec.PodMetadata)
				assert.Equal(t, "prod", wf.Spec.PodMetadata.Labels["env"])
				assert.Equal(t, "ml", wf.Spec.PodMetadata.Labels["team"])
				assert.Equal(t, "enabled", wf.Spec.PodMetadata.Annotations["monitoring"])
			},
		},
		{
			name: "merge with existing pod metadata",
			patchJSON: `{
				"podMetadata": {
					"labels": {"new-label": "new-value"}
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				assert.NotNil(t, wf.Spec.PodMetadata)
				// Should merge with existing labels
				assert.Equal(t, "existing-value", wf.Spec.PodMetadata.Labels["existing-label"])
				assert.Equal(t, "new-value", wf.Spec.PodMetadata.Labels["new-label"])
			},
		},
		{
			name:        "invalid JSON should return error",
			patchJSON:   `{"invalid": json}`,
			expectError: true,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				// Should remain unchanged
				assert.Equal(t, "original-sa", wf.Spec.ServiceAccountName)
			},
		},
		{
			name:        "malformed JSON should return error",
			patchJSON:   `{incomplete`,
			expectError: true,
			validateFunc: func(t *testing.T, wf *wfapi.Workflow) {
				// Should remain unchanged
				assert.Equal(t, "original-sa", wf.Spec.ServiceAccountName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test workflow with some initial values
			wf := &wfapi.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-workflow",
				},
				Spec: wfapi.WorkflowSpec{
					ServiceAccountName: "original-sa",
					PodMetadata: &wfapi.Metadata{
						Labels: map[string]string{
							"existing-label": "existing-value",
						},
					},
				},
			}

			// Create a workflow compiler with the test workflow
			compiler := &workflowCompiler{
				wf: wf,
			}

			// Apply the patch
			err := compiler.ApplyWorkflowSpecPatch(tt.patchJSON)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Run validation function
			if tt.validateFunc != nil {
				tt.validateFunc(t, wf)
			}
		})
	}
}

func TestApplyWorkflowSpecPatch_NilWorkflow(t *testing.T) {
	compiler := &workflowCompiler{
		wf: nil,
	}

	err := compiler.ApplyWorkflowSpecPatch(`{"serviceAccountName": "test"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow is nil")
}

func TestApplyWorkflowSpecPatch_ComplexPatch(t *testing.T) {
	// Test a more complex patch with various field types
	wf := &wfapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "complex-test-workflow",
		},
		Spec: wfapi.WorkflowSpec{
			ServiceAccountName: "default",
		},
	}

	compiler := &workflowCompiler{
		wf: wf,
	}
	// Match the actual compiler behavior: only SeccompProfile at workflow level
	wf.Spec.SecurityContext = &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	complexPatch := `{
		"serviceAccountName": "complex-sa",
		"activeDeadlineSeconds": 3600,
		"parallelism": 5,
		"nodeSelector": {
			"kubernetes.io/arch": "amd64",
			"node-pool": "compute"
		},
		"tolerations": [
			{
				"key": "dedicated",
				"operator": "Equal",
				"value": "ml-workload",
				"effect": "NoSchedule"
			}
		],
		"securityContext": {
			"runAsUser": 1000,
			"runAsGroup": 1000,
			"fsGroup": 1000
		},
		"hostNetwork": false,
		"dnsPolicy": "ClusterFirst"
	}`

	err := compiler.ApplyWorkflowSpecPatch(complexPatch)
	assert.NoError(t, err)

	// Validate all patched fields
	assert.Equal(t, "complex-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, int64(3600), *wf.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int64(5), *wf.Spec.Parallelism)
	assert.Equal(t, "amd64", wf.Spec.NodeSelector["kubernetes.io/arch"])
	assert.Equal(t, "compute", wf.Spec.NodeSelector["node-pool"])
	assert.Len(t, wf.Spec.Tolerations, 1)
	assert.Equal(t, "dedicated", wf.Spec.Tolerations[0].Key)
	assert.Equal(t, corev1.TolerationOperator("Equal"), wf.Spec.Tolerations[0].Operator)
	assert.Equal(t, "ml-workload", wf.Spec.Tolerations[0].Value)
	assert.Equal(t, corev1.TaintEffectNoSchedule, wf.Spec.Tolerations[0].Effect)
	assert.NotNil(t, wf.Spec.SecurityContext)
	assert.Equal(t, int64(1000), *wf.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, int64(1000), *wf.Spec.SecurityContext.RunAsGroup)
	assert.Equal(t, int64(1000), *wf.Spec.SecurityContext.FSGroup)
	assert.NotNil(t, wf.Spec.SecurityContext.SeccompProfile)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, wf.Spec.SecurityContext.SeccompProfile.Type)
	assert.Equal(t, false, *wf.Spec.HostNetwork)
	assert.Equal(t, corev1.DNSClusterFirst, *wf.Spec.DNSPolicy)
}
