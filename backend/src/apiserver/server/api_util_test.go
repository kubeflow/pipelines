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
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
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
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key: &apiv1beta1.ResourceKey{
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "456"},
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
				Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123"},
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
				Type: apiv1beta1.ResourceType_EXPERIMENT},
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
				Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: apiv1beta1.Relationship_CREATOR,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected relationship for the experiment")
}

func TestValidateExperimentResourceReference_ExperimentNotExist(t *testing.T) {
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	defer clients.Close()
	err := ValidateExperimentResourceReference(manager, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest or pipeline manifest")
}

func TestValidatePipelineSpecAndResourceReferences_WorkflowManifestAndPipelineID(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId:       DefaultFakeUUID,
		WorkflowManifest: testWorkflow.ToStringForStore()}
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
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version)")
}

func TestValidatePipelineSpecAndResourceReferences_EmptyPipelineSpecAndEmptyPipelineVersion(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a (workflow manifest or pipeline manifest) or (pipeline id or/and pipeline version)")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{PipelineId: "not-found"}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineId failed")
}

func TestValidatePipelineSpecAndResourceReferences_InvalidPipelineVersionId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	err := ValidatePipelineSpecAndResourceReferences(manager, nil, referencesOfInvalidPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipelineVersionId failed")
}

func TestValidatePipelineSpecAndResourceReferences_PipelineIdNotParentOfPipelineVersionId(t *testing.T) {
	clients := initWithExperimentsAndTwoPipelineVersions(t)
	manager := resource.NewResourceManager(clients, map[string]interface{}{"DefaultNamespace": "default", "ApiVersion": "v2beta1"})
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId: NonDefaultFakeUUID}
	err := ValidatePipelineSpecAndResourceReferences(manager, spec, validReferencesOfExperimentAndPipelineVersion)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "pipeline ID should be parent of pipeline version")
}

func TestValidatePipelineSpecAndResourceReferences_ParameterTooLongWithPipelineId(t *testing.T) {
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
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
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
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
	clients, manager, _ := initWithExperimentAndPipelineVersion(t)
	defer clients.Close()
	spec := &apiv1beta1.PipelineSpec{
		PipelineId: DefaultFakeUUID}
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
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns"},
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
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123"},
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
						Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "123"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
				{
					Key: &apiv1beta1.ResourceKey{
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns"},
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
						Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns"},
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

// func TestValidateRunMetricV1_Pass(t *testing.T) {
// 	metric := &apiv1beta1.RunMetric{
// 		Name:   "foo",
// 		NodeId: "node-1",
// 	}

// 	err := ValidateRunMetricV1(metric)

// 	assert.Nil(t, err)
// }

// func TestValidateRunMetricV1_InvalidNames(t *testing.T) {
// 	metric := &apiv1beta1.RunMetric{
// 		NodeId: "node-1",
// 	}

// 	// Empty name
// 	err := ValidateRunMetricV1(metric)
// 	AssertUserError(t, err, codes.InvalidArgument)

// 	// Unallowed character
// 	metric.Name = "$"
// 	err = ValidateRunMetricV1(metric)
// 	AssertUserError(t, err, codes.InvalidArgument)

// 	// Name is too long
// 	bytes := make([]byte, 65)
// 	for i := range bytes {
// 		bytes[i] = 'a'
// 	}
// 	metric.Name = string(bytes)
// 	err = ValidateRunMetricV1(metric)
// 	AssertUserError(t, err, codes.InvalidArgument)
// }

// func TestValidateRunMetricV1_InvalidNodeIDs(t *testing.T) {
// 	metric := &apiv1beta1.RunMetric{
// 		Name: "a",
// 	}

// 	// Empty node ID
// 	err := ValidateRunMetricV1(metric)
// 	AssertUserError(t, err, codes.InvalidArgument)

// 	// Node ID is too long
// 	metric.NodeId = string(make([]byte, 129))
// 	err = ValidateRunMetricV1(metric)
// 	AssertUserError(t, err, codes.InvalidArgument)
// }

// func TestnewReportRunMetricResultV1_OK(t *testing.T) {
// 	tests := []struct {
// 		metricName string
// 	}{
// 		{"metric-1"},
// 		{"Metric_2"},
// 		{"Metric3Name"},
// 	}

// 	for _, tc := range tests {
// 		expected := newReportRunMetricResultV1(tc.metricName, "node-1")
// 		expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
// 		actual := newReportRunMetricResultV1(expected.GetMetricName(), expected.GetMetricNodeId(), nil)

// 		assert.Equalf(t, expected, actual, "TestNewReportRunMetricResult_OK metric name '%s' should be OK", tc.metricName)
// 	}
// }

// func TestnewReportRunMetricResultV1_UnknownError(t *testing.T) {
// 	expected := newReportRunMetricResultV1("metric-1", "node-1")
// 	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR

// 	actual := newReportRunMetricResultV1(
// 		expected.GetMetricName(), expected.GetMetricNodeId())

// 	assert.Equal(t, expected, actual)
// }

// func TestnewReportRunMetricResultV1_InternalError(t *testing.T) {
// 	expected := newReportRunMetricResultV1("metric-1", "node-1")
// 	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR
// 	expected.Message = "Internal Server Error"
// 	actual := newReportRunMetricResultV1(
// 		expected.GetMetricName(), expected.GetMetricNodeId())

// 	assert.Equal(t, expected, actual)
// }

// func TestnewReportRunMetricResultV1_InvalidArgument(t *testing.T) {
// 	expected := newReportRunMetricResultV1("metric-1", "node-1")
// 	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT
// 	expected.Message = "Foo is invalid"
// 	error := util.NewInvalidInputError(expected.Message)

// 	actual := newReportRunMetricResultV1(
// 		expected.GetMetricName(), expected.GetMetricNodeId(), error)

// 	assert.Equal(t, expected, actual)
// }

// func TestnewReportRunMetricResultV1_AlreadyExist(t *testing.T) {
// 	expected := newReportRunMetricResultV1("metric-1", "node-1")
// 	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_DUPLICATE_REPORTING
// 	expected.Message = "Foo is duplicate"
// 	error := util.NewAlreadyExistError(expected.Message)

// 	actual := newReportRunMetricResultV1(
// 		expected.GetMetricName(), expected.GetMetricNodeId(), error)

// 	assert.Equal(t, expected, actual)
// }

// func newReportRunMetricResultV1(metricName string, nodeID string) *apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult {
// 	return &apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
// 		MetricName:   metricName,
// 		MetricNodeId: nodeID,
// 	}
// }

// func newReportRunMetricResult(metricName string, nodeID string) *apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult {
// 	return &apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult{
// 		MetricName:   metricName,
// 		MetricNodeId: nodeID,
// 	}
// }

// func TestValidateRunMetric_Pass(t *testing.T) {
// 	metric := &apiv2beta1.RunMetric{
// 		DisplayName: "foo",
// 		NodeId:      "node-1",
// 	}

// 	err := ValidateRunMetric(metric)

// 	assert.Nil(t, err)
// }

// func TestNewReportRunMetricResult_OK(t *testing.T) {
// 	tests := []struct {
// 		metricName string
// 	}{
// 		{"metric-1"},
// 		{"Metric_2"},
// 		{"Metric3Name"},
// 	}

// 	for _, tc := range tests {
// 		expected := newReportRunMetricResult(tc.metricName, "node-1")
// 		expected.Status = apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
// 		actual := NewReportRunMetricResult(expected.GetMetricName(), expected.GetMetricNodeId(), nil)

// 		assert.Equalf(t, expected, actual, "TestNewReportRunMetricResult_OK metric name '%s' should be OK", tc.metricName)
// 	}
// }
