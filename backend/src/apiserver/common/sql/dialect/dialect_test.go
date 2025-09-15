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
)

func TestGetDialect_MySQL(t *testing.T) {
	d := NewDBDialect("mysql")
	if d.Name() != "mysql" {
		t.Errorf("Expected mysql dialect, got %s", d.Name())
	}
	if d.QuoteIdentifier("abc") != "`abc`" {
		t.Errorf("MySQL quote failed")
	}
	if d.LengthFunc() != "CHAR_LENGTH" {
		t.Errorf("Expected CHAR_LENGTH, got %s", d.LengthFunc())
	}
	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint() != "database exists" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint())
	}
}

func TestGetDialect_Pgx(t *testing.T) {
	d := NewDBDialect("pgx")
	if d.Name() != "pgx" {
		t.Errorf("Expected pgx dialect, got %s", d.Name())
	}
	if d.QuoteIdentifier("abc") != `"abc"` {
		t.Errorf("Pgx quote failed")
	}
	if d.LengthFunc() != "CHAR_LENGTH" {
		t.Errorf("Expected CHAR_LENGTH, got %s", d.LengthFunc())
	}
	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint() != "already exists" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint())
	}
}

func TestGetDialect_SQLite(t *testing.T) {
	d := NewDBDialect("sqlite")
	if d.Name() != "sqlite" {
		t.Errorf("Expected sqlite dialect, got %s", d.Name())
	}
	if d.QuoteIdentifier("abc") != `"abc"` {
		t.Errorf("SQLite quote failed")
	}
	if d.LengthFunc() != "LENGTH" {
		t.Errorf("Expected LENGTH, got %s", d.LengthFunc())
	}
	sql, _, err := d.QueryBuilder().Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint() != "" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint())
	}
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

			if got != want {
				t.Fatalf("ConcatAgg mismatch.\n got: %s\nwant: %s", got, want)
			}
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

			if got != want {
				t.Fatalf("empty sep mismatch for %s.\n got: %s\nwant: %s", name, got, want)
			}
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
			assert.Equal(t, tc.want, escapeSQLString(tc.input))
		})
	}
}
