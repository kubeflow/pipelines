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

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// DeletedScheduledWorkflowGeneration permanently fences reports for a deleted
// resource. Kubernetes never reuses its UID, even if its name is recreated.
const DeletedScheduledWorkflowGeneration int64 = -1

// ChangeJobModeAtGeneration persists the generation returned by the Kubernetes
// patch together with the requested mode. Even a mode that already matches must
// advance the fence, and older patch completions must not undo newer changes.
func (s *JobStore) ChangeJobModeAtGeneration(id string, enabled bool, generation int64) error {
	if generation < DeletedScheduledWorkflowGeneration || (generation == DeletedScheduledWorkflowGeneration && enabled) {
		return util.NewInvalidInputError("Invalid ScheduledWorkflow generation %d", generation)
	}
	q := s.dbDialect.QuoteIdentifier
	update := s.dbDialect.QueryBuilder().Update(q("jobs")).SetMap(sq.Eq{
		q("Enabled"):        enabled,
		q("UpdatedAtInSec"): s.time.Now().Unix(),
	}).Where(sq.Eq{q("Enabled"): !enabled})
	return s.updateJobAtGeneration(id, generation, update)
}

// updateJobAtGeneration serializes reports and mode changes across API-server
// replicas. Lock the job first, matching scheduling claims and deletion, so lazy
// fence creation is serialized and cannot leave an orphan after job deletion.
// Generation zero preserves legacy reports only until a real generation arrives.
func (s *JobStore) updateJobAtGeneration(id string, generation int64, update sq.UpdateBuilder) error {
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to start recurring-run report transaction")
	}
	defer tx.Rollback()
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	query, args, err := qb.Select(q("UUID")).From(q("jobs")).Where(sq.Eq{q("UUID"): id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to build recurring-run report lock")
	}
	var jobID string
	if err = tx.QueryRow(s.dbDialect.SelectForUpdate(query), args...).Scan(&jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return util.NewInvalidInputError("There is no job corresponding to this scheduled workflow: %s", id)
		}
		return util.NewInternalServerError(err, "Failed to lock recurring run %s", id)
	}
	query, args, err = qb.Select(q("Generation")).From(q("recurring_run_report_states")).Where(sq.Eq{q("JobUUID"): id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to build recurring-run report-state query")
	}
	var current int64
	// This is the first consistent read after the jobs lock, so MySQL's
	// REPEATABLE READ snapshot includes the preceding writer's fence. A
	// locking read here would gap-lock absent state rows across different jobs.
	err = tx.QueryRow(query, args...).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return util.NewInternalServerError(err, "Failed to read recurring-run report state %s", id)
	}
	if current == DeletedScheduledWorkflowGeneration || (generation != DeletedScheduledWorkflowGeneration && generation < current) {
		return nil
	}
	query, args, err = s.dbDialect.Upsert("recurring_run_report_states", []string{"JobUUID"}, true, []string{"Generation"}).
		Columns(q("JobUUID"), q("Generation")).Values(id, generation).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to build recurring-run report-state update")
	}
	if _, err = tx.Exec(query, args...); err != nil {
		return util.NewInternalServerError(err, "Failed to save recurring-run report state %s", id)
	}
	query, args, err = update.Where(sq.Eq{q("UUID"): id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to build recurring-run update")
	}
	if _, err = tx.Exec(query, args...); err != nil {
		return util.NewInternalServerError(err, "Failed to update recurring run %s", id)
	}
	if err = tx.Commit(); err != nil {
		return util.NewInternalServerError(err, "Failed to commit recurring-run report %s", id)
	}
	return nil
}
