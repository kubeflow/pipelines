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
	"github.com/pkg/errors"
	"github.com/robfig/cron"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type JobServer struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServer) CreateJob(ctx context.Context, request *api.CreateJobRequest) (*api.Job, error) {
	err := s.validateCreateJobRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create job request failed.")
	}
	err = CanAccessExperimentInResourceReferences(s.resourceManager, ctx, request.Job.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	newJob, err := s.resourceManager.CreateJob(request.Job)
	if err != nil {
		return nil, err
	}
	return ToApiJob(newJob), nil
}

func (s *JobServer) GetJob(ctx context.Context, request *api.GetJobRequest) (*api.Job, error) {
	err := s.canAccessJob(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	job, err := s.resourceManager.GetJob(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiJob(job), nil
}

func (s *JobServer) ListJobs(ctx context.Context, request *api.ListJobsRequest) (*api.ListJobsResponse, error) {
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
			err = isAuthorized(s.resourceManager, ctx, namespace)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with namespace resource reference.")
			}
		} else if refKey.Type == common.Experiment {
			experimentID := refKey.ID
			if len(experimentID) == 0 {
				return nil, util.NewInvalidInputError("Invalid resource references for job. Experiment ID is empty.")
			}
			err = CanAccessExperiment(s.resourceManager, ctx, experimentID)
			if err != nil {
				return nil, util.Wrap(err, "Failed to authorize with experiment resource reference.")
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
	err := s.canAccessJob(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	return s.enableJob(request.Id, true)
}

func (s *JobServer) DisableJob(ctx context.Context, request *api.DisableJobRequest) (*empty.Empty, error) {
	err := s.canAccessJob(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	return s.enableJob(request.Id, false)
}

func (s *JobServer) DeleteJob(ctx context.Context, request *api.DeleteJobRequest) (*empty.Empty, error) {
	err := s.canAccessJob(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	err = s.resourceManager.DeleteJob(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *JobServer) UpdateJob(ctx context.Context, request *api.UpdateJobRequest) (*empty.Empty, error) {
	err := s.canAccessJob(ctx, request.Job.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	err = s.validateJobUpdate(request.Job)
	if err != nil {
		return nil, util.Wrap(err, "Validate job failed.")
	}

	return &empty.Empty{}, s.resourceManager.UpdateJob(request.Job)
}

func (s *JobServer) validateCreateJobRequest(request *api.CreateJobRequest) error {
	job := request.Job

	if err := ValidatePipelineSpec(s.resourceManager, job.PipelineSpec); err != nil {
		if _, errResourceReference := CheckPipelineVersionReference(s.resourceManager, job.ResourceReferences); errResourceReference != nil {
			return util.Wrap(err, "Neither pipeline spec nor pipeline version is valid."+errResourceReference.Error())
		}
	}

	if job.MaxConcurrency > 10 || job.MaxConcurrency < 1 {
		return util.NewInvalidInputError("supported max concurrency in range [1, 10], got %v.", job.MaxConcurrency)
	}

	return s.validateTrigger(job.Trigger)
}

func (s *JobServer) validateJobUpdate(job *api.Job) error {
	if job.Name == "" {
		return util.NewInvalidInputError("job name missing")
	}

	if job.GetPipelineSpec() == nil {
		return util.NewInvalidInputError("pipeline spec missing")
	}

	if job.MaxConcurrency > 10 || job.MaxConcurrency < 1 {
		return util.NewInvalidInputError("supported max concurrency in range [1, 10], got %v.", job.MaxConcurrency)
	}

	if job.Trigger == nil {
		return util.NewInvalidInputError("trigger missing")
	}
	return s.validateTrigger(job.Trigger)
}

func (s *JobServer) validateTrigger(trigger *api.Trigger) error {
	if trigger.GetCronSchedule() != nil {
		if _, err := cron.Parse(trigger.GetCronSchedule().Cron); err != nil {
			return util.NewInvalidInputError(
				"Schedule cron is not a supported format(https://godoc.org/github.com/robfig/cron). Error: %v", err)
		}
	}

	if trigger.GetPeriodicSchedule() != nil {
		periodicScheduleInterval := trigger.GetPeriodicSchedule().IntervalSecond
		if periodicScheduleInterval < 1 {
			return util.NewInvalidInputError(
				"Found invalid period schedule interval %v. Set at interval to least 1 second.", periodicScheduleInterval)
		}
	}

	return nil
}

func (s *JobServer) enableJob(id string, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnableJob(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *JobServer) canAccessJob(ctx context.Context, jobID string) error {
	if common.IsMultiUserMode() == false {
		// Skip authorization if not multi-user mode.
		return nil
	}
	namespace, err := s.resourceManager.GetNamespaceFromJobID(jobID)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with the job ID.")
	}
	if len(namespace) == 0 {
		return util.NewInternalServerError(errors.New("Empty namespace"), "The job doesn't have a valid namespace.")
	}

	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}

func NewJobServer(resourceManager *resource.ResourceManager) *JobServer {
	return &JobServer{resourceManager: resourceManager}
}
