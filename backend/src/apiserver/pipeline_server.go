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
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

var pipelineModelFieldsBySortableAPIFields = map[string]string{
	"id":         "ID",
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
	sortByModelField, ok := pipelineModelFieldsBySortableAPIFields[request.SortBy]
	if request.SortBy != "" && !ok {
		return nil, util.NewInvalidInputError("Received invalid sort by field %v.", request.SortBy)
	}
	pipelines, nextPageToken, err := s.resourceManager.ListPipelines(request.PageToken, int(request.PageSize), sortByModelField)
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

func (s *PipelineServer) enablePipeline(id uint32, enabled bool) (*empty.Empty, error) {
	err := s.resourceManager.EnablePipeline(id, enabled)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
