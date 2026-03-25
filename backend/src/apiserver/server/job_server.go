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
	"time"

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

// Metric variables. Please prefix the metric names with job_server_.
var (
	// Used to calculate the request rate.
	createJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_create_requests",
		Help: "The total number of CreateJob requests",
	})

	getJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_get_requests",
		Help: "The total number of GetJob requests",
	})

	listJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_list_requests",
		Help: "The total number of ListJobs requests",
	})

	deleteJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_delete_requests",
		Help: "The total number of DeleteJob requests",
	})

	disableJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_disable_requests",
		Help: "The total number of DisableJob requests",
	})

	enableJobRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "job_server_enable_requests",
		Help: "The total number of EnableJob requests",
	})

	jobCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "job_server_job_count",
		Help: "The current number of jobs in Kubeflow Pipelines instance",
	})

	jobServerRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "job_server_requests_total",
		Help: "Total number of requests in the job server, labeled by operation and result",
	}, []string{"operation", "result"})

	jobServerRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "job_server_request_duration_seconds",
		Help:    "Latency of requests in the job server in seconds, labeled by operation",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
)

type JobServerOptions struct {
	CollectMetrics bool
}

// BaseJobServer wraps JobServer and JobServerV1
// to enable method sharing. It can be removed once JobServerV1
// is removed.
type BaseJobServer struct {
	resourceManager *resource.ResourceManager
	options         *JobServerOptions
}

type JobServer struct {
	*BaseJobServer
	apiv2beta1.UnimplementedRecurringRunServiceServer
}

type JobServerV1 struct {
	*BaseJobServer
	apiv1beta1.UnimplementedJobServiceServer
}

func (s *BaseJobServer) recordMetrics(operation string, start time.Time, err *error) func() {
	return func() {
		result := "success"
		if *err != nil {
			result = "error"
		}
		jobServerRequests.WithLabelValues(operation, result).Inc()
		jobServerRequestDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	}
}

func (s *BaseJobServer) createJob(ctx context.Context, job *model.Job) (*model.Job, error) {
	// Validate user inputs
	if job.DisplayName == "" {
		return nil, util.NewInvalidInputError("Recurring run name is empty. Please specify a valid name")
	}
	experimentId, namespace, err := s.resourceManager.GetValidExperimentNamespacePair(job.ExperimentId, job.Namespace)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a recurring run due to invalid experimentId and namespace combination")
	}
	if common.IsMultiUserMode() && namespace == "" {
		return nil, util.NewInvalidInputError("Recurring run cannot have an empty namespace in multi-user mode")
	}
	job.ExperimentId = experimentId
	job.Namespace = namespace
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: job.Namespace,
		Verb:      common.RbacResourceVerbCreate,
		Name:      job.DisplayName,
	}
	if err := s.canAccessJob(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to create a recurring run due to authorization error. Check if you have write permission to namespace %s", job.Namespace)
	}
	return s.resourceManager.CreateJob(ctx, job)
}

func (s *JobServerV1) CreateJob(ctx context.Context, request *apiv1beta1.CreateJobRequest) (_ *apiv1beta1.Job, err error) {
	if s.options.CollectMetrics {
		createJobRequests.Inc()
		defer s.recordMetrics("create", time.Now(), &err)()
	}

	modelJob, err := toModelJob(request.GetJob())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run due to conversion error")
	}

	// In the v2 API, the pipeline version being empty means always pick the latest at run submission time. In v1,
	// it means to use the latest pipeline version at recurring run creation time. Handle this case here since
	// modelJob does not have the concept of which API version it came from.
	if modelJob.PipelineId != "" && modelJob.WorkflowSpecManifest == "" && modelJob.PipelineSpecManifest == "" && modelJob.PipelineVersionId == "" {
		pipelineVersion, err := s.resourceManager.GetLatestPipelineVersion(modelJob.PipelineId)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to fetch a pipeline version from pipeline %v", modelJob.PipelineId)
		}

		modelJob.PipelineVersionId = pipelineVersion.UUID
	}

	newJob, err := s.createJob(ctx, modelJob)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run")
	}

	if s.options.CollectMetrics {
		jobCount.Inc()
	}
	return toApiJobV1(newJob), nil
}

func (s *BaseJobServer) getJob(ctx context.Context, jobId string) (*model.Job, error) {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.GetJob(jobId)
}

func (s *JobServerV1) GetJob(ctx context.Context, request *apiv1beta1.GetJobRequest) (_ *apiv1beta1.Job, err error) {
	if s.options.CollectMetrics {
		getJobRequests.Inc()
		defer s.recordMetrics("get", time.Now(), &err)()
	}

	recurringRun, err := s.getJob(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch a v1beta1 recurring run")
	}

	apiJob := toApiJobV1(recurringRun)
	if apiJob == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal recurring run representation to its v1beta1 API counterpart"), "Failed to fetch a v1beta1 recurring run")
	}
	return apiJob, nil
}

func (s *BaseJobServer) listJobs(ctx context.Context, pageToken string, pageSize int, sortBy string, opts *list.Options, namespace string, experimentId string) ([]*model.Job, int, string, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if experimentId != "" {
		ns, err := s.resourceManager.GetNamespaceFromExperimentId(experimentId)
		if err != nil {
			return nil, 0, "", util.Wrapf(err, "Failed to list recurring runs due to error fetching namespace for experiment %s. Try filtering based on namespace", experimentId)
		}
		namespace = ns
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	err := s.canAccessJob(ctx, "", resourceAttributes)
	if err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list recurring runs due to authorization error. Check if you have permission to access namespace %s", namespace)
	}

	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}
	if experimentId != "" {
		if err := s.resourceManager.CheckExperimentBelongsToNamespace(experimentId, namespace); err != nil {
			return nil, 0, "", util.Wrap(err, "Failed to list recurring runs due to namespace mismatch")
		}
		filterContext = &model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: experimentId},
		}
	}
	jobs, totalSize, token, err := s.resourceManager.ListJobs(filterContext, opts)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "Failed to list recurring runs")
	}
	return jobs, totalSize, token, nil
}

func (s *JobServerV1) ListJobs(ctx context.Context, r *apiv1beta1.ListJobsRequest) (_ *apiv1beta1.ListJobsResponse, err error) {
	if s.options.CollectMetrics {
		listJobRequests.Inc()
		defer s.recordMetrics("list", time.Now(), &err)()
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

	opts, err := validatedListOptions(&model.Job{}, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), "v1beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to list jobs due to error parsing the listing options")
	}

	jobs, total_size, nextPageToken, err := s.listJobs(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), opts, namespace, experimentId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list jobs")
	}
	apiJobs := toApiJobsV1(jobs)
	if apiJobs == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal recurring run representations to their v1beta1 API counterparts"), "Failed to list v1beta1 recurring runs")
	}
	return &apiv1beta1.ListJobsResponse{
		Jobs:          apiJobs,
		TotalSize:     int32(total_size),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *JobServerV1) EnableJob(ctx context.Context, request *apiv1beta1.EnableJobRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		enableJobRequests.Inc()
		defer s.recordMetrics("enable", time.Now(), &err)()
	}
	err = s.enableJob(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to enable a v1beta1 recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *BaseJobServer) disableJob(ctx context.Context, jobId string) error {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDisable})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ChangeJobMode(ctx, jobId, false)
}

func (s *JobServerV1) DisableJob(ctx context.Context, request *apiv1beta1.DisableJobRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		disableJobRequests.Inc()
		defer s.recordMetrics("disable", time.Now(), &err)()
	}

	err = s.disableJob(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to disable a v1beta1 recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *BaseJobServer) deleteJob(ctx context.Context, jobID string, propagationPolicy apiv2beta1.DeletePropagationPolicy) error {
	err := s.canAccessJob(ctx, jobID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}

	return s.resourceManager.DeleteJob(ctx, jobID, propagationPolicy)
}

func (s *JobServerV1) DeleteJob(ctx context.Context, request *apiv1beta1.DeleteJobRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		deleteJobRequests.Inc()
		defer s.recordMetrics("delete", time.Now(), &err)()
	}
	err = s.deleteJob(ctx, request.GetId(), apiv2beta1.DeletePropagationPolicy_DELETE_PROPAGATION_POLICY_UNSPECIFIED)
	if err != nil {
		return nil, util.Wrap(err, "Failed to disable a recurring run")
	}
	if s.options.CollectMetrics {
		jobCount.Dec()
	}
	return &emptypb.Empty{}, nil
}

func (s *BaseJobServer) enableJob(ctx context.Context, jobId string) error {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbEnable})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ChangeJobMode(ctx, jobId, true)
}

func (s *JobServer) CreateRecurringRun(ctx context.Context, request *apiv2beta1.CreateRecurringRunRequest) (_ *apiv2beta1.RecurringRun, err error) {
	if s.options.CollectMetrics {
		createJobRequests.Inc()
		defer s.recordMetrics("create", time.Now(), &err)()
	}

	modelJob, err := toModelJob(request.GetRecurringRun())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run due to conversion error")
	}
	newRecurringRun, err := s.createJob(ctx, modelJob)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run")
	}

	if s.options.CollectMetrics {
		jobCount.Inc()
	}
	apiRecurringRun := toApiRecurringRun(newRecurringRun)
	if apiRecurringRun == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal recurring run representation to its API counterpart"), "Failed to create a recurring run")
	}

	return apiRecurringRun, nil
}

func (s *JobServer) GetRecurringRun(ctx context.Context, request *apiv2beta1.GetRecurringRunRequest) (_ *apiv2beta1.RecurringRun, err error) {
	if s.options.CollectMetrics {
		getJobRequests.Inc()
		defer s.recordMetrics("get", time.Now(), &err)()
	}
	recurringRun, err := s.getJob(ctx, request.GetRecurringRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch a recurring run")
	}

	apiRecurringRun := toApiRecurringRun(recurringRun)
	if apiRecurringRun == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal recurring run representation to its API counterpart"), "Failed to fetch a recurring run")
	}

	return apiRecurringRun, nil
}

func (s *JobServer) ListRecurringRuns(ctx context.Context, r *apiv2beta1.ListRecurringRunsRequest) (_ *apiv2beta1.ListRecurringRunsResponse, err error) {
	if s.options.CollectMetrics {
		listJobRequests.Inc()
		defer s.recordMetrics("list", time.Now(), &err)()
	}

	opts, err := validatedListOptions(&model.Job{}, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter(), "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to list recurring runs due to error parsing the listing options")
	}

	jobs, total_size, nextPageToken, err := s.listJobs(ctx, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), opts, r.GetNamespace(), r.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to list jobs")
	}
	apiRecurringRuns := toApiRecurringRuns(jobs)
	if apiRecurringRuns == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal recurring run representations to their API counterparts"), "Failed to list recurring runs")
	}
	return &apiv2beta1.ListRecurringRunsResponse{
		RecurringRuns: apiRecurringRuns,
		TotalSize:     int32(total_size),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *JobServer) EnableRecurringRun(ctx context.Context, request *apiv2beta1.EnableRecurringRunRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		enableJobRequests.Inc()
		defer s.recordMetrics("enable", time.Now(), &err)()
	}
	err = s.enableJob(ctx, request.GetRecurringRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to enable a recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *JobServer) DisableRecurringRun(ctx context.Context, request *apiv2beta1.DisableRecurringRunRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		disableJobRequests.Inc()
		defer s.recordMetrics("disable", time.Now(), &err)()
	}

	err = s.disableJob(ctx, request.GetRecurringRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to disable a recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *JobServer) DeleteRecurringRun(ctx context.Context, request *apiv2beta1.DeleteRecurringRunRequest) (_ *emptypb.Empty, err error) {
	if s.options.CollectMetrics {
		deleteJobRequests.Inc()
		defer s.recordMetrics("delete", time.Now(), &err)()
	}
	err = s.deleteJob(ctx, request.GetRecurringRunId(), request.GetPropagationPolicy())
	if err != nil {
		return nil, util.Wrap(err, "Failed to delete a recurring run")
	}
	if s.options.CollectMetrics {
		jobCount.Dec()
	}
	return &emptypb.Empty{}, nil
}

func (s *BaseJobServer) canAccessJob(ctx context.Context, jobID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if jobID != "" {
		job, err := s.resourceManager.GetJob(jobID)
		if err != nil {
			return util.Wrap(err, "failed to authorize with the recurring run ID")
		}
		if s.resourceManager.IsEmptyNamespace(job.Namespace) {
			experiment, err := s.resourceManager.GetExperiment(job.ExperimentId)
			if err != nil {
				return util.NewInternalServerError(err, "recurring run %v has an empty namespace and the parent experiment %v could not be fetched", jobID, job.ExperimentId)
			}
			resourceAttributes.Namespace = experiment.Namespace
		} else {
			resourceAttributes.Namespace = job.Namespace
		}
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = job.DisplayName
		}
	}
	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return util.NewInvalidInputError("a recurring run cannot have an empty namespace in multi-user mode")
	}
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeJobs

	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "failed to authorize with API")
	}
	return nil
}

func NewJobServer(resourceManager *resource.ResourceManager, options *JobServerOptions) *JobServer {
	return &JobServer{
		BaseJobServer: &BaseJobServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}

func NewJobServerV1(resourceManager *resource.ResourceManager, options *JobServerOptions) *JobServerV1 {
	return &JobServerV1{
		BaseJobServer: &BaseJobServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}
