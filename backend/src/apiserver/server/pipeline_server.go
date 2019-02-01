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

package server

import (
	"context"
	"net/http"
	"net/url"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type PipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
}

func (s *PipelineServer) CreatePipeline(ctx context.Context, request *api.CreatePipelineRequest) (*api.Pipeline, error) {
	if err := ValidateCreatePipelineRequest(request); err != nil {
		return nil, err
	}

	pipelineUrl := request.Pipeline.Url.PipelineUrl
	resp, err := s.httpClient.Get(pipelineUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v "+
			"Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
	}
	pipelineFileName := path.Base(pipelineUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	}

	pipelineName, err := GetPipelineName(request.Pipeline.Name, pipelineFileName)
	if err != nil {
		return nil, util.Wrap(err, "Invalid pipeline name.")
	}

	pipeline, err := s.resourceManager.CreatePipeline(pipelineName, "", pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed.")
	}

	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) GetPipeline(ctx context.Context, request *api.GetPipelineRequest) (*api.Pipeline, error) {
	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline failed.")
	}
	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) ListPipelines(ctx context.Context, request *api.ListPipelinesRequest) (*api.ListPipelinesResponse, error) {
	opts, err := validatedListOptions(&model.Pipeline{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	pipelines, total_size, nextPageToken, err := s.resourceManager.ListPipelines(opts)
	if err != nil {
		return nil, util.Wrap(err, "List pipelines failed.")
	}
	apiPipelines := ToApiPipelines(pipelines)
	return &api.ListPipelinesResponse{Pipelines: apiPipelines, TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *PipelineServer) DeletePipeline(ctx context.Context, request *api.DeletePipelineRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeletePipeline(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Delete pipelines failed.")
	}

	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetTemplate(ctx context.Context, request *api.GetTemplateRequest) (*api.GetTemplateResponse, error) {
	template, err := s.resourceManager.GetPipelineTemplate(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &api.GetTemplateResponse{Template: string(template)}, nil
}

func ValidateCreatePipelineRequest(request *api.CreatePipelineRequest) error {
	if request.Pipeline.Url == nil || request.Pipeline.Url.PipelineUrl == "" {
		return util.NewInvalidInputError("Pipeline URL is empty. Please specify a valid URL.")
	}

	if _, err := url.ParseRequestURI(request.Pipeline.Url.PipelineUrl); err != nil {
		return util.NewInvalidInputError(
			"Invalid Pipeline URL %v. Please specify a valid URL", request.Pipeline.Url.PipelineUrl)
	}
	return nil
}

func NewPipelineServer(resourceManager *resource.ResourceManager) *PipelineServer {
	return &PipelineServer{resourceManager: resourceManager, httpClient: http.DefaultClient}
}
