// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func recurringRunAdoptionQuery(q dialect.QuoteFunction, jobID string) sq.SelectBuilder {
	run := dialect.QualifiedColumn(q, "run_details")
	ref := dialect.QualifiedColumn(q, "resource_references")
	query := sq.Select().From(q("run_details"))
	for _, column := range []string{"UUID", "DisplayName", "Name", "ImportedFrom", "CreatedAtInSec", "ScheduledAtInSec"} {
		query = query.Column(run(column))
	}
	for _, column := range []string{"State", "Conditions", "WorkflowSpecManifest", "WorkflowRuntimeManifest"} {
		query = query.Column("COALESCE(" + run(column) + ", '')")
	}
	// Match GetRun: native fields take precedence over legacy references. The
	// (ResourceUUID, ResourceType, ReferenceType) primary key bounds each fallback.
	for _, field := range []struct {
		column    string
		reference model.ResourceType
	}{
		{"Namespace", model.NamespaceResourceType}, {"PipelineVersionId", model.PipelineVersionResourceType},
	} {
		fallback := sq.Select(ref("ReferenceUUID")).From(q("resource_references")).
			Where(sq.Expr(ref("ResourceUUID") + " = " + run("UUID"))).
			Where(sq.Eq{ref("ResourceType"): model.RunResourceType, ref("ReferenceType"): field.reference})
		query = query.Column(sq.Expr("COALESCE(NULLIF("+run(field.column)+", ''), (?), '')", fallback))
	}
	return query.Where(recurringRunAssociation(q, jobID)).OrderBy(run("UUID"))
}

// ListRunsForRecurringRunAdoption reads only scheduling evidence in one cancellable
// query. It deliberately includes records whose runtime manifest has not yet been
// reported: unlike GetRun's API response, their persisted spec is valid evidence.
// Metrics, tasks and unrelated pipeline payloads are not needed by the planner.
func (s *RunStore) ListRunsForRecurringRunAdoption(ctx context.Context, jobID string) ([]*model.Run, error) {
	fail := func(err error) ([]*model.Run, error) {
		return nil, util.NewInternalServerError(err, "Failed to inventory runs for recurring run %s; retry adoption after restoring database access", jobID)
	}
	query, args, err := s.dbDialect.FinalizeSelect(recurringRunAdoptionQuery(s.dbDialect.QuoteIdentifier, jobID))
	if err != nil {
		return fail(err)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fail(err)
	}
	defer rows.Close()
	var runs []*model.Run
	for rows.Next() {
		run := &model.Run{RecurringRunId: jobID}
		if err := rows.Scan(&run.UUID, &run.DisplayName, &run.K8SName, &run.ImportedFrom,
			&run.CreatedAtInSec, &run.ScheduledAtInSec, &run.State, &run.Conditions,
			&run.WorkflowSpecManifest, &run.WorkflowRuntimeManifest, &run.Namespace, &run.PipelineVersionId); err != nil {
			return fail(fmt.Errorf("read adoption inventory: %w", err))
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return fail(err)
	}
	return runs, nil
}
