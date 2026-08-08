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
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// pgUniqueViolation is the PostgreSQL SQLSTATE for a unique constraint
// violation. See https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgUniqueViolation = "23505"

// DB a struct wrapping plain sql library with SQL dialect, to solve any feature
// difference between MySQL, which is used in production, and Sqlite, which is used
// for unit testing.
type DB struct {
	*sql.DB
	SQLDialect
}

// NewDB creates a DB.
func NewDB(db *sql.DB, dialect SQLDialect) *DB {
	return &DB{db, dialect}
}

// SQLDialect abstracts common sql queries which vary in different dialect.
// It is used to bridge the difference between mysql (production) and sqlite
// (test).
type SQLDialect interface {
	// GroupConcat builds query to group concatenate `expr` in each row and use `separator`
	// to join rows in a group.
	GroupConcat(expr string, separator string) string

	// Concat builds query to concatenete a list of `exprs` into a single string with
	// a separator in between.
	Concat(exprs []string, separator string) string

	// Check whether the error is a SQL duplicate entry error or not
	IsDuplicateError(err error) bool

	// Modifies the SELECT clause in query to return one that locks the selected
	// row for update.
	SelectForUpdate(query string) string

	// Inserts new rows and updates duplicates based on the key column.
	Upsert(query string, key string, overwrite bool, columns ...string) string

	// Updates a table using UPDATE with JOIN (mysql/production) or UPDATE FROM (sqlite/test).
	UpdateWithJointOrFrom(targetTable, joinTable, setClause, joinClause, whereClause string) string
}

// MySQLDialect implements SQLDialect with mysql dialect implementation.
type MySQLDialect struct{}

func (d MySQLDialect) GroupConcat(expr string, separator string) string {
	var buffer bytes.Buffer
	buffer.WriteString("GROUP_CONCAT(")
	buffer.WriteString(expr)
	if separator != "" {
		buffer.WriteString(fmt.Sprintf(" SEPARATOR \"%s\"", separator))
	}
	buffer.WriteString(")")
	return buffer.String()
}

func (d MySQLDialect) Concat(exprs []string, separator string) string {
	separatorSQL := ","
	if separator != "" {
		separatorSQL = fmt.Sprintf(`,"%s",`, separator)
	}
	return fmt.Sprintf("CONCAT(%s)", strings.Join(exprs, separatorSQL))
}

func (d MySQLDialect) IsDuplicateError(err error) bool {
	sqlError, ok := err.(*mysql.MySQLError)
	return ok && sqlError.Number == mysqlerr.ER_DUP_ENTRY
}

func (d MySQLDialect) UpdateWithJointOrFrom(targetTable, joinTable, setClause, joinClause, whereClause string) string {
	return fmt.Sprintf("UPDATE %s INNER JOIN %s ON %s SET %s WHERE %s", targetTable, joinTable, joinClause, setClause, whereClause)
}

// SQLiteDialect implements SQLDialect with sqlite dialect implementation.
type SQLiteDialect struct{}

func (d SQLiteDialect) GroupConcat(expr string, separator string) string {
	var buffer bytes.Buffer
	buffer.WriteString("GROUP_CONCAT(")
	buffer.WriteString(expr)
	if separator != "" {
		buffer.WriteString(fmt.Sprintf(", \"%s\"", separator))
	}
	buffer.WriteString(")")
	return buffer.String()
}

func (d SQLiteDialect) Concat(exprs []string, separator string) string {
	separatorSQL := "||"
	if separator != "" {
		separatorSQL = fmt.Sprintf(`||"%s"||`, separator)
	}
	return strings.Join(exprs, separatorSQL)
}

func (d MySQLDialect) SelectForUpdate(query string) string {
	return query + " FOR UPDATE"
}

func (d SQLiteDialect) SelectForUpdate(query string) string {
	return query
}

func (d MySQLDialect) Upsert(query string, key string, overwrite bool, columns ...string) string {
	return fmt.Sprintf("%v ON DUPLICATE KEY UPDATE %v", query, prepareUpdateSuffixMySQL(columns, overwrite))
}

func (d SQLiteDialect) Upsert(query string, key string, overwrite bool, columns ...string) string {
	return fmt.Sprintf("%v ON CONFLICT(%v) DO UPDATE SET %v", query, key, prepareUpdateSuffixSQLite(columns, overwrite))
}

func (d SQLiteDialect) IsDuplicateError(err error) bool {
	sqlError, ok := err.(sqlite3.Error)
	return ok && sqlError.Code == sqlite3.ErrConstraint
}

func (d SQLiteDialect) UpdateWithJointOrFrom(targetTable, joinTable, setClause, joinClause, whereClause string) string {
	return fmt.Sprintf("UPDATE %s SET %s FROM %s WHERE %s AND %s", targetTable, setClause, joinTable, joinClause, whereClause)
}

// PostgreSQLDialect implements SQLDialect with PostgreSQL dialect implementation.
type PostgreSQLDialect struct{}

// GroupConcat uses STRING_AGG, PostgreSQL's equivalent of GROUP_CONCAT. Unlike
// MySQL, the separator is a required argument, so an empty separator falls back
// to the comma MySQL's GROUP_CONCAT defaults to. The expression is cast to text
// because STRING_AGG has no implicit conversion from other types.
func (d PostgreSQLDialect) GroupConcat(expr string, separator string) string {
	if separator == "" {
		separator = ","
	}
	return fmt.Sprintf("STRING_AGG(%s::text, '%s')", expr, separator)
}

// Concat uses the SQL-standard CONCAT function, which PostgreSQL supports.
// Separators are single-quoted, since PostgreSQL reads double quotes as an
// identifier rather than a string literal.
func (d PostgreSQLDialect) Concat(exprs []string, separator string) string {
	separatorSQL := ","
	if separator != "" {
		separatorSQL = fmt.Sprintf(`,'%s',`, separator)
	}
	return fmt.Sprintf("CONCAT(%s)", strings.Join(exprs, separatorSQL))
}

// IsDuplicateError reports whether err is a PostgreSQL unique-violation
// (SQLSTATE 23505). errors.As is used because the pgx stdlib driver wraps
// *pgconn.PgError before returning it through database/sql.
func (d PostgreSQLDialect) IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

func (d PostgreSQLDialect) SelectForUpdate(query string) string {
	return query + " FOR UPDATE"
}

// Upsert uses INSERT ... ON CONFLICT DO UPDATE, which PostgreSQL shares with
// SQLite, rather than MySQL's ON DUPLICATE KEY UPDATE.
func (d PostgreSQLDialect) Upsert(query string, key string, overwrite bool, columns ...string) string {
	return fmt.Sprintf("%v ON CONFLICT(%v) DO UPDATE SET %v", query, key, prepareUpdateSuffixPostgreSQL(columns, overwrite))
}

// UpdateWithJointOrFrom uses UPDATE ... FROM, which PostgreSQL shares with
// SQLite, rather than MySQL's UPDATE ... INNER JOIN.
func (d PostgreSQLDialect) UpdateWithJointOrFrom(targetTable, joinTable, setClause, joinClause, whereClause string) string {
	return fmt.Sprintf("UPDATE %s SET %s FROM %s WHERE %s AND %s", targetTable, setClause, joinTable, joinClause, whereClause)
}

func NewMySQLDialect() MySQLDialect {
	return MySQLDialect{}
}

func NewSQLiteDialect() SQLiteDialect {
	return SQLiteDialect{}
}

func NewPostgreSQLDialect() PostgreSQLDialect {
	return PostgreSQLDialect{}
}

func prepareUpdateSuffixMySQL(columns []string, overwrite bool) string {
	columnsExtended := make([]string, 0)
	if overwrite {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=VALUES(%[1]v)", c))
		}
	} else {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=%[1]v", c))
		}
	}
	return strings.Join(columnsExtended, ",")
}

// prepareUpdateSuffixPostgreSQL builds the SET clause of an ON CONFLICT DO
// UPDATE. PostgreSQL spells the pseudo-table EXCLUDED, matching SQLite's
// excluded, rather than MySQL's VALUES().
func prepareUpdateSuffixPostgreSQL(columns []string, overwrite bool) string {
	columnsExtended := make([]string, 0)
	if overwrite {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=EXCLUDED.%[1]v", c))
		}
	} else {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=%[1]v", c))
		}
	}
	return strings.Join(columnsExtended, ",")
}

func prepareUpdateSuffixSQLite(columns []string, overwrite bool) string {
	columnsExtended := make([]string, 0)
	if overwrite {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=excluded.%[1]v", c))
		}
	} else {
		for _, c := range columns {
			columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=%[1]v", c))
		}
	}
	return strings.Join(columnsExtended, ",")
}
