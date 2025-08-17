// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package storage provides persistence layer for KFP API server, including
// SQL-backed implementations of stores and utility filters.
package storage

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Quoter represents a dialect-aware identifier quoting function (e.g., s.dialect.QuoteIdentifier).
type Quoter = func(string) string

// selectWithQuotedColumns builds a SELECT ... FROM ... using the provided dialect-aware
// builder and quoter. It maps all column identifiers through q() and quotes the table name.
// When selectCount is true, it selects count(*) instead of the provided columns.
func selectWithQuotedColumns(
	qb sq.StatementBuilderType,
	q Quoter,
	table string,
	columns []string,
	selectCount bool,
) sq.SelectBuilder {
	if selectCount {
		return qb.Select("count(*)").From(q(table))
	}
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = q(c)
	}
	return qb.Select(quotedCols...).From(q(table))
}

// FilterByExperiment builds a SELECT for the given table/columns, filtered by ExperimentUUID.
// The query uses the provided dialect-aware builder (qb) and quoter (q) so that identifiers
// and placeholders are correct for the active database.
func FilterByExperiment(
	qb sq.StatementBuilderType,
	q Quoter,
	table string,
	columns []string,
	selectCount bool,
	experimentID string,
) (sq.SelectBuilder, error) {
	sel := selectWithQuotedColumns(qb, q, table, columns, selectCount)
	sel = sel.Where(sq.Eq{q("ExperimentUUID"): experimentID})
	return sel, nil
}

// FilterByNamespace builds a SELECT for the given table/columns, filtered by Namespace.
// It relies on the injected builder (qb) and quoter (q) for dialect-correct SQL.
func FilterByNamespace(
	qb sq.StatementBuilderType,
	q Quoter,
	table string,
	columns []string,
	selectCount bool,
	namespace string,
) (sq.SelectBuilder, error) {
	sel := selectWithQuotedColumns(qb, q, table, columns, selectCount)
	sel = sel.Where(sq.Eq{q("Namespace"): namespace})
	return sel, nil
}

// FilterByResourceReference builds a SELECT filtered by a resource reference relation.
// For Jobs/Runs/etc, this constrains rows whose UUID appears in resource_references matching
// the provided FilterContext. It uses an `IN (subquery)` pattern which works reliably across
// MySQL/PostgreSQL/SQLite.
func FilterByResourceReference(
	qb sq.StatementBuilderType,
	q Quoter,
	table string,
	columns []string,
	resourceType model.ResourceType,
	selectCount bool,
	filterContext *model.FilterContext,
) (sq.SelectBuilder, error) {
	sel := selectWithQuotedColumns(qb, q, table, columns, selectCount)

	if filterContext == nil || filterContext.ReferenceKey == nil {
		return sel, nil
	}
	// Keep existing behavior: in multi-user mode, allow filtering even when ReferenceKey.ID is empty.
	if filterContext.ID == "" && !common.IsMultiUserMode() {
		return sel, nil
	}

	// Subquery: SELECT rf.ResourceUUID FROM resource_references AS rf WHERE ...
	sub, args, err := qb.
		Select(q("ResourceUUID")).
		From(q("resource_references") + " AS " + q("rf")).
		Where(sq.And{
			sq.Eq{q("rf") + "." + q("ResourceType"): resourceType},
			sq.Eq{q("rf") + "." + q("ReferenceUUID"): filterContext.ID},
			sq.Eq{q("rf") + "." + q("ReferenceType"): filterContext.Type},
		}).
		ToSql()
	if err != nil {
		return sel, util.NewInternalServerError(err, "Failed to create subquery to filter by resource reference: %v", err.Error())
	}

	// WHERE UUID IN (<subquery>) — quote UUID to preserve exact case on Postgres.
	sel = sel.Where(sq.Expr(fmt.Sprintf("%s IN (%s)", q("UUID"), sub), args...))
	return sel, nil
}
