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

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
)

type PipelineServer struct {
	resourceManager *resource.ResourceManager
}

func (s *PipelineServer) GetPipeline(ctx context.Context, request *api.GetPipelineRequest) (*api.Pipeline, error) {
	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, err
	}
	return ToApiPipeline(pipeline), nil
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
	apiPipelines := ToApiPipelines(pipelines)
	return &api.ListPipelinesResponse{Pipelines: apiPipelines, NextPageToken: nextPageToken}, nil
}

func (s *PipelineServer) DeletePipeline(ctx context.Context, request *api.DeletePipelineRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeletePipeline(request.Id)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetTemplate(ctx context.Context, request *api.GetTemplateRequest) (*api.GetTemplateResponse, error) {
	template, err := s.resourceManager.GetPipelineTemplate(request.Id)
	if err != nil {
		return nil, err
	}

	return &api.GetTemplateResponse{Template: string(template)}, nil
}
