package main

import (
	"context"
	"ml/backend/api"
	"ml/backend/src/resource"

	"github.com/golang/protobuf/ptypes/empty"
)

type PipelineServer struct {
	resourceManager *resource.ResourceManager
}

func (s *PipelineServer) CreatePipeline(ctx context.Context, request *api.CreatePipelineRequest) (*api.Pipeline, error) {
	newPipeline, err := s.resourceManager.CreatePipeline(ToModelPipeline(request.Pipeline))
	if err != nil {
		return nil, err
	}
	return ToApiPipeline(newPipeline), nil
}

func (s *PipelineServer) GetPipeline(ctx context.Context, request *api.GetPipelineRequest) (*api.Pipeline, error) {
	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) ListPipelines(ctx context.Context, request *api.ListPipelinesRequest) (*api.ListPipelinesResponse, error) {
	pipelines, err := s.resourceManager.ListPipelines()
	if err != nil {
		return nil, err
	}
	return &api.ListPipelinesResponse{Pipelines: ToApiPipelines(pipelines)}, nil
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

func (s *PipelineServer) enablePipeline(id uint32, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnablePipeline(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
