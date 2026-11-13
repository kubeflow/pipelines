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
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// migrationLockDatabase is a randomly named MySQL database or PostgreSQL schema.
// open returns a new connection pool to it, standing in for one API server.
// sibling returns another new database on the same engine.
type migrationLockDatabase struct {
	dialect sqldrv.DBDialect
	open    func(t *testing.T) *gorm.DB
	sibling func(t *testing.T) migrationLockDatabase
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
		sibling: func(t *testing.T) migrationLockDatabase {
			return newMigrationLockDatabase(t, dialectName, dsn)
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

// schemaSnapshot lists every column, index and constraint in the database, so
// two databases can be compared. Index definitions leave out the schema name.
func schemaSnapshot(t *testing.T, database migrationLockDatabase) []string {
	t.Helper()
	queries := map[string][]string{
		"mysql": {
			"SELECT CONCAT(TABLE_NAME, '.', COLUMN_NAME, ' ', COLUMN_TYPE, ' ', IS_NULLABLE, ' ', COALESCE(COLUMN_DEFAULT, 'NULL')) " +
				"FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE()",
			"SELECT CONCAT(TABLE_NAME, ' index ', INDEX_NAME, ' ', SEQ_IN_INDEX, ' ', COALESCE(COLUMN_NAME, EXPRESSION), ' ', NON_UNIQUE) " +
				"FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE()",
			"SELECT CONCAT(TABLE_NAME, ' constraint ', CONSTRAINT_NAME, ' ', CONSTRAINT_TYPE) " +
				"FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = DATABASE()",
		},
		"pgx": {
			"SELECT table_name || '.' || column_name || ' ' || data_type || ' ' || COALESCE(character_maximum_length::text, '') || ' ' || " +
				"is_nullable || ' ' || REPLACE(COALESCE(column_default, 'NULL'), current_schema() || '.', '') " +
				"FROM information_schema.columns WHERE table_schema = current_schema()",
			"SELECT tablename || ' index ' || REPLACE(indexdef, current_schema() || '.', '') FROM pg_indexes WHERE schemaname = current_schema()",
			// NOT NULL appears as CHECK constraints with generated names; the
			// column listing above already covers nullability.
			"SELECT table_name || ' constraint ' || constraint_name || ' ' || constraint_type " +
				"FROM information_schema.table_constraints WHERE table_schema = current_schema() AND constraint_type <> 'CHECK'",
		},
	}
	session := database.open(t)
	var snapshot []string
	for _, query := range queries[database.dialect.Name()] {
		var lines []string
		require.NoError(t, session.Raw(query).Scan(&lines).Error)
		snapshot = append(snapshot, lines...)
	}
	sort.Strings(snapshot)
	require.NotEmpty(t, snapshot)
	return snapshot
}

// migrateOnce migrates database with a single API server, for comparison.
func migrateOnce(t *testing.T, database migrationLockDatabase) {
	t.Helper()
	require.NoError(t, withStartupMigrationLock(database.open(t), database.dialect, 0, func(lockedSession *gorm.DB) error {
		return runStartupMigrations(lockedSession, database.dialect)
	}))
}

func TestStartupMigrationLock_ConcurrentFreshInstall(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
			assert.NoError(t, replicaError, "replica %d", index)
		}
		installed := database.open(t)
		assert.True(t, installed.Migrator().HasTable("pipeline_version_tags"))
		assert.True(t, startupMigrationLockIsFree(t, database))
		reference := database.sibling(t)
		migrateOnce(t, reference)
		assert.Equal(t, schemaSnapshot(t, reference), schemaSnapshot(t, database),
			"three replicas must leave the same schema as one")
		if database.dialect.Name() == "pgx" {
			// A replica that dropped another replica's in-progress
			// CONCURRENTLY build could leave an index missing or invalid.
			for _, spec := range expressionIndexes {
				valid, validError := isPostgresIndexValid(installed, spec.name)
				require.NoError(t, validError)
				assert.True(t, valid, "expression index %s", spec.name)
			}
		}
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

// A normal restart finds the schema current and leaves it unchanged.
func TestStartupMigrationLock_ConcurrentRestartWithCurrentSchema(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		migrateOnce(t, database)
		before := schemaSnapshot(t, database)
		for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
			assert.NoError(t, replicaError, "replica %d", index)
		}
		assert.Equal(t, before, schemaSnapshot(t, database))
	})
}

// A replica that stops partway through the migrations releases the lock, and
// the next replicas finish them with the same result as an uninterrupted run.
func TestStartupMigrationLock_NextReplicaFinishesInterruptedMigration(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		stopped := errors.New("replica stopped")
		lockError := withStartupMigrationLock(database.open(t), database.dialect, 0, func(lockedSession *gorm.DB) error {
			require.NoError(t, lockedSession.AutoMigrate(model.AllModels()[:3]...))
			return stopped
		})
		require.ErrorIs(t, lockError, stopped)

		for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
			assert.NoError(t, replicaError, "replica %d", index)
		}
		reference := database.sibling(t)
		migrateOnce(t, reference)
		assert.Equal(t, schemaSnapshot(t, reference), schemaSnapshot(t, database))
	})
}

// seedLegacySchema makes database look like a pre-2.15 one with rows:
// pipelines.UUID is wider than 64 characters, the run backfill is pending, and
// run-without-experiment has no ExperimentUUID but references its experiment.
func seedLegacySchema(t *testing.T, database migrationLockDatabase) {
	t.Helper()
	previousRelease := database.open(t)
	require.NoError(t, runStartupMigrations(previousRelease, database.dialect))
	require.NoError(t, previousRelease.Create(&model.Pipeline{
		UUID: "pipeline-legacy", Name: "pipeline-legacy", DisplayName: "pipeline-legacy", Namespace: "ns",
		Status: model.PipelineReady,
	}).Error)
	require.NoError(t, previousRelease.Create(&model.Experiment{
		UUID: "exp-1", Name: "exp-1", Namespace: "ns", StorageState: model.StorageStateAvailable,
	}).Error)
	insertLegacyRunWithPipelineRef(t, previousRelease, "run-legacy", "pipeline-legacy")
	// One connection, so FOREIGN_KEY_CHECKS applies to every statement; each
	// insert needs its own gorm session on it.
	require.NoError(t, previousRelease.Connection(func(session *gorm.DB) error {
		if setError := session.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; setError != nil {
			return setError
		}
		if createError := session.Session(&gorm.Session{}).Create(&model.Run{
			UUID: "run-without-experiment", DisplayName: "run-without-experiment", K8SName: "run-without-experiment",
			Namespace: "ns", StorageState: model.StorageStateAvailable,
		}).Error; createError != nil {
			return createError
		}
		if createError := session.Session(&gorm.Session{}).Create(&model.ResourceReference{
			ResourceUUID: "run-without-experiment", ResourceType: model.RunResourceType,
			ReferenceUUID: "exp-1", ReferenceName: "exp-1", ReferenceType: model.ExperimentResourceType,
			Relationship: model.OwnerRelationship,
		}).Error; createError != nil {
			return createError
		}
		return session.Exec("ALTER TABLE pipelines MODIFY `UUID` VARCHAR(255) NOT NULL").Error
	}))
	require.NoError(t, previousRelease.Where(map[string]any{"Name": backfillPipelineRefsMigration}).Delete(&model.MigrationStatus{}).Error)
	legacy, legacyError := isLegacySchema(previousRelease)
	require.NoError(t, legacyError)
	require.True(t, legacy)
}

// assertLegacyUpgradeFinished checks the rows seedLegacySchema added after the
// legacy upgrade, including both of its backfills.
func assertLegacyUpgradeFinished(t *testing.T, database migrationLockDatabase) {
	t.Helper()
	upgraded := database.open(t)
	legacy, legacyError := isLegacySchema(upgraded)
	require.NoError(t, legacyError)
	assert.False(t, legacy, "pipelines.UUID should be shrunk back to 64 characters")
	var runs, pipelines int64
	require.NoError(t, upgraded.Model(&model.Run{}).Count(&runs).Error)
	require.NoError(t, upgraded.Model(&model.Pipeline{}).Count(&pipelines).Error)
	assert.Equal(t, int64(2), runs)
	assert.Equal(t, int64(1), pipelines)
	var pipelineID string
	require.NoError(t, upgraded.Raw("SELECT `PipelineId` FROM run_details WHERE `UUID` = ?", "run-legacy").Scan(&pipelineID).Error)
	assert.Equal(t, "pipeline-legacy", pipelineID, "the run backfill should fill PipelineId from its resource reference")
	assert.Equal(t, "exp-1", storedRunExperimentID(t, upgraded, "run-without-experiment"),
		"the experiment backfill should fill ExperimentUUID from its resource reference")
}

// The legacy (pre-2.15) upgrade is MySQL only: dropAllFKConstraints does not
// implement it for PostgreSQL.
func legacyUpgradeDatabase(t *testing.T) migrationLockDatabase {
	t.Helper()
	dsn, ok := migrationLockEngines(t)["mysql"]
	if !ok {
		t.Skip("set KFP_MIGRATION_LOCK_MYSQL_TEST_DSN to run the legacy upgrade tests")
	}
	return newMigrationLockDatabase(t, "mysql", dsn)
}

func TestStartupMigrationLock_ConcurrentLegacyUpgrade(t *testing.T) {
	database := legacyUpgradeDatabase(t)
	seedLegacySchema(t, database)
	for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
		assert.NoError(t, replicaError, "replica %d", index)
	}
	assertLegacyUpgradeFinished(t, database)
}

// lengthSpecsBeforePipelineUUID returns the column shrinks that run before
// pipelines.UUID, whose width isLegacySchema checks.
func lengthSpecsBeforePipelineUUID(t *testing.T) []validation.ColLenSpec {
	t.Helper()
	for index, spec := range validation.LengthSpecs {
		if _, isPipeline := spec.Model.(*model.Pipeline); isPipeline && spec.Field == "UUID" {
			return validation.LengthSpecs[:index]
		}
	}
	t.Fatal("validation.LengthSpecs has no pipelines.UUID entry")
	return nil
}

// A replica that stops during the legacy upgrade before pipelines.UUID is
// shrunk leaves the schema detected as legacy, so the next replica runs the
// whole upgrade. Stopping after that shrink is
// https://github.com/kubeflow/pipelines/issues/15513.
func TestStartupMigrationLock_LegacyUpgradeStoppedBeforePipelineUUIDShrink(t *testing.T) {
	uninterrupted := legacyUpgradeDatabase(t)
	seedLegacySchema(t, uninterrupted)
	migrateOnce(t, uninterrupted)

	database := uninterrupted.sibling(t)
	seedLegacySchema(t, database)
	stopped := errors.New("replica stopped")
	lockError := withStartupMigrationLock(database.open(t), database.dialect, 0, func(lockedSession *gorm.DB) error {
		require.NoError(t, dropAllFKConstraints(lockedSession, database.dialect.Name()))
		require.NoError(t, dropLegacyIndexes(lockedSession, database.dialect))
		require.NoError(t, shrinkColumns(lockedSession, lengthSpecsBeforePipelineUUID(t)))
		return stopped
	})
	require.ErrorIs(t, lockError, stopped)
	legacy, legacyError := isLegacySchema(database.open(t))
	require.NoError(t, legacyError)
	require.True(t, legacy, "the schema must still be detected as legacy")

	for index, replicaError := range runStartupMigrationsConcurrently(t, database, 3) {
		assert.NoError(t, replicaError, "replica %d", index)
	}
	assertLegacyUpgradeFinished(t, database)
	assert.Equal(t, schemaSnapshot(t, uninterrupted), schemaSnapshot(t, database))
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

// runStartupMigrationsOnOneConnection migrates with a pool of one connection,
// which the lock holds, so the migrations finish only if every statement uses
// the locked session. A helper that asks the locked session for the pool, as
// with gorm's DB method, gets an "invalid db" error; one that reaches the pool
// another way waits for a second connection.
func runStartupMigrationsOnOneConnection(t *testing.T, database migrationLockDatabase) {
	t.Helper()
	replica := database.open(t)
	pool, poolError := replica.DB()
	require.NoError(t, poolError)
	pool.SetMaxOpenConns(1)
	finished := make(chan error, 1)
	go func() {
		finished <- withStartupMigrationLock(replica, database.dialect, 0, func(lockedSession *gorm.DB) error {
			return runStartupMigrations(lockedSession, database.dialect)
		})
	}()
	select {
	case migrationError := <-finished:
		require.NoError(t, migrationError)
	case <-time.After(time.Minute):
		t.Fatal("a migration statement waited for a connection other than the locked session")
	}
}

func TestStartupMigrationLock_FreshInstallUsesOnlyLockedSession(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		runStartupMigrationsOnOneConnection(t, database)
	})
}

func TestStartupMigrationLock_LegacyUpgradeUsesOnlyLockedSession(t *testing.T) {
	database := legacyUpgradeDatabase(t)
	seedLegacySchema(t, database)
	runStartupMigrationsOnOneConnection(t, database)
	assertLegacyUpgradeFinished(t, database)
}

// The lock takes no table locks, so other API servers keep serving requests
// while one replica migrates.
func TestStartupMigrationLock_RequestsSucceedWhileHeld(t *testing.T) {
	forEachMigrationLockEngine(t, func(t *testing.T, database migrationLockDatabase) {
		migrateOnce(t, database)
		holdStartupMigrationLock(t, database)
		server := database.open(t)
		requestContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, server.WithContext(requestContext).Create(&model.Experiment{
			UUID: "exp-while-locked", Name: "exp-while-locked", Namespace: "ns", StorageState: model.StorageStateAvailable,
		}).Error)
		var experiments int64
		require.NoError(t, server.WithContext(requestContext).Model(&model.Experiment{}).
			Where(map[string]any{"UUID": "exp-while-locked"}).Count(&experiments).Error)
		assert.Equal(t, int64(1), experiments)
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
