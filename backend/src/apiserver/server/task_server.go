package server

import (
	"context"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
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

