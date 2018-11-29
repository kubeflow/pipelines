// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type RunServer struct {
	resourceManager *resource.ResourceManager
}

func (s *RunServer) CreateRun(ctx context.Context, request *api.CreateRunRequest) (*api.RunDetail, error) {
	err := s.validateCreateRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}
	run, err := s.resourceManager.CreateRun(request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
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
	paginationContext, err := ValidatePagination(
		request.PageToken, int(request.PageSize), model.GetRunTablePrimaryKeyColumn(),
		request.SortBy, runModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, util.Wrap(err, "Validating pagination failed.")
	}
	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}
	runs, nextPageToken, err := s.resourceManager.ListRuns(filterContext, paginationContext)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list runs.")
	}
	return &api.ListRunsResponse{Runs: ToApiRuns(runs), NextPageToken: nextPageToken}, nil
}

func (s *RunServer) ReportRunMetrics(ctx context.Context, request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	// Makes sure run exists
	_, err := s.resourceManager.GetRun(request.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &api.ReportRunMetricsResponse{
		Results: []*api.ReportRunMetricsResponse_ReportRunMetricResult{},
	}
	for _, metric := range request.GetMetrics() {
		err := ValidateRunMetric(metric)
		if err == nil {
			err = s.resourceManager.ReportMetric(metric, request.GetRunId())
		}
		response.Results = append(
			response.Results,
			NewReportRunMetricResult(metric.GetName(), metric.GetNodeId(), err))
	}
	return response, nil
}

func (s *RunServer) ReadArtifact(ctx context.Context, request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	content, err := s.resourceManager.ReadArtifact(
		request.GetRunId(), request.GetNodeId(), request.GetArtifactName())
	if err != nil {
		return nil, util.Wrapf(err, "failed to read artifact '%+v'.", request)
	}
	return &api.ReadArtifactResponse{
		Data: content,
	}, nil
}

func (s *RunServer) validateCreateRunRequest(request *api.CreateRunRequest) error {
	run := request.Run
	if run.Name == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}
	// Run must be created under an experiment.
	if err := ValidateExperimentResourceReference(s.resourceManager, run.ResourceReferences); err != nil {
		return util.Wrap(err, "The run must have a valid experiment resource reference.")
	}

	if err := ValidatePipelineSpec(s.resourceManager, run.PipelineSpec); err != nil {
		return util.Wrap(err, "The pipeline spec is invalid.")
	}
	return nil
}

func NewRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{resourceManager: resourceManager}
}
