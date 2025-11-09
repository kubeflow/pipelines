// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kfpapi

import (
	"context"
	"fmt"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// API is a minimal interface exposing KFP API operations needed by drivers and launchers.
// It abstracts over RunService, ArtifactService, and PipelineService.
//
// This indirection lets us unit test components and also evolve the underlying
// client without touching driver/launcher logic.
//
// Note: We intentionally do not expose the full apiclient.Client here.
// Only the small surface area needed is included.

type API interface {
	// Run operations
	GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error)

	// Task operations
	CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error)
	UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error)
	UpdateTasksBulk(ctx context.Context, req *gc.UpdateTasksBulkRequest) (*gc.UpdateTasksBulkResponse, error)
	GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error)
	ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error)

	// Artifact operations
	CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error)
	CreateArtifactsBulk(ctx context.Context, req *gc.CreateArtifactsBulkRequest) (*gc.CreateArtifactsBulkResponse, error)
	ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error)
	ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error)
	CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error)
	CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error)

	// Pipeline version operations
	GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error)
	FetchPipelineSpecFromRun(ctx context.Context, run *gc.Run) (*structpb.Struct, error)

	// Propagate status updates up the DAG
	UpdateStatuses(ctx context.Context, run *gc.Run, pipelineSpec *structpb.Struct, currentTask *gc.PipelineTaskDetail) error
}

// clientAdapter adapts apiclient.Client to API.
// It is a thin wrapper delegating to the generated gRPC clients.

type clientAdapter struct {
	c *apiclient.Client
}

// New wraps the apiclient.Client into an API interface.
func New(c *apiclient.Client) API {
	return &clientAdapter{c: c}
}

// Implement API by forwarding calls to typed clients.

func (k *clientAdapter) GetRun(ctx context.Context, req *gc.GetRunRequest) (*gc.Run, error) {
	return k.c.Run.GetRun(ctx, req)
}

func (k *clientAdapter) CreateTask(ctx context.Context, req *gc.CreateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.CreateTask(ctx, req)
}

func (k *clientAdapter) UpdateTask(ctx context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.UpdateTask(ctx, req)
}

func (k *clientAdapter) UpdateTasksBulk(ctx context.Context, req *gc.UpdateTasksBulkRequest) (*gc.UpdateTasksBulkResponse, error) {
	return k.c.Run.UpdateTasksBulk(ctx, req)
}

func (k *clientAdapter) GetTask(ctx context.Context, req *gc.GetTaskRequest) (*gc.PipelineTaskDetail, error) {
	return k.c.Run.GetTask(ctx, req)
}

func (k *clientAdapter) ListTasks(ctx context.Context, req *gc.ListTasksRequest) (*gc.ListTasksResponse, error) {
	return k.c.Run.ListTasks(ctx, req)
}

func (k *clientAdapter) CreateArtifact(ctx context.Context, req *gc.CreateArtifactRequest) (*gc.Artifact, error) {
	return k.c.Artifact.CreateArtifact(ctx, req)
}

func (k *clientAdapter) CreateArtifactsBulk(ctx context.Context, req *gc.CreateArtifactsBulkRequest) (*gc.CreateArtifactsBulkResponse, error) {
	return k.c.Artifact.CreateArtifactsBulk(ctx, req)
}

func (k *clientAdapter) ListArtifactsByURI(ctx context.Context, uri, namespace string) ([]*gc.Artifact, error) {
	predicates := []*gc.Predicate{
		{Key: "uri", Operation: gc.Predicate_EQUALS, Value: &gc.Predicate_StringValue{StringValue: uri}},
	}
	filter := &gc.Filter{
		Predicates: predicates,
	}
	mo := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	filterJSON, err := mo.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter: %v", err)
	}

	const pageSize = 100
	var allArtifacts []*gc.Artifact
	nextPageToken := ""

	for {
		artifactsResponse, err := k.c.Artifact.ListArtifacts(ctx, &gc.ListArtifactRequest{
			Namespace: namespace,
			Filter:    string(filterJSON),
			PageSize:  pageSize,
			PageToken: nextPageToken,
		})
		if err != nil {
			return nil, err
		}

		allArtifacts = append(allArtifacts, artifactsResponse.GetArtifacts()...)
		nextPageToken = artifactsResponse.GetNextPageToken()

		if nextPageToken == "" {
			break
		}
	}

	return allArtifacts, nil
}

func (k *clientAdapter) ListArtifactTasks(ctx context.Context, req *gc.ListArtifactTasksRequest) (*gc.ListArtifactTasksResponse, error) {
	return k.c.Artifact.ListArtifactTasks(ctx, req)
}

func (k *clientAdapter) CreateArtifactTask(ctx context.Context, req *gc.CreateArtifactTaskRequest) (*gc.ArtifactTask, error) {
	return k.c.Artifact.CreateArtifactTask(ctx, req)
}

func (k *clientAdapter) CreateArtifactTasks(ctx context.Context, req *gc.CreateArtifactTasksBulkRequest) (*gc.CreateArtifactTasksBulkResponse, error) {
	return k.c.Artifact.CreateArtifactTasksBulk(ctx, req)
}

func (k *clientAdapter) GetPipelineVersion(ctx context.Context, req *gc.GetPipelineVersionRequest) (*gc.PipelineVersion, error) {
	return k.c.Pipeline.GetPipelineVersion(ctx, req)
}

func (k *clientAdapter) FetchPipelineSpecFromRun(ctx context.Context, run *gc.Run) (*structpb.Struct, error) {
	var pipelineSpecStruct *structpb.Struct
	if run.GetPipelineSpec() != nil {
		pipelineSpecStruct = run.GetPipelineSpec()
	} else if run.GetPipelineVersionReference() != nil {
		pvr := run.GetPipelineVersionReference()
		pipeline, err := k.GetPipelineVersion(ctx, &gc.GetPipelineVersionRequest{
			PipelineId:        pvr.GetPipelineId(),
			PipelineVersionId: pvr.GetPipelineVersionId(),
		})
		if err != nil {
			return nil, err
		}
		pipelineSpecStruct = pipeline.GetPipelineSpec()
	} else {
		return nil, fmt.Errorf("pipeline spec is not set")
	}
	// When platform_spec is included, then the structure of the PipelineSpec is different.
	if spec, ok := pipelineSpecStruct.GetFields()["pipeline_spec"]; ok {
		return spec.GetStructValue(), nil
	}
	return pipelineSpecStruct, nil
}

// UpdateStatuses Traverse up the dag until we find a parent task that still has other children with "RUNNING" status
// or when we have reached Root. If the parent task has other children in running that means this parent is also running.
// However, if the currentTask is a parent task, and all children tasks have been created (though not necessarily completed):
//   - if all children in this DAG are all CACHED, then the currentTask should be updated to be "CACHED"
//   - if any of the children in this DAG are FAILED, then the currentTask should be updated to be "FAILED"
//   - if all children in this DAG were SKIPPED, then the currentTask should be updated to be "SKIPPED"
//   - In any other case the state is SUCCEEDED
//
// TODO(HumairAK): Let's have API Server handle this call instead of doing it here.
func (k *clientAdapter) UpdateStatuses(ctx context.Context, run *gc.Run, pipelineSpec *structpb.Struct, currentTask *gc.PipelineTaskDetail) error {
	return updateStatuses(ctx, run, k, pipelineSpec, currentTask)
}

// updateStatuses traverses up the dag until we find a parent task that still has other children with "RUNNING" status
// or when we have reached Root.
// This function is separated from UpdateStatuses so that it can be used by the mock api client in tests.
func updateStatuses(ctx context.Context, run *gc.Run, kfpAPIClient API, pipelineSpec *structpb.Struct, currentTask *gc.PipelineTaskDetail) error {
	// Create a map of task IDs to tasks for quick lookup
	taskMap := make(map[string]*gc.PipelineTaskDetail)
	for _, task := range run.GetTasks() {
		taskMap[task.GetTaskId()] = task
	}

	// Start with the current task and traverse up
	for {
		// If current task has no parent, we've reached the root
		if currentTask.ParentTaskId == nil || *currentTask.ParentTaskId == "" {
			// Evaluate the root task's status based on its children
			if err := evaluateAndUpdateParentStatus(ctx, run, currentTask, kfpAPIClient); err != nil {
				return fmt.Errorf("failed to evaluate root task %s status: %w", currentTask.GetTaskId(), err)
			}
			break
		}

		// Get the parent task
		parentTask, exists := taskMap[*currentTask.ParentTaskId]
		if !exists {
			return fmt.Errorf("parent task %s not found for task %s", *currentTask.ParentTaskId, currentTask.GetTaskId())
		}

		// Determine the total number of child tasks by inspecting the parent dag's
		// task count within it's component spec.
		// We need to use the parent task's scope path, not the current task's scope path
		// Note this doesn't factor in the number of iterations of these child tasks when in a loop.
		getScopePath, err := util.ScopePathFromStringPath(pipelineSpec, parentTask.GetScopePath())
		if err != nil {
			return fmt.Errorf("failed to get scope path for parent task %s: %w", parentTask.GetTaskId(), err)
		}
		if getScopePath.GetLast() == nil || getScopePath.GetLast().GetComponentSpec() == nil || getScopePath.GetLast().GetComponentSpec().GetDag() == nil {
			return fmt.Errorf("failed to get dag for parent task %s (scope: %s): component spec or dag is nil", parentTask.GetTaskId(), parentTask.GetScopePath())
		}
		getScopePath.GetLast().GetComponentSpec().GetDag().GetTasks()
		numberOfTasksInThisDag := len(getScopePath.GetLast().GetComponentSpec().GetDag().GetTasks())

		// Before we proceed to update this parent task's status, we need to ensure that all child tasks have been
		// created (irrespective of their status).
		var expectedTotalChildTasks int
		if parentTask.GetType() == gc.PipelineTaskDetail_LOOP {
			typeAttrs := parentTask.GetTypeAttributes()
			if typeAttrs == nil || typeAttrs.IterationCount == nil {
				return fmt.Errorf("loop task %s is missing iteration_count attribute", parentTask.GetTaskId())
			}
			expectedTotalChildTasks = int(*typeAttrs.IterationCount) * numberOfTasksInThisDag
		} else {
			expectedTotalChildTasks = numberOfTasksInThisDag
		}
		// Now count the actual number of child tasks created.
		var childCount int
		for _, task := range run.GetTasks() {
			if task.ParentTaskId != nil && *task.ParentTaskId == parentTask.GetTaskId() {
				if task.GetState() == gc.PipelineTaskDetail_RUNNING {
					return nil
				}
				childCount++
			}
		}

		// If not all children created yet, exit traversal
		if childCount < expectedTotalChildTasks {
			return nil
		}

		// Evaluate and update parent's status based on its children
		if err := evaluateAndUpdateParentStatus(ctx, run, parentTask, kfpAPIClient); err != nil {
			return fmt.Errorf("failed to evaluate parent task %s status: %w", parentTask.GetTaskId(), err)
		}

		// Move to the parent for next iteration
		currentTask = parentTask
	}
	return nil
}

// evaluateAndUpdateParentStatus evaluates a parent task's status based on its direct children and updates it accordingly
func evaluateAndUpdateParentStatus(
	ctx context.Context,
	run *gc.Run,
	parentTask *gc.PipelineTaskDetail,
	kfpAPIClient API,
) error {
	// Collect all direct children of this parent
	var children []*gc.PipelineTaskDetail
	for _, task := range run.GetTasks() {
		if task.ParentTaskId != nil && *task.ParentTaskId == parentTask.GetTaskId() {
			children = append(children, task)
		}
	}

	// If no children, nothing to evaluate
	if len(children) == 0 {
		return nil
	}

	// Evaluate child statuses
	allCached := true
	allSkipped := true
	anyFailed := false

	for _, child := range children {
		status := child.GetState()

		// Check for FAILED
		if status == gc.PipelineTaskDetail_FAILED {
			anyFailed = true
		}

		// Check if all are CACHED
		if status != gc.PipelineTaskDetail_CACHED {
			allCached = false
		}

		// Check if all are SKIPPED
		if status != gc.PipelineTaskDetail_SKIPPED {
			allSkipped = false
		}
	}

	// Determine the new status for the parent
	var newStatus gc.PipelineTaskDetail_TaskState
	if anyFailed {
		newStatus = gc.PipelineTaskDetail_FAILED
	} else if allCached {
		newStatus = gc.PipelineTaskDetail_CACHED
	} else if allSkipped {
		newStatus = gc.PipelineTaskDetail_SKIPPED
	} else {
		newStatus = gc.PipelineTaskDetail_SUCCEEDED
	}

	// Update the parent task status
	parentTask.State = newStatus
	parentTask.EndTime = timestamppb.New(time.Now())
	_, err := kfpAPIClient.UpdateTask(ctx, &gc.UpdateTaskRequest{
		TaskId: parentTask.GetTaskId(),
		Task:   parentTask,
	})
	if err != nil {
		return fmt.Errorf("failed to update parent task %s status to %s: %w", parentTask.GetTaskId(), newStatus, err)
	}

	return nil
}
