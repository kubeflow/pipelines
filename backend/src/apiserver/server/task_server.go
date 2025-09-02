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
	"strings"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type TaskServer struct {
	resourceManager *resource.ResourceManager
	apiv1beta1.UnimplementedTaskServiceServer
}

// Creates a task.
// Supports v1beta1 behavior.
func (s *TaskServer) CreateTaskV1(ctx context.Context, request *api.CreateTaskRequest) (*api.Task, error) {
	err := s.validateCreateTaskRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task due to validation error")
	}

	modelTask, err := toModelTask(request.GetTask())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task due to conversion error")
	}

	task, err := s.resourceManager.CreateTask(modelTask)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task")
	}

	return toApiTaskV1(task), nil
}

func (s *TaskServer) validateCreateTaskRequest(request *api.CreateTaskRequest) error {
	if request == nil {
		return util.NewInvalidInputError("CreatTaskRequest is nil")
	}
	task := request.GetTask()

	errMustSpecify := func(s string) error { return util.NewInvalidInputError("Invalid task: must specify %s", s) }

	if task.GetId() != "" {
		return util.NewInvalidInputError("Invalid task: Id should not be set")
	}
	if task.GetPipelineName() == "" {
		return errMustSpecify("PipelineName")
	}
	if strings.HasPrefix(task.GetPipelineName(), "namespace/") {
		s := strings.SplitN(task.GetPipelineName(), "/", 4)
		if len(s) != 4 {
			return util.NewInvalidInputError("invalid PipelineName for namespaced pipelines, need to follow 'namespace/${namespace}/pipeline/${pipelineName}': %s", task.GetPipelineName())
		}
		namespace := s[1]
		if task.GetNamespace() != "" && namespace != task.GetNamespace() {
			return util.NewInvalidInputError("the namespace %s extracted from pipelineName is not equal to the namespace %s in task", namespace, task.GetNamespace())
		}
	}
	if task.GetRunId() == "" {
		return errMustSpecify("RunId")
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

	tasks, total_size, nextPageToken, err := s.resourceManager.ListTasks(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "List tasks failed")
	}
	return &api.ListTasksResponse{
			Tasks:         toApiTasksV1(tasks),
			TotalSize:     int32(total_size),
			NextPageToken: nextPageToken,
		},
		nil
}

func NewTaskServer(resourceManager *resource.ResourceManager) *TaskServer {
	return &TaskServer{resourceManager: resourceManager}
}
