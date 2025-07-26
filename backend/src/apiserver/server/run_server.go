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

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/golang/glog"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
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

	runCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "run_server_run_count",
		Help: "The current number of runs in Kubeflow Pipelines instance",
	})
)

type RunServerOptions struct {
	CollectMetrics bool `json:"collect_metrics,omitempty"`
}

// BaseRunServer wraps RunServer and RunServerV1
// to enable method sharing. It can be removed once RunServerV1
// is removed.
type BaseRunServer struct {
	resourceManager *resource.ResourceManager
	options         *RunServerOptions
}

type RunServer struct {
	*BaseRunServer
	apiv2beta1.UnimplementedRunServiceServer
}

type RunServerV1 struct {
	*BaseRunServer
	apiv1beta1.UnimplementedRunServiceServer
}

func NewRunServer(resourceManager *resource.ResourceManager, options *RunServerOptions) *RunServer {
	return &RunServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}

func NewRunServerV1(resourceManager *resource.ResourceManager, options *RunServerOptions) *RunServerV1 {
	return &RunServerV1{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}

// Creates a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) createRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	// Validate user inputs
	if run.DisplayName == "" {
		return nil, util.Wrapf(util.NewInvalidInputError("The run name is empty. Please specify a valid name"), "Failed to create a run due to invalid name")
	}
	experimentId, namespace, err := s.resourceManager.GetValidExperimentNamespacePair(run.ExperimentId, run.Namespace)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a run due to invalid experimentId and namespace combination")
	}
	if common.IsMultiUserMode() && namespace == "" {
		return nil, util.NewInvalidInputError("A run cannot have an empty namespace in multi-user mode")
	}
	run.ExperimentId = experimentId
	run.Namespace = namespace
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: run.Namespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	if err := s.canAccessRun(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to create a run due to authorization error. Check if you have write permissions to namespace %s", run.Namespace)
	}
	return s.resourceManager.CreateRun(ctx, run)
}

// Creates a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) CreateRunV1(ctx context.Context, request *apiv1beta1.CreateRunRequest) (*apiv1beta1.RunDetail, error) {
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
func (s *BaseRunServer) getRun(ctx context.Context, runId string) (*model.Run, error) {
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
func (s *RunServerV1) GetRunV1(ctx context.Context, request *apiv1beta1.GetRunRequest) (*apiv1beta1.RunDetail, error) {
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
func (s *BaseRunServer) listRuns(ctx context.Context, pageToken string, pageSize int, sortBy string, opts *list.Options, namespace string, experimentId string) ([]*model.Run, int, string, error) {
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

	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}
	if experimentId != "" {
		if err := s.resourceManager.CheckExperimentBelongsToNamespace(experimentId, namespace); err != nil {
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
func (s *RunServerV1) ListRunsV1(ctx context.Context, r *apiv1beta1.ListRunsRequest) (*apiv1beta1.ListRunsResponse, error) {
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

	opts, err := validatedListOptions(&model.Run{}, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), "v1beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	runs, runsCount, nextPageToken, err := s.listRuns(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), opts, namespace, experimentId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list v1beta1 runs")
	}
	apiRuns := toApiRunsV1(runs)
	if apiRuns == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal run representations to their v1beta1 API counterparts"), "Failed to list v1beta1 runs")
	}
	return &apiv1beta1.ListRunsResponse{
		Runs:          apiRuns,
		TotalSize:     int32(runsCount),
		NextPageToken: nextPageToken,
	}, nil
}

// Archives a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) archiveRun(ctx context.Context, runId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbArchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ArchiveRun(runId)
}

// Archives a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) ArchiveRunV1(ctx context.Context, request *apiv1beta1.ArchiveRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}
	err := s.archiveRun(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to archive a v1beta1 run")
	}
	return &emptypb.Empty{}, nil
}

// Un-archives a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) unarchiveRun(ctx context.Context, runId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.UnarchiveRun(runId)
}

// Un-archives a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) UnarchiveRunV1(ctx context.Context, request *apiv1beta1.UnarchiveRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}
	err := s.unarchiveRun(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to unarchive a v1beta1 run")
	}
	return &emptypb.Empty{}, nil
}

// Deletes a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) deleteRun(ctx context.Context, runId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.DeleteRun(ctx, runId)
}

// Deletes a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) DeleteRunV1(ctx context.Context, request *apiv1beta1.DeleteRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}
	if err := s.deleteRun(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete a v1beta1 run")
	}
	if s.options.CollectMetrics {
		if util.GetMetricValue(runCount) > 0 {
			runCount.Dec()
		}
	}
	return &emptypb.Empty{}, nil
}

// Reports run metrics.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) reportRunMetrics(ctx context.Context, metrics []*model.RunMetric, runId string) ([]map[string]string, error) {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	// Verify that the run exists for single user mode.
	// Multi-user model will verify this when checking authorization above.
	if !common.IsMultiUserMode() {
		if _, err := s.resourceManager.GetRun(runId); err != nil {
			return nil, util.Wrap(err, "Failed to fetch the requested run")
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
func (s *RunServerV1) ReportRunMetricsV1(ctx context.Context, request *apiv1beta1.ReportRunMetricsRequest) (*apiv1beta1.ReportRunMetricsResponse, error) {
	if s.options.CollectMetrics {
		reportRunMetricsRequests.Inc()
	}

	if _, err := s.resourceManager.GetRun(request.GetRunId()); err != nil {
		// Use the standard ResourceNotFoundError so that AssertUserError
		// sees codes.NotFound and the right error message.
		return nil, util.NewResourceNotFoundError(
			"Run %s not found", request.GetRunId(),
		)
	}

	// Convert, validate, and report each metric in input order.
	var apiResults []*apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult
	for _, m := range request.GetMetrics() {
		modelMetric, err := toModelRunMetric(m, request.GetRunId())
		if err != nil {
			// Conversion error: record as INVALID_ARGUMENT
			msg := err.Error()
			if userErr, ok := err.(*util.UserError); ok {
				msg = userErr.ExternalMessage()
			}
			apiResults = append(apiResults, toApiReportMetricsResultV1(
				m.Name, m.NodeId, "invalid", msg,
			))
			continue
		}
		// Report this metric
		results, err := s.reportRunMetrics(ctx, []*model.RunMetric{modelMetric}, request.GetRunId())
		if err != nil {
			return nil, util.Wrap(err, "Failed to report v1beta1 run metrics")
		}
		// results slice will have exactly one entry
		r := results[0]
		apiResults = append(apiResults, toApiReportMetricsResultV1(
			r["Name"], r["NodeId"], r["ErrorCode"], r["ErrorMessage"],
		))
	}
	return &apiv1beta1.ReportRunMetricsResponse{Results: apiResults}, nil
}

// Reads an artifact.
// Supports v1beta1 behavior.
func (s *RunServerV1) ReadArtifactV1(ctx context.Context, request *apiv1beta1.ReadArtifactRequest) (*apiv1beta1.ReadArtifactResponse, error) {
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
func (s *BaseRunServer) terminateRun(ctx context.Context, runId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbTerminate})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.TerminateRun(ctx, runId)
}

// Retries a run.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *BaseRunServer) retryRun(ctx context.Context, runId string) error {
	err := s.canAccessRun(ctx, runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbRetry})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.RetryRun(ctx, runId)
}

// Terminates a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) TerminateRunV1(ctx context.Context, request *apiv1beta1.TerminateRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}
	err := s.terminateRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to terminate a v1beta1 run")
	}
	return &emptypb.Empty{}, nil
}

// Retries a run.
// Supports v1beta1 behavior.
func (s *RunServerV1) RetryRunV1(ctx context.Context, request *apiv1beta1.RetryRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.retryRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to retry a run")
	}

	return &emptypb.Empty{}, nil
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
	opts, err := validatedListOptions(&model.Run{}, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}
	runs, runsCount, nextPageToken, err := s.listRuns(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), opts, r.GetNamespace(), r.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs")
	}
	return &apiv2beta1.ListRunsResponse{Runs: toApiRuns(runs), TotalSize: int32(runsCount), NextPageToken: nextPageToken}, nil
}

// Archives a run.
// Supports v2beta1 behavior.
func (s *RunServer) ArchiveRun(ctx context.Context, request *apiv2beta1.ArchiveRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}
	err := s.archiveRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to archive a run")
	}
	return &emptypb.Empty{}, nil
}

// Un-archives a run.
// Supports v2beta1 behavior.
func (s *RunServer) UnarchiveRun(ctx context.Context, request *apiv2beta1.UnarchiveRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}
	err := s.unarchiveRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to unarchive a run")
	}
	return &emptypb.Empty{}, nil
}

// Deletes a run.
// Supports v2beta1 behavior.
func (s *RunServer) DeleteRun(ctx context.Context, request *apiv2beta1.DeleteRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}
	if err := s.deleteRun(ctx, request.GetRunId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete a run")
	}
	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &emptypb.Empty{}, nil
}

// Reads an artifact.
// Supports v2beta1 behavior.
func (s *RunServer) ReadArtifact(ctx context.Context, request *apiv2beta1.ReadArtifactRequest) (*apiv2beta1.ReadArtifactResponse, error) {
	if s.options.CollectMetrics {
		readArtifactRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.GetRunId(), &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadArtifact})
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
func (s *RunServer) TerminateRun(ctx context.Context, request *apiv2beta1.TerminateRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}
	err := s.terminateRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to terminate a run")
	}
	return &emptypb.Empty{}, nil
}

// Retries a run.
// Supports v2beta1 behavior.
func (s *RunServer) RetryRun(ctx context.Context, request *apiv2beta1.RetryRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.retryRun(ctx, request.GetRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to retry a run")
	}

	return &emptypb.Empty{}, nil
}

// Checks if a user can access a run.
// Adds namespace of the parent experiment of a run id,
// API group, version, and resource type.
func (s *BaseRunServer) canAccessRun(ctx context.Context, runId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}
	if runId != "" {
		run, err := s.resourceManager.GetRun(runId)
		if err != nil {
			return util.Wrapf(err, "Failed to authorize with the run ID %v", runId)
		}
		if s.resourceManager.IsEmptyNamespace(run.Namespace) {
			experiment, err := s.resourceManager.GetExperiment(run.ExperimentId)
			if err != nil {
				return util.NewInvalidInputError("run %v has an empty namespace and the parent experiment %v could not be fetched: %s", runId, run.ExperimentId, err.Error())
			}
			resourceAttributes.Namespace = experiment.Namespace
		} else {
			resourceAttributes.Namespace = run.Namespace
		}
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = run.K8SName
		}
	}
	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return util.NewInvalidInputError("A run cannot have an empty namespace in multi-user mode")
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeRuns
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to access run %s. Check if you have access to namespace %s", runId, resourceAttributes.Namespace)
	}
	return nil
}
