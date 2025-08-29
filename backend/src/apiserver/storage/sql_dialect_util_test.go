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
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDuplicateError(t *testing.T) {
	mysqlDialect := dialect.NewDBDialect("mysql")
	pgDialect := dialect.NewDBDialect("pgx")
	sqliteDialect := dialect.NewDBDialect("sqlite") // Assuming default is sqlite for tests

	testCases := []struct {
		name    string
		dialect dialect.DBDialect
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
			assert.Equal(t, tc.isDup, isDuplicateError(tc.dialect, tc.err))
		})
	}
}

func TestInsertUpsert(t *testing.T) {
	mysqlDialect := dialect.NewDBDialect("mysql")
	pgDialect := dialect.NewDBDialect("pgx")

	testCases := []struct {
		name       string
		dialect    dialect.DBDialect
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
			keyCols:    []string{"id"}, // keyCols not used by mysql variant
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
			builder := insertUpsert(tc.dialect, tc.table, tc.keyCols, tc.overwrite, tc.updateCols)
			sql, _, err := builder.Columns("foo").Values("bar").ToSql()
			require.NoError(t, err)

			normalizedWant := normalizeSQL(tc.wantSQL)
			normalizedGot := normalizeSQL(sql)

			assert.Contains(t, normalizedGot, normalizedWant)
		})
	}
}
