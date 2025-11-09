// Copyright 2025 The Kubeflow Authors
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

package server

import (
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
)

// TestToApiTask_MetricsGrouping tests that multiple metric artifacts with the same
// ArtifactKey, Type, and Producer are consolidated into a single IOArtifact
func TestToApiTask_MetricsGrouping(t *testing.T) {
	// Create multiple metric artifacts with same key
	accuracy := 0.95
	precision := 0.87
	recall := 0.91

	accuracyMetadata := model.JSONData{
		"accuracy": accuracy,
	}
	precisionMetadata := model.JSONData{
		"precision": precision,
	}
	recallMetadata := model.JSONData{
		"recall": recall,
	}

	modelTask := &model.Task{
		UUID:      "task-123",
		RunUUID:   "run-456",
		Name:      "metrics-task",
		Namespace: "ns1",
		OutputArtifactsHydrated: []model.TaskArtifactHydrated{
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-1",
					Name:        "accuracy",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &accuracy,
					Metadata:    accuracyMetadata,
				},
				Producer: &model.IOProducer{
					TaskName: "metrics-task",
				},
			},
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-2",
					Name:        "precision",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &precision,
					Metadata:    precisionMetadata,
				},
				Producer: &model.IOProducer{
					TaskName: "metrics-task",
				},
			},
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-3",
					Name:        "recall",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &recall,
					Metadata:    recallMetadata,
				},
				Producer: &model.IOProducer{
					TaskName: "metrics-task",
				},
			},
		},
	}

	apiTask, err := toAPITask(modelTask, nil)
	assert.NoError(t, err)
	assert.NotNil(t, apiTask)

	// Verify we have ONE IOArtifact for all three metrics
	assert.Equal(t, 1, len(apiTask.Outputs.Artifacts), "Should have exactly one IOArtifact for grouped metrics")

	ioArtifact := apiTask.Outputs.Artifacts[0]
	assert.Equal(t, "metrics", ioArtifact.ArtifactKey)
	assert.Equal(t, apiv2beta1.IOType_OUTPUT, ioArtifact.Type)

	// Verify all three metric artifacts are in the same IOArtifact
	assert.Equal(t, 3, len(ioArtifact.Artifacts), "Should have all three metric artifacts in one IOArtifact")

	// Verify each artifact is present
	artifactNames := make(map[string]bool)
	for _, artifact := range ioArtifact.Artifacts {
		artifactNames[artifact.Name] = true
		assert.Equal(t, apiv2beta1.Artifact_Metric, artifact.Type)
	}

	assert.True(t, artifactNames["accuracy"], "Should contain accuracy metric")
	assert.True(t, artifactNames["precision"], "Should contain precision metric")
	assert.True(t, artifactNames["recall"], "Should contain recall metric")
}

// TestToApiTask_MetricsGrouping_DifferentProducers tests that metrics from different
// producers are NOT grouped together
func TestToApiTask_MetricsGrouping_DifferentProducers(t *testing.T) {
	accuracy := 0.95
	precision := 0.87

	modelTask := &model.Task{
		UUID:      "task-123",
		RunUUID:   "run-456",
		Name:      "metrics-task",
		Namespace: "ns1",
		OutputArtifactsHydrated: []model.TaskArtifactHydrated{
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-1",
					Name:        "accuracy",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &accuracy,
				},
				Producer: &model.IOProducer{
					TaskName: "task-a",
				},
			},
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-2",
					Name:        "precision",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &precision,
				},
				Producer: &model.IOProducer{
					TaskName: "task-b",
				},
			},
		},
	}

	apiTask, err := toAPITask(modelTask, nil)
	assert.NoError(t, err)
	assert.NotNil(t, apiTask)

	// Verify we have TWO IOArtifacts (different producers)
	assert.Equal(t, 2, len(apiTask.Outputs.Artifacts), "Should have two IOArtifacts for different producers")

	// Each IOArtifact should have one artifact
	for _, ioArtifact := range apiTask.Outputs.Artifacts {
		assert.Equal(t, 1, len(ioArtifact.Artifacts), "Each IOArtifact should have one artifact")
	}
}

// TestToApiTask_MetricsGrouping_WithIterations tests that metrics from different
// loop iterations are NOT grouped together
func TestToApiTask_MetricsGrouping_WithIterations(t *testing.T) {
	accuracy1 := 0.95
	accuracy2 := 0.87
	iter0 := int64(0)
	iter1 := int64(1)

	modelTask := &model.Task{
		UUID:      "task-123",
		RunUUID:   "run-456",
		Name:      "loop-task",
		Namespace: "ns1",
		OutputArtifactsHydrated: []model.TaskArtifactHydrated{
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_ITERATOR_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-1",
					Name:        "accuracy",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &accuracy1,
				},
				Producer: &model.IOProducer{
					TaskName:  "loop-task",
					Iteration: &iter0,
				},
			},
			{
				Key:  "metrics",
				Type: apiv2beta1.IOType_ITERATOR_OUTPUT,
				Value: &model.Artifact{
					UUID:        "artifact-2",
					Name:        "accuracy",
					Type:        model.ArtifactType(apiv2beta1.Artifact_Metric),
					NumberValue: &accuracy2,
				},
				Producer: &model.IOProducer{
					TaskName:  "loop-task",
					Iteration: &iter1,
				},
			},
		},
	}

	apiTask, err := toAPITask(modelTask, nil)
	assert.NoError(t, err)
	assert.NotNil(t, apiTask)

	// Verify we have TWO IOArtifacts (different iterations)
	assert.Equal(t, 2, len(apiTask.Outputs.Artifacts), "Should have two IOArtifacts for different iterations")

	// Verify iterations are correct
	iterations := make(map[int64]bool)
	for _, ioArtifact := range apiTask.Outputs.Artifacts {
		assert.NotNil(t, ioArtifact.Producer)
		assert.NotNil(t, ioArtifact.Producer.Iteration)
		iterations[*ioArtifact.Producer.Iteration] = true
	}

	assert.True(t, iterations[0], "Should have iteration 0")
	assert.True(t, iterations[1], "Should have iteration 1")
}

// TestToApiTask_NonMetrics tests that non-metric artifacts are not grouped
func TestToApiTask_NonMetrics(t *testing.T) {
	uri1 := "s3://bucket/dataset1"
	uri2 := "s3://bucket/dataset2"

	modelTask := &model.Task{
		UUID:      "task-123",
		RunUUID:   "run-456",
		Name:      "data-task",
		Namespace: "ns1",
		OutputArtifactsHydrated: []model.TaskArtifactHydrated{
			{
				Key:  "datasets",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID: "artifact-1",
					Name: "dataset1",
					Type: model.ArtifactType(apiv2beta1.Artifact_Dataset),
					URI:  &uri1,
				},
				Producer: &model.IOProducer{
					TaskName: "data-task",
				},
			},
			{
				Key:  "datasets",
				Type: apiv2beta1.IOType_OUTPUT,
				Value: &model.Artifact{
					UUID: "artifact-2",
					Name: "dataset2",
					Type: model.ArtifactType(apiv2beta1.Artifact_Dataset),
					URI:  &uri2,
				},
				Producer: &model.IOProducer{
					TaskName: "data-task",
				},
			},
		},
	}

	apiTask, err := toAPITask(modelTask, nil)
	assert.NoError(t, err)
	assert.NotNil(t, apiTask)

	// Verify we have TWO IOArtifacts (non-metrics are not grouped)
	assert.Equal(t, 2, len(apiTask.Outputs.Artifacts), "Non-metric artifacts should not be grouped")

	// Each IOArtifact should have one artifact
	for _, ioArtifact := range apiTask.Outputs.Artifacts {
		assert.Equal(t, 1, len(ioArtifact.Artifacts), "Each IOArtifact should have one artifact")
	}
}
