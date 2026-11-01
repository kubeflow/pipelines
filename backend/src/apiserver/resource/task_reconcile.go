// Copyright 2026 The Kubeflow Authors
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

package resource

import (
	"fmt"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	nodeTypeRetry = "Retry"
	nodeTypePod   = "Pod"
)

// executorFailure is terminal failure evidence for the current attempt of a task.
type executorFailure struct {
	message    string
	finishedAt int64
}

func isTerminalFailureNodeState(state string) bool {
	return state == "Failed" || state == "Error"
}

// executorPodNamesFromTask returns the names of the executor pods recorded on the task. Driver and
// agent pods are left out on purpose: their outcome says nothing about the user's container.
func executorPodNamesFromTask(task *model.Task) []string {
	var names []string
	for _, pod := range task.Pods {
		podMap, ok := pod.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := podMap["name"].(string)
		if name == "" {
			continue
		}
		switch typ := podMap["type"].(type) {
		case string:
			if typ == apiv2.PipelineTask_EXECUTOR.String() {
				names = append(names, name)
			}
		case float64:
			if int32(typ) == int32(apiv2.PipelineTask_EXECUTOR) {
				names = append(names, name)
			}
		}
	}
	return names
}

// executorFailureForTask reports whether the workflow holds terminal failure evidence for the
// current attempt of the task's executor pod. It only ever looks at the executor pod, and it
// stays silent when:
//   - the task is not RUNNING, or is a DAG task (a parent says nothing about its children),
//   - no executor pod of the task is in the workflow yet,
//   - the newest executor pod has not ended in Failed or Error,
//   - the pod sits in a Retry group where it is not the last attempt, or where the group itself
//     has not failed (a retry that is still in flight or recovered must not fail the task).
func executorFailureForTask(task *model.Task, nodes map[string]util.NodeStatus, resolved map[string]string) (executorFailure, bool) {
	if task == nil || task.State != model.TaskStatus(apiv2.PipelineTask_RUNNING) ||
		task.Type == model.TaskType(apiv2.PipelineTask_DAG) {
		return executorFailure{}, false
	}
	podNames := executorPodNamesFromTask(task)
	if len(podNames) == 0 {
		return executorFailure{}, false
	}

	// Node map keys differ from pod names, and Children refer to map keys.
	parents := make(map[string]string, len(nodes))
	for key, node := range nodes {
		for _, child := range node.Children {
			parents[child] = key
		}
	}

	var (
		currentKey string
		current    util.NodeStatus
		found      bool
	)
	for key, node := range nodes {
		if node.Type != nodeTypePod || !containsString(podNames, node.ID) {
			continue
		}
		if !found || node.CreateTime > current.CreateTime ||
			(node.CreateTime == current.CreateTime && key > currentKey) {
			currentKey, current, found = key, node, true
		}
	}
	if !found || !isTerminalFailureNodeState(current.State) {
		return executorFailure{}, false
	}

	// Walk up to the nearest Retry group: the pod has to be its last child, and the group has to
	// have given up too.
	child := currentKey
	visited := map[string]bool{currentKey: true}
	for parentKey, ok := parents[child]; ok; parentKey, ok = parents[child] {
		if visited[parentKey] {
			return executorFailure{}, false
		}
		visited[parentKey] = true
		parent := nodes[parentKey]
		if parent.Type == nodeTypeRetry {
			if len(parent.Children) == 0 || parent.Children[len(parent.Children)-1] != child {
				return executorFailure{}, false
			}
			if !isTerminalFailureNodeState(parent.State) {
				return executorFailure{}, false
			}
			break
		}
		child = parentKey
	}

	msg := resolved[currentKey]
	if msg == "" {
		msg = util.NormalizeLifecycleMessage(current.Message, current.State)
	}
	if msg == "" {
		msg = fmt.Sprintf("executor pod %s ended in state %s", current.ID, current.State)
	}
	finishedAt := current.FinishTime
	if finishedAt <= 0 {
		finishedAt = time.Now().Unix()
	}
	return executorFailure{message: msg, finishedAt: finishedAt}, true
}

// reconcileTaskFailure moves a RUNNING task to FAILED when executorFailureForTask finds evidence,
// without overwriting a state the launcher reported in the meantime. It returns true when the task
// was updated.
func (r *ResourceManager) reconcileTaskFailure(task *model.Task, nodes map[string]util.NodeStatus, resolved map[string]string) (bool, error) {
	failure, ok := executorFailureForTask(task, nodes, resolved)
	if !ok {
		return false, nil
	}
	metadata, err := model.ProtoMessageToJSONData(&apiv2.PipelineTask_StatusMetadata{Message: failure.message})
	if err != nil {
		return false, err
	}
	_, applied, err := r.taskStore.UpdateTaskIfState(
		model.TaskStatus(apiv2.PipelineTask_RUNNING),
		&model.Task{
			UUID:           task.UUID,
			State:          model.TaskStatus(apiv2.PipelineTask_FAILED),
			FinishedInSec:  failure.finishedAt,
			StatusMetadata: metadata,
		},
	)
	if err != nil || !applied {
		return false, err
	}
	task.State = model.TaskStatus(apiv2.PipelineTask_FAILED)
	task.FinishedInSec = failure.finishedAt
	task.StatusMetadata = metadata
	return true, nil
}
