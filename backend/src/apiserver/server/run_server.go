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
	"encoding/json"

	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type RunServer struct {
	resourceManager *resource.ResourceManager
}

func (s *RunServer) CreateRun(ctx context.Context, request *api.CreateRunRequest) (*api.RunDetail, error) {
	err := ValidateCreateRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create run request failed.")
	}
	run, err := s.resourceManager.CreateRun(request.Run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new run.")
	}
	return ToApiRunDetail(run), nil
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
	paginationContext, err := ValidateListRequest(
		request.PageToken, int(request.PageSize), model.GetRunTablePrimaryKeyColumn(),
		request.SortBy, runModelFieldsBySortableAPIFields)
	if err != nil {
		return nil, err
	}
	runs, nextPageToken, err := s.resourceManager.ListRunsV2(paginationContext)
	if err != nil {
		return nil, err
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
	// TODO(hongyes): Implement the action.
	return &api.ReadArtifactResponse{}, nil
}

func ValidateCreateRunRequest(request *api.CreateRunRequest) error {
	run := request.Run
	if run.Name == "" {
		return util.NewInvalidInputError("The run name is empty. Please specify a valid name.")
	}
	if err := ValidateRunResourceReference(run.ResourceReferences); err != nil {
		return util.Wrap(err, "The resource reference is invalid.")
	}

	if err := ValidatePipelineSpec(run.PipelineSpec); err != nil {
		return util.Wrap(err, "The pipeline spec is invalid.")
	}
	return nil
}

func ValidatePipelineSpec(spec *api.PipelineSpec) error {
	if spec == nil || (spec.GetPipelineId() == "" && spec.GetWorkflowManifest() == "") {
		return util.NewInvalidInputError("Please specify a pipeline by providing a pipeline ID or workflow manifest.")
	}
	if spec.GetPipelineId() != "" && spec.GetWorkflowManifest() != "" {
		return util.NewInvalidInputError("Please either specify a pipeline ID or a workflow manifest, not both.")
	}
	paramsBytes, err := json.Marshal(spec.Parameters)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to Marshall the pipeline parameters into bytes. Parameters: %s",
			printParameters(spec.Parameters))
	}
	if len(paramsBytes) > util.MaxParameterBytes {
		return util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
	}
	return nil
}

func ValidateRunResourceReference(references []*api.ResourceReference) error {
	if references == nil || len(references) == 0 || references[0] == nil {
		return util.NewInvalidInputError("The resource reference is empty. Please specify which experiment owns this run.")
	}
	if len(references) > 1 {
		return util.NewInvalidInputError("Got more resource references than expected. Please only specify which experiment owns this run.")
	}
	if references[0].Key.Type != api.ResourceType_EXPERIMENT {
		return util.NewInvalidInputError("Unexpected resource type. Expected:%v. Got: %v",
			api.ResourceType_EXPERIMENT, references[0].Key.Type)
	}
	if references[0].Key.Id == "" {
		return util.NewInvalidInputError("Resource ID is empty. Please specify a valid ID")
	}
	if references[0].Relationship != api.Relationship_OWNER {
		return util.NewInvalidInputError("Unexpected relationship for the experiment. Expected: %v. Got: %v",
			api.Relationship_OWNER, references[0].Relationship)
	}
	return nil
}

func NewRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{resourceManager: resourceManager}
}
