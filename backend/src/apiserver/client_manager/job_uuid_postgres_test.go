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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func jobUUIDPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("KFP_RECURRING_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("KFP_RECURRING_POSTGRES_TEST_DSN not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin := stdlib.OpenDB(*cfg)
	name := fmt.Sprintf("kfp_jobuuid_%d", time.Now().UnixNano())
	_, err = admin.Exec(`CREATE SCHEMA "` + name + `"`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec(`DROP SCHEMA "` + name + `" CASCADE`)
		require.NoError(t, err)
		require.NoError(t, admin.Close())
	})
	cfg.RuntimeParams["search_path"] = name
	pool := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestJobUUIDPostgresPreservesExistingSchema(t *testing.T) {
	for _, columnType := range []string{"character varying(255)", "character varying(191)", "text"} {
		t.Run(columnType, func(t *testing.T) {
			db := jobUUIDPostgres(t)
			require.NoError(t, preserveRunJobUUID(db).AutoMigrate(model.AllModels()...))
			require.NoError(t, db.Exec(`ALTER TABLE run_details ALTER COLUMN "JobUUID" TYPE `+columnType).Error)
			association := strings.Repeat("a", 191)
			if columnType != "character varying(191)" {
				association += strings.Repeat("b", 64)
			}
			require.NoError(t, db.Create(&model.Run{UUID: "preserved", RecurringRunId: association}).Error)
			for range 2 {
				require.NoError(t, autoMigrate(db))
				var gotType, gotValue string
				require.NoError(t, db.Raw(`SELECT format_type(a.atttypid, a.atttypmod) FROM pg_attribute a WHERE a.attrelid = 'run_details'::regclass AND a.attname = 'JobUUID'`).Scan(&gotType).Error)
				require.Equal(t, columnType, gotType)
				require.NoError(t, db.Raw(`SELECT "JobUUID" FROM run_details WHERE "UUID" = 'preserved'`).Scan(&gotValue).Error)
				require.Equal(t, association, gotValue)
				pool, err := db.DB()
				require.NoError(t, err)
				conn, err := pool.Conn(context.Background())
				require.NoError(t, err)
				ready, err := recurringRunIndexReady(context.Background(), conn, "postgres")
				require.NoError(t, conn.Close())
				require.NoError(t, err)
				require.True(t, ready)
			}
		})
	}
}

func TestJobUUIDPostgresRejectsConflictingAndInterruptedIndexes(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		db := jobUUIDPostgres(t)
		require.NoError(t, db.Exec(`CREATE TABLE run_details ("UUID" text, "JobUUID" varchar(255))`).Error)
		require.NoError(t, db.Exec(`CREATE INDEX idx_run_details_job_uuid ON run_details ("UUID")`).Error)
		require.ErrorContains(t, ensureRecurringRunIndex(db), "unexpected definition")
		var definition string
		require.NoError(t, db.Raw(`SELECT pg_get_indexdef('idx_run_details_job_uuid'::regclass)`).Scan(&definition).Error)
		require.Contains(t, definition, `("UUID")`)
	})
	t.Run("interrupted concurrent build", func(t *testing.T) {
		db := jobUUIDPostgres(t)
		require.NoError(t, db.Exec(`CREATE TABLE run_details ("UUID" text, "JobUUID" varchar(255))`).Error)
		pool, err := db.DB()
		require.NoError(t, err)
		tx, err := pool.Begin()
		require.NoError(t, err)
		defer tx.Rollback()
		_, err = tx.Exec(`INSERT INTO run_details VALUES ('run', 'schedule')`)
		require.NoError(t, err)
		builder, err := pool.Conn(context.Background())
		require.NoError(t, err)
		defer builder.Close()
		var builderPID int
		require.NoError(t, builder.QueryRowContext(context.Background(), `SELECT pg_backend_pid()`).Scan(&builderPID))
		// Concurrent creation waits for this writer transaction. Client timeout
		// returns before pgx finishes asynchronously canceling the backend.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err = builder.ExecContext(ctx, recurringRunIndexDDL("postgres"))
		require.ErrorIs(t, err, context.DeadlineExceeded)
		// Keep the writer blocked until cancellation actually stops the build,
		// otherwise the index can become valid before cancellation arrives.
		waitCtx, stopWaiting := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopWaiting()
		require.Eventually(t, func() bool {
			var active bool
			err := pool.QueryRowContext(waitCtx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1)`, builderPID).Scan(&active)
			return err == nil && !active
		}, 5*time.Second, 10*time.Millisecond, "canceled index builder must exit before releasing the writer")
		require.NoError(t, tx.Rollback())
		var valid bool
		require.NoError(t, db.Raw(`SELECT indisvalid FROM pg_index WHERE indexrelid = 'idx_run_details_job_uuid'::regclass`).Scan(&valid).Error)
		require.False(t, valid)
		require.ErrorContains(t, ensureRecurringRunIndex(db), "invalid or has an unexpected definition")
		// Refusal preserves the index for inspection instead of silently replacing it.
		require.NoError(t, db.Raw(`SELECT indisvalid FROM pg_index WHERE indexrelid = 'idx_run_details_job_uuid'::regclass`).Scan(&valid).Error)
		require.False(t, valid)
	})
}

func TestJobUUIDPostgresPreparationLockIsSchemaScoped(t *testing.T) {
	first := jobUUIDPostgres(t)
	second := jobUUIDPostgres(t)
	for _, db := range []*gorm.DB{first, second} {
		require.NoError(t, db.Exec(`CREATE TABLE run_details ("UUID" text, "JobUUID" varchar(255))`).Error)
	}
	pool, err := first.DB()
	require.NoError(t, err)
	conn, err := pool.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	release, err := lockRecurringRunIndex(context.Background(), conn, "postgres")
	require.NoError(t, err)
	defer release()
	require.ErrorContains(t, ensureRecurringRunIndex(first), "another replica")
	require.NoError(t, ensureRecurringRunIndex(second), "independent schemas must not share the build lock")
}
