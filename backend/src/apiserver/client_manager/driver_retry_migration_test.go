// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDriverRetryMigrationProductionDatabases exercises the production upgrade
// entrypoint on populated prior schemas, using the existing database CI services.
func TestDriverRetryMigrationProductionDatabases(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			orm, db := driverRetryMigrationDatabase(t, engine)
			var version string
			require.NoError(t, db.QueryRow("SELECT VERSION()").Scan(&version))
			t.Logf("database version: %s", version)
			require.NoError(t, autoMigrate(orm))
			require.NoError(t, orm.Create(&model.Run{UUID: "migration-run", DisplayName: "before upgrade", Namespace: "test"}).Error)
			q := (&driverRetryMigration{engine: engine}).quote
			require.NoError(t, orm.Migrator().DropTable(&model.DriverTaskStop{}))
			require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
			for _, column := range driverRetryColumns {
				require.NoError(t, orm.Exec("ALTER TABLE "+q(column.table)+" DROP COLUMN "+q(column.name)).Error)
			}
			const rowCount = 10000
			longFingerprint := strings.Repeat("x", 255)
			for start := 0; start < rowCount; start += 100 {
				var values []string
				var args []any
				for i := 0; i < 100; i++ {
					values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
					args = append(args, fmt.Sprintf("task-%08d", start+i), "migration-run", "test", "[]", 0, longFingerprint, 0, 0, "{}")
				}
				var names []string
				for _, name := range []string{"UUID", "RunUUID", "Namespace", "pods", "CreatedAtInSec", "Fingerprint", "State", "Type", "TypeAttrs"} {
					names = append(names, q(name))
				}
				require.NoError(t, orm.Exec("INSERT INTO tasks ("+strings.Join(names, ",")+") VALUES "+strings.Join(values, ","), args...).Error)
			}

			var beforeID string
			if engine == "mysql" {
				require.NoError(t, db.QueryRow("SELECT GROUP_CONCAT(CONCAT(SUBSTRING_INDEX(NAME, '/', -1), ':', TABLE_ID) ORDER BY NAME) FROM information_schema.INNODB_TABLES WHERE NAME IN (CONCAT(DATABASE(), '/tasks'), CONCAT(DATABASE(), '/run_details'))").Scan(&beforeID))
			} else {
				require.NoError(t, db.QueryRow("SELECT CONCAT('tasks:', pg_relation_filenode('tasks'), ',run_details:', pg_relation_filenode('run_details'))").Scan(&beforeID))
			}

			// Both reads and writes must finish before migration returns. This covers
			// the live path, rather than merely checking requests after locks release.
			var completed atomic.Int64
			var maximum atomic.Int64
			stop := make(chan struct{})
			started := make(chan struct{})
			writerErrors := make(chan error, 1)
			go func() {
				defer close(writerErrors)
				close(started)
				for {
					select {
					case <-stop:
						return
					default:
					}
					began := time.Now()
					_, err := db.Exec("UPDATE tasks SET " + q("Name") + " = 'still writable' WHERE " + q("UUID") + " = 'task-00000000'")
					if err != nil {
						writerErrors <- err
						return
					}
					var value string
					if err := db.QueryRow("SELECT " + q("Fingerprint") + " FROM tasks WHERE " + q("UUID") + " = 'task-00000000'").Scan(&value); err != nil {
						writerErrors <- err
						return
					}
					if value != longFingerprint {
						writerErrors <- fmt.Errorf("fingerprint changed during migration")
						return
					}
					elapsed := time.Since(began).Nanoseconds()
					for previous := maximum.Load(); elapsed > previous && !maximum.CompareAndSwap(previous, elapsed); previous = maximum.Load() {
					}
					completed.Add(1)
				}
			}()
			<-started
			began := time.Now()
			migrationErr := autoMigrate(orm)
			during := completed.Load()
			close(stop)
			for err := range writerErrors {
				require.NoError(t, err)
			}
			require.NoError(t, migrationErr)
			require.Positive(t, during, "requests must finish while upgrade is active")
			t.Logf("engine=%s rows=%d migration=%s completed_reads_and_writes=%d max_latency=%s", engine, rowCount, time.Since(began), during, time.Duration(maximum.Load()))
			require.Less(t, time.Duration(maximum.Load()), 2*time.Second)
			if engine == "mysql" {
				var afterID string
				require.NoError(t, db.QueryRow("SELECT GROUP_CONCAT(CONCAT(SUBSTRING_INDEX(NAME, '/', -1), ':', TABLE_ID) ORDER BY NAME) FROM information_schema.INNODB_TABLES WHERE NAME IN (CONCAT(DATABASE(), '/tasks'), CONCAT(DATABASE(), '/run_details'))").Scan(&afterID))
				require.Equal(t, beforeID, afterID, "INSTANT additions and INPLACE secondary index must not rebuild the table")
				t.Logf("MySQL InnoDB table identities unchanged: %s", beforeID)
			} else {
				var afterID string
				require.NoError(t, db.QueryRow("SELECT CONCAT('tasks:', pg_relation_filenode('tasks'), ',run_details:', pg_relation_filenode('run_details'))").Scan(&afterID))
				require.Equal(t, beforeID, afterID, "PostgreSQL additive metadata must not rewrite the heap")
				t.Logf("PostgreSQL heap identities unchanged: %s", beforeID)
			}
			var count int64
			require.NoError(t, orm.Model(&model.Task{}).Where(q("Fingerprint")+" = ?", longFingerprint).Count(&count).Error)
			require.EqualValues(t, rowCount, count)
			var nullCount int64
			require.NoError(t, orm.Model(&model.Task{}).Where(q("DriverRetryGeneration")+" IS NULL").Count(&nullCount).Error)
			require.EqualValues(t, rowCount, nullCount)
			var run model.Run
			require.NoError(t, orm.First(&run, q("UUID")+" = ?", "migration-run").Error)
			require.False(t, run.DriverRetryTasksPresent)
			require.Nil(t, run.DriverRetryFinalizedGeneration)

			// Verify the exact finalization predicate can select the recovery
			// index, including its deterministic UUID traversal order.
			require.NoError(t, orm.Exec("UPDATE tasks SET "+q("DriverRetryGeneration")+" = 1 WHERE "+q("UUID")+" = 'task-00000000'").Error)
			query := "SELECT " + q("UUID") + " FROM tasks WHERE " + q("RunUUID") + " = 'migration-run' AND " + q("DriverRetryGeneration") + " = 1 ORDER BY " + q("UUID") + " LIMIT 100"
			var selected string
			require.NoError(t, db.QueryRow(query).Scan(&selected))
			require.Equal(t, "task-00000000", selected)
			if engine == "mysql" {
				require.NoError(t, orm.Exec("ANALYZE TABLE tasks").Error)
				var plan string
				require.NoError(t, db.QueryRow("EXPLAIN FORMAT=JSON "+query).Scan(&plan))
				require.Contains(t, plan, `"key": "idx_task_driver_retry"`)
			} else {
				require.NoError(t, orm.Exec("ANALYZE tasks").Error)
				rows, err := db.Query("EXPLAIN " + query)
				require.NoError(t, err)
				var plan []string
				for rows.Next() {
					var line string
					require.NoError(t, rows.Scan(&line))
					plan = append(plan, line)
				}
				require.NoError(t, rows.Err())
				require.NoError(t, rows.Close())
				require.Contains(t, strings.Join(plan, "\n"), "idx_task_driver_retry")
			}

			// Ordinary GORM reconciliation must not emit any recovery DDL afterward.
			statements := &driverRetryDDLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
			require.NoError(t, autoMigrate(orm.Session(&gorm.Session{Logger: statements})))
			require.Empty(t, statements.ddl, "repeated ORM reconciliation must not alter recovery columns or indexes")

			t.Run("incompatible_existing_schema_is_not_converted", func(t *testing.T) {
				require.NoError(t, orm.Exec("ALTER TABLE run_details ALTER COLUMN "+q("DriverRetryTasksPresent")+" SET DEFAULT true").Error)
				require.ErrorContains(t, autoMigrate(orm), "incompatible existing column")
				require.NoError(t, orm.Exec("ALTER TABLE run_details ALTER COLUMN "+q("DriverRetryTasksPresent")+" SET DEFAULT false").Error)
				if engine == "mysql" {
					require.NoError(t, orm.Exec("ALTER TABLE driver_task_stops MODIFY `StopKey` varchar(96) NOT NULL").Error)
				} else {
					require.NoError(t, orm.Exec(`ALTER TABLE driver_task_stops ALTER COLUMN "StopKey" TYPE varchar(96)`).Error)
				}
				require.ErrorContains(t, autoMigrate(orm), "incompatible driver_task_stops.StopKey")
				if engine == "mysql" {
					require.NoError(t, orm.Exec("ALTER TABLE driver_task_stops MODIFY `StopKey` varchar(64) NOT NULL").Error)
				} else {
					require.NoError(t, orm.Exec(`ALTER TABLE driver_task_stops ALTER COLUMN "StopKey" TYPE varchar(64)`).Error)
				}
				require.NoError(t, autoMigrate(orm))
			})
			t.Run("replica_ownership", func(t *testing.T) {
				conn, err := db.Conn(context.Background())
				require.NoError(t, err)
				defer conn.Close()
				settingsQuery := "SELECT @@SESSION.lock_wait_timeout"
				if engine == "postgres" {
					settingsQuery = "SHOW lock_timeout"
				}
				var originalTimeout string
				require.NoError(t, conn.QueryRowContext(context.Background(), settingsQuery).Scan(&originalTimeout))
				migration := &driverRetryMigration{conn: conn, engine: engine}
				require.NoError(t, migration.acquire(context.Background()))
				began := time.Now()
				err = migrateDriverRetrySchema(orm)
				require.ErrorContains(t, err, "another replica")
				require.Less(t, time.Since(began), time.Second)
				migration.release()
				var restoredTimeout string
				require.NoError(t, conn.QueryRowContext(context.Background(), settingsQuery).Scan(&restoredTimeout))
				require.Equal(t, originalTimeout, restoredTimeout, "pooled sessions must retain their original timeout")
				require.NoError(t, migrateDriverRetrySchema(orm))
			})
			t.Run("metadata_lock_timeout_and_resume", func(t *testing.T) {
				require.NoError(t, orm.Exec("ALTER TABLE tasks DROP COLUMN "+q("DriverCachedOutputs")).Error)
				tx, err := db.Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				var n int
				require.NoError(t, tx.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&n))
				began := time.Now()
				err = migrateDriverRetrySchema(orm)
				require.Error(t, err)
				require.Less(t, time.Since(began), 4*time.Second)
				require.NoError(t, tx.Rollback())
				require.NoError(t, migrateDriverRetrySchema(orm))
			})
			t.Run("cancellation_cleanup", func(t *testing.T) {
				require.NoError(t, orm.Exec("ALTER TABLE tasks DROP COLUMN "+q("DriverCachedOutputs")).Error)
				tx, err := db.Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				var n int
				require.NoError(t, tx.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&n))
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				require.Error(t, migrateDriverRetrySchema(orm.WithContext(ctx)))
				require.NoError(t, tx.Rollback())
				if engine == "mysql" {
					// The driver discards the canceled connection. MySQL can
					// retain its session until the bounded metadata wait exits.
					require.Eventually(t, func() bool {
						var owner sql.NullInt64
						err := db.QueryRow("SELECT IS_USED_LOCK(CONCAT('kfp_driver_retry_', MD5(DATABASE())))").Scan(&owner)
						return err == nil && !owner.Valid
					}, 5*time.Second, 20*time.Millisecond)
				} else {
					require.Eventually(t, func() bool {
						var owned bool
						err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND database = (SELECT oid FROM pg_database WHERE datname = current_database()) AND classid = hashtext('kubeflow_pipelines')::oid AND objid = hashtext('driver_retry_schema')::oid AND objsubid = 2)`).Scan(&owned)
						return err == nil && !owned
					}, 5*time.Second, 20*time.Millisecond)
				}
				require.NoError(t, migrateDriverRetrySchema(orm))
			})
			t.Run("incompatible_index_is_preserved", func(t *testing.T) {
				require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
				require.NoError(t, orm.Exec("CREATE INDEX "+driverRetryIndex+" ON tasks ("+q("UUID")+")").Error)
				require.ErrorContains(t, migrateDriverRetrySchema(orm), "incompatible")
				require.True(t, orm.Migrator().HasIndex(&model.Task{}, driverRetryIndex))
				require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
				require.NoError(t, migrateDriverRetrySchema(orm))
			})
			if engine == "postgres" {
				testDriverRetryInterruptedIndex(t, orm, db)
			} else {
				testDriverRetryMySQLActiveIndex(t, orm, db)
			}
		})
	}
}

func testDriverRetryMySQLActiveIndex(t *testing.T, orm *gorm.DB, db *sql.DB) {
	t.Helper()
	t.Run("reads_and_writes_during_online_index", func(t *testing.T) {
		// A representative index build must stay active long enough to observe
		// actual concurrent requests; ten thousand rows alone finish too quickly.
		for batch := 1; batch < 25; batch++ {
			_, err := db.Exec("INSERT INTO tasks (`UUID`, `RunUUID`, `Namespace`, `pods`, `CreatedAtInSec`, `Fingerprint`, `State`, `Type`, `TypeAttrs`) SELECT CONCAT(?, `UUID`), `RunUUID`, `Namespace`, `pods`, `CreatedAtInSec`, `Fingerprint`, `State`, `Type`, `TypeAttrs` FROM tasks WHERE `UUID` LIKE 'task-%'", fmt.Sprintf("batch-%02d-", batch))
			require.NoError(t, err)
		}
		require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
		done := make(chan error, 1)
		go func() { done <- migrateDriverRetrySchema(orm) }()
		require.Eventually(t, func() bool {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM performance_schema.threads WHERE PROCESSLIST_DB = DATABASE() AND PROCESSLIST_INFO LIKE 'ALTER TABLE `tasks` ADD INDEX `idx_task_driver_retry`%'").Scan(&count)
			return err == nil && count != 0
		}, 10*time.Second, time.Millisecond)
		began := time.Now()
		_, err := db.Exec("UPDATE tasks SET `Name` = 'mysql-online-index-write' WHERE `UUID` = 'task-00000000'")
		require.NoError(t, err)
		var fingerprint string
		require.NoError(t, db.QueryRow("SELECT `Fingerprint` FROM tasks WHERE `UUID` = 'task-00000000'").Scan(&fingerprint))
		require.Len(t, fingerprint, 255)
		select {
		case err := <-done:
			t.Fatalf("index finished before live request completed: %v", err)
		default:
		}
		t.Logf("MySQL read/write completed during active 250000-row online index build in %s", time.Since(began))
		require.NoError(t, <-done)
	})
}

func testDriverRetryInterruptedIndex(t *testing.T, orm *gorm.DB, db *sql.DB) {
	t.Helper()
	t.Run("active_concurrent_index_is_preserved", func(t *testing.T) {
		require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		require.NoError(t, err)
		defer tx.Rollback()
		var n int
		require.NoError(t, tx.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&n))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := db.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_task_driver_retry ON tasks ("RunUUID", "DriverRetryGeneration", "UUID")`)
			done <- err
		}()
		require.Eventually(t, func() bool {
			var active bool
			err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_index i JOIN pg_locks l ON l.relation = i.indrelid WHERE i.indexrelid = to_regclass('idx_task_driver_retry') AND NOT i.indisvalid AND l.mode = 'ShareUpdateExclusiveLock')`).Scan(&active)
			return err == nil && active
		}, time.Second, 10*time.Millisecond)
		// The concurrent build is held active by the old snapshot. New reads
		// and writes must complete before that snapshot is released.
		began := time.Now()
		_, err = db.Exec(`UPDATE tasks SET "Name" = 'online-index-write' WHERE "UUID" = 'task-00000000'`)
		require.NoError(t, err)
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE "Name" = 'online-index-write'`).Scan(&n))
		require.Equal(t, 1, n)
		t.Logf("read/write completed during active concurrent index build in %s", time.Since(began))
		select {
		case err := <-done:
			t.Fatalf("index finished before the live request check: %v", err)
		default:
		}
		require.ErrorContains(t, migrateDriverRetrySchema(orm), "active build")
		require.NoError(t, tx.Rollback())
		require.NoError(t, <-done)
		require.NoError(t, migrateDriverRetrySchema(orm))
	})
	t.Run("interrupted_concurrent_index", func(t *testing.T) {
		require.NoError(t, orm.Migrator().DropIndex(&model.Task{}, driverRetryIndex))
		// An old snapshot makes the concurrent index wait after entering the
		// catalog. Server-side cancellation leaves an abandoned invalid index
		// that cannot finish in the background after the client disconnects.
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		require.NoError(t, err)
		defer tx.Rollback()
		var n int
		require.NoError(t, tx.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&n))
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		defer conn.Close()
		var pid int
		require.NoError(t, conn.QueryRowContext(context.Background(), "SELECT pg_backend_pid()").Scan(&pid))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := conn.ExecContext(ctx, `CREATE INDEX CONCURRENTLY idx_task_driver_retry ON tasks ("RunUUID", "DriverRetryGeneration", "UUID")`)
			done <- err
		}()
		require.Eventually(t, func() bool {
			var invalid bool
			err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('idx_task_driver_retry') AND NOT indisvalid)`).Scan(&invalid)
			return err == nil && invalid
		}, time.Second, 10*time.Millisecond)
		var canceled bool
		require.NoError(t, db.QueryRow("SELECT pg_cancel_backend($1)", pid).Scan(&canceled))
		require.True(t, canceled)
		require.ErrorContains(t, <-done, "canceling statement")
		require.NoError(t, tx.Rollback())

		var valid bool
		require.NoError(t, db.QueryRow("SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass('idx_task_driver_retry')").Scan(&valid))
		require.False(t, valid)
		require.NoError(t, migrateDriverRetrySchema(orm))
		require.NoError(t, db.QueryRow("SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass('idx_task_driver_retry')").Scan(&valid))
		require.True(t, valid)
	})
}

type driverRetryDDLRecorder struct {
	logger.Interface
	mu  sync.Mutex
	ddl []string
}

func (l *driverRetryDDLRecorder) Trace(ctx context.Context, began time.Time, sql func() (string, int64), err error) {
	statement, _ := sql()
	isDDL := strings.HasPrefix(statement, "ALTER ") || strings.HasPrefix(statement, "CREATE ") || strings.HasPrefix(statement, "DROP ")
	isRecovery := strings.Contains(statement, "DriverRetry") || strings.Contains(statement, "DriverStopped") || strings.Contains(statement, "DriverCheckpoint") || strings.Contains(statement, "DriverCachedOutputs") || strings.Contains(statement, "driver_task_stops") || strings.Contains(statement, driverRetryIndex)
	if isDDL && isRecovery {
		l.mu.Lock()
		l.ddl = append(l.ddl, statement)
		l.mu.Unlock()
	}
	l.Interface.Trace(ctx, began, sql, err)
}

func driverRetryMigrationDatabase(t *testing.T, engine string) (*gorm.DB, *sql.DB) {
	t.Helper()
	env := "KFP_RECURRING_MYSQL_TEST_DSN"
	if engine == "postgres" {
		env = "KFP_RECURRING_POSTGRES_TEST_DSN"
	}
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip("set " + env + " for recovery migration coverage")
	}
	name := fmt.Sprintf("kfp_driver_migration_%d", time.Now().UnixNano())
	var admin, db *sql.DB
	var dialector gorm.Dialector
	var create, drop string
	if engine == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		config.DBName = ""
		admin, err = sql.Open("mysql", config.FormatDSN())
		require.NoError(t, err)
		create = "CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"
		drop = "DROP DATABASE `" + name + "`"
		config.DBName = name
		db, err = sql.Open("mysql", config.FormatDSN())
		require.NoError(t, err)
		dialector = mysql.New(mysql.Config{Conn: db})
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin = stdlib.OpenDB(*config)
		create = `CREATE SCHEMA "` + name + `"`
		drop = `DROP SCHEMA "` + name + `" CASCADE`
		config.RuntimeParams["search_path"] = name
		db = stdlib.OpenDB(*config)
		dialector = postgres.New(postgres.Config{Conn: db})
	}
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	_, err := admin.Exec(create)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec(drop); require.NoError(t, err) })
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	return orm, db
}
