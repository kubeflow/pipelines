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

// Package dialect provides a minimal, shared SQL dialect configuration
// for apiserver components. It centralizes identifier quoting and
// placeholder styles for different backends, and returns a Squirrel
// StatementBuilder configured for the target database.
package dialect

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// DBDialect holds read-only runtime configuration for a SQL backend.
// All fields are private; callers must use the exported getter methods.
type DBDialect struct {
	name                 string
	quoteIdentifier      func(string) string
	lengthFunc           string
	statementBuilder     sq.StatementBuilderType
	existDatabaseErrHint string
}

// Get constructs a DBDialect for the given backend name.
// Supported names: "mysql", "pgx", "sqlite" (sqlite is for tests).
func NewDBDialect(name string) DBDialect {
	switch name {
	case "mysql":
		return DBDialect{
			name:            "mysql",
			quoteIdentifier: func(id string) string { return "`" + id + "`" },
			lengthFunc:      "CHAR_LENGTH",
			statementBuilder: sq.StatementBuilder.
				PlaceholderFormat(sq.Question),
			existDatabaseErrHint: "database exists",
		}
	case "pgx":
		return DBDialect{
			name:            "pgx",
			quoteIdentifier: func(id string) string { return `"` + id + `"` },
			lengthFunc:      "CHAR_LENGTH",
			statementBuilder: sq.StatementBuilder.
				PlaceholderFormat(sq.Dollar),
			existDatabaseErrHint: "already exists",
		}
	case "sqlite": // only for tests
		return DBDialect{
			name:            "sqlite",
			quoteIdentifier: func(id string) string { return `"` + id + `"` },
			lengthFunc:      "LENGTH",
			statementBuilder: sq.StatementBuilder.
				PlaceholderFormat(sq.Question),
			existDatabaseErrHint: "",
		}
	default:
		panic("unsupported dialect: " + name)
	}
}

// Name returns the backend name (e.g., "mysql", "pgx", "sqlite").
func (d DBDialect) Name() string { return d.name }

// QuoteIdentifier returns the dialect-appropriate quoted identifier for id.
func (d DBDialect) QuoteIdentifier(id string) string { return d.quoteIdentifier(id) }

// LengthFunc returns the SQL length function name for this dialect.
func (d DBDialect) LengthFunc() string { return d.lengthFunc }

// QueryBuilder returns a Squirrel StatementBuilderType configured with the
// correct placeholder format for this dialect.
func (d DBDialect) QueryBuilder() sq.StatementBuilderType { return d.statementBuilder }

// ExistDatabaseErrHint returns a backend-specific substring that may appear
// in errors when attempting to create a database that already exists.
func (d DBDialect) ExistDatabaseErrHint() string { return d.existDatabaseErrHint }

// ConcatAgg returns a dialect-specific SQL expression for concatenating
// string values from multiple rows into a single string, using the given
// separator.
func (d DBDialect) ConcatAgg(distinct bool, expr, sep string) string {
	dist := ""
	if distinct {
		dist = "DISTINCT "
	}
	switch d.name {
	case "mysql":
		// GROUP_CONCAT(expr SEPARATOR ',')
		return "GROUP_CONCAT(" + dist + expr + " SEPARATOR '" + sep + "')"
	case "pgx":
		// string_agg(expr, ',')
		return "string_agg(" + dist + expr + ", '" + sep + "')"
	case "sqlite":
		// SQLite ignores DISTINCT: regardless of the distinct value, it should not contain DISTINCT
		// group_concat(expr, ',')
		return "GROUP_CONCAT(" + expr + ", '" + sep + "')"
	default:
		panic("unsupported dialect: " + d.name)
	}
}

// escapeSQLString escapes single quotes for use in SQL string literals by doubling them.
// Example: O'Reilly -> O”Reilly
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ConcatExprs returns a dialect-specific SQL expression that concatenates the provided
// expressions in order, inserting the given separator between each adjacent pair.
// The `exprs` are assumed to be valid SQL expressions (already quoted/escaped as needed).
// The `sep` is treated as a SQL string literal (properly single-quoted and escaped here).
//
// Examples:
//
//	MySQL:    CONCAT(expr1, ',', expr2, ',', expr3)
//	Postgres: expr1 || ',' || expr2 || ',' || expr3
//	SQLite:   expr1 || ',' || expr2 || ',' || expr3
//
// Edge cases:
//   - len(exprs) == 0 -> returns ”
//   - len(exprs) == 1 -> returns exprs[0]
func (d DBDialect) ConcatExprs(exprs []string, sep string) string {
	n := len(exprs)
	if n == 0 {
		return "''"
	}
	if n == 1 {
		return exprs[0]
	}
	var lit string
	if sep != "" {
		lit = "'" + escapeSQLString(sep) + "'"
	}

	switch d.name {
	case "mysql":
		// CONCAT(expr1, 'sep', expr2, 'sep', ...)
		parts := make([]string, 0, n*2-1)
		for i, e := range exprs {
			if i > 0 && lit != "" {
				parts = append(parts, lit)
			}
			parts = append(parts, e)
		}
		return "CONCAT(" + strings.Join(parts, ", ") + ")"
	case "pgx", "sqlite":
		// expr1 || 'sep' || expr2 || 'sep' || ...
		parts := make([]string, 0, n*2-1)
		for i, e := range exprs {
			if i > 0 && lit != "" {
				parts = append(parts, lit)
			}
			parts = append(parts, e)
		}
		return strings.Join(parts, " || ")
	default:
		panic("unsupported dialect: " + d.name)
	}
}
