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
	"encoding/json"

	workflow "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type ReportServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ReportServer) ReportWorkflow(ctx context.Context,
	request *api.ReportWorkflowRequest) (*empty.Empty, error) {
	workflow, err := ValidateReportWorkflowRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed.")
	}
	err = s.resourceManager.ReportWorkflowResource(workflow)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed.")
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

func ValidateReportWorkflowRequest(request *api.ReportWorkflowRequest) (*util.Workflow, error) {
	var workflow1 workflow.Workflow
	err := json.Unmarshal([]byte(request.Workflow), &workflow1)
	if err != nil {
		return nil, util.NewInvalidInputError("Could not unmarshal workflow: %v: %v", err, request.Workflow)
	}
	workflow := util.NewWorkflow(&workflow1)
	if workflow.Name == "" {
		return nil, util.NewInvalidInputError("The workflow must have a name: %+v", workflow.Workflow)
	}
	if workflow.Namespace == "" {
		return nil, util.NewInvalidInputError("The workflow must have a namespace: %+v", workflow.Workflow)
	}
	ownerUID := workflow.ScheduledWorkflowUUIDAsStringOrEmpty()
	if ownerUID == "" {
		return nil, util.NewInvalidInputError("The workflow must have a valid owner: %+v", workflow.Workflow)
	}

	if workflow.UID == "" {
		return nil, util.NewInvalidInputError("The workflow must have a UID: %+v", workflow.Workflow)
	}
	return workflow, nil
}

func NewReportServer(resourceManager *resource.ResourceManager) *ReportServer {
	return &ReportServer{resourceManager: resourceManager}
}
