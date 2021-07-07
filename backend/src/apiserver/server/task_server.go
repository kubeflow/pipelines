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
	if request == nil {
		return util.NewInvalidInputError("CreatTaskRequst is nil")
	}
	task := *request.Task

	errMustSpecify := func(s string) error { return util.NewInvalidInputError("Invalid task: must specify %s", s) }

	if task.GetId() != "" {
		return util.NewInvalidInputError("Invalid task: Id should not be set")
	}
	if task.GetNamespace() == "" {
		return errMustSpecify("Namespace")
	}
	if task.GetPipelineName() == "" {
		return errMustSpecify("PipelineName")
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

func NewTaskServer(resourceManager *resource.ResourceManager) *TaskServer {
	return &TaskServer{resourceManager: resourceManager}
}
