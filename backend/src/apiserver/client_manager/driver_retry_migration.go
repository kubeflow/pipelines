// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"gorm.io/gorm"
)

const driverRetryIndex = "idx_task_driver_retry"

// These additions never change existing values. PostgreSQL 11+ stores the
// constant boolean default in its catalog; MySQL must accept INSTANT, otherwise
// startup fails rather than rebuilding a populated table.
var driverRetryColumns = []struct {
	table, name, kind string
	required          bool
}{
	{"tasks", "DriverRetryGeneration", "bigint", false},
	{"tasks", "DriverRetryAttempt", "bigint", false},
	{"tasks", "DriverStoppedGeneration", "bigint", false},
	{"tasks", "DriverCheckpoint", "json", false},
	{"tasks", "DriverCachedOutputs", "json", false},
	{"run_details", "DriverRetryTasksPresent", "boolean", true},
	{"run_details", "DriverRetryFinalizedGeneration", "bigint", false},
}

// migrateDriverRetrySchema owns only the additive recovery schema. Existing
// unrelated migrations retain their own availability requirements. Failure
// aborts startup before readiness; a later startup resumes from the catalog.
func migrateDriverRetrySchema(db *gorm.DB) error {
	engine := db.Name()
	if engine == "sqlite" {
		return nil
	}
	if engine != "mysql" && engine != "postgres" {
		return fmt.Errorf("driver retry schema: unsupported database %q", engine)
	}
	ctx, cancel := context.WithTimeout(db.Statement.Context, 2*time.Minute)
	defer cancel()
	pool, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	m := &driverRetryMigration{conn: conn, engine: engine}
	if err := m.acquire(ctx); err != nil {
		return fmt.Errorf("driver retry schema ownership: %w; retry startup after the other migration or transaction completes", err)
	}
	defer m.release()
	glog.Info("driver_retry_schema migration started; metadata lock timeout=1s, total timeout=2m")
	if err := m.run(ctx); err != nil {
		return fmt.Errorf("driver retry schema upgrade failed without a blocking fallback: %w; resolve contention or incompatible schema and restart", err)
	}
	return nil
}

type driverRetryMigration struct {
	conn     *sql.Conn
	engine   string
	lockName string
	settings []string
}

func (m *driverRetryMigration) acquire(ctx context.Context) (err error) {
	// On partial setup failure cleanup still runs, including discarding a
	// connection whose session settings or ownership cannot be restored.
	defer func() {
		if err != nil {
			m.release()
		}
	}()
	if m.engine == "postgres" {
		var version int
		if err = m.conn.QueryRowContext(ctx, "SHOW server_version_num").Scan(&version); err != nil {
			return err
		}
		if version < 110000 {
			return fmt.Errorf("PostgreSQL 11 or newer is required for metadata-only constant defaults")
		}
		// Record the attempted lock before querying: a lost response can still
		// mean this session acquired it. Cleanup must attempt the unlock.
		m.lockName = "postgres"
		var owned bool
		if err = m.conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext('kubeflow_pipelines'), hashtext('driver_retry_schema'))").Scan(&owned); err != nil {
			return err
		}
		if !owned {
			m.lockName = ""
			return errors.New("another replica owns the recovery schema migration")
		}
		for _, name := range []string{"lock_timeout", "statement_timeout"} {
			var value string
			if err = m.conn.QueryRowContext(ctx, "SHOW "+name).Scan(&value); err != nil {
				return err
			}
			m.settings = append(m.settings, value)
		}
		_, err = m.conn.ExecContext(ctx, "SET lock_timeout = '1s'; SET statement_timeout = '120s'")
		return err
	}
	if err = m.conn.QueryRowContext(ctx, "SELECT CONCAT('kfp_driver_retry_', MD5(DATABASE()))").Scan(&m.lockName); err != nil {
		return err
	}
	var owned sql.NullInt64
	if err = m.conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", m.lockName).Scan(&owned); err != nil {
		return err
	}
	if !owned.Valid || owned.Int64 != 1 {
		m.lockName = ""
		return errors.New("another replica owns the recovery schema migration")
	}
	var timeout string
	if err = m.conn.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&timeout); err != nil {
		return err
	}
	m.settings = append(m.settings, timeout)
	_, err = m.conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = 1")
	return err
}

func (m *driverRetryMigration) release() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err error
	if m.engine == "postgres" {
		for i, value := range m.settings {
			_, e := m.conn.ExecContext(ctx, "SELECT set_config($1, $2, false)", []string{"lock_timeout", "statement_timeout"}[i], value)
			err = errors.Join(err, e)
		}
		if m.lockName != "" {
			_, e := m.conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext('kubeflow_pipelines'), hashtext('driver_retry_schema'))")
			err = errors.Join(err, e)
		}
	} else {
		if len(m.settings) != 0 {
			_, e := m.conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = "+m.settings[0])
			err = errors.Join(err, e)
		}
		if m.lockName != "" {
			_, e := m.conn.ExecContext(ctx, "SELECT RELEASE_LOCK(?)", m.lockName)
			err = errors.Join(err, e)
		}
	}
	if err != nil {
		glog.Warningf("driver_retry_schema session cleanup failed; discarding connection: %v", err)
		_ = m.conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}

func (m *driverRetryMigration) quote(value string) string {
	if m.engine == "mysql" {
		return "`" + value + "`"
	}
	return `"` + value + `"`
}

func (m *driverRetryMigration) tableExists(ctx context.Context, table string) (bool, error) {
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	if m.engine == "postgres" {
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1"
	}
	var count int
	err := m.conn.QueryRowContext(ctx, query, table).Scan(&count)
	return count != 0, err
}

func (m *driverRetryMigration) run(ctx context.Context) error {
	for _, column := range driverRetryColumns {
		exists, err := m.tableExists(ctx, column.table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		query := "SELECT CASE WHEN column_type LIKE '%unsigned%' THEN CONCAT(data_type, ' unsigned') ELSE data_type END, is_nullable, column_default FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
		if m.engine == "postgres" {
			query = "SELECT data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2"
		}
		var kind, nullable string
		var defaultValue sql.NullString
		err = m.conn.QueryRowContext(ctx, query, column.table, column.name).Scan(&kind, &nullable, &defaultValue)
		if err == nil {
			expected := column.kind
			if m.engine == "mysql" && expected == "boolean" {
				expected = "tinyint"
			}
			validDefault := !defaultValue.Valid || strings.EqualFold(defaultValue.String, "NULL")
			if column.required {
				validDefault = defaultValue.Valid && (defaultValue.String == "0" || defaultValue.String == "false")
			}
			if kind != expected || (nullable == "NO") != column.required || !validDefault {
				return fmt.Errorf("incompatible existing column %s.%s; expected %s with required=%t and default false/NULL; automatic conversion is disabled", column.table, column.name, expected, column.required)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		definition := column.kind + " NULL DEFAULT NULL"
		if column.required {
			definition = "boolean NOT NULL DEFAULT false"
		}
		statement := "ALTER TABLE " + m.quote(column.table) + " ADD COLUMN " + m.quote(column.name) + " " + definition
		if m.engine == "mysql" {
			statement += ", ALGORITHM=INSTANT"
		}
		glog.Infof("driver_retry_schema adding %s.%s", column.table, column.name)
		if _, err := m.execDDL(ctx, statement); err != nil {
			return err
		}
	}
	if exists, err := m.tableExists(ctx, "run_details"); err != nil {
		return err
	} else if exists {
		// New table only; this never scans or copies run_details. Its foreign key
		// metadata lock uses the same bounded session as the column additions.
		q := m.quote
		statement := "CREATE TABLE IF NOT EXISTS driver_task_stops (" + q("StopKey") + " varchar(64) PRIMARY KEY, " + q("RunUUID") + " varchar(191) NOT NULL, CONSTRAINT " + q("driver_task_stops_RunUUID_run_details_UUID_foreign") + " FOREIGN KEY (" + q("RunUUID") + ") REFERENCES run_details (" + q("UUID") + ") ON DELETE CASCADE ON UPDATE CASCADE)"
		if _, err := m.execDDL(ctx, statement); err != nil {
			return err
		}
		if err := m.validateStopTable(ctx); err != nil {
			return err
		}
		if err := m.index(ctx, "driver_task_stops", "idx_driver_stop_run", []string{"RunUUID"}); err != nil {
			return err
		}
	}
	exists, err := m.tableExists(ctx, "tasks")
	if err != nil || !exists {
		return err
	}
	return m.index(ctx, "tasks", driverRetryIndex, []string{"RunUUID", "DriverRetryGeneration", "UUID"})
}

func (m *driverRetryMigration) index(ctx context.Context, table, index string, wanted []string) error {
	var quoted []string
	for _, column := range wanted {
		quoted = append(quoted, m.quote(column))
	}
	columnSQL := strings.Join(quoted, ", ")
	wantedColumns := strings.Join(wanted, ",")
	if m.engine == "mysql" {
		rows, err := m.conn.QueryContext(ctx, "SELECT COLUMN_NAME, NON_UNIQUE, SUB_PART, INDEX_TYPE, IS_VISIBLE FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ? ORDER BY SEQ_IN_INDEX", table, index)
		if err != nil {
			return err
		}
		var columns []string
		for rows.Next() {
			var name, kind, visible string
			var nonunique int
			var prefix sql.NullInt64
			if err := rows.Scan(&name, &nonunique, &prefix, &kind, &visible); err != nil {
				rows.Close()
				return err
			}
			if nonunique != 1 || prefix.Valid || kind != "BTREE" || visible != "YES" {
				rows.Close()
				return fmt.Errorf("incompatible index %s; preserve it and resolve its definition before retrying", index)
			}
			columns = append(columns, name)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(columns) != 0 {
			if strings.Join(columns, ",") != wantedColumns {
				return fmt.Errorf("incompatible columns in %s", index)
			}
			return nil
		}
		_, err = m.execDDL(ctx, "ALTER TABLE "+m.quote(table)+" ADD INDEX "+m.quote(index)+" ("+columnSQL+"), ALGORITHM=INPLACE, LOCK=NONE")
		return err
	}
	var valid, compatible bool
	err := m.conn.QueryRowContext(ctx, `SELECT i.indisvalid,
  NOT i.indisunique AND i.indpred IS NULL AND i.indexprs IS NULL AND am.amname = 'btree'
  AND i.indnatts = $4 AND (SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
  FROM unnest(i.indkey) WITH ORDINALITY k(attnum, ordinality)
  JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum) = $3
  FROM pg_index i JOIN pg_class idx ON idx.oid = i.indexrelid JOIN pg_am am ON am.oid = idx.relam
  WHERE i.indrelid = to_regclass($2) AND idx.relname = $1`, index, table, wantedColumns, len(wanted)).Scan(&valid, &compatible)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		if !compatible {
			return fmt.Errorf("incompatible index %s; preserve it and resolve its definition before retrying", index)
		}
		if valid {
			return nil
		}
		// Our advisory lock serializes KFP replicas. Check relation locks as well
		// before removing an invalid index left by an interrupted external build.
		var active bool
		if err := m.conn.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid <> pg_backend_pid() AND relation IN (to_regclass($2), to_regclass($1)) AND mode = 'ShareUpdateExclusiveLock')`, index, table).Scan(&active); err != nil {
			return err
		}
		if active {
			return errors.New("invalid recovery index has an active build; retry after that build finishes")
		}
		if _, err := m.execDDL(ctx, "DROP INDEX CONCURRENTLY "+m.quote(index)); err != nil {
			return err
		}
	}
	_, err = m.execDDL(ctx, "CREATE INDEX CONCURRENTLY "+m.quote(index)+" ON "+m.quote(table)+" ("+columnSQL+")")
	return err
}

// validateStopTable prevents GORM from repairing an incompatible partial or
// pre-existing table using an unconstrained ALTER after this migration returns.
func (m *driverRetryMigration) validateStopTable(ctx context.Context) error {
	for column, width := range map[string]int{"StopKey": 64, "RunUUID": 191} {
		query := "SELECT data_type, character_maximum_length, is_nullable, column_default FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'driver_task_stops' AND column_name = ?"
		expected := "varchar"
		if m.engine == "postgres" {
			query = "SELECT data_type, character_maximum_length, is_nullable, column_default FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'driver_task_stops' AND column_name = $1"
			expected = "character varying"
		}
		var kind, nullable string
		var length sql.NullInt64
		var defaultValue sql.NullString
		if err := m.conn.QueryRowContext(ctx, query, column).Scan(&kind, &length, &nullable, &defaultValue); err != nil {
			return fmt.Errorf("inspect driver_task_stops.%s: %w", column, err)
		}
		if kind != expected || !length.Valid || length.Int64 != int64(width) || nullable != "NO" || defaultValue.Valid {
			return fmt.Errorf("incompatible driver_task_stops.%s; automatic column conversion is disabled", column)
		}
	}
	if m.engine == "postgres" {
		var primary, foreign bool
		err := m.conn.QueryRowContext(ctx, `SELECT
   EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid = to_regclass('driver_task_stops') AND c.contype = 'p'
    AND (SELECT string_agg(a.attname, ',' ORDER BY k.ordinality) FROM unnest(c.conkey) WITH ORDINALITY k(attnum, ordinality) JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum) = 'StopKey'),
   EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid = to_regclass('driver_task_stops') AND c.contype = 'f'
    AND c.conname = 'driver_task_stops_RunUUID_run_details_UUID_foreign' AND c.confrelid = to_regclass('run_details')
    AND c.confdeltype = 'c' AND c.confupdtype = 'c' AND c.convalidated
    AND (SELECT string_agg(a.attname, ',' ORDER BY k.ordinality) FROM unnest(c.conkey) WITH ORDINALITY k(attnum, ordinality) JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum) = 'RunUUID'
    AND (SELECT string_agg(a.attname, ',' ORDER BY k.ordinality) FROM unnest(c.confkey) WITH ORDINALITY k(attnum, ordinality) JOIN pg_attribute a ON a.attrelid = c.confrelid AND a.attnum = k.attnum) = 'UUID')`).Scan(&primary, &foreign)
		if err != nil {
			return err
		}
		if !primary || !foreign {
			return errors.New("incompatible driver_task_stops primary or foreign key; automatic constraint conversion is disabled")
		}
		return nil
	}
	var primary string
	if err := m.conn.QueryRowContext(ctx, "SELECT COALESCE(GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX), '') FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'driver_task_stops' AND index_name = 'PRIMARY'").Scan(&primary); err != nil {
		return err
	}
	var foreign int
	err := m.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.referential_constraints r
  JOIN information_schema.key_column_usage k ON k.constraint_schema = r.constraint_schema AND k.table_name = r.table_name AND k.constraint_name = r.constraint_name
  WHERE r.constraint_schema = DATABASE() AND r.table_name = 'driver_task_stops'
  AND r.constraint_name = 'driver_task_stops_RunUUID_run_details_UUID_foreign'
  AND r.delete_rule = 'CASCADE' AND r.update_rule = 'CASCADE'
  AND k.column_name = 'RunUUID' AND k.referenced_table_name = 'run_details' AND k.referenced_column_name = 'UUID'`).Scan(&foreign)
	if err != nil {
		return err
	}
	if primary != "StopKey" || foreign != 1 {
		return errors.New("incompatible driver_task_stops primary or foreign key; automatic constraint conversion is disabled")
	}
	return nil
}

func (m *driverRetryMigration) execDDL(ctx context.Context, statement string) (sql.Result, error) {
	// Statements contain only fixed schema identifiers and definitions, never
	// user data or connection credentials. Keep the enforced algorithm visible.
	glog.Infof("driver_retry_schema ddl=%s", statement)
	return m.conn.ExecContext(ctx, statement)
}
