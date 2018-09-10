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

type ReportServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ReportServer) ReportWorkflow(ctx context.Context,
	request *api.ReportWorkflowRequest) (*empty.Empty, error) {
	err := s.resourceManager.ReportWorkflowResource(request.Workflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportScheduledWorkflow(ctx context.Context,
	request *api.ReportScheduledWorkflowRequest) (*empty.Empty, error) {
	err := s.resourceManager.ReportScheduledWorkflowResource(request.ScheduledWorkflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
