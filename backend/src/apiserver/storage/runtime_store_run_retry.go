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
	"math"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

// RetryRun atomically reopens a cleaned-up managed run for an explicit user retry.
func (s *runtimeStore) RetryRun(runID string, expectedGeneration int64, taskIDs []string, manifest model.LargeText, resetWorkspace bool) error {
	if runID == "" || expectedGeneration < 0 || expectedGeneration == math.MaxInt64 || strings.TrimSpace(string(manifest)) == "" {
		return errors.New("managed run retry requires a run ID, a nonnegative incrementable generation, and a runtime manifest")
	}
	if err := checkLeaseRun(s.tasks.lease, runID); err != nil {
		return err
	}
	selected := make(map[string]bool, len(taskIDs))
	for _, id := range taskIDs {
		if id == "" {
			return errors.New("managed run retry requires nonempty selected task IDs")
		}
		selected[id] = true
	}
	tx, err := beginStoreMutation(s.tasks.db, s.tasks.lease)
	if err != nil {
		return fmt.Errorf("begin atomic managed run retry: %w", err)
	}
	defer tx.Rollback()
	current, err := s.lockManagedRunRetry(tx, runID, expectedGeneration)
	if err != nil {
		return err
	}
	tasks, err := s.lockManagedRunRetryTasks(tx, runID, selected)
	if err != nil {
		return err
	}
	retryStartedAt := s.tasks.time.Now().Unix()
	for _, task := range tasks {
		if hasTaskRetryDiagnostics(task) {
			task.StateHistory, err = s.snapshotAttemptHistory(task, "")
			if err != nil {
				return err
			}
		}
		if err := s.deleteRetriedOutputRelationships(tx, task); err != nil {
			return err
		}
		if err := s.tasks.resetTaskForRetry(tx, task, retryStartedAt); err != nil {
			return err
		}
		if err := s.insert(tx, &model.RuntimeTask{
			TaskUUID: task.UUID, RunUUID: runID, State: model.RuntimeTaskWaiting,
		}); err != nil {
			return err
		}
	}
	if err := s.reopenRetriedRun(tx, runID, current, manifest, resetWorkspace); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit atomic managed run retry: %w", err)
	}
	return nil
}

type managedRunLifecycle struct {
	state      sql.NullString
	conditions sql.NullString
	storage    sql.NullString
	executor   sql.NullString
	generation sql.NullInt64
}

func (current *managedRunLifecycle) effectiveState() model.RuntimeState {
	if !current.state.Valid || current.state.String == "" {
		return model.RuntimeState(current.conditions.String).ToV2()
	}
	return model.RuntimeState(current.state.String).ToV2()
}

func (s *runtimeStore) lockManagedRunLifecycle(tx sqlTransaction, runID string) (*managedRunLifecycle, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Select(q("State"), q("Conditions"), q("StorageState"), q("ExecutorType"), q("RetryGeneration")).
		From(q("run_details")).Where(sq.Eq{q("UUID"): runID}).ToSql()
	if err != nil {
		return nil, err
	}
	current := &managedRunLifecycle{}
	if err := tx.QueryRow(s.tasks.dbDialect.SelectForUpdate(query), args...).
		Scan(&current.state, &current.conditions, &current.storage, &current.executor, &current.generation); errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: managed run %q no longer exists", ErrRuntimeTaskConflict, runID)
	} else if err != nil {
		return nil, fmt.Errorf("lock managed run lifecycle: %w", err)
	}
	return current, nil
}

func (s *runtimeStore) lockManagedRunRetry(tx sqlTransaction, runID string, expectedGeneration int64) (*managedRunLifecycle, error) {
	current, err := s.lockManagedRunLifecycle(tx, runID)
	if err != nil {
		return nil, err
	}
	if model.StorageState(current.storage.String).ToV2() == model.StorageStateArchived {
		return nil, NewArchivedRunRetryError(runID)
	}
	state := current.effectiveState()
	if !current.executor.Valid || current.executor.String == "" || (state != model.RuntimeStateFailed && state != model.RuntimeStateCanceled) ||
		current.generation.Int64 != expectedGeneration {
		return nil, fmt.Errorf("%w: retry requires a FAILED/CANCELED managed run at generation %d; reload the run", ErrRuntimeTaskConflict, expectedGeneration)
	}
	return current, nil
}

func (s *runtimeStore) checkRunRuntimeCleanup(tx sqlTransaction, runID string) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Select(q("TaskUUID")).From(q("runtime_tasks")).
		Where(sq.Eq{q("RunUUID"): runID}).Limit(1).ToSql()
	if err != nil {
		return err
	}
	var taskID string
	if err := tx.QueryRow(s.tasks.dbDialect.SelectForUpdate(query), args...).Scan(&taskID); err == nil {
		return fmt.Errorf("%w: finish prior runtime task cleanup before changing terminal run %q", ErrRuntimeTaskConflict, runID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check managed run cleanup: %w", err)
	}
	return nil
}

func (s *runtimeStore) lockManagedRunRetryTasks(tx sqlTransaction, runID string, selected map[string]bool) ([]*model.Task, error) {
	if err := s.checkRunRuntimeCleanup(tx, runID); err != nil {
		return nil, err
	}
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Select(dialect.QuoteAll(q, taskColumns)...).From(q("tasks")).
		Where(sq.Eq{q("RunUUID"): runID}).OrderBy(q("UUID")).ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(s.tasks.dbDialect.SelectForUpdate(query), args...)
	if err != nil {
		return nil, fmt.Errorf("lock managed retry public tasks: %w", err)
	}
	defer rows.Close()
	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		if !terminalRuntimeTask(task.State) {
			return nil, fmt.Errorf("%w: public task %q must be terminal before run retry", ErrRuntimeTaskConflict, task.UUID)
		}
		if selected[task.UUID] {
			tasks = append(tasks, task)
			delete(selected, task.UUID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(selected) != 0 {
		return nil, fmt.Errorf("%w: all selected task IDs must belong to run %q; reload the task selection", ErrRuntimeTaskConflict, runID)
	}
	return tasks, nil
}

func (s *runtimeStore) deleteRetriedOutputRelationships(tx sqlTransaction, task *model.Task) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Delete(q("artifact_tasks")).
		Where(sq.Eq{q("TaskID"): task.UUID, q("RunUUID"): task.RunUUID, q("Type"): outputArtifactTaskTypes()}).
		Where(sq.NotEq{q("ArtifactKey"): "executor-logs"}).ToSql()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("clear retried task outputs while retaining executor logs: %w", err)
	}
	return nil
}

func (s *runtimeStore) reopenRetriedRun(tx sqlTransaction, runID string, current *managedRunLifecycle, manifest model.LargeText, resetWorkspace bool) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	values := sq.Eq{
		q("State"): model.RuntimeStatePending, q("Conditions"): model.RuntimeStatePending.ToExecutionPhase(),
		q("FinishedAtInSec"): 0, q("RetryGeneration"): current.generation.Int64 + 1,
		q("RetryClaimedAtInSec"): 0, q("PipelineRuntimeManifest"): string(manifest),
	}
	if resetWorkspace {
		values[q("WorkspaceReference")] = nil
	}
	query, args, err := s.tasks.dbDialect.QueryBuilder().Update(q("run_details")).SetMap(values).
		Where(sq.Eq{q("UUID"): runID, q("State"): current.state, q("RetryGeneration"): current.generation}).ToSql()
	if err != nil {
		return err
	}
	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("reopen managed run and persist its retry manifest: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrRuntimeTaskConflict
	}
	return nil
}
