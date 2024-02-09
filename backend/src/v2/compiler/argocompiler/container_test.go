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
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
)

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
