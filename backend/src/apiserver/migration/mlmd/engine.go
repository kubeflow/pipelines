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

package mlmd

import (
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed schema.sql
var schemaSQL string

// Report contains the completed graph counts and auditable source omissions.
type Report struct {
	Tasks              int                       `json:"tasks"`
	Artifacts          int                       `json:"artifacts"`
	Links              int                       `json:"links"`
	CacheKeysVerified  int                       `json:"cache_keys_verified"`
	CacheKeysWritten   int                       `json:"cache_keys_written"`
	CacheKeysPreserved int                       `json:"cache_keys_preserved"`
	MigratedByKind     map[string]int            `json:"migrated_by_kind,omitempty"`
	SkippedByKind      map[string]int            `json:"skipped_by_kind,omitempty"`
	SkippedByReason    map[string]map[string]int `json:"skipped_by_reason,omitempty"`
	Skipped            []Skip                    `json:"skipped"`
	Warnings           []string                  `json:"warnings"`
	DiskCapacity       *CapacityReport           `json:"disk_capacity,omitempty"`
}

func schemaStatements() []string {
	parts := strings.Split(schemaSQL, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		if statement := strings.TrimSpace(part); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}

func tableExists(ctx context.Context, db *gorm.DB, name string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", name).Scan(&count).Error
	return count != 0, err
}

func columnExists(ctx context.Context, db *gorm.DB, table, column string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?", table, column).Scan(&count).Error
	return count != 0, err
}

func qualifyTaskSchema(ctx context.Context, db *gorm.DB, table string, legacy bool) error {
	if legacy {
		const query = `SELECT COLUMN_NAME, DATA_TYPE FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`
		return qualifyCatalogEntries(ctx, db, table, "column", query, map[string]string{
			"UUID": "varchar", "RunUUID": "varchar", "Namespace": "varchar",
			"MLMDExecutionID": "varchar", "PodName": "varchar", "State": "varchar",
			"CreatedTimestamp": "bigint", "StartedTimestamp": "bigint", "FinishedTimestamp": "bigint",
		})
	}
	if err := qualifyColumns(ctx, db, table, map[string]string{
		"UUID": "varchar(191) NOT NULL", "Namespace": "varchar(63) NOT NULL", "RunUUID": "varchar(191) NOT NULL",
		"pods": "json NOT NULL", "CreatedAtInSec": "bigint NOT NULL", "StartedInSec": "bigint", "FinishedInSec": "bigint",
		"Fingerprint": "varchar(255) NOT NULL", "CachedTaskUUID": "varchar(191)", "Name": "varchar(128)",
		"DisplayName": "varchar(128)", "ParentTaskUUID": "varchar(191)", "State": "int NOT NULL",
		"StatusMetadata": "json", "StateHistory": "json", "InputParameters": "json", "OutputParameters": "json",
		"Type": "int NOT NULL", "TypeAttrs": "json NOT NULL", "ScopePath": "text", "LogicalKey": "varchar(64)",
		"LifecycleMessage": "longtext", "LifecycleCategory": "varchar(32)",
	}); err != nil {
		return err
	}
	if err := qualifyIndexes(ctx, db, table, map[string]string{
		"PRIMARY": "0:UUID:A", "idx_tasks_uuid_run": "0:UUID:A,RunUUID:A",
		"idx_tasks_logical_key": "0:LogicalKey:A",
		"idx_task_cache_lookup": "1:Fingerprint:A,State:A,Namespace:A,CreatedAtInSec:D",
	}); err != nil {
		return err
	}
	return qualifyForeignKeys(ctx, db, table, map[string]string{
		"fk_run_details_tasks": "RunUUID:run_details.UUID/RESTRICT/RESTRICT",
		"fk_tasks_parent_task": "ParentTaskUUID:" + table + ".UUID/CASCADE/CASCADE",
	})
}

func qualifyColumns(ctx context.Context, db *gorm.DB, table string, expected map[string]string) error {
	const query = `SELECT COLUMN_NAME, CONCAT(COLUMN_TYPE, IF(IS_NULLABLE='NO', ' NOT NULL', ''))
		FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`
	return qualifyCatalogEntries(ctx, db, table, "column", query, expected)
}

func qualifyIndexes(ctx context.Context, db *gorm.DB, table string, expected map[string]string) error {
	const query = `SELECT INDEX_NAME, CONCAT(NON_UNIQUE, ':',
		GROUP_CONCAT(CONCAT(COALESCE(COLUMN_NAME, '<expression>'), ':', COALESCE(COLLATION, ''),
			IF(SUB_PART IS NULL, '', CONCAT(':', SUB_PART)))
			ORDER BY SEQ_IN_INDEX SEPARATOR ','))
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND IS_VISIBLE='YES'
		GROUP BY INDEX_NAME, NON_UNIQUE`
	return qualifyCatalogEntries(ctx, db, table, "index", query, expected)
}

func qualifyForeignKeys(ctx context.Context, db *gorm.DB, table string, expected map[string]string) error {
	const query = `SELECT k.CONSTRAINT_NAME,
		CONCAT(GROUP_CONCAT(CONCAT(k.COLUMN_NAME, ':', k.REFERENCED_TABLE_NAME, '.', k.REFERENCED_COLUMN_NAME)
			ORDER BY k.ORDINAL_POSITION SEPARATOR ','), '/',
			IF(r.DELETE_RULE='NO ACTION', 'RESTRICT', r.DELETE_RULE), '/',
			IF(r.UPDATE_RULE='NO ACTION', 'RESTRICT', r.UPDATE_RULE))
		FROM information_schema.KEY_COLUMN_USAGE k
		JOIN information_schema.REFERENTIAL_CONSTRAINTS r ON k.CONSTRAINT_SCHEMA=r.CONSTRAINT_SCHEMA
			AND k.TABLE_NAME=r.TABLE_NAME AND k.CONSTRAINT_NAME=r.CONSTRAINT_NAME
		WHERE k.TABLE_SCHEMA=DATABASE() AND k.TABLE_NAME=? AND k.REFERENCED_TABLE_SCHEMA=DATABASE()
		GROUP BY k.CONSTRAINT_NAME, r.DELETE_RULE, r.UPDATE_RULE`
	return qualifyCatalogEntries(ctx, db, table, "foreign key", query, expected)
}

func qualifyCatalogEntries(ctx context.Context, db *gorm.DB, table, kind, query string, expected map[string]string) error {
	rows, err := db.WithContext(ctx).Raw(query, table).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	actual := map[string]string{}
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			return err
		}
		actual[name] = definition
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for name, definition := range expected {
		if actual[name] != definition {
			return fmt.Errorf("%s %s %s has definition %q, expected %q; refusing unqualified schema", table, kind, name, actual[name], definition)
		}
	}
	return nil
}

func qualifyArtifactTable(ctx context.Context, db *gorm.DB, table string) error {
	if table == "artifacts" {
		if err := qualifyColumns(ctx, db, table, map[string]string{
			"UUID": "varchar(191) NOT NULL", "Namespace": "varchar(63) NOT NULL", "Type": "int",
			"URI": "varchar(2048)", "URIHash": "varchar(64)", "Name": "varchar(128)", "Description": "text",
			"CreatedAtInSec": "bigint NOT NULL", "LastUpdateInSec": "bigint NOT NULL",
			"Metadata": "json", "NumberValue": "double", "IdentityKey": "varchar(64)",
		}); err != nil {
			return err
		}
		return qualifyIndexes(ctx, db, table, map[string]string{
			"PRIMARY": "0:UUID:A", "idx_artifact_identity": "0:Namespace:A,IdentityKey:A",
		})
	}
	if err := qualifyColumns(ctx, db, table, map[string]string{
		"UUID": "varchar(191) NOT NULL", "ArtifactID": "varchar(191) NOT NULL", "TaskID": "varchar(191) NOT NULL",
		"RunUUID": "varchar(191) NOT NULL", "Type": "int NOT NULL", "Iteration": "bigint NOT NULL",
		"Producer": "json", "ArtifactKey": "varchar(191) NOT NULL",
	}); err != nil {
		return err
	}
	if err := qualifyIndexes(ctx, db, table, map[string]string{
		"PRIMARY": "0:UUID:A", "UniqueLink": "0:ArtifactID:A,TaskID:A,Type:A,Iteration:A,ArtifactKey:A",
	}); err != nil {
		return err
	}
	return qualifyForeignKeys(ctx, db, table, map[string]string{
		"fk_artifact_tasks_artifacts": "ArtifactID:artifacts.UUID/CASCADE/CASCADE",
		"fk_artifact_tasks_runs":      "RunUUID:run_details.UUID/CASCADE/CASCADE",
		"fk_artifact_tasks_task_run":  "TaskID:tasks.UUID,RunUUID:tasks.RunUUID/CASCADE/CASCADE",
	})
}

func loadRuns(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	rows, err := db.WithContext(ctx).Raw("SELECT r.UUID, COALESCE(NULLIF(NULLIF(r.Namespace, ''), '-'), NULLIF(e.Namespace, '-'), ''), r.State, r.FinishedAtInSec FROM run_details r LEFT JOIN experiments e ON e.UUID=r.ExperimentUUID").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := map[string]string{}
	for rows.Next() {
		var id, namespace string
		var state sql.NullString
		var finished sql.NullInt64
		if err := rows.Scan(&id, &namespace, &state, &finished); err != nil {
			return nil, err
		}
		switch strings.ToUpper(state.String) {
		case "SUCCEEDED", "FAILED", "SKIPPED", "CANCELED", "ERROR", "DONE", "DISABLED":
			if !finished.Valid || finished.Int64 <= 0 {
				return nil, fmt.Errorf("KFP run %s has no persisted finish time; wait for its final report before migration", id)
			}
		default:
			return nil, fmt.Errorf("KFP run %s is not terminal (%q); drain runs before migration", id, state.String)
		}
		runs[id] = namespace
	}
	return runs, rows.Err()
}

func loadLegacyTasks(ctx context.Context, db *gorm.DB, table string) (map[int64]LegacyTask, []Skip, error) {
	query := "SELECT UUID, MLMDExecutionID, RunUUID, Namespace, PodName, State, StateHistory, CreatedTimestamp, StartedTimestamp, FinishedTimestamp, Fingerprint, Name, ParentTaskUUID, PipelineName, MLMDInputs, MLMDOutputs, ChildrenPods, Payload FROM `" + table + "` WHERE MLMDExecutionID <> ''"
	rows, err := db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	legacy := map[int64]LegacyTask{}
	ambiguous := map[int64]bool{}
	var skipped []Skip
	for rows.Next() {
		var row LegacyTask
		var idText string
		var state, history, name, parent, inputs, outputs, children, payload sql.NullString
		if err := rows.Scan(&row.UUID, &idText, &row.RunUUID, &row.Namespace, &row.PodName,
			&state, &history, &row.CreatedAtInSec, &row.StartedInSec, &row.FinishedInSec,
			&row.Fingerprint, &name, &parent, &row.PipelineName, &inputs, &outputs, &children, &payload); err != nil {
			return nil, nil, err
		}
		row.State, row.StateHistory, row.Name, row.ParentTaskUUID = state.String, history.String, name.String, parent.String
		row.MLMDInputs, row.MLMDOutputs, row.ChildrenPods, row.Payload = inputs.String, outputs.String, children.String, payload.String
		id, err := strconv.ParseInt(idText, 10, 64)
		if err != nil || id <= 0 {
			skipped = append(skipped, Skip{Kind: "legacy_cache_row", Source: row.UUID, Reason: "invalid MLMDExecutionID"})
			continue
		}
		if previous, duplicate := legacy[id]; duplicate {
			skipped = append(skipped, Skip{Kind: "legacy_cache_row", ID: id, Source: previous.UUID, Reason: "ambiguous MLMD execution join"})
			delete(legacy, id)
			ambiguous[id] = true
		}
		if ambiguous[id] {
			skipped = append(skipped, Skip{Kind: "legacy_cache_row", ID: id, Source: row.UUID, Reason: "ambiguous MLMD execution join"})
			continue
		}
		row.ExecutionID = id
		legacy[id] = row
	}
	return legacy, skipped, rows.Err()
}

func createTaskCopy(ctx context.Context, db *gorm.DB) error {
	statements := schemaStatements()
	if len(statements) != 3 {
		return fmt.Errorf("migration schema has %d statements, want three", len(statements))
	}
	taskSQL := strings.ReplaceAll(statements[0], "`tasks`", "`tasks_migration_tmp`")
	return db.WithContext(ctx).Exec(taskSQL).Error
}

func rejectInsertWarnings(tx *gorm.DB) error {
	rows, err := tx.Raw("SHOW WARNINGS").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var level, message string
		var code int
		if err := rows.Scan(&level, &code, &message); err != nil {
			return err
		}
		if code != 1062 {
			return fmt.Errorf("MySQL INSERT IGNORE produced warning code %d (%s); source value was not preserved", code, level)
		}
	}
	return rows.Err()
}

func verifyParents(ctx context.Context, db *gorm.DB, tasks []model.Task) error {
	expected := map[string]string{}
	for _, task := range tasks {
		expected[task.UUID] = ""
		if task.ParentTaskUUID != nil {
			expected[task.UUID] = *task.ParentTaskUUID
		}
	}
	ids := make([]string, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	for start := 0; start < len(ids); start += 100 {
		end := start + 100
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		args := make([]any, len(batch))
		for index, id := range batch {
			args[index] = id
		}
		query := "SELECT UUID,ParentTaskUUID FROM tasks WHERE UUID IN (" + strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",") + ")"
		rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			var parent sql.NullString
			if err := rows.Scan(&id, &parent); err != nil {
				rows.Close()
				return err
			}
			if parent.String != expected[id] {
				rows.Close()
				return fmt.Errorf("task %s has unresolved parent", id)
			}
			delete(expected, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	for id := range expected {
		return fmt.Errorf("task %s has unresolved parent", id)
	}
	return nil
}

func transientMigrationError(err error) bool {
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}
	var networkError *net.OpError
	if errors.As(err, &networkError) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	}
	var mysqlError *mysqlDriver.MySQLError
	if errors.As(err, &mysqlError) {
		switch mysqlError.Number {
		case 1040, 1213, 2006, 2013:
			return true
		}
	}
	return false
}

func releaseMigrationLock(connection *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var released sql.NullInt64
	err := connection.QueryRowContext(ctx, "SELECT RELEASE_LOCK('kfp_mlmd_migration_lock')").Scan(&released)
	if err != nil || !released.Valid || released.Int64 != 1 {
		_ = connection.Raw(func(any) error { return driver.ErrBadConn })
	}
}

// Migrate copies a stable MLMD snapshot to the native MySQL schema. The caller
// is responsible for quiescing writers and backing up the KFP database first.
func Migrate(ctx context.Context, source pb.MetadataStoreServiceClient, database *sql.DB, budget DiskBudget) (Report, error) {
	if _, ok := database.Driver().(*mysqlDriver.MySQLDriver); !ok {
		return Report{}, fmt.Errorf("legacy MLMD migration requires MySQL; PostgreSQL upgrades are unsupported")
	}
	var report Report
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		report, err = migrateOnce(ctx, source, database, budget)
		var mysqlError *mysqlDriver.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1205 {
			return report, fmt.Errorf("MySQL lock wait expired during MLMD migration; release the blocking database transaction before retrying: %w", err)
		}
		if err == nil || !transientMigrationError(err) {
			return report, err
		}
		if attempt == 2 {
			break
		}
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case <-time.After(time.Duration(200<<attempt) * time.Millisecond):
		}
	}
	return report, fmt.Errorf("migration exhausted connection retries: %w", err)
}

func migrateOnce(ctx context.Context, source pb.MetadataStoreServiceClient, database *sql.DB, budget DiskBudget) (Report, error) {
	report := Report{}
	connection, err := database.Conn(ctx)
	if err != nil {
		return report, err
	}
	defer connection.Close()
	var locked int
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK('kfp_mlmd_migration_lock', 10)").Scan(&locked); err != nil {
		return report, err
	}
	if locked != 1 {
		return report, fmt.Errorf("MLMD migration already in progress")
	}
	defer releaseMigrationLock(connection)
	var lockWait int
	if err := connection.QueryRowContext(ctx, "SELECT @@SESSION.lock_wait_timeout").Scan(&lockWait); err != nil {
		return report, err
	}
	if lockWait > 10 {
		if _, err := connection.ExecContext(ctx, "SET SESSION lock_wait_timeout=10"); err != nil {
			return report, err
		}
		defer func() {
			restoreCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := connection.ExecContext(restoreCtx, fmt.Sprintf("SET SESSION lock_wait_timeout=%d", lockWait)); err != nil {
				_ = connection.Raw(func(any) error { return driver.ErrBadConn })
			}
		}()
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return report, err
	}
	exists, err := tableExists(ctx, db, "tasks")
	if err != nil {
		return report, err
	}
	if !exists {
		return report, fmt.Errorf("KFP tasks table is missing; this CLI requires a released MySQL installation")
	}
	legacySchema, err := columnExists(ctx, db, "tasks", "MLMDExecutionID")
	if err != nil {
		return report, err
	}
	if err := qualifyTaskSchema(ctx, db, "tasks", legacySchema); err != nil {
		return report, err
	}
	backupExists, err := tableExists(ctx, db, "tasks_pre_migration")
	if err != nil {
		return report, err
	}
	if legacySchema && backupExists {
		return report, fmt.Errorf("both legacy tasks and tasks_pre_migration exist; restore the backup before retrying")
	}
	for _, table := range []string{"artifacts", "artifact_tasks"} {
		present, err := tableExists(ctx, db, table)
		if err != nil {
			return report, err
		}
		switch {
		case legacySchema && present:
			return report, fmt.Errorf("legacy tasks coexist with %s; refusing unqualified partial schema", table)
		case present:
			if err := qualifyArtifactTable(ctx, db, table); err != nil {
				return report, err
			}
		case !legacySchema && !backupExists:
			return report, fmt.Errorf("native tasks has no %s table and no legacy backup", table)
		}
	}
	if !legacySchema && !backupExists {
		temporaryExists, err := tableExists(ctx, db, "tasks_migration_tmp")
		if err != nil {
			return report, err
		}
		if temporaryExists {
			return report, fmt.Errorf("native tasks still has a staging table and no legacy backup")
		}
		err = countNativeRows(ctx, db, &report)
		return report, err
	}
	legacyTable := "tasks"
	if backupExists {
		legacyTable = "tasks_pre_migration"
		if err := qualifyTaskSchema(ctx, db, legacyTable, true); err != nil {
			return report, err
		}
	}
	temporaryExists, err := tableExists(ctx, db, "tasks_migration_tmp")
	if err != nil {
		return report, err
	}
	if temporaryExists {
		if !legacySchema {
			return report, fmt.Errorf("native tasks and the staging table coexist; refusing unqualified partial schema")
		}
		if err := qualifyTaskSchema(ctx, db, "tasks_migration_tmp", false); err != nil {
			return report, err
		}
	}
	capacity, err := checkDiskCapacity(ctx, connection, budget)
	report.DiskCapacity = &capacity
	if err != nil {
		return report, err
	}
	runs, err := loadRuns(ctx, db)
	if err != nil {
		return report, err
	}
	legacy, skippedLegacy, err := loadLegacyTasks(ctx, db, legacyTable)
	if err != nil {
		return report, err
	}
	if source == nil {
		return report, fmt.Errorf("incomplete MLMD migration requires the installation's source metadata store")
	}
	snapshot, err := ReadSnapshot(ctx, source)
	if err != nil {
		return report, err
	}
	plan := BuildPlan(snapshot, runs, legacy)
	plan.Skipped = append(plan.Skipped, skippedLegacy...)
	cacheRuns, err := loadCacheRuns(ctx, db)
	if err != nil {
		return report, err
	}
	projectParameterOrigins(plan.Tasks, cacheRuns)
	cutoff, err := cacheCutoff(ctx, db, legacySchema || backupExists)
	if err != nil {
		return report, err
	}
	report.CacheKeysVerified = backfillNativeCacheKeys(&plan, snapshot, legacy, cacheRuns, cutoff)
	sort.Slice(plan.Skipped, func(i, j int) bool {
		left, right := plan.Skipped[i], plan.Skipped[j]
		switch {
		case left.Kind != right.Kind:
			return left.Kind < right.Kind
		case left.ID != right.ID:
			return left.ID < right.ID
		case left.Source != right.Source:
			return left.Source < right.Source
		default:
			return left.Reason < right.Reason
		}
	})
	sort.Strings(plan.Warnings)
	report.Skipped = plan.Skipped
	report.Warnings = plan.Warnings
	report.SkippedByKind = map[string]int{}
	report.SkippedByReason = map[string]map[string]int{}
	for _, skip := range plan.Skipped {
		report.SkippedByKind[skip.Kind]++
		if report.SkippedByReason[skip.Kind] == nil {
			report.SkippedByReason[skip.Kind] = map[string]int{}
		}
		report.SkippedByReason[skip.Kind][skip.Reason]++
	}
	statements := schemaStatements()
	if legacySchema {
		if !temporaryExists {
			if err := createTaskCopy(ctx, db); err != nil {
				return report, err
			}
		}
		if err := insertTasks(ctx, db, "tasks_migration_tmp", plan.Tasks); err != nil {
			return report, err
		}
		var staged int
		if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM tasks_migration_tmp").Row().Scan(&staged); err != nil {
			return report, err
		}
		if staged != len(plan.Tasks) {
			return report, fmt.Errorf("staged task count %d does not match %d verified source tasks", staged, len(plan.Tasks))
		}
		if err := db.WithContext(ctx).Exec("RENAME TABLE tasks TO tasks_pre_migration, tasks_migration_tmp TO tasks").Error; err != nil {
			return report, err
		}
	} else if err := insertTasks(ctx, db, "tasks", plan.Tasks); err != nil {
		return report, err
	}
	for _, statement := range statements[1:] {
		name := "artifacts"
		if strings.Contains(statement, "CREATE TABLE `artifact_tasks`") {
			name = "artifact_tasks"
		}
		present, err := tableExists(ctx, db, name)
		if err != nil {
			return report, err
		}
		if !present {
			if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
				return report, err
			}
		}
	}
	if err := insertArtifacts(ctx, db, plan.Artifacts, plan.artifactSizes); err != nil {
		return report, err
	}
	if err := insertLinks(ctx, db, plan.Links); err != nil {
		return report, err
	}
	for _, task := range plan.Tasks {
		if task.ParentTaskUUID != nil {
			if err := db.WithContext(ctx).Exec("UPDATE tasks SET ParentTaskUUID=? WHERE UUID=? AND ParentTaskUUID IS NULL", *task.ParentTaskUUID, task.UUID).Error; err != nil {
				return report, err
			}
		}
	}
	report.CacheKeysWritten, err = writeNativeCacheKeys(ctx, db, plan.Tasks)
	if err != nil {
		return report, err
	}
	report.CacheKeysPreserved = report.CacheKeysVerified - report.CacheKeysWritten
	if err := verifyParents(ctx, db, plan.Tasks); err != nil {
		return report, err
	}
	temporaryExists, err = tableExists(ctx, db, "tasks_migration_tmp")
	if err != nil {
		return report, err
	}
	if temporaryExists {
		return report, fmt.Errorf("tasks_migration_tmp remains after task cutover; inspect the partial migration before cleanup")
	}
	if err := countNativeRows(ctx, db, &report); err != nil {
		return report, err
	}
	if report.Tasks != len(plan.Tasks) || report.Artifacts != len(plan.Artifacts) || report.Links != len(plan.Links) {
		return report, fmt.Errorf("migrated table counts do not match the verified source graph")
	}
	if err := db.WithContext(ctx).Exec("DROP TABLE tasks_pre_migration").Error; err != nil {
		return report, err
	}
	report.MigratedByKind = map[string]int{"execution": report.Tasks, "artifact": len(plan.artifactSizes), "event": report.Links}
	return report, nil
}

func countNativeRows(ctx context.Context, db *gorm.DB, report *Report) error {
	for _, count := range []struct {
		table string
		value *int
	}{{"tasks", &report.Tasks}, {"artifacts", &report.Artifacts}, {"artifact_tasks", &report.Links}} {
		if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM `" + count.table + "`").Row().Scan(count.value); err != nil {
			return err
		}
	}
	return nil
}
