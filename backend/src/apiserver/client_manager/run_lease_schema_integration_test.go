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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLeaseSchemaLiveAdditiveUpgrade(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, peer, trace := runtimeSchemaEngine(t, engine)
			runtimeSeedPriorRuns(t, db, 2000)
			require.NoError(t, autoMigrate(db))
			require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
			require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
			q := func(name string) string { return runtimeSchemaQuote(db, name) }
			require.NoError(t, db.Exec("UPDATE "+q("run_details")+" SET "+q("ExecutorType")+" = 'kubernetes', "+q("State")+" = 'SUCCEEDED'").Error)
			require.NoError(t, db.Exec("UPDATE "+q("run_details")+" SET "+q("State")+" = 'RUNNING' WHERE "+q("UUID")+" < 'seed-00020'").Error)
			before := runtimeTableIdentity(t, db, engine)
			trace.reset()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var active atomic.Bool
			var completions atomic.Int64
			var maximumLatency atomic.Int64
			ready, done := make(chan struct{}), make(chan error, 1)
			go func() {
				first := true
				for ctx.Err() == nil {
					start := time.Now()
					if err := peer.WithContext(ctx).Exec("UPDATE " + q("run_details") + " SET " + q("FinishedAtInSec") + " = " + q("FinishedAtInSec") + " + 1 WHERE " + q("UUID") + " = 'seed-00000'").Error; err != nil {
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
					if active.Load() {
						completions.Add(1)
						latency := time.Since(start).Nanoseconds()
						for old := maximumLatency.Load(); latency > old && !maximumLatency.CompareAndSwap(old, latency); old = maximumLatency.Load() {
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
				t.Fatal("DML stopped before lease upgrade")
			case <-time.After(10 * time.Second):
				t.Fatal("DML did not become ready")
			}
			active.Store(true)
			start := time.Now()
			err := autoMigrate(db)
			elapsed := time.Since(start)
			active.Store(false)
			cancel()
			require.NoError(t, <-done)
			require.NoError(t, err)
			assert.Positive(t, completions.Load(), "requests must finish during the actual startup entrypoint")
			assert.Equal(t, before, runtimeTableIdentity(t, db, engine), "online index creation must not rebuild run_details")
			require.NoError(t, validateRunLeaseSchema(db))
			indexDDL := 0
			for _, query := range trace.queries() {
				if strings.Contains(query, runLeaseDiscoveryIndex) && (strings.HasPrefix(query, "ALTER TABLE") || strings.HasPrefix(query, "CREATE INDEX")) {
					indexDDL++
					if engine == "mysql" {
						assert.Contains(t, query, "ALGORITHM=INPLACE")
						assert.Contains(t, query, "LOCK=NONE")
					} else {
						assert.Contains(t, query, "CREATE INDEX CONCURRENTLY")
					}
				}
			}
			assert.Equal(t, 1, indexDDL)
			var count int64
			require.NoError(t, db.Model(&model.Run{}).Count(&count).Error)
			assert.EqualValues(t, 2000, count)
			require.NoError(t, db.Model(&model.Task{}).Where(q("Fingerprint")+" = ?", strings.Repeat("f", 255)).Count(&count).Error)
			assert.EqualValues(t, 2000, count)
			selectSQL := "SELECT " + q("UUID") + " FROM " + q("run_details") +
				" WHERE " + q("ExecutorType") + " = 'kubernetes' AND " + q("State") +
				" = 'RUNNING' ORDER BY " + q("CreatedAtInSec") + ", " + q("UUID") + " LIMIT 10"
			explain := "EXPLAIN (COSTS OFF) "
			if engine == "mysql" {
				require.NoError(t, db.Exec("ANALYZE TABLE "+q("run_details")).Error)
				explain = "EXPLAIN FORMAT=JSON "
			} else {
				require.NoError(t, db.Exec("ANALYZE "+q("run_details")).Error)
			}
			planRows, err := db.Raw(explain + selectSQL).Rows()
			require.NoError(t, err)
			var plan strings.Builder
			for planRows.Next() {
				var line string
				require.NoError(t, planRows.Scan(&line))
				plan.WriteString(line)
			}
			require.NoError(t, planRows.Err())
			require.NoError(t, planRows.Close())
			assert.Contains(t, plan.String(), runLeaseDiscoveryIndex, "the discovery index must be useful for the bounded executor/state lookup")
			t.Logf("engine=%s lease_upgrade=%s rows=2000 tasks=2000 concurrent_DML=%d max_latency=%s parent_table_identity=%d preserved",
				engine, elapsed, completions.Load(), time.Duration(maximumLatency.Load()), before)
			require.NoError(t, autoMigrate(db))
		})
	}
}

func TestRunLeaseSchemaLivePostgresInvalidOrCollidingIndexIsNotReplaced(t *testing.T) {
	db, _, _ := runtimeSchemaEngine(t, "postgres")
	runtimeSeedPriorRuns(t, db, 2)
	require.NoError(t, autoMigrate(db))
	require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
	require.NoError(t, db.Exec(`UPDATE run_details SET "ExecutorType" = 'kubernetes'`).Error)
	require.Error(t, db.Exec(`CREATE UNIQUE INDEX CONCURRENTLY `+runLeaseDiscoveryIndex+` ON run_details ("ExecutorType")`).Error)
	var oidBefore int64
	require.NoError(t, db.Raw("SELECT oid FROM pg_class WHERE relname = ? AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())", runLeaseDiscoveryIndex).Scan(&oidBefore).Error)
	require.NotZero(t, oidBefore)
	require.ErrorContains(t, autoMigrate(db), "refusing implicit schema repair")
	var oidAfter int64
	require.NoError(t, db.Raw("SELECT oid FROM pg_class WHERE relname = ? AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())", runLeaseDiscoveryIndex).Scan(&oidAfter).Error)
	assert.Equal(t, oidBefore, oidAfter, "startup must not drop an invalid index or another process's build")
}

func TestRunLeaseSchemaLivePostgresTableCreationRollsBack(t *testing.T) {
	db, _, _ := runtimeSchemaEngine(t, "postgres")
	require.NoError(t, autoMigrate(db))
	require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
	require.NoError(t, db.Exec("CREATE TABLE lease_index_collision (id INTEGER PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE INDEX idx_run_leases_expiry_run ON lease_index_collision (id)").Error)
	require.Error(t, autoMigrate(db))
	exists, err := runtimeSchemaHasTable(db, "run_leases")
	require.NoError(t, err)
	assert.False(t, exists, "separate PostgreSQL index creation must not leave a partial lease table")
	require.NoError(t, db.Exec("DROP TABLE lease_index_collision").Error)
	require.NoError(t, autoMigrate(db))
}

func TestRunLeaseSchemaLiveMySQLIndexRefusesBlockingFallback(t *testing.T) {
	db, _, trace := runtimeSchemaEngine(t, "mysql")
	require.NoError(t, db.Exec(`CREATE TABLE run_details (
		UUID VARCHAR(191) PRIMARY KEY, ExecutorType VARCHAR(191), State VARCHAR(255),
		CreatedAtInSec BIGINT NOT NULL) ENGINE=MyISAM`).Error)
	trace.reset()
	require.ErrorContains(t, createRunLeaseDiscoveryIndex(db), "without fallback")
	alterations := 0
	for _, query := range trace.queries() {
		if strings.HasPrefix(query, "ALTER TABLE") {
			alterations++
			assert.Contains(t, query, "ALGORITHM=INPLACE")
			assert.Contains(t, query, "LOCK=NONE")
		}
	}
	assert.Equal(t, 1, alterations)
	assert.False(t, db.Migrator().HasIndex(&model.Run{}, runLeaseDiscoveryIndex))
}

func TestRunLeaseSchemaLiveDiscoveryIndexLockWaitIsBounded(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, peer, _ := runtimeSchemaEngine(t, engine)
			runtimeSeedPriorRuns(t, db, 1)
			require.NoError(t, autoMigrate(db))
			require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
			blocker := peer.Begin()
			require.NoError(t, blocker.Error)
			defer blocker.Rollback()
			var count int64
			require.NoError(t, blocker.Model(&model.Run{}).Count(&count).Error)
			start := time.Now()
			err := withRuntimeSchemaConnection(db, createRunLeaseDiscoveryIndex)
			elapsed := time.Since(start)
			require.ErrorContains(t, err, "without fallback")
			assert.GreaterOrEqual(t, elapsed, 4*time.Second)
			assert.Less(t, elapsed, 8*time.Second)
			require.NoError(t, blocker.Rollback().Error)
			if engine == "postgres" {
				// Only this disposable test owns the failed build. Production
				// deliberately refuses to drop invalid/active indexes.
				var active int64
				require.NoError(t, peer.Raw("SELECT COUNT(*) FROM pg_stat_progress_create_index WHERE relid = 'run_details'::regclass").Scan(&active).Error)
				assert.Zero(t, active)
				require.NoError(t, db.Exec("DROP INDEX CONCURRENTLY IF EXISTS "+runtimeSchemaQuote(db, runLeaseDiscoveryIndex)).Error)
			}
			require.NoError(t, withRuntimeSchemaConnection(db, createRunLeaseDiscoveryIndex))
			t.Logf("engine=%s online_index_lock_wait=%s bounded; ownership/timeouts released; recovery passed", engine, elapsed)
		})
	}
}
