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

package dialect

import (
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgxDialect struct{}

func (pgxDialect) Name() string { return "pgx" }

func (pgxDialect) QuoteIdentifier(id string) string {
	escaped := strings.ReplaceAll(id, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}

func (pgxDialect) LengthFunc() string { return "CHAR_LENGTH" }

func (pgxDialect) QueryBuilder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

func (pgxDialect) ExistDatabaseErrHint() string { return "already exists" }

func (pgxDialect) StringCollation() string { return "" }

func (d pgxDialect) ConcatAgg(distinct bool, expr, sep string) string {
	sep = escapeSQLString(sep)
	dist := ""
	if distinct {
		dist = "DISTINCT "
	}
	return fmt.Sprintf("string_agg(%s%s, '%s')", dist, expr, sep)
}

func (d pgxDialect) ConcatExprs(exprs []string, sep string) string {
	n := len(exprs)
	if n == 0 {
		return "''"
	}
	if n == 1 {
		return exprs[0]
	}
	var lit string
	if sep != "" {
		lit = fmt.Sprintf("'%s'", escapeSQLString(sep))
	}
	parts := make([]string, 0, n*2-1)
	for i, e := range exprs {
		if i > 0 && lit != "" {
			parts = append(parts, lit)
		}
		parts = append(parts, e)
	}
	return strings.Join(parts, " || ")
}

func (pgxDialect) IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code == pgerrcode.UniqueViolation
	}
	return false
}

func (d pgxDialect) Upsert(table string, keyCols []string, overwrite bool, updateCols []string) sq.InsertBuilder {
	q := d.QuoteIdentifier
	ib := d.QueryBuilder().Insert(q(table))
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
}

func (pgxDialect) FinalizeSelect(builder sq.SelectBuilder) (string, []interface{}, error) {
	return builder.PlaceholderFormat(sq.Dollar).ToSql()
}

func joinQuoted(q func(string) string, cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = q(c)
	}
	return strings.Join(out, ", ")
}
