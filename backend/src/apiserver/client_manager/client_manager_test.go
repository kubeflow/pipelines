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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
)

// getTestSQLite returns an isolated in-memory sqlite DB for each test.
// We use a unique DSN per test to avoid "table already exists" collisions when tests reuse the same shared cache.
func getTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	// sanitize test name to be a valid sqlite file identifier
	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func createOldExperimentSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Ensure a clean slate if a previous test already created it in the shared cache.
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS experiments`).Error)

	stmt := `
CREATE TABLE IF NOT EXISTS experiments (
  UUID TEXT NOT NULL,
  Name TEXT NOT NULL,
  Namespace TEXT NOT NULL,
  CreatedAtInSec INTEGER NOT NULL DEFAULT 0,
  LastRunCreatedAtInSec INTEGER NOT NULL DEFAULT 0,
  StorageState TEXT NOT NULL DEFAULT 'STORAGESTATE_AVAILABLE',
  PRIMARY KEY (UUID)
);`
	require.NoError(t, db.Exec(stmt).Error)
}

// insertTooLongExperimentName inserts one row whose Name exceeds 128 chars.
func insertTooLongExperimentName(t *testing.T, db *gorm.DB) {
	t.Helper()
	longName := strings.Repeat("x", 150)
	err := db.Exec(`
INSERT INTO experiments (UUID, Name, Namespace, CreatedAtInSec, LastRunCreatedAtInSec, StorageState)
VALUES (?, ?, 'ns', 0, 0, 'STORAGESTATE_AVAILABLE')`, "uuid-1", longName).Error
	require.NoError(t, err)
}

func TestRunPreflightLengthChecks_FailOnTooLong(t *testing.T) {
	db := getTestSQLite(t)

	createOldExperimentSchema(t, db)
	insertTooLongExperimentName(t, db)

	specs := []validation.ColLenSpec{
		{Model: &model.Experiment{}, Field: "Name", Max: 128},
	}

	dialect := GetDialect("sqlite")
	err := runPreflightLengthChecks(db, dialect, specs)
	t.Logf("FULL ERR:\n%+v", err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Preflight")
}

func TestRunPreflightLengthChecks_PassWhenOK(t *testing.T) {
	db := getTestSQLite(t)

	createOldExperimentSchema(t, db)
	// no long rows

	dialect := GetDialect("sqlite")
	err := runPreflightLengthChecks(db, dialect, []validation.ColLenSpec{
		{Model: &model.Experiment{}, Field: "Name", Max: 128},
	})
	require.NoError(t, err)
}

func TestFieldMeta_TaskRunId(t *testing.T) {
	// FieldMeta only inspects schema; sqlite driver is sufficient.
	db := getTestSQLite(t)
	table, dbCol, err := FieldMeta(db, &model.Task{}, "RunId")
	require.NoError(t, err)
	assert.Equal(t, "tasks", table)
	assert.Equal(t, "RunUUID", dbCol)
}
