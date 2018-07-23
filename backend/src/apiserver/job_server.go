package main

import (
	"context"

	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
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
	job, err := s.resourceManager.GetJob(request.PipelineId, request.JobId)
	if err != nil {
		return nil, err
	}
	return ToApiJobDetail(job), nil
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
