// Copyright 2018-2022 The Kubeflow Authors
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
	"errors"
	"testing"

	"google.golang.org/grpc/codes"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestValidateRunMetricV1_Pass(t *testing.T) {
	metric := &apiv1beta1.RunMetric{
		Name:   "foo",
		NodeId: "node-1",
	}

	err := ValidateRunMetricV1(metric)

	assert.Nil(t, err)
}

func TestValidateRunMetricV1_InvalidNames(t *testing.T) {
	metric := &apiv1beta1.RunMetric{
		NodeId: "node-1",
	}

	// Empty name
	err := ValidateRunMetricV1(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Unallowed character
	metric.Name = "$"
	err = ValidateRunMetricV1(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Name is too long
	bytes := make([]byte, 65)
	for i := range bytes {
		bytes[i] = 'a'
	}
	metric.Name = string(bytes)
	err = ValidateRunMetricV1(metric)
	AssertUserError(t, err, codes.InvalidArgument)
}

func TestValidateRunMetricV1_InvalidNodeIDs(t *testing.T) {
	metric := &apiv1beta1.RunMetric{
		Name: "a",
	}

	// Empty node ID
	err := ValidateRunMetricV1(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Node ID is too long
	metric.NodeId = string(make([]byte, 129))
	err = ValidateRunMetricV1(metric)
	AssertUserError(t, err, codes.InvalidArgument)
}

func TestNewReportRunMetricResultV1_OK(t *testing.T) {
	tests := []struct {
		metricName string
	}{
		{"metric-1"},
		{"Metric_2"},
		{"Metric3Name"},
	}

	for _, tc := range tests {
		expected := newReportRunMetricResultV1(tc.metricName, "node-1")
		expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
		actual := NewReportRunMetricResultV1(expected.GetMetricName(), expected.GetMetricNodeId(), nil)

		assert.Equalf(t, expected, actual, "TestNewReportRunMetricResult_OK metric name '%s' should be OK", tc.metricName)
	}
}

func TestNewReportRunMetricResultV1_UnknownError(t *testing.T) {
	expected := newReportRunMetricResultV1("metric-1", "node-1")
	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR

	actual := NewReportRunMetricResultV1(
		expected.GetMetricName(), expected.GetMetricNodeId(), errors.New("test"))

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResultV1_InternalError(t *testing.T) {
	expected := newReportRunMetricResultV1("metric-1", "node-1")
	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR
	expected.Message = "Internal Server Error"
	error := util.NewInternalServerError(errors.New("test"), "Foo Error")

	actual := NewReportRunMetricResultV1(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResultV1_InvalidArgument(t *testing.T) {
	expected := newReportRunMetricResultV1("metric-1", "node-1")
	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT
	expected.Message = "Foo is invalid"
	error := util.NewInvalidInputError(expected.Message)

	actual := NewReportRunMetricResultV1(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResultV1_AlreadyExist(t *testing.T) {
	expected := newReportRunMetricResultV1("metric-1", "node-1")
	expected.Status = apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult_DUPLICATE_REPORTING
	expected.Message = "Foo is duplicate"
	error := util.NewAlreadyExistError(expected.Message)

	actual := NewReportRunMetricResultV1(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func newReportRunMetricResultV1(metricName string, nodeID string) *apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult {
	return &apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{
		MetricName:   metricName,
		MetricNodeId: nodeID,
	}
}

func newReportRunMetricResult(metricName string, nodeID string) *apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult {
	return &apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult{
		MetricName:   metricName,
		MetricNodeId: nodeID,
	}
}

func TestValidateRunMetric_Pass(t *testing.T) {
	metric := &apiv2beta1.RunMetric{
		DisplayName: "foo",
		NodeId:      "node-1",
	}

	err := ValidateRunMetric(metric)

	assert.Nil(t, err)
}

func TestNewReportRunMetricResult_OK(t *testing.T) {
	tests := []struct {
		metricName string
	}{
		{"metric-1"},
		{"Metric_2"},
		{"Metric3Name"},
	}

	for _, tc := range tests {
		expected := newReportRunMetricResult(tc.metricName, "node-1")
		expected.Status = apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult_OK
		actual := NewReportRunMetricResult(expected.GetMetricName(), expected.GetMetricNodeId(), nil)

		assert.Equalf(t, expected, actual, "TestNewReportRunMetricResult_OK metric name '%s' should be OK", tc.metricName)
	}
}
