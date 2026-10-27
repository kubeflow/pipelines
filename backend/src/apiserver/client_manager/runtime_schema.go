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
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
)

const (
	runtimeSchemaLockWait = 5 * time.Second
	// Connection cleanup has a separate 5s deadline after the work budget.
	runtimeSchemaTimeout = 30 * time.Second
)

type runtimeRunColumn struct {
	name string
	kind string
}

var runtimeRunColumns = []runtimeRunColumn{
	{name: "ExecutorType", kind: "varchar(191)"},
	{name: "WorkspaceProvisioner", kind: "varchar(191)"},
	{name: "WorkspaceReference", kind: "json"},
}

// autoMigrate is shared by fresh installs, current-version upgrades, and the
// legacy upgrade flow after its existing preflight/cleanup steps. Runtime DDL
// is explicit; the pre-existing core-schema reconciliation remains separate.
func autoMigrate(db *gorm.DB) error {
	glog.Infof("Running AutoMigrate.")
	return withRuntimeSchemaConnection(db, func(locked *gorm.DB) error {
		if err := addRuntimeRunColumns(locked); err != nil {
			return err
		}
		tasksExist, err := runtimeSchemaHasTable(locked, "tasks")
		if err != nil {
			return err
		}
		if tasksExist {
			if err := requireRuntimeIndex(locked, &model.Task{}, "idx_tasks_uuid_run", "UUID,RunUUID", true); err != nil {
				return fmt.Errorf("runtime task/run foreign key needs the existing candidate index; apply a separately verified online index migration before startup: %w", err)
			}
		}
		models := make([]any, 0, len(model.AllModels())-1)
		for _, candidate := range model.AllModels() {
			switch candidate.(type) {
			case *model.RuntimeTask, *model.RunLease:
				continue
			default:
				models = append(models, candidate)
			}
		}
		if err := locked.AutoMigrate(models...); err != nil {
			return fmt.Errorf("AutoMigrate failed: %w", err)
		}
		if err := createRuntimeTaskTable(locked); err != nil {
			return err
		}
		return createRunLeaseSchema(locked)
	})
}

// withRuntimeSchemaConnection keeps ownership, lock timeouts, and DDL on one
// physical connection. Cleanup failure discards it instead of leaking session
// locks or altered timeout settings to application requests.
func withRuntimeSchemaConnection(db *gorm.DB, migrate func(*gorm.DB) error) (result error) {
	ctx, cancel := context.WithTimeout(db.Statement.Context, runtimeSchemaTimeout)
	defer cancel()
	pool, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire runtime schema connection: %w", err)
	}
	defer conn.Close()
	var cleanups []func(context.Context) error
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), runtimeSchemaLockWait)
		defer cleanupCancel()
		var cleanupErr error
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanupErr = errors.Join(cleanupErr, cleanups[i](cleanupCtx))
		}
		if cleanupErr != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			result = errors.Join(result, fmt.Errorf("restore runtime migration connection (connection discarded): %w", cleanupErr))
		}
	}()

	switch db.Name() {
	case "mysql":
		var previous int64
		if err := conn.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&previous); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = 5"); err != nil {
			return err
		}
		cleanups = append(cleanups, func(ctx context.Context) error {
			_, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION lock_wait_timeout = %d", previous))
			return err
		})
		var database string
		if err := conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
			return err
		}
		lockName := "kfp-runtime-schema:" + util.NewDeterministicUUID(database)
		var owned sql.NullInt64
		if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 5)", lockName).Scan(&owned); err != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			return err
		}
		if !owned.Valid || owned.Int64 != 1 {
			return errors.New("another API server owns the runtime schema migration; retry startup after it completes")
		}
		cleanups = append(cleanups, func(ctx context.Context) error {
			var released sql.NullInt64
			if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", lockName).Scan(&released); err != nil {
				return err
			}
			if !released.Valid || released.Int64 != 1 {
				return errors.New("runtime schema ownership was lost")
			}
			return nil
		})
	case "postgres":
		var lockTimeout, statementTimeout string
		if err := conn.QueryRowContext(ctx, "SELECT current_setting('lock_timeout'), current_setting('statement_timeout')").
			Scan(&lockTimeout, &statementTimeout); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "SELECT set_config('lock_timeout', '5s', false), set_config('statement_timeout', '30s', false)"); err != nil {
			return err
		}
		cleanups = append(cleanups, func(ctx context.Context) error {
			_, err := conn.ExecContext(ctx, "SELECT set_config('lock_timeout', $1, false), set_config('statement_timeout', $2, false)", lockTimeout, statementTimeout)
			return err
		})
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(1180261217, 1919775353)"); err != nil {
			// Cancellation can race successful acquisition. Closing the session
			// also releases a lock whose successful response was never received.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			return fmt.Errorf("acquire runtime schema ownership within 5s; retry startup after the other API server completes: %w", err)
		}
		cleanups = append(cleanups, func(ctx context.Context) error {
			var released bool
			if err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock(1180261217, 1919775353)").Scan(&released); err != nil {
				return err
			}
			if !released {
				return errors.New("runtime schema ownership was lost")
			}
			return nil
		})
	case "sqlite":
		var previous int64
		if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&previous); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
			return err
		}
		cleanups = append(cleanups, func(ctx context.Context) error {
			_, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", previous))
			return err
		})
	default:
		return fmt.Errorf("runtime schema migration does not support dialect %q", db.Name())
	}
	locked := db.Session(&gorm.Session{NewDB: true}).WithContext(ctx)
	locked.Statement.ConnPool = conn
	if db.Name() == "sqlite" {
		// SQLite's database write lock serializes DDL. A competing upgrader
		// fails explicitly on contention; no table-copy fallback is attempted.
		return locked.Transaction(migrate)
	}
	return migrate(locked)
}

// addRuntimeRunColumns adds only nullable, default-free metadata. MySQL must
// accept INSTANT; rejecting it aborts startup without retrying another algorithm.
// PostgreSQL performs metadata-only additions with bounded metadata-lock waits.
func addRuntimeRunColumns(db *gorm.DB) error {
	exists, err := runtimeSchemaHasTable(db, "run_details")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := db.Migrator().ColumnTypes(&model.Run{})
	if err != nil {
		return fmt.Errorf("inventory runtime run columns: %w", err)
	}
	existing := make(map[string]gorm.ColumnType, len(columns))
	for _, column := range columns {
		existing[column.Name()] = column
	}
	var additions []string
	for _, spec := range runtimeRunColumns {
		if column, found := existing[spec.name]; found {
			if err := validateRuntimeRunColumn(column, spec); err != nil {
				return err
			}
			continue
		}
		additions = append(additions, "ADD COLUMN "+runtimeSchemaQuote(db, spec.name)+" "+spec.kind+" NULL")
	}
	if len(additions) == 0 {
		return nil
	}
	glog.Infof("runtime_schema adding %d nullable run columns on %s (metadata-only DDL, 5s lock wait, 30s startup migration budget)", len(additions), db.Name())
	if db.Name() == "sqlite" {
		for _, addition := range additions {
			if err := db.Exec("ALTER TABLE " + runtimeSchemaQuote(db, "run_details") + " " + addition).Error; err != nil {
				return fmt.Errorf("add nullable runtime run column without table rebuild: %w", err)
			}
		}
		return nil
	}
	query := "ALTER TABLE " + runtimeSchemaQuote(db, "run_details") + " " + strings.Join(additions, ", ")
	if db.Name() == "mysql" {
		query += ", ALGORITHM=INSTANT"
	}
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("add nullable runtime run columns using metadata-only DDL (no fallback; resolve lock contention or unsupported INSTANT DDL before restarting): %w", err)
	}
	return nil
}

func validateRuntimeRunColumn(column gorm.ColumnType, spec runtimeRunColumn) error {
	nullable, known := column.Nullable()
	if !known || !nullable {
		return fmt.Errorf("run_details.%s must be nullable; refusing implicit schema repair", spec.name)
	}
	kind := strings.ToLower(column.DatabaseTypeName())
	if spec.kind == "json" {
		if kind != "json" {
			return fmt.Errorf("run_details.%s must use JSON, found %s; refusing implicit type conversion", spec.name, kind)
		}
	} else {
		length, known := column.Length()
		if kind != "varchar" || !known || length != 191 {
			return fmt.Errorf("run_details.%s must use VARCHAR(191); refusing implicit type conversion", spec.name)
		}
	}
	if value, hasDefault := column.DefaultValue(); hasDefault && value != "NULL" && value != "null" {
		return fmt.Errorf("run_details.%s must not have a non-null default; refusing implicit schema repair", spec.name)
	}
	return nil
}

func runtimeSchemaQuote(db *gorm.DB, identifier string) string {
	var quoted strings.Builder
	db.QuoteTo(&quoted, identifier)
	return quoted.String()
}

func runtimeSchemaHasTable(db *gorm.DB, table string) (bool, error) {
	var count int64
	var query string
	switch db.Name() {
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	case "postgres":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?"
	case "sqlite":
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	default:
		return false, fmt.Errorf("unsupported runtime schema dialect %q", db.Name())
	}
	if err := db.Raw(query, table).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("inventory runtime schema table %q: %w", table, err)
	}
	return count == 1, nil
}

// createRuntimeTaskTable creates only a new empty table. Existing runtime
// schemas are checked, never reconciled with potentially copying ALTER TABLE.
func createRuntimeTaskTable(db *gorm.DB) error {
	exists, err := runtimeSchemaHasTable(db, "runtime_tasks")
	if err != nil {
		return err
	}
	if !exists {
		glog.Info("runtime_schema creating empty runtime_tasks table with cascading task/run foreign keys")
		create := func(connection *gorm.DB) error {
			if err := connection.Migrator().CreateTable(&model.RuntimeTask{}); err != nil {
				return err
			}
			// CREATE INDEX IF NOT EXISTS can silently find an index belonging
			// to another table. Validate before committing the new schema.
			return validateRuntimeTaskSchema(connection)
		}
		var err error
		if db.Name() == "postgres" {
			// PostgreSQL emits secondary indexes separately. Keep only this
			// empty-table creation atomic so interruption cannot strand a
			// partial schema; do not extend locks over core reconciliation.
			err = db.Transaction(create)
		} else {
			err = create(db)
		}
		if err != nil {
			return fmt.Errorf("create empty runtime_tasks table with bounded metadata-lock waits: %w", err)
		}
		return nil
	}
	return validateRuntimeTaskSchema(db)
}

func validateRuntimeTaskSchema(db *gorm.DB) error {
	if err := validateRuntimeTaskColumns(db); err != nil {
		return err
	}
	for _, spec := range []struct {
		name   string
		column string
		unique bool
	}{
		{"idx_runtime_tasks_attempt", "AttemptUUID", true},
		{"idx_runtime_tasks_run", "RunUUID", false},
	} {
		if err := requireRuntimeIndex(db, &model.RuntimeTask{}, spec.name, spec.column, spec.unique); err != nil {
			return err
		}
	}
	for _, constraint := range []string{"fk_runtime_tasks_task_run", "fk_runtime_tasks_run"} {
		if !db.Migrator().HasConstraint(&model.RuntimeTask{}, constraint) {
			return fmt.Errorf("runtime_tasks constraint %s is missing; refusing implicit schema repair", constraint)
		}
	}
	return validateRuntimeForeignKeys(db)
}

func validateRuntimeTaskColumns(db *gorm.DB) error {
	columns, err := db.Migrator().ColumnTypes(&model.RuntimeTask{})
	if err != nil {
		return fmt.Errorf("inventory runtime task columns: %w", err)
	}
	existing := make(map[string]gorm.ColumnType, len(columns))
	for _, column := range columns {
		existing[column.Name()] = column
	}
	for _, spec := range []struct {
		name     string
		kind     string
		length   int64
		nullable bool
	}{
		{"TaskUUID", "varchar", 191, false}, {"RunUUID", "varchar", 191, false}, {"AttemptUUID", "varchar", 191, true},
		{"State", "varchar", 32, false}, {"RetryCount", "integer", 0, false},
		{"NextEligibleAt", "integer", 0, true}, {"ExecutionRequest", "json", 0, true}, {"ExecutionReference", "json", 0, true},
	} {
		column, found := existing[spec.name]
		if !found {
			return fmt.Errorf("runtime_tasks.%s is missing; refusing implicit schema repair", spec.name)
		}
		nullable, known := column.Nullable()
		kind := strings.ToLower(column.DatabaseTypeName())
		validKind := kind == spec.kind || (spec.kind == "integer" && (kind == "bigint" || kind == "int8"))
		if !known || nullable != spec.nullable || !validKind {
			return fmt.Errorf("runtime_tasks.%s has incompatible type or nullability; refusing implicit schema repair", spec.name)
		}
		if spec.length != 0 {
			length, known := column.Length()
			if !known || length != spec.length {
				return fmt.Errorf("runtime_tasks.%s has incompatible width; refusing implicit schema repair", spec.name)
			}
		}
		primary, known := column.PrimaryKey()
		if !known || primary != (spec.name == "TaskUUID") {
			return errors.New("runtime_tasks must have TaskUUID as its sole primary key; refusing implicit schema repair")
		}
	}
	return nil
}

type runtimeForeignKey struct {
	Columns          string
	ReferenceTable   string
	ReferenceColumns string
	DeleteRule       string
	UpdateRule       string
}

func validateRuntimeForeignKeys(db *gorm.DB) error {
	return validateRuntimeTableForeignKeys(db, "runtime_tasks", []runtimeForeignKey{
		{Columns: "TaskUUID,RunUUID", ReferenceTable: "tasks", ReferenceColumns: "UUID,RunUUID", DeleteRule: "CASCADE", UpdateRule: "CASCADE"},
		{Columns: "RunUUID", ReferenceTable: "run_details", ReferenceColumns: "UUID", DeleteRule: "CASCADE", UpdateRule: "CASCADE"},
	})
}

func validateRuntimeTableForeignKeys(db *gorm.DB, table string, expectedKeys []runtimeForeignKey) error {
	var keys []runtimeForeignKey
	switch db.Name() {
	case "postgres":
		if err := db.Raw(`SELECT
			(SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
			 FROM unnest(c.conkey) WITH ORDINALITY k(attnum, ordinality)
			 JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum) AS "Columns",
			rt.relname AS "ReferenceTable",
			(SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
			 FROM unnest(c.confkey) WITH ORDINALITY k(attnum, ordinality)
			 JOIN pg_attribute a ON a.attrelid = c.confrelid AND a.attnum = k.attnum) AS "ReferenceColumns",
			CASE WHEN c.confdeltype = 'c' THEN 'CASCADE' ELSE 'OTHER' END AS "DeleteRule",
			CASE WHEN c.confupdtype = 'c' THEN 'CASCADE' ELSE 'OTHER' END AS "UpdateRule"
			FROM pg_constraint c JOIN pg_class t ON t.oid = c.conrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			JOIN pg_class rt ON rt.oid = c.confrelid
			JOIN pg_namespace rn ON rn.oid = rt.relnamespace
			WHERE c.contype = 'f' AND c.convalidated AND n.nspname = current_schema()
			 AND rn.nspname = current_schema() AND t.relname = ?`, table).Scan(&keys).Error; err != nil {
			return err
		}
	case "mysql":
		if err := db.Raw(`SELECT
			GROUP_CONCAT(k.COLUMN_NAME ORDER BY k.ORDINAL_POSITION SEPARATOR ',') AS `+"`Columns`"+`,
			k.REFERENCED_TABLE_NAME AS ReferenceTable,
			GROUP_CONCAT(k.REFERENCED_COLUMN_NAME ORDER BY k.ORDINAL_POSITION SEPARATOR ',') AS ReferenceColumns,
			r.DELETE_RULE AS DeleteRule, r.UPDATE_RULE AS UpdateRule
			FROM information_schema.KEY_COLUMN_USAGE k
			JOIN information_schema.REFERENTIAL_CONSTRAINTS r ON r.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA
			 AND r.CONSTRAINT_NAME = k.CONSTRAINT_NAME AND r.TABLE_NAME = k.TABLE_NAME
			WHERE k.TABLE_SCHEMA = DATABASE() AND k.REFERENCED_TABLE_SCHEMA = DATABASE() AND k.TABLE_NAME = ?
			GROUP BY k.CONSTRAINT_NAME, k.REFERENCED_TABLE_NAME, r.DELETE_RULE, r.UPDATE_RULE`, table).Scan(&keys).Error; err != nil {
			return err
		}
	case "sqlite":
		var rows []struct {
			ID       int
			Seq      int
			Table    string
			From     string
			To       string
			OnUpdate string
			OnDelete string
		}
		if err := db.Raw("PRAGMA foreign_key_list(" + runtimeSchemaQuote(db, table) + ")").Scan(&rows).Error; err != nil {
			return err
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].ID < rows[j].ID || (rows[i].ID == rows[j].ID && rows[i].Seq < rows[j].Seq)
		})
		for i, row := range rows {
			if i == 0 || row.ID != rows[i-1].ID {
				keys = append(keys, runtimeForeignKey{ReferenceTable: row.Table, DeleteRule: row.OnDelete, UpdateRule: row.OnUpdate})
			}
			key := &keys[len(keys)-1]
			if key.Columns != "" {
				key.Columns += ","
				key.ReferenceColumns += ","
			}
			key.Columns += row.From
			key.ReferenceColumns += row.To
		}
	}
	for _, expected := range expectedKeys {
		found := false
		for _, actual := range keys {
			found = found || actual == expected
		}
		if !found {
			return fmt.Errorf("%s foreign key (%s) to %s must cascade; refusing implicit schema repair", table, expected.Columns, expected.ReferenceTable)
		}
	}
	return nil
}

func requireRuntimeIndex(db *gorm.DB, table any, name, columns string, unique bool) error {
	if db.Name() == "postgres" {
		// GORM's PostgreSQL GetIndexes excludes indexes referenced by foreign
		// keys and does not preserve key order. Query the actual catalog.
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(table); err != nil {
			return err
		}
		var index struct {
			Unique  bool
			Ready   bool
			Columns string
		}
		result := db.Raw(`SELECT i.indisunique AS "Unique",
			(i.indisvalid AND i.indisready AND i.indpred IS NULL AND i.indexprs IS NULL) AS "Ready",
			(SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
			 FROM unnest(i.indkey) WITH ORDINALITY AS k(attnum, ordinality)
			 JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum
			 WHERE k.ordinality <= i.indnkeyatts) AS "Columns"
			FROM pg_index i JOIN pg_class t ON t.oid = i.indrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			JOIN pg_class ix ON ix.oid = i.indexrelid
			WHERE n.nspname = current_schema() AND t.relname = ? AND ix.relname = ?`,
			statement.Table, name).Scan(&index)
		if result.Error != nil {
			return fmt.Errorf("inspect PostgreSQL runtime index: %w", result.Error)
		}
		if result.RowsAffected != 1 || !index.Ready || index.Unique != unique || index.Columns != columns {
			return fmt.Errorf("index %s is missing or incompatible; refusing implicit schema repair", name)
		}
		return nil
	}
	indexes, err := db.Migrator().GetIndexes(table)
	if err != nil {
		return fmt.Errorf("inspect runtime schema indexes: %w", err)
	}
	var actual string
	for _, index := range indexes {
		foundUnique, known := index.Unique()
		actual += fmt.Sprintf("%s:columns=%v unique=%v known=%v; ", index.Name(), index.Columns(), foundUnique, known)
		if index.Name() == name && known && foundUnique == unique && strings.Join(index.Columns(), ",") == columns {
			return nil
		}
	}
	return fmt.Errorf("index %s is missing or incompatible (%s); refusing implicit schema repair", name, actual)
}
