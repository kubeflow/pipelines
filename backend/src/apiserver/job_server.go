package main

import (
	"context"
	"ml/backend/api"
	"ml/backend/src/resource"
)

type JobServer struct {
	resourceManager *resource.ResourceManager
}

func (s *JobServer) GetJob(ctx context.Context, request *api.GetJobRequest) (*api.JobDetail, error) {
	jobDetails, err := s.resourceManager.GetJob(request.PipelineId, request.JobName)
	if err != nil {
		return nil, err
	}
	return ToApiJobDetail(jobDetails), nil
}

func (s *JobServer) ListJobs(ctx context.Context, request *api.ListJobsRequest) (*api.ListJobsResponse, error) {
	jobs, err := s.resourceManager.ListJobs(request.PipelineId)
	if err != nil {
		return nil, err
	}
	return &api.ListJobsResponse{Jobs: ToApiJobs(jobs)}, nil
}
