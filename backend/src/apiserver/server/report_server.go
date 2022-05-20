// Copyright 2018 The Kubeflow Authors
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

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
)

type ReportServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ReportServer) ReportWorkflow(ctx context.Context,
	request *api.ReportWorkflowRequest) (*empty.Empty, error) {
	execSpec, err := ValidateReportWorkflowRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed.")
	}
	err = s.resourceManager.ReportWorkflowResource(ctx, execSpec)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed.")
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportScheduledWorkflow(ctx context.Context,
	request *api.ReportScheduledWorkflowRequest) (*empty.Empty, error) {
	scheduledWorkflow, err := ValidateReportScheduledWorkflowRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Report scheduled workflow failed.")
	}
	err = s.resourceManager.ReportScheduledWorkflowResource(scheduledWorkflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func ValidateReportWorkflowRequest(request *api.ReportWorkflowRequest) (util.ExecutionSpec, error) {
	execSpec, err := util.NewExecutionSpecJSON([]byte(request.Workflow))
	if err != nil {
		return nil, util.NewInvalidInputError("Could not unmarshal workflow: %v: %v", err, request.Workflow)
	}
	if execSpec.ExecutionName() == "" {
		return nil, util.NewInvalidInputError("The workflow must have a name: %+v", execSpec)
	}
	if execSpec.ExecutionNamespace() == "" {
		return nil, util.NewInvalidInputError("The workflow must have a namespace: %+v", execSpec)
	}
	if execSpec.ExecutionUID() == "" {
		return nil, util.NewInvalidInputError("The workflow must have a UID: %+v", execSpec)
	}
	return execSpec, nil
}

func ValidateReportScheduledWorkflowRequest(request *api.ReportScheduledWorkflowRequest) (*util.ScheduledWorkflow, error) {
	var scheduledWorkflow scheduledworkflow.ScheduledWorkflow
	err := json.Unmarshal([]byte(request.ScheduledWorkflow), &scheduledWorkflow)
	if err != nil {
		return nil, util.NewInvalidInputError("Could not unmarshal scheduled workflow: %v: %v",
			err, request.ScheduledWorkflow)
	}
	swf := util.NewScheduledWorkflow(&scheduledWorkflow)
	if swf.Name == "" {
		return nil, util.NewInvalidInputError("The resource must have a name: %+v", swf.ScheduledWorkflow)
	}
	if swf.Namespace == "" {
		return nil, util.NewInvalidInputError("The resource must have a namespace: %+v", swf.ScheduledWorkflow)
	}
	if swf.UID == "" {
		return nil, util.NewInvalidInputError("The resource must have a UID: %+v", swf.UID)
	}
	return swf, nil
}

func NewReportServer(resourceManager *resource.ResourceManager) *ReportServer {
	return &ReportServer{resourceManager: resourceManager}
}
