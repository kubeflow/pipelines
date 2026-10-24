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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ErrRuntimeTaskConflict means a task attempt or managed run lifecycle precondition
// is no longer current. Callers must reload rather than replay stale mutations.
var ErrRuntimeTaskConflict = errors.New("runtime task attempt no longer current; reload durable task state")

// ErrRuntimeRunCanceled prevents completed work from resurrecting a canceled run.
var ErrRuntimeRunCanceled = errors.New("runtime run is canceling or canceled; only cancellation cleanup may continue")

// RuntimeStoreInterface persists coordination state and attempt-fenced results.
// Update compares NULL when expectedAttempt is empty and may assign the first
// attempt only to a non-executing pending row. Checkpoint uses task's AttemptUUID
// as its fence and never changes that identity. BeginRetry replaces an attempt
// and increments RetryCount for automatic retries. A user-requested run retry
// instead uses RetryRun to reopen the run and restore selected tasks atomically.
type RuntimeStoreInterface interface {
	Get(taskID string) (*model.RuntimeTask, error)
	// CreateTask atomically creates the public task and a WAITING runtime row
	// with a NULL attempt. Logical-identity replays retain existing progress;
	// terminal tasks never regain a runtime row.
	CreateTask(task *model.Task) (*model.Task, error)
	Create(task *model.RuntimeTask) error
	Update(task *model.RuntimeTask, expectedAttempt string) error
	Checkpoint(task *model.RuntimeTask, taskUpdate *model.Task) error
	// CheckpointCancellation is framework-only: stopped-workload log capture and
	// cleanup progress may advance while cancellation blocks launcher reports.
	CheckpointCancellation(task *model.RuntimeTask, taskUpdate *model.Task) error
	// CheckpointCancellationArtifacts is worker-only. It checkpoints deterministic
	// executor-logs OUTPUT artifacts after a current attempt enters cancellation.
	// The public task stays RUNNING; normal outputs cannot use this bypass.
	CheckpointCancellationArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error
	// CheckpointArtifacts atomically stores provisional results and output
	// relationships during execution, publication, failure-log, or finalization phases.
	// IDs must be attempt-scoped and deterministic; identical replays are safe,
	// collisions are errors.
	CheckpointArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error
	// Finalize may remove a leftover row for an already-terminal public task
	// only when the outcome matches; it does not rewrite terminal task metadata.
	Finalize(taskID, attemptID string, taskUpdate *model.Task) error
	// FinalizePending atomically finalizes a non-executing NULL-attempt task,
	// attaches optional existing cached outputs, and deletes its runtime row.
	// Links must name the destination task/run; empty link IDs are deterministic.
	// Cancellation cleanup requires SKIPPED and an existing cancellation phase.
	// FAILED/CANCELED runs may also skip never-started WAITING/READY rows with
	// empty execution request/reference, without changing the run outcome.
	// Already-terminal tasks permit same-outcome row cleanup without new links.
	// Missing/finalized rows and active attempts return ErrRuntimeTaskConflict.
	FinalizePending(taskID string, taskUpdate *model.Task, links []*model.ArtifactTask) error
	BeginRetry(taskID, previousAttemptID string, next *model.RuntimeTask) error
	// RetryRun reopens an unarchived, cleaned-up FAILED/CANCELED managed run at
	// expectedGeneration. It resets exactly the selected terminal tasks, retaining
	// executor-logs links/history, and writes PipelineRuntimeManifest atomically.
	// Preconditions that changed return ErrRuntimeTaskConflict; scoped stores
	// also enforce their current run lease throughout the transaction.
	RetryRun(runID string, expectedGeneration int64, taskIDs []string, manifest model.LargeText, resetWorkspace bool) error
	// DeleteRun checks the terminal managed lifecycle, generation, cleanup, and
	// any scoped lease before deleting the run and its related rows atomically.
	// Successful deletion also removes the lease row.
	DeleteRun(runID string, expectedGeneration int64) error
	ListByRun(runID string) ([]*model.RuntimeTask, error)
	// FinalizeRun uses the run store's lifecycle CAS. Accepted cancellation
	// can only remain canceling or advance to canceled, never to success.
	FinalizeRun(runID string, state model.RuntimeState, finishedAt int64) error
}

type runtimeStore struct {
	tasks *TaskStore
}

var _ RuntimeStoreInterface = (*runtimeStore)(nil)

// RuntimeStore returns the runtime store using this task store's DB and dialect.
func (s *TaskStore) RuntimeStore() RuntimeStoreInterface {
	return &runtimeStore{tasks: s}
}

var runtimeTaskColumns = []string{
	"TaskUUID", "RunUUID", "AttemptUUID", "State", "RetryCount",
	"NextEligibleAt", "ExecutionRequest", "ExecutionReference",
}

func scanRuntimeTask(row interface{ Scan(...any) error }) (*model.RuntimeTask, error) {
	task := &model.RuntimeTask{}
	err := row.Scan(&task.TaskUUID, &task.RunUUID, &task.AttemptUUID, &task.State,
		&task.RetryCount, &task.NextEligibleAt, &task.ExecutionRequest, &task.ExecutionReference)
	return task, err
}

func (s *runtimeStore) Get(taskID string) (*model.RuntimeTask, error) {
	return s.getWithExecutor(s.tasks.db, taskID, false)
}

func (s *runtimeStore) getWithExecutor(executor taskSQLExecutor, taskID string, lock bool) (*model.RuntimeTask, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Select(dialect.QuoteAll(q, runtimeTaskColumns)...).From(q("runtime_tasks")).
		Where(sq.Eq{q("TaskUUID"): taskID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build runtime task query: %w", err)
	}
	if lock {
		query = s.tasks.dbDialect.SelectForUpdate(query)
	}
	task, err := scanRuntimeTask(executor.QueryRow(query, args...))
	if err != nil {
		return nil, fmt.Errorf("get runtime task %q: %w", taskID, err)
	}
	return task, nil
}

func validateRuntimeTask(task *model.RuntimeTask) error {
	if task == nil || task.TaskUUID == "" || task.RunUUID == "" {
		return errors.New("runtime task requires task and run IDs")
	}
	if task.AttemptUUID != nil && *task.AttemptUUID == "" {
		return errors.New("runtime attempt ID must be nonempty or NULL")
	}
	if task.RetryCount < 0 {
		return errors.New("runtime retry count cannot be negative")
	}
	switch task.State {
	case model.RuntimeTaskWaiting, model.RuntimeTaskReady, model.RuntimeTaskSubmitting,
		model.RuntimeTaskRunning, model.RuntimeTaskPublishing, model.RuntimeTaskFailed,
		model.RuntimeTaskBackoff, model.RuntimeTaskInfrastructureBackoff,
		model.RuntimeTaskCanceling, model.RuntimeTaskFinalizing,
		model.RuntimeTaskFailureFinalizing, model.RuntimeTaskCancelFinalizing:
		return nil
	default:
		return fmt.Errorf("invalid runtime task phase %q", task.State)
	}
}

func (s *runtimeStore) values(task *model.RuntimeTask) (sq.Eq, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	request, err := nullableRuntimeJSON(task.ExecutionRequest)
	if err != nil {
		return nil, fmt.Errorf("encode runtime execution request: %w", err)
	}
	reference, err := nullableRuntimeJSON(task.ExecutionReference)
	if err != nil {
		return nil, fmt.Errorf("encode runtime execution reference: %w", err)
	}
	return sq.Eq{
		q("TaskUUID"): task.TaskUUID, q("RunUUID"): task.RunUUID,
		q("AttemptUUID"): task.AttemptUUID, q("State"): task.State,
		q("RetryCount"): task.RetryCount, q("NextEligibleAt"): task.NextEligibleAt,
		q("ExecutionRequest"): request, q("ExecutionReference"): reference,
	}, nil
}

func nullableRuntimeJSON(value model.JSONData) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	return string(data), err
}

func (s *runtimeStore) Create(task *model.RuntimeTask) error {
	if err := validateRuntimeTask(task); err != nil {
		return err
	}
	if err := checkLeaseRun(s.tasks.lease, task.RunUUID); err != nil {
		return err
	}
	tx, err := beginStoreMutation(s.tasks.db, s.tasks.lease)
	if err != nil {
		return fmt.Errorf("begin runtime task creation: %w", err)
	}
	defer tx.Rollback()
	if err := s.lockRun(tx, task.TaskUUID, false); err != nil {
		return err
	}
	public, err := s.tasks.getTaskForUpdate(tx, task.TaskUUID)
	if err != nil {
		return err
	}
	if public.RunUUID != task.RunUUID {
		return errors.New("runtime task run ID does not match the owning public task")
	}
	if terminalRuntimeTask(public.State) {
		return ErrRuntimeTaskConflict
	}
	if err := s.insert(tx, task); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit runtime task creation: %w", err)
	}
	return nil
}

func (s *runtimeStore) insert(tx sqlTransaction, task *model.RuntimeTask) error {
	values, err := s.values(task)
	if err != nil {
		return err
	}
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Insert(s.tasks.dbDialect.QuoteIdentifier("runtime_tasks")).SetMap(values).ToSql()
	if err != nil {
		return fmt.Errorf("build runtime task insert: %w", err)
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("create runtime task %q: %w", task.TaskUUID, err)
	}
	return nil
}

// withAttempt locks the public task before its coordination row, consistently
// with task deletion and FK checks. No read/merge/write crosses a transaction.
func (s *runtimeStore) withAttempt(taskID, attemptID string, allowCanceled bool, change func(sqlTransaction, *model.RuntimeTask, *model.Task) error) error {
	tx, err := beginStoreMutation(s.tasks.db, s.tasks.lease)
	if err != nil {
		return fmt.Errorf("begin runtime task transaction: %w", err)
	}
	defer tx.Rollback()
	if err := s.lockRun(tx, taskID, allowCanceled); err != nil {
		return err
	}
	public, err := s.tasks.getTaskForUpdate(tx, taskID)
	if IsNotFound(err) {
		return fmt.Errorf("%w: public task %q unavailable: %v", ErrRuntimeTaskConflict, taskID, err)
	}
	if err != nil {
		return err
	}
	current, err := s.getWithExecutor(tx, taskID, true)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRuntimeTaskConflict
	}
	if err != nil {
		return fmt.Errorf("lock runtime task %q: %w", taskID, err)
	}
	if (attemptID == "" && current.AttemptUUID != nil) || runtimeAttempt(current) != attemptID || current.RunUUID != public.RunUUID {
		return ErrRuntimeTaskConflict
	}
	if err := change(tx, current, public); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit runtime task %q: %w", taskID, err)
	}
	return nil
}

func (s *runtimeStore) lockRun(tx sqlTransaction, taskID string, allowCanceled bool) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	qb := s.tasks.dbDialect.QueryBuilder()
	query, args, err := qb.Select(q("RunUUID")).From(q("tasks")).Where(sq.Eq{q("UUID"): taskID}).ToSql()
	if err != nil {
		return err
	}
	var runID string
	if err := tx.QueryRow(query, args...).Scan(&runID); errors.Is(err, sql.ErrNoRows) {
		return ErrRuntimeTaskConflict
	} else if err != nil {
		return fmt.Errorf("read runtime task ownership: %w", err)
	}
	return s.lockRunID(tx, runID, allowCanceled)
}

func (s *runtimeStore) lockRunID(tx sqlTransaction, runID string, allowCanceled bool) error {
	if err := checkLeaseRun(s.tasks.lease, runID); err != nil {
		return err
	}
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Select(q("State"), q("Conditions")).From(q("run_details")).
		Where(sq.Eq{q("UUID"): runID}).ToSql()
	if err != nil {
		return err
	}
	var state, conditions sql.NullString
	if err := tx.QueryRow(s.tasks.dbDialect.SelectForUpdate(query), args...).Scan(&state, &conditions); errors.Is(err, sql.ErrNoRows) {
		return ErrRuntimeTaskConflict
	} else if err != nil {
		return fmt.Errorf("lock owning runtime run: %w", err)
	}
	current := model.RuntimeState(state.String).ToV2()
	if current == model.RuntimeStateUnspecified {
		current = model.RuntimeState(conditions.String).ToV2()
	}
	if !allowCanceled && (current == model.RuntimeStateCancelling || current == model.RuntimeStateCanceled) {
		return ErrRuntimeRunCanceled
	}
	return nil
}

func runtimeAttempt(task *model.RuntimeTask) string {
	if task.AttemptUUID == nil {
		return ""
	}
	return *task.AttemptUUID
}

func (s *runtimeStore) replace(tx sqlTransaction, task *model.RuntimeTask) error {
	values, err := s.values(task)
	if err != nil {
		return err
	}
	q := s.tasks.dbDialect.QuoteIdentifier
	delete(values, q("TaskUUID"))
	delete(values, q("RunUUID"))
	query, args, err := s.tasks.dbDialect.QueryBuilder().Update(q("runtime_tasks")).
		SetMap(values).Where(sq.Eq{q("TaskUUID"): task.TaskUUID, q("RunUUID"): task.RunUUID}).ToSql()
	if err != nil {
		return fmt.Errorf("build runtime task update: %w", err)
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("update runtime task %q: %w", task.TaskUUID, err)
	}
	return nil
}

func validateRuntimeReplacement(current, next *model.RuntimeTask, allowNewAttempt bool) error {
	if runtimeCancellationPhase(current.State) && (next.State != current.State || runtimeAttempt(current) != runtimeAttempt(next)) {
		return ErrRuntimeRunCanceled
	}
	return validateRuntimeAttemptFields(current, next, allowNewAttempt)
}

func runtimeCancellationPhase(state string) bool {
	return state == model.RuntimeTaskCanceling || state == model.RuntimeTaskCancelFinalizing
}

func validateRuntimeAttemptFields(current, next *model.RuntimeTask, allowNewAttempt bool) error {
	if current.RunUUID != next.RunUUID {
		return errors.New("runtime task cannot change its owning run")
	}
	if next.RetryCount != current.RetryCount {
		return errors.New("only BeginRetry may change the runtime retry count")
	}
	if runtimeAttempt(current) != runtimeAttempt(next) && (!allowNewAttempt || current.AttemptUUID != nil) {
		return errors.New("only BeginRetry may replace an existing runtime attempt")
	}
	if current.AttemptUUID != nil && current.ExecutionRequest != nil {
		oldRequest, err := canonicalRuntimeRequest(current.ExecutionRequest)
		if err != nil {
			return err
		}
		newRequest, err := canonicalRuntimeRequest(next.ExecutionRequest)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(oldRequest, newRequest) {
			return errors.New("execution request is immutable within an attempt")
		}
	}
	return nil
}

func canonicalRuntimeRequest(request model.JSONData) (any, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode immutable runtime execution request: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil, fmt.Errorf("normalize immutable runtime execution request: %w", err)
	}
	return normalized, nil
}

func (s *runtimeStore) Update(task *model.RuntimeTask, expectedAttempt string) error {
	if err := validateRuntimeTask(task); err != nil {
		return err
	}
	return s.withAttempt(task.TaskUUID, expectedAttempt, task.State == model.RuntimeTaskCanceling, func(tx sqlTransaction, current *model.RuntimeTask, _ *model.Task) error {
		if err := validateRuntimeReplacement(current, task, true); err != nil {
			return err
		}
		if current.AttemptUUID == nil && task.AttemptUUID != nil {
			if runtimeCancellationPhase(task.State) {
				return ErrRuntimeRunCanceled
			}
			if !runtimePendingPhase(current) {
				return errors.New("first runtime attempt requires a non-executing WAITING or READY row")
			}
		}
		return s.replace(tx, task)
	})
}

func validatePublicRuntimeUpdate(taskID, runID string, update *model.Task) error {
	if update == nil || update.UUID != taskID {
		return errors.New("runtime public task update must specify the same task ID")
	}
	if update.RunUUID != "" && update.RunUUID != runID {
		return errors.New("runtime public task update cannot change the owning run")
	}
	return nil
}

func terminalRuntimeTask(state model.TaskStatus) bool {
	switch apiv2.PipelineTask_TaskState(state) {
	case apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_FAILED, apiv2.PipelineTask_SKIPPED, apiv2.PipelineTask_CACHED:
		return true
	default:
		return false
	}
}

func (s *runtimeStore) Checkpoint(task *model.RuntimeTask, taskUpdate *model.Task) error {
	return s.checkpoint(task, taskUpdate, nil, nil)
}

func (s *runtimeStore) CheckpointCancellation(task *model.RuntimeTask, taskUpdate *model.Task) error {
	return s.checkpointCancellation(task, taskUpdate, nil, nil, false)
}

func (s *runtimeStore) CheckpointCancellationArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	return s.checkpointCancellation(task, taskUpdate, artifacts, links, true)
}

func (s *runtimeStore) checkpointCancellation(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask, logsOnly bool) error {
	if err := validateRuntimeTask(task); err != nil {
		return err
	}
	if !runtimeCancellationPhase(task.State) {
		return errors.New("cancellation checkpoint requires CANCELING or cancel-finalizing phase")
	}
	if err := validatePublicRuntimeUpdate(task.TaskUUID, task.RunUUID, taskUpdate); err != nil {
		return err
	}
	if terminalRuntimeTask(taskUpdate.State) {
		return errors.New("cancellation checkpoint must keep public results provisional until Finalize")
	}
	if logsOnly {
		if err := validateCancellationLogArtifacts(task, taskUpdate, links); err != nil {
			return err
		}
	}
	return s.withAttempt(task.TaskUUID, runtimeAttempt(task), true, func(tx sqlTransaction, current *model.RuntimeTask, public *model.Task) error {
		if logsOnly {
			if !runtimeCancellationPhase(current.State) {
				return ErrRuntimeRunCanceled
			}
			if public.State != model.TaskStatus(apiv2.PipelineTask_RUNNING) {
				return ErrRuntimeTaskConflict
			}
		}
		if current.State == model.RuntimeTaskCancelFinalizing && task.State != model.RuntimeTaskCancelFinalizing {
			return errors.New("cancellation checkpoint cannot return from cleanup to log capture")
		}
		if err := validateRuntimeAttemptFields(current, task, false); err != nil {
			return err
		}
		return s.writeCheckpoint(tx, current, public, task, taskUpdate, artifacts, links)
	})
}

func validateCancellationLogArtifacts(task *model.RuntimeTask, update *model.Task, links []*model.ArtifactTask) error {
	if task.AttemptUUID == nil {
		return errors.New("cancellation log artifacts require a current nonempty execution attempt")
	}
	if update.State != model.TaskStatus(apiv2.PipelineTask_RUNNING) || len(update.OutputParameters) != 0 {
		return errors.New("cancellation log artifacts require RUNNING provisional public state without normal output parameters")
	}
	for _, output := range update.OutputArtifactsHydrated {
		if output.Key != "executor-logs" || output.Type != apiv2.IOType_OUTPUT {
			return errors.New("cancellation artifact checkpoints only permit executor-logs OUTPUT artifacts")
		}
	}
	for _, link := range links {
		if link == nil || link.ArtifactKey != "executor-logs" || link.Type != model.IOType(apiv2.IOType_OUTPUT) {
			return errors.New("cancellation artifact checkpoints only permit executor-logs OUTPUT links")
		}
	}
	return nil
}

func (s *runtimeStore) CheckpointArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	if task != nil {
		switch task.State {
		case model.RuntimeTaskRunning, model.RuntimeTaskPublishing, model.RuntimeTaskFailed,
			model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing:
			return s.checkpoint(task, taskUpdate, artifacts, links)
		}
	}
	return errors.New("artifact checkpoint requires a runtime execution, publication, failure, or finalization phase")
}

func (s *runtimeStore) checkpoint(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	if err := validateRuntimeTask(task); err != nil {
		return err
	}
	if err := validatePublicRuntimeUpdate(task.TaskUUID, task.RunUUID, taskUpdate); err != nil {
		return err
	}
	if terminalRuntimeTask(taskUpdate.State) {
		return errors.New("checkpoint outputs remain provisional; use Finalize for terminal task state")
	}
	return s.withAttempt(task.TaskUUID, runtimeAttempt(task), false, func(tx sqlTransaction, current *model.RuntimeTask, public *model.Task) error {
		if runtimeCancellationPhase(current.State) {
			return ErrRuntimeRunCanceled
		}
		if err := validateRuntimeReplacement(current, task, false); err != nil {
			return err
		}
		return s.writeCheckpoint(tx, current, public, task, taskUpdate, artifacts, links)
	})
}

func (s *runtimeStore) writeCheckpoint(tx sqlTransaction, current *model.RuntimeTask, public *model.Task, task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	if err := s.tasks.updateTaskInTransaction(tx, taskUpdate, false); err != nil {
		return err
	}
	if err := s.checkpointArtifacts(tx, current, public, taskUpdate.OutputArtifactsHydrated); err != nil {
		return err
	}
	if err := s.checkpointSuppliedArtifacts(tx, public, artifacts, links); err != nil {
		return err
	}
	return s.replace(tx, task)
}

func (s *runtimeStore) Finalize(taskID, attemptID string, taskUpdate *model.Task) error {
	if attemptID == "" {
		return errors.New("runtime attempt finalization requires an attempt ID; use FinalizePending for NULL attempts")
	}
	if taskUpdate == nil || !terminalRuntimeTask(taskUpdate.State) {
		return errors.New("runtime finalization requires a terminal public task state")
	}
	allowCanceled := taskUpdate.State == model.TaskStatus(apiv2.PipelineTask_FAILED) || taskUpdate.State == model.TaskStatus(apiv2.PipelineTask_SKIPPED)
	return s.withAttempt(taskID, attemptID, true, func(tx sqlTransaction, current *model.RuntimeTask, public *model.Task) error {
		if err := validatePublicRuntimeUpdate(taskID, current.RunUUID, taskUpdate); err != nil {
			return err
		}
		if !terminalRuntimeTask(public.State) {
			if runtimeCancellationPhase(current.State) && !allowCanceled {
				return ErrRuntimeRunCanceled
			}
			if !allowCanceled {
				if err := s.lockRunID(tx, public.RunUUID, false); err != nil {
					return err
				}
			}
		}
		return s.finalize(tx, current, public, taskUpdate)
	})
}

func (s *runtimeStore) finalize(tx sqlTransaction, current *model.RuntimeTask, public, taskUpdate *model.Task) error {
	if terminalRuntimeTask(public.State) {
		if public.State != taskUpdate.State {
			return ErrRuntimeTaskConflict
		}
	} else {
		if err := s.tasks.updateTaskInTransaction(tx, taskUpdate, true); err != nil {
			return err
		}
		if err := s.checkpointArtifacts(tx, current, public, taskUpdate.OutputArtifactsHydrated); err != nil {
			return err
		}
	}
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Delete(q("runtime_tasks")).
		Where(sq.Eq{q("TaskUUID"): current.TaskUUID, q("RunUUID"): current.RunUUID, q("AttemptUUID"): current.AttemptUUID}).ToSql()
	if err != nil {
		return fmt.Errorf("build runtime finalization: %w", err)
	}
	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("delete finalized runtime task: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count finalized runtime tasks: %w", err)
	}
	if rows != 1 {
		return ErrRuntimeTaskConflict
	}
	return nil
}

func (s *runtimeStore) BeginRetry(taskID, previousAttemptID string, next *model.RuntimeTask) error {
	if err := validateRuntimeTask(next); err != nil {
		return err
	}
	if taskID != next.TaskUUID || previousAttemptID == "" || next.AttemptUUID == nil || *next.AttemptUUID == previousAttemptID {
		return errors.New("retry requires the same task ID and a new nonempty attempt ID")
	}
	var retryCount int
	err := s.withAttempt(taskID, previousAttemptID, false, func(tx sqlTransaction, current *model.RuntimeTask, public *model.Task) error {
		if runtimeCancellationPhase(current.State) {
			return ErrRuntimeRunCanceled
		}
		if current.RunUUID != next.RunUUID {
			return errors.New("retry cannot change the owning run")
		}
		retryCount = current.RetryCount + 1
		replacement := *next
		replacement.RetryCount = retryCount
		if err := s.resetPublicAttempt(tx, public, previousAttemptID); err != nil {
			return err
		}
		return s.replace(tx, &replacement)
	})
	if err == nil {
		next.RetryCount = retryCount
	}
	return err
}

func (s *runtimeStore) snapshotAttemptHistory(task *model.Task, previousAttemptID string) (model.JSONSlice, error) {
	// Keep the old attempt's diagnostics API-readable after clearing current
	// metadata. Struct details preserve log references without extending TaskStatus.
	snapshot, err := json.Marshal(map[string]any{
		"attempt_uuid": previousAttemptID, "status_metadata": task.StatusMetadata,
		"pods": task.Pods, "lifecycle_message": task.LifecycleMessage, "lifecycle_category": task.LifecycleCategory,
	})
	if err != nil {
		return nil, fmt.Errorf("encode previous attempt diagnostics: %w", err)
	}
	details := &structpb.Struct{}
	if err := details.UnmarshalJSON(snapshot); err != nil {
		return nil, fmt.Errorf("encode previous attempt diagnostic details: %w", err)
	}
	diagnostic, err := anypb.New(details)
	if err != nil {
		return nil, err
	}
	entry, err := model.ProtoSliceToJSONSlice([]*apiv2.PipelineTask_TaskStatus{{
		UpdateTime: timestamppb.New(s.tasks.time.Now()), State: apiv2.PipelineTask_TaskState(task.State),
		Error: &rpcstatus.Status{Message: "Previous execution attempt diagnostics", Details: []*anypb.Any{diagnostic}},
	}})
	if err != nil {
		return nil, err
	}
	return append(append(model.JSONSlice{}, task.StateHistory...), entry...), nil
}

func hasTaskRetryDiagnostics(task *model.Task) bool {
	return task.StatusMetadata != nil || len(task.Pods) != 0 || task.LifecycleMessage != nil || task.LifecycleCategory != nil
}

func (s *runtimeStore) resetPublicAttempt(tx sqlTransaction, task *model.Task, previousAttemptID string) error {
	retryHistory, err := s.snapshotAttemptHistory(task, previousAttemptID)
	if err != nil {
		return err
	}
	history, err := json.Marshal(retryHistory)
	if err != nil {
		return fmt.Errorf("encode retry state history: %w", err)
	}
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Update(q("tasks")).SetMap(sq.Eq{
		q("State"): model.TaskStatus(apiv2.PipelineTask_RUNNING), q("StartedInSec"): 0, q("FinishedInSec"): 0,
		q("StatusMetadata"): nil, q("pods"): "[]", q("OutputParameters"): "[]",
		q("StateHistory"): string(history), q("LifecycleMessage"): nil, q("LifecycleCategory"): nil,
	}).Where(sq.Eq{q("UUID"): task.UUID}).ToSql()
	if err != nil {
		return fmt.Errorf("build public retry reset: %w", err)
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("reset public retry attempt: %w", err)
	}
	return s.deleteOutputRelationships(tx, task)
}

func (s *runtimeStore) deleteOutputRelationships(tx sqlTransaction, task *model.Task) error {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Delete(q("artifact_tasks")).
		Where(sq.Eq{q("TaskID"): task.UUID, q("RunUUID"): task.RunUUID, q("Type"): model.IOType(apiv2.IOType_OUTPUT)}).ToSql()
	if err != nil {
		return fmt.Errorf("build prior output relationship cleanup: %w", err)
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("delete prior output relationships: %w", err)
	}
	return nil
}

func (s *runtimeStore) ListByRun(runID string) ([]*model.RuntimeTask, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Select(dialect.QuoteAll(q, runtimeTaskColumns)...).From(q("runtime_tasks")).
		Where(sq.Eq{q("RunUUID"): runID}).OrderBy(q("TaskUUID")).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build runtime run query: %w", err)
	}
	rows, err := s.tasks.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime tasks for run %q: %w", runID, err)
	}
	defer rows.Close()
	tasks := []*model.RuntimeTask{}
	for rows.Next() {
		task, err := scanRuntimeTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan runtime task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime tasks: %w", err)
	}
	return tasks, nil
}

func (s *runtimeStore) FinalizeRun(runID string, state model.RuntimeState, finishedAt int64) error {
	if !state.IsValid() {
		return fmt.Errorf("cannot finalize runtime run with invalid state %q", state)
	}
	runs := NewRunStore(s.tasks.db, s.tasks.time, s.tasks.dbDialect)
	runs.lease = s.tasks.lease
	run, err := runs.GetRun(runID, false)
	if err != nil {
		return err
	}
	current := run.State.ToV2()
	state = state.ToV2()
	if (current == model.RuntimeStateCancelling || current == model.RuntimeStateCanceled) &&
		state != model.RuntimeStateCancelling && state != model.RuntimeStateCanceled {
		return ErrRuntimeRunCanceled
	}
	run.State, run.Conditions, run.FinishedAtInSec = state, string(state.ToExecutionPhase()), finishedAt
	// Reuse the run store's state/history merge and transactional lifecycle CAS.
	// A cancellation accepted after GetRun must win against this finalization.
	updated, err := runs.UpdateRunFromWorkflow(run, current, run.WorkflowRuntimeManifest, run.PipelineRuntimeManifest)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("runtime run lifecycle changed before finalization; reload the run")
	}
	return nil
}
