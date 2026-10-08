// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// recurringRunAssociation matches GetRun's legacy fallback without allowing a
// stale reference to override an explicit JobUUID. EXISTS cannot multiply runs.
func recurringRunAssociation(q dialect.QuoteFunction, jobID string) sq.Sqlizer {
	run := dialect.QualifiedColumn(q, "run_details")
	ref := dialect.QualifiedColumn(q, "resource_references")
	references := sq.Select("1").From(q("resource_references")).
		Where(sq.Expr(ref("ResourceUUID") + " = " + run("UUID"))).
		Where(sq.Eq{ref("ResourceType"): model.RunResourceType,
			ref("ReferenceType"): model.JobResourceType, ref("ReferenceUUID"): jobID})
	return sq.Or{
		sq.Eq{run("JobUUID"): jobID},
		sq.And{
			sq.Or{sq.Eq{run("JobUUID"): ""}, sq.Eq{run("JobUUID"): nil}},
			sq.Expr("EXISTS (?)", references),
		},
	}
}

// ListRunIDsForRecurringRun inventories only runs associated with one schedule,
// including legacy references. Call GetRun to resolve each row's other fields.
func (s *RunStore) ListRunIDsForRecurringRun(jobID string) ([]string, error) {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.FinalizeSelect(sq.Select(q("UUID")).From(q("run_details")).
		Where(recurringRunAssociation(q, jobID)).OrderBy(q("UUID")))
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build run inventory for recurring run %s", jobID)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to inventory runs for recurring run %s", jobID)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to read run inventory for recurring run %s", jobID)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to finish run inventory for recurring run %s", jobID)
	}
	return ids, nil
}
