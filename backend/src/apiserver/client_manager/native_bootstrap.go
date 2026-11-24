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
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
)

// The empty marker table records intent before the first native model DDL.
// Its existence survives MySQL's implicit DDL commits and process interruption.
const nativeBootstrapMarker = "kfp_native_bootstrap_pending"

func bootstrapNativeSchema(ctx context.Context, database *gorm.DB, dialect string) (bool, error) {
	bootstrapped := false
	err := database.WithContext(ctx).Connection(func(db *gorm.DB) error {
		connection := db.Statement.ConnPool.(*sql.Conn)
		models := model.AllModels()
		ownedTables := make(map[string]bool, len(models))
		schemas := make([]*gorm.Statement, 0, len(models))
		for _, entry := range models {
			stmt := &gorm.Statement{DB: db}
			if err := stmt.Parse(entry); err != nil {
				return err
			}
			ownedTables[stmt.Schema.Table] = true
			schemas = append(schemas, stmt)
		}
		inspect := func() ([]string, bool, error) {
			tables, err := db.Migrator().GetTables()
			if err != nil {
				return nil, false, fmt.Errorf("inspect KFP bootstrap catalog: %w", err)
			}
			if slices.Contains(tables, nativeBootstrapMarker) {
				return tables, true, nil
			}
			for _, table := range tables {
				if !ownedTables[table] {
					continue
				}
				state, err := mlmd.ClassifyStartup(ctx, connection, dialect)
				if err != nil || state != mlmd.Fresh {
					return tables, false, err
				}
				return nil, false, fmt.Errorf("unmarked KFP table %s prevents native bootstrap; use a new empty database or restore a verified backup", table)
			}
			return tables, true, nil
		}
		_, needed, err := inspect()
		if err != nil || !needed {
			return err
		}
		unlock, err := lockNativeBootstrap(ctx, connection, dialect)
		if err != nil {
			return err
		}
		defer unlock()
		tables, needed, err := inspect()
		if err != nil || !needed {
			return err
		}
		marked := slices.Contains(tables, nativeBootstrapMarker)
		for _, object := range []string{"tasks_pre_migration", "tasks_migration_tmp", "mlmd_migration_state",
			"mlmd_migration_complete", "transfer_identities_pre_migration", "transfer_identities_migration_tmp", "recurring_run_adoptions"} {
			if slices.Contains(tables, object) {
				return fmt.Errorf("existing KFP table %s prevents native bootstrap; inspect or restore the database", object)
			}
		}
		for i, stmt := range schemas {
			if !slices.Contains(tables, stmt.Schema.Table) {
				continue
			}
			// A CREATE TABLE in the bootstrap creates all model columns at once.
			// Index creation can be interrupted independently, so GORM repairs indexes.
			columns, err := db.Migrator().ColumnTypes(models[i])
			if err != nil {
				return err
			}
			if len(columns) != len(stmt.Schema.DBNames) {
				return fmt.Errorf("unexpected column count in native bootstrap table %s", stmt.Schema.Table)
			}
			for _, column := range columns {
				if !slices.Contains(stmt.Schema.DBNames, column.Name()) {
					return fmt.Errorf("unexpected column %s in native bootstrap table %s", column.Name(), stmt.Schema.Table)
				}
			}
		}
		if !marked {
			if err := db.Exec("CREATE TABLE " + nativeBootstrapMarker + " (id integer PRIMARY KEY)").Error; err != nil {
				return err
			}
		}
		if err := autoMigrate(db); err != nil {
			return err
		}
		state, err := mlmd.ClassifyStartup(ctx, connection, dialect)
		if err != nil {
			return err
		}
		if state != mlmd.Native {
			return fmt.Errorf("native bootstrap did not produce a complete KFP schema")
		}
		if err := db.Exec("DROP TABLE " + nativeBootstrapMarker).Error; err != nil {
			return err
		}
		bootstrapped = true
		return nil
	})
	return bootstrapped, err
}

func lockNativeBootstrap(ctx context.Context, connection *sql.Conn, dialect string) (func(), error) {
	var acquire, release string
	if dialect == "mysql" {
		const key = "SHA2(CONCAT('kfp_native_bootstrap:', DATABASE()), 256)"
		acquire = "SELECT GET_LOCK(" + key + ", 10) = 1"
		release = "SELECT RELEASE_LOCK(" + key + ") = 1"
	} else {
		const key = "hashtext(current_database()), hashtext('kfp_native_bootstrap:' || current_schema())"
		acquire = "SELECT pg_try_advisory_lock(" + key + ")"
		release = "SELECT pg_advisory_unlock(" + key + ")"
	}
	var locked sql.NullBool
	if err := connection.QueryRowContext(ctx, acquire).Scan(&locked); err != nil {
		_ = connection.Raw(func(any) error { return driver.ErrBadConn })
		return nil, fmt.Errorf("acquire KFP native bootstrap lock: %w", err)
	}
	if !locked.Bool {
		return nil, fmt.Errorf("another KFP replica is initializing the database; retry startup")
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released sql.NullBool
		if err := connection.QueryRowContext(releaseCtx, release).Scan(&released); err != nil || !released.Bool {
			_ = connection.Raw(func(any) error { return driver.ErrBadConn })
		}
	}, nil
}
