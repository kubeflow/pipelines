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

package clientmanager

import (
	"testing"
)

func TestGetDialect_MySQL(t *testing.T) {
	d := GetDialect("mysql")
	if d.Name != "mysql" {
		t.Errorf("Expected mysql dialect, got %s", d.Name)
	}
	if d.QuoteIdentifier("abc") != "`abc`" {
		t.Errorf("MySQL quote failed")
	}
	if d.LengthFunc != "CHAR_LENGTH" {
		t.Errorf("Expected CHAR_LENGTH, got %s", d.LengthFunc)
	}
	sql, _, err := d.StatementBuilder.Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint != "database exists" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint)
	}
}

func TestGetDialect_Pgx(t *testing.T) {
	d := GetDialect("pgx")
	if d.Name != "pgx" {
		t.Errorf("Expected pgx dialect, got %s", d.Name)
	}
	if d.QuoteIdentifier("abc") != `"abc"` {
		t.Errorf("Pgx quote failed")
	}
	if d.LengthFunc != "CHAR_LENGTH" {
		t.Errorf("Expected CHAR_LENGTH, got %s", d.LengthFunc)
	}
	sql, _, err := d.StatementBuilder.Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint != "already exists" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint)
	}
}

func TestGetDialect_SQLite(t *testing.T) {
	d := GetDialect("sqlite")
	if d.Name != "sqlite" {
		t.Errorf("Expected sqlite dialect, got %s", d.Name)
	}
	if d.QuoteIdentifier("abc") != `"abc"` {
		t.Errorf("SQLite quote failed")
	}
	if d.LengthFunc != "LENGTH" {
		t.Errorf("Expected LENGTH, got %s", d.LengthFunc)
	}
	sql, _, err := d.StatementBuilder.Select("1").ToSql()
	if err != nil {
		t.Errorf("Failed to build SQL: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("Expected 'SELECT 1', got '%s'", sql)
	}
	if d.ExistDatabaseErrHint != "" {
		t.Errorf("Incorrect error hint: %s", d.ExistDatabaseErrHint)
	}
}
