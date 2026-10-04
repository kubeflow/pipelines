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

package component

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuntimeArtifactSchema_RoundTrip(t *testing.T) {
	for _, title := range []string{
		"system.Artifact", "system.Dataset", "system.Model", "system.Metrics",
		"system.ClassificationMetrics", "system.SlicedClassificationMetrics",
		"system.HTML", "system.Markdown", "google.VertexModel", "example.CustomDataset",
	} {
		t.Run(title, func(t *testing.T) {
			declared := &pipelinespec.ArtifactTypeSchema{
				Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: title},
				SchemaVersion: "2.3.4",
			}
			nativeType, preservedTitle, err := inferArtifactType(declared)
			require.NoError(t, err)
			userMetadata := map[string]*structpb.Value{"owner": structpb.NewStringValue("user")}
			stored := preserveArtifactSchema(userMetadata, preservedTitle, declared.SchemaVersion)
			// Both ordinary reads and cached reads reconstruct from the native artifact.
			restored, visibleMetadata := RuntimeArtifactSchemaAndMetadata(nativeType, stored)
			require.Equal(t, title, restored.GetSchemaTitle())
			require.Equal(t, "2.3.4", restored.GetSchemaVersion())
			require.Equal(t, userMetadata, visibleMetadata)
			require.Len(t, userMetadata, 1, "registration must not mutate executor metadata")
			require.Contains(t, stored, artifactSchemaVersionMetadataKey, "restoration must not mutate the stored artifact")
		})
	}
}

func TestRuntimeArtifactSchema_ExistingNativeArtifacts(t *testing.T) {
	for title, nativeType := range artifactTypeSchemaToArtifactTypeMap {
		t.Run(title, func(t *testing.T) {
			schema, metadata := RuntimeArtifactSchemaAndMetadata(nativeType, nil)
			require.Equal(t, title, schema.GetSchemaTitle())
			require.Empty(t, schema.GetSchemaVersion(), "do not invent a version absent from stored data")
			require.Nil(t, metadata)
		})
	}
	// Unknown native enum values are not silently assigned a system schema.
	schema, _ := RuntimeArtifactSchemaAndMetadata(apiV2beta1.Artifact_TYPE_UNSPECIFIED, nil)
	require.Equal(t, "TYPE_UNSPECIFIED", schema.GetSchemaTitle())
}

func TestImportSpecToArtifact_PreservesSchema(t *testing.T) {
	for _, title := range []string{"system.Dataset", "example.CustomDataset"} {
		t.Run(title, func(t *testing.T) {
			launcher := &ImportLauncher{opts: LauncherV2Options{
				ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
					ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
						Value: &pipelinespec.ValueOrRuntimeParameter_Constant{Constant: structpb.NewStringValue("gs://bucket/dataset")},
					},
					TypeSchema: &pipelinespec.ArtifactTypeSchema{
						Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: title},
						SchemaVersion: "1.2.3",
					},
				},
			}}
			artifact, err := launcher.ImportSpecToArtifact()
			require.NoError(t, err)
			schema, metadata := RuntimeArtifactSchemaAndMetadata(artifact.Type, artifact.Metadata)
			require.Equal(t, title, schema.GetSchemaTitle())
			require.Equal(t, "1.2.3", schema.GetSchemaVersion())
			require.Nil(t, metadata)
		})
	}
}
