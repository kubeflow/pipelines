// Copyright 2026 The Kubeflow Authors
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

package clientmanager

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
)

const runLeaseDiscoveryIndex = "idx_run_runtime_lease_discovery"

func createRunLeaseSchema(db *gorm.DB) error {
	exists, err := runtimeSchemaHasTable(db, "run_leases")
	if err != nil {
		return err
	}
	if !exists {
		create := func(connection *gorm.DB) error {
			if err := connection.Migrator().CreateTable(&model.RunLease{}); err != nil {
				return err
			}
			return validateRunLeaseSchema(connection)
		}
		glog.Info("runtime_schema creating empty run_leases table with retained generations")
		if db.Name() == "postgres" {
			err = db.Transaction(create)
		} else {
			err = create(db)
		}
		if err != nil {
			return fmt.Errorf("create empty run_leases table (no implicit repair): %w", err)
		}
	} else if err := validateRunLeaseSchema(db); err != nil {
		return err
	}
	return createRunLeaseDiscoveryIndex(db)
}

func validateRunLeaseSchema(db *gorm.DB) error {
	columns, err := db.Migrator().ColumnTypes(&model.RunLease{})
	if err != nil {
		return err
	}
	found := make(map[string]gorm.ColumnType, len(columns))
	for _, column := range columns {
		found[column.Name()] = column
	}
	for _, spec := range []struct {
		name     string
		kind     string
		nullable bool
	}{
		{"RunUUID", "varchar", false}, {"OwnerID", "varchar", true},
		{"Generation", "integer", false}, {"ExpiresAt", "integer", true},
	} {
		column, ok := found[spec.name]
		if !ok {
			return fmt.Errorf("run_leases.%s is missing; refusing implicit schema repair", spec.name)
		}
		kind := strings.ToLower(column.DatabaseTypeName())
		validKind := kind == spec.kind || (spec.kind == "integer" && (kind == "bigint" || kind == "int8"))
		nullable, known := column.Nullable()
		if !validKind || !known || nullable != spec.nullable {
			return fmt.Errorf("run_leases.%s has incompatible type/nullability; refusing implicit schema repair", spec.name)
		}
		if spec.kind == "varchar" {
			length, known := column.Length()
			if !known || length != 191 {
				return fmt.Errorf("run_leases.%s must remain VARCHAR(191); refusing implicit conversion", spec.name)
			}
		}
		primary, known := column.PrimaryKey()
		if !known || primary != (spec.name == "RunUUID") {
			return errors.New("run_leases must have RunUUID as its sole primary key; refusing implicit repair")
		}
	}
	if err := requireRuntimeIndex(db, &model.RunLease{}, "idx_run_leases_expiry_run", "ExpiresAt,RunUUID", false); err != nil {
		return err
	}
	if !db.Migrator().HasConstraint(&model.RunLease{}, "fk_run_leases_run") {
		return errors.New("run_leases requires its cascading run foreign key; refusing implicit schema repair")
	}
	return validateRuntimeTableForeignKeys(db, "run_leases", []runtimeForeignKey{
		{Columns: "RunUUID", ReferenceTable: "run_details", ReferenceColumns: "UUID", DeleteRule: "CASCADE", UpdateRule: "CASCADE"},
	})
}

func createRunLeaseDiscoveryIndex(db *gorm.DB) error {
	var count int64
	var inventory string
	switch db.Name() {
	case "mysql":
		inventory = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'run_details' AND index_name = ?"
	case "postgres":
		inventory = "SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = current_schema() AND c.relname = ? AND c.relkind IN ('i','I')"
	case "sqlite":
		inventory = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?"
	default:
		return fmt.Errorf("unsupported lease discovery index dialect %q", db.Name())
	}
	if err := db.Raw(inventory, runLeaseDiscoveryIndex).Scan(&count).Error; err != nil {
		return fmt.Errorf("inventory run lease discovery index: %w", err)
	}
	if count > 0 {
		// Incompatible or invalid indexes are never dropped/replaced: an
		// external concurrent build may still own them.
		return requireRuntimeIndex(db, &model.Run{}, runLeaseDiscoveryIndex, "ExecutorType,State,CreatedAtInSec,UUID", false)
	}
	q := func(name string) string { return runtimeSchemaQuote(db, name) }
	columns := q("ExecutorType") + ", " + q("State") + ", " + q("CreatedAtInSec") + ", " + q("UUID")
	var query string
	switch db.Name() {
	case "mysql":
		query = "ALTER TABLE " + q("run_details") + " ADD INDEX " + q(runLeaseDiscoveryIndex) + " (" + columns + "), ALGORITHM=INPLACE, LOCK=NONE"
	case "postgres":
		query = "CREATE INDEX CONCURRENTLY " + q(runLeaseDiscoveryIndex) + " ON " + q("run_details") + " (" + columns + ")"
	case "sqlite":
		query = "CREATE INDEX " + q(runLeaseDiscoveryIndex) + " ON " + q("run_details") + " (" + columns + ")"
	}
	glog.Infof("runtime_schema creating online lease discovery index on %s (5s metadata-lock wait; no algorithm fallback)", db.Name())
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("create online lease discovery index without fallback; resolve contention or inspect an interrupted concurrent build before restarting: %w", err)
	}
	return requireRuntimeIndex(db, &model.Run{}, runLeaseDiscoveryIndex, "ExecutorType,State,CreatedAtInSec,UUID", false)
}
