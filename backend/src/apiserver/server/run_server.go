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

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
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

	// TODO(jingzhang36): error count and success count.

	runCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "run_server_run_count",
		Help: "The current number of runs in Kubeflow Pipelines instance",
	})
)

type RunServerOptions struct {
	CollectMetrics bool
}

type RunServer struct {
	resourceManager *resource.ResourceManager
	options         *RunServerOptions
}

func (s *RunServer) CreateRun(ctx context.Context, request *api.CreateRunRequest) (*api.RunDetail, error) {
	if s.options.CollectMetrics {
		createRunRequests.Inc()
	}

	err := s.validateCreateRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}

	if common.IsMultiUserMode() {
		experimentID := common.GetExperimentIDFromAPIResourceReferences(request.Run.ResourceReferences)
		if experimentID == "" {
			return nil, util.NewInvalidInputError("Job has no experiment.")
		}
		namespace, err := s.resourceManager.GetNamespaceFromExperimentID(experimentID)
		if err != nil {
			return nil, util.Wrap(err, "Failed to get experiment for job.")
		}
		if namespace == "" {
			return nil, util.NewInvalidInputError("Job's experiment has no namespace.")
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbCreate,
			Name:      request.Run.Name,
		}
		err = s.canAccessRun(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize the request")
		}
	}

	run, err := s.resourceManager.CreateRun(request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}

	if s.options.CollectMetrics {
		runCount.Inc()
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) GetRun(ctx context.Context, request *api.GetRunRequest) (*api.RunDetail, error) {
	if s.options.CollectMetrics {
		getRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	run, err := s.resourceManager.GetRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) ListRuns(ctx context.Context, request *api.ListRunsRequest) (*api.ListRunsResponse, error) {
	if s.options.CollectMetrics {
		listRunRequests.Inc()
	}

	opts, err := validatedListOptions(&model.Run{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}

	if common.IsMultiUserMode() {
		refKey := filterContext.ReferenceKey
		if refKey == nil {
			return nil, util.NewInvalidInputError("ListRuns must filter by resource reference in multi-user mode.")
		}
		if refKey.Type == common.Namespace {
			namespace := refKey.ID
			if len(namespace) == 0 {
				return nil, util.NewInvalidInputError("Invalid resource references for ListRuns. Namespace is empty.")
			}
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbList,
			}
			err = s.canAccessRun(ctx, "", resourceAttributes)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with namespace resource reference.")
			}
		} else if refKey.Type == common.Experiment || refKey.Type == "ExperimentUUID" {
			// "ExperimentUUID" was introduced for perf optimization. We accept both refKey.Type for backward-compatible reason.
			experimentID := refKey.ID
			if len(experimentID) == 0 {
				return nil, util.NewInvalidInputError("Invalid resource references for run. Experiment ID is empty.")
			}
			namespace, err := s.resourceManager.GetNamespaceFromExperimentID(experimentID)
			if err != nil {
				return nil, util.Wrap(err, "Run's experiment has no namespace.")
			}
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbList,
			}
			err = s.canAccessRun(ctx, "", resourceAttributes)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with namespace in experiment resource reference.")
			}
		} else {
			return nil, util.NewInvalidInputError("Invalid resource references for ListRuns. Got %+v", request.ResourceReferenceKey)
		}
	}

	runs, total_size, nextPageToken, err := s.resourceManager.ListRuns(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs.")
	}
	return &api.ListRunsResponse{Runs: ToApiRuns(runs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *RunServer) ArchiveRun(ctx context.Context, request *api.ArchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbArchive})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.ArchiveRun(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) UnarchiveRun(ctx context.Context, request *api.UnarchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.UnarchiveRun(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) DeleteRun(ctx context.Context, request *api.DeleteRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.DeleteRun(request.Id)
	if err != nil {
		return nil, err
	}

	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) ReportRunMetrics(ctx context.Context, request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	if s.options.CollectMetrics {
		reportRunMetricsRequests.Inc()
	}

	// Makes sure run exists
	_, err := s.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := ValidateRunMetric(metric)
		if err == nil {
			err = s.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			NewReportRunMetricResult(metric.GetName(), metric.GetNodeId(), err))
	}
	return response, nil
}

func (s *RunServer) ReadArtifact(ctx context.Context, request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	if s.options.CollectMetrics {
		readArtifactRequests.Inc()
	}

	content, err := s.resourceManager.ReadArtifact(
		request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		return nil, util.Wrapf(err, "failed to read artifact '%+v'.", request)
	}
	return &api.ReadArtifactResponse{
		Data: content,
	}, nil
}

func (s *RunServer) validateCreateRunRequest(request *api.CreateRunRequest) error {
	run := request.Run
	if run.Name == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}
	return ValidatePipelineSpecAndResourceReferences(s.resourceManager, run.PipelineSpec, run.ResourceReferences)
}

func (s *RunServer) TerminateRun(ctx context.Context, request *api.TerminateRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbTerminate})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.TerminateRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) RetryRun(ctx context.Context, request *api.RetryRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbRetry})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.RetryRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil

}

func (s *RunServer) canAccessRun(ctx context.Context, runId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if common.IsMultiUserMode() == false {
		// Skip authz if not multi-user mode.
		return nil
	}

	if len(runId) > 0 {
		runDetail, err := s.resourceManager.GetRun(runId)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the experiment ID.")
		}
		if len(resourceAttributes.Namespace) == 0 {
			if len(runDetail.Namespace) == 0 {
				return util.NewInternalServerError(
					errors.New("Empty namespace"),
					"The run doesn't have a valid namespace.",
				)
			}
			resourceAttributes.Namespace = runDetail.Namespace
		}
		if len(resourceAttributes.Name) == 0 {
			resourceAttributes.Name = runDetail.Name
		}
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeRuns

	err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}

func NewRunServer(resourceManager *resource.ResourceManager, options *RunServerOptions) *RunServer {
	return &RunServer{resourceManager: resourceManager, options: options}
}
