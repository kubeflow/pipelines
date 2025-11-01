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
	"fmt"
	"testing"

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
	require.Equal(t, "database exists", d.ExistDatabaseErrHint())
}

func TestGetDialect_Pgx(t *testing.T) {
	d := NewDBDialect("pgx")
	require.Equal(t, "pgx", d.Name())
	require.Equal(t, `"abc"`, d.QuoteIdentifier("abc"))
	require.Equal(t, "CHAR_LENGTH", d.LengthFunc())

	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", sql)
	require.Equal(t, "already exists", d.ExistDatabaseErrHint())
}

func TestGetDialect_SQLite(t *testing.T) {
	d := NewDBDialect("sqlite")
	require.Equal(t, "sqlite", d.Name())
	require.Equal(t, `"abc"`, d.QuoteIdentifier("abc"))
	require.Equal(t, "LENGTH", d.LengthFunc())

	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", sql)
	require.Equal(t, "", d.ExistDatabaseErrHint())
}

// TestConcatAgg_TableDriven verifies the aggregation SQL snippet generation across dialects.
func TestConcatAgg_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		dialectName string
		distinct    bool
		sep         string
	}{
		{"mysql_no_distinct_comma", "mysql", false, ","},
		{"mysql_distinct_pipe", "mysql", true, "|"},
		{"pgx_no_distinct_comma", "pgx", false, ","},
		{"pgx_distinct_empty_sep", "pgx", true, ""},
		{"sqlite_no_distinct_comma", "sqlite", false, ","},
		{"sqlite_distinct_ignored", "sqlite", true, ","},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDBDialect(tc.dialectName)
			q := d.QuoteIdentifier
			expr := q("r") + "." + q("Payload")

			got := d.ConcatAgg(tc.distinct, expr, tc.sep)

			var want string
			switch tc.dialectName {
			case "mysql":
				if tc.distinct {
					want = fmt.Sprintf("GROUP_CONCAT(DISTINCT %s SEPARATOR '%s')", expr, tc.sep)
				} else {
					want = fmt.Sprintf("GROUP_CONCAT(%s SEPARATOR '%s')", expr, tc.sep)
				}
			case "pgx":
				if tc.distinct {
					want = fmt.Sprintf("string_agg(DISTINCT %s, '%s')", expr, tc.sep)
				} else {
					want = fmt.Sprintf("string_agg(%s, '%s')", expr, tc.sep)
				}
			case "sqlite":
				want = fmt.Sprintf("GROUP_CONCAT(%s, '%s')", expr, tc.sep)
			default:
				t.Fatalf("unknown dialect: %s", tc.dialectName)
			}

			require.Equal(t, want, got)
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

// Additional coverage: when the separator is an empty string, ensure it renders as ” (consistent across all dialects)
func TestConcatAgg_EmptySeparator(t *testing.T) {
	for _, name := range []string{"mysql", "pgx", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			d := NewDBDialect(name)
			q := d.QuoteIdentifier
			expr := q("t") + "." + q("col")

			got := d.ConcatAgg(false, expr, "")

			var want string
			switch name {
			case "mysql":
				want = fmt.Sprintf("GROUP_CONCAT(%s SEPARATOR '')", expr)
			case "pgx":
				want = fmt.Sprintf("string_agg(%s, '')", expr)
			case "sqlite":
				want = fmt.Sprintf("GROUP_CONCAT(%s, '')", expr)
			}

			require.Equal(t, want, got)
		})
	}
}

func TestEscapeSQLString(t *testing.T) {
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
		{"empty_string", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, EscapeSQLString(tc.input))
		})
	}
}
