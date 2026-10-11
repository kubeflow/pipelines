// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

// recurringRunBranches scopes direct and legacy associations to one schedule.
// The branches are disjoint, and the reference primary key permits at most one
// matching (ResourceUUID, ResourceType, ReferenceType) reference per run.
func recurringRunBranches(q dialect.QuoteFunction, jobID string, columns ...string) (sq.SelectBuilder, sq.SelectBuilder) {
	run := dialect.QualifiedColumn(q, "run_details")
	ref := dialect.QualifiedColumn(q, "resource_references")
	direct := sq.Select(columns...).From(q("run_details")).
		Where(sq.Eq{run("JobUUID"): jobID})
	legacy := sq.Select(columns...).From(q("resource_references")).
		Join(q("run_details") + " ON " + run("UUID") + " = " + ref("ResourceUUID")).
		Where(sq.Eq{ref("ResourceType"): model.RunResourceType,
			ref("ReferenceType"): model.JobResourceType, ref("ReferenceUUID"): jobID}).
		// Apply the fallback after the reference lookup, without indexing all
		// empty JobUUIDs. Explicit associations win and cannot be counted twice.
		Where(sq.Expr("COALESCE(" + run("JobUUID") + ", '') = ''"))
	return direct, legacy
}

func recurringRunActiveCount(q dialect.QuoteFunction, jobID string) sq.SelectBuilder {
	run := dialect.QualifiedColumn(q, "run_details")
	effectiveState := fmt.Sprintf("COALESCE(NULLIF(%s, ''), %s, '')", run("State"), run("Conditions"))
	active := sq.NotEq{effectiveState: terminalRunStateStrings}
	direct, legacy := recurringRunBranches(q, jobID, "COUNT(*)")
	return sq.Select().Column(sq.Expr("(?) + (?)", direct.Where(active), legacy.Where(active)))
}

func recurringRunReplayQuery(q dialect.QuoteFunction, jobID, displayName string) sq.SelectBuilder {
	run := dialect.QualifiedColumn(q, "run_details")
	direct, legacy := recurringRunBranches(q, jobID, run("UUID"), run("CreatedAtInSec"))
	name := sq.Eq{run("DisplayName"): displayName}
	candidates := direct.Where(name).SuffixExpr(sq.Expr("UNION ALL ?", legacy.Where(name)))
	// Choose the newest match across both branches, including the UUID tie-break.
	return sq.Select(q("UUID")).FromSelect(candidates, q("candidates")).
		OrderBy(q("CreatedAtInSec")+" DESC", q("UUID")+" DESC").Limit(1)
}
