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
	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// preserveRunJobUUID scopes this exception to schema migration, leaving the
// caller's connection and dialect unchanged. Existing installations can have
// varchar(255), varchar(191), or text here. Reconcile neither their type nor
// length: changing it can copy a large run_details table. Missing columns still
// use the model definition, and all other columns retain normal reconciliation.
func preserveRunJobUUID(db *gorm.DB) *gorm.DB {
	migrationDB := db.Session(&gorm.Session{})
	migrationDB.Dialector = jobUUIDPreservingDialector{Dialector: db.Dialector}
	return migrationDB
}

type jobUUIDPreservingDialector struct {
	gorm.Dialector
}

func (d jobUUIDPreservingDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return jobUUIDPreservingMigrator{Migrator: d.Dialector.Migrator(db)}
}

type jobUUIDPreservingMigrator struct {
	gorm.Migrator
}

func (m jobUUIDPreservingMigrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	if field.Schema.Table == "run_details" && field.DBName == "JobUUID" {
		return nil
	}
	return m.Migrator.MigrateColumn(value, field, columnType)
}

// GORM uses this additional interface when creating tables with indexes.
func (m jobUUIDPreservingMigrator) BuildIndexOptions(options []schema.IndexOption, stmt *gorm.Statement) []interface{} {
	return m.Migrator.(migrator.BuildIndexOptionsInterface).BuildIndexOptions(options, stmt)
}
