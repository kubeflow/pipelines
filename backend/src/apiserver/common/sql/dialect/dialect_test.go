// Copyright 2018-2025 The Kubeflow Authors
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
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDialect_MySQL(t *testing.T) {
	d := NewDBDialect("mysql")
	require.Equal(t, "mysql", d.Name())
	require.Equal(t, "`abc`", d.QuoteIdentifier("abc"))
	require.Equal(t, "CHAR_LENGTH", d.LengthFunc())

	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", sql)
}

func TestGetDialect_Pgx(t *testing.T) {
	d := NewDBDialect("pgx")
	require.Equal(t, "pgx", d.Name())
	require.Equal(t, `"abc"`, d.QuoteIdentifier("abc"))
	require.Equal(t, "CHAR_LENGTH", d.LengthFunc())

	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", sql)
}

func TestGetDialect_SQLite(t *testing.T) {
	d := NewDBDialect("sqlite")
	require.Equal(t, "sqlite", d.Name())
	require.Equal(t, `"abc"`, d.QuoteIdentifier("abc"))
	require.Equal(t, "LENGTH", d.LengthFunc())

	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", sql)
}

// TestConcatAgg_TableDriven verifies the aggregation SQL snippet generation across dialects.
func TestConcatAgg_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		dialectName string
		distinct    bool
		sep         string
		want        string
	}{
		{"mysql_no_distinct_comma", "mysql", false, ",", "GROUP_CONCAT(`r`.`Payload` SEPARATOR ',')"},
		{"mysql_distinct_pipe", "mysql", true, "|", "GROUP_CONCAT(DISTINCT `r`.`Payload` SEPARATOR '|')"},
		{"pgx_no_distinct_comma", "pgx", false, ",", `string_agg("r"."Payload", ',')`},
		{"pgx_distinct_empty_sep", "pgx", true, "", `string_agg(DISTINCT "r"."Payload", '')`},
		{"sqlite_no_distinct_comma", "sqlite", false, ",", `GROUP_CONCAT("r"."Payload", ',')`},
		{"sqlite_distinct_ignored", "sqlite", true, ",", `GROUP_CONCAT("r"."Payload", ',')`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDBDialect(tc.dialectName)
			q := d.QuoteIdentifier
			expr := q("r") + "." + q("Payload")

			got := d.ConcatAgg(tc.distinct, expr, tc.sep)

			require.Equal(t, tc.want, got)
		})
	}
}

func TestConcatExprs(t *testing.T) {
	testCases := []struct {
		name        string
		dialectName string
		exprs       []string
		sep         string
		want        string
	}{
		// MySQL cases
		{"mysql_multiple_exprs", "mysql", []string{"'a'", "'b'", "'c'"}, ",", "CONCAT('a', ',', 'b', ',', 'c')"},
		{"mysql_single_expr", "mysql", []string{"'a'"}, ",", "'a'"},
		{"mysql_zero_exprs", "mysql", []string{}, ",", "''"},
		{"mysql_sep_with_quote", "mysql", []string{"'a'", "'b'"}, "','", "CONCAT('a', ''',''', 'b')"},
		{"mysql_empty_sep", "mysql", []string{"'a'", "'b'"}, "", "CONCAT('a', 'b')"},

		// Postgres (pgx) cases
		{"pgx_multiple_exprs", "pgx", []string{"'a'", "'b'", "'c'"}, ",", "'a' || ',' || 'b' || ',' || 'c'"},
		{"pgx_single_expr", "pgx", []string{"'a'"}, ",", "'a'"},
		{"pgx_zero_exprs", "pgx", []string{}, ",", "''"},
		{"pgx_sep_with_quote", "pgx", []string{"'a'", "'b'"}, "','", "'a' || ''',''' || 'b'"},
		{"pgx_empty_sep", "pgx", []string{"'a'", "'b'"}, "", "'a' || 'b'"},

		// SQLite cases
		{"sqlite_multiple_exprs", "sqlite", []string{"'a'", "'b'", "'c'"}, ",", "'a' || ',' || 'b' || ',' || 'c'"},
		{"sqlite_single_expr", "sqlite", []string{"'a'"}, ",", "'a'"},
		{"sqlite_zero_exprs", "sqlite", []string{}, ",", "''"},
		{"sqlite_sep_with_quote", "sqlite", []string{"'a'", "'b'"}, "','", "'a' || ''',''' || 'b'"},
		{"sqlite_empty_sep", "sqlite", []string{"'a'", "'b'"}, "", "'a' || 'b'"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDBDialect(tc.dialectName)
			got := d.ConcatExprs(tc.exprs, tc.sep)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Additional coverage: when the separator is an empty string, ensure it renders as " (consistent across all dialects)
func TestConcatAgg_EmptySeparator(t *testing.T) {
	testCases := []struct {
		name        string
		dialectName string
		want        string
	}{
		{"mysql_empty_sep", "mysql", "GROUP_CONCAT(`t`.`col` SEPARATOR '')"},
		{"pgx_empty_sep", "pgx", `string_agg("t"."col", '')`},
		{"sqlite_empty_sep", "sqlite", `GROUP_CONCAT("t"."col", '')`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDBDialect(tc.dialectName)
			q := d.QuoteIdentifier
			expr := q("t") + "." + q("col")

			got := d.ConcatAgg(false, expr, "")

			require.Equal(t, tc.want, got)
		})
	}
}

func TestQuoteIdentifier_EscapesEmbeddedQuotes(t *testing.T) {
	testCases := []struct {
		name        string
		dialectName string
		input       string
		expected    string
	}{
		// PostgreSQL (pgx) - escapes double quotes
		{"pgx_normal", "pgx", "column", `"column"`},
		{"pgx_with_quote", "pgx", `column"name`, `"column""name"`},
		{"pgx_injection_attempt", "pgx", `accuracy"; DROP TABLE x; --`, `"accuracy""; DROP TABLE x; --"`},
		{"pgx_multiple_quotes", "pgx", `a"b"c`, `"a""b""c"`},

		// MySQL - escapes backticks
		{"mysql_normal", "mysql", "column", "`column`"},
		{"mysql_with_backtick", "mysql", "column`name", "`column``name`"},
		{"mysql_injection_attempt", "mysql", "accuracy`; DROP TABLE x; --", "`accuracy``; DROP TABLE x; --`"},
		{"mysql_multiple_backticks", "mysql", "a`b`c", "`a``b``c`"},

		// SQLite - escapes double quotes
		{"sqlite_normal", "sqlite", "column", `"column"`},
		{"sqlite_with_quote", "sqlite", `column"name`, `"column""name"`},
		{"sqlite_injection_attempt", "sqlite", `accuracy"; DROP TABLE x; --`, `"accuracy""; DROP TABLE x; --"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDBDialect(tc.dialectName)
			result := d.QuoteIdentifier(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func squeezeSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// normalizeSQL normalizes SQL for fragment-based comparison (HasPrefix / Contains).
// Strips identifier quotes and lowercases so dialect-agnostic fragments match both MySQL and PostgreSQL.
//
// NOTE: Do NOT use for exact equality assertions — stripping quotes hides missing-quote bugs.
// For full-string equality tests, compare raw SQL directly.
func normalizeSQL(s string) string {
	s = squeezeSpaces(s)
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, `"`, "")
	return strings.ToLower(s)
}

func TestIsDuplicateKeyError(t *testing.T) {
	mysqlDialect := NewDBDialect("mysql")
	pgDialect := NewDBDialect("pgx")
	sqliteDialect := NewDBDialect("sqlite")

	testCases := []struct {
		name    string
		dialect DBDialect
		err     error
		isDup   bool
	}{
		{"MySQL Duplicate", mysqlDialect, &mysql.MySQLError{Number: 1062}, true},
		{"MySQL Other Error", mysqlDialect, &mysql.MySQLError{Number: 1045}, false},
		{"Postgres Duplicate", pgDialect, &pgconn.PgError{Code: pgerrcode.UniqueViolation}, true},
		{"Postgres Other Error", pgDialect, &pgconn.PgError{Code: pgerrcode.InvalidPassword}, false},
		{"SQLite Duplicate", sqliteDialect, sqlite3.Error{Code: sqlite3.ErrConstraint}, true},
		{"SQLite Other Error", sqliteDialect, sqlite3.Error{Code: sqlite3.ErrBusy}, false},
		{"Generic Error", mysqlDialect, errors.New("generic error"), false},
		{"Nil Error", pgDialect, nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isDup, tc.dialect.IsDuplicateKeyError(tc.err))
		})
	}
}

func TestIsDuplicateDatabaseError(t *testing.T) {
	mysqlDialect := NewDBDialect("mysql")
	pgDialect := NewDBDialect("pgx")
	sqliteDialect := NewDBDialect("sqlite")

	testCases := []struct {
		name    string
		dialect DBDialect
		err     error
		isDup   bool
	}{
		{"Postgres DuplicateDatabase", pgDialect, &pgconn.PgError{Code: pgerrcode.DuplicateDatabase}, true},
		{"Postgres UniqueViolation", pgDialect, &pgconn.PgError{Code: pgerrcode.UniqueViolation}, false},
		{"MySQL DuplicateDatabase", mysqlDialect, &mysql.MySQLError{Number: 1007}, true},
		{"MySQL DuplicateKey", mysqlDialect, &mysql.MySQLError{Number: 1062}, false},
		{"SQLite Always False", sqliteDialect, sqlite3.Error{Code: sqlite3.ErrConstraint}, false},
		{"Generic Error", pgDialect, errors.New("already exists"), false},
		{"Nil Error", pgDialect, nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isDup, tc.dialect.IsDuplicateDatabaseError(tc.err))
		})
	}
}

func TestQuoteAll(t *testing.T) {
	quote := func(s string) string { return "`" + s + "`" }
	testCases := []struct {
		name string
		cols []string
		want []string
	}{
		{"Multiple Columns", []string{"col1", "col2"}, []string{"`col1`", "`col2`"}},
		{"Single Column", []string{"col1"}, []string{"`col1`"}},
		{"Empty Slice", []string{}, []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := QuoteAll(quote, tc.cols)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestQualifiedColumn(t *testing.T) {
	quote := func(s string) string { return "`" + s + "`" }
	col := QualifiedColumn(quote, "my_table")

	assert.Equal(t, "`my_table`.`col1`", col("col1"))
	assert.Equal(t, "`my_table`.`col2`", col("col2"))
}

func TestUpsert(t *testing.T) {
	mysqlDialect := NewDBDialect("mysql")
	pgDialect := NewDBDialect("pgx")

	testCases := []struct {
		name       string
		dialect    DBDialect
		table      string
		keyCols    []string
		overwrite  bool
		updateCols []string
		wantSQL    string
	}{
		{
			name:       "MySQL Overwrite",
			dialect:    mysqlDialect,
			table:      "my_table",
			keyCols:    []string{"id"},
			overwrite:  true,
			updateCols: []string{"col1", "col2"},
			wantSQL:    "ON DUPLICATE KEY UPDATE `col1`=VALUES(`col1`), `col2`=VALUES(`col2`)",
		},
		{
			name:       "MySQL Keep Existing",
			dialect:    mysqlDialect,
			table:      "my_table",
			keyCols:    []string{"id"},
			overwrite:  false,
			updateCols: []string{"col1", "col2"},
			wantSQL:    "ON DUPLICATE KEY UPDATE `col1`=`col1`, `col2`=`col2`",
		},
		{
			name:       "Postgres Overwrite Single Key",
			dialect:    pgDialect,
			table:      "my_table",
			keyCols:    []string{"id"},
			overwrite:  true,
			updateCols: []string{"col1", "col2"},
			wantSQL:    `ON CONFLICT ("id") DO UPDATE SET "col1"=excluded."col1", "col2"=excluded."col2"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.dialect.Upsert(tc.table, tc.keyCols, tc.overwrite, tc.updateCols)
			sql, _, err := builder.Columns("foo").Values("bar").ToSql()
			assert.NoError(t, err)

			normalizedWant := normalizeSQL(tc.wantSQL)
			normalizedGot := normalizeSQL(sql)

			assert.Contains(t, normalizedGot, normalizedWant)
		})
	}
}

func Test_escapeSQLString(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"no_quotes", "hello world", "hello world"},
		{"single_quote", "O'Reilly", "O''Reilly"},
		{"multiple_quotes", "it's a 'test'", "it''s a ''test''"},
		{"leading_quote", "'test", "''test"},
		{"trailing_quote", "test'", "test''"},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"backslash_with_quote", `it\'s`, `it\\''s`},
		{"empty_string", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, escapeSQLString(tc.input))
		})
	}
}
