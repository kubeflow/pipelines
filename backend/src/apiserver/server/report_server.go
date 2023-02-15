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
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
)

type ReportServer struct {
	resourceManager *resource.ResourceManager
}

// Extracts task details from an execution spec and reports them to storage.
func (s ReportServer) reportTasksFromExecution(wf *util.Workflow) ([]*model.Task, error) {
	if len(wf.Status.Nodes) == 0 {
		return nil, nil
	}
	tasks, err := toModelTasks(wf.Status.Nodes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report tasks of an execution")
	}
	return s.resourceManager.CreateOrUpdateTasks(tasks)
}

// Report a workflow based on execution spec.
func (s *ReportServer) reportWorkflow(ctx context.Context, execSpec *util.ExecutionSpec) (*empty.Empty, error) {
	wf, err := util.NewWorkflowFromInterface(execSpec)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert execution spec to Argo workflow")
	}
	_, err = s.reportTasksFromExecution(wf)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report task details")
	}
	err = s.resourceManager.ReportWorkflowResource(ctx, *execSpec)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report workflow")
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportWorkflowV1(ctx context.Context,
	request *apiv1beta1.ReportWorkflowRequest,
) (*empty.Empty, error) {
	execSpec, err := validateReportWorkflowRequestV1(request)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed")
	}
	return s.reportWorkflow(ctx, execSpec)
}

func (s *ReportServer) ReportWorkflow(ctx context.Context,
	request *apiv2beta1.ReportWorkflowRequest,
) (*empty.Empty, error) {
	execSpec, err := validateReportWorkflowRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed")
	}
	return s.reportWorkflow(ctx, execSpec)
}

func (s *ReportServer) ReportScheduledWorkflowV1(ctx context.Context,
	request *apiv1beta1.ReportScheduledWorkflowRequest,
) (*empty.Empty, error) {
	scheduledWorkflow, err := validateReportScheduledWorkflowRequestV1(request)
	if err != nil {
		return nil, util.Wrap(err, "Report scheduled workflow failed")
	}
	err = s.resourceManager.ReportScheduledWorkflowResource(scheduledWorkflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportScheduledWorkflow(ctx context.Context,
	request *apiv2beta1.ReportScheduledWorkflowRequest,
) (*empty.Empty, error) {
	scheduledWorkflow, err := validateReportScheduledWorkflowRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Report scheduled workflow failed")
	}
	err = s.resourceManager.ReportScheduledWorkflowResource(scheduledWorkflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func validateReportWorkflowRequestV1(request *apiv1beta1.ReportWorkflowRequest) (*util.ExecutionSpec, error) {
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(request.Workflow))
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
	return &execSpec, nil
}

func validateReportWorkflowRequest(request *apiv2beta1.ReportWorkflowRequest) (*util.ExecutionSpec, error) {
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(request.Workflow))
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
	return &execSpec, nil
}

func validateReportScheduledWorkflowRequestV1(request *apiv1beta1.ReportScheduledWorkflowRequest) (*util.ScheduledWorkflow, error) {
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

func validateReportScheduledWorkflowRequest(request *apiv2beta1.ReportScheduledWorkflowRequest) (*util.ScheduledWorkflow, error) {
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
