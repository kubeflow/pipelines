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
	"github.com/robfig/cron"
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

type JobServer struct {
	resourceManager *resource.ResourceManager
	options         *JobServerOptions
}

func (s *JobServer) CreateJob(ctx context.Context, request *api.CreateJobRequest) (*api.Job, error) {
	if s.options.CollectMetrics {
		createJobRequests.Inc()
	}

	err := s.validateCreateJobRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create job request failed.")
	}

	if common.IsMultiUserMode() {
		experimentID := common.GetExperimentIDFromAPIResourceReferences(request.Job.ResourceReferences)
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
			Name:      request.Job.Name,
		}
		err = s.canAccessJob(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize the request")
		}
	}

	newJob, err := s.resourceManager.CreateJob(request.Job)
	if err != nil {
		return nil, err
	}

	if s.options.CollectMetrics {
		jobCount.Inc()
	}
	return ToApiJob(newJob), nil
}

func (s *JobServer) GetJob(ctx context.Context, request *api.GetJobRequest) (*api.Job, error) {
	if s.options.CollectMetrics {
		getJobRequests.Inc()
	}

	err := s.canAccessJob(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	job, err := s.resourceManager.GetJob(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiJob(job), nil
}

func (s *JobServer) ListJobs(ctx context.Context, request *api.ListJobsRequest) (*api.ListJobsResponse, error) {
	if s.options.CollectMetrics {
		listJobRequests.Inc()
	}

	opts, err := validatedListOptions(&model.Job{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

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
			return nil, util.NewInvalidInputError("ListJobs must filter by resource reference in multi-user mode.")
		}
		if refKey.Type == common.Namespace {
			namespace := refKey.ID
			if len(namespace) == 0 {
				return nil, util.NewInvalidInputError("Invalid resource references for ListJobs. Namespace is empty.")
			}
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbList,
			}
			err = s.canAccessJob(ctx, "", resourceAttributes)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with namespace resource reference.")
			}
		} else if refKey.Type == common.Experiment {
			experimentID := refKey.ID
			if len(experimentID) == 0 {
				return nil, util.NewInvalidInputError("Invalid resource references for job. Experiment ID is empty.")
			}
			namespace, err := s.resourceManager.GetNamespaceFromExperimentID(experimentID)
			if err != nil {
				return nil, util.Wrap(err, "Failed to get namespace for job's experiment.")
			}
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbList,
			}
			err = s.canAccessJob(ctx, "", resourceAttributes)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with namespace in experiment resource reference.")
			}
		} else {
			return nil, util.NewInvalidInputError("Invalid resource references for ListJobs. Got %+v", request.ResourceReferenceKey)
		}
	}

	jobs, total_size, nextPageToken, err := s.resourceManager.ListJobs(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list jobs.")
	}
	return &api.ListJobsResponse{Jobs: ToApiJobs(jobs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *JobServer) EnableJob(ctx context.Context, request *api.EnableJobRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		enableJobRequests.Inc()
	}

	err := s.canAccessJob(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbEnable})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return s.enableJob(request.Id, true)
}

func (s *JobServer) DisableJob(ctx context.Context, request *api.DisableJobRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		disableJobRequests.Inc()
	}

	err := s.canAccessJob(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDisable})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return s.enableJob(request.Id, false)
}

func (s *JobServer) DeleteJob(ctx context.Context, request *api.DeleteJobRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deleteJobRequests.Inc()
	}

	err := s.canAccessJob(ctx, request.Id, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	err = s.resourceManager.DeleteJob(request.Id)
	if err != nil {
		return nil, err
	}

	if s.options.CollectMetrics {
		jobCount.Dec()
	}
	return &empty.Empty{}, nil
}

func (s *JobServer) validateCreateJobRequest(request *api.CreateJobRequest) error {
	job := request.Job

	if err := ValidatePipelineSpecAndResourceReferences(s.resourceManager, job.PipelineSpec, job.ResourceReferences); err != nil {
		return err
	}
	if job.MaxConcurrency > 10 || job.MaxConcurrency < 1 {
		return util.NewInvalidInputError("The max concurrency of the job is out of range. Support 1-10. Received %v.", job.MaxConcurrency)
	}
	if job.Trigger != nil && job.Trigger.GetCronSchedule() != nil {
		if _, err := cron.Parse(job.Trigger.GetCronSchedule().Cron); err != nil {
			return util.NewInvalidInputError(
				"Schedule cron is not a supported format(https://godoc.org/github.com/robfig/cron). Error: %v", err)
		}
	}
	if job.Trigger != nil && job.Trigger.GetPeriodicSchedule() != nil {
		periodicScheduleInterval := job.Trigger.GetPeriodicSchedule().IntervalSecond
		if periodicScheduleInterval < 1 {
			return util.NewInvalidInputError(
				"Found invalid period schedule interval %v. Set at interval to least 1 second.", periodicScheduleInterval)
		}
	}
	return nil
}

func (s *JobServer) enableJob(id string, enabled bool) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		enableJobRequests.Inc()
	}

	err := s.resourceManager.EnableJob(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *JobServer) canAccessJob(ctx context.Context, jobID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if common.IsMultiUserMode() == false {
		// Skip authorization if not multi-user mode.
		return nil
	}

	if len(jobID) > 0 {
		job, err := s.resourceManager.GetJob(jobID)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the job ID.")
		}
		if len(resourceAttributes.Namespace) == 0 {
			if len(job.Namespace) == 0 {
				return util.NewInternalServerError(
					errors.New("Empty namespace"),
					"The job doesn't have a valid namespace.",
				)
			}
			resourceAttributes.Namespace = job.Namespace
		}
		if len(resourceAttributes.Name) == 0 {
			resourceAttributes.Name = job.Name
		}
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeJobs

	err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}

func NewJobServer(resourceManager *resource.ResourceManager, options *JobServerOptions) *JobServer {
	return &JobServer{resourceManager: resourceManager, options: options}
}
