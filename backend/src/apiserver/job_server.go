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

package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
)

type JobServer struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServer) CreateJob(ctx context.Context, request *api.CreateJobRequest) (*api.Job, error) {
	jobs, err := ToModelJob(request.Job)
	if err != nil {
		return nil, err
	}
	newJob, err := s.resourceManager.CreateJob(jobs)
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
	sortByModelField, isDesc, err := parseSortByQueryString(request.SortBy, jobModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	jobs, nextPageToken, err := s.resourceManager.ListJobs(request.PageToken, int(request.PageSize), sortByModelField, isDesc)
	if err != nil {
		return nil, err
	}
	apiJobs, err := ToApiJobs(jobs)
	if err != nil {
		return nil, err
	}
	return &api.ListJobsResponse{Jobs: apiJobs, NextPageToken: nextPageToken}, nil
}

func (s *JobServer) ListJobRuns(ctx context.Context, request *api.ListJobRunsRequest) (*api.ListJobRunsResponse, error) {
	sortByModelField, isDesc, err := parseSortByQueryString(request.SortBy, runModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	runs, nextPageToken, err := s.resourceManager.ListRuns(
		request.JobId, request.PageToken, int(request.PageSize), sortByModelField, isDesc)
	if err != nil {
		return nil, err
	}
	return &api.ListJobRunsResponse{Runs: ToApiRuns(runs), NextPageToken: nextPageToken}, nil
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

func (s *JobServer) enableJob(id string, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnableJob(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
