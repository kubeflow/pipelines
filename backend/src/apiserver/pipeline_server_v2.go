package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/resource"
	"github.com/googleprivate/ml/backend/src/util"
)

var pipelineV2ModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
	"package_id": "PackageId",
}

type PipelineServerV2 struct {
	resourceManager *resource.ResourceManager
}

func (s *PipelineServerV2) CreatePipeline(ctx context.Context, request *api.CreatePipelineRequestV2) (*api.PipelineV2, error) {
	pipelines, err := ToModelPipelineV2(request.Pipeline)
	if err != nil {
		return nil, err
	}
	newPipeline, err := s.resourceManager.CreatePipelineV2(pipelines)
	if err != nil {
		return nil, err
	}
	return ToApiPipelineV2(newPipeline)
}

func (s *PipelineServerV2) GetPipeline(ctx context.Context, request *api.GetPipelineRequestV2) (*api.PipelineV2, error) {
	pipeline, err := s.resourceManager.GetPipelineV2(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiPipelineV2(pipeline)
}

func (s *PipelineServerV2) ListPipelines(ctx context.Context, request *api.ListPipelinesRequestV2) (*api.ListPipelinesResponseV2, error) {
	sortByModelField, ok := pipelineV2ModelFieldsBySortableAPIFields[request.SortBy]
	if request.SortBy != "" && !ok {
		return nil, util.NewInvalidInputError("Received invalid sort by field %v.", request.SortBy)
	}
	pipelines, nextPageToken, err := s.resourceManager.ListPipelinesV2(request.PageToken, int(request.PageSize), sortByModelField)
	if err != nil {
		return nil, err
	}
	apiPipelines, err := ToApiPipelinesV2(pipelines)
	if err != nil {
		return nil, err
	}
	return &api.ListPipelinesResponseV2{Pipelines: apiPipelines, NextPageToken: nextPageToken}, nil
}

func (s *PipelineServerV2) EnablePipeline(ctx context.Context, request *api.EnablePipelineRequestV2) (*empty.Empty, error) {
	return s.enablePipeline(request.Id, true)
}

func (s *PipelineServerV2) DisablePipeline(ctx context.Context, request *api.DisablePipelineRequestV2) (*empty.Empty, error) {
	return s.enablePipeline(request.Id, false)
}

func (s *PipelineServerV2) DeletePipeline(ctx context.Context, request *api.DeletePipelineRequestV2) (*empty.Empty, error) {
	err := s.resourceManager.DeletePipelineV2(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *PipelineServerV2) enablePipeline(id string, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnablePipelineV2(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
