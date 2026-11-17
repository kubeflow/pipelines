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

	"github.com/golang/glog"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	nodeTypeRetry = "Retry"
	nodeTypePod   = "Pod"
)

// executorFailure is terminal failure evidence for the current attempt of a task.
type executorFailure struct {
	// podName is the executor pod the evidence is about. The write is fenced on it. Empty when
	// viaNode is true, since that path has no registered pod to fence on.
	podName    string
	message    string
	finishedAt int64
	// viaNode is true when this evidence came from the task's own Argo node (see
	// executorFailureForTaskViaNode) rather than a registered executor pod. The write this
	// produces is fenced on the stored task still having no executor pod recorded at all.
	viaNode bool
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
// current attempt of the task's executor pod. It stays silent when:
//   - the task is not RUNNING, or is a DAG task (a parent says nothing about its children),
//   - a registered executor pod exists but has not ended in Failed or Error,
//   - a registered executor pod sits in a Retry group where it is not the last attempt, or where
//     the group itself has not failed (a retry that is still in flight or recovered must not fail
//     the task).
//
// When no executor pod is registered on the task at all, it falls back to
// executorFailureForTaskViaNode instead of staying silent outright.
func executorFailureForTask(task *model.Task, nodes map[string]util.NodeStatus, resolved map[string]string, tasksByUUID map[string]*model.Task) (executorFailure, bool) {
	if task == nil || task.State != model.TaskStatus(apiv2.PipelineTask_RUNNING) ||
		task.Type == model.TaskType(apiv2.PipelineTask_DAG) {
		return executorFailure{}, false
	}
	podNames := executorPodNamesFromTask(task)
	if len(podNames) == 0 {
		// The launcher never managed to register an executor pod on this task at all, which an
		// outage covering registration can now cause (persistExecutorPod is required, not best
		// effort). There is nothing in task.Pods to correlate with the workflow, so fall back to
		// matching the task's own Argo node by name and scope instead.
		return executorFailureForTaskViaNode(task, nodes, resolved, tasksByUUID)
	}
	// Index pods by their recorded attempt order so a CreateTime tie (same-second
	// retries) is broken by that order rather than by the unrelated node map key.
	podOrder := make(map[string]int, len(podNames))
	for i, name := range podNames {
		podOrder[name] = i
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
			(node.CreateTime == current.CreateTime && podOrder[node.ID] > podOrder[current.ID]) {
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
	return executorFailure{podName: current.ID, message: msg, finishedAt: finishedAt}, true
}

// executorFailureForTaskViaNode derives failure evidence for a task whose launcher never managed
// to register an executor pod on it at all: for example, the API server had an outage spanning
// both the required registration write and every fallback/finalization write, so the launcher
// exited having never reported anything. With no pod in task.Pods to correlate with the workflow,
// this instead matches the task's own Argo Retry node by name and enclosing scope via
// findScopedTaskNode, the same match lifecycleMessageForTaskNode uses when no pod has been
// registered yet. The node's own State already reflects the aggregated outcome of every attempt
// beneath it (Argo only settles a Retry node's state once its own retries are exhausted), so
// unlike the pod-based path there is no separate "last attempt, group failed too" walk to do here.
func executorFailureForTaskViaNode(task *model.Task, nodes map[string]util.NodeStatus, resolved map[string]string, tasksByUUID map[string]*model.Task) (executorFailure, bool) {
	nodeID, ok := findScopedTaskNode(task, nodes, tasksByUUID)
	if !ok {
		return executorFailure{}, false
	}
	node := nodes[nodeID]
	if !isTerminalFailureNodeState(node.State) {
		return executorFailure{}, false
	}
	msg := resolved[nodeID]
	if msg == "" {
		msg = util.NormalizeLifecycleMessage(node.Message, node.State)
	}
	if msg == "" {
		msg = fmt.Sprintf("task %s ended in state %s with no executor pod ever recorded", task.Name, node.State)
	}
	finishedAt := nodeFinishTime(nodeID, nodes, map[string]bool{})
	if finishedAt <= 0 {
		finishedAt = time.Now().Unix()
	}
	return executorFailure{message: msg, finishedAt: finishedAt, viaNode: true}, true
}

// nodeFinishTime returns node's own FinishTime if it has one, otherwise the latest finish time
// recorded by its descendants. A Retry or DAG group node is not guaranteed to have its own
// FinishTime populated once it goes terminal, so this walks down the chain (task Retry -> task DAG
// -> executor Retry -> executor Pod) to whichever node actually recorded one.
//
// Descent only follows an edge that nodeParent agrees is containment, by checking the candidate's
// own computed parent rather than trusting node.Children directly: Children also records plain
// dependency ordering between siblings (a predecessor's last attempt pointing at a dependent's
// Retry node, as nodeParent's own doc comment explains and siblingDependencyNodes exercises), so a
// blind Children walk here could wander onto a downstream sibling's node and report its finish
// time as this one's. visited guards against a malformed cycle; real Argo node graphs are trees,
// but nothing here should hang on data that isn't.
func nodeFinishTime(nodeID string, nodes map[string]util.NodeStatus, visited map[string]bool) int64 {
	if visited[nodeID] {
		return 0
	}
	visited[nodeID] = true
	node, ok := nodes[nodeID]
	if !ok {
		return 0
	}
	if node.FinishTime > 0 {
		return node.FinishTime
	}
	var latest int64
	for _, childID := range node.Children {
		if parentID, ok := nodeParent(childID, nodes); !ok || parentID != nodeID {
			// Not actually contained by node: a dependency edge, not a descent.
			continue
		}
		if t := nodeFinishTime(childID, nodes, visited); t > latest {
			latest = t
		}
	}
	return latest
}

// failureStatusMetadata returns the task's stored status metadata with the failure message set.
// Everything else in it, such as custom properties a plugin recorded before the pod died, is kept.
// Metadata that cannot be decoded is replaced by a message-only value rather than failing the report.
func failureStatusMetadata(existing model.JSONData, message string) (model.JSONData, error) {
	statusMetadata := &apiv2.PipelineTask_StatusMetadata{}
	if len(existing) > 0 {
		decoded, err := model.JSONDataToProtoMessage(existing, func() *apiv2.PipelineTask_StatusMetadata {
			return &apiv2.PipelineTask_StatusMetadata{}
		})
		if err != nil {
			glog.Warningf("Dropping unreadable status metadata while recording a task failure: %v", err)
		} else {
			statusMetadata = decoded
		}
	}
	statusMetadata.Message = message
	return model.ProtoMessageToJSONData(statusMetadata)
}

// reconcileTaskFailure moves a RUNNING task to FAILED when executorFailureForTask finds evidence.
// The write only lands while the stored task is still RUNNING with the same executor-pod identity
// the evidence was read from: that exact pod, or, when the evidence came from the task's own node
// because no pod was ever registered, still no pod at all. So neither a state the launcher
// reported in the meantime, nor a registration that eventually landed, nor a retry that reset the
// task after this report was read, is overwritten. It returns true when the task was updated.
func (r *ResourceManager) reconcileTaskFailure(task *model.Task, nodes map[string]util.NodeStatus, resolved map[string]string, tasksByUUID map[string]*model.Task) (bool, error) {
	failure, ok := executorFailureForTask(task, nodes, resolved, tasksByUUID)
	if !ok {
		return false, nil
	}
	metadata, err := failureStatusMetadata(task.StatusMetadata, failure.message)
	if err != nil {
		return false, err
	}
	guard := storage.TaskAttemptGuard{State: model.TaskStatus(apiv2.PipelineTask_RUNNING)}
	if failure.viaNode {
		guard.RequireNoExecutorPod = true
	} else {
		guard.ExecutorPod = failure.podName
	}
	_, applied, err := r.taskStore.UpdateTaskIfAttempt(
		guard,
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
