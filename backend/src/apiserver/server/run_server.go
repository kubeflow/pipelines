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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	return s.getRunWithHydration(ctx, runId, false)
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
	return s.listRunsWithHydration(ctx, pageToken, pageSize, sortBy, opts, namespace, experimentId, true)
}

func (s *BaseRunServer) listRunsWithHydration(ctx context.Context, pageToken string, pageSize int, sortBy string, opts *list.Options, namespace string, experimentID string, hydrateTasks bool) ([]*model.Run, int, string, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if experimentID != "" {
		ns, err := s.resourceManager.GetNamespaceFromExperimentId(experimentID)
		if err != nil {
			return nil, 0, "", util.Wrapf(err, "Failed to list runs due to error fetching namespace for experiment %s. Try filtering based on namespace", experimentID)
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
	if experimentID != "" {
		if err := s.resourceManager.CheckExperimentBelongsToNamespace(experimentID, namespace); err != nil {
			return nil, 0, "", util.Wrap(err, "Failed to list runs due to namespace mismatch")
		}
		filterContext = &model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: experimentID},
		}
	}
	runs, totalSize, token, err := s.resourceManager.ListRunsWithHydration(filterContext, opts, hydrateTasks)
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
func (s *BaseRunServer) reportRunMetricsV1(ctx context.Context, metrics []*model.RunMetricV1, runID string) ([]map[string]string, error) {
	err := s.canAccessRun(ctx, runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	// Verify that the run exists for single user mode.
	// Multi-user model will verify this when checking authorization above.
	if !common.IsMultiUserMode() {
		if _, err := s.resourceManager.GetRun(runID); err != nil {
			return nil, util.Wrap(err, "Failed to fetch the requested run")
		}
	}
	results := make([]map[string]string, 0)
	for _, metric := range metrics {
		temp := map[string]string{"Name": metric.Name, "NodeId": metric.NodeID, "ErrorCode": "", "ErrorMessage": ""}
		if err := validateRunMetricV1(metric); err != nil {
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

// ReportRunMetricsV1 reports run metrics.
// Supports v1beta1 API. Deprecated.
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
		modelMetric, err := toModelRunMetricV1(m, request.GetRunId())
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
		results, err := s.reportRunMetricsV1(ctx, []*model.RunMetricV1{modelMetric}, request.GetRunId())
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

	// Determine if we should hydrate tasks based on view parameter
	// Default view (or unspecified) means no task hydration, only task count
	// FULL view means full task hydration
	hydrateTasks := request.View != nil && *request.View == apiv2beta1.GetRunRequest_FULL

	run, err := s.getRunWithHydration(ctx, request.RunId, hydrateTasks)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get a run")
	}

	return toApiRun(run), nil
}

func (s *BaseRunServer) getRunWithHydration(ctx context.Context, runID string, hydrateTasks bool) (*model.Run, error) {
	err := s.canAccessRun(ctx, runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	run, err := s.resourceManager.GetRunWithHydration(runID, hydrateTasks)
	if err != nil {
		return nil, err
	}
	return run, nil
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

	// Determine if we should hydrate tasks based on view parameter
	// Default view (or unspecified) means no task hydration, only task count
	// FULL view means full task hydration
	hydrateTasks := r.View != nil && *r.View == apiv2beta1.ListRunsRequest_FULL

	runs, runsCount, nextPageToken, err := s.listRunsWithHydration(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), opts, r.GetNamespace(), r.GetExperimentId(), hydrateTasks)
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

// CreateTask Creates an API Task
func (s *RunServer) CreateTask(ctx context.Context, request *apiv2beta1.CreateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	task := request.GetTask()
	if task == nil {
		return nil, util.NewInvalidInputError("Task is required")
	}

	// Check authorization - Tasks inherit permissions from their parent run
	err := s.canAccessRun(ctx, task.GetRunId(), &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize task creation")
	}

	modelTask, err := toModelTask(task)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert task to model")
	}
	createdTask, err := s.resourceManager.CreateTask(modelTask)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create task")
	}

	// A newly created task has no children
	var noChildTasks []*model.Task

	return toAPITask(createdTask, noChildTasks)
}

// UpdateTask updates an existing task with the specified task ID and details provided in the request.
// It validates input, ensures authorization, and returns the updated task details or an error if the update fails.
func (s *RunServer) UpdateTask(ctx context.Context, request *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	taskID := request.GetTaskId()
	task := request.GetTask()
	if taskID == "" {
		return nil, util.NewInvalidInputError("Task ID is required")
	}
	if task == nil {
		return nil, util.NewInvalidInputError("Task is required")
	}
	// Ensure task IDs match - prefer the path parameter for authorization
	if task.GetTaskId() != "" && task.GetTaskId() != taskID {
		return nil, util.NewInvalidInputError("Task ID in path parameter does not match task ID in request body")
	}

	// First get the existing task to find the run UUID for authorization
	existingTask, err := s.resourceManager.GetTask(taskID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get existing task for authorization")
	}

	// Check authorization using the existing task's run UUID
	err = s.canAccessRun(ctx, existingTask.RunUUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize task update")
	}

	modelTask, err := toModelTask(task)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert task to model")
	}
	modelTask.UUID = taskID // Always use the path parameter task ID
	updatedTask, err := s.resourceManager.UpdateTask(modelTask)
	if err != nil {
		return nil, util.Wrap(err, "Failed to update task")
	}

	taskChildren, err := s.resourceManager.GetTaskChildren(updatedTask.UUID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get task children")
	}
	return toAPITask(updatedTask, taskChildren)
}

// UpdateTasksBulk updates multiple tasks in bulk.
func (s *RunServer) UpdateTasksBulk(ctx context.Context, request *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	if request == nil || len(request.GetTasks()) == 0 {
		return nil, util.NewInvalidInputError("UpdateTasksBulkRequest must contain at least one task")
	}

	response := &apiv2beta1.UpdateTasksBulkResponse{
		Tasks: make(map[string]*apiv2beta1.PipelineTaskDetail),
	}

	// Validate and update each task
	for taskID, task := range request.GetTasks() {
		if taskID == "" {
			return nil, util.NewInvalidInputError("Task ID is required")
		}
		if task == nil {
			return nil, util.NewInvalidInputError("Task is required for task ID %s", taskID)
		}

		// Ensure task IDs match - prefer the map key for authorization
		if task.GetTaskId() != "" && task.GetTaskId() != taskID {
			return nil, util.NewInvalidInputError("Task ID in map key does not match task ID in task detail for task %s", taskID)
		}

		// First get the existing task to find the run UUID for authorization
		existingTask, err := s.resourceManager.GetTask(taskID)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to get existing task %s for authorization", taskID)
		}

		// Check authorization using the existing task's run UUID
		err = s.canAccessRun(ctx, existingTask.RunUUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate})
		if err != nil {
			return nil, util.Wrapf(err, "Failed to authorize task update for task %s", taskID)
		}

		modelTask, err := toModelTask(task)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to convert task to model for task %s", taskID)
		}
		modelTask.UUID = taskID // Always use the map key task ID

		updatedTask, err := s.resourceManager.UpdateTask(modelTask)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to update task %s", taskID)
		}

		taskChildren, err := s.resourceManager.GetTaskChildren(updatedTask.UUID)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to get task children for task %s", taskID)
		}

		apiTask, err := toAPITask(updatedTask, taskChildren)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to convert task to API for task %s", taskID)
		}
		response.Tasks[taskID] = apiTask
	}

	return response, nil
}

// GetTask retrieves the details of a specific task based on its ID and performs authorization checks.
func (s *RunServer) GetTask(ctx context.Context, request *apiv2beta1.GetTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	taskID := request.GetTaskId()
	if taskID == "" {
		return nil, util.NewInvalidInputError("Task ID is required")
	}

	task, err := s.resourceManager.GetTask(taskID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get task")
	}

	// Check authorization using the task's run UUID
	err = s.canAccessRun(ctx, task.RunUUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize task access")
	}

	childTasks, err := s.resourceManager.GetTaskChildren(task.UUID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get task children")
	}
	return toAPITask(task, childTasks)
}

// ListTasks retrieves a list of tasks based on a specified run ID, parent task ID, or namespace, enforcing mutual exclusivity.
// It validates authorization, processes pagination options, and ensures namespace consistency within the data.
func (s *RunServer) ListTasks(ctx context.Context, request *apiv2beta1.ListTasksRequest) (*apiv2beta1.ListTasksResponse, error) {
	// Check which filter is set using the oneof field
	// This allows empty namespace in single-user mode
	var runID, parentID, namespace string
	var filterType string

	switch filter := request.ParentFilter.(type) {
	case *apiv2beta1.ListTasksRequest_RunId:
		runID = filter.RunId
		filterType = "run_id"
	case *apiv2beta1.ListTasksRequest_ParentId:
		parentID = filter.ParentId
		filterType = "parent_id"
	case *apiv2beta1.ListTasksRequest_Namespace:
		namespace = filter.Namespace
		filterType = "namespace"
	default:
		// One of these fields is required to enforce RBAC on this request in multi-user mode.
		// In the case of run IDs, we use the associated run's namespace to enforce RBAC.
		// In the namespace case, we use the namespace to enforce RBAC.
		if common.IsMultiUserMode() {
			return nil, util.NewInvalidInputError("Either run_id, parent_id, or namespace is required in multi-user mode")
		}
		// In single-user mode, allow the namespace to be empty.
		filterType = "namespace"
	}

	// Check authorization and get expected namespace based on filter type
	switch filterType {
	case "run_id":
		err := s.canAccessRun(ctx, runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbList})
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize task listing")
		}
	case "parent_id":
		// parent_id is provided, get the parent task to find the run_id and namespace
		parentTask, err := s.resourceManager.GetTask(parentID)
		if err != nil {
			return nil, util.Wrap(err, "Failed to get parent task for authorization")
		}
		err = s.canAccessRun(ctx, parentTask.RunUUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbList})
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize task listing")
		}
	case "namespace":
		// For namespace filtering, check if user has get permission on runs in this namespace
		if common.IsMultiUserMode() {
			if namespace == "" {
				return nil, util.NewInvalidInputError("Namespace is required in multi-user mode")
			}
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbGet,
				Group:     common.RbacPipelinesGroup,
				Version:   common.RbacPipelinesVersion,
				Resource:  common.RbacResourceTypeRuns,
			}
			err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
			if err != nil {
				return nil, util.Wrapf(err, "Failed to authorize task listing by namespace. Check if you have access to runs in namespace %s", namespace)
			}
		} else {
			// It doesn't matter if users specify a namespace in single-user mode, namespaces are not set in this mode
			// so we just list everything.
			namespace = ""
		}
	}

	opts, err := validatedListOptions(&model.Task{}, request.GetPageToken(), int(request.GetPageSize()), request.GetOrderBy(), request.GetFilter(), "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	// Pass namespaceSet=true when namespace filter was explicitly set
	tasks, totalSize, nextPageToken, err := s.resourceManager.ListTasks(runID, parentID, namespace, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list tasks")
	}

	apiTasks := make([]*apiv2beta1.PipelineTaskDetail, len(tasks))
	for i, task := range tasks {
		taskChildren, err := s.resourceManager.GetTaskChildren(task.UUID)
		if err != nil {
			return nil, util.Wrap(err, "Failed to get task children")
		}
		apiTasks[i], err = toAPITask(task, taskChildren)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert task to API")
		}
	}

	return &apiv2beta1.ListTasksResponse{
		Tasks:         apiTasks,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}, nil
}

// canAccessRun verifies if the current user has access to a specified run utilizing the provided resource attributes.
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
