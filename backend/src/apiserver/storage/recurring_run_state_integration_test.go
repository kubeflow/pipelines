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

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRecurringRunProductionDatabases uses disposable databases supplied through
// KFP_RECURRING_{MYSQL,POSTGRES}_TEST_DSN. It creates and removes only a uniquely
// named database (MySQL) or schema (PostgreSQL). The role needs DDL privileges
// and read access to lock-wait metadata (performance_schema on MySQL).
func TestRecurringRunProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			placeholder := "?"
			if driver == "pgx" {
				placeholder = "$1"
			}
			clock := util.NewFakeTimeForEpoch()
			jobs := make([]*JobStore, len(dbs))
			runs := make([]*RunStore, len(dbs))
			for i, db := range dbs {
				jobs[i] = NewJobStore(db, clock, nil, d)
				runs[i] = NewRunStore(db, clock, d)
			}
			createJob := func(t *testing.T, id string) {
				_, err := jobs[0].CreateJob(&model.Job{UUID: id, DisplayName: id, Namespace: "test", Enabled: true, MaxConcurrency: 1})
				require.NoError(t, err)
			}
			runFor := func(id, key string) *model.Run {
				return &model.Run{UUID: util.NewDeterministicUUID(id + "/tick/1"), DisplayName: key, RecurringRunId: id,
					Namespace: "test", RunDetails: model.RunDetails{State: model.RuntimeStateRunning}}
			}
			for _, sameKey := range []bool{false, true} {
				t.Run(fmt.Sprintf("concurrent-claims/same-key-%v", sameKey), func(t *testing.T) {
					id := fmt.Sprintf("claims-%v", sameKey)
					createJob(t, id)
					claims := make([]*model.RecurringRunState, len(dbs))
					errs := recurringConcurrent(len(dbs), func(i int) error {
						key := "tick"
						if !sameKey {
							key = fmt.Sprintf("tick-%d", i)
						}
						var err error
						claims[i], err = jobs[i].ClaimRecurringRun(id, key, 0, 100, int64(110+i), "version-a")
						return err
					})
					stored, err := jobs[0].GetRecurringRunState(id)
					require.NoError(t, err)
					require.Equal(t, int64(1), stored.LastRunIndex)
					successes := 0
					for i, err := range errs {
						if err == nil {
							successes++
							require.Equal(t, stored, claims[i])
						} else {
							require.ErrorContains(t, err, "pending")
						}
					}
					if sameKey {
						require.Equal(t, len(dbs), successes)
					} else {
						require.Equal(t, 1, successes)
					}
					// Concurrent retries persist one deterministic run and one completed tombstone.
					errs = recurringConcurrent(len(dbs), func(i int) error {
						_, err := runs[i].CreateRun(runFor(id, stored.RequestKey))
						return err
					})
					for _, err := range errs {
						require.NoError(t, err)
					}
					var count int
					require.NoError(t, dbs[0].QueryRow("SELECT COUNT(*) FROM "+d.QuoteIdentifier("run_details")+" WHERE "+d.QuoteIdentifier("JobUUID")+" = "+placeholder, id).Scan(&count))
					require.Equal(t, 1, count)
					completed, err := jobs[0].GetRecurringRunState(id)
					require.NoError(t, err)
					require.False(t, completed.Pending)
					// Every independent session must observe the committed active execution.
					errs = recurringConcurrent(len(dbs), func(i int) error {
						_, err := jobs[i].ClaimRecurringRun(id, "next-tick", 1, 200, 210, "")
						return err
					})
					for _, err := range errs {
						require.ErrorContains(t, err, "maximum concurrency")
					}
					require.NoError(t, runs[0].DeleteRun(runFor(id, stored.RequestKey).UUID))
					_, err = jobs[1].ClaimRecurringRun(id, stored.RequestKey, 1, 200, 210, "")
					require.ErrorContains(t, err, "already completed")
					next, err := jobs[1].ClaimRecurringRun(id, "next-tick", 1, 200, 210, "")
					require.NoError(t, err)
					require.Equal(t, int64(2), next.LastRunIndex)
				})
			}
			for _, disable := range []bool{false, true} {
				t.Run(fmt.Sprintf("blocked-claim/disable-%v", disable), func(t *testing.T) {
					id := fmt.Sprintf("blocked-%v", disable)
					createJob(t, id)
					var sessionID int64
					sessionQuery := "SELECT CONNECTION_ID()"
					if driver == "pgx" {
						sessionQuery = "SELECT pg_backend_pid()"
					}
					require.NoError(t, dbs[1].QueryRow(sessionQuery).Scan(&sessionID))
					tx, err := dbs[0].Begin()
					require.NoError(t, err)
					defer tx.Rollback()
					q := d.QuoteIdentifier
					qb := d.QueryBuilder()
					if disable {
						query, args, err := qb.Update(q("jobs")).Set(q("Enabled"), false).Where(sq.Eq{q("UUID"): id}).ToSql()
						require.NoError(t, err)
						_, err = tx.Exec(query, args...)
						require.NoError(t, err)
					} else {
						// Emulate the completion transaction: the state lock hides
						// the new active run until both changes commit together.
						query, args, err := qb.Update(q("recurring_run_states")).Set(q("Pending"), false).Where(sq.Eq{q("JobUUID"): id}).ToSql()
						require.NoError(t, err)
						_, err = tx.Exec(query, args...)
						require.NoError(t, err)
						run := runFor(id, "active")
						var dialector gorm.Dialector
						if driver == "mysql" {
							dialector = mysql.New(mysql.Config{Conn: tx, SkipInitializeWithVersion: true})
						} else {
							dialector = postgres.New(postgres.Config{Conn: tx})
						}
						orm, err := gorm.Open(dialector, &gorm.Config{SkipDefaultTransaction: true})
						require.NoError(t, err)
						require.NoError(t, orm.Create(run).Error)
					}
					result := make(chan error, 1)
					go func() { _, err := jobs[1].ClaimRecurringRun(id, "next-tick", 0, 100, 110, ""); result <- err }()
					// Observe an actual database lock wait; elapsed time alone is
					// not evidence that the contender reached the locked row.
					recurringWaitForLock(t, dbs[2], driver, sessionID)
					require.NoError(t, tx.Commit())
					select {
					case err := <-result:
						if disable {
							require.ErrorContains(t, err, "disabled")
						} else {
							require.ErrorContains(t, err, "maximum concurrency")
						}
					case <-time.After(10 * time.Second):
						t.Fatal("claim did not finish after lock release")
					}
					state, err := jobs[3].GetRecurringRunState(id)
					require.NoError(t, err)
					require.Zero(t, state.LastRunIndex)
					require.False(t, state.Pending)
				})
			}
			t.Run("completion-rollback-and-reporter-recovery", func(t *testing.T) {
				const id = "rollback"
				createJob(t, id)
				claim, err := jobs[0].ClaimRecurringRun(id, "tick", 0, 100, 110, "version-a")
				require.NoError(t, err)
				if driver == "mysql" {
					_, err = dbs[0].Exec("CREATE TRIGGER fail_completion BEFORE UPDATE ON recurring_run_states FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'simulated completion failure'")
				} else {
					_, err = dbs[0].Exec(`CREATE FUNCTION fail_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated completion failure'; END $$`)
					require.NoError(t, err)
					_, err = dbs[0].Exec(`CREATE TRIGGER fail_completion BEFORE UPDATE ON recurring_run_states FOR EACH ROW EXECUTE FUNCTION fail_completion()`)
				}
				require.NoError(t, err)
				run := runFor(id, "tick")
				_, err = runs[1].CreateRun(run)
				require.ErrorContains(t, err, "simulated completion failure")
				stored, err := jobs[2].GetRecurringRunState(id)
				require.NoError(t, err)
				require.Equal(t, claim, stored)
				var count int
				require.NoError(t, dbs[2].QueryRow("SELECT COUNT(*) FROM "+d.QuoteIdentifier("run_details")+" WHERE "+d.QuoteIdentifier("UUID")+" = "+placeholder, run.UUID).Scan(&count))
				require.Zero(t, count)
				drop := "DROP TRIGGER fail_completion"
				if driver == "pgx" {
					drop += " ON recurring_run_states"
				}
				_, err = dbs[0].Exec(drop)
				require.NoError(t, err)
				// A persistence-agent report recovers the reserved identity and timestamps.
				recovered := runFor(id, "tick")
				recovered.K8SName = "run-" + util.NewDeterministicUUID(recovered.UUID)
				recovered.DisplayName = recovered.K8SName
				recovered.CreatedAtInSec = 999
				_, err = runs[3].CreateRun(recovered)
				require.NoError(t, err)
				persisted, err := runs[4].GetRun(recovered.UUID)
				require.NoError(t, err)
				require.Equal(t, "tick", persisted.DisplayName)
				require.Equal(t, "version-a", persisted.PipelineVersionId)
				require.Equal(t, int64(100), persisted.ScheduledAtInSec)
				require.Equal(t, int64(110), persisted.CreatedAtInSec)
				stored, err = jobs[5].GetRecurringRunState(id)
				require.NoError(t, err)
				require.False(t, stored.Pending)
			})
		})
	}
}

func recurringConcurrent(n int, fn func(int) error) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; errs[i] = fn(i) }()
	}
	close(start)
	wg.Wait()
	return errs
}

func recurringIntegrationDatabases(t *testing.T, driver string) ([]*sql.DB, dialect.DBDialect) {
	t.Helper()
	env := "KFP_RECURRING_MYSQL_TEST_DSN"
	if driver == "pgx" {
		env = "KFP_RECURRING_POSTGRES_TEST_DSN"
	}
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip("set " + env + " for production-database concurrency coverage")
	}
	d := dialect.NewDBDialect(driver)
	name := fmt.Sprintf("kfp_recurring_%d", time.Now().UnixNano())
	var admin *sql.DB
	var open func() *sql.DB
	var ormDialector func(*sql.DB) gorm.Dialector
	var create, drop string
	if driver == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		config.DBName = ""
		admin, err = sql.Open(driver, config.FormatDSN())
		require.NoError(t, err)
		create = "CREATE DATABASE " + d.QuoteIdentifier(name) + " CHARACTER SET utf8mb4"
		drop = "DROP DATABASE " + d.QuoteIdentifier(name)
		config.DBName = name
		open = func() *sql.DB { db, err := sql.Open(driver, config.FormatDSN()); require.NoError(t, err); return db }
		ormDialector = func(db *sql.DB) gorm.Dialector { return mysql.New(mysql.Config{Conn: db}) }
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin = stdlib.OpenDB(*config)
		create = "CREATE SCHEMA " + d.QuoteIdentifier(name)
		drop = "DROP SCHEMA " + d.QuoteIdentifier(name) + " CASCADE"
		config.RuntimeParams["search_path"] = name
		open = func() *sql.DB { return stdlib.OpenDB(*config) }
		ormDialector = func(db *sql.DB) gorm.Dialector { return postgres.New(postgres.Config{Conn: db}) }
	}
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	_, err := admin.Exec(create)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec(drop); require.NoError(t, err) })
	dbs := make([]*sql.DB, 12)
	// Independent pools each hold one live connection: concurrency cannot silently
	// collapse into SQLite's single-connection test serialization.
	for i := range dbs {
		db := open()
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		require.NoError(t, db.Ping())
		var isolation string
		if driver == "mysql" {
			_, err := db.Exec("SET SESSION innodb_lock_wait_timeout = 15")
			require.NoError(t, err)
			require.NoError(t, db.QueryRow("SELECT @@transaction_isolation").Scan(&isolation))
			require.Equal(t, "REPEATABLE-READ", isolation)
		} else {
			_, err := db.Exec("SET statement_timeout = '15s'")
			require.NoError(t, err)
			require.NoError(t, db.QueryRow("SHOW transaction_isolation").Scan(&isolation))
			require.Equal(t, "read committed", isolation)
		}
		dbs[i] = db
	}
	orm, err := gorm.Open(ormDialector(dbs[0]), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))
	return dbs, d
}

func recurringWaitForLock(t *testing.T, db *sql.DB, driver string, sessionID int64) {
	t.Helper()
	query := `SELECT COUNT(*) FROM performance_schema.data_lock_waits w JOIN performance_schema.threads t ON t.THREAD_ID = w.REQUESTING_THREAD_ID WHERE t.PROCESSLIST_ID = ?`
	if driver == "pgx" {
		query = `SELECT COUNT(*) FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock'`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, query, sessionID).Scan(&count))
		if count > 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("claim never reached the database row lock")
		}
	}
}
