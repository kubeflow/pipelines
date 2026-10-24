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

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func (s *runtimeStore) CreateTask(task *model.Task) (*model.Task, error) {
	if task == nil || task.RunUUID == "" {
		return nil, errors.New("runtime task creation requires a public task and owning run ID")
	}
	if terminalRuntimeTask(task.State) {
		return nil, errors.New("runtime task creation requires a non-terminal public task; use FinalizePending for cached or skipped tasks")
	}
	if err := checkLeaseRun(s.tasks.lease, task.RunUUID); err != nil {
		return nil, err
	}
	tx, err := beginStoreMutation(s.tasks.db, s.tasks.lease)
	if err != nil {
		return nil, fmt.Errorf("begin pending runtime task creation: %w", err)
	}
	defer tx.Rollback()
	if err := s.lockRunID(tx, task.RunUUID, false); err != nil {
		return nil, err
	}
	if task.ParentTaskUUID != nil && *task.ParentTaskUUID != "" {
		parent, err := s.tasks.getTaskForUpdate(tx, *task.ParentTaskUUID)
		if err != nil {
			return nil, err
		}
		if parent.RunUUID != task.RunUUID {
			return nil, errors.New("runtime task parent must belong to the same run")
		}
	}
	public, err := s.tasks.createTaskWithExecutor(tx, task)
	if err != nil {
		return nil, err
	}
	if err := validatePendingTaskIdentity(public, task); err != nil {
		return nil, err
	}
	current, err := s.getWithExecutor(tx, public.UUID, true)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if !terminalRuntimeTask(public.State) {
			if err := s.insert(tx, &model.RuntimeTask{
				TaskUUID: public.UUID, RunUUID: public.RunUUID, State: model.RuntimeTaskWaiting,
			}); err != nil {
				return nil, err
			}
		}
	case err != nil:
		return nil, err
	case current.RunUUID != public.RunUUID || terminalRuntimeTask(public.State):
		return nil, ErrRuntimeTaskConflict
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pending runtime task creation: %w", err)
	}
	return public, nil
}

func runtimePendingPhase(task *model.RuntimeTask) bool {
	return runtimeWaitingPhase(task.State) && task.ExecutionReference == nil
}

func runtimeWaitingPhase(state string) bool {
	return state == model.RuntimeTaskWaiting || state == model.RuntimeTaskReady
}

func runtimeNeverStartedPending(task *model.RuntimeTask) bool {
	return task.AttemptUUID == nil && runtimeWaitingPhase(task.State) &&
		len(task.ExecutionRequest) == 0 && len(task.ExecutionReference) == 0
}

func validatePendingTaskIdentity(public, update *model.Task) error {
	if update.RunUUID != "" && update.RunUUID != public.RunUUID {
		return errors.New("pending runtime task cannot change its owning run")
	}
	if (update.Namespace != "" && update.Namespace != public.Namespace) ||
		(update.Name != "" && update.Name != public.Name) ||
		(update.ScopePath != "" && update.ScopePath != public.ScopePath) ||
		(update.Type != 0 && update.Type != public.Type) ||
		(update.ParentTaskUUID != nil && normalizedParentTaskUUID(update.ParentTaskUUID) != normalizedParentTaskUUID(public.ParentTaskUUID)) {
		return errors.New("pending runtime task cannot change its namespace or logical identity")
	}
	if update.TypeAttrs != nil {
		oldIteration, err := taskIterationIndex(public.TypeAttrs)
		if err != nil {
			return err
		}
		newIteration, err := taskIterationIndex(update.TypeAttrs)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(oldIteration, newIteration) {
			return errors.New("pending runtime task cannot change its iteration identity")
		}
	}
	return nil
}

func (s *runtimeStore) FinalizePending(taskID string, taskUpdate *model.Task, links []*model.ArtifactTask) error {
	if taskUpdate == nil || !terminalRuntimeTask(taskUpdate.State) {
		return errors.New("pending runtime finalization requires a terminal public task state")
	}
	if len(taskUpdate.OutputArtifactsHydrated) != 0 {
		return errors.New("pending runtime finalization reuses cached output links, not new attempt artifacts")
	}
	if len(links) != 0 && taskUpdate.State != model.TaskStatus(apiv2.PipelineTask_CACHED) {
		return errors.New("only cached pending tasks may copy output links")
	}
	skipped := taskUpdate.State == model.TaskStatus(apiv2.PipelineTask_SKIPPED)
	return s.withAttempt(taskID, "", true, func(tx sqlTransaction, current *model.RuntimeTask, public *model.Task) error {
		if err := validatePublicRuntimeUpdate(taskID, current.RunUUID, taskUpdate); err != nil {
			return err
		}
		if err := validatePendingTaskIdentity(public, taskUpdate); err != nil {
			return err
		}
		if terminalRuntimeTask(public.State) {
			if !runtimeNeverStartedPending(current) && (!runtimeCancellationPhase(current.State) || current.ExecutionReference != nil) {
				return ErrRuntimeTaskConflict
			}
			if len(links) != 0 {
				return errors.New("already-terminal pending task cleanup cannot copy output links")
			}
			return s.finalize(tx, current, public, taskUpdate)
		}
		run, err := s.lockManagedRunLifecycle(tx, public.RunUUID)
		if err != nil {
			return err
		}
		runState := run.effectiveState()
		if (runState == model.RuntimeStateFailed || runState == model.RuntimeStateCanceled) && runtimeWaitingPhase(current.State) {
			// A rejected/recovered run may retain framework rows that never
			// reached request preparation or submission. Only skip those rows;
			// an opaque request/reference requires the normal cleanup fence.
			if taskUpdate.State != model.TaskStatus(apiv2.PipelineTask_SKIPPED) ||
				!runtimeNeverStartedPending(current) || len(taskUpdate.OutputParameters) != 0 {
				if runState == model.RuntimeStateCanceled {
					return ErrRuntimeRunCanceled
				}
				return ErrRuntimeTaskConflict
			}
			return s.finalize(tx, current, public, taskUpdate)
		}
		if runtimeCancellationPhase(current.State) {
			if !skipped {
				return ErrRuntimeRunCanceled
			}
			if current.ExecutionReference != nil {
				return ErrRuntimeTaskConflict
			}
		} else {
			if !runtimePendingPhase(current) {
				return ErrRuntimeTaskConflict
			}
			// A terminal update alone cannot bypass accepted run cancellation:
			// the framework must first fence the row through cancellation.
			if err := s.lockRunID(tx, public.RunUUID, false); err != nil {
				return err
			}
		}
		if err := s.copyCachedOutputLinks(tx, public, links); err != nil {
			return err
		}
		return s.finalize(tx, current, public, taskUpdate)
	})
}
