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
	"strings"
	"time"
)

type transferSQL interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func transferTableExists(ctx context.Context, db transferSQL, table string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table).Scan(&count)
	return count > 0, err
}

func transferColumnExists(ctx context.Context, db transferSQL, table, column string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?", table, column).Scan(&count)
	return count > 0, err
}

func qualifyTransferColumn(ctx context.Context, db transferSQL, table, column, expectedType string) error {
	var actualType string
	if err := db.QueryRowContext(ctx, "SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?", table, column).Scan(&actualType); err != nil {
		return fmt.Errorf("inspect %s.%s: %w", table, column, err)
	}
	if !strings.EqualFold(actualType, expectedType) {
		return fmt.Errorf("%s.%s has type %s, expected %s; refusing unqualified transfer identity schema", table, column, actualType, expectedType)
	}
	return nil
}

func qualifyTransferKey(ctx context.Context, db transferSQL, table, key string) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME='PRIMARY' AND COLUMN_NAME=?", table, key).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%s has no primary key on %s; refusing unqualified transfer identity schema", table, key)
	}
	return nil
}

func qualifyTransferTable(ctx context.Context, db transferSQL, table, key, keyType, uuidType string) error {
	for _, column := range []struct{ name, typ string }{{key, keyType}, {"UUID", uuidType}} {
		if err := qualifyTransferColumn(ctx, db, table, column.name, column.typ); err != nil {
			return err
		}
	}
	if err := qualifyTransferKey(ctx, db, table, key); err != nil {
		return err
	}
	var columns, indexes int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table).Scan(&columns); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", table).Scan(&indexes); err != nil {
		return err
	}
	if columns != 2 || indexes != 1 {
		return fmt.Errorf("%s has unexpected columns or indexes; refusing unqualified transfer identity schema", table)
	}
	return nil
}

// needsTransferIdentityUpgrade detects the 2.18 installation identity layout.
// This table is reconciled only during the MLMD maintenance cutover.
func needsTransferIdentityUpgrade(ctx context.Context, db *sql.DB) (bool, error) {
	complete, err := transferTableExists(ctx, db, "mlmd_migration_complete")
	if err != nil || !complete {
		return false, err
	}
	legacy, err := transferColumnExists(ctx, db, "transfer_identities", "ID")
	if err != nil {
		return false, err
	}
	backup, err := transferTableExists(ctx, db, "transfer_identities_pre_migration")
	return legacy || backup, err
}

func reconcileLegacyTransferIdentity(ctx context.Context, db *sql.DB) error {
	connection, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire transfer identity migration connection: %w", err)
	}
	defer connection.Close()
	var locked int
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK('kfp_mlmd_migration_lock', 10)").Scan(&locked); err != nil {
		return fmt.Errorf("acquire MLMD migration lock for transfer identity: %w", err)
	}
	if locked != 1 {
		return fmt.Errorf("MLMD migration lock is held by another replica; retry startup")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released int
		if err := connection.QueryRowContext(releaseCtx, "SELECT RELEASE_LOCK('kfp_mlmd_migration_lock')").Scan(&released); err != nil || released != 1 {
			_ = connection.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	return reconcileLegacyTransferIdentityLocked(ctx, connection)
}

func reconcileLegacyTransferIdentityLocked(ctx context.Context, db transferSQL) error {
	legacy, err := transferColumnExists(ctx, db, "transfer_identities", "ID")
	if err != nil {
		return err
	}
	native, err := transferColumnExists(ctx, db, "transfer_identities", "Key")
	if err != nil {
		return err
	}
	if legacy && native {
		return fmt.Errorf("transfer_identities has both legacy and native keys; restore the database backup before retrying")
	}
	backup, err := transferTableExists(ctx, db, "transfer_identities_pre_migration")
	if err != nil {
		return err
	}
	if !legacy && !native {
		if backup {
			return fmt.Errorf("transfer identity backup exists without a current table; restore the database backup")
		}
		return nil
	}
	if legacy && backup {
		return fmt.Errorf("legacy transfer_identities and its backup both exist; inspect the partial upgrade")
	}
	if legacy {
		if err := qualifyTransferTable(ctx, db, "transfer_identities", "ID", "tinyint unsigned", "varchar(36)"); err != nil {
			return err
		}
		var unexpected int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfer_identities WHERE ID<>1").Scan(&unexpected); err != nil {
			return err
		}
		if unexpected > 0 {
			return fmt.Errorf("transfer_identities has %d unrecognized IDs; preserve them before upgrading", unexpected)
		}
		const create = "CREATE TABLE IF NOT EXISTS transfer_identities_migration_tmp (`Key` varchar(32) NOT NULL PRIMARY KEY, `UUID` varchar(64) NOT NULL) ENGINE=InnoDB"
		if _, err := db.ExecContext(ctx, create); err != nil {
			return err
		}
		if err := qualifyTransferTable(ctx, db, "transfer_identities_migration_tmp", "Key", "varchar(32)", "varchar(64)"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "INSERT IGNORE INTO transfer_identities_migration_tmp (`Key`,`UUID`) SELECT 'installation', UUID FROM transfer_identities WHERE ID=1"); err != nil {
			return err
		}
		var sourceUUID, copiedUUID sql.NullString
		if err := db.QueryRowContext(ctx, "SELECT UUID FROM transfer_identities WHERE ID=1").Scan(&sourceUUID); err != nil && err != sql.ErrNoRows {
			return err
		}
		if err := db.QueryRowContext(ctx, "SELECT UUID FROM transfer_identities_migration_tmp WHERE `Key`='installation'").Scan(&copiedUUID); err != nil && err != sql.ErrNoRows {
			return err
		}
		if sourceUUID != copiedUUID {
			return fmt.Errorf("transfer identity copy does not match the released source; inspect the partial upgrade")
		}
		var copiedRows int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfer_identities_migration_tmp").Scan(&copiedRows); err != nil {
			return err
		}
		if copiedRows > 1 {
			return fmt.Errorf("transfer identity copy contains unexpected rows; inspect the partial upgrade")
		}
		if _, err := db.ExecContext(ctx, "RENAME TABLE transfer_identities TO transfer_identities_pre_migration, transfer_identities_migration_tmp TO transfer_identities"); err != nil {
			return err
		}
	}
	backup, err = transferTableExists(ctx, db, "transfer_identities_pre_migration")
	if err != nil || !backup {
		return err
	}
	if err := qualifyTransferTable(ctx, db, "transfer_identities", "Key", "varchar(32)", "varchar(64)"); err != nil {
		return err
	}
	var unexpected int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfer_identities_pre_migration WHERE ID<>1").Scan(&unexpected); err != nil {
		return err
	}
	if unexpected > 0 {
		return fmt.Errorf("transfer identity backup has %d unrecognized IDs; inspect the partial upgrade", unexpected)
	}
	var oldUUID, currentUUID sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT UUID FROM transfer_identities_pre_migration WHERE ID=1").Scan(&oldUUID); err != nil && err != sql.ErrNoRows {
		return err
	}
	if err := db.QueryRowContext(ctx, "SELECT UUID FROM transfer_identities WHERE `Key`='installation'").Scan(&currentUUID); err != nil && err != sql.ErrNoRows {
		return err
	}
	if oldUUID != currentUUID {
		return fmt.Errorf("transfer identity differs after table rename; inspect the partial upgrade")
	}
	_, err = db.ExecContext(ctx, "DROP TABLE transfer_identities_pre_migration")
	return err
}
