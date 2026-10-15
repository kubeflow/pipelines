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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func TestPreserveRunJobUUIDMigration(t *testing.T) {
	for _, existingType := range []string{"varchar(255)", "varchar(191)", "text"} {
		t.Run(existingType, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			defer sqlDB.Close()
			require.NoError(t, db.Exec(fmt.Sprintf("CREATE TABLE run_details (UUID varchar(64) PRIMARY KEY, JobUUID %s DEFAULT NULL)", existingType)).Error)
			originalDialect := db.Dialector
			migrationDB := preserveRunJobUUID(db)
			require.Same(t, originalDialect, db.Dialector, "must not wrap the caller's dialect")
			// Include all models and associations: dependency ordering must not reintroduce
			// migration of the original Run field through another model's relationship.
			require.NoError(t, migrationDB.AutoMigrate(model.AllModels()...))
			require.NoError(t, migrationDB.AutoMigrate(model.AllModels()...))
			var columnType string
			require.NoError(t, db.Raw("SELECT type FROM pragma_table_info('run_details') WHERE name = 'JobUUID'").Scan(&columnType).Error)
			require.Equal(t, existingType, strings.ToLower(columnType))
			require.True(t, db.Migrator().HasColumn(&model.Run{}, "PipelineVersionId"))
			require.True(t, db.Migrator().HasTable(&model.RunMetric{}))
		})
	}
}

func TestPreserveRunJobUUIDCreatesMissingColumn(t *testing.T) {
	for _, existingTable := range []bool{false, true} {
		t.Run(fmt.Sprint(existingTable), func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			defer sqlDB.Close()
			if existingTable {
				require.NoError(t, db.Exec("CREATE TABLE run_details (UUID varchar(64) PRIMARY KEY)").Error)
			}
			require.NoError(t, preserveRunJobUUID(db).AutoMigrate(model.AllModels()...))
			var columnType string
			require.NoError(t, db.Raw("SELECT type FROM pragma_table_info('run_details') WHERE name = 'JobUUID'").Scan(&columnType).Error)
			require.Equal(t, "varchar(255)", columnType)
		})
	}
}

// The exception must never disable unrelated schema reconciliation.
func TestJobUUIDPreservationDelegatesOtherColumns(t *testing.T) {
	for _, tc := range []struct {
		table, column string
		preserved     bool
	}{
		{"run_details", "JobUUID", true},
		{"run_details", "PipelineId", false},
		{"jobs", "JobUUID", false},
	} {
		t.Run(tc.table+"/"+tc.column, func(t *testing.T) {
			sentinel := errors.New("underlying migrator result")
			delegate := &recordingColumnMigrator{err: sentinel}
			migrator := jobUUIDPreservingMigrator{Migrator: delegate}
			field := &schema.Field{DBName: tc.column, Schema: &schema.Schema{Table: tc.table}}
			err := migrator.MigrateColumn(nil, field, nil)
			if tc.preserved {
				require.NoError(t, err)
				require.False(t, delegate.called)
			} else {
				require.ErrorIs(t, err, sentinel)
				require.True(t, delegate.called)
			}
		})
	}
}

type recordingColumnMigrator struct {
	gorm.Migrator
	called bool
	err    error
}

func (m *recordingColumnMigrator) MigrateColumn(_ interface{}, _ *schema.Field, _ gorm.ColumnType) error {
	m.called = true
	return m.err
}
