// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientmanager

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecurringRunIndexPreservesValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	require.NoError(t, db.Exec("CREATE TABLE run_details (UUID text PRIMARY KEY, JobUUID varchar(255))").Error)
	require.NoError(t, db.Exec("INSERT INTO run_details VALUES ('run', 'schedule')").Error)
	for range 2 {
		require.NoError(t, ensureRecurringRunIndex(db))
	}
	var id string
	require.NoError(t, db.Raw("SELECT JobUUID FROM run_details WHERE UUID = 'run'").Scan(&id).Error)
	require.Equal(t, "schedule", id)
	var columnType string
	require.NoError(t, db.Raw("SELECT type FROM pragma_table_info('run_details') WHERE name = 'JobUUID'").Scan(&columnType).Error)
	require.Equal(t, "varchar(255)", columnType)
}

func TestRecurringRunIndexRejectsConflict(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	require.NoError(t, db.Exec("CREATE TABLE run_details (UUID text PRIMARY KEY, JobUUID varchar(255))").Error)
	require.NoError(t, db.Exec("CREATE INDEX idx_run_details_job_uuid ON run_details(UUID)").Error)
	require.ErrorContains(t, ensureRecurringRunIndex(db), "unexpected definition")
	var column string
	require.NoError(t, db.Raw("SELECT name FROM pragma_index_info('idx_run_details_job_uuid')").Scan(&column).Error)
	require.Equal(t, "UUID", column, "conflicting indexes must never be replaced")
}

func TestRecurringRunIndexOnlineDDL(t *testing.T) {
	require.Equal(t, "ALTER TABLE run_details ADD INDEX idx_run_details_job_uuid (JobUUID(191)), ALGORITHM=INPLACE, LOCK=NONE", recurringRunIndexDDL("mysql"))
	require.Equal(t, `CREATE INDEX CONCURRENTLY "idx_run_details_job_uuid" ON "run_details" ("JobUUID")`, recurringRunIndexDDL("postgres"))
}
