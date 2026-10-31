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

package clientmanager

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
)

const taskLogHistoryMigrationLock = "kfp_task_log_history_v1"

// createTaskLogHistory creates only the new standalone table and its index. It
// never reconciles an existing table, adds foreign keys, or backfills old rows.
// Startup fails on contention rather than waiting behind an unbounded DDL lock.
func createTaskLogHistory(db *gorm.DB) (result error) {
	ctx, cancel := context.WithTimeout(db.Statement.Context, 10*time.Second)
	defer cancel()
	if db.Name() == "sqlite" {
		if db.Migrator().HasTable(&model.TaskLogHistory{}) {
			return nil
		}
		return db.WithContext(ctx).Migrator().CreateTable(&model.TaskLogHistory{})
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	session := db.Session(&gorm.Session{NewDB: true}).WithContext(ctx)
	session.Statement.ConnPool = conn

	// Cleanup uses a fresh bounded context even if startup was canceled. If
	// session settings/ownership cannot be restored, discard this connection.
	var cleanup func(context.Context) error
	defer func() {
		if cleanup == nil {
			return
		}
		cleanupCtx, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		if err := cleanup(cleanupCtx); err != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			result = errors.Join(result, fmt.Errorf("release retry log migration session: %w", err))
		}
	}()
	switch db.Name() {
	case "mysql":
		cleanup = func(ctx context.Context) error {
			var released sql.NullInt64
			return conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(SHA2(CONCAT(DATABASE(), ?), 256))", taskLogHistoryMigrationLock).Scan(&released)
		}
		var acquired int
		if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(SHA2(CONCAT(DATABASE(), ?), 256), 0)", taskLogHistoryMigrationLock).Scan(&acquired); err != nil {
			return err
		}
		if acquired != 1 {
			return fmt.Errorf("retry log history migration is owned by another API server; retry startup")
		}
		var previous int
		if err := conn.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&previous); err != nil {
			return err
		}
		cleanup = func(ctx context.Context) error {
			_, restore := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION lock_wait_timeout = %d", previous))
			var released int
			release := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(SHA2(CONCAT(DATABASE(), ?), 256))", taskLogHistoryMigrationLock).Scan(&released)
			return errors.Join(restore, release)
		}
		if _, err := conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = 1"); err != nil {
			return err
		}
	case "postgres":
		cleanup = func(ctx context.Context) error {
			var released bool
			return conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock(hashtext($1))", taskLogHistoryMigrationLock).Scan(&released)
		}
		var acquired bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", taskLogHistoryMigrationLock).Scan(&acquired); err != nil {
			return err
		}
		if !acquired {
			return fmt.Errorf("retry log history migration is owned by another API server; retry startup")
		}
		var previous string
		if err := conn.QueryRowContext(ctx, "SHOW lock_timeout").Scan(&previous); err != nil {
			return err
		}
		cleanup = func(ctx context.Context) error {
			_, restore := conn.ExecContext(ctx, "SELECT set_config('lock_timeout', $1, false)", previous)
			var released bool
			release := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock(hashtext($1))", taskLogHistoryMigrationLock).Scan(&released)
			return errors.Join(restore, release)
		}
		if _, err := conn.ExecContext(ctx, "SET lock_timeout = '1s'"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("retry log history migration does not support database %q", db.Name())
	}
	if session.Migrator().HasTable(&model.TaskLogHistory{}) {
		return nil
	}
	create := func(tx *gorm.DB) error { return tx.Migrator().CreateTable(&model.TaskLogHistory{}) }
	if db.Name() == "postgres" {
		err = session.Transaction(create)
	} else {
		err = create(session)
	}
	if err != nil {
		return fmt.Errorf("create retry log history table (bounded lock wait; retry startup after resolving contention): %w", err)
	}
	return nil
}
