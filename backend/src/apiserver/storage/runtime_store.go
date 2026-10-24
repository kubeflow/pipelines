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

// ErrRuntimeTaskConflict means the attempt was replaced or finalized. Callers
// must reload rather than recreate coordination state from a stale completion.
var ErrRuntimeTaskConflict = errors.New("runtime task attempt no longer current; reload durable task state")

// ErrRuntimeRunCanceled prevents completed work from resurrecting a canceled run.
var ErrRuntimeRunCanceled = errors.New("runtime run is canceling or canceled; only cancellation cleanup may continue")

// RuntimeStoreInterface persists coordination state and attempt-fenced results.
// Update compares NULL when expectedAttempt is empty. Checkpoint uses task's
// AttemptUUID as its fence and never changes that identity. BeginRetry is the
// only operation that replaces a non-null attempt or increments RetryCount.
type RuntimeStoreInterface interface {
	Get(taskID string) (*model.RuntimeTask, error)
	Create(task *model.RuntimeTask) error
	Update(task *model.RuntimeTask, expectedAttempt string) error
	Checkpoint(task *model.RuntimeTask, taskUpdate *model.Task) error
	// CheckpointCancellation is framework-only: stopped-workload log capture and
	// cleanup progress may advance while cancellation blocks launcher reports.
	CheckpointCancellation(task *model.RuntimeTask, taskUpdate *model.Task) error
	// CheckpointArtifacts atomically stores provisional results and output
	// relationships with publication or failure-log phases. IDs must be attempt-scoped and
	// deterministic; replaying identical metadata is safe, collisions are errors.
	CheckpointArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error
	Finalize(taskID, attemptID string, taskUpdate *model.Task) error
	BeginRetry(taskID, previousAttemptID string, next *model.RuntimeTask) error
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
	"TaskUUID", "RunUUID", "AttemptUUID", "State", "RetryCount", "PublicationRetryCount",
	"NextEligibleAt", "ExecutionRequest", "ExecutionReference",
}

func scanRuntimeTask(row interface{ Scan(...any) error }) (*model.RuntimeTask, error) {
	task := &model.RuntimeTask{}
	err := row.Scan(&task.TaskUUID, &task.RunUUID, &task.AttemptUUID, &task.State,
		&task.RetryCount, &task.PublicationRetryCount, &task.NextEligibleAt, &task.ExecutionRequest, &task.ExecutionReference)
	return task, err
}

func (s *runtimeStore) Get(taskID string) (*model.RuntimeTask, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Select(dialect.QuoteAll(q, runtimeTaskColumns)...).From(q("runtime_tasks")).
		Where(sq.Eq{q("TaskUUID"): taskID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build runtime task query: %w", err)
	}
	task, err := scanRuntimeTask(s.tasks.db.QueryRow(query, args...))
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
	if task.RetryCount < 0 || task.PublicationRetryCount < 0 {
		return errors.New("runtime retry counts cannot be negative")
	}
	switch task.State {
	case model.RuntimeTaskWaiting, model.RuntimeTaskReady, model.RuntimeTaskSubmitting,
		model.RuntimeTaskRunning, model.RuntimeTaskPublishing, model.RuntimeTaskFailed,
		model.RuntimeTaskBackoff, model.RuntimeTaskCanceling, model.RuntimeTaskFinalizing,
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
		q("PublicationRetryCount"): task.PublicationRetryCount,
		q("ExecutionRequest"):      request, q("ExecutionReference"): reference,
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
	values, err := s.values(task)
	if err != nil {
		return err
	}
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Insert(s.tasks.dbDialect.QuoteIdentifier("runtime_tasks")).SetMap(values).ToSql()
	if err != nil {
		return fmt.Errorf("build runtime task insert: %w", err)
	}
	tx, err := s.tasks.db.Begin()
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
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("create runtime task %q: %w", task.TaskUUID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit runtime task creation: %w", err)
	}
	return nil
}

// withAttempt locks the public task before its coordination row, consistently
// with task deletion and FK checks. No read/merge/write crosses a transaction.
func (s *runtimeStore) withAttempt(taskID, attemptID string, allowCanceled bool, change func(*sql.Tx, *model.RuntimeTask, *model.Task) error) error {
	tx, err := s.tasks.db.Begin()
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
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().
		Select(dialect.QuoteAll(q, runtimeTaskColumns)...).From(q("runtime_tasks")).
		Where(sq.Eq{q("TaskUUID"): taskID}).ToSql()
	if err != nil {
		return fmt.Errorf("build runtime attempt lock: %w", err)
	}
	current, err := scanRuntimeTask(tx.QueryRow(s.tasks.dbDialect.SelectForUpdate(query), args...))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRuntimeTaskConflict
	}
	if err != nil {
		return fmt.Errorf("lock runtime task %q: %w", taskID, err)
	}
	if runtimeAttempt(current) != attemptID || current.RunUUID != public.RunUUID {
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

func (s *runtimeStore) lockRun(tx *sql.Tx, taskID string, allowCanceled bool) error {
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
	query, args, err = qb.Select(q("State"), q("Conditions")).From(q("run_details")).
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

func (s *runtimeStore) replace(tx *sql.Tx, task *model.RuntimeTask) error {
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
	if runtimeCancellationPhase(current.State) && next.State != current.State {
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
	return s.withAttempt(task.TaskUUID, expectedAttempt, task.State == model.RuntimeTaskCanceling, func(tx *sql.Tx, current *model.RuntimeTask, _ *model.Task) error {
		if err := validateRuntimeReplacement(current, task, true); err != nil {
			return err
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
	return s.withAttempt(task.TaskUUID, runtimeAttempt(task), true, func(tx *sql.Tx, current *model.RuntimeTask, public *model.Task) error {
		if current.State == model.RuntimeTaskCancelFinalizing && task.State != model.RuntimeTaskCancelFinalizing {
			return errors.New("cancellation checkpoint cannot return from cleanup to log capture")
		}
		if err := validateRuntimeAttemptFields(current, task, false); err != nil {
			return err
		}
		if err := s.tasks.updateTaskInTransaction(tx, taskUpdate, false); err != nil {
			return err
		}
		if err := s.checkpointArtifacts(tx, current, public, taskUpdate.OutputArtifactsHydrated); err != nil {
			return err
		}
		return s.replace(tx, task)
	})
}

func (s *runtimeStore) CheckpointArtifacts(task *model.RuntimeTask, taskUpdate *model.Task, artifacts []*model.Artifact, links []*model.ArtifactTask) error {
	if task == nil || (task.State != model.RuntimeTaskPublishing && task.State != model.RuntimeTaskFailed) {
		return errors.New("artifact checkpoint requires the runtime PUBLISHING or FAILED phase")
	}
	return s.checkpoint(task, taskUpdate, artifacts, links)
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
	return s.withAttempt(task.TaskUUID, runtimeAttempt(task), false, func(tx *sql.Tx, current *model.RuntimeTask, public *model.Task) error {
		if runtimeCancellationPhase(current.State) {
			return ErrRuntimeRunCanceled
		}
		if err := validateRuntimeReplacement(current, task, false); err != nil {
			return err
		}
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
	})
}

func (s *runtimeStore) Finalize(taskID, attemptID string, taskUpdate *model.Task) error {
	if taskUpdate == nil || !terminalRuntimeTask(taskUpdate.State) {
		return errors.New("runtime finalization requires a terminal public task state")
	}
	allowCanceled := taskUpdate.State == model.TaskStatus(apiv2.PipelineTask_FAILED) || taskUpdate.State == model.TaskStatus(apiv2.PipelineTask_SKIPPED)
	return s.withAttempt(taskID, attemptID, allowCanceled, func(tx *sql.Tx, current *model.RuntimeTask, public *model.Task) error {
		if runtimeCancellationPhase(current.State) && !allowCanceled {
			return ErrRuntimeRunCanceled
		}
		if err := validatePublicRuntimeUpdate(taskID, current.RunUUID, taskUpdate); err != nil {
			return err
		}
		if err := s.tasks.updateTaskInTransaction(tx, taskUpdate, true); err != nil {
			return err
		}
		if err := s.checkpointArtifacts(tx, current, public, taskUpdate.OutputArtifactsHydrated); err != nil {
			return err
		}
		q := s.tasks.dbDialect.QuoteIdentifier
		query, args, err := s.tasks.dbDialect.QueryBuilder().Delete(q("runtime_tasks")).
			Where(sq.Eq{q("TaskUUID"): taskID}).ToSql()
		if err != nil {
			return fmt.Errorf("build runtime finalization: %w", err)
		}
		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("delete finalized runtime task: %w", err)
		}
		return nil
	})
}

func (s *runtimeStore) BeginRetry(taskID, previousAttemptID string, next *model.RuntimeTask) error {
	if err := validateRuntimeTask(next); err != nil {
		return err
	}
	if taskID != next.TaskUUID || previousAttemptID == "" || next.AttemptUUID == nil || *next.AttemptUUID == previousAttemptID {
		return errors.New("retry requires the same task ID and a new nonempty attempt ID")
	}
	var retryCount int
	err := s.withAttempt(taskID, previousAttemptID, false, func(tx *sql.Tx, current *model.RuntimeTask, public *model.Task) error {
		if runtimeCancellationPhase(current.State) {
			return ErrRuntimeRunCanceled
		}
		if current.RunUUID != next.RunUUID {
			return errors.New("retry cannot change the owning run")
		}
		retryCount = current.RetryCount + 1
		replacement := *next
		replacement.RetryCount = retryCount
		replacement.PublicationRetryCount = 0
		if err := s.resetPublicAttempt(tx, public, previousAttemptID); err != nil {
			return err
		}
		return s.replace(tx, &replacement)
	})
	if err == nil {
		next.RetryCount = retryCount
		next.PublicationRetryCount = 0
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

func (s *runtimeStore) resetPublicAttempt(tx *sql.Tx, task *model.Task, previousAttemptID string) error {
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

func (s *runtimeStore) deleteOutputRelationships(tx *sql.Tx, task *model.Task) error {
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
