/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta1

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8syaml "sigs.k8s.io/yaml"
)

func TestFromPipelineVersionModel_PipelineSpecOnly(t *testing.T) {
	pipeline := model.Pipeline{
		UUID:        "pipeline-123",
		Name:        "test-pipeline",
		Namespace:   "default",
		DisplayName: "Test Pipeline",
		Description: "A test pipeline",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:          "version-456",
		Name:          "test-version",
		DisplayName:   "Test Version",
		Description:   "A test version",
		PipelineId:    "pipeline-123",
		CodeSourceUrl: "https://github.com/test/pipeline",
		PipelineSpec: `pipelineInfo:
  name: test-pipeline
  displayName: Test Pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
sdkVersion: kfp-2.13.0`,
		PipelineSpecURI: "gs://bucket/pipeline.yaml",
	}

	result, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check ObjectMeta
	assert.Equal(t, "test-version", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, types.UID("version-456"), result.UID)
	assert.Equal(t, "pipeline-123", result.Labels["pipelines.kubeflow.org/pipeline-id"])

	// Check OwnerReferences
	require.Len(t, result.OwnerReferences, 1)
	ownerRef := result.OwnerReferences[0]
	assert.Equal(t, GroupVersion.String(), ownerRef.APIVersion)
	assert.Equal(t, "Pipeline", ownerRef.Kind)
	assert.Equal(t, types.UID("pipeline-123"), ownerRef.UID)
	assert.Equal(t, "test-pipeline", ownerRef.Name)

	// Check Spec
	assert.Equal(t, "Test Version", result.Spec.DisplayName)
	assert.Equal(t, "A test version", result.Spec.Description)
	assert.Equal(t, "test-pipeline", result.Spec.PipelineName)
	assert.Equal(t, "https://github.com/test/pipeline", result.Spec.CodeSourceURL)
	assert.Equal(t, "gs://bucket/pipeline.yaml", result.Spec.PipelineSpecURI)

	// Check PipelineSpec
	assert.NotNil(t, result.Spec.PipelineSpec.Value)
	pipelineSpecMap, ok := result.Spec.PipelineSpec.Value.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-pipeline", pipelineSpecMap["pipelineInfo"].(map[string]interface{})["name"])
	assert.Equal(t, "2.1.0", pipelineSpecMap["schemaVersion"])

	// Check PlatformSpec should be nil
	assert.Nil(t, result.Spec.PlatformSpec)
}

func TestFromPipelineVersionModel_PipelineAndPlatformSpecs(t *testing.T) {
	pipeline := model.Pipeline{
		UUID:        "pipeline-123",
		Name:        "test-pipeline",
		Namespace:   "default",
		DisplayName: "Test Pipeline",
		Description: "A test pipeline",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:          "version-456",
		Name:          "test-version",
		DisplayName:   "Test Version",
		Description:   "A test version",
		PipelineId:    "pipeline-123",
		CodeSourceUrl: "https://github.com/test/pipeline",
		PipelineSpec: `pipelineInfo:
  name: test-pipeline
  displayName: Test Pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
sdkVersion: kfp-2.13.0
---
platforms:
  kubernetes:
    pipelineConfig:
      workspace:
        kubernetes:
          pvcSpecPatch:
            accessModes:
            - ReadWriteOnce
        size: 10Gi`,
		PipelineSpecURI: "gs://bucket/pipeline.yaml",
	}

	result, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check PipelineSpec
	assert.NotNil(t, result.Spec.PipelineSpec.Value)
	pipelineSpecMap, ok := result.Spec.PipelineSpec.Value.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-pipeline", pipelineSpecMap["pipelineInfo"].(map[string]interface{})["name"])
	assert.Equal(t, "2.1.0", pipelineSpecMap["schemaVersion"])

	// Check PlatformSpec
	assert.NotNil(t, result.Spec.PlatformSpec)
	platformSpecMap, ok := result.Spec.PlatformSpec.Value.(map[string]interface{})
	require.True(t, ok)

	platforms, ok := platformSpecMap["platforms"].(map[string]interface{})
	require.True(t, ok)

	kubernetes, ok := platforms["kubernetes"].(map[string]interface{})
	require.True(t, ok)

	pipelineConfig, ok := kubernetes["pipelineConfig"].(map[string]interface{})
	require.True(t, ok)

	workspace, ok := pipelineConfig["workspace"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "10Gi", workspace["size"])

	kubernetesConfig, ok := workspace["kubernetes"].(map[string]interface{})
	require.True(t, ok)

	pvcSpecPatch, ok := kubernetesConfig["pvcSpecPatch"].(map[string]interface{})
	require.True(t, ok)

	accessModes, ok := pvcSpecPatch["accessModes"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "ReadWriteOnce", accessModes[0])
}

func TestFromPipelineVersionModel_MultiplePipelineSpecs(t *testing.T) {
	pipeline := model.Pipeline{
		UUID:      "pipeline-123",
		Name:      "test-pipeline",
		Namespace: "default",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:       "version-456",
		Name:       "test-version",
		PipelineId: "pipeline-123",
		PipelineSpec: `pipelineInfo:
  name: test-pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
---
pipelineInfo:
  name: another-pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"`,
	}

	_, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple pipeline specs provided")
}

func TestFromPipelineVersionModel_MultiplePlatformSpecs(t *testing.T) {
	pipeline := model.Pipeline{
		UUID:      "pipeline-123",
		Name:      "test-pipeline",
		Namespace: "default",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:       "version-456",
		Name:       "test-version",
		PipelineId: "pipeline-123",
		PipelineSpec: `pipelineInfo:
  name: test-pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
---
platforms:
  kubernetes:
    pipelineConfig:
      workspace:
        size: 10Gi
---
platforms:
  kubernetes:
    pipelineConfig:
      workspace:
        size: 20Gi`,
	}

	_, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple platform specs provided")
}

func TestFromPipelineVersionModel_NoPipelineSpec(t *testing.T) {
	pipeline := model.Pipeline{
		UUID:      "pipeline-123",
		Name:      "test-pipeline",
		Namespace: "default",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:       "version-456",
		Name:       "test-version",
		PipelineId: "pipeline-123",
		PipelineSpec: `platforms:
  kubernetes:
    pipelineConfig:
      workspace:
        size: 10Gi`,
	}

	_, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pipeline spec is provided")
}

func TestToModel_PipelineSpecOnly(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        "pipeline-123",
					Name:       "test-pipeline",
				},
			},
		},
		Spec: PipelineVersionSpec{
			DisplayName:     "Test Version",
			Description:     "A test version",
			PipelineName:    "test-pipeline",
			CodeSourceURL:   "https://github.com/test/pipeline",
			PipelineSpecURI: "gs://bucket/pipeline.yaml",
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name":        "test-pipeline",
						"displayName": "Test Pipeline",
					},
					"root": map[string]interface{}{
						"dag": map[string]interface{}{
							"tasks": map[string]interface{}{},
						},
					},
					"schemaVersion": "2.1.0",
					"sdkVersion":    "kfp-2.13.0",
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check basic fields
	assert.Equal(t, "version-456", result.UUID)
	assert.Equal(t, "test-version", result.Name)
	assert.Equal(t, "Test Version", result.DisplayName)
	assert.Equal(t, "A test version", result.Description)
	assert.Equal(t, "pipeline-123", result.PipelineId)
	assert.Equal(t, "https://github.com/test/pipeline", result.CodeSourceUrl)
	assert.Equal(t, "gs://bucket/pipeline.yaml", result.PipelineSpecURI)

	// Check PipelineSpec contains only pipeline spec (no platform spec)
	assert.Contains(t, result.PipelineSpec, "pipelineInfo:")
	assert.Contains(t, result.PipelineSpec, "name: test-pipeline")
	assert.Contains(t, result.PipelineSpec, "schemaVersion: 2.1.0")
	assert.NotContains(t, result.PipelineSpec, "platforms:")
	assert.NotContains(t, result.PipelineSpec, "---")
}

func TestToModel_PipelineAndPlatformSpecs(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        "pipeline-123",
					Name:       "test-pipeline",
				},
			},
		},
		Spec: PipelineVersionSpec{
			DisplayName:     "Test Version",
			Description:     "A test version",
			PipelineName:    "test-pipeline",
			CodeSourceURL:   "https://github.com/test/pipeline",
			PipelineSpecURI: "gs://bucket/pipeline.yaml",
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name":        "test-pipeline",
						"displayName": "Test Pipeline",
					},
					"root": map[string]interface{}{
						"dag": map[string]interface{}{
							"tasks": map[string]interface{}{},
						},
					},
					"schemaVersion": "2.1.0",
					"sdkVersion":    "kfp-2.13.0",
				},
			},
			PlatformSpec: &IRSpec{
				Value: map[string]interface{}{
					"platforms": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"pipelineConfig": map[string]interface{}{
								"workspace": map[string]interface{}{
									"kubernetes": map[string]interface{}{
										"pvcSpecPatch": map[string]interface{}{
											"accessModes": []interface{}{
												"ReadWriteOnce",
											},
										},
									},
									"size": "10Gi",
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check basic fields
	assert.Equal(t, "version-456", result.UUID)
	assert.Equal(t, "test-version", result.Name)
	assert.Equal(t, "Test Version", result.DisplayName)
	assert.Equal(t, "A test version", result.Description)
	assert.Equal(t, "pipeline-123", result.PipelineId)
	assert.Equal(t, "https://github.com/test/pipeline", result.CodeSourceUrl)
	assert.Equal(t, "gs://bucket/pipeline.yaml", result.PipelineSpecURI)

	// Check PipelineSpec contains both pipeline and platform specs
	assert.Contains(t, result.PipelineSpec, "pipelineInfo:")
	assert.Contains(t, result.PipelineSpec, "name: test-pipeline")
	assert.Contains(t, result.PipelineSpec, "schemaVersion: 2.1.0")
	assert.Contains(t, result.PipelineSpec, "platforms:")
	assert.Contains(t, result.PipelineSpec, "kubernetes:")
	assert.Contains(t, result.PipelineSpec, "size: 10Gi")
	assert.Contains(t, result.PipelineSpec, "accessModes:")
	assert.Contains(t, result.PipelineSpec, "ReadWriteOnce")
	assert.Contains(t, result.PipelineSpec, "---")
}

func TestToModel_WithStatus(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        "pipeline-123",
					Name:       "test-pipeline",
				},
			},
		},
		Spec: PipelineVersionSpec{
			DisplayName: "Test Version",
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name": "test-pipeline",
					},
					"schemaVersion": "2.1.0",
				},
			},
		},
		Status: PipelineVersionStatus{
			Conditions: []SimplifiedCondition{
				{
					Type:    "PipelineVersionStatus",
					Status:  "True",
					Reason:  "READY",
					Message: "Pipeline version is ready",
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check status is properly converted
	assert.Equal(t, model.PipelineVersionStatus("READY"), result.Status)
}

func TestToModel_DisplayNameFallback(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        "pipeline-123",
					Name:       "test-pipeline",
				},
			},
		},
		Spec: PipelineVersionSpec{
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name": "test-pipeline",
					},
					"schemaVersion": "2.1.0",
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that display name falls back to name when not set
	assert.Equal(t, "test-version", result.DisplayName)
}

func TestToModel_InvalidPipelineSpec(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
		},
		Spec: PipelineVersionSpec{
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name": "test-pipeline",
					},
					"schemaVersion": "2.1.0",
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Test with a value that can't be marshaled to YAML
	pipelineVersion.Spec.PipelineSpec.Value = "-invalid-"
	_, _ = pipelineVersion.ToModel()
}

func TestToModel_InvalidPlatformSpec(t *testing.T) {
	pipelineVersion := &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-version",
			Namespace: "default",
			UID:       "version-456",
		},
		Spec: PipelineVersionSpec{
			PipelineSpec: IRSpec{
				Value: map[string]interface{}{
					"pipelineInfo": map[string]interface{}{
						"name": "test-pipeline",
					},
					"schemaVersion": "2.1.0",
				},
			},
			PlatformSpec: &IRSpec{
				Value: map[string]interface{}{
					"platforms": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"pipelineConfig": map[string]interface{}{
								"workspace": map[string]interface{}{
									"size": "10Gi",
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := pipelineVersion.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Test with a value that can't be marshaled to YAML
	pipelineVersion.Spec.PlatformSpec.Value = "-invalid-"
	_, _ = pipelineVersion.ToModel()
}

func TestFromPipelineVersionModel_RoundTrip(t *testing.T) {
	// This test uses the exact example from the Jira issue
	pipelineSpecYAML := `# PIPELINE DEFINITION
# Name: hello-world
components:
  comp-hello-world:
    executorLabel: exec-hello-world
    inputDefinitions:
      parameters:
        msg:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-hello-world:
      container:
        args:
        - --executor_input
        - '{{$}}'
        - --function_to_execute
        - hello_world
        command:
        - sh
        - -c
        - "\nif ! [ -x \"$(command -v pip)\" ]; then\n    python3 -m ensurepip ||\
          \ python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1\
          \ python3 -m pip install --quiet --no-warn-script-location 'kfp==2.13.0'\
          \ '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<\"3.9\"' && \"\
          $0\" \"$@\"\n"
        - sh
        - -ec
        - 'program_path=$(mktemp -d)


          printf "%s" "$0" > "$program_path/ephemeral_component.py"

          _KFP_RUNTIME=true python3 -m kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"

          '
        - "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import\
          \ *\n\ndef hello_world(msg: str):\n    print(msg)\n\n"
        image: python:3.9
pipelineInfo:
  name: hello-world
root:
  dag:
    tasks:
      hello-world:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-hello-world
        inputs:
          parameters:
            msg:
              runtimeValue:
                constant: Hello, World!
        taskInfo:
          name: hello-world
schemaVersion: 2.1.0
sdkVersion: kfp-2.13.0
---
platforms:
  kubernetes:
    pipelineConfig:
      workspace:
        kubernetes:
          pvcSpecPatch:
            accessModes:
            - ReadWriteOnce
        size: 10Gi`

	pipeline := model.Pipeline{
		UUID:      "pipeline-123",
		Name:      "hello-world",
		Namespace: "default",
	}

	pipelineVersion := model.PipelineVersion{
		UUID:         "version-456",
		Name:         "hello-world-v1",
		DisplayName:  "Hello World Pipeline v1",
		Description:  "A simple hello world pipeline with workspace configuration",
		PipelineSpec: pipelineSpecYAML,
		PipelineId:   "pipeline-123",
	}

	// Convert from model to CRD
	result, err := FromPipelineVersionModel(pipeline, pipelineVersion)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Convert back to model
	roundTripModel, err := result.ToModel()
	require.NoError(t, err)
	assert.NotNil(t, roundTripModel)

	// Unmarshal both YAMLs into objects for semantic comparison
	originalPipelineSpecSplit := strings.Split(pipelineSpecYAML, "\n---\n")

	require.Len(t, originalPipelineSpecSplit, 2)
	var originalPipelineSpec map[string]interface{}
	err = k8syaml.Unmarshal([]byte(originalPipelineSpecSplit[0]), &originalPipelineSpec)
	require.NoError(t, err)

	var originalPlatformSpec map[string]interface{}
	err = k8syaml.Unmarshal([]byte(originalPipelineSpecSplit[1]), &originalPlatformSpec)
	require.NoError(t, err)

	roundTripPipelineSpecSplit := strings.Split(roundTripModel.PipelineSpec, "\n---\n")
	require.Len(t, roundTripPipelineSpecSplit, 2)
	var roundTripPipelineSpec map[string]interface{}
	err = k8syaml.Unmarshal([]byte(roundTripPipelineSpecSplit[0]), &roundTripPipelineSpec)
	require.NoError(t, err)

	var roundTripPlatformSpec map[string]interface{}
	err = k8syaml.Unmarshal([]byte(roundTripPipelineSpecSplit[1]), &roundTripPlatformSpec)
	require.NoError(t, err)

	// Verify we have the same number of documents
	assert.Equal(t, len(originalPipelineSpec), len(roundTripPipelineSpec),
		"Should have the same number of YAML documents")

	// Compare all documents using reflect.DeepEqual
	for i := range originalPipelineSpec {
		assert.True(t, reflect.DeepEqual(originalPipelineSpec[i], roundTripPipelineSpec[i]),
			"YAML document %d should be deeply equal after round-trip", i)
	}

	// Verify basic metadata is preserved
	assert.Equal(t, "version-456", roundTripModel.UUID)
	assert.Equal(t, "hello-world-v1", roundTripModel.Name)
	assert.Equal(t, "Hello World Pipeline v1", roundTripModel.DisplayName)
	assert.Equal(t, "A simple hello world pipeline with workspace configuration", roundTripModel.Description)
	assert.Equal(t, "pipeline-123", roundTripModel.PipelineId)
}
