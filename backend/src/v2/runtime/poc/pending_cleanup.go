package poc

import (
	"fmt"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func (c *Coordinator) cleanupNeverDispatchedTask(run *model.Run, attempt *model.RuntimeTask) error {
	if attempt.AttemptUUID != nil || len(attempt.ExecutionRequest) != 0 || len(attempt.ExecutionReference) != 0 ||
		(attempt.State != model.RuntimeTaskWaiting && attempt.State != model.RuntimeTaskReady) {
		return fmt.Errorf("task %s does not have confirmed never-dispatched scheduling state", attempt.TaskUUID)
	}
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return err
	}
	if task.RunUUID != run.UUID || task.Namespace != run.Namespace {
		return fmt.Errorf("pending task %s does not belong to run %s", task.UUID, run.UUID)
	}
	if isTerminalTaskState(task.State) {
		return c.runtimeStore.FinalizePending(task.UUID, &model.Task{UUID: task.UUID, State: task.State}, nil)
	}
	if run.State != model.RuntimeStateFailed && run.State != model.RuntimeStateCanceled {
		return fmt.Errorf("run %s has no failed or canceled outcome authorizing undispatched cleanup", run.UUID)
	}
	return c.runtimeStore.FinalizePending(task.UUID, &model.Task{
		UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SKIPPED), FinishedInSec: c.time.Now().Unix(),
		StatusMetadata: mergeJSONData(task.StatusMetadata, model.JSONData{"message": "task was never dispatched before the run ended"}),
	}, nil)
}
