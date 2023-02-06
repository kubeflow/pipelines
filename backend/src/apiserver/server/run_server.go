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

package server

import (
	"context"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// Metric variables. Please prefix the metric names with run_server_.
var (
	// Used to calculate the request rate.
	createRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_create_requests",
		Help: "The total number of CreateRun requests",
	})

	getRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_get_requests",
		Help: "The total number of GetRun requests",
	})

	listRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_list_requests",
		Help: "The total number of ListRuns requests",
	})

	deleteRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_delete_requests",
		Help: "The total number of DeleteRun requests",
	})

	archiveRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_archive_requests",
		Help: "The total number of ArchiveRun requests",
	})

	unarchiveRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_unarchive_requests",
		Help: "The total number of UnarchiveRun requests",
	})

	reportRunMetricsRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_report_metrics_requests",
		Help: "The total number of ReportRunMetrics requests",
	})

	readArtifactRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_read_artifact_requests",
		Help: "The total number of ReadArtifact requests",
	})

	terminateRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_terminate_requests",
		Help: "The total number of TerminateRun requests",
	})

	retryRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_server_retry_requests",
		Help: "The total number of RetryRun requests",
	})

	// TODO(jingzhang36): error count and success count.

	runCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "run_server_run_count",
		Help: "The current number of runs in Kubeflow Pipelines instance",
	})
)

type RunServerOptions struct {
	CollectMetrics bool `json:"collect_metrics,omitempty"`
}

type RunServer struct {
	resourceManager *resource.ResourceManager
	options         *RunServerOptions
}

func NewRunServer(resourceManager *resource.ResourceManager, options *RunServerOptions) *RunServer {
	return &RunServer{resourceManager: resourceManager, options: options}
}

// Creates a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) createRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	if common.IsMultiUserMode() {
		if run.ExperimentId == "" {
			return nil, util.NewInvalidInputError("Failed to create a run due to missing parent experiment id")
		}
	}
	if err := s.resourceManager.ValidateExperimentNamespace(run.ExperimentId, run.Namespace); err != nil {
		return nil, util.Wrapf(err, "Failed to create a run due to namespace mismatch. Specified namespace %s is different from what the parent experiment %s has", run.Namespace, run.ExperimentId)
	}
	ns, err := s.resourceManager.GetNamespaceFromExperimentId(run.ExperimentId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a run due to error fetching parent experiment's %s namespace", run.ExperimentId)
	}
	run.Namespace = ns

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: run.Namespace,
		Verb:      common.RbacResourceVerbCreate,
		Resource:  common.RbacResourceTypePipelines,
	}
	if err := s.resourceManager.IsAuthorized(ctx, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to create a run due to authorization error. Check if you have write permissions to namespace %s", run.Namespace)
	}

	// Validate the pipeline. Fail fast if this is corrupted.
	if err := s.validateRun(run); err != nil {
		return nil, util.Wrap(err, "Failed to create a run as run validation failed")
	}
	return s.resourceManager.CreateRun(ctx, run)
}

// Creates a run.
// Supports v1beta1 behavior.
func (s *RunServer) CreateRunV1(ctx context.Context, request *apiv1beta1.CreateRunRequest) (*apiv1beta1.RunDetail, error) {
	if s.options.CollectMetrics {
		createRunRequests.Inc()
	}

	modelRun, err := toModelRun(request.GetRun())
	if err != nil {
		return nil, util.Wrap(err, "CreateJob(job.ToV2())Failed to create a v1beta1 run due to conversion error")
	}

	run, err := s.createRun(ctx, modelRun)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new v1beta1 run")
	}

	if s.options.CollectMetrics {
		runCount.Inc()
	}
	return toApiRunDetailV1(run), nil
}

// Fetches a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) getRun(ctx context.Context, runId string) (*model.Run, error) {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.resourceManager.GetRun(runId)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// Fetches a run.
// Supports v1beta1 behavior.
func (s *RunServer) GetRunV1(ctx context.Context, request *apiv1beta1.GetRunRequest) (*apiv1beta1.RunDetail, error) {
	if s.options.CollectMetrics {
		getRunRequests.Inc()
	}

	run, err := s.getRun(ctx, request.RunId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get a v1beta1 run")
	}

	return toApiRunDetailV1(run), nil
}

// Fetches all runs that conform to the specified filter and listing options.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) listRuns(ctx context.Context, pageToken string, pageSize int, sortBy string, filter string, namespace string, experimentId string) ([]*model.Run, int, string, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if experimentId != "" {
		ns, err := s.resourceManager.GetNamespaceFromExperimentId(experimentId)
		if err != nil {
			return nil, 0, "", util.Wrapf(err, "Failed to list runs due to error fetching namespace for experiment %s. Try filtering based on namespace", experimentId)
		}
		namespace = ns
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	err := s.canAccessRun(ctx, "", resourceAttributes)
	if err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list runs due to authorization error. Check if you have permission to access namespace %s", namespace)
	}

	opts, err := validatedListOptions(&model.Run{}, pageToken, pageSize, sortBy, filter)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "Failed to create list options")
	}
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}
	if experimentId != "" {
		if err := s.resourceManager.ValidateExperimentNamespace(experimentId, namespace); err != nil {
			return nil, 0, "", util.Wrap(err, "Failed to list runs due to namespace mismatch")
		}
		filterContext = &model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: experimentId},
		}
	}
	runs, totalSize, token, err := s.resourceManager.ListRuns(filterContext, opts)
	if err != nil {
		return nil, 0, "", err
	}
	return runs, totalSize, token, nil
}

// Fetches runs given query parameters.
// Supports v1beta1 behavior.
func (s *RunServer) ListRunsV1(ctx context.Context, r *apiv1beta1.ListRunsRequest) (*apiv1beta1.ListRunsResponse, error) {
	if s.options.CollectMetrics {
		listRunRequests.Inc()
	}

	filterContext, err := validateFilterV1(r.GetResourceReferenceKey())
	if err != nil {
		return nil, util.Wrap(err, "Failed to list v1beta1 runs: validating filter failed")
	}
	namespace := ""
	experimentId := ""

	if filterContext.ReferenceKey != nil {
		switch filterContext.ReferenceKey.Type {
		case model.NamespaceResourceType:
			namespace = filterContext.ReferenceKey.ID
		case model.ExperimentResourceType:
			experimentId = filterContext.ReferenceKey.ID
		}
	}
	runs, runsCount, nextPageToken, err := s.listRuns(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), namespace, experimentId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list v1beta1 runs")
	}

	return &apiv1beta1.ListRunsResponse{Runs: toApiRunsV1(runs), TotalSize: int32(runsCount), NextPageToken: nextPageToken}, nil
}

// Archives a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) archiveRun(ctx context.Context, runId string, experimentId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbArchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.getRun(ctx, runId)
	if err != nil {
		return util.Wrap(err, "Failed to fetch a run")
	}
	if experimentId != "" && run.ExperimentId != experimentId && run.ExperimentId != "" {
		return util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, run.ExperimentId, experimentId), "Failed to archive a run")
	}
	return s.resourceManager.ArchiveRun(runId)
}

// Archives a run.
// Supports v1beta1 behavior.
func (s *RunServer) ArchiveRunV1(ctx context.Context, request *apiv1beta1.ArchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}
	err := s.archiveRun(ctx, request.GetId(), "")
	if err != nil {
		return nil, util.Wrap(err, "Failed to archive a v1beta1 run")
	}
	return &empty.Empty{}, nil
}

// Un-archives a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) unarchiveRun(ctx context.Context, runId string, experimentId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.getRun(ctx, runId)
	if err != nil {
		return util.Wrap(err, "Failed to fetch a run")
	}
	if experimentId != "" && run.ExperimentId != experimentId && run.ExperimentId != "" {
		return util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, run.ExperimentId, experimentId), "Failed to unarchive a run")
	}
	return s.resourceManager.UnarchiveRun(runId)
}

// Un-archives a run.
// Supports v1beta1 behavior.
func (s *RunServer) UnarchiveRunV1(ctx context.Context, request *apiv1beta1.UnarchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}
	err := s.unarchiveRun(ctx, request.GetId(), "")
	if err != nil {
		return nil, util.Wrap(err, "Failed to unarchive a v1beta1 run")
	}
	return &empty.Empty{}, nil
}

// Deletes a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) deleteRun(ctx context.Context, runId string, experimentId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.getRun(ctx, runId)
	if err != nil {
		return util.Wrap(err, "Failed to fetch a run")
	}
	if experimentId != "" && run.ExperimentId != experimentId && run.ExperimentId != "" {
		return util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, run.ExperimentId, experimentId), "Failed to delete a run")
	}
	return s.resourceManager.DeleteRun(ctx, runId)
}

// Deletes a run.
// Supports v1beta1 behavior.
func (s *RunServer) DeleteRunV1(ctx context.Context, request *apiv1beta1.DeleteRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}
	if err := s.deleteRun(ctx, request.GetId(), ""); err != nil {
		return nil, util.Wrap(err, "Failed to delete a v1beta1 run")
	}
	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &empty.Empty{}, nil
}

// Reports run metrics.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) reportRunMetrics(ctx context.Context, metrics []*model.RunMetric, runId string, experimentId string) ([]map[string]string, error) {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	existingRun, err := s.resourceManager.GetRun(runId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch the requested run")
	}
	if experimentId != "" && existingRun.ExperimentId != experimentId && existingRun.ExperimentId != "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, existingRun.ExperimentId, experimentId), "Failed to report run metrics")
	}
	if experimentId != "" {
		if err := s.resourceManager.ValidateExperimentNamespace(experimentId, existingRun.Namespace); err != nil {
			return nil, util.Wrap(err, "Failed to report run metrics due to namespace mismatch")
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: existingRun.Namespace,
			Verb:      common.RbacResourceVerbList,
		}
		if err := s.canAccessRun(ctx, "", resourceAttributes); err != nil {
			return nil, util.Wrapf(err, "Failed to report run metrics due to authorization error. Check if you have permission to access namespace %s", existingRun.Namespace)
		}
	}
	results := make([]map[string]string, 0)
	for _, metric := range metrics {
		temp := map[string]string{"Name": metric.Name, "NodeId": metric.NodeID, "ErrorCode": "", "ErrorMessage": ""}
		if err := validateRunMetric(metric); err != nil {
			temp["ErrorCode"] = "invalid"
			results = append(results, temp)
			continue
		}
		err = s.resourceManager.ReportMetric(metric)
		if err == nil {
			temp["ErrorCode"] = "ok"
			results = append(results, temp)
			continue
		}
		err, ok := err.(*util.UserError)
		if !ok {
			temp["ErrorCode"] = "internal"
			results = append(results, temp)
			continue
		}
		temp["ErrorMessage"] = err.ExternalMessage()
		switch err.ExternalStatusCode() {
		case codes.AlreadyExists:
			temp["ErrorCode"] = "duplicate"
		case codes.InvalidArgument:
			temp["ErrorCode"] = "invalid"
		default:
			temp["ErrorCode"] = "internal"
		}
		if temp["ErrorCode"] == "internal" {
			glog.Errorf("Internal error '%v' when reporting metric '%s/%s'", err, metric.NodeID, metric.Name)
		}
		results = append(results, temp)
	}
	return results, nil
}

// Reports run metrics.
// Supports v1beta1 API.
func (s *RunServer) ReportRunMetricsV1(ctx context.Context, request *apiv1beta1.ReportRunMetricsRequest) (*apiv1beta1.ReportRunMetricsResponse, error) {
	if s.options.CollectMetrics {
		reportRunMetricsRequests.Inc()
	}
	metrics := make([]*model.RunMetric, 0)
	for _, metric := range request.GetMetrics() {
		modelMetric, err := toModelRunMetric(metric, request.GetRunId())
		if err != nil {
			return nil, util.Wrap(err, "Failed to create v1beta1 run metrics due to data conversion error")
		}
		metrics = append(metrics, modelMetric)
	}
	results, err := s.reportRunMetrics(ctx, metrics, request.GetRunId(), "")
	if err != nil {
		return nil, util.Wrap(err, "Failed to report v1beta1 run metrics")
	}
	apiResults := make([]*apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult, 0)
	for _, result := range results {
		apiResults = append(apiResults, toApiReportMetricsResultV1(result["Name"], result["NodeId"], result["ErrorCode"], result["ErrorMessage"]))
	}
	return &apiv1beta1.ReportRunMetricsResponse{
		Results: apiResults,
	}, nil
}

// Fetches run metrics for a given run id.
// Supports v1beta1 behavior.
func (s *RunServer) getRunMetrics(ctx context.Context, runId string) ([]*model.RunMetric, error) {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "CreateJob(job.ToV2())Failed to fetch run metrics due to authorization error")
	}
	_, err = s.resourceManager.GetRun(runId)
	if err != nil {
		return nil, util.Wrapf(err, "CreateJob(job.ToV2())Failed to fetch run metrics. Check if run %s can be fetched", runId)
	}
	return s.resourceManager.GetRunMetrics(runId)
}

// Reads an artifact.
// Supports v1beta1 behavior.
func (s *RunServer) ReadArtifactV1(ctx context.Context, request *apiv1beta1.ReadArtifactRequest) (*apiv1beta1.ReadArtifactResponse, error) {
	if s.options.CollectMetrics {
		readArtifactRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadArtifact})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	content, err := s.resourceManager.ReadArtifact(
		request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		return nil, util.Wrapf(err, "failed to read artifact '%+v'", request)
	}
	return &apiv1beta1.ReadArtifactResponse{
		Data: content,
	}, nil
}

// Terminates a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) terminateRun(ctx context.Context, runId string, experimentId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.getRun(ctx, runId)
	if err != nil {
		return util.Wrap(err, "Failed to fetch a run")
	}
	if experimentId != "" && run.ExperimentId != experimentId && run.ExperimentId != "" {
		return util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, run.ExperimentId, experimentId), "Failed to terminate a run")
	}
	return s.resourceManager.TerminateRun(ctx, runId)
}

// Retries a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *RunServer) retryRun(ctx context.Context, runId string, experimentId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbRetry})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.getRun(ctx, runId)
	if err != nil {
		return util.Wrap(err, "Failed to fetch a run")
	}
	if experimentId != "" && run.ExperimentId != experimentId && run.ExperimentId != "" {
		return util.NewInternalServerError(util.NewInvalidInputError("The requested run '%s' belongs to experiment '%s' (requested experiment '%s')", runId, run.ExperimentId, experimentId), "Failed to retry a run")
	}
	return s.resourceManager.RetryRun(ctx, runId)
}

// Terminates a run.
// Supports v1beta1 behavior.
func (s *RunServer) TerminateRunV1(ctx context.Context, request *apiv1beta1.TerminateRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}
	err := s.terminateRun(ctx, request.GetRunId(), "")
	if err != nil {
		return nil, util.Wrap(err, "Failed to terminate a v1beta1 run")
	}
	return &empty.Empty{}, nil
}

// Retries a run.
// Supports v1beta1 behavior.
func (s *RunServer) RetryRunV1(ctx context.Context, request *apiv1beta1.RetryRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.retryRun(ctx, request.GetRunId(), "")
	if err != nil {
		return nil, util.Wrap(err, "Failed to retry a run")
	}

	return &empty.Empty{}, nil
}

// Creates a run.
// Supports v2beta1 behavior.
func (s *RunServer) CreateRun(ctx context.Context, request *apiv2beta1.CreateRunRequest) (*apiv2beta1.Run, error) {
	if s.options.CollectMetrics {
		createRunRequests.Inc()
	}

	modelRun, err := toModelRun(request.GetRun())
	if err != nil {
		return nil, util.Wrap(err, "CreateJob(job.ToV2())Failed to create a run due to conversion error")
	}

	run, err := s.createRun(ctx, modelRun)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run")
	}

	if s.options.CollectMetrics {
		runCount.Inc()
	}
	return toApiRun(run), nil
}

// Fetches a run.
// Supports v2beta1 behavior.
func (s *RunServer) GetRun(ctx context.Context, request *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
	if s.options.CollectMetrics {
		getRunRequests.Inc()
	}

	run, err := s.getRun(ctx, request.RunId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get a run")
	}

	return toApiRun(run), nil
}

// Fetches runs given query parameters.
// Supports v2beta1 behavior.
func (s *RunServer) ListRuns(ctx context.Context, r *apiv2beta1.ListRunsRequest) (*apiv2beta1.ListRunsResponse, error) {
	if s.options.CollectMetrics {
		listRunRequests.Inc()
	}
	runs, runsCount, nextPageToken, err := s.listRuns(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), r.GetNamespace(), r.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs")
	}
	return &apiv2beta1.ListRunsResponse{Runs: toApiRuns(runs), TotalSize: int32(runsCount), NextPageToken: nextPageToken}, nil
}

// Archives a run.
// Supports v2beta1 behavior.
func (s *RunServer) ArchiveRun(ctx context.Context, request *apiv2beta1.ArchiveRunRequest) (*empty.Empty, error) {
	// TODO(gkcalat): consider validating the experiment id.
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}
	err := s.archiveRun(ctx, request.GetRunId(), request.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to archive a run")
	}
	return &empty.Empty{}, nil
}

// Un-archives a run.
// Supports v2beta1 behavior.
func (s *RunServer) UnarchiveRun(ctx context.Context, request *apiv2beta1.UnarchiveRunRequest) (*empty.Empty, error) {
	// TODO(gkcalat): consider validating the experiment id.
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}
	err := s.unarchiveRun(ctx, request.GetRunId(), request.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to unarchive a run")
	}
	return &empty.Empty{}, nil
}

// Deletes a run.
// Supports v2beta1 behavior.
func (s *RunServer) DeleteRun(ctx context.Context, request *apiv2beta1.DeleteRunRequest) (*empty.Empty, error) {
	// TODO(gkcalat): consider validating the experiment id.
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}
	if err := s.deleteRun(ctx, request.GetRunId(), request.GetExperimentId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete a run")
	}
	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &empty.Empty{}, nil
}

// Reads an artifact.
// Supports v2beta1 behavior.
func (s *RunServer) ReadArtifact(ctx context.Context, request *apiv2beta1.ReadArtifactRequest) (*apiv2beta1.ReadArtifactResponse, error) {
	if s.options.CollectMetrics {
		readArtifactRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadArtifact})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	content, err := s.resourceManager.ReadArtifact(
		request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		return nil, util.Wrapf(err, "failed to read artifact '%+v'", request)
	}
	return &apiv2beta1.ReadArtifactResponse{
		Data: content,
	}, nil
}

// Terminates a run.
// Supports v2beta1 behavior.
func (s *RunServer) TerminateRun(ctx context.Context, request *apiv2beta1.TerminateRunRequest) (*empty.Empty, error) {
	// TODO(gkcalat): consider validating the experiment id.
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}
	err := s.terminateRun(ctx, request.GetRunId(), request.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to terminate a run")
	}
	return &empty.Empty{}, nil
}

// Retries a run.
// Supports v2beta1 behavior.
func (s *RunServer) RetryRun(ctx context.Context, request *apiv2beta1.RetryRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.retryRun(ctx, request.GetRunId(), request.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to retry a run")
	}

	return &empty.Empty{}, nil
}

func (s *RunServer) validateRun(r *model.Run) error {
	if r.DisplayName == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name")
	}
	if r.ExperimentId == "" && common.IsMultiUserMode() {
		return util.NewInvalidInputError("Experiment id can not be empty in run")
	}
	return nil
}

func (s *RunServer) canAccessRun(ctx context.Context, runId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}

	if len(runId) > 0 {
		runDetail, err := s.resourceManager.GetRun(runId)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the run ID")
		}
		if len(resourceAttributes.Namespace) == 0 {
			if len(runDetail.Namespace) == 0 {
				return util.NewInternalServerError(
					errors.New("Empty namespace"),
					"The run doesn't have a valid namespace",
				)
			}
			resourceAttributes.Namespace = runDetail.Namespace
		}
		if len(resourceAttributes.Name) == 0 {
			resourceAttributes.Name = runDetail.K8SName
		}
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeRuns

	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}
