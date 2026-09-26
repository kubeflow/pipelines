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

	// TODO(jingzhang36): error count and success count.

	jobCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "job_server_job_count",
		Help: "The current number of jobs in Kubeflow Pipelines instance",
	})
)

type JobServerOptions struct {
	CollectMetrics bool
}

type BaseJobServer struct {
	resourceManager *resource.ResourceManager
	options         *JobServerOptions
}

type JobServer struct {
	*BaseJobServer
	apiv2beta1.UnimplementedRecurringRunServiceServer
}

func (s *BaseJobServer) createJob(ctx context.Context, job *model.Job) (*model.Job, error) {
	// Validate user inputs
	if job.DisplayName == "" {
		return nil, util.NewInvalidInputError("Recurring run name is empty. Please specify a valid name")
	}
	// Resolving an empty experiment id creates the namespace's default
	// experiment, so authorize the requested namespace before that write.
	if common.IsMultiUserMode() && job.ExperimentId == "" {
		if err := s.canAccessJob(ctx, "", &authorizationv1.ResourceAttributes{
			Namespace: job.Namespace,
			Verb:      common.RbacResourceVerbCreate,
			Name:      job.DisplayName,
		}); err != nil {
			return nil, util.Wrapf(err, "Failed to create a recurring run due to authorization error. Check if you have write permission to namespace %s", job.Namespace)
		}
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
	if err := canAccessReferencedPipeline(ctx, s.resourceManager, &job.PipelineSpec, job.Namespace); err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run due to authorization error on the referenced pipeline")
	}
	return s.resourceManager.CreateJob(ctx, job)
}

func (s *BaseJobServer) getJob(ctx context.Context, jobId string) (*model.Job, error) {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.GetJob(jobId)
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

func (s *BaseJobServer) disableJob(ctx context.Context, jobId string) error {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDisable})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ChangeJobMode(ctx, jobId, false)
}

func (s *BaseJobServer) deleteJob(ctx context.Context, jobID string, propagationPolicy apiv2beta1.DeletePropagationPolicy) error {
	err := s.canAccessJob(ctx, jobID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}

	return s.resourceManager.DeleteJob(ctx, jobID, propagationPolicy)
}

func (s *BaseJobServer) enableJob(ctx context.Context, jobId string) error {
	err := s.canAccessJob(ctx, jobId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbEnable})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ChangeJobMode(ctx, jobId, true)
}

func (s *JobServer) CreateRecurringRun(ctx context.Context, request *apiv2beta1.CreateRecurringRunRequest) (*apiv2beta1.RecurringRun, error) {
	if s.options.CollectMetrics {
		createJobRequests.Inc()
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

func (s *JobServer) GetRecurringRun(ctx context.Context, request *apiv2beta1.GetRecurringRunRequest) (*apiv2beta1.RecurringRun, error) {
	if s.options.CollectMetrics {
		getJobRequests.Inc()
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

func (s *JobServer) ListRecurringRuns(ctx context.Context, r *apiv2beta1.ListRecurringRunsRequest) (*apiv2beta1.ListRecurringRunsResponse, error) {
	if s.options.CollectMetrics {
		listJobRequests.Inc()
	}

	opts, err := validatedListOptions(&model.Job{}, r.GetPageToken(), int(r.GetPageSize()), r.GetSortBy(), r.GetFilter())
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

func (s *JobServer) EnableRecurringRun(ctx context.Context, request *apiv2beta1.EnableRecurringRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		enableJobRequests.Inc()
	}
	err := s.enableJob(ctx, request.GetRecurringRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to enable a recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *JobServer) DisableRecurringRun(ctx context.Context, request *apiv2beta1.DisableRecurringRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		disableJobRequests.Inc()
	}

	err := s.disableJob(ctx, request.GetRecurringRunId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to disable a recurring run")
	}
	return &emptypb.Empty{}, nil
}

func (s *JobServer) DeleteRecurringRun(ctx context.Context, request *apiv2beta1.DeleteRecurringRunRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deleteJobRequests.Inc()
	}
	err := s.deleteJob(ctx, request.GetRecurringRunId(), request.GetPropagationPolicy())
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
