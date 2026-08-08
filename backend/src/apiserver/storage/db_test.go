// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestMySQLDialect_GroupConcat_WithSeparator(t *testing.T) {
	mysqlDialect := NewMySQLDialect()

	actualQuery := mysqlDialect.GroupConcat(`col1,",",col2`, ";")

	expectedQuery := `GROUP_CONCAT(col1,",",col2 SEPARATOR ";")`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestMySQLDialect_GroupConcat_WithoutSeparator(t *testing.T) {
	mysqlDialect := NewMySQLDialect()

	actualQuery := mysqlDialect.GroupConcat(`col1,",",col2`, "")

	expectedQuery := `GROUP_CONCAT(col1,",",col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestMySQLDialect_Concat_WithSeparator(t *testing.T) {
	mysqlDialect := NewMySQLDialect()

	actualQuery := mysqlDialect.Concat([]string{"col1", "col2"}, ",")

	expectedQuery := `CONCAT(col1,",",col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestMySQLDialect_Concat_WithoutSeparator(t *testing.T) {
	mysqlDialect := NewMySQLDialect()

	actualQuery := mysqlDialect.Concat([]string{"col1", "col2"}, "")

	expectedQuery := `CONCAT(col1,col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_GroupConcat_WithSeparator(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()

	actualQuery := sqliteDialect.GroupConcat(`col1||","||col2`, ";")

	expectedQuery := `GROUP_CONCAT(col1||","||col2, ";")`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_GroupConcat_WithoutSeparator(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()

	actualQuery := sqliteDialect.GroupConcat(`col1||","||col2`, "")

	expectedQuery := `GROUP_CONCAT(col1||","||col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_Concat_WithSeparator(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()

	actualQuery := sqliteDialect.Concat([]string{"col1", "col2"}, ",")

	expectedQuery := `col1||","||col2`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_Concat_WithoutSeparator(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()

	actualQuery := sqliteDialect.Concat([]string{"col1", "col2"}, "")

	expectedQuery := `col1||col2`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_Upsert(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()
	actualQuery := sqliteDialect.Upsert(`insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow")`, "namespace", true, []string{"uuid", "name"}...)
	expectedQuery := `insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow") ON CONFLICT(namespace) DO UPDATE SET uuid=excluded.uuid,name=excluded.name`
	assert.Equal(t, expectedQuery, actualQuery)
	actualQuery2 := sqliteDialect.Upsert(`insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow")`, "namespace", false, []string{"uuid", "name"}...)
	expectedQuery2 := `insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow") ON CONFLICT(namespace) DO UPDATE SET uuid=uuid,name=name`
	assert.Equal(t, expectedQuery2, actualQuery2)
}

func TestMySQLDialect_Upsert(t *testing.T) {
	mysqlDialect := NewMySQLDialect()
	actualQuery := mysqlDialect.Upsert(`insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow")`, "namespace", true, []string{"uuid", "name"}...)
	expectedQuery := `insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow") ON DUPLICATE KEY UPDATE uuid=VALUES(uuid),name=VALUES(name)`
	assert.Equal(t, expectedQuery, actualQuery)
	actualQuery2 := mysqlDialect.Upsert(`insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow")`, "namespace", false, []string{"uuid", "name"}...)
	expectedQuery2 := `insert into table (uuid, name, namespace) values ("a", "item1", "kubeflow"),("b", "item1", "kubeflow") ON DUPLICATE KEY UPDATE uuid=uuid,name=name`
	assert.Equal(t, expectedQuery2, actualQuery2)
}

func TestMySQLDialect_UpdateWithJointOrFrom(t *testing.T) {
	mysqlDialect := NewMySQLDialect()
	actualQuery := mysqlDialect.UpdateWithJointOrFrom(
		"target_table",
		"other_table",
		"State = ?",
		"target_table.Name = other_table.Name",
		"target_table.status = ?")
	expectedQuery := `UPDATE target_table INNER JOIN other_table ON target_table.Name = other_table.Name SET State = ? WHERE target_table.status = ?`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestSQLiteDialect_UpdateWithJointOrFrom(t *testing.T) {
	sqliteDialect := NewSQLiteDialect()
	actualQuery := sqliteDialect.UpdateWithJointOrFrom(
		"target_table",
		"other_table",
		"State = ?",
		"target_table.Name = other_table.Name",
		"target_table.status = ?")
	expectedQuery := `UPDATE target_table SET State = ? FROM other_table WHERE target_table.Name = other_table.Name AND target_table.status = ?`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_GroupConcat_WithSeparator(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.GroupConcat("col1", ";")

	expectedQuery := `STRING_AGG(col1::text, ';')`
	assert.Equal(t, expectedQuery, actualQuery)
}

// PostgreSQL requires an explicit separator, so an empty one must fall back to
// the comma that MySQL's GROUP_CONCAT uses by default rather than emitting a
// single-argument STRING_AGG, which does not parse.
func TestPostgreSQLDialect_GroupConcat_WithoutSeparator(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.GroupConcat("col1", "")

	expectedQuery := `STRING_AGG(col1::text, ',')`
	assert.Equal(t, expectedQuery, actualQuery)
}

// Separators must be single-quoted: PostgreSQL reads a double-quoted token as
// an identifier, not a string literal.
func TestPostgreSQLDialect_Concat_WithSeparator(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.Concat([]string{"col1", "col2"}, "-")

	expectedQuery := `CONCAT(col1,'-',col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_Concat_WithoutSeparator(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.Concat([]string{"col1", "col2"}, "")

	expectedQuery := `CONCAT(col1,col2)`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_Upsert_Overwrite(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.Upsert("INSERT INTO tbl VALUES (?)", "UUID", true, "Name", "State")

	expectedQuery := `INSERT INTO tbl VALUES (?) ON CONFLICT(UUID) DO UPDATE SET Name=EXCLUDED.Name,State=EXCLUDED.State`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_Upsert_NoOverwrite(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	actualQuery := postgresDialect.Upsert("INSERT INTO tbl VALUES (?)", "UUID", false, "Name")

	expectedQuery := `INSERT INTO tbl VALUES (?) ON CONFLICT(UUID) DO UPDATE SET Name=Name`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_SelectForUpdate(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	assert.Equal(t, "SELECT * FROM tbl FOR UPDATE", postgresDialect.SelectForUpdate("SELECT * FROM tbl"))
}

// PostgreSQL uses UPDATE ... FROM, like SQLite, rather than MySQL's
// UPDATE ... INNER JOIN.
func TestPostgreSQLDialect_UpdateWithJointOrFrom(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()
	actualQuery := postgresDialect.UpdateWithJointOrFrom(
		"target_table",
		"other_table",
		"State = ?",
		"target_table.Name = other_table.Name",
		"target_table.status = ?")
	expectedQuery := `UPDATE target_table SET State = ? FROM other_table WHERE target_table.Name = other_table.Name AND target_table.status = ?`
	assert.Equal(t, expectedQuery, actualQuery)
}

func TestPostgreSQLDialect_IsDuplicateError(t *testing.T) {
	postgresDialect := NewPostgreSQLDialect()

	assert.True(t, postgresDialect.IsDuplicateError(&pgconn.PgError{Code: pgUniqueViolation}))
	// A wrapped error must still be recognised: the pgx stdlib driver wraps
	// *pgconn.PgError before it surfaces through database/sql.
	assert.True(t, postgresDialect.IsDuplicateError(fmt.Errorf("insert failed: %w", &pgconn.PgError{Code: pgUniqueViolation})))

	// A different SQLSTATE is not a duplicate-key error.
	assert.False(t, postgresDialect.IsDuplicateError(&pgconn.PgError{Code: "23503"}))
	assert.False(t, postgresDialect.IsDuplicateError(errors.New("some other failure")))
	assert.False(t, postgresDialect.IsDuplicateError(nil))
}

// The MySQL dialect must not treat a PostgreSQL unique violation as a
// duplicate-key error, which is what happened while every install used
// MySQLDialect regardless of driver.
func TestMySQLDialect_IsDuplicateError_IgnoresPostgresError(t *testing.T) {
	mysqlDialect := NewMySQLDialect()

	assert.False(t, mysqlDialect.IsDuplicateError(&pgconn.PgError{Code: pgUniqueViolation}))
}
