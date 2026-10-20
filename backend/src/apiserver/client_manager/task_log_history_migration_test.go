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

type retryMigrationDDL struct {
	logger.Interface
	mu         sync.Mutex
	statements []string
}

func (l *retryMigrationDDL) Trace(ctx context.Context, started time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if strings.Contains(sql, "task_log_history") && (strings.HasPrefix(sql, "CREATE ") || strings.HasPrefix(sql, "ALTER ")) {
		l.mu.Lock()
		l.statements = append(l.statements, sql)
		l.mu.Unlock()
	}
	l.Interface.Trace(ctx, started, fc, err)
}

func retryHistoryDatabase(t *testing.T, engine string) *gorm.DB {
	t.Helper()
	env := "KFP_RECURRING_" + strings.ToUpper(engine) + "_TEST_DSN"
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip("set " + env + " for production-engine migration coverage")
	}
	name := fmt.Sprintf("retry_history_%d", time.Now().UnixNano())
	var admin, pool *sql.DB
	var dialector gorm.Dialector
	if engine == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		config.DBName = ""
		admin, err = sql.Open("mysql", config.FormatDSN())
		require.NoError(t, err)
		_, err = admin.Exec("CREATE DATABASE " + name)
		require.NoError(t, err)
		t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err); admin.Close() })
		config.DBName = name
		pool, err = sql.Open("mysql", config.FormatDSN())
		require.NoError(t, err)
		dialector = mysql.New(mysql.Config{Conn: pool})
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin = stdlib.OpenDB(*config)
		_, err = admin.Exec("CREATE SCHEMA " + name)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := admin.Exec("DROP SCHEMA " + name + " CASCADE")
			require.NoError(t, err)
			admin.Close()
		})
		config.RuntimeParams["search_path"] = name
		pool = stdlib.OpenDB(*config)
		dialector = postgres.New(postgres.Config{Conn: pool})
	}
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestRetryLogHistoryMigrationProductionDatabases(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db := retryHistoryDatabase(t, engine)
			// Construct the immediately preceding schema, without the new model.
			prior := []any{}
			for _, entry := range model.AllModels() {
				if _, history := entry.(*model.TaskLogHistory); !history {
					prior = append(prior, entry)
				}
			}
			require.NoError(t, db.AutoMigrate(prior...))
			require.NoError(t, db.Create(&model.Experiment{UUID: "prior-experiment", Name: "existing experiment", Namespace: "test"}).Error)
			require.NoError(t, db.Create(&model.Run{UUID: "prior-run", ExperimentId: "prior-experiment", Namespace: "test", DisplayName: "existing run"}).Error)
			longScope := strings.Repeat("long-existing-task-scope/", 64)
			for i := 0; i < 20; i++ {
				require.NoError(t, db.Table("tasks").Create(map[string]any{"UUID": fmt.Sprintf("prior-task-%d", i), "RunUUID": "prior-run", "Namespace": "test", "pods": "[]", "TypeAttrs": "{}", "ScopePath": longScope, "CreatedAtInSec": int64(0), "Fingerprint": "", "State": 0, "Type": 0}).Error)
			}

			require.NoError(t, db.Exec("CREATE TABLE migration_live_requests (id INTEGER PRIMARY KEY, payload TEXT NOT NULL, writes INTEGER NOT NULL)").Error)
			tableIdentity := func() string {
				var identity string
				if engine == "mysql" {
					require.NoError(t, db.Raw("SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME=CONCAT(DATABASE(), '/tasks')").Scan(&identity).Error)
				} else {
					require.NoError(t, db.Raw("SELECT 'tasks'::regclass::oid::text").Scan(&identity).Error)
				}
				return identity
			}
			beforeIdentity := tableIdentity()
			require.NotEmpty(t, beforeIdentity)
			payload := strings.Repeat("historical-value-", 64)
			for i := 0; i < 100; i++ {
				require.NoError(t, db.Exec("INSERT INTO migration_live_requests VALUES (?, ?, 0)", i, payload).Error)
			}
			var active atomic.Bool
			var completed atomic.Int64
			var maxLatency atomic.Int64
			stop := make(chan struct{})
			workerErrors := make(chan error, 1)
			var worker sync.WaitGroup
			active.Store(true)
			worker.Add(1)
			go func() {
				defer worker.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					started := time.Now()
					if err := db.Model(&model.Task{}).Where(map[string]any{"UUID": "prior-task-0"}).UpdateColumn("FinishedInSec", gorm.Expr("?", int64(42))).Error; err != nil {
						workerErrors <- err
						return
					}
					if err := db.Exec("UPDATE migration_live_requests SET writes=writes+1 WHERE id=0").Error; err != nil {
						workerErrors <- err
						return
					}
					var value string
					if err := db.Raw("SELECT payload FROM migration_live_requests WHERE id=0").Scan(&value).Error; err != nil {
						workerErrors <- err
						return
					}
					if value != payload {
						workerErrors <- fmt.Errorf("prior data changed")
						return
					}
					if active.Load() {
						completed.Add(1)
					}
					latency := time.Since(started).Nanoseconds()
					for old := maxLatency.Load(); latency > old; old = maxLatency.Load() {
						if maxLatency.CompareAndSwap(old, latency) {
							break
						}
					}
				}
			}()
			ddl := &retryMigrationDDL{Interface: db.Logger}
			migrationErr := autoMigrate(db.Session(&gorm.Session{Logger: ddl}))
			active.Store(false)
			close(stop)
			worker.Wait()
			require.NoError(t, migrationErr)
			require.NotEmpty(t, ddl.statements)
			for _, statement := range ddl.statements {
				require.NotContains(t, strings.ToUpper(statement), "ALTER TABLE")
				require.NotContains(t, strings.ToUpper(statement), "FOREIGN KEY")
				t.Logf("new-table DDL: %s", statement)
			}
			select {
			case err := <-workerErrors:
				require.NoError(t, err)
			default:
			}
			require.Positive(t, completed.Load(), "requests must finish during the actual startup migration")
			require.Equal(t, beforeIdentity, tableIdentity(), "existing tasks table must not be rebuilt")
			t.Logf("concurrent read/write pairs=%d maximum pair latency=%s; prior fixture rows=100 payload bytes=%d", completed.Load(), time.Duration(maxLatency.Load()), len(payload))
			var priorTask model.Task
			require.NoError(t, db.Where(map[string]any{"UUID": "prior-task-0"}).First(&priorTask).Error)
			require.Equal(t, longScope, priorTask.ScopePath)
			require.True(t, db.Migrator().HasTable(&model.TaskLogHistory{}))
			require.True(t, db.Migrator().HasIndex(&model.TaskLogHistory{}, "idx_task_log_history_run"))
			require.NoError(t, autoMigrate(db))
			require.NoError(t, db.Create(&model.TaskLogHistory{TaskID: "task", BeforeRetryGeneration: 1, RunUUID: "run", Snapshot: model.LargeText("retained")}).Error)
			require.NoError(t, autoMigrate(db))
			var retained model.TaskLogHistory
			require.NoError(t, db.First(&retained).Error)
			require.Equal(t, model.LargeText("retained"), retained.Snapshot)
			t.Run("competing startup creation", func(t *testing.T) {
				require.NoError(t, db.Migrator().DropTable(&model.TaskLogHistory{}))
				start := make(chan struct{})
				results := make(chan error, 2)
				for range 2 {
					go func() { <-start; results <- createTaskLogHistory(db) }()
				}
				close(start)
				successes := 0
				for range 2 {
					err := <-results
					if err == nil {
						successes++
					} else {
						require.ErrorContains(t, err, "owned by another")
					}
				}
				require.Positive(t, successes)
				require.NoError(t, createTaskLogHistory(db))
				require.True(t, db.Migrator().HasIndex(&model.TaskLogHistory{}, "idx_task_log_history_run"))
			})
			if engine == "postgres" {
				t.Run("blocked DDL timeout and recovery", func(t *testing.T) {
					require.NoError(t, db.Migrator().DropTable(&model.TaskLogHistory{}))
					pool, err := db.DB()
					require.NoError(t, err)
					owner, err := pool.Begin()
					require.NoError(t, err)
					_, err = owner.Exec("CREATE TABLE task_log_history (blocked INTEGER)")
					require.NoError(t, err)
					started := time.Now()
					migrationErr := createTaskLogHistory(db)
					require.Error(t, migrationErr)
					require.Less(t, time.Since(started), 4*time.Second)
					require.NoError(t, owner.Rollback())
					require.NoError(t, createTaskLogHistory(db))
					require.True(t, db.Migrator().HasIndex(&model.TaskLogHistory{}, "idx_task_log_history_run"))
				})
			}
			t.Run("competing owner and cancellation", func(t *testing.T) {
				pool, err := db.DB()
				require.NoError(t, err)
				owner, err := pool.Conn(context.Background())
				require.NoError(t, err)
				defer owner.Close()
				if engine == "mysql" {
					var got int
					require.NoError(t, owner.QueryRowContext(context.Background(), "SELECT GET_LOCK(SHA2(CONCAT(DATABASE(), ?), 256),0)", taskLogHistoryMigrationLock).Scan(&got))
					require.Equal(t, 1, got)
				} else {
					var got bool
					require.NoError(t, owner.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock(hashtext($1))", taskLogHistoryMigrationLock).Scan(&got))
					require.True(t, got)
				}
				started := time.Now()
				require.ErrorContains(t, createTaskLogHistory(db), "owned by another")
				require.Less(t, time.Since(started), 3*time.Second)
				if engine == "mysql" {
					_, err = owner.ExecContext(context.Background(), "SELECT RELEASE_LOCK(SHA2(CONCAT(DATABASE(), ?), 256))", taskLogHistoryMigrationLock)
				} else {
					_, err = owner.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", taskLogHistoryMigrationLock)
				}
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				require.Error(t, createTaskLogHistory(db.WithContext(ctx)))
				require.NoError(t, createTaskLogHistory(db))
			})
		})
	}
}
