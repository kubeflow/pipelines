// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolver

import (
	"fmt"
	"slices"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/proto"
)

// taskStatusAtExit projects the controller's completed dependency into the
// handler input. Native finalization can happen later, after all exit hooks.
// This is read-only so an older hook cannot mutate a newer run generation.
func taskStatusAtExit(opts common.Options, name string, task *api.PipelineTask) (*api.PipelineTask, error) {
	if opts.ExitTaskName != name || opts.ExitTaskStatus == "" {
		return task, nil
	}
	if opts.Task.GetTriggerPolicy().GetStrategy() != pipelinespec.PipelineTaskSpec_TriggerPolicy_ALL_UPSTREAM_TASKS_COMPLETED || !slices.Contains(opts.Task.GetDependentTasks(), name) {
		return nil, fmt.Errorf("completed task status requires a matching exit-hook dependency; recompile the pipeline")
	}
	state := task.GetState()
	switch opts.ExitTaskStatus {
	case "Failed", "Error":
		state = api.PipelineTask_FAILED
	case "Succeeded":
		// Controller completion does not establish native execution success.
		return task, nil
	case "Skipped", "Omitted":
		if state != api.PipelineTask_FAILED {
			state = api.PipelineTask_SKIPPED
		}
	default:
		return task, nil
	}
	message := ""
	if state == api.PipelineTask_FAILED {
		message = exitTaskFailureMessage(task, opts.Run.GetTasks())
		if message == "" {
			message = fmt.Sprintf("Task %q ended with Argo status %s.", name, opts.ExitTaskStatus)
		}
	}
	if task == nil {
		// The driver may have failed before creating its native task record.
		task = &api.PipelineTask{Name: name}
	} else {
		task = proto.Clone(task).(*api.PipelineTask)
	}
	task.State = state
	if task.StatusMetadata == nil {
		task.StatusMetadata = &api.PipelineTask_StatusMetadata{}
	}
	task.StatusMetadata.Message = message
	return task, nil
}

func exitTaskFailureMessage(task *api.PipelineTask, tasks []*api.PipelineTask) string {
	if message := task.GetStatusMetadata().GetMessage(); message != "" {
		return message
	}
	if task.GetTaskId() == "" {
		return ""
	}
	children := make(map[string][]*api.PipelineTask)
	for _, child := range tasks {
		children[child.GetParentTaskId()] = append(children[child.GetParentTaskId()], child)
	}
	pending := []*api.PipelineTask{task}
	seen := make(map[string]bool)
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if seen[current.GetTaskId()] {
			continue
		}
		seen[current.GetTaskId()] = true
		if current.GetState() == api.PipelineTask_FAILED || current.GetState() == api.PipelineTask_RUNNING {
			if message := current.GetStatusMetadata().GetMessage(); message != "" {
				return message
			}
		}
		pending = append(pending, children[current.GetTaskId()]...)
	}
	return ""
}
