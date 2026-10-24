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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func reusableRuntimeTask(state model.TaskStatus) bool {
	switch apiv2.PipelineTask_TaskState(state) {
	case apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_CACHED, apiv2.PipelineTask_SKIPPED:
		return true
	default:
		return false
	}
}

// lockRetryRuns follows the runtime's run-before-task lock order, including for
// unscoped API retries. Run ownership and cancellation cannot change mid-reset.
func (s *runtimeStore) lockRetryRuns(tx sqlTransaction, taskIDs []string) (map[string]bool, error) {
	q := s.tasks.dbDialect.QuoteIdentifier
	query, args, err := s.tasks.dbDialect.QueryBuilder().Select(q("RunUUID")).Distinct().
		From(q("tasks")).Where(sq.Eq{q("UUID"): taskIDs}).OrderBy(q("RunUUID")).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build retry run ownership query: %w", err)
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("read retry run ownership: %w", err)
	}
	defer rows.Close()
	var runIDs []string
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			return nil, err
		}
		runIDs = append(runIDs, runID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	managed := make(map[string]bool, len(runIDs))
	for _, runID := range runIDs {
		if err := s.lockRunID(tx, runID, false); err != nil {
			return nil, err
		}
		query, args, err := s.tasks.dbDialect.QueryBuilder().Select(q("ExecutorType")).
			From(q("run_details")).Where(sq.Eq{q("UUID"): runID}).ToSql()
		if err != nil {
			return nil, err
		}
		var executor sql.NullString
		if err := tx.QueryRow(query, args...).Scan(&executor); err != nil {
			return nil, fmt.Errorf("read retry run executor: %w", err)
		}
		managed[runID] = executor.Valid && executor.String != ""
	}
	return managed, nil
}
