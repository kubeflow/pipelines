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
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"gorm.io/gorm"
)

const recurringRunIndexName = "idx_run_details_job_uuid"

// ensureRecurringRunIndex never lets the server choose a copying/blocking
// fallback. The prefix covers ordinary UUIDs and supports historical TEXT and
// varchar columns without changing their stored values. Queries still compare
// the complete JobUUID, so longer values sharing a prefix remain distinct.
// Builds consume IO and briefly acquire metadata locks; they are not free of
// operational impact. Other startup schema migrations have their own behavior.
func ensureRecurringRunIndex(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(db.Statement.Context, 30*time.Minute)
	defer cancel()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	dialect := db.Name()
	if dialect != "mysql" && dialect != "postgres" && dialect != "sqlite" {
		return fmt.Errorf("cannot prepare %s online for unsupported database %q", recurringRunIndexName, dialect)
	}
	cleanup, err := lockRecurringRunIndex(ctx, conn, dialect)
	if err != nil {
		return fmt.Errorf("coordinate %s preparation: %w", recurringRunIndexName, err)
	}
	defer cleanup()
	exists, err := recurringRunIndexReady(ctx, conn, dialect)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	started := time.Now()
	glog.Infof("Preparing %s online on %s; existing JobUUID type and width are preserved", recurringRunIndexName, dialect)
	if _, err := conn.ExecContext(ctx, recurringRunIndexDDL(dialect)); err != nil {
		return fmt.Errorf("online creation of %s failed; no copying or write-blocking fallback was attempted; resolve the database lock, permission, or online-DDL limitation and retry startup: %w", recurringRunIndexName, err)
	}
	ready, err := recurringRunIndexReady(ctx, conn, dialect)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("%s is missing after online creation", recurringRunIndexName)
	}
	glog.Infof("Prepared %s online in %s", recurringRunIndexName, time.Since(started))
	return nil
}

func recurringRunIndexDDL(dialect string) string {
	switch dialect {
	case "mysql":
		return "ALTER TABLE run_details ADD INDEX idx_run_details_job_uuid (JobUUID(191)), ALGORITHM=INPLACE, LOCK=NONE"
	case "postgres":
		return `CREATE INDEX CONCURRENTLY "idx_run_details_job_uuid" ON "run_details" ("JobUUID")`
	default:
		return `CREATE INDEX "idx_run_details_job_uuid" ON "run_details" ("JobUUID")`
	}
}

// Session locks must stay on the same physical connection as the DDL. Cleanup
// uses a fresh deadline so a canceled build cannot strand a lock or modified
// session setting in the pool. A failed cleanup discards the connection.
func lockRecurringRunIndex(ctx context.Context, conn *sql.Conn, dialect string) (func(), error) {
	noop := func() {}
	if dialect == "sqlite" {
		return noop, nil
	}
	var releaseSQL string
	var originalLockTimeout int64
	switch dialect {
	case "mysql":
		var acquired sql.NullInt64
		// Include the current database: separate KFP databases on one server should
		// not serialize unrelated builds. Keep the lock name below MySQL's limit.
		err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(CONCAT('kfp:job-index:', LEFT(SHA2(DATABASE(), 256), 40)), 10)`).Scan(&acquired)
		if err != nil {
			return nil, err
		}
		if !acquired.Valid || acquired.Int64 != 1 {
			return nil, fmt.Errorf("another replica is preparing %s; retry startup after its build completes", recurringRunIndexName)
		}
		releaseSQL = `SELECT RELEASE_LOCK(CONCAT('kfp:job-index:', LEFT(SHA2(DATABASE(), 256), 40)))`
	case "postgres":
		var acquired bool
		if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext(current_database()), hashtext(current_schema() || ':kfp:job-index'))`).Scan(&acquired); err != nil {
			return nil, err
		}
		if !acquired {
			return nil, fmt.Errorf("another replica is preparing %s; retry startup after its build completes", recurringRunIndexName)
		}
		releaseSQL = `SELECT pg_advisory_unlock(hashtext(current_database()), hashtext(current_schema() || ':kfp:job-index'))`
	}
	restoreSQL := ""
	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var cleanupErr error
		if restoreSQL != "" {
			_, cleanupErr = conn.ExecContext(cleanupCtx, restoreSQL)
		}
		if _, err := conn.ExecContext(cleanupCtx, releaseSQL); err != nil {
			cleanupErr = err
		}
		if cleanupErr != nil {
			// database/sql removes connections when Raw returns driver.ErrBadConn.
			_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
		}
	}
	if dialect == "mysql" {
		if err := conn.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&originalLockTimeout); err != nil {
			cleanup()
			return nil, err
		}
		restoreSQL = fmt.Sprintf("SET SESSION lock_wait_timeout = %d", originalLockTimeout)
		if _, err := conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = 5"); err != nil {
			cleanup()
			return nil, err
		}
	} else {
		var original string
		if err := conn.QueryRowContext(ctx, "SHOW lock_timeout").Scan(&original); err != nil {
			cleanup()
			return nil, err
		}
		// set_config accepts a bound value; save restoration as a separately quoted
		// server-returned setting using PostgreSQL's quote_literal.
		var quoted string
		if err := conn.QueryRowContext(ctx, "SELECT quote_literal($1)", original).Scan(&quoted); err != nil {
			cleanup()
			return nil, err
		}
		restoreSQL = "SET lock_timeout = " + quoted
		if _, err := conn.ExecContext(ctx, "SET lock_timeout = '5s'"); err != nil {
			cleanup()
			return nil, err
		}
	}
	return cleanup, nil
}

func recurringRunIndexReady(ctx context.Context, conn *sql.Conn, dialect string) (bool, error) {
	conflict := func() (bool, error) {
		return false, fmt.Errorf("%s exists but is invalid or has an unexpected definition; inspect and repair this index before retrying startup; it was not dropped or replaced", recurringRunIndexName)
	}
	switch dialect {
	case "mysql":
		rows, err := conn.QueryContext(ctx, `SELECT COLUMN_NAME, SUB_PART, NON_UNIQUE, INDEX_TYPE FROM information_schema.statistics WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'run_details' AND INDEX_NAME = 'idx_run_details_job_uuid' ORDER BY SEQ_IN_INDEX`)
		if err != nil {
			return false, err
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			count++
			var column, kind string
			var prefix sql.NullInt64
			var nonUnique int
			if err := rows.Scan(&column, &prefix, &nonUnique, &kind); err != nil {
				return false, err
			}
			if count != 1 || column != "JobUUID" || nonUnique != 1 || kind != "BTREE" || (prefix.Valid && prefix.Int64 < 191) {
				return conflict()
			}
		}
		return count == 1, rows.Err()
	case "postgres":
		var valid bool
		err := conn.QueryRowContext(ctx, `SELECT i.indisvalid AND i.indisready AND NOT i.indisunique AND i.indnatts = 1 AND i.indpred IS NULL AND i.indexprs IS NULL AND t.relname = 'run_details' AND a.attname = 'JobUUID' AND am.amname = 'btree'
   FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
   JOIN pg_index i ON i.indexrelid = c.oid JOIN pg_class t ON t.oid = i.indrelid
   JOIN pg_am am ON am.oid = c.relam
   LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = i.indkey[0]
   WHERE n.nspname = current_schema() AND c.relname = 'idx_run_details_job_uuid'`).Scan(&valid)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !valid {
			return conflict()
		}
		return true, nil
	default:
		var count, unique, partial int
		err := conn.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX("unique"), 0), COALESCE(MAX(partial), 0) FROM pragma_index_list('run_details') WHERE name = 'idx_run_details_job_uuid'`).Scan(&count, &unique, &partial)
		if err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
		var columns int
		var column string
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(name), '') FROM pragma_index_info('idx_run_details_job_uuid')`).Scan(&columns, &column); err != nil {
			return false, err
		}
		if unique != 0 || partial != 0 || columns != 1 || column != "JobUUID" {
			return conflict()
		}
		return true, nil
	}
}
