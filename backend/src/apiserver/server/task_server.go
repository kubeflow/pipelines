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

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

type TaskServer struct {
	runServer *BaseRunServer
	api.UnimplementedTaskServiceServer
}

// Creates a task.
// Supports v1beta1 behavior.
func (s *TaskServer) CreateTaskV1(ctx context.Context, request *api.CreateTaskRequest) (*api.Task, error) {
	err := s.validateCreateTaskRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task due to validation error")
	}

	if err := s.runServer.canAccessRun(ctx, request.GetTask().GetRunId(), &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbCreateTask}); err != nil {
		return nil, util.Wrap(err, "Failed to authorize task creation")
	}
	modelTask, err := toModelTask(request.GetTask())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task due to conversion error")
	}

	task, err := s.runServer.resourceManager.CreateTask(modelTask)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task")
	}

	return toApiTaskV1(task), nil
}

func (s *TaskServer) validateCreateTaskRequest(request *api.CreateTaskRequest) error {
	if request == nil {
		return util.NewInvalidInputError("CreateTaskRequest is nil")
	}
	task := request.GetTask()

	errMustSpecify := func(s string) error { return util.NewInvalidInputError("Invalid task: must specify %s", s) }

	if task.GetId() != "" {
		return util.NewInvalidInputError("Invalid task: Id should not be set")
	}
	if task.GetPipelineName() == "" {
		return errMustSpecify("PipelineName")
	}
	if err := util.ValidateTaskPipelineNamespace(task.GetPipelineName(), task.GetNamespace()); err != nil {
		return err
	}
	if task.GetRunId() == "" {
		return errMustSpecify("RunID")
	}
	if task.GetMlmdExecutionID() == "" {
		return errMustSpecify("MlmdExecutionID")
	}
	if task.GetFingerprint() == "" {
		return errMustSpecify("FingerPrint")
	}
	if task.GetCreatedAt() == nil {
		return errMustSpecify("CreatedAt")
	}
	return nil
}

// Fetches tasks given query parameters.
// Supports v1beta1 behavior.
func (s *TaskServer) ListTasksV1(ctx context.Context, request *api.ListTasksRequest) (
	*api.ListTasksResponse, error,
) {
	opts, err := validatedListOptions(&model.Task{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter, "v1beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext, err := validateFilterV1(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed")
	}

	tasks, totalSize, nextPageToken, err := s.runServer.resourceManager.ListTasks(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "List tasks failed")
	}
	return &api.ListTasksResponse{
			Tasks:         toApiTasksV1(tasks),
			TotalSize:     int32(totalSize),
			NextPageToken: nextPageToken,
		},
		nil
}

func NewTaskServer(resourceManager *resource.ResourceManager) *TaskServer {
	return &TaskServer{runServer: &BaseRunServer{resourceManager: resourceManager}}
}
