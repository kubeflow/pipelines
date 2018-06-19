package main

import (
	"context"

	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/resource"
	"github.com/googleprivate/ml/backend/src/util"
)

var jobV2ModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

type JobServerV2 struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServerV2) GetJob(ctx context.Context, request *api.GetJobRequestV2) (*api.JobDetailV2, error) {
	job, err := s.resourceManager.GetJobV2(request.PipelineId, request.JobId)
	if err != nil {
		return nil, err
	}
	return ToApiJobDetailV2(job), nil
}

func (s *JobServerV2) ListJobs(ctx context.Context, request *api.ListJobsRequestV2) (*api.ListJobsResponseV2, error) {
	sortByModelField, ok := jobV2ModelFieldsBySortableAPIFields[request.SortBy]
	if request.SortBy != "" && !ok {
		return nil, util.NewInvalidInputError("Received invalid sort by field %v.", request.SortBy)
	}
	jobs, nextPageToken, err := s.resourceManager.ListJobsV2(
		request.PipelineId, request.PageToken, int(request.PageSize), sortByModelField)
	if err != nil {
		return nil, err
	}
	return &api.ListJobsResponseV2{Jobs: ToApiJobsV2(jobs), NextPageToken: nextPageToken}, nil
}
