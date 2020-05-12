// Copyright 2018 Google LLC
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

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestValidateRunMetric_Pass(t *testing.T) {
	metric := &api.RunMetric{
		Name:   "foo",
		NodeId: "node-1",
	}

	err := ValidateRunMetric(metric)

	assert.Nil(t, err)
}

func TestValidateRunMetric_InvalidNames(t *testing.T) {
	metric := &api.RunMetric{
		NodeId: "node-1",
	}

	// Empty name
	err := ValidateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Unallowed character
	metric.Name = "$"
	err = ValidateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Name is too long
	bytes := make([]byte, 65)
	for i := range bytes {
		bytes[i] = 'a'
	}
	metric.Name = string(bytes)
	err = ValidateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)
}

func TestValidateRunMetric_InvalidNodeIDs(t *testing.T) {
	metric := &api.RunMetric{
		Name: "a",
	}

	// Empty node ID
	err := ValidateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)

	// Node ID is too long
	metric.NodeId = string(make([]byte, 129))
	err = ValidateRunMetric(metric)
	AssertUserError(t, err, codes.InvalidArgument)
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
		expected.Status = api.ReportRunMetricsResponse_ReportRunMetricResult_OK
		actual := NewReportRunMetricResult(expected.GetMetricName(), expected.GetMetricNodeId(), nil)

		assert.Equalf(t, expected, actual, "TestNewReportRunMetricResult_OK metric name '%s' should be OK", tc.metricName)
	}
}

func TestNewReportRunMetricResult_UnknownError(t *testing.T) {
	expected := newReportRunMetricResult("metric-1", "node-1")
	expected.Status = api.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR

	actual := NewReportRunMetricResult(
		expected.GetMetricName(), expected.GetMetricNodeId(), errors.New("test"))

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResult_InternalError(t *testing.T) {
	expected := newReportRunMetricResult("metric-1", "node-1")
	expected.Status = api.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR
	expected.Message = "Internal Server Error"
	error := util.NewInternalServerError(errors.New("test"), "Foo Error")

	actual := NewReportRunMetricResult(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResult_InvalidArgument(t *testing.T) {
	expected := newReportRunMetricResult("metric-1", "node-1")
	expected.Status = api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT
	expected.Message = "Foo is invalid"
	error := util.NewInvalidInputError(expected.Message)

	actual := NewReportRunMetricResult(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func TestNewReportRunMetricResult_AlreadyExist(t *testing.T) {
	expected := newReportRunMetricResult("metric-1", "node-1")
	expected.Status = api.ReportRunMetricsResponse_ReportRunMetricResult_DUPLICATE_REPORTING
	expected.Message = "Foo is duplicate"
	error := util.NewAlreadyExistError(expected.Message)

	actual := NewReportRunMetricResult(
		expected.GetMetricName(), expected.GetMetricNodeId(), error)

	assert.Equal(t, expected, actual)
}

func newReportRunMetricResult(metricName string, nodeID string) *api.ReportRunMetricsResponse_ReportRunMetricResult {
	return &api.ReportRunMetricsResponse_ReportRunMetricResult{
		MetricName:   metricName,
		MetricNodeId: nodeID,
	}
}
