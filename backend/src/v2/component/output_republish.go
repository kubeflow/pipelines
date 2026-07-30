// Copyright 2026 The Kubeflow Authors
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

package component

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"google.golang.org/protobuf/types/known/structpb"
)

// republishChildTasksPageSize is the ListTasks page size used when collecting
// preserved children. Tests may lower it to exercise pagination.
var republishChildTasksPageSize int32 = 200

// DAGOutputRepublishOptions configures re-propagation of preserved child
// outputs into a DAG/ROOT parent whose attempt-local outputs were cleared on
// retry.
type DAGOutputRepublishOptions struct {
	Run          *apiV2beta1.Run
	ParentTask   *apiV2beta1.PipelineTask
	ParentScope  util.ScopePath
	PipelineSpec *structpb.Struct
}

// ParentNeedsOutputRepublish reports whether a DAG/ROOT parent is missing any
// declared output and therefore may need preserved-child outputs re-applied
// after a retry reset or a partial prior republish. Callers should pass a
// hydrated task (GetTask / ListTasks).
func ParentNeedsOutputRepublish(
	parent *apiV2beta1.PipelineTask,
	outputDefs *pipelinespec.ComponentOutputsSpec,
) bool {
	if parent == nil || outputDefs == nil {
		return false
	}
	declaredParams := outputDefs.GetParameters()
	declaredArtifacts := outputDefs.GetArtifacts()
	if len(declaredParams) == 0 && len(declaredArtifacts) == 0 {
		return false
	}

	presentParams := make(map[string]struct{})
	presentArtifacts := make(map[string]struct{})
	if outputs := parent.GetOutputs(); outputs != nil {
		for _, parameter := range outputs.GetParameters() {
			presentParams[parameter.GetParameterKey()] = struct{}{}
		}
		for _, artifactIO := range outputs.GetArtifacts() {
			presentArtifacts[artifactIO.GetArtifactKey()] = struct{}{}
		}
	}
	for key := range declaredParams {
		if _, ok := presentParams[key]; !ok {
			return true
		}
	}
	for key := range declaredArtifacts {
		if _, ok := presentArtifacts[key]; !ok {
			return true
		}
	}
	return false
}

func isPreservedTaskStateForRetry(state apiV2beta1.PipelineTask_TaskState) bool {
	switch state {
	case apiV2beta1.PipelineTask_SUCCEEDED,
		apiV2beta1.PipelineTask_CACHED,
		apiV2beta1.PipelineTask_SKIPPED:
		return true
	default:
		return false
	}
}

func childHasPropagatableOutputs(child *apiV2beta1.PipelineTask) bool {
	if child == nil {
		return false
	}
	outputs := child.GetOutputs()
	if outputs == nil {
		return false
	}
	return len(outputs.GetParameters()) > 0 || len(outputs.GetArtifacts()) > 0
}

func listAllChildTasks(
	ctx context.Context,
	apiClient kfpapi.API,
	runID, parentTaskID string,
) ([]*apiV2beta1.PipelineTask, error) {
	var children []*apiV2beta1.PipelineTask
	pageToken := ""
	for {
		resp, err := apiClient.ListTasks(ctx, &apiV2beta1.ListTasksRequest{
			RunId: runID,
			ParentFilter: &apiV2beta1.ListTasksRequest_ParentId{
				ParentId: parentTaskID,
			},
			PageSize:  republishChildTasksPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, err
		}
		children = append(children, resp.GetTasks()...)
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return children, nil
}

// RepublishPreservedChildOutputsToDAG re-drives first-level (and upward)
// output propagation from preserved children into parent. It is a no-op when
// the parent already has all declared outputs, has no output definitions, or
// has no preserved children with outputs.
//
// This repairs the retry case where resetRetriedTaskState clears a failed
// parent's outputs while successful siblings are preserved and will not rerun
// (and therefore will not propagate again).
func RepublishPreservedChildOutputsToDAG(
	ctx context.Context,
	opts DAGOutputRepublishOptions,
	clientManager client_manager.ClientManagerInterface,
) error {
	if clientManager == nil {
		return fmt.Errorf("client manager is nil")
	}
	if opts.Run == nil || opts.Run.GetRunId() == "" {
		return fmt.Errorf("run is required")
	}
	if opts.ParentTask == nil || opts.ParentTask.GetTaskId() == "" {
		return nil
	}
	if opts.PipelineSpec == nil {
		return nil
	}

	apiClient := clientManager.KFPAPIClient()
	parentTask, err := apiClient.GetTask(ctx, &apiV2beta1.GetTaskRequest{
		TaskId: opts.ParentTask.GetTaskId(),
		RunId:  opts.Run.GetRunId(),
	})
	if err != nil {
		return fmt.Errorf("failed to refresh parent task %s before output republish: %w", opts.ParentTask.GetTaskId(), err)
	}

	parentScope := opts.ParentScope
	if parentScope.GetSize() == 0 {
		parentScope, err = util.ScopePathFromDotNotation(opts.PipelineSpec, parentTask.GetScopePath())
		if err != nil {
			return fmt.Errorf("failed to resolve parent scope path for output republish: %w", err)
		}
	}
	if parentScope.GetLast() == nil {
		return nil
	}
	parentComponentSpec := parentScope.GetLast().GetComponentSpec()
	if parentComponentSpec == nil {
		return nil
	}
	parentOutputDefs := parentComponentSpec.GetOutputDefinitions()
	if !ParentNeedsOutputRepublish(parentTask, parentOutputDefs) {
		return nil
	}

	children, err := listAllChildTasks(ctx, apiClient, opts.Run.GetRunId(), parentTask.GetTaskId())
	if err != nil {
		return fmt.Errorf("failed to list child tasks for output republish: %w", err)
	}

	batchUpdater := NewBatchUpdater()
	for _, child := range children {
		if child == nil || !isPreservedTaskStateForRetry(child.GetState()) || !childHasPropagatableOutputs(child) {
			continue
		}
		childScope, err := util.ScopePathFromStringPathWithNewTask(
			opts.PipelineSpec,
			parentTask.GetScopePath(),
			child.GetName(),
		)
		if err != nil {
			return fmt.Errorf("failed to build scope path for preserved child %s: %w", child.GetName(), err)
		}
		if err := propagateOutputsUpDAG(ctx, OutputPropagationOptions{
			Run:          opts.Run,
			Task:         child,
			ParentTask:   parentTask,
			ScopePath:    childScope,
			PipelineSpec: opts.PipelineSpec,
		}, apiClient, batchUpdater); err != nil {
			return fmt.Errorf("failed to republish outputs from preserved child %s: %w", child.GetName(), err)
		}
	}

	// A prior partial flush may have persisted artifact links already for the
	// immediate parent and/or ancestors. Drop those queued links so UniqueLink
	// does not fail while parameter repair still proceeds.
	if err := batchUpdater.OmitArtifactTasksAlreadyPresentOnTasks(ctx, apiClient, opts.Run.GetRunId()); err != nil {
		return err
	}

	if err := batchUpdater.Flush(ctx, apiClient); err != nil {
		return fmt.Errorf("failed to flush republished DAG outputs: %w", err)
	}
	return nil
}
