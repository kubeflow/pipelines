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
	"github.com/go-sql-driver/mysql"
)

type mysqlDialect struct{}

func (mysqlDialect) Name() string { return "mysql" }

func (mysqlDialect) QuoteIdentifier(id string) string {
	escaped := strings.ReplaceAll(id, "`", "``")
	return fmt.Sprintf("`%s`", escaped)
}

func (mysqlDialect) LengthFunc() string { return "CHAR_LENGTH" }

func (mysqlDialect) QueryBuilder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Question)
}

func (mysqlDialect) IsDuplicateDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1007
	}
	return false
}

func (mysqlDialect) StringCollation() string { return "" }

func (d mysqlDialect) ConcatAgg(distinct bool, expr, sep string) string {
	sep = escapeSQLString(sep)
	dist := ""
	if distinct {
		dist = "DISTINCT "
	}
	return fmt.Sprintf("GROUP_CONCAT(%s%s SEPARATOR '%s')", dist, expr, sep)
}

func (d mysqlDialect) ConcatExprs(exprs []string, sep string) string {
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
	return fmt.Sprintf("CONCAT(%s)", strings.Join(parts, ", "))
}

func (mysqlDialect) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}

func (d mysqlDialect) Upsert(table string, keyCols []string, overwrite bool, updateCols []string) sq.InsertBuilder {
	q := d.QuoteIdentifier
	ib := d.QueryBuilder().Insert(q(table))
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

func (mysqlDialect) FinalizeSelect(builder sq.SelectBuilder) (string, []interface{}, error) {
	return builder.ToSql()
}

func (mysqlDialect) SelectForUpdate(query string) string {
	return query + " FOR UPDATE"
}
