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
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBackfillPipelineRefs_ReplicaContentionAndShutdown(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, d := setupBackfillTestDB(t, engine)
			insertBackfillCandidates(t, db, 1)
			sessionQuery := "SELECT CONNECTION_ID()"
			sleepQuery := "SELECT SLEEP(30)"
			activeQuery := "SELECT COUNT(*) FROM information_schema.processlist WHERE id = ? AND info = ?"
			if engine == "postgres" {
				sessionQuery = "SELECT pg_backend_pid()"
				sleepQuery = "SELECT pg_sleep(30)"
				activeQuery = "SELECT COUNT(*) FROM pg_stat_activity WHERE pid = ? AND state = 'active' AND query = ?"
			}

			session := make(chan int64, 1)
			var once sync.Once
			require.NoError(t, db.Callback().Raw().Before("gorm:raw").Register("test:simulate-orphaned-statement", func(tx *gorm.DB) {
				if !isRunBackfillWrite(tx) {
					return
				}
				once.Do(func() {
					// Simulate a statement that keeps running after the client
					// cancels. Only a separate session can interrupt it.
					var id int64
					conn := tx.Statement.ConnPool
					ctx, stopSleep := context.WithTimeout(context.Background(), 15*time.Second)
					defer stopSleep()
					if err := conn.QueryRowContext(ctx, sessionQuery).Scan(&id); err != nil {
						tx.AddError(err)
						return
					}
					session <- id
					_, err := conn.ExecContext(ctx, sleepQuery)
					tx.AddError(err)
				})
			}))

			ctx, cancel := context.WithCancel(context.Background())
			finished := make(chan struct{})
			defer func() {
				cancel()
				<-finished
			}()
			result := make(chan error, 1)
			go func() {
				defer close(finished)
				_, err := runPipelineRefBackfillAttempt(ctx, db, d)
				result <- err
			}()
			var id int64
			select {
			case id = <-session:
			case err := <-result:
				t.Fatalf("worker stopped before its first update: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("worker did not reach its first update")
			}
			require.Eventually(t, func() bool {
				var count int
				return db.Raw(activeQuery, id, sleepQuery).Scan(&count).Error == nil && count == 1
			}, 5*time.Second, 25*time.Millisecond)

			waitCtx, stopWait := context.WithTimeout(context.Background(), 2*time.Second)
			defer stopWait()
			done, err := runPipelineRefBackfillAttempt(waitCtx, db, d)
			require.NoError(t, err)
			require.False(t, done, "another replica owns the migration lock")
			cancel()
			select {
			case err := <-result:
				require.Error(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown did not interrupt the backfill statement")
			}

			applied, err := migrationApplied(db, backfillPipelineRefsMigration)
			require.NoError(t, err)
			require.False(t, applied)
			done, err = runPipelineRefBackfillAttempt(context.Background(), db, d)
			require.NoError(t, err)
			require.True(t, done, "a new worker can resume after cancellation")
		})
	}
}

func TestBackfillPipelineRefs_MySQLMetadataLockTimeout(t *testing.T) {
	db, dialect := setupBackfillTestDB(t, "mysql")
	require.NoError(t, db.Connection(func(holder *gorm.DB) error {
		holder = holder.Session(&gorm.Session{})
		require.NoError(t, holder.Exec("LOCK TABLES resource_references WRITE").Error)
		defer func() { require.NoError(t, holder.Exec("UNLOCK TABLES").Error) }()

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		done, err := runPipelineRefBackfillAttempt(ctx, db, dialect)
		require.False(t, done)
		var mysqlErr *mysql.MySQLError
		require.ErrorAs(t, err, &mysqlErr)
		require.Equal(t, uint16(1205), mysqlErr.Number, "server metadata-lock timeout must precede context cancellation")
		return nil
	}))
	applied, err := migrationApplied(db, backfillPipelineRefsMigration)
	require.NoError(t, err)
	require.False(t, applied)
	done, err := runPipelineRefBackfillAttempt(context.Background(), db, dialect)
	require.NoError(t, err)
	require.True(t, done)
}

func TestBackfillPipelineRefs_MySQLStatementBinlog(t *testing.T) {
	db, dialect := setupBackfillTestDB(t, "mysql")
	insertBackfillCandidates(t, db, 2)

	// Report on the next candidate from a separate connection while the
	// worker still owns the first row. This also detects a multi-row write.
	var once sync.Once
	reported := false
	require.NoError(t, db.Callback().Raw().After("gorm:raw").Register("test:report-on-neighbor", func(tx *gorm.DB) {
		if !isRunBackfillWrite(tx) || tx.Error != nil {
			return
		}
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			// Use the underlying pool so this independent UPDATE does not
			// recursively invoke the worker callback.
			pool, err := db.DB()
			if err == nil {
				_, err = pool.ExecContext(ctx, "UPDATE run_details SET DisplayName = 'reported' WHERE UUID = 'run-000001'")
			}
			tx.AddError(err)
			reported = err == nil
		})
	}))

	require.NoError(t, db.Connection(func(pinned *gorm.DB) error {
		pinned = pinned.Session(&gorm.Session{})
		var logging bool
		var original string
		require.NoError(t, pinned.Raw("SELECT @@GLOBAL.log_bin AND @@SESSION.sql_log_bin, @@SESSION.binlog_format").Row().Scan(&logging, &original))
		require.True(t, logging, "MySQL integration service must have its binary log enabled")
		require.NoError(t, pinned.Exec("SET SESSION binlog_format = 'STATEMENT'").Error)
		defer func() { require.NoError(t, pinned.Exec("SET SESSION binlog_format = ?", original).Error) }()
		return backfillPipelineRefsToRunTable(pinned, dialect)
	}))
	require.True(t, reported)
	var repaired int
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM run_details WHERE PipelineId = 'pipeline'").Scan(&repaired).Error)
	require.Equal(t, 2, repaired)
}
