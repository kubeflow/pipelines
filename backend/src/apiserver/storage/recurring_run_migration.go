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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// RecurringRunMigrationCandidate identifies a job requiring operator review and recreation.
// Execution inputs are deliberately excluded from the startup inventory.
type RecurringRunMigrationCandidate struct {
	ID        string
	Namespace string
	Name      string
	Enabled   bool
}

// ListJobsWithoutRecurringRunState returns an ID-ordered page, including disabled jobs.
func (s *JobStore) ListJobsWithoutRecurringRunState(afterID string, limit uint64) ([]RecurringRunMigrationCandidate, error) {
	if limit == 0 || limit > 1000 {
		return nil, util.NewInvalidInputError("Recurring-run migration inventory page size must be between 1 and 1000")
	}
	q := s.dbDialect.QuoteIdentifier
	job := func(column string) string { return q("jobs") + "." + q(column) }
	state := func(column string) string { return q("recurring_run_states") + "." + q(column) }
	query, args, err := s.dbDialect.QueryBuilder().
		Select(job("UUID"), job("Namespace"), job("Name"), job("Enabled")).
		From(q("jobs")).
		LeftJoin(fmt.Sprintf("%s ON %s = %s", q("recurring_run_states"), state("JobUUID"), job("UUID"))).
		Where(sq.Eq{state("JobUUID"): nil}).
		Where(sq.Gt{job("UUID"): afterID}).
		OrderBy(job("UUID") + " ASC").Limit(limit).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build recurring-run migration inventory")
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to query recurring-run migration inventory")
	}
	defer rows.Close()
	var candidates []RecurringRunMigrationCandidate
	for rows.Next() {
		var candidate RecurringRunMigrationCandidate
		if err := rows.Scan(&candidate.ID, &candidate.Namespace, &candidate.Name, &candidate.Enabled); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to read recurring-run migration inventory")
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to finish recurring-run migration inventory")
	}
	return candidates, nil
}
