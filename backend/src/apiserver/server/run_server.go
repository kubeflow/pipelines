// Copyright 2018-2022 The Kubeflow Authors
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
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
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

func NewRunServer(resourceManager *resource.ResourceManager, options *RunServerOptions) *RunServer {
	return &RunServer{resourceManager: resourceManager, options: options}
}

func (s *RunServer) CreateRunV1(ctx context.Context, request *apiv1beta1.CreateRunRequest) (*apiv1beta1.RunDetail, error) {
	if s.options.CollectMetrics {
		createRunRequests.Inc()
	}

	err := s.validateCreateRunRequestV1(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}

	// In multi-user mode, verify the user has access to the resources related to this run.
	if common.IsMultiUserMode() {
		// User must provide the experiment ID, which must belong to a namespace the user is authorized with.
		experimentID := common.GetExperimentIDFromAPIResourceReferences(request.Run.ResourceReferences)
		if experimentID == "" {
			return nil, util.NewInvalidInputError("Run has no experiment.")
		}
		namespace, err := s.resourceManager.GetNamespaceFromExperimentID(experimentID)
		if err != nil {
			return nil, util.Wrap(err, "Failed to get namespace for run.")
		}
		if namespace == "" {
			return nil, util.NewInvalidInputError("Run's experiment has no namespace.")
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

	run, err := s.resourceManager.CreateRun(ctx, request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}

	if s.options.CollectMetrics {
		runCount.Inc()
	}
	return ToApiRunDetailV1(run), nil
}

func (s *RunServer) GetRunV1(ctx context.Context, request *apiv1beta1.GetRunRequest) (*apiv1beta1.RunDetail, error) {
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
	return ToApiRunDetailV1(run), nil
}

func (s *RunServer) ListRunsV1(ctx context.Context, request *apiv1beta1.ListRunsRequest) (*apiv1beta1.ListRunsResponse, error) {
	if s.options.CollectMetrics {
		listRunRequests.Inc()
	}

	opts, err := validatedListOptions(&model.Run{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext, err := ValidateFilterV1(request.ResourceReferenceKey)
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
	return &apiv1beta1.ListRunsResponse{Runs: toApiRunsV1(runs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *RunServer) ArchiveRunV1(ctx context.Context, request *apiv1beta1.ArchiveRunRequest) (*empty.Empty, error) {
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

func (s *RunServer) UnarchiveRunV1(ctx context.Context, request *apiv1beta1.UnarchiveRunRequest) (*empty.Empty, error) {
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

func (s *RunServer) DeleteRunV1(ctx context.Context, request *apiv1beta1.DeleteRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.DeleteRun(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) ReportRunMetricsV1(ctx context.Context, request *apiv1beta1.ReportRunMetricsRequest) (*apiv1beta1.ReportRunMetricsResponse, error) {
	if s.options.CollectMetrics {
		reportRunMetricsRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	// Makes sure run exists
	_, err = s.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &apiv1beta1.ReportRunMetricsResponse{
		Results: []*apiv1beta1.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := ValidateRunMetricV1(metric)
		if err == nil {
			err = s.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			NewReportRunMetricResultV1(metric.GetName(), metric.GetNodeId(), err))
	}
	return response, nil
}

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
		return nil, util.Wrapf(err, "failed to read artifact '%+v'.", request)
	}
	return &apiv1beta1.ReadArtifactResponse{
		Data: content,
	}, nil
}

func (s *RunServer) TerminateRunV1(ctx context.Context, request *apiv1beta1.TerminateRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbTerminate})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.TerminateRun(ctx, request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) RetryRunV1(ctx context.Context, request *apiv1beta1.RetryRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbRetry})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.RetryRun(ctx, request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil

}

func (s *RunServer) CreateRun(ctx context.Context, request *apiv2beta1.CreateRunRequest) (*apiv2beta1.Run, error) {
	if s.options.CollectMetrics {
		createRunRequests.Inc()
	}

	err := s.validateCreateRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}

	// In multi-user mode, verify the user has access to the resources related to this run.
	if common.IsMultiUserMode() {
		// User must provide the experiment ID and the namespace. The experiment must belong to the namespace,
		// and user must be authorized with this namespace.
		if request.Run.ExperimentId == "" {
			return nil, util.NewInvalidInputError("Run experiment ID is empty.")
		}
		namespace, err := s.resourceManager.GetNamespaceFromExperimentID(request.Run.ExperimentId)
		if err != nil {
			return nil, util.Wrap(err, "Failed to get namespace for run.")
		}
		if namespace == "" {
			return nil, util.NewInvalidInputError("Run's experiment has no namespace.")
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbCreate,
			Name:      request.Run.DisplayName,
		}
		err = s.canAccessRun(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize the request")
		}
	}

	run, err := s.resourceManager.CreateRun(ctx, request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}

	if s.options.CollectMetrics {
		runCount.Inc()
	}
	return toApiRun(&run.Run), nil

}

func (s *RunServer) GetRun(ctx context.Context, request *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
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
	return toApiRun(&run.Run), nil
}

func (s *RunServer) ListRuns(ctx context.Context, request *apiv2beta1.ListRunsRequest) (*apiv2beta1.ListRunsResponse, error) {
	if s.options.CollectMetrics {
		listRunRequests.Inc()
	}

	opts, err := validatedListOptions(&model.Run{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext := &common.FilterContext{}

	if common.IsMultiUserMode() {
		// In multi-user mode, users must provide the namespace they are authorized with.
		// If the ExperimentId field is empty, then return all recurring runs in this namespace.
		// If the ExperimentId is provided, the experiment must belong to the namespace user is authorized with.

		// Apply Namespace filter.
		if request.Namespace == "" {
			return nil, util.NewInvalidInputError("Invalid ListRuns request. No namespace provided in multi-user mode.")
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: request.Namespace,
			Verb:      common.RbacResourceVerbList,
		}
		err = s.canAccessRun(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API")
		}
		filterContext = &common.FilterContext{
			ReferenceKey: &common.ReferenceKey{Type: common.Namespace, ID: request.Namespace},
		}

		// Apply experiment filter if non-empty.
		if request.ExperimentId != "" {
			// Verify that the requested experiment belongs to this authorized namespace.
			experimentNamespace, err := s.resourceManager.GetNamespaceFromExperimentID(request.ExperimentId)
			if err != nil {
				return nil, util.Wrap(err, "Failed to get namespace of the experiment")
			}
			if experimentNamespace != request.Namespace {
				return nil, util.NewInvalidInputError("Error Listing runs: in multi user mode, experiment filter does not belong to the authorized namespace.")
			}

			filterContext = &common.FilterContext{
				ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: request.ExperimentId},
			}
		}

	} else {
		// In single-user mode, Namespace must be empty.
		if request.Namespace != "" {
			return nil, util.NewInvalidInputError("Invalid ListRuns request. Namespace should not be provided in single-user mode.")
		}
		// Apply experiment filter.
		if request.ExperimentId != "" {
			filterContext = &common.FilterContext{
				ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: request.ExperimentId},
			}
		}
	}

	runs, total_size, nextPageToken, err := s.resourceManager.ListRuns(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs.")
	}

	return &apiv2beta1.ListRunsResponse{Runs: toApiRuns(runs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil

}

func (s *RunServer) ArchiveRun(ctx context.Context, request *apiv2beta1.ArchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		archiveRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbArchive})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.ArchiveRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) UnarchiveRun(ctx context.Context, request *apiv2beta1.UnarchiveRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.UnarchiveRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) DeleteRun(ctx context.Context, request *apiv2beta1.DeleteRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.DeleteRun(ctx, request.RunId)
	if err != nil {
		return nil, err
	}

	if s.options.CollectMetrics {
		runCount.Dec()
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) ReportRunMetrics(ctx context.Context, request *apiv2beta1.ReportRunMetricsRequest) (*apiv2beta1.ReportRunMetricsResponse, error) {
	if s.options.CollectMetrics {
		reportRunMetricsRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReportMetrics})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	// Makes sure run exists
	_, err = s.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &apiv2beta1.ReportRunMetricsResponse{
		Results: []*apiv2beta1.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := ValidateRunMetric(metric)
		if err == nil {
			err = s.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			NewReportRunMetricResult(metric.GetDisplayName(), metric.GetNodeId(), err))
	}
	return response, nil
}

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
		return nil, util.Wrapf(err, "failed to read artifact '%+v'.", request)
	}
	return &apiv2beta1.ReadArtifactResponse{
		Data: content,
	}, nil
}

func (s *RunServer) TerminateRun(ctx context.Context, request *apiv2beta1.TerminateRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		terminateRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbTerminate})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.TerminateRun(ctx, request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) RetryRun(ctx context.Context, request *apiv2beta1.RetryRunRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		retryRunRequests.Inc()
	}

	err := s.canAccessRun(ctx, request.RunId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbRetry})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	err = s.resourceManager.RetryRun(ctx, request.RunId)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *RunServer) validateCreateRunRequestV1(request *apiv1beta1.CreateRunRequest) error {
	run := request.Run
	if run.Name == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}
	return ValidatePipelineSpecAndResourceReferences(s.resourceManager, run.PipelineSpec, run.ResourceReferences)
}

func (s *RunServer) validateCreateRunRequest(request *apiv2beta1.CreateRunRequest) error {
	run := request.Run
	if run.DisplayName == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}
	return ValidatePipelineSource(s.resourceManager, run.GetPipelineId(), run.GetPipelineSpec())

}

func (s *RunServer) canAccessRun(ctx context.Context, runId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if common.IsMultiUserMode() == false {
		// Skip authz if not multi-user mode.
		return nil
	}

	if len(runId) > 0 {
		runDetail, err := s.resourceManager.GetRun(runId)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the run ID.")
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
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}
