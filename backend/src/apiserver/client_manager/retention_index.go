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

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GarbageCollectorIndexPreparer returns a background preparation step. It never
// archives or deletes runs; true reports that this invocation built an index.
func (c *ClientManager) GarbageCollectorIndexPreparer() func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		timeout, err := common.GetRunsGCIndexBuildTimeout()
		if err != nil {
			return false, err
		}
		return prepareGarbageCollectorIndexes(ctx, c.db, c.dbDialect, timeout)
	}
}

// Both builds share a session advisory lock, preventing API replicas from
// racing or interpreting another replica's in-progress PostgreSQL index as
// invalid. Each build gets its own deadline, separate from the short lock wait.
func prepareGarbageCollectorIndexes(ctx context.Context, db *sql.DB, dialect sqldrv.DBDialect, buildTimeout time.Duration) (bool, error) {
	if buildTimeout <= 0 {
		return false, fmt.Errorf("RUNS_GC_INDEX_BUILD_TIMEOUT must be positive")
	}
	if dialect.Name() != "mysql" && dialect.Name() != "pgx" {
		return false, fmt.Errorf("online retention index preparation is unsupported for database %q", dialect.Name())
	}
	setupCtx, cancelSetup := context.WithTimeout(ctx, 15*time.Second)
	defer cancelSetup()
	conn, err := db.Conn(setupCtx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	cleanup, err := lockGarbageCollectorIndexes(setupCtx, conn, dialect.Name())
	if err != nil {
		return false, err
	}
	defer cleanup()
	cancelSetup()
	var dialector gorm.Dialector
	if dialect.Name() == "mysql" {
		dialector = mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true})
	} else {
		dialector = postgres.New(postgres.Config{Conn: conn})
	}
	orm, err := gorm.Open(dialector, &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return false, err
	}
	built := false
	for _, spec := range garbageCollectorRequiredIndexes {
		created, err := prepareGarbageCollectorIndex(ctx, conn, orm, dialect, spec, buildTimeout)
		built = built || created
		if err != nil {
			return built, err
		}
	}
	return built, nil
}

func prepareGarbageCollectorIndex(ctx context.Context, conn *sql.Conn, db *gorm.DB, dialect sqldrv.DBDialect, spec garbageCollectorIndexSpec, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db = db.WithContext(ctx)
	ready, err := validateGarbageCollectorIndex(db, dialect, spec)
	if err != nil {
		return false, err
	}
	if ready {
		return false, nil
	}
	// The readiness validator intentionally conflates missing and unusable.
	// Distinguish them before DDL, including a same-name index on another PG table.
	var exists bool
	if dialect.Name() == "mysql" {
		err = conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.statistics WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'run_details' AND INDEX_NAME = ?)`, spec.name).Scan(&exists)
	} else {
		err = conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = current_schema() AND c.relname = $1)`, spec.name).Scan(&exists)
	}
	if err != nil {
		return false, err
	}
	if exists && dialect.Name() == "pgx" {
		if err := recoverPostgresGarbageCollectorIndex(ctx, conn, spec); err != nil {
			return false, err
		}
	} else if exists {
		return false, fmt.Errorf("retention index %s exists but is invalid or has an unexpected definition; GC remains paused; inspect and repair the index using docs/agents/development.md before retrying; it was not dropped or replaced", spec.name)
	}
	started := time.Now()
	glog.Infof("Preparing retention index %s online (build timeout %s); GC remains paused; index creation consumes database IO and may briefly wait for metadata locks", spec.name, timeout)
	if _, err := conn.ExecContext(ctx, garbageCollectorIndexDDL(dialect.Name(), spec)); err != nil {
		return false, fmt.Errorf("online creation of retention index %s failed; GC remains paused and no copying or write-blocking fallback was attempted; check locks, disk space, online-DDL support and permissions; after a timeout consider RUNS_GC_INDEX_BUILD_TIMEOUT, abandoned matching PostgreSQL indexes are recovered on a later attempt: %w", spec.name, err)
	}
	ready, err = validateGarbageCollectorIndex(db, dialect, spec)
	if err != nil {
		return true, err
	}
	if !ready {
		return true, fmt.Errorf("retention index %s is not usable after creation; GC remains paused; inspect the database index definition", spec.name)
	}
	glog.Infof("Prepared retention index %s online in %s; retention collection runs separately", spec.name, time.Since(started))
	return true, nil
}

// The two fixed retention-index names are reserved for the preparer. Only an
// invalid index with exactly the generated definition can be removed. The
// session advisory lock serializes KFP builders; the relation-lock check also
// defers to observed external maintenance. Administrators changing these indexes
// must stop preparation or take the same advisory lock: a catalog check cannot
// serialize arbitrary external DDL that starts after the check.
func recoverPostgresGarbageCollectorIndex(ctx context.Context, conn *sql.Conn, spec garbageCollectorIndexSpec) error {
	// Per-column pg_get_indexdef omits sort options, operator classes, and
	// collations, so validate those catalog fields separately before cleanup.
	var recoverable, active bool
	err := conn.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM pg_index i
  JOIN pg_class idx ON idx.oid = i.indexrelid
  JOIN pg_namespace ns ON ns.oid = idx.relnamespace
  JOIN pg_class tbl ON tbl.oid = i.indrelid
  JOIN pg_am am ON am.oid = idx.relam
  WHERE ns.nspname = current_schema() AND idx.relname = $1
    AND tbl.relnamespace = ns.oid AND tbl.relname = 'run_details'
    AND idx.relkind = 'i' AND tbl.relkind = 'r'
    AND idx.reloptions IS NULL AND idx.reltablespace = 0
    AND NOT idx.relispartition AND NOT tbl.relispartition
    AND NOT i.indisvalid AND NOT i.indisunique AND NOT i.indisprimary
    AND NOT i.indisexclusion AND NOT i.indisreplident
    AND i.indpred IS NULL AND i.indexprs IS NULL AND am.amname = 'btree'
    AND i.indnatts = 2 AND i.indnkeyatts = 2
    AND NOT EXISTS (SELECT 1 FROM unnest(i.indoption::smallint[]) AS index_option(value) WHERE value <> 0)
    AND NOT EXISTS (
      SELECT 1 FROM unnest(i.indkey::smallint[], i.indclass::oid[], i.indcollation::oid[]) AS index_key(attnum, opclass, collation_oid)
      JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = index_key.attnum
      JOIN pg_opclass opc ON opc.oid = index_key.opclass
      WHERE NOT opc.opcdefault OR opc.opcmethod <> am.oid OR index_key.collation_oid <> attr.attcollation
    )
    AND pg_get_indexdef(idx.oid, 1, false) || ',' || pg_get_indexdef(idx.oid, 2, false) = $2
    AND NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conindid = idx.oid)
    AND NOT EXISTS (SELECT 1 FROM pg_inherits h WHERE h.inhrelid IN (idx.oid, tbl.oid) OR h.inhparent IN (idx.oid, tbl.oid))
), EXISTS (
  SELECT 1 FROM pg_locks l JOIN pg_class c ON c.oid = l.relation
  JOIN pg_namespace n ON n.oid = c.relnamespace
  WHERE l.locktype = 'relation' AND l.database = (SELECT oid FROM pg_database WHERE datname = current_database())
    AND n.nspname = current_schema() AND c.relname IN ('run_details', $1)
    AND l.pid IS DISTINCT FROM pg_backend_pid()
    AND l.mode IN ('ShareUpdateExclusiveLock', 'ShareLock', 'ShareRowExclusiveLock', 'ExclusiveLock', 'AccessExclusiveLock')
)`, spec.name, `"`+strings.ReplaceAll(spec.columns, ",", `","`)+`"`).Scan(&recoverable, &active)
	if err != nil {
		return fmt.Errorf("inspect interrupted retention index %s: %w", spec.name, err)
	}
	if !recoverable {
		return fmt.Errorf("retention index %s exists but does not match the recoverable invalid index definition; GC remains paused; inspect the index using docs/agents/development.md; it was not dropped or replaced", spec.name)
	}
	if active {
		return fmt.Errorf("retention index %s recovery deferred because database maintenance is active or waiting; GC remains paused and will retry", spec.name)
	}
	glog.Warningf("Recovering interrupted PostgreSQL retention index %s using concurrent drop and rebuild; GC remains paused", spec.name)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP INDEX CONCURRENTLY "%s"`, spec.name)); err != nil {
		return fmt.Errorf("concurrent cleanup of interrupted retention index %s failed; GC remains paused and will retry: %w", spec.name, err)
	}
	return nil
}

// Names and columns come only from the fixed required-index specifications.
func garbageCollectorIndexDDL(dialect string, spec garbageCollectorIndexSpec) string {
	if dialect == "mysql" {
		return fmt.Sprintf("ALTER TABLE run_details ADD INDEX %s (%s), ALGORITHM=INPLACE, LOCK=NONE", spec.name, spec.columns)
	}
	return fmt.Sprintf(`CREATE INDEX CONCURRENTLY "%s" ON "run_details" ("%s")`, spec.name, strings.ReplaceAll(spec.columns, ",", `","`))
}

// Cleanup runs even after cancellation and uses the same physical connection.
// If cleanup cannot restore session state, discard it rather than return a
// held advisory lock or modified timeout to the pool.
func lockGarbageCollectorIndexes(ctx context.Context, conn *sql.Conn, dialect string) (func(), error) {
	var releaseSQL string
	if dialect == "mysql" {
		var acquired sql.NullInt64
		if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(CONCAT('kfp:retention-index:', LEFT(SHA2(DATABASE(), 256), 40)), 0)`).Scan(&acquired); err != nil {
			return nil, err
		}
		if !acquired.Valid || acquired.Int64 != 1 {
			return nil, fmt.Errorf("another replica is preparing retention indexes; GC remains paused until a later readiness check")
		}
		releaseSQL = `SELECT RELEASE_LOCK(CONCAT('kfp:retention-index:', LEFT(SHA2(DATABASE(), 256), 40)))`
	} else {
		var acquired bool
		if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext(current_database()), hashtext(current_schema() || ':kfp:retention-index'))`).Scan(&acquired); err != nil {
			return nil, err
		}
		if !acquired {
			return nil, fmt.Errorf("another replica is preparing retention indexes; GC remains paused until a later readiness check")
		}
		releaseSQL = `SELECT pg_advisory_unlock(hashtext(current_database()), hashtext(current_schema() || ':kfp:retention-index'))`
	}
	var restoreSQL string
	var restoreArgs []interface{}
	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var cleanupErr error
		if restoreSQL != "" {
			_, cleanupErr = conn.ExecContext(cleanupCtx, restoreSQL, restoreArgs...)
		}
		if _, err := conn.ExecContext(cleanupCtx, releaseSQL); err != nil {
			cleanupErr = err
		}
		if cleanupErr != nil {
			glog.Warningf("Discarding retention-index database connection after session cleanup failed: %v", cleanupErr)
			_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
		}
	}
	if dialect == "mysql" {
		var original int64
		if err := conn.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&original); err != nil {
			cleanup()
			return nil, err
		}
		restoreSQL = fmt.Sprintf("SET SESSION lock_wait_timeout = %d", original)
		if _, err := conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = 5"); err != nil {
			cleanup()
			return nil, err
		}
	} else {
		var lockTimeout, statementTimeout string
		if err := conn.QueryRowContext(ctx, "SELECT current_setting('lock_timeout'), current_setting('statement_timeout')").Scan(&lockTimeout, &statementTimeout); err != nil {
			cleanup()
			return nil, err
		}
		restoreSQL = "SELECT set_config('lock_timeout', $1, false), set_config('statement_timeout', $2, false)"
		restoreArgs = []interface{}{lockTimeout, statementTimeout}
		// The Go context supplies the configurable build deadline. Disable a shorter
		// inherited statement timeout without changing other pooled sessions.
		if _, err := conn.ExecContext(ctx, "SELECT set_config('lock_timeout', '5s', false), set_config('statement_timeout', '0', false)"); err != nil {
			cleanup()
			return nil, err
		}
	}
	return cleanup, nil
}
