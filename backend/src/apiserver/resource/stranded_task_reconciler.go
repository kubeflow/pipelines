package resource

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func (r *ResourceManager) reconcileStrandedTasks(ctx context.Context, run *model.Run, execSpec util.ExecutionSpec, isFinalState bool) error {
	if run == nil || execSpec == nil || execSpec.ExecutionStatus() == nil {
		return nil
	}

	// Retry fencing: do not let reports from older retry generations reconcile retried tasks
	if run.RetryGeneration > 0 && reportedRetryGeneration(execSpec.ExecutionObjectMeta()) < run.RetryGeneration {
		return nil
	}

	nodeStatuses := execSpec.ExecutionStatus().NodeStatuses()
	if len(nodeStatuses) == 0 {
		return nil
	}

	opts, _ := list.NewOptions(&model.Task{}, 1000, "", nil)
	tasks, _, _, err := r.taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}}, opts)
	if err != nil {
		return fmt.Errorf("failed to list tasks for run %s during stranded task reconciliation: %w", run.UUID, err)
	}

	// 1. Reconcile terminal engine nodes with RUNNING tasks
	reconciledAny := false
	for _, task := range tasks {
		if task.State != model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) {
			continue
		}

		if task.Type == model.TaskType(apiv2beta1.PipelineTask_DAG) {
			continue // DAG states are determined by their children, not their driver pods
		}

		podNames := podNamesFromTask(task)
		if len(podNames) == 0 {
			continue
		}

		var matchingNode *util.NodeStatus
		for _, podName := range podNames {
			if node, exists := nodeStatuses[podName]; exists {
				matchingNode = &node
				break
			}
		}
		if matchingNode == nil {
			continue
		}

		if isTerminalNodeState(matchingNode.State, isFinalState) {
			glog.Infof("Reconciling stranded task %s which is RUNNING but its engine node is %s", task.UUID, matchingNode.State)

			// Map engine state to MLMD state
			var newState apiv2beta1.PipelineTask_TaskState
			switch matchingNode.State {
			case "Succeeded":
				newState = apiv2beta1.PipelineTask_SUCCEEDED
			case "Failed", "Error":
				newState = apiv2beta1.PipelineTask_FAILED
			case "Skipped", "Omitted":
				newState = apiv2beta1.PipelineTask_SKIPPED
			default:
				newState = apiv2beta1.PipelineTask_FAILED
			}

			task.State = model.TaskStatus(newState)
			task.FinishedInSec = matchingNode.FinishTime
			if task.FinishedInSec == 0 {
				task.FinishedInSec = r.time.Now().Unix()
			}
			if matchingNode.Message != "" {
				task.StatusMetadata = model.JSONData{"message": matchingNode.Message}
				lm := model.LargeText(matchingNode.Message)
				task.LifecycleMessage = &lm
			}

			if _, err := r.taskStore.UpdateTask(task); err != nil {
				return fmt.Errorf("failed to atomically reconcile stranded task %s: %w", task.UUID, err)
			}
			reconciledAny = true
		}
	}

	// 2. If we reconciled any task, we must aggregate the DAG states to ensure parent DAGs fail before exit handlers run.
	if reconciledAny {
		if err := r.aggregateDAGTaskStatuses(run.UUID); err != nil {
			return err
		}
	}
	return nil
}

// aggregateDAGTaskStatuses traverses the DAG tree from bottom to top, updating parent task states.
func (r *ResourceManager) aggregateDAGTaskStatuses(runID string) error {
	// Re-fetch tasks since some might have been updated by reconciliation
	opts, _ := list.NewOptions(&model.Task{}, 1000, "", nil)
	tasks, _, _, err := r.taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID}}, opts)
	if err != nil {
		return fmt.Errorf("failed to list tasks for run %s during DAG aggregation: %w", runID, err)
	}

	// Map parent UUID -> list of child tasks
	childrenByParent := make(map[string][]*model.Task)
	tasksByID := make(map[string]*model.Task)
	for _, t := range tasks {
		tasksByID[t.UUID] = t
		if t.ParentTaskUUID != nil && *t.ParentTaskUUID != "" {
			childrenByParent[*t.ParentTaskUUID] = append(childrenByParent[*t.ParentTaskUUID], t)
		}
	}

	// A simple topological aggregation: repeat until no more changes.
	// Since the maximum depth is small (usually <10), this is very fast.
	changed := true
	for changed {
		changed = false
		for parentID, children := range childrenByParent {
			parent, exists := tasksByID[parentID]
			if !exists || parent.State != model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) {
				continue
			}

			// We simplify aggregation for stranded DAGs:
			// If ANY child is FAILED, and NO children are RUNNING, the parent is FAILED.
			// Wait, if a sibling is still RUNNING, we DO NOT fail the parent yet!
			anyFailed := false
			anyRunning := false

			for _, child := range children {
				if child.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED) {
					anyFailed = true
				} else if child.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) {
					anyRunning = true
				}
			}

			if anyRunning {
				continue
			}

			// If no children are running, and at least one failed, fail the DAG.
			// If all succeeded, we leave it alone (launchers will handle it, or we could mark SUCCEEDED, but stranded tasks are mostly failures).
			if anyFailed {
				glog.Infof("Aggregating FAILED state to parent DAG task %s", parentID)
				parent.State = model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
				parent.FinishedInSec = r.time.Now().Unix()

				if _, err := r.taskStore.UpdateTask(parent); err != nil {
					return fmt.Errorf("failed to atomically update aggregated DAG task %s: %w", parentID, err)
				}
				changed = true
			}
		}
	}
	return nil
}

func isTerminalNodeState(state string, isFinalState bool) bool {
	if state == "Failed" || state == "Error" || state == "Skipped" || state == "Omitted" {
		return true
	}
	// For Succeeded, we only reconcile if the workflow itself is in a final state,
	// to avoid racing with the launcher reporting success and outputs.
	if state == "Succeeded" && isFinalState {
		return true
	}
	return false
}
