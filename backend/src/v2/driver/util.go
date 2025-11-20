// Copyright 2021-2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
)

// validateRootDAG contains validation for root DAG driver options.
func validateRootDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("parent task is required")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

// validateDAG validates non-root DAG options.
func validateDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid DAG driver args: %w", err)
		}
	}()
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	return validateNonRoot(opts)
}

func validateNonRoot(opts common.Options) error {
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.Task.GetTaskInfo().GetName() == "" {
		return fmt.Errorf("task spec is required")
	}
	if opts.RuntimeConfig != nil {
		return fmt.Errorf("runtime config is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("parent task is required")
	}
	if opts.ParentTask.GetScopePath() == "" {
		return fmt.Errorf("parent task scope path is required for DAG")
	}
	return nil
}

// handleInputTaskParametersCreation creates a new PipelineTaskDetail_InputOutputs_IOParameter
// for each parameter in the executor input.
func handleInputTaskParametersCreation(
	parameterMetadata []resolver.ParameterMetadata,
	task *apiV2beta1.PipelineTaskDetail,
) (*apiV2beta1.PipelineTaskDetail, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	if task.Inputs == nil {
		task.Inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{},
		}
	} else if task.Inputs.Parameters == nil {
		task.Inputs.Parameters = []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}

	for _, pm := range parameterMetadata {
		parameterNew := pm.ParameterIO
		task.Inputs.Parameters = append(task.Inputs.Parameters, parameterNew)
	}
	return task, nil
}

// handleInputTaskArtifactsCreation creates a new ArtifactTask for each input artifact.
// The artifactsTasks are created as input artifacts. This allows KFP backend to
// list input artifacts for this task. Parameters do not require this additional overhead
// because parameters are stored in the task itself.
func handleInputTaskArtifactsCreation(
	ctx context.Context,
	opts common.Options,
	artifactMetadata []resolver.ArtifactMetadata,
	task *apiV2beta1.PipelineTaskDetail,
	kfpAPI kfpapi.API,
) error {
	var artifactTasks []*apiV2beta1.ArtifactTask
	for _, am := range artifactMetadata {
		for _, artifact := range am.ArtifactIO.Artifacts {
			if artifact.ArtifactId == "" {
				return fmt.Errorf("artifact id is required")
			}
			at := &apiV2beta1.ArtifactTask{
				ArtifactId: artifact.ArtifactId,
				RunId:      opts.Run.GetRunId(),
				TaskId:     task.TaskId,
				Type:       am.ArtifactIO.Type,
				Producer:   am.ArtifactIO.Producer,
				Key:        am.ArtifactIO.ArtifactKey,
			}
			artifactTasks = append(artifactTasks, at)
		}
	}
	if len(artifactTasks) > 0 {
		request := apiV2beta1.CreateArtifactTasksBulkRequest{ArtifactTasks: artifactTasks}
		_, err := kfpAPI.CreateArtifactTasks(ctx, &request)
		if err != nil {
			return err
		}
	}
	return nil
}
