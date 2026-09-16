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

// Package dialect provides a minimal, shared SQL dialect abstraction
// for apiserver components. It centralizes identifier quoting, placeholder
// styles, upsert syntax, and error classification for different backends.
package dialect

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// QuoteFunction is the signature of a dialect-aware identifier quoting function.
type QuoteFunction = func(string) string

// DBDialect abstracts SQL dialect differences so that storage-layer code
// can be written without switching on the backend name.
type DBDialect interface {
	// Name returns the backend name (e.g., "mysql", "pgx", "sqlite").
	Name() string

	// QuoteIdentifier returns the dialect-appropriate quoted form of id.
	QuoteIdentifier(id string) string

	// LengthFunc returns the SQL function name for string length.
	LengthFunc() string

	// QueryBuilder returns a Squirrel StatementBuilderType configured with
	// the correct placeholder format for this dialect.
	QueryBuilder() sq.StatementBuilderType

	// IsDuplicateDatabaseError returns true if err indicates that a
	// CREATE DATABASE statement failed because the database already exists.
	IsDuplicateDatabaseError(err error) bool

	// StringCollation returns the SQL collation clause to append after
	// LOWER() expressions when sorting or comparing string fields.
	StringCollation() string

	// ConcatAgg returns a dialect-specific SQL expression for concatenating
	// string values from multiple rows into a single string.
	ConcatAgg(distinct bool, expr, sep string) string

	// ConcatExprs returns a dialect-specific SQL expression that concatenates
	// the provided expressions in order, inserting sep between each pair.
	ConcatExprs(exprs []string, sep string) string

	// IsDuplicateKeyError returns true if err indicates a unique/duplicate-key
	// violation for this dialect.
	IsDuplicateKeyError(err error) bool

	// Upsert starts an INSERT builder and appends the dialect-specific
	// upsert clause. Callers continue chaining .Columns(...).Values(...).
	Upsert(table string, keyCols []string, overwrite bool, updateCols []string) sq.InsertBuilder

	// FinalizeSelect applies the dialect's placeholder format to the
	// outermost SelectBuilder and calls ToSql(). Use this instead of
	// calling ToSql() directly when the query may contain nested
	// sub-selects built with sq.Question placeholders.
	FinalizeSelect(builder sq.SelectBuilder) (string, []interface{}, error)

	// SelectForUpdate appends a row-locking clause to query for dialects
	// that support it. Dialects with no concurrent-writer story are a no-op.
	SelectForUpdate(query string) string
}

// NewDBDialect constructs a DBDialect for the given backend name.
// Supported names: "mysql", "pgx", "sqlite" (sqlite is for tests).
func NewDBDialect(name string) DBDialect {
	switch name {
	case "mysql":
		return mysqlDialect{}
	case "pgx":
		return pgxDialect{}
	case "sqlite":
		return sqliteDialect{}
	default:
		panic(fmt.Sprintf("unsupported dialect: %s", name))
	}
}

// QuoteAll applies q to each element of cols and returns a new slice.
// Use this when passing columns into squirrel.Select(...), so each identifier
// is properly quoted for the current SQL dialect.
func QuoteAll(q QuoteFunction, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = q(c)
	}
	return out
}

// QualifiedColumn returns a function that qualifies a column name with the
// given table, quoting both identifiers with q.
func QualifiedColumn(q QuoteFunction, table string) func(string) string {
	quotedTable := q(table)
	return func(col string) string {
		return fmt.Sprintf("%s.%s", quotedTable, q(col))
	}
}

// escapeSQLString escapes backslashes and single quotes for use in SQL string literals.
func escapeSQLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, "'", "''")
}
