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
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/stretchr/testify/require"
)

// This small scripted driver exercises session ownership/cleanup and verifies
// that incompatible indexes never trigger destructive repair or fallback DDL.
type retentionDBStep struct {
	query  string
	values []driver.Value
	err    error
	cancel context.CancelFunc
}
type retentionDBScript struct {
	t      *testing.T
	steps  []retentionDBStep
	closed bool
}

func (s *retentionDBScript) Connect(context.Context) (driver.Conn, error) { return s, nil }
func (s *retentionDBScript) Driver() driver.Driver                        { return s }
func (s *retentionDBScript) Open(string) (driver.Conn, error)             { return s, nil }
func (s *retentionDBScript) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (s *retentionDBScript) Close() error { s.closed = true; return nil }
func (s *retentionDBScript) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (s *retentionDBScript) next(ctx context.Context, query string) retentionDBStep {
	s.t.Helper()
	require.NoError(s.t, ctx.Err(), "session cleanup must use a fresh context")
	require.NotEmpty(s.t, s.steps, "unexpected query: %s", query)
	step := s.steps[0]
	s.steps = s.steps[1:]
	require.Contains(s.t, query, step.query)
	if step.cancel != nil {
		step.cancel()
	}
	return step
}
func (s *retentionDBScript) QueryContext(ctx context.Context, q string, _ []driver.NamedValue) (driver.Rows, error) {
	step := s.next(ctx, q)
	if step.err != nil {
		return nil, step.err
	}
	return &retentionDBRows{values: step.values}, nil
}
func (s *retentionDBScript) ExecContext(ctx context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	step := s.next(ctx, q)
	return driver.RowsAffected(0), step.err
}

type retentionDBRows struct {
	values []driver.Value
	done   bool
}

func (r *retentionDBRows) Columns() []string {
	c := make([]string, len(r.values))
	for i := range c {
		c[i] = "value"
	}
	return c
}
func (r *retentionDBRows) Close() error { return nil }
func (r *retentionDBRows) Next(dest []driver.Value) error {
	if r.done || r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.done = true
	return nil
}
func retentionScriptDB(t *testing.T, steps ...retentionDBStep) (*sql.DB, *retentionDBScript) {
	t.Helper()
	s := &retentionDBScript{t: t, steps: steps}
	db := sql.OpenDB(s)
	t.Cleanup(func() { require.Empty(t, s.steps); db.Close() })
	return db, s
}
func retentionMySQLLockSteps() []retentionDBStep {
	return []retentionDBStep{
		{query: "GET_LOCK", values: []driver.Value{int64(1)}},
		{query: "@@SESSION.lock_wait_timeout", values: []driver.Value{int64(77)}},
		{query: "SET SESSION lock_wait_timeout = 5"},
	}
}
func retentionMySQLCleanupSteps() []retentionDBStep {
	return []retentionDBStep{
		{query: "SET SESSION lock_wait_timeout = 77"}, {query: "RELEASE_LOCK"},
	}
}
func retentionReadyStep(spec garbageCollectorIndexSpec) retentionDBStep {
	return retentionDBStep{query: "GROUP_CONCAT", values: []driver.Value{"kfp", "kfp", "run_details", spec.name, spec.columns, true, true, true}}
}

func TestRetentionIndexPreparationReusesReadyIndexes(t *testing.T) {
	steps := retentionMySQLLockSteps()
	for _, spec := range garbageCollectorRequiredIndexes {
		steps = append(steps, retentionReadyStep(spec))
	}
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.NoError(t, err)
	require.False(t, built)
}

func TestRetentionIndexPreparationRejectsConflictingIndex(t *testing.T) {
	steps := append(retentionMySQLLockSteps(), retentionDBStep{query: "GROUP_CONCAT"}, retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{true}})
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.ErrorContains(t, err, "was not dropped or replaced")
	require.False(t, built)
}

func TestRetentionIndexPreparationBuildsSequentially(t *testing.T) {
	steps := retentionMySQLLockSteps()
	for _, spec := range garbageCollectorRequiredIndexes {
		steps = append(steps, retentionDBStep{query: "GROUP_CONCAT"}, retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{false}}, retentionDBStep{query: garbageCollectorIndexDDL("mysql", spec)}, retentionReadyStep(spec))
	}
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.NoError(t, err)
	require.True(t, built)
}

func TestRetentionIndexPreparationNoBlockingFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	steps := append(retentionMySQLLockSteps(), retentionDBStep{query: "GROUP_CONCAT"}, retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{false}}, retentionDBStep{query: "ALGORITHM=INPLACE, LOCK=NONE", err: context.Canceled, cancel: cancel})
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(ctx, db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.ErrorContains(t, err, "no copying or write-blocking fallback")
	require.False(t, built)
}

func TestRetentionIndexLockDoesNotWaitForAnotherBuilder(t *testing.T) {
	db, _ := retentionScriptDB(t, retentionDBStep{query: ", 0)", values: []driver.Value{int64(0)}})
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	_, err = lockGarbageCollectorIndexes(context.Background(), conn, "mysql")
	require.ErrorContains(t, err, "another replica")
}

func TestRetentionIndexSessionCleanupDiscardsConnectionOnFailure(t *testing.T) {
	steps := append(retentionMySQLLockSteps(), retentionDBStep{query: "SET SESSION lock_wait_timeout = 77", err: errors.New("restore failed")}, retentionDBStep{query: "RELEASE_LOCK"})
	db, script := retentionScriptDB(t, steps...)
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	cleanup, err := lockGarbageCollectorIndexes(context.Background(), conn, "mysql")
	require.NoError(t, err)
	cleanup()
	require.True(t, script.closed)
}

func TestRetentionPostgresSessionSettingsRestored(t *testing.T) {
	db, _ := retentionScriptDB(t,
		retentionDBStep{query: "pg_try_advisory_lock", values: []driver.Value{true}},
		retentionDBStep{query: "current_setting", values: []driver.Value{"1s", "2s"}},
		retentionDBStep{query: "set_config('lock_timeout', '5s'"},
		retentionDBStep{query: "set_config('lock_timeout', $1"},
		retentionDBStep{query: "pg_advisory_unlock"})
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	cleanup, err := lockGarbageCollectorIndexes(context.Background(), conn, "pgx")
	require.NoError(t, err)
	cleanup()
	for _, spec := range garbageCollectorRequiredIndexes {
		ddl := garbageCollectorIndexDDL("pgx", spec)
		require.True(t, strings.HasPrefix(ddl, "CREATE INDEX CONCURRENTLY"))
		require.Contains(t, ddl, `"StorageState","`)
	}
}

func TestRetentionIndexPreparationPreservesCompletedBuildOnLaterFailure(t *testing.T) {
	first := garbageCollectorRequiredIndexes[0]
	steps := append(retentionMySQLLockSteps(),
		retentionDBStep{query: "GROUP_CONCAT"},
		retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{false}},
		retentionDBStep{query: garbageCollectorIndexDDL("mysql", first)},
		retentionReadyStep(first),
		retentionDBStep{query: "GROUP_CONCAT"},
		retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{false}},
		retentionDBStep{query: "ALGORITHM=INPLACE, LOCK=NONE", err: errors.New("insufficient disk space")})
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.True(t, built)
	require.ErrorContains(t, err, "insufficient disk space")
}

func TestRetentionIndexPreparationRejectsInvisibleIndex(t *testing.T) {
	spec := garbageCollectorRequiredIndexes[0]
	invisible := retentionReadyStep(spec)
	invisible.query = "IS_VISIBLE = 'YES'"
	invisible.values[7] = false
	steps := append(retentionMySQLLockSteps(), invisible,
		retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{true}})
	steps = append(steps, retentionMySQLCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("mysql"), time.Hour)
	require.False(t, built)
	require.ErrorContains(t, err, "was not dropped or replaced")
}

func retentionPostgresLockSteps() []retentionDBStep {
	return []retentionDBStep{
		{query: "pg_try_advisory_lock", values: []driver.Value{true}},
		{query: "current_setting", values: []driver.Value{"0", "0"}},
		{query: "set_config('lock_timeout', '5s'"},
	}
}

func retentionPostgresCleanupSteps() []retentionDBStep {
	return []retentionDBStep{
		{query: "set_config('lock_timeout', $1"},
		{query: "pg_advisory_unlock"},
	}
}

func retentionPostgresReadyStep(spec garbageCollectorIndexSpec) retentionDBStep {
	return retentionDBStep{query: "string_agg", values: []driver.Value{"public", "public", "run_details", spec.name, spec.columns, true, true, true}}
}

func TestRetentionPostgresPreparationRecoversInterruptedIndex(t *testing.T) {
	first := garbageCollectorRequiredIndexes[0]
	steps := append(retentionPostgresLockSteps(),
		retentionDBStep{query: "string_agg"},
		retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{true}},
		retentionDBStep{query: "idx.reloptions IS NULL AND idx.reltablespace = 0", values: []driver.Value{true, false}},
		retentionDBStep{query: `DROP INDEX CONCURRENTLY "` + first.name + `"`},
		retentionDBStep{query: garbageCollectorIndexDDL("pgx", first)},
		retentionPostgresReadyStep(first),
		retentionPostgresReadyStep(garbageCollectorRequiredIndexes[1]))
	steps = append(steps, retentionPostgresCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("pgx"), time.Hour)
	require.NoError(t, err)
	require.True(t, built)
}

func TestRetentionPostgresRecoveryRefusesUnsafeOrActiveIndexes(t *testing.T) {
	for _, tc := range []struct {
		name                string
		recoverable, active bool
		want                string
	}{
		{"mismatched definition", false, false, "was not dropped or replaced"},
		{"active external maintenance", true, true, "maintenance is active or waiting"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps := append(retentionPostgresLockSteps(),
				retentionDBStep{query: "string_agg"},
				retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{true}},
				retentionDBStep{query: "pg_get_indexdef", values: []driver.Value{tc.recoverable, tc.active}})
			steps = append(steps, retentionPostgresCleanupSteps()...)
			db, _ := retentionScriptDB(t, steps...)
			built, err := prepareGarbageCollectorIndexes(context.Background(), db, sqldrv.NewDBDialect("pgx"), time.Hour)
			require.ErrorContains(t, err, tc.want)
			require.False(t, built)
		})
	}
}

func TestRetentionPostgresRecoveryCancellationStopsBeforeRebuild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	steps := append(retentionPostgresLockSteps(),
		retentionDBStep{query: "string_agg"},
		retentionDBStep{query: "SELECT EXISTS", values: []driver.Value{true}},
		retentionDBStep{query: "pg_get_indexdef", values: []driver.Value{true, false}},
		retentionDBStep{query: "DROP INDEX CONCURRENTLY", err: context.Canceled, cancel: cancel})
	steps = append(steps, retentionPostgresCleanupSteps()...)
	db, _ := retentionScriptDB(t, steps...)
	built, err := prepareGarbageCollectorIndexes(ctx, db, sqldrv.NewDBDialect("pgx"), time.Hour)
	require.ErrorContains(t, err, "concurrent cleanup")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, built)
}
