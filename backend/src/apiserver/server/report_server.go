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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	authorizationv1 "k8s.io/api/authorization/v1"

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
func (s ReportServer) reportTasksFromExecution(execSpec util.ExecutionSpec, runId string) ([]*model.Task, error) {
	if !execSpec.ExecutionStatus().HasNodes() {
		return nil, nil
	}
	tasks, err := toModelTasks(execSpec)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report tasks of an execution")
	}
	return s.resourceManager.CreateOrUpdateTasks(tasks)
}

// Reports a workflow.
func (s *ReportServer) reportWorkflow(ctx context.Context, workflow string) (*empty.Empty, error) {
	execSpec, err := validateReportWorkflowRequest(workflow)
	if err != nil {
		return nil, util.Wrap(err, "Report workflow failed")
	}

	executionName := (*execSpec).ExecutionName()
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb:     common.RbacResourceVerbReport,
		Resource: common.RbacResourceTypeWorkflows,
	}

	if err := s.canAccessWorkflow(ctx, executionName, resourceAttributes); err != nil {
		return nil, err
	}

	newExecSpec, err := s.resourceManager.ReportWorkflowResource(ctx, *execSpec)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report workflow")
	}

	runId := newExecSpec.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowRunId]
	_, err = s.reportTasksFromExecution(newExecSpec, runId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to report task details")
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportWorkflowV1(ctx context.Context,
	request *apiv1beta1.ReportWorkflowRequest,
) (*empty.Empty, error) {
	return s.reportWorkflow(ctx, request.GetWorkflow())
}

func (s *ReportServer) ReportWorkflow(ctx context.Context,
	request *apiv2beta1.ReportWorkflowRequest,
) (*empty.Empty, error) {
	return s.reportWorkflow(ctx, request.GetWorkflow())
}

// Reports a scheduled workflow.
func (s *ReportServer) reportScheduledWorkflow(ctx context.Context, swf string) (*empty.Empty, error) {
	scheduledWorkflow, err := validateReportScheduledWorkflowRequest(swf)
	if err != nil {
		return nil, util.Wrap(err, "Report scheduled workflow failed")
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb:     common.RbacResourceVerbReport,
		Resource: common.RbacResourceTypeScheduledWorkflows,
	}
	err = s.canAccessWorkflow(ctx, string(scheduledWorkflow.UID), resourceAttributes)
	if err != nil {
		return nil, err
	}

	err = s.resourceManager.ReportScheduledWorkflowResource(scheduledWorkflow)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *ReportServer) ReportScheduledWorkflowV1(ctx context.Context,
	request *apiv1beta1.ReportScheduledWorkflowRequest,
) (*empty.Empty, error) {
	return s.reportScheduledWorkflow(ctx, request.GetScheduledWorkflow())
}

func (s *ReportServer) ReportScheduledWorkflow(ctx context.Context,
	request *apiv2beta1.ReportScheduledWorkflowRequest,
) (*empty.Empty, error) {
	return s.reportScheduledWorkflow(ctx, request.GetScheduledWorkflow())
}

func validateReportWorkflowRequest(wfManifest string) (*util.ExecutionSpec, error) {
	execSpec, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(wfManifest))
	if err != nil {
		return nil, util.NewInvalidInputError("Could not unmarshal workflow: %v: %v", err, wfManifest)
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

func validateReportScheduledWorkflowRequest(swfManifest string) (*util.ScheduledWorkflow, error) {
	var scheduledWorkflow scheduledworkflow.ScheduledWorkflow
	err := json.Unmarshal([]byte(swfManifest), &scheduledWorkflow)
	if err != nil {
		return nil, util.NewInvalidInputError("Could not unmarshal scheduled workflow: %v: %v",
			err, swfManifest)
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

func (s *ReportServer) canAccessWorkflow(ctx context.Context, executionName string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to report %s `%s` due to authorization error.", resourceAttributes.Resource, executionName)
	}
	return nil
}

func NewReportServer(resourceManager *resource.ResourceManager) *ReportServer {
	return &ReportServer{resourceManager: resourceManager}
}
