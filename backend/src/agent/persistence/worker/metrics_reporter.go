// Copyright 2018 Google LLC
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

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
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
func (r MetricsReporter) ReportMetrics(workflow *util.Workflow) error {
	if workflow.Status.Nodes == nil {
		return nil
	}
	if _, ok := workflow.ObjectMeta.Labels[util.LabelKeyWorkflowRunId]; !ok {
		// Skip reporting if the workflow doesn't have the run id label
		return nil
	}
	runID := workflow.ObjectMeta.Labels[util.LabelKeyWorkflowRunId]
	runMetrics := []*api.RunMetric{}
	partialFailures := []error{}
	for _, nodeStatus := range workflow.Status.Nodes {
		nodeMetrics, err := r.collectNodeMetricsOrNil(runID, nodeStatus)
		if err != nil {
			partialFailures = append(partialFailures, err)
			continue
		}
		if nodeMetrics != nil {
			if len(runMetrics)+len(nodeMetrics) >= maxMetricsCountLimit {
				leftQuota := maxMetricsCountLimit - len(runMetrics)
				runMetrics = append(runMetrics, nodeMetrics[0:leftQuota]...)
				// TODO(#1426): report the error back to api server to notify user
				log.Errorf("Reported metrics are more than the limit %v", maxMetricsCountLimit)
				break
			}
			runMetrics = append(runMetrics, nodeMetrics...)
		}
	}
	if len(runMetrics) == 0 {
		return aggregateErrors(partialFailures)
	}
	reportMetricsResponse, err := r.pipelineClient.ReportRunMetrics(&api.ReportRunMetricsRequest{
		RunId:   runID,
		Metrics: runMetrics,
	})
	if err != nil {
		return err
	}

	partialFailures = append(partialFailures, processReportMetricResults(reportMetricsResponse)...)
	return aggregateErrors(partialFailures)
}

func (r MetricsReporter) collectNodeMetricsOrNil(
	runID string, nodeStatus workflowapi.NodeStatus) (
	[]*api.RunMetric, error) {
	if !nodeStatus.Completed() {
		return nil, nil
	}
	metricsJSON, err := r.readNodeMetricsJSONOrEmpty(runID, nodeStatus)
	if err != nil || metricsJSON == "" {
		return nil, err
	}

	// Proto json lib requires a proto message before unmarshal data from JSON. We use
	// ReportRunMetricsRequest as a workaround to hold user's metrics, which is a superset of what
	// user can provide.
	reportMetricsRequest := new(api.ReportRunMetricsRequest)
	err = jsonpb.UnmarshalString(metricsJSON, reportMetricsRequest)
	if err != nil {
		// User writes invalid metrics JSON.
		// TODO(#1426): report the error back to api server to notify user
		log.WithFields(log.Fields{
			"run":         runID,
			"node":        nodeStatus.ID,
			"raw_content": metricsJSON,
			"error":       err.Error(),
		}).Warning("Failed to unmarshal metrics file.")
		return nil, util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"failed to unmarshal metrics file from (%s, %s).", runID, nodeStatus.ID)
	}
	if reportMetricsRequest.GetMetrics() == nil {
		return nil, nil
	}
	for _, metric := range reportMetricsRequest.GetMetrics() {
		// User metrics just have name and value but no NodeId.
		metric.NodeId = nodeStatus.ID
	}
	return reportMetricsRequest.GetMetrics(), nil
}

func (r MetricsReporter) readNodeMetricsJSONOrEmpty(runID string, nodeStatus workflowapi.NodeStatus) (string, error) {
	if nodeStatus.Outputs == nil || nodeStatus.Outputs.Artifacts == nil {
		return "", nil // No output artifacts, skip the reporting
	}

	var foundMetricsArtifact bool = false
	for _, artifact := range nodeStatus.Outputs.Artifacts {
		if artifact.Name == metricsArtifactName {
			foundMetricsArtifact = true
		}
	}
	if !foundMetricsArtifact {
		return "", nil // No metrics artifact, skip the reporting
	}

	artifactRequest := &api.ReadArtifactRequest{
		RunId:        runID,
		NodeId:       nodeStatus.ID,
		ArtifactName: metricsArtifactName,
	}
	artifactResponse, err := r.pipelineClient.ReadArtifact(artifactRequest)
	if err != nil {
		return "", err
	}
	if artifactResponse == nil || artifactResponse.GetData() == nil || len(artifactResponse.GetData()) == 0 {
		// If artifact is not found or empty content, skip the reporting.
		return "", nil
	}
	archivedFiles, err := util.ExtractTgz(string(artifactResponse.GetData()))
	if err != nil {
		// Invalid tgz file. This should never happen unless there is a bug in the system and
		// it is a unrecoverable error.
		return "", util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Unable to extract metrics tgz file read from (%+v): %v", artifactRequest, err)
	}
	//There needs to be exactly one metrics file in the artifact archive. We load that file.
	if len(archivedFiles) == 1 {
		for _, value := range archivedFiles {
			return value, nil
		}
	}
	return "", util.NewCustomErrorf(util.CUSTOM_CODE_PERMANENT,
		"There needs to be exactly one metrics file in the artifact archive, but zero or multiple files were found.")
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
