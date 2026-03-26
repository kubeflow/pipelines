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
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type TaskServerV1 struct {
	resourceManager *resource.ResourceManager
	apiv1beta1.UnimplementedTaskServiceServer
}
type TaskServerV2 struct {
	resourceManager *resource.ResourceManager
	apiv2beta1.UnimplementedTaskServiceServer
}

// Creates a task.
// Supports v1beta1 behavior.
func (s *TaskServerV1) CreateTaskV1(ctx context.Context, request *apiv1beta1.CreateTaskRequest) (*apiv1beta1.Task, error) {
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

func (s *TaskServerV1) validateCreateTaskRequest(request *apiv1beta1.CreateTaskRequest) error {
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
func (s *TaskServerV1) ListTasksV1(ctx context.Context, request *apiv1beta1.ListTasksRequest) (
	*apiv1beta1.ListTasksResponse, error,
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
	return &apiv1beta1.ListTasksResponse{
			Tasks:         toApiTasksV1(tasks),
			TotalSize:     int32(total_size),
			NextPageToken: nextPageToken,
		},
		nil
}
func (s *TaskServerV2) GetPipelineTask(ctx context.Context, request *apiv2beta1.GetPipelineTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	if request == nil {
		return nil, util.NewInvalidInputError("GetPipelineTaskRequest is nil")
	}
	taskID := request.GetTaskId()
	if taskID == "" {
		return nil, util.NewInvalidInputError("task_id must be provided")
	}
	task, err := s.resourceManager.GetTask(taskID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get a task")
	}
	return toApiPipelineTaskDetail(task), nil
}
func (s *TaskServerV2) ListPipelineTasks(ctx context.Context, request *apiv2beta1.ListPipelineTasksRequest) (*apiv2beta1.ListPipelineTasksResponse, error) {
	opts, err := validatedListOptions(&model.Task{}, request.GetPageToken(), int(request.GetPageSize()), request.GetSortBy(), request.GetFilter(),
		"v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}
	runID := request.GetRunId()
	var filterContext *model.FilterContext
	if runID != "" {
		filterContext = &model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID},
		}
	}
	tasks, totalSize, nextPageToken, err := s.resourceManager.ListTasks(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "Failed to list tasks")
	}
	return &apiv2beta1.ListPipelineTasksResponse{
		Tasks:         toApiPipelineTaskDetails(tasks),
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}, nil
}
func NewTaskServer(resourceManager *resource.ResourceManager) *TaskServerV2 {
	return &TaskServerV2{resourceManager: resourceManager}
}

func NewTaskServerV1(resourceManager *resource.ResourceManager) *TaskServerV1 {
	return &TaskServerV1{resourceManager: resourceManager}
}
