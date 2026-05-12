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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

type TaskServer struct {
	*BaseRunServer
	apiv1beta1.UnimplementedTaskServiceServer
}

// Creates a task.
// Supports v1beta1 behavior.
func (s *TaskServer) CreateTaskV1(ctx context.Context, request *api.CreateTaskRequest) (*api.Task, error) {
	err := s.validateCreateTaskRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a new task due to validation error")
	}

	if err := s.canAccessRun(ctx, request.GetTask().GetRunId(), &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbCreate}); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
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

	// Resolve the namespace the caller is asking for and authorize the list
	// against that namespace. Tasks belong to runs, so we reuse the runs
	// RBAC verb model (the same pattern RunArtifactServer follows).
	namespace, err := s.resolveListTasksNamespace(filterContext)
	if err != nil {
		return nil, util.Wrap(err, "Failed to resolve namespace for list tasks request")
	}
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err := s.canAccessRun(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to list tasks due to authorization error. Check if you have permission to access namespace %s", namespace)
	}
	// Ensure the storage query is scoped to the authorized namespace when no
	// other reference key is present, so TaskStore.ListTasks applies a
	// Namespace WHERE clause and does not leak rows across tenants.
	if filterContext.ReferenceKey == nil && namespace != "" {
		filterContext.ReferenceKey = &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace}
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

// resolveListTasksNamespace extracts the caller's intended namespace from a
// v1beta1 ListTasksRequest filter. A NAMESPACE reference key carries the
// namespace directly; a RUN reference key resolves it via the run's parent
// experiment. Other reference key types (and empty) yield "" — the
// authorization check enforces an empty-namespace ban in multi-user mode.
func (s *TaskServer) resolveListTasksNamespace(filterContext *model.FilterContext) (string, error) {
	if filterContext == nil || filterContext.ReferenceKey == nil {
		return "", nil
	}
	switch filterContext.ReferenceKey.Type {
	case model.NamespaceResourceType:
		return filterContext.ReferenceKey.ID, nil
	case model.RunResourceType:
		run, err := s.resourceManager.GetRun(filterContext.ReferenceKey.ID)
		if err != nil {
			return "", util.Wrapf(err, "Failed to resolve namespace for run %s", filterContext.ReferenceKey.ID)
		}
		if !s.resourceManager.IsEmptyNamespace(run.Namespace) {
			return run.Namespace, nil
		}
		return s.resourceManager.GetNamespaceFromExperimentId(run.ExperimentId)
	}
	return "", nil
}

func NewTaskServer(resourceManager *resource.ResourceManager) *TaskServer {
	return &TaskServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         &RunServerOptions{CollectMetrics: false},
		},
	}
}
