// Copyright 2025 The Kubeflow Authors
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

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

type ArtifactServer struct {
	resourceManager *resource.ResourceManager
	apiv2beta1.UnimplementedArtifactServiceServer
}

// NewArtifactServer creates a new ArtifactServer.
func NewArtifactServer(resourceManager *resource.ResourceManager) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager}
}

// CreateArtifact creates a new artifact.
func (s *ArtifactServer) CreateArtifact(ctx context.Context, request *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	err := s.validateCreateArtifactRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifact due to validation error")
	}

	// Extract namespace for authorization
	namespace := s.resourceManager.ReplaceNamespace(request.GetArtifact().GetNamespace())

	// Check authorization - artifacts are accessible if user can access runs in the namespace
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	if err = s.canAccessArtifacts(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	task, err := s.validateArtifactOwnershipNoAuth(request.GetRunId(), request.GetTaskId(), namespace)
	if err != nil {
		return nil, util.Wrap(err, "Failed to validate artifact ownership")
	}
	if err := s.canAccessRun(ctx, task.RunUUID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate}); err != nil {
		return nil, util.Wrap(err, "Failed to authorize access to the task run")
	}

	modelArtifact, err := toModelArtifact(request.GetArtifact())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifact due to conversion error")
	}

	// Set the validated namespace
	modelArtifact.Namespace = namespace

	// Build the IOProducer with task name
	producer := &apiv2beta1.IOProducer{
		TaskName: task.Name,
	}
	// Add iteration index if provided
	if request.IterationIndex != nil {
		producer.Iteration = request.IterationIndex
	}

	artifactTask := &apiv2beta1.ArtifactTask{
		TaskId: task.UUID,
		RunId:  task.RunUUID,
		// An artifact at creation is an output of the associated task.
		Type:     apiv2beta1.IOType_OUTPUT,
		Producer: producer,
		Key:      request.GetProducerKey(),
	}

	modelAT, err := toModelArtifactTask(artifactTask)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert artifact_task")
	}

	artifact, _, err := s.resourceManager.CreateArtifactWithTask(modelArtifact, modelAT)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifact and artifact-task")
	}

	return toAPIArtifact(artifact)
}

// CreateArtifactsBulk creates multiple artifacts in bulk.
func (s *ArtifactServer) CreateArtifactsBulk(ctx context.Context, request *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	if request == nil || len(request.GetArtifacts()) == 0 {
		return nil, util.NewInvalidInputError("CreateArtifactsBulkRequest must contain at least one artifact")
	}

	bulkNamespace := ""
	for i, artifactReq := range request.GetArtifacts() {
		err := s.validateCreateArtifactRequest(artifactReq)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to create artifact %d due to validation error", i)
		}

		namespace := s.resourceManager.ReplaceNamespace(artifactReq.GetArtifact().GetNamespace())
		if bulkNamespace == "" {
			bulkNamespace = namespace
			continue
		}
		if namespace != bulkNamespace {
			return nil, util.NewInvalidInputError(
				"CreateArtifactsBulkRequest must use a single namespace: expected %s, got %s",
				bulkNamespace,
				namespace,
			)
		}
	}

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: bulkNamespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	if err := s.canAccessArtifacts(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	modelArtifacts := make([]*model.Artifact, 0, len(request.GetArtifacts()))
	modelArtifactTasks := make([]*model.ArtifactTask, 0, len(request.GetArtifacts()))
	authorizedRunIDs := make(map[string]struct{})

	authorizeRunUpdate := func(runID string) error {
		if runID == "" {
			return nil
		}
		if _, ok := authorizedRunIDs[runID]; ok {
			return nil
		}
		if err := s.canAccessRun(ctx, runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate}); err != nil {
			return util.Wrap(err, "Failed to authorize access to the task run")
		}
		authorizedRunIDs[runID] = struct{}{}
		return nil
	}

	// Validate and create each artifact
	for i, artifactReq := range request.GetArtifacts() {
		// Extract namespace for authorization
		namespace := s.resourceManager.ReplaceNamespace(artifactReq.GetArtifact().GetNamespace())

		task, err := s.validateArtifactOwnershipNoAuth(artifactReq.GetRunId(), artifactReq.GetTaskId(), namespace)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to validate ownership for artifact %d", i)
		}
		if err := authorizeRunUpdate(task.RunUUID); err != nil {
			return nil, util.Wrapf(err, "Failed to validate ownership for artifact %d", i)
		}

		modelArtifact, err := toModelArtifact(artifactReq.GetArtifact())
		if err != nil {
			return nil, util.Wrapf(err, "Failed to create artifact %d due to conversion error", i)
		}

		// Set the validated namespace
		modelArtifact.Namespace = namespace

		// Build the IOProducer with task name
		producer := &apiv2beta1.IOProducer{
			TaskName: task.Name,
		}
		// Add iteration index if provided
		if artifactReq.IterationIndex != nil {
			producer.Iteration = artifactReq.IterationIndex
		}

		artifactTask := &apiv2beta1.ArtifactTask{
			TaskId: task.UUID,
			RunId:  task.RunUUID,
			// An artifact at creation is an output of the associated task.
			Type:     apiv2beta1.IOType_OUTPUT,
			Producer: producer,
			Key:      artifactReq.GetProducerKey(),
		}

		modelAT, err := toModelArtifactTask(artifactTask)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to convert artifact_task for artifact %d", i)
		}
		modelArtifacts = append(modelArtifacts, modelArtifact)
		modelArtifactTasks = append(modelArtifactTasks, modelAT)
	}

	createdArtifacts, _, err := s.resourceManager.CreateArtifactsWithTasks(modelArtifacts, modelArtifactTasks)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifacts and artifact-tasks")
	}

	response := &apiv2beta1.CreateArtifactsBulkResponse{
		Artifacts: make([]*apiv2beta1.Artifact, 0, len(createdArtifacts)),
	}
	for i, artifact := range createdArtifacts {
		apiArtifact, err := toAPIArtifact(artifact)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to convert artifact %d to API", i)
		}
		response.Artifacts = append(response.Artifacts, apiArtifact)
	}

	return response, nil
}

func (s *ArtifactServer) validateArtifactOwnershipNoAuth(runID, taskID, artifactNamespace string) (*model.Task, error) {
	task, err := s.resourceManager.GetTask(taskID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get task")
	}
	if task.RunUUID != runID {
		return nil, util.NewInvalidInputError("Task ID does not belong to this Run ID")
	}
	if !common.IsMultiUserMode() {
		return task, nil
	}

	run, err := s.resourceManager.GetRun(task.RunUUID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get run")
	}

	taskNamespace := task.Namespace
	if taskNamespace == "" {
		taskNamespace = run.Namespace
	}
	if taskNamespace != artifactNamespace || run.Namespace != artifactNamespace {
		return nil, util.NewInvalidInputError(
			"artifact, task, and run must be in the same namespace: artifact=%s task=%s run=%s",
			artifactNamespace,
			taskNamespace,
			run.Namespace,
		)
	}

	return task, nil
}

// GetArtifact finds a specific artifact by ID.
func (s *ArtifactServer) GetArtifact(ctx context.Context, request *apiv2beta1.GetArtifactRequest) (*apiv2beta1.Artifact, error) {
	artifactID := request.GetArtifactId()
	if artifactID == "" {
		return nil, util.NewInvalidInputError("Artifact ID is required")
	}

	artifact, err := s.resourceManager.GetArtifact(artifactID)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get artifact")
	}

	// Check authorization using the artifact's namespace
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: artifact.Namespace,
		Verb:      common.RbacResourceVerbGet,
	}
	if err = s.canAccessArtifacts(ctx, artifactID, resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return toAPIArtifact(artifact)
}

// ListArtifacts finds all artifacts within the specified namespace.
func (s *ArtifactServer) ListArtifacts(ctx context.Context, request *apiv2beta1.ListArtifactRequest) (*apiv2beta1.ListArtifactResponse, error) {
	opts, err := validatedListOptions(&model.Artifact{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter, "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	// Handle namespace and authorization
	namespace := s.resourceManager.ReplaceNamespace(request.GetNamespace())

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err = s.canAccessArtifacts(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	filterContext, err := validateFilterV2Beta1Artifact(namespace)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed")
	}

	artifacts, totalSize, nextPageToken, err := s.resourceManager.ListArtifacts([]*model.FilterContext{filterContext}, opts)
	if err != nil {
		return nil, util.Wrap(err, "List artifacts failed")
	}

	return &apiv2beta1.ListArtifactResponse{
		Artifacts:     toAPIArtifacts(artifacts),
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}, nil
}

// CreateArtifactTask creates an artifact-task relationship.
func (s *ArtifactServer) CreateArtifactTask(ctx context.Context, request *apiv2beta1.CreateArtifactTaskRequest) (*apiv2beta1.ArtifactTask, error) {
	if request == nil || request.GetArtifactTask() == nil {
		return nil, util.NewInvalidInputError("CreateArtifactTaskRequest and artifact_task are required")
	}
	at := request.GetArtifactTask()
	if at.GetArtifactId() == "" {
		return nil, util.NewInvalidInputError("artifact_task.artifact_id is required")
	}
	if at.GetTaskId() == "" {
		return nil, util.NewInvalidInputError("artifact_task.task_id is required")
	}
	if at.GetRunId() == "" {
		return nil, util.NewInvalidInputError("artifact_task.run_id is required")
	}
	if at.GetType() == apiv2beta1.IOType_UNSPECIFIED {
		return nil, util.NewInvalidInputError("artifact_task.type is required")
	}
	if at.GetProducer() == nil {
		return nil, util.NewInvalidInputError("artifact_task.producer is required")
	}
	if at.GetKey() == "" {
		return nil, util.NewInvalidInputError("artifact_task.key is required")
	}

	// Fetch task and artifact for validation and authorization
	task, err := s.resourceManager.GetTask(at.GetTaskId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch task for CreateArtifactTask")
	}
	if task.RunUUID != at.GetRunId() {
		return nil, util.NewInvalidInputError("artifact_task.run_id must match the task's run_id")
	}
	artifact, err := s.resourceManager.GetArtifact(at.GetArtifactId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch artifact for CreateArtifactTask")
	}

	// Enforce same-namespace linkage in multi-user mode.
	if common.IsMultiUserMode() && task.Namespace != "" && artifact.Namespace != "" && task.Namespace != artifact.Namespace {
		return nil, util.NewInvalidInputError("artifact and task must be in the same namespace: artifact=%s task=%s", artifact.Namespace, task.Namespace)
	}

	// Authorize create in the task's namespace
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: task.Namespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	if err = s.canAccessArtifacts(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	if err = s.canAccessRun(ctx, at.GetRunId(), &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate}); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	modelAT, err := toModelArtifactTask(at)
	if err != nil {
		return nil, util.Wrap(err, "Failed to convert artifact_task")
	}
	modelAT.RunUUID = task.RunUUID

	created, err := s.resourceManager.CreateArtifactTask(modelAT)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifact-task")
	}
	return toAPIArtifactTask(created), nil
}

// ListArtifactTasks lists artifact-task relationships.
func (s *ArtifactServer) ListArtifactTasks(ctx context.Context, request *apiv2beta1.ListArtifactTasksRequest) (*apiv2beta1.ListArtifactTasksResponse, error) {
	opts, err := validatedListOptions(&model.ArtifactTask{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter, "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	// Authorization check - we need to verify access to the runs/namespaces involved
	// For now, require at least one filter to determine namespace context
	if len(request.TaskIds) == 0 && len(request.RunIds) == 0 && len(request.ArtifactIds) == 0 {
		return nil, util.NewInvalidInputError("At least one filter (task_ids, run_ids, or artifact_ids) is required")
	}

	// Check authorization based on provided filters
	err = s.authorizeArtifactTaskAccess(ctx, request.TaskIds, request.RunIds, request.ArtifactIds)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	filterContexts, err := validateFilterV2Beta1ArtifactTask(request.TaskIds, request.RunIds, request.ArtifactIds)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed")
	}

	// Convert IOType from proto to model if provided
	var ioType *model.IOType
	if request.Type != apiv2beta1.IOType_UNSPECIFIED {
		modelIOType := model.IOType(request.Type)
		ioType = &modelIOType
	}

	artifactTasks, totalSize, nextPageToken, err := s.resourceManager.ListArtifactTasks(filterContexts, ioType, opts)
	if err != nil {
		return nil, util.Wrap(err, "List artifact tasks failed")
	}

	return &apiv2beta1.ListArtifactTasksResponse{
		ArtifactTasks: toAPIArtifactTasks(artifactTasks),
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}, nil
}

// CreateArtifactTasksBulk creates multiple artifact-task relationships in bulk.
func (s *ArtifactServer) CreateArtifactTasksBulk(ctx context.Context, request *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	if request == nil || len(request.GetArtifactTasks()) == 0 {
		return nil, util.NewInvalidInputError("CreateArtifactTasksBulkRequest must contain at least one artifact task")
	}

	// Validate all artifact tasks and check authorization
	modelArtifactTasks := make([]*model.ArtifactTask, 0, len(request.GetArtifactTasks()))
	bulkRunID := ""
	bulkNamespace := ""
	for _, apiAT := range request.GetArtifactTasks() {
		if apiAT.GetArtifactId() == "" {
			return nil, util.NewInvalidInputError("artifact_task.artifact_id is required")
		}
		if apiAT.GetTaskId() == "" {
			return nil, util.NewInvalidInputError("artifact_task.task_id is required")
		}
		if apiAT.GetRunId() == "" {
			return nil, util.NewInvalidInputError("artifact_task.run_id is required")
		}
		if apiAT.GetType() == apiv2beta1.IOType_UNSPECIFIED {
			return nil, util.NewInvalidInputError("artifact_task.type is required")
		}
		if apiAT.GetProducer() == nil {
			return nil, util.NewInvalidInputError("artifact_task.producer is required")
		}
		if apiAT.GetKey() == "" {
			return nil, util.NewInvalidInputError("artifact_task.key is required")
		}

		// Fetch task and artifact for validation and authorization
		task, err := s.resourceManager.GetTask(apiAT.GetTaskId())
		if err != nil {
			return nil, util.Wrap(err, "Failed to fetch task for CreateArtifactTasksBulk")
		}
		if task.RunUUID != apiAT.GetRunId() {
			return nil, util.NewInvalidInputError("artifact_task.run_id must match the task's run_id")
		}
		artifact, err := s.resourceManager.GetArtifact(apiAT.GetArtifactId())
		if err != nil {
			return nil, util.Wrap(err, "Failed to fetch artifact for CreateArtifactTasksBulk")
		}
		if task.RunUUID != apiAT.GetRunId() {
			return nil, util.NewInvalidInputError("artifact_task.run_id must match the task's run_id")
		}
		if bulkRunID == "" {
			bulkRunID = task.RunUUID
		} else if task.RunUUID != bulkRunID {
			return nil, util.NewInvalidInputError("CreateArtifactTasksBulkRequest must use a single run_id")
		}

		// Optional: enforce same-namespace linkage
		if common.IsMultiUserMode() && task.Namespace != "" && artifact.Namespace != "" && task.Namespace != artifact.Namespace {
			return nil, util.NewInvalidInputError("artifact and task must be in the same namespace: artifact=%s task=%s", artifact.Namespace, task.Namespace)
		}
		if bulkNamespace == "" {
			bulkNamespace = task.Namespace
		} else if task.Namespace != bulkNamespace {
			return nil, util.NewInvalidInputError("CreateArtifactTasksBulkRequest must use a single namespace")
		}

		modelAT, err := toModelArtifactTask(apiAT)
		if err != nil {
			return nil, util.Wrap(err, "Failed to convert artifact_task")
		}
		modelAT.RunUUID = task.RunUUID
		modelArtifactTasks = append(modelArtifactTasks, modelAT)
	}

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: bulkNamespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	if err := s.canAccessArtifacts(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	if err := s.canAccessRun(ctx, bulkRunID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate}); err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	// Create all artifact tasks in bulk
	createdArtifactTasks, err := s.resourceManager.CreateArtifactTasks(modelArtifactTasks)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create artifact-tasks in bulk")
	}

	return &apiv2beta1.CreateArtifactTasksBulkResponse{
		ArtifactTasks: toAPIArtifactTasks(createdArtifactTasks),
	}, nil
}

// Authorization helper functions

// canAccessRun checks if the user can access runs in the given namespace
// Following the same pattern as BaseRunServer.canAccessRun
func (s *ArtifactServer) canAccessRun(ctx context.Context, runID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}

	if runID != "" {
		run, err := s.resourceManager.GetRun(runID)
		if err != nil {
			return util.Wrapf(err, "Failed to authorize with the run ID %v", runID)
		}
		if s.resourceManager.IsEmptyNamespace(run.Namespace) {
			experiment, err := s.resourceManager.GetExperiment(run.ExperimentId)
			if err != nil {
				return util.NewInvalidInputError("run %v has an empty namespace and the parent experiment %v could not be fetched: %s", runID, run.ExperimentId, err.Error())
			}
			resourceAttributes.Namespace = experiment.Namespace
		} else {
			resourceAttributes.Namespace = run.Namespace
		}
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = run.K8SName
		}
	}

	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return util.NewInvalidInputError("A resource cannot have an empty namespace in multi-user mode")
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeRuns
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to access resource. Check if you have access to namespace %s", resourceAttributes.Namespace)
	}
	return nil
}

func (s *ArtifactServer) canAccessArtifacts(ctx context.Context, artifactID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}

	if artifactID != "" {
		artifact, err := s.resourceManager.GetArtifact(artifactID)
		if err != nil {
			return util.Wrapf(err, "Failed to authorize with the artifact ID %v", artifactID)
		}
		if s.resourceManager.IsEmptyNamespace(artifact.Namespace) {
			return util.NewInvalidInputError("artifact %v has an empty namespace", artifactID)
		}
		resourceAttributes.Namespace = artifact.Namespace
	}

	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return util.NewInvalidInputError("A resource cannot have an empty namespace in multi-user mode")
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeArtifacts
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to access resource. Check if you have access to namespace %s", resourceAttributes.Namespace)
	}
	return nil
}

// authorizeArtifactTaskAccess authorizes access to artifact-task relationships
// TODO(HumairAK): Make this more efficient by doing bulk calls to the database,
// and aggregating namespaces down to unique namespace calls
func (s *ArtifactServer) authorizeArtifactTaskAccess(ctx context.Context, taskIDs, runIDs, artifactIDs []string) error {
	authorizedRunIDs := make(map[string]struct{})
	authorizedNamespaces := make(map[string]struct{})

	authorizeRunID := func(runID string) error {
		if runID == "" {
			return nil
		}
		if _, ok := authorizedRunIDs[runID]; ok {
			return nil
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Verb: common.RbacResourceVerbGet,
		}
		if err := s.canAccessRun(ctx, runID, resourceAttributes); err != nil {
			return err
		}
		authorizedRunIDs[runID] = struct{}{}
		return nil
	}

	authorizeNamespace := func(namespace string) error {
		if namespace == "" {
			return nil
		}
		if _, ok := authorizedNamespaces[namespace]; ok {
			return nil
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbGet,
		}
		if err := s.canAccessRun(ctx, "", resourceAttributes); err != nil {
			return err
		}
		authorizedNamespaces[namespace] = struct{}{}
		return nil
	}

	// Check authorization for run IDs (direct access)
	for _, runID := range runIDs {
		if err := authorizeRunID(runID); err != nil {
			return err
		}
	}

	// Check authorization for task IDs (get namespace from task)
	for _, taskID := range taskIDs {
		task, err := s.resourceManager.GetTask(taskID)
		if err != nil {
			return util.Wrap(err, "Failed to get task for authorization")
		}
		if err = authorizeRunID(task.RunUUID); err != nil {
			return err
		}
	}

	// Check authorization for artifact IDs (get namespace from artifact)
	for _, artifactID := range artifactIDs {
		artifact, err := s.resourceManager.GetArtifact(artifactID)
		if err != nil {
			return util.Wrap(err, "Failed to get artifact for authorization")
		}
		if err = authorizeNamespace(artifact.Namespace); err != nil {
			return err
		}
	}
	return nil
}

func (s *ArtifactServer) validateCreateArtifactRequest(request *apiv2beta1.CreateArtifactRequest) error {
	if request == nil {
		return util.NewInvalidInputError("CreateArtifactRequest is nil")
	}
	artifact := request.GetArtifact()
	if artifact == nil {
		return util.NewInvalidInputError("Artifact is required")
	}
	if artifact.GetArtifactId() != "" {
		return util.NewInvalidInputError("Artifact ID should not be set on create")
	}
	if artifact.GetNamespace() == "" {
		return util.NewInvalidInputError("Artifact namespace is required")
	}
	if request.GetArtifact().GetType() == apiv2beta1.Artifact_TYPE_UNSPECIFIED {
		return util.NewInvalidInputError("Artifact type is required")
	}
	if request.GetArtifact().GetName() == "" {
		return util.NewInvalidInputError("Artifact name is required")
	}
	if request.GetRunId() == "" {
		return util.NewInvalidInputError("Run ID is required")
	}
	if request.GetTaskId() == "" {
		return util.NewInvalidInputError("Task ID is required")
	}
	if request.GetProducerKey() == "" {
		return util.NewInvalidInputError("Producer key is required")
	}
	// Metrics validation
	if request.GetArtifact().GetType() == apiv2beta1.Artifact_Metric &&
		request.GetArtifact().NumberValue == nil {
		return util.NewInvalidInputError("number_value is required for a Metric artifact")
	}
	if (request.GetArtifact().GetType() == apiv2beta1.Artifact_ClassificationMetric ||
		request.GetArtifact().GetType() == apiv2beta1.Artifact_SlicedClassificationMetric) &&
		request.GetArtifact().GetMetadata() == nil {
		return util.NewInvalidInputError("No metric or metadata was found for %s artifact", request.GetArtifact().GetType())
	}
	if request.GetProducerKey() == "" {
		return util.NewInvalidInputError("Producer key is required")
	}
	return nil
}
