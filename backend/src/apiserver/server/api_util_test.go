// Copyright 2018 The Kubeflow Authors
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
	"os"
	"strings"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

func TestValidateExperimentResourceReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	assert.Nil(t, ValidateExperimentResourceReference(manager, validReference))
}

func TestValidateExperimentResourceReference_MoreThanOneRef(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123",
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "456",
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "more resource references than expected")
}

func TestValidateExperimentResourceReference_UnexpectedType(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123",
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestValidateExperimentResourceReference_EmptyID(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT,
			},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Resource ID is empty")
}

func TestValidateExperimentResourceReference_UnexpectedRelationship(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*apiv1beta1.ResourceReference{
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123",
			},
			Relationship: apiv1beta1.Relationship_CREATOR,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected relationship for the experiment")
}

func TestValidateExperimentResourceReference_ExperimentNotExist(t *testing.T) {
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clients.Close()
	err := ValidateExperimentResourceReference(manager, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		WorkflowManifest: testWorkflow.ToStringForStore(),
	}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineID(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId:       DefaultFakeUUID,
		WorkflowManifest: testWorkflow.ToStringForStore(),
	}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{WorkflowManifest: "I am an invalid manifest"}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid argo workflow format")
}

func TestValidatePipelineSpecAndResourceReferences_NilPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version)")
}

func TestValidatePipelineSpecAndResourceReferences_EmptyPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version)")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineId(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{PipelineId: "not-found"}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineId failed")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineVersionId(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, referencesOfInvalidPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed")
}

func TestValidatePipelineSpecAndResourceReferences_PipelineIdNotParentOfPipelineVersionId(t *testing.T) {
	clients := initWithExperimentsAndTwoPipelineVersions(t)
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId: NonDefaultFakeUUID,
	}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "pipeline ID should be parent of pipeline version")
}

func TestValidatePipelineSpecAndResourceReferences_ParameterTooLongWithPipelineId(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	var params []*apiv1beta1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &apiv1beta1.Parameter{Name: "param2", Value: "world"})
	}
	spec := &apiv1beta1.PipelineSpec{PipelineId: DefaultFakeUUID, Parameters: params}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestValidatePipelineSpecAndResourceReferences_ParameterTooLongWithWorkflowManifest(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	var params []*apiv1beta1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &apiv1beta1.Parameter{Name: "param2", Value: "world"})
	}
	spec := &apiv1beta1.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore(), Parameters: params}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestValidatePipelineSpecAndResourceReferences_ValidPipelineIdAndPipelineVersionId(t *testing.T) {
	clients, manager, _, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId: DefaultFakeUUID,
	}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.Nil(t, err)
}

func TestValidatePipelineSpecAndResourceReferences_ValidWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.Nil(t, err)
}

func TestGetNamespaceFromResourceReferences(t *testing.T) {
	tests := []struct {
		name              string
		references        []*apiv1beta1.ResourceReference
		expectedNamespace string
	}{
		{
			"resource reference with namespace and experiment",
			[]*apiv1beta1.ResourceReference{
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123",
					},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns",
					},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
			"ns",
		},
		{
			"resource reference with experiment only",
			[]*apiv1beta1.ResourceReference{
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123",
					},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
			},
			"",
		},
	}
	for _, tc := range tests {
		namespace := getNamespaceFromResourceReferenceV1(tc.references)
		assert.Equal(t, tc.expectedNamespace, namespace,
			"TestGetNamespaceFromResourceReferences(%v) has unexpected result", tc.name)
	}
}

func TestGetExperimentIDFromResourceReferences(t *testing.T) {
	tests := []struct {
		name                 string
		references           []*apiv1beta1.ResourceReference
		expectedExperimentID string
	}{
		{
			"resource reference with namespace and experiment",
			[]*apiv1beta1.ResourceReference{
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123",
					},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns",
					},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
			"123",
		},
		{
			"resource reference with namespace only",
			[]*apiv1beta1.ResourceReference{
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns",
					},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
			"",
		},
	}
	for _, tc := range tests {
		experimentID := getExperimentIdFromResourceReferencesV1(tc.references)
		assert.Equal(t, tc.expectedExperimentID, experimentID,
			"TestGetExperimentIDFromResourceReferences(%v) has unexpected result", tc.name)
	}
}

func TestValidateRunMetric_Pass(t *testing.T) {
	metric := &model.RunMetric{
		Name:   "foo",
		NodeID: "node-1",
	}
	err := validateRunMetric(metric)

	assert.Nil(t, err)
}

func TestValidateRunMetric_InvalidNames(t *testing.T) {
	metric := &model.RunMetric{
		NodeID: "node-1",
	}

	// Empty name
	err := validateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Unallowed character
	metric.Name = "$"
	err = validateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Name is too long
	bytes := make([]byte, 65)
	for i := range bytes {
		bytes[i] = 'a'
	}
	metric.Name = string(bytes)
	err = validateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)
}

func TestValidateRunMetric_InvalidNodeIDs(t *testing.T) {
	metric := &model.RunMetric{
		Name: "a",
	}

	// Empty node ID
	err := validateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Node ID is too long
	metric.NodeID = string(make([]byte, 129))
	err = validateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)
}

func loadYaml(t *testing.T, path string) string {
	res, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	return string(res)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
func TestPipelineSpecStructToYamlString_DirectSpec(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipeline structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))
	protojson.Unmarshal(pipelineSpecJson, &pipeline)

	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshalled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipeline.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
func TestPipelineSpecStructToYamlString_WithPlatform(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipelineSpec structpb.Struct
	var platformSpec structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))
	protojson.Unmarshal(pipelineSpecJson, &pipelineSpec)

	platformSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[1]))
	protojson.Unmarshal(platformSpecJson, &platformSpec)

	pipelineSpecValue := structpb.NewStructValue(&pipelineSpec)
	platformSpecValue := structpb.NewStructValue(&platformSpec)

	pipeline := structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pipeline_spec": pipelineSpecValue,
			"platform_spec": platformSpecValue,
		},
	}
	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshalled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipeline.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}

// Tests both YamlStringToPipelineSpecStruct and pipelineSpecStructToYamlString.
// In this case although the received pipeline spec is nested, because platform spec is empty,
// we return the pipeline spec directly.
func TestPipelineSpecStructToYamlString_NestedPipelineSpec(t *testing.T) {
	template := loadYaml(t, "test/pipeline_with_volume.yaml")

	var pipelineSpec structpb.Struct

	splitTemplate := strings.Split(template, "\n---\n")
	pipelineSpecJson, _ := yaml.YAMLToJSON([]byte(splitTemplate[0]))
	protojson.Unmarshal(pipelineSpecJson, &pipelineSpec)
	pipelineSpecValue := structpb.NewStructValue(&pipelineSpec)

	pipeline := structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pipeline_spec": pipelineSpecValue,
		},
	}
	actualTemplate, err := pipelineSpecStructToYamlString(&pipeline)
	assert.Nil(t, err)

	actualPipeline, err := YamlStringToPipelineSpecStruct(actualTemplate)
	assert.Nil(t, err)

	// Compare the marshalled JSON due to flakiness of structpb values
	// See https://github.com/stretchr/testify/issues/758
	j1, _ := pipelineSpec.MarshalJSON()
	j2, _ := actualPipeline.MarshalJSON()
	assert.Equal(t, j1, j2)
}
