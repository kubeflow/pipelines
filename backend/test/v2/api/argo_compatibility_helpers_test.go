// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindArgoCompatibilityPodName(t *testing.T) {
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "unrelated-pod",
				Annotations: map[string]string{argoNodeNameAnnotation: "artifact-pipeline.root.mantle"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "artifact-pod",
				Annotations: map[string]string{argoNodeNameAnnotation: "artifact-pipeline.write-artifact"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "artifact-pod-by-image"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{
				{Image: "docker.io/alpine:3.23"},
			}},
		},
	}

	if got := findArgoCompatibilityPodName(pods); got != "artifact-pod" {
		t.Fatalf("findArgoCompatibilityPodName() = %q, want %q", got, "artifact-pod")
	}
	if got := findArgoCompatibilityPodName(pods[:1]); got != "" {
		t.Fatalf("findArgoCompatibilityPodName() = %q, want no match", got)
	}
	if got := findArgoCompatibilityPodName(pods[2:]); got != "artifact-pod-by-image" {
		t.Fatalf("findArgoCompatibilityPodName() = %q, want %q", got, "artifact-pod-by-image")
	}
}

func TestArgoCompatibilityMetadataReady(t *testing.T) {
	executionOnly := &run_model.V2beta1PipelineTaskDetail{ExecutionID: "11"}
	artifactOnly := &run_model.V2beta1PipelineTaskDetail{
		Outputs: map[string]run_model.V2beta1ArtifactList{
			"dataset": {ArtifactIds: []string{"22"}},
		},
	}
	executionWithArtifact := &run_model.V2beta1PipelineTaskDetail{
		ExecutionID: "33",
		Outputs: map[string]run_model.V2beta1ArtifactList{
			"dataset": {ArtifactIds: []string{"44"}},
		},
	}

	tests := []struct {
		name        string
		run         *run_model.V2beta1Run
		expectReady bool
	}{
		{name: "missing run details", run: &run_model.V2beta1Run{}},
		{
			name: "execution ID only",
			run: &run_model.V2beta1Run{RunDetails: &run_model.V2beta1RunDetails{
				TaskDetails: []*run_model.V2beta1PipelineTaskDetail{executionOnly},
			}},
		},
		{
			name: "artifact ID only",
			run: &run_model.V2beta1Run{RunDetails: &run_model.V2beta1RunDetails{
				TaskDetails: []*run_model.V2beta1PipelineTaskDetail{artifactOnly},
			}},
		},
		{
			name: "execution and artifact IDs on separate tasks",
			run: &run_model.V2beta1Run{RunDetails: &run_model.V2beta1RunDetails{
				TaskDetails: []*run_model.V2beta1PipelineTaskDetail{executionOnly, artifactOnly},
			}},
		},
		{
			name: "execution and artifact IDs on the same task",
			run: &run_model.V2beta1Run{RunDetails: &run_model.V2beta1RunDetails{
				TaskDetails: []*run_model.V2beta1PipelineTaskDetail{executionWithArtifact},
			}},
			expectReady: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := argoCompatibilityMetadataReady(test.run); got != test.expectReady {
				t.Fatalf("argoCompatibilityMetadataReady() = %t, want %t", got, test.expectReady)
			}
		})
	}
}
