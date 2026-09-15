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
	"github.com/mattn/go-sqlite3"
)

type sqliteDialect struct{}

func (sqliteDialect) Name() string { return "sqlite" }

func (sqliteDialect) QuoteIdentifier(id string) string {
	escaped := strings.ReplaceAll(id, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}

func (sqliteDialect) LengthFunc() string { return "LENGTH" }

func (sqliteDialect) QueryBuilder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Question)
}

func (sqliteDialect) IsDuplicateDatabaseError(err error) bool {
	return false
}

func (sqliteDialect) StringCollation() string { return "" }

func (sqliteDialect) ConcatAgg(distinct bool, expr, sep string) string {
	sep = escapeSQLString(sep)
	// SQLite ignores DISTINCT in GROUP_CONCAT
	return "GROUP_CONCAT(" + expr + ", '" + sep + "')"
}

func (d sqliteDialect) ConcatExprs(exprs []string, sep string) string {
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

func (sqliteDialect) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var se sqlite3.Error
	if errors.As(err, &se) {
		return errors.Is(se.Code, sqlite3.ErrConstraint)
	}
	return false
}

func (d sqliteDialect) Upsert(table string, keyCols []string, overwrite bool, updateCols []string) sq.InsertBuilder {
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

func (sqliteDialect) FinalizeSelect(builder sq.SelectBuilder) (string, []interface{}, error) {
	return builder.ToSql()
}

// SelectForUpdate is a no-op: SQLite has no concurrent-writer story (tests only).
func (sqliteDialect) SelectForUpdate(query string) string {
	return query
}
