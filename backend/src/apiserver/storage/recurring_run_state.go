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
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type recurringRunStateQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// GetRecurringRunState fetches scheduling progress that cannot be changed by CR reports.
func (s *JobStore) GetRecurringRunState(jobID string) (*model.RecurringRunState, error) {
	return s.getRecurringRunState(s.db, jobID, false)
}

func (s *JobStore) getRecurringRunState(db recurringRunStateQueryer, jobID string, lock bool) (*model.RecurringRunState, error) {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().
		Select(q("JobUUID"), q("RequestKey"), q("PipelineVersionID"), q("LastRunIndex"), q("LastScheduledAtInSec"), q("LastCreatedAtInSec"), q("Pending")).
		From(q("recurring_run_states")).Where(sq.Eq{q("JobUUID"): jobID}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build scheduling-state query for recurring run %s", jobID)
	}
	if lock {
		query = s.dbDialect.SelectForUpdate(query)
	}
	state := &model.RecurringRunState{}
	err = db.QueryRow(query, args...).Scan(&state.JobUUID, &state.RequestKey, &state.PipelineVersionID, &state.LastRunIndex,
		&state.LastScheduledAtInSec, &state.LastCreatedAtInSec, &state.Pending)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, util.NewFailedPreconditionError(err,
			"Recurring run %s has no trusted scheduling state; recreate it through the KFP API", jobID)
	}
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to read scheduling state for recurring run %s", jobID)
	}
	return state, nil
}

// ClaimRecurringRun serializes reservations with enable/disable and other claims.
// The caller computes scheduledAt from the stored schedule and expectedIndex.
func (s *JobStore) ClaimRecurringRun(jobID, requestKey string, expectedIndex, scheduledAt, createdAt int64, pipelineVersionID string) (*model.RecurringRunState, error) {
	if requestKey == "" {
		return nil, util.NewInvalidInputError("Provide an idempotency key when claiming a recurring run")
	}
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start a recurring-run claim transaction")
	}
	defer tx.Rollback()

	// Lock the job before any consistent reads so the active-run count observes
	// runs persisted by the preceding claim, including under MySQL REPEATABLE READ.
	query, args, err := qb.Select(q("Enabled"), q("MaxConcurrency")).
		From(q("jobs")).Where(sq.Eq{q("UUID"): jobID}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build recurring-run claim lock")
	}
	var enabled bool
	var maxConcurrency int64
	if err = tx.QueryRow(s.dbDialect.SelectForUpdate(query), args...).Scan(&enabled, &maxConcurrency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.NewResourceNotFoundError("Recurring run", jobID)
		}
		return nil, util.NewInternalServerError(err, "Failed to lock recurring run %s", jobID)
	}
	if !enabled {
		return nil, util.NewFailedPreconditionError(errors.New("recurring run is disabled"),
			"Enable recurring run %s through the KFP API before triggering runs", jobID)
	}
	state, err := s.getRecurringRunState(tx, jobID, true)
	if err != nil {
		return nil, err
	}
	if state.Pending {
		if state.RequestKey != requestKey {
			return nil, util.NewFailedPreconditionError(errors.New("another tick is pending"),
				"Retry the pending execution of recurring run %s before claiming another tick", jobID)
		}
		// Preserve both timestamps and the index when a request is retried.
		if err = tx.Commit(); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to resume recurring-run claim %s", jobID)
		}
		return state, nil
	}
	if state.RequestKey == requestKey {
		return nil, util.NewFailedPreconditionError(errors.New("recurring-run request was already completed"),
			"Recurring run %s already completed request %s; do not recreate a deleted execution", jobID, requestKey)
	}
	if expectedIndex < 0 || state.LastRunIndex != expectedIndex || state.LastRunIndex == math.MaxInt64 {
		return nil, util.NewFailedPreconditionError(errors.New("scheduling progress changed"),
			"Reload scheduling state for recurring run %s and retry", jobID)
	}
	if state.LastRunIndex > 0 && scheduledAt <= state.LastScheduledAtInSec {
		return nil, util.NewFailedPreconditionError(errors.New("scheduled time does not advance"),
			"Use the next scheduled time for recurring run %s", jobID)
	}

	// Match the controller's existing [1, 10] concurrency limits. Legacy rows
	// without State derive their lifecycle from Conditions.
	maxConcurrency = min(int64(10), max(int64(1), maxConcurrency))
	effectiveState := fmt.Sprintf("COALESCE(NULLIF(%s, ''), %s, '')", q("State"), q("Conditions"))
	query, args, err = qb.Select("COUNT(*)").From(q("run_details")).
		Where(sq.Eq{q("JobUUID"): jobID}).
		Where(sq.NotEq{effectiveState: terminalRunStateStrings}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build active-run count for recurring run %s", jobID)
	}
	var activeRuns int64
	if err = tx.QueryRow(query, args...).Scan(&activeRuns); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to count active runs for recurring run %s", jobID)
	}
	if activeRuns >= maxConcurrency {
		return nil, util.NewFailedPreconditionError(errors.New("recurring run reached maximum concurrency"),
			"Wait for an active execution of recurring run %s to finish before retrying", jobID)
	}
	state.LastRunIndex++
	state.RequestKey = requestKey
	state.PipelineVersionID = pipelineVersionID
	state.LastScheduledAtInSec = scheduledAt
	state.LastCreatedAtInSec = createdAt
	state.Pending = true
	query, args, err = qb.Update(q("recurring_run_states")).SetMap(sq.Eq{
		q("RequestKey"):        requestKey,
		q("PipelineVersionID"): pipelineVersionID,
		q("LastRunIndex"):      state.LastRunIndex, q("LastScheduledAtInSec"): scheduledAt,
		q("LastCreatedAtInSec"): createdAt, q("Pending"): true,
	}).Where(sq.Eq{q("JobUUID"): jobID}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build recurring-run claim %s", jobID)
	}
	if _, err = tx.Exec(query, args...); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to save recurring-run claim %s", jobID)
	}
	if err = tx.Commit(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to commit recurring-run claim %s", jobID)
	}
	return state, nil
}

// completeRecurringRunWithInsert makes the completed-tick tombstone atomic with
// run persistence, so deleting a persisted run cannot reopen its pending claim.
func (s *RunStore) completeRecurringRunWithInsert(tx *sql.Tx, run *model.Run) error {
	if run.RecurringRunId == "" {
		return nil
	}
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	query, args, err := qb.Select(q("RequestKey"), q("LastRunIndex"), q("Pending"),
		q("PipelineVersionID"), q("LastCreatedAtInSec"), q("LastScheduledAtInSec")).
		From(q("recurring_run_states")).Where(sq.Eq{q("JobUUID"): run.RecurringRunId}).ToSql()
	if err != nil {
		return err
	}
	var requestKey string
	var index int64
	var pending bool
	var pipelineVersionID string
	var createdAt, scheduledAt int64
	err = tx.QueryRow(s.dbDialect.SelectForUpdate(query), args...).Scan(&requestKey, &index, &pending,
		&pipelineVersionID, &createdAt, &scheduledAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Legacy and single-user recurring runs do not require a scheduling claim.
		return nil
	}
	if err != nil {
		return err
	}
	if !pending || run.UUID != util.NewDeterministicUUID(run.RecurringRunId+"/tick/"+strconv.FormatInt(index, 10)) {
		return nil
	}
	// The persistence agent reconstructs DisplayName from the workflow name
	// when recovering a workflow whose creating API request did not persist.
	workflowName := "run-" + util.NewDeterministicUUID(run.UUID)
	reportedExecution := run.DisplayName == workflowName && run.K8SName == workflowName
	if run.DisplayName != requestKey && !reportedExecution {
		return nil
	}
	if reportedExecution {
		query, args, err = qb.Update(q("run_details")).SetMap(sq.Eq{
			q("DisplayName"): requestKey, q("PipelineVersionId"): pipelineVersionID,
			q("CreatedAtInSec"): createdAt, q("ScheduledAtInSec"): scheduledAt,
		}).
			Where(sq.Eq{q("UUID"): run.UUID}).ToSql()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(query, args...); err != nil {
			return err
		}
		run.DisplayName = requestKey
		run.PipelineVersionId = pipelineVersionID
		run.CreatedAtInSec = createdAt
		run.ScheduledAtInSec = scheduledAt
	}
	query, args, err = qb.Update(q("recurring_run_states")).Set(q("Pending"), false).
		Where(sq.Eq{q("JobUUID"): run.RecurringRunId, q("LastRunIndex"): index}).ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, args...)
	return err
}
