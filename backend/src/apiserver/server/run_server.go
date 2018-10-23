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

	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type RunServer struct {
	resourceManager *resource.ResourceManager
}

func (s *RunServer) CreateRun(ctx context.Context, request *api.CreateRunRequest) (*api.Run, error) {
	return nil, util.NewBadRequestError(nil, "Not implemented")
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
	// TODO(hongyes): Implement the action.
	return &api.ReportRunMetricsResponse{}, nil
}

func (s *RunServer) ReadArtifact(ctx context.Context, request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	// TODO(hongyes): Implement the action.
	return &api.ReadArtifactResponse{}, nil
}

func NewRunServer(resourceManager *resource.ResourceManager) *RunServer {
	return &RunServer{resourceManager: resourceManager}
}
