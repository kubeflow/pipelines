// Copyright 2018 Google LLC
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
	"testing"

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
