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

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// synchronizeRunFromLiveWorkflow checks whether an active Argo workflow exists in Kubernetes
// and reconciles stranded task states before serving client requests (such as exit-handler input resolution).
func (r *ResourceManager) synchronizeRunFromLiveWorkflow(ctx context.Context, run *model.Run) (bool, error) {
	if run == nil || run.K8SName == "" || run.ImportedFrom != "" {
		return false, nil
	}
	if r.execClient == nil || r.taskStore == nil {
		return false, nil
	}

	// Fetch all tasks for the run to check if any non-terminal tasks or inconsistent DAG states require synchronization
	tasks, err := fetchAllTasksForRun(r, run.UUID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch tasks for run %s: %w", run.UUID, err)
	}
	if len(tasks) == 0 {
		return false, nil
	}

	namespace := run.Namespace
	if namespace == "" || namespace == "-" {
		namespace = common.GetPodNamespace()
	}
	workflowClient := r.getWorkflowClient(namespace)
	if workflowClient == nil {
		return false, fmt.Errorf("workflow client unavailable for live synchronization of run %s in namespace %s", run.UUID, namespace)
	}
	liveWorkflow, err := workflowClient.Get(ctx, run.K8SName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Check if an independently authoritative, persisted completion record exists
			if verifiedPersistedWorkflow, ok := parseVerifiedPersistedWorkflow(run); ok {
				persistedGen := reportedRetryGeneration(verifiedPersistedWorkflow.ExecutionObjectMeta())
				if run.RetryGeneration > 0 && persistedGen < run.RetryGeneration {
					return false, nil
				}
				updated, taskRejected, recErr := r.reconcileStrandedTasks(ctx, run, verifiedPersistedWorkflow, true)
				if recErr != nil {
					return false, fmt.Errorf("persisted workflow reconciliation error for run %s: %w", run.UUID, recErr)
				}
				if !taskRejected && persistedGen >= run.RetryGeneration && verifiedPersistedWorkflow.ExecutionStatus() != nil &&
					(verifiedPersistedWorkflow.ExecutionStatus().Condition() == "Failed" || verifiedPersistedWorkflow.ExecutionStatus().Condition() == "Error") {
					if run.State.ToV2() != model.RuntimeStateFailed {
						run.State = model.RuntimeStateFailed
						run.Conditions = string(model.RuntimeStateFailed)
						if _, err := r.runStore.UpdateRunIfRuntimeManifestsUnchanged(run, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest); err != nil {
							return false, fmt.Errorf("persisted workflow failed to update run %s: %w", run.UUID, err)
						}
					}
				}
				return updated, nil
			}

			// If authoritative final status is required (e.g. exit-handler input resolution)
			// but neither the live workflow nor an authoritative persisted completion record exists,
			// fail closed so that unverified database status cannot be returned as SUCCEEDED.
			if common.IsAuthoritativeFinalStatusRequired(ctx) {
				return false, util.NewNotFoundError(err,
					"cannot authoritatively verify final status for run %s: live workflow %s not found in namespace %s and no verified persisted completion record exists",
					run.UUID, run.K8SName, namespace)
			}

			// If the run in the database is already terminal, then workflow garbage collection after completion
			// is normal and safe. But for non-terminal runs, a missing workflow means state cannot be
			// authoritatively verified, so fail closed.
			switch run.State.ToV2() {
			case model.RuntimeStateSucceeded,
				model.RuntimeStateFailed,
				model.RuntimeStateSkipped,
				model.RuntimeStateCancelling,
				model.RuntimeStateCanceled:
				if hasDAGInconsistencies(tasks) {
					return r.aggregateDAGTaskStatuses(run.UUID, run.RetryGeneration, true)
				}
				return false, nil
			}
			return false, fmt.Errorf("live workflow %s not found in namespace %s for non-terminal run %s: %w", run.K8SName, namespace, run.UUID, err)
		}
		return false, fmt.Errorf("failed to get live workflow %s in namespace %s: %w", run.K8SName, namespace, err)
	}
	if liveWorkflow == nil {
		return false, fmt.Errorf("live workflow %s returned nil in namespace %s", run.K8SName, namespace)
	}

	// Validate workflow identity and attempt before any writes
	storedIdentity, idErr := r.storedWorkflowIdentityForRun(run)
	if idErr == nil {
		if !storedWorkflowMatchesIdentity(storedIdentity, liveWorkflow) &&
			!retryWorkflowMatchesActiveClaim(run, storedIdentity, liveWorkflow) {
			return false, util.NewInvalidInputError(
				"live workflow identity does not match stored workflow for run %s", run.UUID)
		}
	}

	reportedGen := reportedRetryGeneration(liveWorkflow.ExecutionObjectMeta())
	// Retry fencing: do not let observations from older retry generations reconcile retried runs/tasks
	if run.RetryGeneration > 0 && reportedGen < run.RetryGeneration {
		return false, nil
	}

	isFinalState := liveWorkflow.ExecutionStatus() != nil && liveWorkflow.ExecutionStatus().IsInFinalState()
	updated, taskRejected, err := r.reconcileStrandedTasks(ctx, run, liveWorkflow, isFinalState)
	if err != nil {
		return false, fmt.Errorf("live workflow reconciliation error for run %s: %w", run.UUID, err)
	}
	if !taskRejected && reportedGen >= run.RetryGeneration && isFinalState && liveWorkflow.ExecutionStatus() != nil &&
		(liveWorkflow.ExecutionStatus().Condition() == "Failed" || liveWorkflow.ExecutionStatus().Condition() == "Error") {
		if run.State.ToV2() != model.RuntimeStateFailed {
			run.State = model.RuntimeStateFailed
			run.Conditions = string(model.RuntimeStateFailed)
			if _, err := r.runStore.UpdateRunIfRuntimeManifestsUnchanged(run, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest); err != nil {
				return false, fmt.Errorf("live workflow failed to update run %s: %w", run.UUID, err)
			}
		}
	}
	return updated, nil
}

func (r *ResourceManager) reconcileStrandedTasks(ctx context.Context, run *model.Run, execSpec util.ExecutionSpec, isFinalState bool) (bool, bool, error) {
	if run == nil || execSpec == nil || execSpec.ExecutionStatus() == nil {
		return false, false, nil
	}

	reportedGeneration := reportedRetryGeneration(execSpec.ExecutionObjectMeta())

	// Retry fencing: do not let reports from older retry generations reconcile retried tasks
	if run.RetryGeneration > 0 && reportedGeneration < run.RetryGeneration {
		return false, true, nil
	}

	nodeStatuses := execSpec.ExecutionStatus().NodeStatuses()
	if len(nodeStatuses) == 0 {
		updated, err := r.aggregateDAGTaskStatuses(run.UUID, reportedGeneration, isFinalState)
		return updated, false, err
	}

	tasks, err := fetchAllTasksForRun(r, run.UUID)
	if err != nil {
		return false, false, fmt.Errorf("failed to list tasks for run %s during stranded task reconciliation: %w", run.UUID, err)
	}

	nodesByPodName := make(map[string]util.NodeStatus, len(nodeStatuses))
	nodesByNodeID := make(map[string]util.NodeStatus, len(nodeStatuses))
	nodesByDisplayName := make(map[string]util.NodeStatus, len(nodeStatuses))
	for nodeID, node := range nodeStatuses {
		nodesByNodeID[nodeID] = node
		if node.ID != "" {
			nodesByPodName[node.ID] = node
		}
		if node.DisplayName != "" {
			nodesByDisplayName[node.DisplayName] = node
		}
	}

	nowSec := r.time.Now().Unix()
	anyUpdated := false
	anyRejected := false

	reconcileTask := func(task *model.Task, node util.NodeStatus, extraPodName string) error {
		applyNodeStatusToTask(task, node, nowSec)
		if extraPodName != "" {
			hasPod := false
			for _, pod := range task.Pods {
				switch p := pod.(type) {
				case map[string]interface{}:
					name, _ := p["name"].(string)
					if name == "" {
						name, _ = p["Name"].(string)
					}
					if name == extraPodName {
						hasPod = true
						break
					}
				case *apiv2beta1.PipelineTask_TaskPod:
					if p != nil && p.GetName() == extraPodName {
						hasPod = true
						break
					}
				}
			}
			if !hasPod {
				task.Pods = append(task.Pods, map[string]interface{}{
					"name": extraPodName,
					"type": apiv2beta1.PipelineTask_EXECUTOR.String(),
				})
			}
		}
		if _, err := r.taskStore.UpdateTaskWithRetryFence(task, reportedGeneration); err != nil {
			if util.IsUserErrorCodeMatch(err, codes.Aborted) {
				glog.Infof("Skipping task update for %s due to stale retry generation: %v", task.UUID, err)
				anyRejected = true
				return nil
			}
			return fmt.Errorf("failed to atomically reconcile stranded task %s: %w", task.UUID, err)
		}
		anyUpdated = true
		return nil
	}

	for _, task := range tasks {
		if isStructuralTaskType(task.Type) {
			continue
		}

		canReconcile := task.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) ||
			task.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED) ||
			task.State == model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED) ||
			task.State == model.TaskStatus(apiv2beta1.PipelineTask_SKIPPED) ||
			task.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED)

		if !canReconcile {
			continue
		}

		var executorPodNames []string
		var driverPodNames []string
		var allPodNames []string

		for _, pod := range task.Pods {
			podMap, ok := pod.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := podMap["name"].(string)
			if name == "" {
				continue
			}
			allPodNames = append(allPodNames, name)
			if podTypeMatches(podMap, apiv2beta1.PipelineTask_EXECUTOR) {
				executorPodNames = append(executorPodNames, name)
			} else if podTypeMatches(podMap, apiv2beta1.PipelineTask_DRIVER) {
				driverPodNames = append(driverPodNames, name)
			}
		}

		if len(executorPodNames) == 0 && len(driverPodNames) == 0 {
			for _, name := range allPodNames {
				if strings.Contains(name, "-driver") {
					driverPodNames = append(driverPodNames, name)
				} else {
					executorPodNames = append(executorPodNames, name)
				}
			}
		}

		// 1. If driver pods exist, check driver and discovered executor nodes (including retry nodes)
		if len(driverPodNames) > 0 {
			latest := driverPodNames[len(driverPodNames)-1]
			driverNode, found := findNodeForPodName(latest, nodesByPodName, nodesByNodeID, nodesByDisplayName)
			if found {
				if driverNode.State == "Failed" || driverNode.State == "Error" {
					if shouldReconcileTaskFromNode(task.State, driverNode.State, isFinalState) {
						glog.Infof("Reconciling task %s from failed driver node %s", task.UUID, latest)
						if err := reconcileTask(task, driverNode, ""); err != nil {
							return false, anyRejected, err
						}
					}
					continue
				} else if driverNode.State == "Succeeded" {
					executorNodes := findExecutorNodesFromDriver(driverNode, nodesByNodeID)
					if currentAttempt, found := selectCurrentExecutorAttempt(executorNodes); found {
						if shouldReconcileTaskFromNode(task.State, currentAttempt.State, isFinalState) {
							podName := currentAttempt.ID
							if podName == "" {
								podName = currentAttempt.DisplayName
							}
							glog.Infof("Reconciling task %s: driver succeeded, current executor attempt %s (%s)", task.UUID, podName, currentAttempt.State)
							if err := reconcileTask(task, currentAttempt, podName); err != nil {
								return false, anyRejected, err
							}
						}
						// Terminate inspection on the current attempt; do not fall back to older attempts.
						continue
					}
					// If driver succeeded, no executor ran, and the entire workflow has finished: fail closed.
					if task.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) && len(executorNodes) == 0 && isFinalState {
						failNode := util.NodeStatus{
							State:      "Failed",
							FinishTime: nowSec,
							Message:    "Executor failed to start or crashed before reporting",
						}
						glog.Infof("Reconciling stranded task %s: driver succeeded but no executor ran in terminal workflow; failing closed", task.UUID)
						if err := reconcileTask(task, failNode, ""); err != nil {
							return false, anyRejected, err
						}
					}
					continue
				}
			}
		}

		// 2. If executor pods exist directly on the task:
		if len(executorPodNames) > 0 {
			latest := executorPodNames[len(executorPodNames)-1]
			if node, found := findNodeForPodName(latest, nodesByPodName, nodesByNodeID, nodesByDisplayName); found && shouldReconcileTaskFromNode(task.State, node.State, isFinalState) {
				glog.Infof("Reconciling task %s from executor node %s (%s, previous state %s)", task.UUID, latest, node.State, apiv2beta1.PipelineTask_TaskState(task.State).String())
				if err := reconcileTask(task, node, ""); err != nil {
					return false, anyRejected, err
				}
			}
			continue
		}

		// 3. Fallback: match by task name or DisplayName if unambiguous
		if len(allPodNames) == 0 {
			var matchingNodes []util.NodeStatus
			for _, node := range nodesByNodeID {
				if node.DisplayName == task.Name || (task.DisplayName != "" && node.DisplayName == task.DisplayName) {
					if shouldReconcileTaskFromNode(task.State, node.State, isFinalState) {
						matchingNodes = append(matchingNodes, node)
					}
				}
			}
			if len(matchingNodes) == 1 {
				match := matchingNodes[0]
				podName := match.ID
				if podName == "" {
					podName = match.DisplayName
				}
				glog.Infof("Reconciling task %s by name match to node %s (%s)", task.UUID, podName, match.State)
				if err := reconcileTask(task, match, podName); err != nil {
					return false, anyRejected, err
				}
			}
		}
	}

	// Always run DAG aggregation based on persisted task states. This ensures that even if
	// leaf reconciliation occurred on a prior report, DAG parent states still converge.
	dagUpdated, err := r.aggregateDAGTaskStatuses(run.UUID, reportedGeneration, isFinalState)
	if err != nil {
		return false, anyRejected, err
	}
	return anyUpdated || dagUpdated, anyRejected, nil
}

// aggregateDAGTaskStatuses traverses the DAG tree from bottom to top, updating parent task states.
func (r *ResourceManager) aggregateDAGTaskStatuses(runID string, reportedGeneration int64, isFinalState bool) (bool, error) {
	tasks, err := fetchAllTasksForRun(r, runID)
	if err != nil {
		return false, fmt.Errorf("failed to list tasks for run %s during DAG aggregation: %w", runID, err)
	}

	childrenByParent := make(map[string][]*model.Task)
	dagTasks := make(map[string]*model.Task)
	for _, task := range tasks {
		if task.ParentTaskUUID != nil && *task.ParentTaskUUID != "" {
			childrenByParent[*task.ParentTaskUUID] = append(childrenByParent[*task.ParentTaskUUID], task)
		}
		if isStructuralTaskType(task.Type) {
			dagTasks[task.UUID] = task
		}
	}

	nowSec := r.time.Now().Unix()
	changed := true
	anyTaskUpdated := false
	for changed {
		changed = false
		for parentID, parent := range dagTasks {
			children := childrenByParent[parentID]
			if len(children) == 0 {
				continue
			}

			anyFailed := false
			anyRunning := false
			allSkipped := true
			var failedChildMsg string

			for _, child := range children {
				switch apiv2beta1.PipelineTask_TaskState(child.State) {
				case apiv2beta1.PipelineTask_RUNNING, apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED:
					anyRunning = true
				case apiv2beta1.PipelineTask_FAILED:
					anyFailed = true
					allSkipped = false
					if failedChildMsg == "" {
						switch {
						case child.StatusMetadata != nil && child.StatusMetadata["message"] != nil:
							failedChildMsg = fmt.Sprint(child.StatusMetadata["message"])
						case child.LifecycleMessage != nil:
							failedChildMsg = string(*child.LifecycleMessage)
						case child.Name != "":
							failedChildMsg = fmt.Sprintf("Child task %s failed", child.Name)
						}
					}
				case apiv2beta1.PipelineTask_SKIPPED:
					// remains true
				default:
					allSkipped = false
				}
			}

			// Defer parent aggregation while any child is still RUNNING
			if anyRunning {
				continue
			}

			if anyFailed {
				needsUpdate := parent.State != model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
				if !needsUpdate && failedChildMsg != "" && (parent.StatusMetadata == nil || parent.StatusMetadata["message"] == nil) {
					needsUpdate = true
				}
				if needsUpdate {
					glog.Infof("Aggregating FAILED state to parent DAG task %s (previous state: %s)", parentID, apiv2beta1.PipelineTask_TaskState(parent.State).String())
					parent.State = model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
					if parent.FinishedInSec == 0 {
						parent.FinishedInSec = nowSec
					}
					if failedChildMsg != "" {
						parent.StatusMetadata = model.JSONData{"message": failedChildMsg}
						lm := model.LargeText(failedChildMsg)
						parent.LifecycleMessage = &lm
					}

					if _, err := r.taskStore.UpdateTaskWithRetryFence(parent, reportedGeneration); err != nil {
						if util.IsUserErrorCodeMatch(err, codes.Aborted) {
							glog.Infof("Skipping DAG aggregation for %s due to stale retry generation: %v", parentID, err)
							return false, nil
						}
						return false, fmt.Errorf("failed to atomically update aggregated DAG task %s: %w", parentID, err)
					}
					changed = true
					anyTaskUpdated = true
				}
				continue
			}

			if isFinalState {
				// All children terminal and none failed in a finished workflow
				newState := apiv2beta1.PipelineTask_SUCCEEDED
				if allSkipped {
					newState = apiv2beta1.PipelineTask_SKIPPED
				}
				if parent.State != model.TaskStatus(newState) {
					glog.Infof("Aggregating %s state to parent DAG task %s", newState.String(), parentID)
					parent.State = model.TaskStatus(newState)
					if parent.FinishedInSec == 0 {
						parent.FinishedInSec = nowSec
					}
					parent.StatusMetadata = nil
					parent.LifecycleMessage = nil

					if _, err := r.taskStore.UpdateTaskWithRetryFence(parent, reportedGeneration); err != nil {
						if util.IsUserErrorCodeMatch(err, codes.Aborted) {
							glog.Infof("Skipping DAG aggregation for %s due to stale retry generation: %v", parentID, err)
							return false, nil
						}
						return false, fmt.Errorf("failed to atomically update aggregated DAG task %s: %w", parentID, err)
					}
					changed = true
					anyTaskUpdated = true
				}
			}
		}
	}
	return anyTaskUpdated, nil
}

func fetchAllTasksForRun(r *ResourceManager, runID string) ([]*model.Task, error) {
	var allTasks []*model.Task
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID},
	}
	opts, err := list.NewOptions(&model.Task{}, 200, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list options for run %s: %w", runID, err)
	}

	for {
		tasks, _, nextPageToken, err := r.taskStore.ListTasks(filterContext, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for run %s: %w", runID, err)
		}
		allTasks = append(allTasks, tasks...)
		if nextPageToken == "" {
			break
		}
		opts, err = list.NewOptionsFromToken(nextPageToken, 200)
		if err != nil {
			return nil, fmt.Errorf("failed to parse next page token for run %s: %w", runID, err)
		}
	}
	return allTasks, nil
}

func isStructuralTaskType(t model.TaskType) bool {
	switch apiv2beta1.PipelineTask_TaskType(t) {
	case apiv2beta1.PipelineTask_ROOT,
		apiv2beta1.PipelineTask_DAG,
		apiv2beta1.PipelineTask_LOOP,
		apiv2beta1.PipelineTask_CONDITION,
		apiv2beta1.PipelineTask_CONDITION_BRANCH:
		return true
	default:
		return false
	}
}

func isTerminalTaskState(state model.TaskStatus) bool {
	switch apiv2beta1.PipelineTask_TaskState(state) {
	case apiv2beta1.PipelineTask_SUCCEEDED,
		apiv2beta1.PipelineTask_FAILED,
		apiv2beta1.PipelineTask_SKIPPED,
		apiv2beta1.PipelineTask_CACHED:
		return true
	default:
		return false
	}
}

func podTypeMatches(podMap map[string]interface{}, wanted apiv2beta1.PipelineTask_TaskPodType) bool {
	v, exists := podMap["type"]
	return exists && (fmt.Sprint(v) == wanted.String() || fmt.Sprint(v) == fmt.Sprint(int32(wanted)))
}

func findNodeForPodName(podName string, byPod, byID, byDisplay map[string]util.NodeStatus) (util.NodeStatus, bool) {
	if n, ok := byPod[podName]; ok {
		return n, true
	}
	if n, ok := byID[podName]; ok {
		return n, true
	}
	if n, ok := byDisplay[podName]; ok {
		return n, true
	}
	return util.NodeStatus{}, false
}

func findExecutorNodesFromDriver(driverNode util.NodeStatus, nodesByNodeID map[string]util.NodeStatus) []util.NodeStatus {
	var executorNodes []util.NodeStatus
	var queue []string
	queue = append(queue, driverNode.Children...)
	visited := make(map[string]bool)

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]
		if visited[currID] {
			continue
		}
		visited[currID] = true

		childNode, exists := nodesByNodeID[currID]
		if !exists {
			continue
		}
		if childNode.ID == "" {
			childNode.ID = currID
		}
		switch childNode.Type {
		case "Pod", "Container":
			executorNodes = append(executorNodes, childNode)
		case "Retry", "DAG", "Steps", "TaskGroup", "StepGroup":
			queue = append(queue, childNode.Children...)
		default:
			if len(childNode.Children) > 0 {
				queue = append(queue, childNode.Children...)
			} else {
				executorNodes = append(executorNodes, childNode)
			}
		}
	}
	return executorNodes
}

func selectCurrentExecutorAttempt(executorNodes []util.NodeStatus) (util.NodeStatus, bool) {
	if len(executorNodes) == 0 {
		return util.NodeStatus{}, false
	}
	// 1. If an attempt is currently running, that is authoritatively the current attempt.
	for _, n := range executorNodes {
		if n.State == "Running" {
			return n, true
		}
	}
	// 2. Otherwise, start from the last discovered node (matching BFS/retry order)
	// and select the attempt with the latest finish/start/create timestamp.
	current := executorNodes[len(executorNodes)-1]
	for _, n := range executorNodes {
		if n.FinishTime > current.FinishTime {
			current = n
		} else if n.FinishTime == current.FinishTime {
			if n.StartTime > current.StartTime {
				current = n
			} else if n.StartTime == current.StartTime && n.CreateTime > current.CreateTime {
				current = n
			}
		}
	}
	return current, true
}

func applyNodeStatusToTask(task *model.Task, node util.NodeStatus, defaultFinishTime int64) {
	var newState apiv2beta1.PipelineTask_TaskState
	switch node.State {
	case "Running":
		newState = apiv2beta1.PipelineTask_RUNNING
		task.FinishedInSec = 0
	case "Succeeded":
		newState = apiv2beta1.PipelineTask_SUCCEEDED
		task.FinishedInSec = node.FinishTime
		if task.FinishedInSec == 0 {
			task.FinishedInSec = defaultFinishTime
		}
	case "Skipped", "Omitted":
		newState = apiv2beta1.PipelineTask_SKIPPED
		task.FinishedInSec = node.FinishTime
		if task.FinishedInSec == 0 {
			task.FinishedInSec = defaultFinishTime
		}
	default:
		newState = apiv2beta1.PipelineTask_FAILED
		task.FinishedInSec = node.FinishTime
		if task.FinishedInSec == 0 {
			task.FinishedInSec = defaultFinishTime
		}
	}

	task.State = model.TaskStatus(newState)
	if node.Message != "" {
		task.StatusMetadata = model.JSONData{"message": node.Message}
		lm := model.LargeText(node.Message)
		task.LifecycleMessage = &lm
	} else if newState == apiv2beta1.PipelineTask_RUNNING || newState == apiv2beta1.PipelineTask_SUCCEEDED {
		task.StatusMetadata = nil
		task.LifecycleMessage = nil
	}
}

func isTerminalNodeState(state string, isFinalState bool) bool {
	return state == "Failed" || state == "Error" || state == "Skipped" || state == "Omitted" || (state == "Succeeded" && isFinalState)
}

func shouldReconcileTaskFromNode(taskState model.TaskStatus, nodeState string, isFinalState bool) bool {
	if taskState == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) ||
		taskState == model.TaskStatus(apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED) {
		return isTerminalNodeState(nodeState, isFinalState)
	}
	// For already terminal SUCCEEDED or SKIPPED tasks: only reconcile if authoritative node Failed or Errored!
	if taskState == model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED) ||
		taskState == model.TaskStatus(apiv2beta1.PipelineTask_SKIPPED) {
		return nodeState == "Failed" || nodeState == "Error"
	}
	// For already terminal FAILED tasks: allow reconciliation if authoritative node Succeeded or Running (recovering automatic retry)!
	if taskState == model.TaskStatus(apiv2beta1.PipelineTask_FAILED) {
		return nodeState == "Succeeded" || nodeState == "Running"
	}
	return false
}

func hasDAGInconsistencies(tasks []*model.Task) bool {
	tasksByUUID := make(map[string]*model.Task, len(tasks))
	childrenByParent := make(map[string][]*model.Task, len(tasks))
	for _, t := range tasks {
		tasksByUUID[t.UUID] = t
		if t.ParentTaskUUID != nil && *t.ParentTaskUUID != "" {
			childrenByParent[*t.ParentTaskUUID] = append(childrenByParent[*t.ParentTaskUUID], t)
		}
	}
	for parentID, children := range childrenByParent {
		parent, ok := tasksByUUID[parentID]
		if !ok || !isStructuralTaskType(parent.Type) {
			continue
		}
		if parent.State == model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED) ||
			parent.State == model.TaskStatus(apiv2beta1.PipelineTask_SKIPPED) {
			for _, child := range children {
				if child.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED) ||
					child.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) ||
					child.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED) {
					return true
				}
			}
		} else if parent.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED) {
			hasFailureOrRunning := false
			for _, child := range children {
				if child.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED) ||
					child.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) ||
					child.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNTIME_STATE_UNSPECIFIED) {
					hasFailureOrRunning = true
					break
				}
			}
			if !hasFailureOrRunning {
				return true
			}
		}
	}
	return false
}

// parseVerifiedPersistedWorkflow extracts an independently authoritative, persisted terminal
// execution spec from the run's stored manifest if one was verified and persisted by the persistence agent.
func parseVerifiedPersistedWorkflow(run *model.Run) (util.ExecutionSpec, bool) {
	if run == nil {
		return nil, false
	}
	manifest := string(storedWorkflowIdentityManifest(run))
	if manifest == "" {
		return nil, false
	}
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(manifest))
	if err != nil || execSpec == nil || execSpec.ExecutionStatus() == nil {
		return nil, false
	}
	if !execSpec.PersistedFinalState() || !execSpec.ExecutionStatus().IsInFinalState() {
		return nil, false
	}
	return execSpec, true
}
