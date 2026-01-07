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

package driver

import (
	"testing"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestConvertArtifactsToArtifactList_MultipleMetrics tests that multiple metric
// artifacts are merged into a single RuntimeArtifact with combined metadata
func TestConvertArtifactsToArtifactList_MultipleMetrics(t *testing.T) {
	accuracy := 0.95
	precision := 0.87
	recall := 0.91

	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId:  "artifact-1",
			Name:        "accuracy",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &accuracy,
			Metadata: map[string]*structpb.Value{
				"accuracy": structpb.NewNumberValue(accuracy),
			},
		},
		{
			ArtifactId:  "artifact-2",
			Name:        "precision",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &precision,
			Metadata: map[string]*structpb.Value{
				"precision": structpb.NewNumberValue(precision),
			},
		},
		{
			ArtifactId:  "artifact-3",
			Name:        "recall",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &recall,
			Metadata: map[string]*structpb.Value{
				"recall": structpb.NewNumberValue(recall),
			},
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Should have ONE RuntimeArtifact with merged metadata
	assert.Equal(t, 1, len(artifactList.Artifacts), "Should merge multiple metrics into one RuntimeArtifact")

	runtimeArtifact := artifactList.Artifacts[0]
	assert.NotNil(t, runtimeArtifact.Metadata, "Merged artifact should have metadata")

	// Verify all metrics are in the metadata
	metadata := runtimeArtifact.Metadata.Fields
	assert.Equal(t, 3, len(metadata), "Metadata should contain all three metrics")

	// Verify each metric value
	assert.NotNil(t, metadata["accuracy"])
	assert.Equal(t, accuracy, metadata["accuracy"].GetNumberValue())

	assert.NotNil(t, metadata["precision"])
	assert.Equal(t, precision, metadata["precision"].GetNumberValue())

	assert.NotNil(t, metadata["recall"])
	assert.Equal(t, recall, metadata["recall"].GetNumberValue())
}

// TestConvertArtifactsToArtifactList_SingleMetric tests that a single metric
// artifact is converted without merging (normal behavior)
func TestConvertArtifactsToArtifactList_SingleMetric(t *testing.T) {
	accuracy := 0.95

	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId:  "artifact-1",
			Name:        "accuracy",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &accuracy,
			Metadata: map[string]*structpb.Value{
				"accuracy": structpb.NewNumberValue(accuracy),
			},
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Single metric should NOT be merged (normal conversion)
	assert.Equal(t, 1, len(artifactList.Artifacts), "Should have one RuntimeArtifact")

	runtimeArtifact := artifactList.Artifacts[0]
	assert.Equal(t, "accuracy", runtimeArtifact.Name)
	assert.Equal(t, "Metric", runtimeArtifact.Type.GetSchemaTitle())
}

// TestConvertArtifactsToArtifactList_NonMetrics tests that non-metric artifacts
// are not merged and follow normal conversion
func TestConvertArtifactsToArtifactList_NonMetrics(t *testing.T) {
	uri1 := "s3://bucket/dataset1"
	uri2 := "s3://bucket/dataset2"

	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId: "artifact-1",
			Name:       "dataset1",
			Type:       apiV2beta1.Artifact_Dataset,
			Uri:        &uri1,
		},
		{
			ArtifactId: "artifact-2",
			Name:       "dataset2",
			Type:       apiV2beta1.Artifact_Dataset,
			Uri:        &uri2,
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Non-metrics should NOT be merged - each gets its own RuntimeArtifact
	assert.Equal(t, 2, len(artifactList.Artifacts), "Non-metrics should not be merged")

	// Verify each artifact independently
	names := make(map[string]bool)
	for _, artifact := range artifactList.Artifacts {
		names[artifact.Name] = true
		assert.Equal(t, "Dataset", artifact.Type.GetSchemaTitle())
	}

	assert.True(t, names["dataset1"], "Should have dataset1")
	assert.True(t, names["dataset2"], "Should have dataset2")
}

// TestConvertArtifactsToArtifactList_MixedTypes tests that when artifacts
// contain mixed types (not all metrics), they are not merged
func TestConvertArtifactsToArtifactList_MixedTypes(t *testing.T) {
	accuracy := 0.95
	uri := "s3://bucket/model"

	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId:  "artifact-1",
			Name:        "accuracy",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &accuracy,
		},
		{
			ArtifactId: "artifact-2",
			Name:       "model",
			Type:       apiV2beta1.Artifact_Model,
			Uri:        &uri,
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Mixed types should NOT be merged
	assert.Equal(t, 2, len(artifactList.Artifacts), "Mixed types should not be merged")
}

// TestConvertArtifactsToArtifactList_EmptyList tests that empty artifact list
// is handled correctly
func TestConvertArtifactsToArtifactList_EmptyList(t *testing.T) {
	artifacts := []*apiV2beta1.Artifact{}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)
	assert.Equal(t, 0, len(artifactList.Artifacts), "Empty list should return empty ArtifactList")
}

// TestConvertArtifactsToArtifactList_MetricsWithURIAndMetadata tests that
// merged metrics preserve URI from first artifact
func TestConvertArtifactsToArtifactList_MetricsWithURIAndMetadata(t *testing.T) {
	accuracy := 0.95
	precision := 0.87
	uri := "s3://bucket/metrics.json"

	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId:  "artifact-1",
			Name:        "accuracy",
			Type:        apiV2beta1.Artifact_Metric,
			Uri:         &uri,
			NumberValue: &accuracy,
			Metadata: map[string]*structpb.Value{
				"accuracy": structpb.NewNumberValue(accuracy),
			},
		},
		{
			ArtifactId:  "artifact-2",
			Name:        "precision",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &precision,
			Metadata: map[string]*structpb.Value{
				"precision": structpb.NewNumberValue(precision),
			},
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Verify merged artifact has URI from first artifact
	assert.Equal(t, 1, len(artifactList.Artifacts))
	mergedArtifact := artifactList.Artifacts[0]
	assert.Equal(t, uri, mergedArtifact.Uri, "Should preserve URI from first artifact")

	// Verify metadata contains both metrics
	metadata := mergedArtifact.Metadata.Fields
	assert.Equal(t, 2, len(metadata))
	assert.NotNil(t, metadata["accuracy"])
	assert.NotNil(t, metadata["precision"])
}

// TestConvertArtifactsToArtifactList_MetricsNumberValueInMetadata tests that
// NumberValue is properly included in merged metadata for multiple metrics
func TestConvertArtifactsToArtifactList_MetricsNumberValueInMetadata(t *testing.T) {
	accuracy := 0.95
	precision := 0.87

	// Test with multiple metrics where one has no metadata field
	artifacts := []*apiV2beta1.Artifact{
		{
			ArtifactId:  "artifact-1",
			Name:        "accuracy",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &accuracy,
			// No metadata field - NumberValue should still be included in merged metadata
		},
		{
			ArtifactId:  "artifact-2",
			Name:        "precision",
			Type:        apiV2beta1.Artifact_Metric,
			NumberValue: &precision,
			Metadata: map[string]*structpb.Value{
				"precision": structpb.NewNumberValue(precision),
			},
		},
	}

	artifactList, err := convertArtifactsToArtifactList(artifacts, false)
	assert.NoError(t, err)
	assert.NotNil(t, artifactList)

	// Should merge into one RuntimeArtifact
	assert.Equal(t, 1, len(artifactList.Artifacts))
	runtimeArtifact := artifactList.Artifacts[0]
	assert.NotNil(t, runtimeArtifact.Metadata)

	// Verify both metrics are in metadata (including the one without explicit metadata field)
	metadata := runtimeArtifact.Metadata.Fields
	assert.Equal(t, 2, len(metadata), "Should have both metrics in metadata")
	assert.NotNil(t, metadata["accuracy"], "Should have accuracy from NumberValue")
	assert.NotNil(t, metadata["precision"], "Should have precision from metadata")
}
