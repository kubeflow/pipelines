// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"

	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/resource"
	"github.com/googleprivate/ml/backend/src/util"
)

var jobModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

type JobServer struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServer) GetJob(ctx context.Context, request *api.GetJobRequest) (*api.JobDetail, error) {
	jobDetails, err := s.resourceManager.GetJob(request.PipelineId, request.JobName)
	if err != nil {
		return nil, err
	}
	return ToApiJobDetail(jobDetails)
}

func (s *JobServer) ListJobs(ctx context.Context, request *api.ListJobsRequest) (*api.ListJobsResponse, error) {
	sortByModelField, ok := jobModelFieldsBySortableAPIFields[request.SortBy]
	if request.SortBy != "" && !ok {
		return nil, util.NewInvalidInputError("Received invalid sort by field %v.", request.SortBy)
	}
	jobs, nextPageToken, err := s.resourceManager.ListJobs(
		request.PipelineId, request.PageToken, int(request.PageSize), sortByModelField)
	if err != nil {
		return nil, err
	}
	return &api.ListJobsResponse{Jobs: ToApiJobs(jobs), NextPageToken: nextPageToken}, nil
}
