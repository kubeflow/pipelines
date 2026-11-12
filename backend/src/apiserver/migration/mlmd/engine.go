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
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	Tasks             int      `json:"tasks"`
	Artifacts         int      `json:"artifacts"`
	Links             int      `json:"links"`
	CacheKeysVerified int      `json:"cache_keys_verified"`
	Skipped           []Skip   `json:"skipped"`
	Warnings          []string `json:"warnings"`
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

func columnDataType(ctx context.Context, db *gorm.DB, table, column string) (string, error) {
	var dataType sql.NullString
	err := db.WithContext(ctx).Raw("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?", table, column).Row().Scan(&dataType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return dataType.String, nil
}

func qualifyTaskSchema(ctx context.Context, db *gorm.DB, table string, legacy bool) error {
	want := map[string]string{"UUID": "varchar", "RunUUID": "varchar", "Namespace": "varchar"}
	if legacy {
		for column, dataType := range map[string]string{"MLMDExecutionID": "varchar", "PodName": "varchar", "State": "varchar", "CreatedTimestamp": "bigint", "StartedTimestamp": "bigint", "FinishedTimestamp": "bigint"} {
			want[column] = dataType
		}
	} else {
		for column, dataType := range map[string]string{"pods": "json", "State": "int", "Type": "int", "TypeAttrs": "json", "ParentTaskUUID": "varchar"} {
			want[column] = dataType
		}
	}
	for column, expected := range want {
		actual, err := columnDataType(ctx, db, table, column)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("%s.%s has type %q, expected %q; refusing unqualified schema", table, column, actual, expected)
		}
	}
	return nil
}

func loadRuns(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	rows, err := db.WithContext(ctx).Raw("SELECT UUID, Namespace, State, FinishedAtInSec FROM run_details").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := map[string]string{}
	for rows.Next() {
		var id, namespace string
		var state sql.NullString
		var finished int64
		if err := rows.Scan(&id, &namespace, &state, &finished); err != nil {
			return nil, err
		}
		switch strings.ToUpper(state.String) {
		case "SUCCEEDED", "FAILED", "SKIPPED", "CANCELED", "ERROR", "DONE", "DISABLED":
			if finished <= 0 {
				return nil, fmt.Errorf("KFP run %s has no persisted finish time; wait for its final report before migration", id)
			}
		default:
			return nil, fmt.Errorf("KFP run %s is not terminal (%q); drain runs before migration", id, state.String)
		}
		runs[id] = namespace
	}
	return runs, rows.Err()
}

func loadLegacyTasks(ctx context.Context, db *gorm.DB, table string) (map[int64]LegacyTask, []LegacyTask, error) {
	query := "SELECT UUID, MLMDExecutionID, RunUUID, Namespace, PodName, State, StateHistory, CreatedTimestamp, StartedTimestamp, FinishedTimestamp, Fingerprint, Name, ParentTaskUUID, PipelineName, MLMDInputs, MLMDOutputs, ChildrenPods, Payload FROM `" + table + "`"
	rows, err := db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	legacy := map[int64]LegacyTask{}
	unjoined := []LegacyTask{}
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
		if idText == "" {
			unjoined = append(unjoined, row)
			continue
		}
		id, err := strconv.ParseInt(idText, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("legacy task has invalid MLMDExecutionID %q: %w", idText, err)
		}
		if _, duplicate := legacy[id]; duplicate {
			return nil, nil, fmt.Errorf("legacy tasks have duplicate MLMDExecutionID %d", id)
		}
		row.ExecutionID = id
		legacy[id] = row
	}
	return legacy, unjoined, rows.Err()
}

func createTaskCopy(ctx context.Context, db *gorm.DB) error {
	statements := schemaStatements()
	if len(statements) != 3 {
		return fmt.Errorf("migration schema has %d statements, want three", len(statements))
	}
	taskSQL := strings.Replace(statements[0], "CREATE TABLE `tasks`", "CREATE TABLE `tasks_migration_tmp`", 1)
	// The self-FK is installed after the table has its final name; while the
	// copy exists, `tasks` still denotes the legacy table.
	start := strings.Index(taskSQL, ",\n  CONSTRAINT `fk_tasks_parent_task`")
	if start < 0 {
		return fmt.Errorf("target task DDL has no parent foreign key")
	}
	end := strings.Index(taskSQL[start:], "\n)")
	if end < 0 {
		return fmt.Errorf("target task DDL is malformed")
	}
	taskSQL = taskSQL[:start] + taskSQL[start+end:]
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

func insertTasks(ctx context.Context, db *gorm.DB, table string, tasks []model.Task) error {
	query := "INSERT IGNORE INTO `" + table + "` (UUID,Namespace,RunUUID,pods,CreatedAtInSec,StartedInSec,FinishedInSec,Fingerprint,Name,DisplayName,State,StateHistory,InputParameters,OutputParameters,Type,TypeAttrs,StatusMetadata) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	for start := 0; start < len(tasks); start += 100 {
		end := start + 100
		if end > len(tasks) {
			end = len(tasks)
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, task := range tasks[start:end] {
				pods := task.Pods
				if pods == nil {
					pods = model.JSONSlice{}
				}
				attrs := task.TypeAttrs
				if attrs == nil {
					attrs = model.JSONData{}
				}
				values := []any{pods, task.StateHistory, task.InputParameters, task.OutputParameters, attrs, task.StatusMetadata}
				encoded := make([]any, len(values))
				for index, value := range values {
					if value != nil {
						data, err := json.Marshal(value)
						if err != nil {
							return err
						}
						encoded[index] = string(data)
					}
				}
				if err := tx.Exec(query, task.UUID, task.Namespace, task.RunUUID, encoded[0], task.CreatedAtInSec, task.StartedInSec, task.FinishedInSec, task.Fingerprint, task.Name, task.DisplayName, int32(task.State), encoded[1], encoded[2], encoded[3], int32(task.Type), encoded[4], encoded[5]).Error; err != nil {
					return err
				}
				if err := rejectInsertWarnings(tx); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func insertArtifacts(ctx context.Context, db *gorm.DB, artifacts []model.Artifact) error {
	const query = "INSERT IGNORE INTO artifacts (UUID,Namespace,Type,URI,URIHash,Name,Description,CreatedAtInSec,LastUpdateInSec,Metadata,NumberValue) VALUES (?,?,?,?,?,?,?,?,?,?,?)"
	for start := 0; start < len(artifacts); start += 100 {
		end := start + 100
		if end > len(artifacts) {
			end = len(artifacts)
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, artifact := range artifacts[start:end] {
				metadata, err := json.Marshal(artifact.Metadata)
				if err != nil {
					return err
				}
				if err := tx.Exec(query, artifact.UUID, artifact.Namespace, int32(artifact.Type), artifact.URI, artifact.URIHash, artifact.Name, artifact.Description, artifact.CreatedAtInSec, artifact.LastUpdateInSec, string(metadata), artifact.NumberValue).Error; err != nil {
					return err
				}
				if err := rejectInsertWarnings(tx); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func insertLinks(ctx context.Context, db *gorm.DB, links []model.ArtifactTask) error {
	const query = "INSERT IGNORE INTO artifact_tasks (UUID,ArtifactID,TaskID,Type,Iteration,RunUUID,Producer,ArtifactKey) VALUES (?,?,?,?,?,?,?,?)"
	for start := 0; start < len(links); start += 100 {
		end := start + 100
		if end > len(links) {
			end = len(links)
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, link := range links[start:end] {
				var producer any
				if link.Producer != nil {
					data, err := json.Marshal(link.Producer)
					if err != nil {
						return err
					}
					producer = string(data)
				}
				if err := tx.Exec(query, link.UUID, link.ArtifactID, link.TaskID, int32(link.Type), link.Iteration, link.RunUUID, producer, link.ArtifactKey).Error; err != nil {
					return err
				}
				if err := rejectInsertWarnings(tx); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func verifyIdentities(ctx context.Context, db *gorm.DB, table string, ids []string) error {
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
		query := "SELECT UUID FROM `" + table + "` WHERE UUID IN (" + strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",") + ")"
		rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			found[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, id := range batch {
			if !found[id] {
				return fmt.Errorf("%s is missing migrated UUID %s", table, id)
			}
		}
	}
	return nil
}

func verifyParents(ctx context.Context, db *gorm.DB, tasks []model.Task) error {
	expected := map[string]string{}
	for _, task := range tasks {
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
			if !parent.Valid || parent.String != expected[id] {
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
		case 1040, 1205, 1213, 2006, 2013:
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
func Migrate(ctx context.Context, source pb.MetadataStoreServiceClient, database *sql.DB) (Report, error) {
	var report Report
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		report, err = migrateOnce(ctx, source, database)
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

func migrateOnce(ctx context.Context, source pb.MetadataStoreServiceClient, database *sql.DB) (Report, error) {
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
	if legacySchema {
		for _, table := range []string{"artifacts", "artifact_tasks"} {
			present, err := tableExists(ctx, db, table)
			if err != nil {
				return report, err
			}
			if present {
				return report, fmt.Errorf("legacy tasks coexist with %s; refusing unqualified partial schema", table)
			}
		}
	}
	if !legacySchema {
		valid, err := columnExists(ctx, db, "tasks", "TypeAttrs")
		if err != nil {
			return report, err
		}
		if !valid {
			return report, fmt.Errorf("tasks is neither the qualified legacy nor native schema")
		}
	}
	legacyTable := "tasks"
	if backupExists {
		legacyTable = "tasks_pre_migration"
		if err := qualifyTaskSchema(ctx, db, legacyTable, true); err != nil {
			return report, err
		}
	}
	runs, err := loadRuns(ctx, db)
	if err != nil {
		return report, err
	}
	legacy := map[int64]LegacyTask{}
	var unjoined []LegacyTask
	if legacySchema || backupExists {
		legacy, unjoined, err = loadLegacyTasks(ctx, db, legacyTable)
		if err != nil {
			return report, err
		}
	}
	snapshot, err := ReadSnapshot(ctx, source)
	if err != nil {
		return report, err
	}
	plan := BuildPlan(snapshot, runs, legacy)
	if err := appendUnjoinedLegacy(&plan, unjoined, legacy, runs); err != nil {
		return report, err
	}
	plannedTaskIDs := map[string]bool{}
	for _, task := range plan.Tasks {
		plannedTaskIDs[task.UUID] = true
	}
	for id := range legacy {
		if !plannedTaskIDs[ExecutionUUID(id)] {
			return report, fmt.Errorf("legacy KFP task for MLMD execution %d could not be preserved", id)
		}
	}
	cacheRuns, err := loadCacheRuns(ctx, db)
	if err != nil {
		return report, err
	}
	cutoff, err := cacheCutoff(ctx, db, legacySchema || backupExists)
	if err != nil {
		return report, err
	}
	if !legacySchema && !backupExists && cutoff > 0 {
		if err := loadNativeCacheTimestamps(ctx, db, snapshot, legacy); err != nil {
			return report, err
		}
	}
	report.CacheKeysVerified = backfillNativeCacheKeys(&plan, snapshot, legacy, cacheRuns, cutoff)
	report.Skipped = plan.Skipped
	report.Warnings = plan.Warnings
	statements := schemaStatements()
	if legacySchema {
		temporaryExists, err := tableExists(ctx, db, "tasks_migration_tmp")
		if err != nil {
			return report, err
		}
		if !temporaryExists {
			if err := createTaskCopy(ctx, db); err != nil {
				return report, err
			}
		} else if err := qualifyTaskSchema(ctx, db, "tasks_migration_tmp", false); err != nil {
			return report, err
		}
		if err := insertTasks(ctx, db, "tasks_migration_tmp", plan.Tasks); err != nil {
			return report, err
		}
		if err := db.WithContext(ctx).Exec("RENAME TABLE tasks TO tasks_pre_migration, tasks_migration_tmp TO tasks").Error; err != nil {
			return report, err
		}
	} else if err := insertTasks(ctx, db, "tasks", plan.Tasks); err != nil {
		return report, err
	}
	var parentConstraintCount int64
	if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND CONSTRAINT_NAME='fk_tasks_parent_task'").Scan(&parentConstraintCount).Error; err != nil {
		return report, err
	}
	if parentConstraintCount == 0 {
		if err := db.WithContext(ctx).Exec("ALTER TABLE tasks ADD CONSTRAINT fk_tasks_parent_task FOREIGN KEY (ParentTaskUUID) REFERENCES tasks(UUID) ON DELETE CASCADE ON UPDATE CASCADE").Error; err != nil {
			return report, err
		}
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
	if err := insertArtifacts(ctx, db, plan.Artifacts); err != nil {
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
	taskIDs := make([]string, 0, len(plan.Tasks))
	for _, value := range plan.Tasks {
		taskIDs = append(taskIDs, value.UUID)
	}
	artifactIDs := make([]string, 0, len(plan.Artifacts))
	for _, value := range plan.Artifacts {
		artifactIDs = append(artifactIDs, value.UUID)
	}
	linkIDs := make([]string, 0, len(plan.Links))
	for _, value := range plan.Links {
		linkIDs = append(linkIDs, value.UUID)
	}
	for _, check := range []struct {
		table string
		ids   []string
	}{{"tasks", taskIDs}, {"artifacts", artifactIDs}, {"artifact_tasks", linkIDs}} {
		if err := verifyIdentities(ctx, db, check.table, check.ids); err != nil {
			return report, err
		}
	}
	if err := verifyParents(ctx, db, plan.Tasks); err != nil {
		return report, err
	}
	temporaryExists, err := tableExists(ctx, db, "tasks_migration_tmp")
	if err != nil {
		return report, err
	}
	if temporaryExists {
		return report, fmt.Errorf("tasks_migration_tmp remains after task cutover; inspect the partial migration before cleanup")
	}
	backupExists, err = tableExists(ctx, db, "tasks_pre_migration")
	if err != nil {
		return report, err
	}
	if backupExists {
		if err := db.WithContext(ctx).Exec("DROP TABLE tasks_pre_migration").Error; err != nil {
			return report, err
		}
	}
	for _, count := range []struct {
		table string
		value *int
	}{{"tasks", &report.Tasks}, {"artifacts", &report.Artifacts}, {"artifact_tasks", &report.Links}} {
		if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM `" + count.table + "`").Row().Scan(count.value); err != nil {
			return report, err
		}
	}
	return report, nil
}
