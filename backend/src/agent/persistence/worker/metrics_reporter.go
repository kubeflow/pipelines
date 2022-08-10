// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"errors"
	"strings"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	metricsArtifactName = "mlpipeline-metrics"
	// More than 50 metrics is not scalable with current UI design.
	maxMetricsCountLimit = 50
)

// MetricsReporter reports metrics of a workflow to pipeline server.
type MetricsReporter struct {
	pipelineClient client.PipelineClientInterface
}

// NewMetricsReporter creates a new instance of NewMetricsReporter.
func NewMetricsReporter(pipelineClient client.PipelineClientInterface) *MetricsReporter {
	return &MetricsReporter{
		pipelineClient: pipelineClient,
	}
}

// ReportMetrics reports workflow metrics to pipeline server.
func (r MetricsReporter) ReportMetrics(workflow util.ExecutionSpec, user string) error {
	if !workflow.ExecutionStatus().HasMetrics() {
		return nil
	}
	objMeta := workflow.ExecutionObjectMeta()
	runID, ok := objMeta.Labels[util.LabelKeyWorkflowRunId]
	if !ok {
		// Skip reporting if the workflow doesn't have the run id label
		return nil
	}
	runMetrics, partialFailures := workflow.ExecutionStatus().CollectionMetrics(r.pipelineClient.ReadArtifact, user)
	if len(runMetrics) == 0 {
		return aggregateErrors(partialFailures)
	}
	reportMetricsResponse, err := r.pipelineClient.ReportRunMetrics(&api.ReportRunMetricsRequest{
		RunId:   runID,
		Metrics: runMetrics,
	}, user)
	if err != nil {
		return err
	}

	partialFailures = append(partialFailures, processReportMetricResults(reportMetricsResponse)...)
	return aggregateErrors(partialFailures)
}

func processReportMetricResults(
	reportMetricsResponse *api.ReportRunMetricsResponse) []error {
	errors := []error{}
	for _, result := range reportMetricsResponse.GetResults() {
		err := processReportMetricResult(result)
		if err != nil {
			errors = append(errors, processReportMetricResult(result))
		}
	}
	return errors
}

func processReportMetricResult(
	result *api.ReportRunMetricsResponse_ReportRunMetricResult) error {
	switch result.GetStatus() {
	case api.ReportRunMetricsResponse_ReportRunMetricResult_INVALID_ARGUMENT:
		// TODO(#1426): report user error back to API server to notify user.
		return util.NewCustomError(
			errors.New(result.GetMessage()), util.CUSTOM_CODE_PERMANENT,
			"failed to report metric because of invalid arguments: %+v", result)
	case api.ReportRunMetricsResponse_ReportRunMetricResult_INTERNAL_ERROR:
		// Internal error is considered as trasient and should be retried later.
		return util.NewCustomError(
			errors.New(result.GetMessage()), util.CUSTOM_CODE_TRANSIENT,
			"failed to report metric because of internal error: %+v", result)
	default:
		// Ignore OK, DUP_REPORTING and UNSPECIFIED errors.
		return nil
	}
}

func aggregateErrors(errors []error) error {
	if errors == nil || len(errors) == 0 {
		return nil
	}
	code := util.CUSTOM_CODE_PERMANENT
	var errorMsgs []string
	for _, err := range errors {
		if util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT) {
			// Try our  best to recover partial failures.
			code = util.CUSTOM_CODE_TRANSIENT
		}
		errorMsgs = append(errorMsgs, err.Error())
	}
	return util.NewCustomErrorf(code, strings.Join(errorMsgs, "\n"))
}
