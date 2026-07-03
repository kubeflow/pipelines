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

	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/proto"

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

func TestFindArgoCompatibilityMetadataIDs(t *testing.T) {
	containerExecution := &pb.Execution{
		Id:   proto.Int64(33),
		Type: proto.String(string(metadata.ContainerExecutionTypeName)),
	}
	dagExecution := &pb.Execution{
		Id:   proto.Int64(11),
		Type: proto.String(string(metadata.DagExecutionTypeName)),
	}
	tests := []struct {
		name              string
		executions        []*pb.Execution
		events            []*pb.Event
		expectExecutionID int64
		expectArtifactID  int64
	}{
		{
			name:       "container execution without an output artifact",
			executions: []*pb.Execution{containerExecution},
		},
		{
			name:       "output artifact linked to a DAG execution",
			executions: []*pb.Execution{dagExecution, containerExecution},
			events: []*pb.Event{{
				ExecutionId: proto.Int64(11),
				ArtifactId:  proto.Int64(22),
				Type:        pb.Event_OUTPUT.Enum(),
			}},
		},
		{
			name:       "declared output is not a produced artifact",
			executions: []*pb.Execution{containerExecution},
			events: []*pb.Event{{
				ExecutionId: proto.Int64(33),
				ArtifactId:  proto.Int64(44),
				Type:        pb.Event_DECLARED_OUTPUT.Enum(),
			}},
		},
		{
			name:       "output artifact linked to the container execution",
			executions: []*pb.Execution{dagExecution, containerExecution},
			events: []*pb.Event{{
				ExecutionId: proto.Int64(33),
				ArtifactId:  proto.Int64(44),
				Type:        pb.Event_OUTPUT.Enum(),
			}},
			expectExecutionID: 33,
			expectArtifactID:  44,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executionID, artifactID := findArgoCompatibilityMetadataIDs(test.executions, test.events)
			if executionID != test.expectExecutionID || artifactID != test.expectArtifactID {
				t.Fatalf(
					"findArgoCompatibilityMetadataIDs() = (%d, %d), want (%d, %d)",
					executionID,
					artifactID,
					test.expectExecutionID,
					test.expectArtifactID,
				)
			}
		})
	}
}
