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
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// The opt-ins require disposable databases and a role permitted to create its
// own database/schema. No existing database or user schema is altered.
func runtimeSchemaEngine(t *testing.T, engine string) (*gorm.DB, *gorm.DB, *runtimeSchemaTrace) {
	t.Helper()
	env := "KFP_RUNTIME_MIGRATION_MYSQL_DSN"
	if engine == "postgres" {
		env = "KFP_RUNTIME_MIGRATION_POSTGRES_DSN"
	}
	dsn := os.Getenv(env)
	if dsn == "" {
		fallback := "KFP_RECURRING_MYSQL_TEST_DSN"
		if engine == "postgres" {
			fallback = "KFP_RECURRING_POSTGRES_TEST_DSN"
		}
		dsn = os.Getenv(fallback)
	}
	if dsn == "" {
		t.Skip("set " + env + " for live startup migration evidence")
	}
	name := fmt.Sprintf("kfp_runtime_%d", time.Now().UnixNano())
	var admin *sql.DB
	var open func() *sql.DB
	var dialector func(*sql.DB) gorm.Dialector
	var drop string
	if engine == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		config.DBName = ""
		admin, err = sql.Open("mysql", config.FormatDSN())
		require.NoError(t, err)
		_, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4")
		require.NoError(t, err)
		drop = "DROP DATABASE `" + name + "`"
		config.DBName = name
		config.ClientFoundRows = true
		open = func() *sql.DB {
			db, err := sql.Open("mysql", config.FormatDSN())
			require.NoError(t, err)
			return db
		}
		dialector = func(db *sql.DB) gorm.Dialector { return mysql.New(mysql.Config{Conn: db, DefaultStringSize: 255}) }
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin = stdlib.OpenDB(*config)
		_, err = admin.Exec(`CREATE SCHEMA "` + name + `"`)
		require.NoError(t, err)
		drop = `DROP SCHEMA "` + name + `" CASCADE`
		config.RuntimeParams["search_path"] = name
		open = func() *sql.DB { return stdlib.OpenDB(*config) }
		dialector = func(db *sql.DB) gorm.Dialector { return postgres.New(postgres.Config{Conn: db}) }
	}
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	t.Cleanup(func() { _, err := admin.Exec(drop); require.NoError(t, err) })
	trace := &runtimeSchemaTrace{Interface: logger.Default.LogMode(logger.Silent)}
	connections := make([]*gorm.DB, 2)
	for i := range connections {
		raw := open()
		raw.SetMaxOpenConns(1)
		raw.SetMaxIdleConns(1)
		t.Cleanup(func() { require.NoError(t, raw.Close()) })
		db, err := gorm.Open(dialector(raw), &gorm.Config{Logger: trace})
		require.NoError(t, err)
		connections[i] = db
	}
	return connections[0], connections[1], trace
}

func runtimeSeedPriorRuns(t *testing.T, db *gorm.DB, count int) {
	t.Helper()
	require.NoError(t, autoMigrate(db))
	runs := make([]model.Run, count)
	manifest := strings.Repeat("pipeline-input-", 512)
	for i := range runs {
		runs[i] = model.Run{
			UUID: fmt.Sprintf("seed-%05d", i), DisplayName: "long-lived run", StorageState: model.StorageStateAvailable,
			RunDetails: model.RunDetails{State: model.RuntimeStateRunning, PipelineRuntimeManifest: model.LargeText(manifest), WorkflowRuntimeManifest: model.LargeText(manifest)},
		}
	}
	require.NoError(t, db.Omit(clause.Associations).CreateInBatches(runs, 100).Error)
	tasks := make([]map[string]any, count)
	for i := range tasks {
		tasks[i] = map[string]any{
			"UUID": fmt.Sprintf("task-%05d", i), "RunUUID": runs[i].UUID, "Namespace": "",
			"State": model.TaskStatus(apiv2.PipelineTask_RUNNING), "Fingerprint": strings.Repeat("f", 255),
			"pods": "[]", "TypeAttrs": "{}", "CreatedAtInSec": 1, "Type": 0,
		}
	}
	require.NoError(t, db.Table("tasks").CreateInBatches(tasks, 100).Error)
	require.NoError(t, db.Migrator().DropTable(&model.RuntimeTask{}))
	require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
	require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
	for _, column := range runtimeRunColumns {
		require.NoError(t, db.Exec("ALTER TABLE "+runtimeSchemaQuote(db, "run_details")+" DROP COLUMN "+runtimeSchemaQuote(db, column.name)).Error)
	}
}

func runtimeTableIdentity(t *testing.T, db *gorm.DB, engine string) int64 {
	t.Helper()
	var identity int64
	query := "SELECT pg_relation_filenode('run_details'::regclass)"
	if engine == "mysql" {
		query = "SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME = CONCAT(DATABASE(), '/run_details')"
	}
	require.NoError(t, db.Raw(query).Scan(&identity).Error)
	require.NotZero(t, identity)
	return identity
}

func TestRuntimeSchemaLiveUpgradeConcurrentDML(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, peer, trace := runtimeSchemaEngine(t, engine)
			runtimeSeedPriorRuns(t, db, 1000)
			before := runtimeTableIdentity(t, db, engine)
			trace.reset()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var active atomic.Bool
			var completed atomic.Int64
			var maximumLatency atomic.Int64
			ready, done := make(chan struct{}), make(chan error, 1)
			go func() {
				first := true
				for ctx.Err() == nil {
					start := time.Now()
					if err := peer.WithContext(ctx).Exec("UPDATE " + runtimeSchemaQuote(peer, "run_details") +
						" SET " + runtimeSchemaQuote(peer, "FinishedAtInSec") + " = " + runtimeSchemaQuote(peer, "FinishedAtInSec") + " + 1 WHERE " + runtimeSchemaQuote(peer, "UUID") + " = 'seed-00000'").Error; err != nil {
						if ctx.Err() == nil {
							done <- err
							return
						}
						break
					}
					if err := peer.WithContext(ctx).Model(&model.Task{}).Where(map[string]any{"UUID": "task-00000"}).
						UpdateColumn("StartedInSec", gorm.Expr(runtimeSchemaQuote(peer, "StartedInSec")+" + 1")).Error; err != nil {
						if ctx.Err() == nil {
							done <- err
							return
						}
						break
					}
					var count int64
					if err := peer.WithContext(ctx).Model(&model.Run{}).Count(&count).Error; err != nil {
						if ctx.Err() == nil {
							done <- err
							return
						}
						break
					}
					if count != 1000 {
						done <- fmt.Errorf("concurrent read returned %d rows", count)
						return
					}
					latency := time.Since(start).Nanoseconds()
					if active.Load() {
						completed.Add(1)
						for previous := maximumLatency.Load(); latency > previous && !maximumLatency.CompareAndSwap(previous, latency); previous = maximumLatency.Load() {
						}
					}
					if first {
						close(ready)
						first = false
					}
				}
				done <- nil
			}()
			select {
			case <-ready:
			case err := <-done:
				require.NoError(t, err)
				t.Fatal("DML stopped before migration")
			case <-time.After(10 * time.Second):
				t.Fatal("concurrent DML did not become ready")
			}
			active.Store(true)
			start := time.Now()
			err := autoMigrate(db)
			elapsed := time.Since(start)
			active.Store(false)
			cancel()
			require.NoError(t, <-done)
			require.NoError(t, err)
			assert.Positive(t, completed.Load(), "requests must finish while the actual startup entrypoint is active")
			assert.False(t, db.Migrator().HasColumn(&model.RuntimeTask{}, "PublicationRetryCount"))
			assert.Equal(t, before, runtimeTableIdentity(t, db, engine), "metadata additions must not rebuild run_details")
			var intact int64
			require.NoError(t, db.Model(&model.Run{}).Where(runtimeSchemaQuote(db, "PipelineRuntimeManifest")+" = ?", strings.Repeat("pipeline-input-", 512)).Count(&intact).Error)
			assert.EqualValues(t, 1000, intact)
			var nullMetadata int64
			require.NoError(t, db.Model(&model.Run{}).
				Where(runtimeSchemaQuote(db, "ExecutorType")+" IS NULL AND "+runtimeSchemaQuote(db, "WorkspaceProvisioner")+" IS NULL AND "+runtimeSchemaQuote(db, "WorkspaceReference")+" IS NULL").
				Count(&nullMetadata).Error)
			assert.EqualValues(t, 1000, nullMetadata, "existing Argo runs need no backfill and retain null execution metadata")
			var intactTasks int64
			require.NoError(t, db.Model(&model.Task{}).Where(runtimeSchemaQuote(db, "Fingerprint")+" = ?", strings.Repeat("f", 255)).Count(&intactTasks).Error)
			assert.EqualValues(t, 1000, intactTasks, "long task fingerprints must not be narrowed for the runtime foreign key")
			runtimeDDL, inheritedDDL := 0, 0
			for _, query := range trace.queries() {
				if strings.HasPrefix(query, "ALTER TABLE "+runtimeSchemaQuote(db, "run_details")) {
					if strings.Contains(query, runLeaseDiscoveryIndex) {
						assert.Contains(t, query, "ALGORITHM=INPLACE")
						assert.Contains(t, query, "LOCK=NONE")
						continue
					}
					runtimeColumn := false
					for _, column := range runtimeRunColumns {
						runtimeColumn = runtimeColumn || strings.Contains(query, runtimeSchemaQuote(db, column.name))
					}
					if !runtimeColumn {
						inheritedDDL++
						continue
					}
					runtimeDDL++
					assert.Contains(t, query, "ADD COLUMN")
					if engine == "mysql" {
						assert.Contains(t, query, "ALGORITHM=INSTANT")
					}
					assert.NotContains(t, query, "DEFAULT")
				}
			}
			assert.Equal(t, 1, runtimeDDL, "runtime columns must use exactly one constrained additive statement")
			t.Logf("engine=%s rows=1000 startup=%s concurrent_DML_completions=%d max_read_write_latency=%s table_identity=%d preserved pre_existing_core_DDL=%d (runtime-only migration evidence)",
				engine, elapsed, completed.Load(), time.Duration(maximumLatency.Load()), before, inheritedDDL)
			require.NoError(t, autoMigrate(db), "repeated startup must be idempotent")
		})
	}
}

func TestRuntimeSchemaLiveLockTimeoutAndCancellation(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, peer, trace := runtimeSchemaEngine(t, engine)
			runtimeSeedPriorRuns(t, db, 1)
			tx := peer.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			var count int64
			require.NoError(t, tx.Model(&model.Run{}).Count(&count).Error)
			trace.reset()
			start := time.Now()
			err := autoMigrate(db)
			elapsed := time.Since(start)
			require.ErrorContains(t, err, "no fallback")
			assert.GreaterOrEqual(t, elapsed, 4*time.Second)
			assert.Less(t, elapsed, 8*time.Second)
			assert.False(t, db.Migrator().HasColumn(&model.Run{}, "ExecutorType"))
			var ownerPID int64
			if engine == "postgres" {
				require.NoError(t, db.Raw("SELECT pg_backend_pid()").Scan(&ownerPID).Error)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			start = time.Now()
			err = autoMigrate(db.WithContext(ctx))
			cancel()
			require.Error(t, err)
			t.Logf("cancellation error: %v", err)
			assert.Less(t, time.Since(start), 2*time.Second)
			require.NoError(t, tx.Rollback().Error)
			if engine == "postgres" {
				require.Eventually(t, func() bool {
					locks, err := runtimeSchemaOwnershipLockCount(peer, ownerPID)
					return err == nil && locks == 0
				}, time.Second, 10*time.Millisecond, "a canceled/discarded session must release its migration ownership")
			}

			require.NoError(t, autoMigrate(db), "failure and cancellation must release ownership for recovery")
			t.Logf("engine=%s metadata_lock_failure_bounded=%s cancellation_bounded=250ms recovery=passed", engine, elapsed)
		})
	}
}

func runtimeSchemaOwnershipLockCount(db *gorm.DB, ownerPID int64) (int64, error) {
	var locks int64
	err := db.Raw(`SELECT COUNT(*) FROM pg_locks
		WHERE locktype = 'advisory' AND classid = 1180261217 AND objid = 1919775353 AND pid = ?`,
		ownerPID).Scan(&locks).Error
	return locks, err
}

func TestRuntimeSchemaLiveCancellationAllowsAnotherOwner(t *testing.T) {
	db, peer, _ := runtimeSchemaEngine(t, "postgres")
	var canceledPID int64
	require.NoError(t, db.Raw("SELECT pg_backend_pid()").Scan(&canceledPID).Error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := withRuntimeSchemaConnection(db.WithContext(ctx), func(*gorm.DB) error {
		cancel()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)

	require.NoError(t, withRuntimeSchemaConnection(peer, func(locked *gorm.DB) error {
		var nextPID int64
		require.NoError(t, locked.Raw("SELECT pg_backend_pid()").Scan(&nextPID).Error)
		require.NotEqual(t, canceledPID, nextPID)
		locks, err := runtimeSchemaOwnershipLockCount(db, canceledPID)
		require.NoError(t, err)
		require.Zero(t, locks, "another backend's valid migration ownership is not a leaked canceled-session lock")
		locks, err = runtimeSchemaOwnershipLockCount(locked, nextPID)
		require.NoError(t, err)
		require.EqualValues(t, 1, locks, "the next migration must still own its lock")
		return nil
	}))
}

func TestRuntimeSchemaLiveCompetingReplicasAndSessionCleanup(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, peer, _ := runtimeSchemaEngine(t, engine)
			runtimeSeedPriorRuns(t, db, 1)
			pool, err := db.DB()
			require.NoError(t, err)
			pool.SetMaxOpenConns(2)
			held, release := make(chan struct{}), make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
			})
			ownerDone := make(chan error, 1)
			go func() {
				ownerDone <- withRuntimeSchemaConnection(db, func(locked *gorm.DB) error {
					close(held)
					select {
					case <-release:
						return nil
					case <-locked.Statement.Context.Done():
						return locked.Statement.Context.Err()
					}
				})
			}()
			<-held
			competingDone := make(chan error, 1)
			start := time.Now()
			go func() { competingDone <- autoMigrate(peer) }()
			var count int64
			require.NoError(t, db.Model(&model.Run{}).Count(&count).Error)
			require.NoError(t, db.Exec("UPDATE "+runtimeSchemaQuote(db, "run_details")+
				" SET "+runtimeSchemaQuote(db, "FinishedAtInSec")+" = 1").Error)
			err = <-competingDone
			close(release)
			require.NoError(t, <-ownerDone)
			require.Error(t, err, "competing replica must fail rather than perform unowned DDL")
			assert.Less(t, time.Since(start), 8*time.Second)
			if engine == "mysql" {
				var timeout int64
				require.NoError(t, peer.Raw("SELECT @@SESSION.lock_wait_timeout").Scan(&timeout).Error)
				assert.NotEqualValues(t, 5, timeout, "temporary session timeout must be restored")
			} else {
				var timeout string
				require.NoError(t, peer.Raw("SELECT current_setting('lock_timeout')").Scan(&timeout).Error)
				assert.Equal(t, "0", timeout, "temporary session timeout must be restored")
			}
			require.NoError(t, autoMigrate(peer), "ownership handoff must recover without restart-only markers")
			require.NoError(t, autoMigrate(db))
			t.Logf("engine=%s competing_replica_failure_bounded=%s independent_DML=passed session_cleanup=passed ownership_handoff=passed", engine, time.Since(start))
		})
	}
}

func TestRuntimeSchemaLiveMySQLRefusesNonInstantFallback(t *testing.T) {
	db, _, trace := runtimeSchemaEngine(t, "mysql")
	require.NoError(t, db.Exec("CREATE TABLE run_details (UUID VARCHAR(191) NOT NULL PRIMARY KEY) ENGINE=MyISAM").Error)
	trace.reset()
	require.ErrorContains(t, autoMigrate(db), "no fallback")
	alterations := 0
	for _, query := range trace.queries() {
		if strings.HasPrefix(query, "ALTER TABLE") {
			alterations++
			assert.Contains(t, query, "ALGORITHM=INSTANT")
		}
	}
	assert.Equal(t, 1, alterations, "must not retry without the required algorithm")
	assert.False(t, db.Migrator().HasColumn(&model.Run{}, "ExecutorType"))
	assert.False(t, db.Migrator().HasTable(&model.RuntimeTask{}))
}

func TestRuntimeSchemaLivePostgresTableCreationFailureIsAtomic(t *testing.T) {
	db, _, _ := runtimeSchemaEngine(t, "postgres")
	runtimeSeedPriorRuns(t, db, 1)
	require.NoError(t, db.Exec(`CREATE TABLE index_collision (id INTEGER PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_runtime_tasks_attempt ON index_collision (id)`).Error)
	require.Error(t, autoMigrate(db), "index name collision must fail without replacing another table's index")
	exists, err := runtimeSchemaHasTable(db, "runtime_tasks")
	require.NoError(t, err)
	assert.False(t, exists, "an index creation failure must roll back the empty runtime table, not leave an unrecoverable partial schema")
	require.NoError(t, db.Exec("DROP TABLE index_collision").Error)
	require.NoError(t, autoMigrate(db), "startup must recover after the external index-name conflict is resolved")
}
