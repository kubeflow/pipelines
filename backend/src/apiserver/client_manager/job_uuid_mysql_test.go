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
	"sync/atomic"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Requires CREATE DATABASE and PROCESS privileges on a disposable MySQL 8 server.
// Each test owns a separate database and removes it on completion.
func jobUUIDMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("KFP_RECURRING_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("KFP_RECURRING_MYSQL_TEST_DSN not set")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	require.NoError(t, err)
	admin, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	name := fmt.Sprintf("kfp_jobuuid_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec("DROP DATABASE `" + name + "`")
		require.NoError(t, err)
		require.NoError(t, admin.Close())
	})
	cfg.DBName = name
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: cfg.FormatDSN(), DefaultStringSize: 255}), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, preserveRunJobUUID(db).AutoMigrate(model.AllModels()...))
	return db
}

func jobUUIDTableID(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()
	var id uint64
	require.NoError(t, db.Raw("SELECT TABLE_ID FROM information_schema.INNODB_TABLES WHERE NAME = CONCAT(DATABASE(), '/run_details')").Scan(&id).Error)
	require.NotZero(t, id)
	return id
}

func TestJobUUIDMySQLPreservesExistingSchema(t *testing.T) {
	for _, columnType := range []string{"varchar(255)", "varchar(191)", "longtext"} {
		t.Run(columnType, func(t *testing.T) {
			db := jobUUIDMySQL(t)
			require.NoError(t, db.Exec("ALTER TABLE run_details MODIFY COLUMN JobUUID "+columnType+" DEFAULT NULL").Error)
			association := strings.Repeat("a", 191)
			if columnType != "varchar(191)" {
				association += strings.Repeat("b", 64)
			}
			require.NoError(t, db.Create(&model.Run{UUID: "preserved", RecurringRunId: association}).Error)
			before := jobUUIDTableID(t, db)
			for range 2 {
				require.NoError(t, autoMigrate(db))
				require.NoError(t, ensureRecurringRunIndex(db))
				var gotType, gotValue string
				require.NoError(t, db.Raw("SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='run_details' AND COLUMN_NAME='JobUUID'").Scan(&gotType).Error)
				require.Equal(t, columnType, gotType)
				require.NoError(t, db.Raw("SELECT JobUUID FROM run_details WHERE UUID='preserved'").Scan(&gotValue).Error)
				require.Equal(t, association, gotValue)
				require.Equal(t, before, jobUUIDTableID(t, db), "InnoDB table identity must survive migration")
			}
			if columnType != "varchar(191)" {
				other := strings.Repeat("a", 191) + "different-suffix"
				require.NoError(t, db.Create(&model.Run{UUID: "same-prefix", RecurringRunId: other}).Error)
				var count int64
				require.NoError(t, db.Raw("SELECT COUNT(*) FROM run_details FORCE INDEX (idx_run_details_job_uuid) WHERE JobUUID = ?", association).Scan(&count).Error)
				require.EqualValues(t, 1, count, "prefix index must not change full-value equality")
			}
			t.Logf("%s preserved, InnoDB TABLE_ID=%d unchanged after two migrations", columnType, before)
		})
	}
}

func TestJobUUIDMySQLOnlineIndexAllowsWrites(t *testing.T) {
	db := jobUUIDMySQL(t)
	// A sufficiently populated index makes concurrent activity observable.
	require.NoError(t, db.Create(&model.Run{UUID: "0", RecurringRunId: strings.Repeat("a", 255)}).Error)
	columns, err := db.Migrator().ColumnTypes(&model.Run{})
	require.NoError(t, err)
	var names, expressions []string
	for _, column := range columns {
		name := "`" + column.Name() + "`"
		names = append(names, name)
		switch column.Name() {
		case "UUID":
			expressions = append(expressions, "CAST(CAST(UUID AS UNSIGNED) + ? AS CHAR)")
		case "JobUUID":
			expressions = append(expressions, "CONCAT(LPAD(CAST(CAST(UUID AS UNSIGNED) + ? AS CHAR), 16, '0'), REPEAT('a',239))")
		default:
			expressions = append(expressions, name)
		}
	}
	for n := 1; n < 262144; n *= 2 {
		require.NoError(t, db.Exec("INSERT INTO run_details ("+strings.Join(names, ",")+") SELECT "+strings.Join(expressions, ",")+" FROM run_details", n, n).Error)
	}
	before := jobUUIDTableID(t, db)
	pool, err := db.DB()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var duringBuild atomic.Int64
	var longestWrite atomic.Int64
	writerDone := make(chan error, 1)
	go func() {
		for ctx.Err() == nil {
			var active int
			err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE DB=DATABASE() AND INFO LIKE 'ALTER TABLE run_details ADD INDEX%'").Scan(&active)
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				writerDone <- err
				return
			}
			start := time.Now()
			_, err = pool.ExecContext(ctx, "UPDATE run_details SET DisplayName=? WHERE UUID='0'", fmt.Sprint(time.Now().UnixNano()))
			elapsed := time.Since(start).Nanoseconds()
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				writerDone <- err
				return
			}
			if active > 0 {
				// Count only writes that finish while the DDL is still active.
				// A write released by a completed blocking build is not evidence.
				err = pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE DB=DATABASE() AND INFO LIKE 'ALTER TABLE run_details ADD INDEX%'").Scan(&active)
				if err != nil {
					if ctx.Err() != nil {
						break
					}
					writerDone <- err
					return
				}
				if active > 0 {
					duringBuild.Add(1)
				}
			}
			for old := longestWrite.Load(); elapsed > old; old = longestWrite.Load() {
				if longestWrite.CompareAndSwap(old, elapsed) {
					break
				}
			}
		}
		writerDone <- nil
	}()
	started := time.Now()
	migrationErr := autoMigrate(db)
	if migrationErr == nil {
		migrationErr = ensureRecurringRunIndex(db)
	}
	elapsed := time.Since(started)
	cancel()
	require.NoError(t, <-writerDone)
	require.NoError(t, migrationErr)
	require.Positive(t, duringBuild.Load(), "must observe successful writes while ADD INDEX is executing")
	require.Less(t, time.Duration(longestWrite.Load()), 5*time.Second)
	require.Equal(t, before, jobUUIDTableID(t, db), "online secondary-index creation must not rebuild the clustered table")
	t.Logf("262144 rows: migration=%s, writes while ADD INDEX active=%d, maximum write latency=%s, unchanged TABLE_ID=%d", elapsed, duringBuild.Load(), time.Duration(longestWrite.Load()), before)
	// Negative control: the removed narrowing cannot satisfy the online contract.
	err = db.Exec("ALTER TABLE run_details MODIFY COLUMN JobUUID varchar(191) DEFAULT NULL, ALGORITHM=INPLACE, LOCK=NONE").Error
	require.ErrorContains(t, err, "ALGORITHM=INPLACE is not supported")
}

func TestJobUUIDMySQLLockContentionAndRetry(t *testing.T) {
	db := jobUUIDMySQL(t)
	pool, err := db.DB()
	require.NoError(t, err)
	blocker, err := pool.Begin()
	require.NoError(t, err)
	defer blocker.Rollback()
	// Hold a shared metadata lock until the transaction ends.
	_, err = blocker.Exec("SELECT UUID FROM run_details LIMIT 1")
	require.NoError(t, err)
	start := time.Now()
	err = ensureRecurringRunIndex(db)
	require.ErrorContains(t, err, "no copying or write-blocking fallback")
	require.Less(t, time.Since(start), 10*time.Second)
	require.False(t, db.Migrator().HasIndex(&model.Run{}, recurringRunIndexName))
	require.NoError(t, blocker.Rollback())
	require.NoError(t, ensureRecurringRunIndex(db), "retry must succeed after metadata lock is released")
	t.Logf("metadata lock wait failed safely and retry created index")
}

func TestJobUUIDMySQLConcurrentIndexPreparation(t *testing.T) {
	db := jobUUIDMySQL(t)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- ensureRecurringRunIndex(db) }()
	}
	for range 2 {
		require.NoError(t, <-results)
	}
	require.True(t, db.Migrator().HasIndex(&model.Run{}, recurringRunIndexName))
	// Existing full-width indexes from the earlier implementation remain usable.
	require.NoError(t, db.Exec("DROP INDEX idx_run_details_job_uuid ON run_details").Error)
	require.NoError(t, db.Exec("CREATE INDEX idx_run_details_job_uuid ON run_details(JobUUID)").Error)
	require.NoError(t, ensureRecurringRunIndex(db))
	var prefix sql.NullInt64
	require.NoError(t, db.Raw("SELECT SUB_PART FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='run_details' AND INDEX_NAME='idx_run_details_job_uuid'").Scan(&prefix).Error)
	require.False(t, prefix.Valid, "existing full-width index must not be replaced")
}
