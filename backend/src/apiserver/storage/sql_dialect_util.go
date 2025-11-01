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
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mattn/go-sqlite3"

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
			return errors.Is(se.Code, sqlite3.ErrConstraint)
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

// quoteAll applies the dialect's QuoteIdentifier (q) to each column name and returns a new slice.
// Use this when passing columns into squirrel.Select(...), so each identifier is properly quoted
// for the current SQL dialect (e.g., Postgres requires double quotes to preserve case).
func quoteAll(q func(string) string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = q(c)
	}
	return out
}

// qualifyIdentifier quotes identifiers correctly when they are qualified with dots,
// e.g. "experiments.Name" -> `"experiments"."Name"` (or the dialect's quote style).
// If there is no dot, it simply quotes the key.
func qualifyIdentifier(q func(string) string, key string) string {
	if q == nil {
		return key
	}
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		for i := range parts {
			parts[i] = q(parts[i])
		}
		return strings.Join(parts, ".")
	}
	return q(key)
}
