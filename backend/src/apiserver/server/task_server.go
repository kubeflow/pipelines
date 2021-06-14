package server

import (
	"context"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type TaskServer struct {
	resourceManager *resource.ResourceManager
}

func (s *TaskServer) CreateTask(ctx context.Context, request *api.CreateTaskRequest) (*api.Task, error) {
	err := s.validateCreateTaskRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create task request failed.")
	}

	task, err := s.resourceManager.CreateTask(ctx, request.Task)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task.")
	}

	return ToApiTask(task), nil
}

func (s *TaskServer) validateCreateTaskRequest(request *api.CreateTaskRequest) error {
	task := *request.Task

	empty := func(s string) bool { return len(s) == 0 }
	err := func(s string) error { return util.NewInvalidInputError("Invalid task: must specify %s", s) }

	if empty(task.GetId()) {
		return util.NewInvalidInputError("Invalid task: Id should not be set")
	}
	if empty(task.GetNamespace()) {
		return err("Namespace")
	}
	if empty(task.GetPipelineName()) {
		return err("PipelineName")
	}
	if empty(task.GetRunId()) {
		return err("RunId")
	}
	if empty(task.GetMlmdExecutionID()) {
		return err("MlmdExecutionID")
	}
	if empty(task.GetFingerPrint()) {
		return err("FingerPrint")
	}
	if task.GetCreatedAt() == nil {
		return err("CreatedAt")
	}
	return nil
}

func (s *TaskServer) ListTasks(ctx context.Context, request *api.ListTasksRequest) (
	*api.ListTasksResponse, error) {

	opts, err := validatedListOptions(&model.Task{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}

	tasks, total_size, nextPageToken, err := s.resourceManager.ListTasks(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "List tasks failed.")
	}
	return &api.ListTasksResponse{
			Tasks:         ToApiTasks(tasks),
			TotalSize:     int32(total_size),
			NextPageToken: nextPageToken},
		nil
}
