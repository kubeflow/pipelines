package resource

import (
	"context"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func (r *ResourceManager) reconcileStrandedTasks(ctx context.Context, runID string, execSpec util.ExecutionSpec, isFinalState bool) {
	nodeStatuses := execSpec.ExecutionStatus().NodeStatuses()
	if len(nodeStatuses) == 0 {
		return
	}

	opts, _ := list.NewOptions(&model.Task{}, 1000, "", nil)
	tasks, _, _, err := r.taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID}}, opts)
	if err != nil {
		glog.Warningf("Failed to list tasks for run %s during stranded task reconciliation: %v", runID, err)
		return
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

		if len(task.Pods) == 0 {
			continue
		}
		if len(task.Pods) == 0 {
			continue
		}

		podObj, ok := task.Pods[0].(map[string]interface{})
		if !ok {
			continue
		}

		podName, ok := podObj["name"].(string)
		if !ok || podName == "" {
			continue
		}

		nodeStatus, exists := nodeStatuses[podName]
		if !exists {
			continue
		}

		if isTerminalNodeState(nodeStatus.State, isFinalState) {
			glog.Infof("Reconciling stranded task %s (pod: %s) which is RUNNING but its engine node is %s", task.UUID, podName, nodeStatus.State)

			// Map engine state to MLMD state
			var newState apiv2beta1.PipelineTask_TaskState
			switch nodeStatus.State {
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
			task.FinishedInSec = nodeStatus.FinishTime
			if task.FinishedInSec == 0 {
				task.FinishedInSec = r.time.Now().Unix()
			}

			_, err := r.taskStore.UpdateTask(task)
			if err != nil {
				glog.Warningf("Failed to atomically reconcile stranded task %s: %v", task.UUID, err)
				continue
			}
			reconciledAny = true
		}
	}

	// 2. If we reconciled any task, we must aggregate the DAG states to ensure parent DAGs fail before exit handlers run.
	if reconciledAny {
		r.aggregateDAGTaskStatuses(runID)
	}
}

// aggregateDAGTaskStatuses traverses the DAG tree from bottom to top, updating parent task states.
func (r *ResourceManager) aggregateDAGTaskStatuses(runID string) {
	// Re-fetch tasks since some might have been updated by reconciliation
	opts, _ := list.NewOptions(&model.Task{}, 1000, "", nil)
	tasks, _, _, err := r.taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID}}, opts)
	if err != nil {
		glog.Warningf("Failed to list tasks for run %s during DAG aggregation: %v", runID, err)
		return
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

				_, err := r.taskStore.UpdateTask(parent)
				if err != nil {
					glog.Warningf("Failed to atomically update aggregated DAG task %s: %v", parentID, err)
					continue
				}
				changed = true
			}
		}
	}
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
