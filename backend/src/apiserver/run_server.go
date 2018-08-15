package main

import (
	"context"

	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
)

type RunServer struct {
	resourceManager *resource.ResourceManager
}

func (s *RunServer) GetRunV2(ctx context.Context, request *api.GetRunV2Request) (*api.RunDetail, error) {
	run, err := s.resourceManager.GetRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) GetRun(ctx context.Context, request *api.GetRunRequest) (*api.RunDetail, error) {
	run, err := s.resourceManager.GetRun(request.RunId)
	if err != nil {
		return nil, err
	}
	return ToApiRunDetail(run), nil
}

func (s *RunServer) ListRuns(ctx context.Context, request *api.ListRunsRequest) (*api.ListRunsResponse, error) {
	sortByModelField, isDesc, err := parseSortByQueryString(request.SortBy, runModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	runs, nextPageToken, err := s.resourceManager.ListRunsV2(request.PageToken, int(request.PageSize), sortByModelField, isDesc)
	if err != nil {
		return nil, err
	}
	return &api.ListRunsResponse{Runs: ToApiRuns(runs), NextPageToken: nextPageToken}, nil
}
