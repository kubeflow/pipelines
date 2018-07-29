package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
)

var pipelineModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
	"package_id": "PackageId",
}

type PipelineServer struct {
	resourceManager *resource.ResourceManager
}

func (s *PipelineServer) CreatePipeline(ctx context.Context, request *api.CreatePipelineRequest) (*api.Pipeline, error) {
	pipelines, err := ToModelPipeline(request.Pipeline)
	if err != nil {
		return nil, err
	}
	newPipeline, err := s.resourceManager.CreatePipeline(pipelines)
	if err != nil {
		return nil, err
	}
	return ToApiPipeline(newPipeline)
}

func (s *PipelineServer) GetPipeline(ctx context.Context, request *api.GetPipelineRequest) (*api.Pipeline, error) {
	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiPipeline(pipeline)
}

func (s *PipelineServer) ListPipelines(ctx context.Context, request *api.ListPipelinesRequest) (*api.ListPipelinesResponse, error) {
	sortByModelField, isDesc, err := parseSortByQueryString(request.SortBy, pipelineModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	pipelines, nextPageToken, err := s.resourceManager.ListPipelines(request.PageToken, int(request.PageSize), sortByModelField, isDesc)
	if err != nil {
		return nil, err
	}
	apiPipelines, err := ToApiPipelines(pipelines)
	if err != nil {
		return nil, err
	}
	return &api.ListPipelinesResponse{Pipelines: apiPipelines, NextPageToken: nextPageToken}, nil
}

func (s *PipelineServer) EnablePipeline(ctx context.Context, request *api.EnablePipelineRequest) (*empty.Empty, error) {
	return s.enablePipeline(request.Id, true)
}

func (s *PipelineServer) DisablePipeline(ctx context.Context, request *api.DisablePipelineRequest) (*empty.Empty, error) {
	return s.enablePipeline(request.Id, false)
}

func (s *PipelineServer) DeletePipeline(ctx context.Context, request *api.DeletePipelineRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeletePipeline(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *PipelineServer) enablePipeline(id string, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnablePipeline(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
