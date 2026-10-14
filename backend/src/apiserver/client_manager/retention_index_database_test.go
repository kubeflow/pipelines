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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/stretchr/testify/require"
)

// Uses the existing production-database CI services. Every test owns and removes
// a separate database/schema, so concurrent package tests cannot interfere.
func retentionDatabase(t *testing.T, engine string) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("kfp_retention_%d", time.Now().UnixNano())
	var db *sql.DB
	if engine == "mysql" {
		dsn := os.Getenv("KFP_RECURRING_MYSQL_TEST_DSN")
		if dsn == "" {
			t.Skip("KFP_RECURRING_MYSQL_TEST_DSN not set")
		}
		cfg, err := mysql.ParseDSN(dsn)
		require.NoError(t, err)
		admin, err := sql.Open("mysql", dsn)
		require.NoError(t, err)
		_, err = admin.Exec("CREATE DATABASE `" + name + "`")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := admin.Exec("DROP DATABASE `" + name + "`")
			require.NoError(t, err)
			require.NoError(t, admin.Close())
		})
		cfg.DBName = name
		db, err = sql.Open("mysql", cfg.FormatDSN())
		require.NoError(t, err)
	} else {
		dsn := os.Getenv("KFP_RECURRING_POSTGRES_TEST_DSN")
		if dsn == "" {
			t.Skip("KFP_RECURRING_POSTGRES_TEST_DSN not set")
		}
		cfg, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin := stdlib.OpenDB(*cfg)
		_, err = admin.Exec(`CREATE SCHEMA "` + name + `"`)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := admin.Exec(`DROP SCHEMA "` + name + `" CASCADE`)
			require.NoError(t, err)
			require.NoError(t, admin.Close())
		})
		cfg.RuntimeParams["search_path"] = name
		db = stdlib.OpenDB(*cfg)
	}
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	quote := sqldrv.NewDBDialect(engine).QuoteIdentifier
	_, err := db.Exec("CREATE TABLE run_details (" + quote("UUID") + " varchar(191) PRIMARY KEY, " + quote("StorageState") + " varchar(255), " + quote("FinishedAtInSec") + " bigint, " + quote("ArchivedAtInSec") + " bigint, payload text)")
	require.NoError(t, err)
	return db
}

func TestRetentionIndexesMySQLOnline(t *testing.T) {
	db := retentionDatabase(t, "mysql")
	_, err := db.Exec("INSERT INTO run_details VALUES (LPAD('0',36,'0'),'AVAILABLE',1,0,REPEAT('a',1024))")
	require.NoError(t, err)
	for n := 1; n < 262144; n *= 2 {
		_, err = db.Exec("INSERT INTO run_details SELECT LPAD(CAST(CAST(UUID AS UNSIGNED)+? AS CHAR),36,'0'),StorageState,FinishedAtInSec,ArchivedAtInSec,payload FROM run_details", n)
		require.NoError(t, err)
	}
	var before, after uint64
	require.NoError(t, db.QueryRow("SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME=CONCAT(DATABASE(),'/run_details')").Scan(&before))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	during := map[string]int{}
	longest := time.Duration(0)
	go func() {
		defer close(done)
		for ctx.Err() == nil {
			var activeBefore, activeAfter string
			query := "SELECT COALESCE(MAX(INFO),'') FROM information_schema.PROCESSLIST WHERE DB=DATABASE() AND INFO LIKE 'ALTER TABLE%idx_run_gc_%'"
			if err := db.QueryRowContext(ctx, query).Scan(&activeBefore); err != nil {
				if ctx.Err() == nil {
					done <- err
				}
				return
			}
			start := time.Now()
			if _, err := db.ExecContext(ctx, "UPDATE run_details SET FinishedAtInSec=FinishedAtInSec+1 WHERE UUID=LPAD('0',36,'0')"); err != nil {
				if ctx.Err() == nil {
					done <- err
				}
				return
			}
			var finished int64
			if err := db.QueryRowContext(ctx, "SELECT FinishedAtInSec FROM run_details WHERE UUID=LPAD('0',36,'0')").Scan(&finished); err != nil {
				if ctx.Err() == nil {
					done <- err
				}
				return
			}
			elapsed := time.Since(start)
			if elapsed > longest {
				longest = elapsed
			}
			if err := db.QueryRowContext(ctx, query).Scan(&activeAfter); err != nil {
				if ctx.Err() == nil {
					done <- err
				}
				return
			}
			for _, spec := range garbageCollectorRequiredIndexes {
				if strings.Contains(activeBefore, spec.name) && strings.Contains(activeAfter, spec.name) {
					during[spec.name]++
				}
			}
			time.Sleep(time.Millisecond)
		}
	}()
	started := time.Now()
	changed, buildErr := prepareGarbageCollectorIndexes(ctx, db, sqldrv.NewDBDialect("mysql"), 5*time.Minute)
	elapsed := time.Since(started)
	cancel()
	require.NoError(t, <-done)
	require.NoError(t, buildErr)
	require.True(t, changed)
	require.NoError(t, db.QueryRow("SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME=CONCAT(DATABASE(),'/run_details')").Scan(&after))
	require.Equal(t, before, after, "secondary index preparation must not copy/rebuild the clustered table")
	for _, spec := range garbageCollectorRequiredIndexes {
		require.Positive(t, during[spec.name], "writes must complete while %s is still building", spec.name)
	}
	// Record latency rather than imposing a machine-dependent performance SLA.
	// The overlap assertions above prove reads/writes complete before DDL ends.
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM run_details").Scan(&count))
	require.Equal(t, 262144, count)
	changed, err = prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Minute)
	require.NoError(t, err)
	require.False(t, changed)
	t.Logf("262144 rows, sequential index preparation=%s, completed read/write cycles during each active build=%v, maximum cycle latency=%s, unchanged TABLE_ID=%d", elapsed, during, longest, before)
}

func TestRetentionIndexesDatabaseRecovery(t *testing.T) {
	for _, engine := range []string{"mysql", "pgx"} {
		t.Run(engine, func(t *testing.T) {
			db := retentionDatabase(t, engine)
			dialect := sqldrv.NewDBDialect(engine)
			q := dialect.QuoteIdentifier
			_, err := db.Exec("CREATE INDEX idx_run_gc_lifecycle ON run_details (" + q("UUID") + ")")
			require.NoError(t, err)
			_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
			require.Error(t, err)
			drop := "DROP INDEX idx_run_gc_lifecycle"
			if engine == "mysql" {
				drop += " ON run_details"
			}
			// The helper must leave conflicting definitions untouched for inspection.
			_, err = db.Exec(drop)
			require.NoError(t, err)
			tx, err := db.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec("INSERT INTO run_details (" + q("UUID") + ") VALUES ('blocker')")
			require.NoError(t, err)
			started := time.Now()
			_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, 250*time.Millisecond)
			require.Error(t, err)
			require.Less(t, time.Since(started), 5*time.Second)
			require.NoError(t, tx.Rollback())
			if engine == "pgx" {
				// A canceled concurrent build can leave an invalid index. Automated retries
				// must not silently remove it or proceed to retention without ready indexes.
				var exists bool
				require.NoError(t, db.QueryRow("SELECT to_regclass('idx_run_gc_lifecycle') IS NOT NULL").Scan(&exists))
				if exists {
					_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
					require.Error(t, err)
					_, err = db.Exec("DROP INDEX CONCURRENTLY idx_run_gc_lifecycle")
					require.NoError(t, err)
				}
			}
			// MySQL may finish tearing down the canceled server session after the
			// client returns. Its retained advisory lock must prevent overlap.
			require.Eventually(t, func() bool {
				_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
				if err != nil {
					require.Contains(t, err.Error(), "another replica")
				}
				return err == nil
			}, 10*time.Second, 100*time.Millisecond)
			changed, err := prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
			require.NoError(t, err)
			require.False(t, changed)
		})
	}
}

func TestRetentionIndexesDatabaseOwnership(t *testing.T) {
	for _, engine := range []string{"mysql", "pgx"} {
		t.Run(engine, func(t *testing.T) {
			db := retentionDatabase(t, engine)
			dialect := sqldrv.NewDBDialect(engine)
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()
			release, err := lockGarbageCollectorIndexes(context.Background(), conn, engine)
			require.NoError(t, err)
			// A second live session must not build, drop, or replace anything while the
			// first owns preparation, even if PostgreSQL's catalog says an index is invalid.
			defer release()
			started := time.Now()
			_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
			require.ErrorContains(t, err, "another replica")
			require.Less(t, time.Since(started), 5*time.Second)
			release()
			_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
			require.NoError(t, err)
		})
	}
}

func TestRetentionIndexesMySQLInvisible(t *testing.T) {
	db := retentionDatabase(t, "mysql")
	dialect := sqldrv.NewDBDialect("mysql")
	_, err := prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
	require.NoError(t, err)
	_, err = db.Exec("ALTER TABLE run_details ALTER INDEX idx_run_gc_lifecycle INVISIBLE")
	require.NoError(t, err)
	_, err = prepareGarbageCollectorIndexes(context.Background(), db, dialect, time.Minute)
	require.ErrorContains(t, err, "unexpected definition")
	var visible string
	require.NoError(t, db.QueryRow("SELECT IS_VISIBLE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='run_details' AND INDEX_NAME='idx_run_gc_lifecycle' LIMIT 1").Scan(&visible))
	require.Equal(t, "NO", visible, "automation must preserve the conflicting index for inspection")
}

func TestRetentionIndexesMySQLMetadataWait(t *testing.T) {
	db := retentionDatabase(t, "mysql")
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec("SELECT UUID FROM run_details LIMIT 1")
	require.NoError(t, err)
	started := time.Now()
	_, err = prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Minute)
	require.ErrorContains(t, err, "Lock wait timeout exceeded")
	require.Less(t, time.Since(started), 10*time.Second, "metadata wait must stay short even with a longer build deadline")
	require.NoError(t, tx.Rollback())
	_, err = prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Minute)
	require.NoError(t, err)
}
