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
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// migrationLockDatabase is a randomly named MySQL database or PostgreSQL schema.
// open returns a new connection pool to it, standing in for one API server.
type migrationLockDatabase struct {
	dialect sqldrv.DBDialect
	open    func(t *testing.T) *gorm.DB
}

// migrationLockEngines returns the engines whose DSN is set. CI sets both in
// presubmit-backend.yml.
func migrationLockEngines(t *testing.T) map[string]string {
	t.Helper()
	engines := map[string]string{}
	for dialectName, variable := range map[string]string{
		"mysql": "KFP_MIGRATION_LOCK_MYSQL_TEST_DSN",
		"pgx":   "KFP_MIGRATION_LOCK_POSTGRES_TEST_DSN",
	} {
		if dsn := os.Getenv(variable); dsn != "" {
			engines[dialectName] = dsn
		}
	}
	if len(engines) == 0 {
		t.Skip("set KFP_MIGRATION_LOCK_MYSQL_TEST_DSN or KFP_MIGRATION_LOCK_POSTGRES_TEST_DSN to run database tests")
	}
	return engines
}

func newMigrationLockDatabase(t *testing.T, dialectName, dsn string) migrationLockDatabase {
	t.Helper()
	name := "kfp_migration_lock_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	config := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	var rootDialector gorm.Dialector
	var childDSN, createStatement, dropStatement string
	var openDialector func(string) gorm.Dialector
	if dialectName == "mysql" {
		parsed, parseError := mysqldriver.ParseDSN(dsn)
		require.NoError(t, parseError)
		rootDialector = mysql.Open(dsn)
		// Match initDBDriver.
		parsed.DBName = name
		parsed.ParseTime = true
		parsed.ClientFoundRows = true
		childDSN = parsed.FormatDSN()
		openDialector = func(dsn string) gorm.Dialector {
			return mysql.New(mysql.Config{DSN: dsn, DefaultStringSize: 255})
		}
		createStatement, dropStatement = "CREATE DATABASE `"+name+"`", "DROP DATABASE `"+name+"`"
	} else {
		parsed, parseError := url.Parse(dsn)
		require.NoError(t, parseError)
		rootDialector = postgres.Open(dsn)
		query := parsed.Query()
		query.Set("search_path", name)
		parsed.RawQuery = query.Encode()
		childDSN = parsed.String()
		openDialector = postgres.Open
		createStatement, dropStatement = `CREATE SCHEMA "`+name+`"`, `DROP SCHEMA "`+name+`" CASCADE`
	}
	root, openError := gorm.Open(rootDialector, config)
	require.NoError(t, openError)
	rootPool, poolError := root.DB()
	require.NoError(t, poolError)
	require.NoError(t, root.Exec(createStatement).Error)
	t.Cleanup(func() {
		assert.NoError(t, root.Exec(dropStatement).Error)
		rootPool.Close()
	})
	return migrationLockDatabase{
		dialect: sqldrv.NewDBDialect(dialectName),
		open: func(t *testing.T) *gorm.DB {
			t.Helper()
			database, openError := gorm.Open(openDialector(childDSN), config)
			require.NoError(t, openError)
			pool, poolError := database.DB()
			require.NoError(t, poolError)
			// Registered after the schema drop, so it runs before it.
			t.Cleanup(func() { pool.Close() })
			return database
		},
	}
}

// forEachMigrationLockEngine runs test against a new database on every
// configured engine.
func forEachMigrationLockEngine(t *testing.T, test func(t *testing.T, database migrationLockDatabase)) {
	for dialectName, dsn := range migrationLockEngines(t) {
		t.Run(dialectName, func(t *testing.T) {
			test(t, newMigrationLockDatabase(t, dialectName, dsn))
		})
	}
}

// holdStartupMigrationLock takes the lock on its own connection, as another API
// server would, and returns that connection.
func holdStartupMigrationLock(t *testing.T, database migrationLockDatabase) *sql.Conn {
	t.Helper()
	pool, poolError := database.open(t).DB()
	require.NoError(t, poolError)
	connection, connectionError := pool.Conn(context.Background())
	require.NoError(t, connectionError)
	t.Cleanup(func() { connection.Close() })
	statements, statementsError := startupMigrationLockStatementsFor(database.dialect)
	require.NoError(t, statementsError)
	require.NoError(t, acquireStartupMigrationLock(connection, statements, 0))
	return connection
}

// startupMigrationLockIsFree reports whether a new session can take the lock.
func startupMigrationLockIsFree(t *testing.T, database migrationLockDatabase) bool {
	t.Helper()
	acquireError := withStartupMigrationLock(database.open(t), database.dialect, 0, func(*gorm.DB) error { return nil })
	if acquireError != nil {
		require.ErrorContains(t, acquireError, "held the lock")
		return false
	}
	return true
}

// runStartupMigrationsConcurrently starts replicas API servers at once and
// returns each one's error.
func runStartupMigrationsConcurrently(t *testing.T, database migrationLockDatabase, replicas int) []error {
	t.Helper()
	pools := make([]*gorm.DB, replicas)
	for index := range pools {
		pools[index] = database.open(t)
	}
	start := make(chan struct{})
	replicaErrors := make([]error, replicas)
	var running sync.WaitGroup
	for index, pool := range pools {
		running.Add(1)
		go func() {
			defer running.Done()
			<-start
			replicaErrors[index] = withStartupMigrationLock(pool, database.dialect, time.Minute, func(lockedSession *gorm.DB) error {
				return runStartupMigrations(lockedSession, database.dialect)
			})
		}()
	}
	close(start)
	running.Wait()
	return replicaErrors
}

func TestStartupMigrationLock_ConcurrentFreshInstall(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
			assert.NoError(t, replicaError, "replica %d", index)
		}
		assert.True(t, database.open(t).Migrator().HasTable("pipeline_version_tags"))
		assert.True(t, startupMigrationLockIsFree(t, database))
	})
}

func TestStartupMigrationLock_ConcurrentUpgrade(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		previousRelease := database.open(t)
		require.NoError(t, runStartupMigrations(previousRelease, database.dialect))
		// The new release adds a table and an index.
		dropIndex := "DROP INDEX idx_task_type ON tasks"
		if database.dialect.Name() == "pgx" {
			dropIndex = "DROP INDEX idx_task_type"
		}
		require.NoError(t, previousRelease.Exec("DROP TABLE pipeline_version_tags").Error)
		require.NoError(t, previousRelease.Exec(dropIndex).Error)

		for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
			assert.NoError(t, replicaError, "replica %d", index)
		}
		upgraded := database.open(t)
		assert.True(t, upgraded.Migrator().HasTable("pipeline_version_tags"))
		assert.True(t, upgraded.Migrator().HasIndex("tasks", "idx_task_type"))
	})
}

func TestStartupMigrationLock_ReleasedAfterMigrationError(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		migrationError := errors.New("migration failed")
		lockError := withStartupMigrationLock(database.open(t), database.dialect, 0, func(*gorm.DB) error { return migrationError })
		assert.ErrorIs(t, lockError, migrationError)
		assert.True(t, startupMigrationLockIsFree(t, database))
	})
}

func TestStartupMigrationLock_TimesOutWhileHeld(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		holdStartupMigrationLock(t, database)
		migrated := false
		lockError := withStartupMigrationLock(database.open(t), database.dialect, 0, func(*gorm.DB) error {
			migrated = true
			return nil
		})
		assert.ErrorContains(t, lockError, "another API server held the lock")
		assert.False(t, migrated, "a replica that times out must not migrate")
	})
}

// A crashed or disconnected holder's session ends, which releases the lock.
func TestStartupMigrationLock_AcquiredAfterHolderConnectionCloses(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		holder := holdStartupMigrationLock(t, database)
		// Discard instead of returning the connection to the pool, as a crash would.
		go func() {
			time.Sleep(2 * startupMigrationLockRetryPeriod)
			_ = holder.Raw(func(any) error { return driver.ErrBadConn })
			holder.Close()
		}()
		migrated := false
		lockError := withStartupMigrationLock(database.open(t), database.dialect, time.Minute, func(*gorm.DB) error {
			migrated = true
			return nil
		})
		assert.NoError(t, lockError)
		assert.True(t, migrated)
	})
}

// Every statement, including those inside a transaction, must run on the
// session that holds the lock, not on another connection from the pool.
func TestStartupMigrationLock_MigrationUsesLockedSession(t *testing.T) {
	holderQueries := map[string]string{
		"mysql": `SELECT IS_USED_LOCK(CONCAT('kfp:startup-migration:', LEFT(SHA2(DATABASE(), 256), 40)))`,
		"pgx": `SELECT pid FROM pg_locks WHERE locktype = 'advisory' AND granted AND objsubid = 2
			AND classid::bigint = (hashtext(current_database())::bigint & 4294967295)
			AND objid::bigint = (hashtext(current_schema() || ':kfp:startup-migration')::bigint & 4294967295)`,
	}
	sessionQueries := map[string]string{"mysql": "SELECT CONNECTION_ID()", "pgx": "SELECT pg_backend_pid()"}
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		replica := database.open(t)
		pool, poolError := replica.DB()
		require.NoError(t, poolError)
		pool.SetMaxOpenConns(4)
		observer := database.open(t)

		lockError := withStartupMigrationLock(replica, database.dialect, 0, func(lockedSession *gorm.DB) error {
			var holder int64
			require.NoError(t, observer.Raw(holderQueries[database.dialect.Name()]).Scan(&holder).Error)
			require.NotZero(t, holder)
			sessionQuery := sessionQueries[database.dialect.Name()]
			var session int64
			require.NoError(t, lockedSession.Raw(sessionQuery).Scan(&session).Error)
			assert.Equal(t, holder, session)
			lockedSession.Migrator().HasTable("run_details")
			require.NoError(t, lockedSession.Raw(sessionQuery).Scan(&session).Error)
			assert.Equal(t, holder, session, "after a Migrator call")
			return lockedSession.Transaction(func(transaction *gorm.DB) error {
				var transactionSession int64
				require.NoError(t, transaction.Raw(sessionQuery).Scan(&transactionSession).Error)
				assert.Equal(t, holder, transactionSession, "inside a transaction")
				return nil
			})
		})
		assert.NoError(t, lockError)
	})
}

// InitDBClient keeps using its handle after the migrations, so binding the
// locked session to one connection must not change that handle.
func TestStartupMigrationLock_CallerHandleStillUsesPool(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		replica := database.open(t)
		require.NoError(t, withStartupMigrationLock(replica, database.dialect, 0, func(*gorm.DB) error { return nil }))
		pool, poolError := replica.DB()
		require.NoError(t, poolError)
		assert.NoError(t, pool.Ping())
		assert.NoError(t, replica.Exec("SELECT 1").Error)
	})
}

// The legacy upgrade's experiment backfill must not need the connection pool,
// which a session bound to one connection cannot return.
func TestStartupMigrationLock_LegacyExperimentBackfillRunsOnLockedSession(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		replica := database.open(t)
		require.NoError(t, runStartupMigrations(replica, database.dialect))
		lockError := withStartupMigrationLock(replica, database.dialect, 0, backfillExperimentIDToRunTable)
		assert.NoError(t, lockError)
	})
}

// The lock is scoped to one database (MySQL) or schema (PostgreSQL), so separate
// installations on one server do not wait on each other.
func TestStartupMigrationLock_ScopedToOneInstallation(t *testing.T) {
	for dialectName, dsn := range migrationLockEngines(t) {
		t.Run(dialectName, func(t *testing.T) {
			holdStartupMigrationLock(t, newMigrationLockDatabase(t, dialectName, dsn))
			assert.True(t, startupMigrationLockIsFree(t, newMigrationLockDatabase(t, dialectName, dsn)))
		})
	}
}

func TestStartupMigrationLockStatementsFor_UnsupportedDialect(t *testing.T) {
	_, statementsError := startupMigrationLockStatementsFor(sqldrv.NewDBDialect("sqlite"))
	assert.ErrorContains(t, statementsError, `does not support database dialect "sqlite"`)
}
