// Copyright 2025 The Kubeflow Authors
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
	"context"
	"database/sql"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
)

// ---- Duplicate error detection ------------------------------------------------

// isDuplicateError returns true if err indicates a unique/duplicate violation
// for the given dialect.
func isDuplicateError(d dialect.DBDialect, err error) bool {
	if err == nil {
		return false
	}
	switch d.Name() { // use exported Name(); do NOT rely on unexported fields
	case "mysql":
		var me *mysql.MySQLError
		if errors.As(err, &me) {
			// 1062 = ER_DUP_ENTRY
			return me.Number == 1062
		}
		return false
	case "pgx":
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			// Unique violation
			return pe.Code == pgerrcode.UniqueViolation
		}
		return false
	default:
		// Optional: for unit tests that still use sqlite.
		var se sqlite3.Error
		if errors.As(err, &se) {
			return se.Code == sqlite3.ErrConstraint
		}
		return false
	}
}

// ---- Upsert (INSERT ... ON DUPLICATE KEY / ON CONFLICT) ----------------------

// insertUpsert starts an INSERT builder and appends the dialect-specific upsert
// clause. Callers should continue chaining .Columns(...).Values(...).
func insertUpsert(d dialect.DBDialect, table string, keyCols []string, overwrite bool, updateCols []string) sq.InsertBuilder {
	q := d.QuoteIdentifier
	ib := d.QueryBuilder().Insert(q(table))

	switch d.Name() {
	case "pgx", "sqlite":
		sets := make([]string, 0, len(updateCols))
		for _, c := range updateCols {
			if overwrite {
				sets = append(sets, q(c)+"=excluded."+q(c))
			} else {
				sets = append(sets, q(c)+"="+q(c))
			}
		}
		suffix := "ON CONFLICT (" + joinQuoted(q, keyCols) + ") DO UPDATE SET " + strings.Join(sets, ", ")
		return ib.Suffix(suffix)
	default: // mysql
		sets := make([]string, 0, len(updateCols))
		for _, c := range updateCols {
			if overwrite {
				sets = append(sets, q(c)+"=VALUES("+q(c)+")")
			} else {
				sets = append(sets, q(c)+"="+q(c))
			}
		}
		suffix := "ON DUPLICATE KEY UPDATE " + strings.Join(sets, ", ")
		return ib.Suffix(suffix)
	}
}

func joinQuoted(q func(string) string, cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = q(c)
	}
	return strings.Join(out, ", ")
}

// ---- UPDATE ... JOIN / FROM compatibility ------------------------------------

// updateFromPG normalizes "update with join" to PostgreSQL's
//
//	UPDATE <target> SET ... FROM (<subquery>) AS j WHERE <joinOn>
//
// NOTE: squirrel will not merge args from another builder automatically, so we
// pass argsFrom via Suffix(..., argsFrom...).
//
// Usage:
//
//	sel := qb.Select(q("id"), q("new_val")).From(q("src")).Where(...)
//	ub  := qb.Update(q("target")).Set(q("col"), sq.Expr("j.new_val"))
//	ub  = updateFromPG(dialect, ub, sel, "target.id = j.id")
//	sql, args, _ := ub.ToSql()
func updateFromPG(d dialect.DBDialect, ub sq.UpdateBuilder, from sq.SelectBuilder, joinOn string) sq.UpdateBuilder {
	sqlFrom, argsFrom, _ := from.ToSql()
	return ub.Suffix("FROM ("+sqlFrom+") AS j WHERE "+joinOn, argsFrom...)
}

// updateByIDsTwoPhase is a conservative MySQL-compatible approach:
//  1. select matching IDs via a subquery
//  2. UPDATE ... WHERE id IN (...)
//
// exec must be *sql.DB or *sql.Tx (anything implementing ExecContext/QueryContext).
func updateByIDsTwoPhase(
	ctx context.Context,
	exec interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	},
	d *dialect.DBDialect,
	selectIDs sq.SelectBuilder,
	updateUB sq.UpdateBuilder,
	idCol string,
) error {
	sqlIDs, argsIDs, _ := selectIDs.ToSql()
	rows, err := exec.QueryContext(ctx, sqlIDs, argsIDs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := make([]any, 0, 128)
	for rows.Next() {
		var id any
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	q := d.QuoteIdentifier
	updateUB = updateUB.Where(sq.Eq{q(idCol): ids})
	sqlUpd, argsUpd, _ := updateUB.ToSql()
	_, err = exec.ExecContext(ctx, sqlUpd, argsUpd...)
	return err
}
