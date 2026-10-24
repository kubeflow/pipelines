package poc

import (
	"context"
	"encoding/json"
	"fmt"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func materializedIterationIndex(task *model.Task) (int, error) {
	data, err := json.Marshal(task.TypeAttrs)
	if err != nil {
		return 0, err
	}
	var attributes struct {
		Index *int `json:"iterationIndex"`
	}
	if err := json.Unmarshal(data, &attributes); err != nil {
		return 0, err
	}
	if attributes.Index == nil || *attributes.Index < 0 {
		return 0, fmt.Errorf("materialized loop task %s has no valid iteration index", task.UUID)
	}
	return *attributes.Index, nil
}

func (c *Coordinator) reconcileObsoleteIterations(ctx context.Context, run *model.Run, loopID string, count int) error {
	iterations, err := c.taskStore.GetChildTasks(loopID)
	if err != nil {
		return err
	}
	for _, task := range iterations {
		index, err := materializedIterationIndex(task)
		if err != nil {
			return err
		}
		if index >= count {
			if err := c.skipObsoleteIterationTree(ctx, run, task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Coordinator) skipObsoleteIterationTree(ctx context.Context, run *model.Run, task *model.Task) error {
	if err := ctx.Err(); err != nil {
		return pendingAttempt(err)
	}
	if task.RunUUID != run.UUID || task.Namespace != run.Namespace {
		return fmt.Errorf("obsolete iteration task %s belongs to a different run", task.UUID)
	}
	children, err := c.taskStore.GetChildTasks(task.UUID)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := c.skipObsoleteIterationTree(ctx, run, child); err != nil {
			return err
		}
	}
	pending, err := c.runtimeStore.Get(task.UUID)
	if runtimeTaskNotFound(err) && isTerminalTaskState(task.State) {
		return nil
	}
	if err != nil {
		return err
	}
	if pending.AttemptUUID != nil || len(pending.ExecutionRequest) != 0 || len(pending.ExecutionReference) != 0 ||
		(pending.State != model.RuntimeTaskWaiting && pending.State != model.RuntimeTaskReady) {
		return pendingAttempt(fmt.Errorf("obsolete iteration task %s still needs execution recovery or cleanup", task.UUID))
	}
	return c.runtimeStore.FinalizePending(task.UUID, &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED), FinishedInSec: c.time.Now().Unix(),
		StatusMetadata: mergeJSONData(task.StatusMetadata, model.JSONData{"message": "iteration is absent from the recomputed item list"}),
	}, nil)
}
