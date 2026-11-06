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
	"os"
	"strings"
	"testing"
	"time"

	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/stretchr/testify/require"
)

// This test requires the stage/innodb/alter% instruments and
// events_stages_current consumer enabled in the disposable MySQL test server.
// It cancels only after measured index work and an overlapping application write,
// rather than timing out while the DDL is merely waiting for a metadata lock.
func TestRetentionIndexesMySQLInterruptedBuild(t *testing.T) {
	if os.Getenv("KFP_RETENTION_MYSQL_INTERRUPTION_TEST") != "true" {
		t.Skip("KFP_RETENTION_MYSQL_INTERRUPTION_TEST not enabled")
	}
	runMySQLRetentionInterruptedBuild(t, "")
}

// The external harness kills and restarts only its disposable MySQL container
// after this test writes the readiness file. Never point this at a shared server.
func TestRetentionIndexesMySQLCrashRecovery(t *testing.T) {
	readyFile := os.Getenv("KFP_RETENTION_MYSQL_CRASH_READY_FILE")
	if readyFile == "" {
		t.Skip("KFP_RETENTION_MYSQL_CRASH_READY_FILE not set")
	}
	runMySQLRetentionInterruptedBuild(t, readyFile)
}

func runMySQLRetentionInterruptedBuild(t *testing.T, readyFile string) {
	db := retentionDatabase(t, "mysql")
	_, err := db.Exec("INSERT INTO run_details VALUES (LPAD('0',36,'0'),'AVAILABLE',1,0,REPEAT('a',1024))")
	require.NoError(t, err)
	for n := 1; n < 262144; n *= 2 {
		_, err = db.Exec("INSERT INTO run_details SELECT LPAD(CAST(CAST(UUID AS UNSIGNED)+? AS CHAR),36,'0'),StorageState,FinishedAtInSec,ArchivedAtInSec,payload FROM run_details", n)
		require.NoError(t, err)
	}
	var before, after uint64
	require.NoError(t, db.QueryRow("SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME=CONCAT(DATABASE(),'/run_details')").Scan(&before))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	buildCtx, interrupt := context.WithCancel(ctx)
	defer interrupt()
	done := make(chan error, 1)
	go func() {
		_, err := prepareGarbageCollectorIndexes(buildCtx, db, sqldrv.NewDBDialect("mysql"), 4*time.Minute)
		done <- err
	}()
	var stage string
	var completed, estimated uint64
	interrupted := false
	var interruptedAt time.Time
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("build ended before a measured mid-build interruption: %v; enable InnoDB ALTER stage instrumentation", err)
		default:
		}
		err := db.QueryRowContext(ctx, `SELECT s.EVENT_NAME, s.WORK_COMPLETED, s.WORK_ESTIMATED FROM performance_schema.events_stages_current s JOIN performance_schema.threads t ON t.THREAD_ID=s.THREAD_ID WHERE t.PROCESSLIST_DB=DATABASE() AND t.PROCESSLIST_INFO LIKE 'ALTER TABLE%idx_run_gc_lifecycle%' AND s.EVENT_NAME LIKE 'stage/innodb/alter%' AND s.WORK_ESTIMATED > 0 AND s.WORK_COMPLETED >= s.WORK_ESTIMATED / 4 AND s.WORK_COMPLETED <= s.WORK_ESTIMATED * 3 / 4 LIMIT 1`).Scan(&stage, &completed, &estimated)
		if err == sql.ErrNoRows {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		require.NoError(t, err)
		// Activity on both sides proves this cycle completed before cancellation,
		// while index construction was still active.
		active, err := mysqlRetentionActiveBuild(ctx, db)
		require.NoError(t, err)
		require.NoError(t, mysqlRetentionWriteRead(ctx, db))
		stillActive, err := mysqlRetentionActiveBuild(ctx, db)
		require.NoError(t, err)
		if strings.Contains(active, "idx_run_gc_lifecycle") && strings.Contains(stillActive, "idx_run_gc_lifecycle") {
			interruptedAt = time.Now()
			if readyFile == "" {
				interrupt()
			} else {
				require.NoError(t, os.WriteFile(readyFile, []byte("actual index work and overlapping write observed\n"), 0600))
			}
			interrupted = true
			break
		}
	}
	require.True(t, interrupted, "must interrupt actual index construction, not metadata-lock acquisition")
	t.Logf("interruption requested during %s: work %d/%d; waiting for original builder to return", stage, completed, estimated)
	var buildErr error
	select {
	case buildErr = <-done:
	case <-ctx.Done():
		t.Fatalf("original builder did not return after interruption: %v", ctx.Err())
	}
	if readyFile == "" {
		require.ErrorIs(t, buildErr, context.Canceled)
	} else {
		require.Error(t, buildErr, "the crash must interrupt the original builder")
	}
	// The single read/write cycle before interruption committed an increment.
	// Crash recovery must preserve that acknowledged application write.
	require.Eventually(t, func() bool {
		probeCtx, cancelProbe := context.WithTimeout(ctx, 2*time.Second)
		defer cancelProbe()
		return db.PingContext(probeCtx) == nil
	}, 2*time.Minute, 100*time.Millisecond)
	t.Logf("database reachable %s after interruption (includes server cancellation or crash/restart)", time.Since(interruptedAt))
	var persisted int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT FinishedAtInSec FROM run_details WHERE UUID=LPAD('0',36,'0')").Scan(&persisted))
	require.GreaterOrEqual(t, persisted, int64(2), "the acknowledged pre-interruption write must survive recovery")
	t.Logf("interrupted actual build at %s: work %d/%d, after an overlapping read/write cycle", stage, completed, estimated)

	// No cleanup SQL: retry the same production preparer after server-side
	// cancellation/rollback. A lingering first session must retain ownership.
	recoveryDone := make(chan error, 1)
	go func() {
		for {
			_, err := prepareGarbageCollectorIndexes(ctx, db, sqldrv.NewDBDialect("mysql"), 4*time.Minute)
			if err == nil || !strings.Contains(err.Error(), "another replica") {
				recoveryDone <- err
				return
			}
			select {
			case <-ctx.Done():
				recoveryDone <- ctx.Err()
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()
	during := map[string]int{}
	longest := time.Duration(0)
monitor:
	for {
		select {
		case err := <-recoveryDone:
			require.NoError(t, err)
			break monitor
		default:
		}
		active, err := mysqlRetentionActiveBuild(ctx, db)
		require.NoError(t, err)
		started := time.Now()
		require.NoError(t, mysqlRetentionWriteRead(ctx, db))
		if elapsed := time.Since(started); elapsed > longest {
			longest = elapsed
		}
		stillActive, err := mysqlRetentionActiveBuild(ctx, db)
		require.NoError(t, err)
		for _, spec := range garbageCollectorRequiredIndexes {
			if strings.Contains(active, spec.name) && strings.Contains(stillActive, spec.name) {
				during[spec.name]++
			}
		}
		time.Sleep(time.Millisecond)
	}
	// MySQL may finish the canceled build before noticing the disconnect. In
	// either outcome the second index must be built and writes must overlap it.
	require.Positive(t, during["idx_run_gc_archived"])
	if readyFile != "" {
		require.Positive(t, during["idx_run_gc_lifecycle"], "writes must complete during reconstruction of the crash-interrupted index")
	}
	changed, err := prepareGarbageCollectorIndexes(ctx, db, sqldrv.NewDBDialect("mysql"), time.Minute)
	require.NoError(t, err)
	require.False(t, changed, "recovery must leave both indexes ready without manual SQL")
	require.NoError(t, db.QueryRowContext(ctx, "SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME=CONCAT(DATABASE(),'/run_details')").Scan(&after))
	require.Equal(t, before, after)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM run_details").Scan(&count))
	require.Equal(t, 262144, count)
	t.Logf("automatic recovery read/write cycles overlapping active builds=%v; maximum cycle=%s; rows=%d; unchanged TABLE_ID=%d", during, longest, count, after)
}

func mysqlRetentionActiveBuild(ctx context.Context, db *sql.DB) (string, error) {
	var active string
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(INFO),'') FROM information_schema.PROCESSLIST WHERE DB=DATABASE() AND INFO LIKE 'ALTER TABLE%idx_run_gc_%'").Scan(&active)
	return active, err
}

func mysqlRetentionWriteRead(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "UPDATE run_details SET FinishedAtInSec=FinishedAtInSec+1 WHERE UUID=LPAD('0',36,'0')"); err != nil {
		return err
	}
	var finished int64
	return db.QueryRowContext(ctx, "SELECT FinishedAtInSec FROM run_details WHERE UUID=LPAD('0',36,'0')").Scan(&finished)
}
