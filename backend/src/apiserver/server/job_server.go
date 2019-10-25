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
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/robfig/cron"
)

type JobServer struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServer) CreateJob(ctx context.Context, request *api.CreateJobRequest) (*api.Job, error) {
	err := s.validateCreateJobRequest(request)
	if err != nil {
		return nil, err
	}
	newJob, err := s.resourceManager.CreateJob(request.Job)
	if err != nil {
		return nil, err
	}
	return ToApiJob(newJob), nil
}

func (s *JobServer) GetJob(ctx context.Context, request *api.GetJobRequest) (*api.Job, error) {
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
	jobs, total_size, nextPageToken, err := s.resourceManager.ListJobs(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list jobs.")
	}
	return &api.ListJobsResponse{Jobs: ToApiJobs(jobs), TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *JobServer) EnableJob(ctx context.Context, request *api.EnableJobRequest) (*empty.Empty, error) {
	return s.enableJob(request.Id, true)
}

func (s *JobServer) DisableJob(ctx context.Context, request *api.DisableJobRequest) (*empty.Empty, error) {
	return s.enableJob(request.Id, false)
}

func (s *JobServer) DeleteJob(ctx context.Context, request *api.DeleteJobRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeleteJob(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *JobServer) validateCreateJobRequest(request *api.CreateJobRequest) error {
	job := request.Job

	if err := ValidatePipelineSpec(s.resourceManager, job.PipelineSpec); err != nil {
		if _, errResourceReference := CheckPipelineVersionReference(s.resourceManager, job.ResourceReferences); errResourceReference != nil {
			return util.Wrap(err, "Neither pipeline spec nor pipeline version is valid."+errResourceReference.Error())
		}
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
	err := s.resourceManager.EnableJob(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func NewJobServer(resourceManager *resource.ResourceManager) *JobServer {
	return &JobServer{resourceManager: resourceManager}
}
