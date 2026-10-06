// Copyright 2026 The Kubeflow Authors
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

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecurringRunIndexMigration(t *testing.T) {
	orm, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db, err := orm.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))
	testRecurringRunIndexMigration(t, orm)
}

// Exercise both existing installations and repeated startup migrations. The
// production-database suite also calls this against MySQL and PostgreSQL.
func testRecurringRunIndexMigration(t *testing.T, orm *gorm.DB) {
	t.Helper()
	const index = "idx_run_details_job_uuid"
	require.True(t, orm.Migrator().HasIndex(&model.Run{}, index))
	if orm.Name() == "postgres" {
		// GORM's DropIndex emits invalid CURRENT_SCHEMA().index SQL when the
		// connection uses an isolated search_path; resolve through that path.
		require.NoError(t, orm.Exec(`DROP INDEX "idx_run_details_job_uuid"`).Error)
	} else {
		require.NoError(t, orm.Migrator().DropIndex(&model.Run{}, index))
	}
	if orm.Name() == "mysql" {
		require.NoError(t, orm.Exec("ALTER TABLE run_details MODIFY COLUMN JobUUID longtext DEFAULT NULL").Error)
	} else if orm.Name() == "postgres" {
		require.NoError(t, orm.Exec(`ALTER TABLE run_details ALTER COLUMN "JobUUID" TYPE text`).Error)
	}
	run := &model.Run{UUID: "index-migration-run", RecurringRunId: "index-migration-job"}
	require.NoError(t, orm.Create(run).Error)
	for range 2 {
		require.NoError(t, orm.AutoMigrate(model.AllModels()...))
		require.True(t, orm.Migrator().HasIndex(&model.Run{}, index))
		var stored model.Run
		require.NoError(t, orm.First(&stored, &model.Run{UUID: run.UUID}).Error)
		require.Equal(t, run.RecurringRunId, stored.RecurringRunId)
	}
	require.NoError(t, orm.Delete(run).Error)
}
